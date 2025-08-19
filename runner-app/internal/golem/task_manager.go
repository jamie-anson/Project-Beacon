package golem

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
	apperrors "github.com/jamie-anson/project-beacon-runner/internal/errors"
)

// TaskManager handles the lifecycle of Golem tasks
type TaskManager struct {
	client      *YagnaClient
	tasks       map[string]*ManagedTask
	mu          sync.RWMutex
	pollInterval time.Duration
	maxRetries   int
}

// ManagedTask wraps a Golem task with management metadata
type ManagedTask struct {
	Task         *Task
	JobSpecID    string
	Region       string
	Retries      int
	LastPoll     time.Time
	Callbacks    TaskCallbacks
	Context      context.Context
	CancelFunc   context.CancelFunc
}

// TaskCallbacks define hooks for task lifecycle events
type TaskCallbacks struct {
	OnStarted   func(task *Task)
	OnCompleted func(task *Task, results *TaskResults)
	OnFailed    func(task *Task, err error)
	OnProgress  func(task *Task, status TaskStatus)
}

// TaskManagerConfig configures the task manager
type TaskManagerConfig struct {
	PollInterval time.Duration
	MaxRetries   int
	TaskTimeout  time.Duration
}

// DefaultTaskManagerConfig returns sensible defaults
func DefaultTaskManagerConfig() TaskManagerConfig {
	return TaskManagerConfig{
		PollInterval: 10 * time.Second,
		MaxRetries:   3,
		TaskTimeout:  30 * time.Minute,
	}
}

// NewTaskManager creates a new task manager
func NewTaskManager(client *YagnaClient, config TaskManagerConfig) *TaskManager {
	return &TaskManager{
		client:       client,
		tasks:        make(map[string]*ManagedTask),
		pollInterval: config.PollInterval,
		maxRetries:   config.MaxRetries,
	}
}

// SubmitTask submits a task and starts monitoring it
func (tm *TaskManager) SubmitTask(ctx context.Context, jobSpecID string, spec TaskSpec, region string, callbacks TaskCallbacks) (*Task, error) {
	// Create task context with timeout
	taskCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	
	// Submit task to Yagna
	task, err := tm.client.SubmitTask(taskCtx, spec)
	if err != nil {
		cancel()
		return nil, apperrors.Wrap(err, apperrors.ExternalServiceError, "failed to submit task to Yagna")
	}

	// Create managed task
	managedTask := &ManagedTask{
		Task:       task,
		JobSpecID:  jobSpecID,
		Region:     region,
		Retries:    0,
		LastPoll:   time.Now(),
		Callbacks:  callbacks,
		Context:    taskCtx,
		CancelFunc: cancel,
	}

	// Store and start monitoring
	tm.mu.Lock()
	tm.tasks[task.ID] = managedTask
	tm.mu.Unlock()

	// Start monitoring in background
	go tm.monitorTask(managedTask)

	return task, nil
}

// GetTask retrieves a managed task by ID
func (tm *TaskManager) GetTask(taskID string) (*ManagedTask, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	
	task, exists := tm.tasks[taskID]
	return task, exists
}

// ListTasks returns all managed tasks, optionally filtered by JobSpec ID
func (tm *TaskManager) ListTasks(jobSpecID string) []*ManagedTask {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	
	var tasks []*ManagedTask
	for _, task := range tm.tasks {
		if jobSpecID == "" || task.JobSpecID == jobSpecID {
			tasks = append(tasks, task)
		}
	}
	
	return tasks
}

