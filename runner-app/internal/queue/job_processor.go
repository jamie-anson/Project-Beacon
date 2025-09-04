package queue

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/service"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// JobProcessor handles job processing from the Redis queue
type JobProcessor struct {
	queue        *RedisQueue
	jobsService  *service.JobsService
	workers      int
	stopCh       chan struct{}
	wg           sync.WaitGroup
	running      bool
	mu           sync.RWMutex
}

// NewJobProcessor creates a new job processor
func NewJobProcessor(queue *RedisQueue, jobsService *service.JobsService, workers int) *JobProcessor {
	return &JobProcessor{
		queue:       queue,
		jobsService: jobsService,
		workers:     workers,
		stopCh:      make(chan struct{}),
	}
}

// Start begins processing jobs from the queue
func (p *JobProcessor) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return fmt.Errorf("processor is already running")
	}
	p.running = true
	p.mu.Unlock()

	log.Printf("Starting job processor with %d workers", p.workers)

	// Start recovery routine for stale jobs
	p.wg.Add(1)
	go p.recoveryLoop(ctx)

	// Start worker goroutines
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}

	return nil
}

// Stop gracefully stops the job processor
func (p *JobProcessor) Stop() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.running = false
	p.mu.Unlock()

	log.Println("Stopping job processor...")
	close(p.stopCh)
	p.wg.Wait()
	log.Println("Job processor stopped")
}

// worker processes jobs from the queue
func (p *JobProcessor) worker(ctx context.Context, workerID int) {
	defer p.wg.Done()
	
	log.Printf("Worker %d started", workerID)
	defer log.Printf("Worker %d stopped", workerID)

	for {
		select {
		case <-p.stopCh:
			return
		case <-ctx.Done():
			return
		default:
			// Try to get a job from the queue
			message, err := p.queue.Dequeue(ctx)
			if err != nil {
				log.Printf("Worker %d: failed to dequeue job: %v", workerID, err)
				time.Sleep(time.Second)
				continue
			}

			if message == nil {
				// No jobs available, wait a bit
				time.Sleep(time.Second)
				continue
			}

			// Process the job
			if err := p.processJob(ctx, message); err != nil {
				log.Printf("Worker %d: failed to process job %s: %v", workerID, message.ID, err)
				if failErr := p.queue.Fail(ctx, message, err); failErr != nil {
					log.Printf("Worker %d: failed to mark job %s as failed: %v", workerID, message.ID, failErr)
				}
			} else {
				if err := p.queue.Complete(ctx, message); err != nil {
					log.Printf("Worker %d: failed to mark job %s as complete: %v", workerID, message.ID, err)
				}
			}
		}
	}
}

// processJob handles the actual job processing logic
func (p *JobProcessor) processJob(ctx context.Context, message *JobMessage) error {
	log.Printf("Processing job %s (JobSpec: %s, Action: %s, Attempt: %d)", 
		message.ID, message.JobSpecID, message.Action, message.Attempts)

	switch message.Action {
	case "execute":
		return p.executeJob(ctx, message)
	case "validate":
		return p.validateJob(ctx, message)
	case "cleanup":
		return p.cleanupJob(ctx, message)
	default:
		return fmt.Errorf("unknown job action: %s", message.Action)
	}
}

// executeJob handles job execution
func (p *JobProcessor) executeJob(ctx context.Context, message *JobMessage) error {
	// Get the job from the database
	jobSpec, status, err := p.jobsService.GetJob(ctx, message.JobSpecID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	if status != "created" && status != "running" {
		log.Printf("Job %s is in status %s, skipping execution", message.JobSpecID, status)
		return nil
	}

	// Update job status to running
	if err := p.jobsService.JobsRepo.UpdateJobStatus(ctx, message.JobSpecID, "running"); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// TODO: Implement actual Golem execution logic
	// For now, we'll simulate job execution
	if err := p.simulateJobExecution(ctx, jobSpec); err != nil {
		// Update job status to failed
		p.jobsService.JobsRepo.UpdateJobStatus(ctx, message.JobSpecID, "failed")
		return fmt.Errorf("job execution failed: %w", err)
	}

	log.Printf("Job %s executed successfully", message.JobSpecID)
	return nil
}

// validateJob handles job validation
func (p *JobProcessor) validateJob(ctx context.Context, message *JobMessage) error {
	// Get the job from the database
	jobSpec, _, err := p.jobsService.GetJob(ctx, message.JobSpecID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	// Validate the JobSpec
	validator := models.NewJobSpecValidator()
	if err := validator.ValidateAndVerify(jobSpec); err != nil {
		// Update job status to failed
		p.jobsService.JobsRepo.UpdateJobStatus(ctx, message.JobSpecID, "failed")
		return fmt.Errorf("job validation failed: %w", err)
	}

	log.Printf("Job %s validated successfully", message.JobSpecID)
	return nil
}

// cleanupJob handles job cleanup
func (p *JobProcessor) cleanupJob(_ context.Context, message *JobMessage) error {
	log.Printf("Cleaning up job %s", message.JobSpecID)
	
	// TODO: Implement cleanup logic (remove temporary files, etc.)
	
	return nil
}

// simulateJobExecution simulates job execution for testing
func (p *JobProcessor) simulateJobExecution(ctx context.Context, jobSpec *models.JobSpec) error {
	// Simulate execution time (5-10 seconds)
	executionTime := 5 * time.Second
	
	log.Printf("Simulating execution of job %s for %v", jobSpec.ID, executionTime)
	
	select {
	case <-time.After(executionTime):
		// Simulate successful execution
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// recoveryLoop periodically recovers stale jobs
func (p *JobProcessor) recoveryLoop(ctx context.Context) {
	defer p.wg.Done()
	
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := p.queue.RecoverStaleJobs(ctx); err != nil {
				log.Printf("Failed to recover stale jobs: %v", err)
			}
		}
	}
}

// GetStats returns processor statistics
func (p *JobProcessor) GetStats(ctx context.Context) (map[string]interface{}, error) {
	queueStats, err := p.queue.GetQueueStats(ctx)
	if err != nil {
		return nil, err
	}

	p.mu.RLock()
	running := p.running
	p.mu.RUnlock()

	return map[string]interface{}{
		"running":     running,
		"workers":     p.workers,
		"queue_stats": queueStats,
	}, nil
}