// CancelTask cancels a managed task
func (tm *TaskManager) CancelTask(taskID string) error {
	tm.mu.RLock()
	managedTask, exists := tm.tasks[taskID]
	tm.mu.RUnlock()
	
	if !exists {
		return apperrors.Newf(apperrors.NotFoundError, "task %s not found", taskID)
	}

	// Cancel the task context
	managedTask.CancelFunc()

	// Cancel on Yagna
	if err := tm.client.CancelTask(managedTask.Context, taskID); err != nil {
		// Log but don't fail - task might already be completed
	}

	// Update status
	managedTask.Task.Status = TaskStatusCancelled

	// Trigger callback
	if managedTask.Callbacks.OnFailed != nil {
		managedTask.Callbacks.OnFailed(managedTask.Task, apperrors.New(apperrors.InternalError, "task cancelled"))
	}

	// Remove from tracking
	tm.mu.Lock()
	delete(tm.tasks, taskID)
	tm.mu.Unlock()

	return nil
}

// monitorTask monitors a single task until completion
func (tm *TaskManager) monitorTask(managedTask *ManagedTask) {
	ticker := time.NewTicker(tm.pollInterval)
	defer ticker.Stop()
	defer managedTask.CancelFunc()

	for {
		select {
		case <-managedTask.Context.Done():
			// Task cancelled or timed out
			tm.handleTaskTimeout(managedTask)
			return

		case <-ticker.C:
			if err := tm.pollTask(managedTask); err != nil {
				tm.handleTaskError(managedTask, err)
				return
			}

			// Check if task is complete
			if tm.isTaskComplete(managedTask.Task.Status) {
				tm.handleTaskCompletion(managedTask)
				return
			}
		}
	}
}

// pollTask polls Yagna for task status
func (tm *TaskManager) pollTask(managedTask *ManagedTask) error {
	task, err := tm.client.GetTask(managedTask.Context, managedTask.Task.ID)
	if err != nil {
		return err
	}

	// Update task state
	oldStatus := managedTask.Task.Status
	managedTask.Task = task
	managedTask.LastPoll = time.Now()

	// Trigger progress callback if status changed
	if oldStatus != task.Status && managedTask.Callbacks.OnProgress != nil {
		managedTask.Callbacks.OnProgress(task, task.Status)
	}

	// Trigger started callback
	if oldStatus == TaskStatusPending && task.Status == TaskStatusRunning && managedTask.Callbacks.OnStarted != nil {
		managedTask.Callbacks.OnStarted(task)
	}

	return nil
}

// isTaskComplete checks if a task has reached a terminal state
func (tm *TaskManager) isTaskComplete(status TaskStatus) bool {
	return status == TaskStatusCompleted || status == TaskStatusFailed || status == TaskStatusCancelled
}

// handleTaskCompletion handles successful task completion
func (tm *TaskManager) handleTaskCompletion(managedTask *ManagedTask) {
	// Remove from tracking
	tm.mu.Lock()
	delete(tm.tasks, managedTask.Task.ID)
	tm.mu.Unlock()

	// Trigger appropriate callback
	switch managedTask.Task.Status {
	case TaskStatusCompleted:
		if managedTask.Callbacks.OnCompleted != nil {
			managedTask.Callbacks.OnCompleted(managedTask.Task, managedTask.Task.Results)
		}
	case TaskStatusFailed:
		if managedTask.Callbacks.OnFailed != nil {
			err := apperrors.New(apperrors.ExternalServiceError, managedTask.Task.Error)
			managedTask.Callbacks.OnFailed(managedTask.Task, err)
		}
	case TaskStatusCancelled:
		if managedTask.Callbacks.OnFailed != nil {
			err := apperrors.New(apperrors.InternalError, "task was cancelled")
			managedTask.Callbacks.OnFailed(managedTask.Task, err)
		}
	}
}

// handleTaskError handles polling errors with retry logic
func (tm *TaskManager) handleTaskError(managedTask *ManagedTask, err error) {
	managedTask.Retries++

	// If we've exceeded max retries, fail the task
	if managedTask.Retries >= tm.maxRetries {
		tm.mu.Lock()
		delete(tm.tasks, managedTask.Task.ID)
		tm.mu.Unlock()

		if managedTask.Callbacks.OnFailed != nil {
			wrappedErr := apperrors.Wrapf(err, apperrors.ExternalServiceError, "task polling failed after %d retries", managedTask.Retries)
			managedTask.Callbacks.OnFailed(managedTask.Task, wrappedErr)
		}
		return
	}

	// Otherwise, continue monitoring with exponential backoff
	backoff := time.Duration(managedTask.Retries) * tm.pollInterval
	time.Sleep(backoff)
}

// handleTaskTimeout handles task timeout
func (tm *TaskManager) handleTaskTimeout(managedTask *ManagedTask) {
	// Try to cancel on Yagna
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	tm.client.CancelTask(ctx, managedTask.Task.ID)

	// Remove from tracking
	tm.mu.Lock()
	delete(tm.tasks, managedTask.Task.ID)
	tm.mu.Unlock()

	// Trigger failure callback
	if managedTask.Callbacks.OnFailed != nil {
		err := apperrors.New(apperrors.TimeoutError, "task execution timed out")
		managedTask.Callbacks.OnFailed(managedTask.Task, err)
	}
}

// GetStats returns task manager statistics
func (tm *TaskManager) GetStats() TaskManagerStats {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	stats := TaskManagerStats{
		TotalTasks:   len(tm.tasks),
		PollInterval: tm.pollInterval,
		MaxRetries:   tm.maxRetries,
		TasksByStatus: make(map[TaskStatus]int),
		TasksByRegion: make(map[string]int),
	}

	for _, task := range tm.tasks {
		stats.TasksByStatus[task.Task.Status]++
		stats.TasksByRegion[task.Region]++
	}

	return stats
}

// TaskManagerStats represents task manager statistics
type TaskManagerStats struct {
	TotalTasks    int                    `json:"total_tasks"`
	PollInterval  time.Duration          `json:"poll_interval"`
	MaxRetries    int                    `json:"max_retries"`
	TasksByStatus map[TaskStatus]int     `json:"tasks_by_status"`
	TasksByRegion map[string]int         `json:"tasks_by_region"`
}

// CreateTaskSpecFromJobSpec converts a JobSpec to a Golem TaskSpec
func CreateTaskSpecFromJobSpec(jobSpec *models.JobSpec, region string) TaskSpec {
	// Extract container image and commands from JobSpec
	image := jobSpec.Benchmark.Container.Image
	commands := make([]string, len(jobSpec.Benchmark.Container.Command))
	copy(commands, jobSpec.Benchmark.Container.Command)

	// Convert resource requirements
	resources := TaskResources{
		CPU:    1.0, // Default to 1 CPU core
		Memory: 512, // Default to 512MB
		Disk:   1024, // Default to 1GB
	}

	// Apply any resource constraints from JobSpec
	if len(jobSpec.Constraints.Regions) > 0 {
		// TODO: Map JobSpec constraints to TaskResources
	}

	// Create constraints for provider selection
	constraints := TaskConstraints{
		Regions:       []string{region},
		MinReputation: 0.8, // Require good reputation
		MaxPrice:      1.0,  // Max $1 per hour
	}

	return TaskSpec{
		Name:        fmt.Sprintf("job-%s-%s", jobSpec.ID, region),
		Image:       image,
		Commands:    commands,
		Resources:   resources,
		Constraints: constraints,
		Timeout:     30 * time.Minute,
		Env:         make(map[string]string),
	}
}

// Shutdown gracefully shuts down the task manager
func (tm *TaskManager) Shutdown(ctx context.Context) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Cancel all running tasks
	for taskID, managedTask := range tm.tasks {
		managedTask.CancelFunc()
		
		// Try to cancel on Yagna
		if err := tm.client.CancelTask(ctx, taskID); err != nil {
			// Log but continue shutdown
		}
	}

	// Clear task map
	tm.tasks = make(map[string]*ManagedTask)

	return nil
}
