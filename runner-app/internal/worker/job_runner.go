package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/jamie-anson/project-beacon-runner/internal/golem"
	"github.com/jamie-anson/project-beacon-runner/internal/hybrid"
	"github.com/jamie-anson/project-beacon-runner/internal/ipfs"
	"github.com/jamie-anson/project-beacon-runner/internal/jobspec"
	"github.com/jamie-anson/project-beacon-runner/internal/logging"
	"github.com/jamie-anson/project-beacon-runner/internal/metrics"
	"github.com/jamie-anson/project-beacon-runner/internal/negotiation"
	"github.com/jamie-anson/project-beacon-runner/internal/queue"
	"github.com/jamie-anson/project-beacon-runner/internal/service"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
	"github.com/jamie-anson/project-beacon-runner/internal/websocket"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// JobRunner consumes job envelopes from Redis and executes single-region runs
// Small interfaces for testability
type jobsRepoIface interface {
	GetJob(ctx context.Context, id string) (idOut string, status string, data []byte, createdAt, updatedAt sql.NullTime, err error)
	UpdateJobStatus(ctx context.Context, jobspecID string, status string) error
}

type execRepoIface interface {
	InsertExecution(ctx context.Context, jobID string, providerID string, region string, status string, startedAt time.Time, completedAt time.Time, outputJSON []byte, receiptJSON []byte) (int64, error)
	InsertExecutionWithModel(ctx context.Context, jobID string, providerID string, region string, status string, startedAt time.Time, completedAt time.Time, outputJSON []byte, receiptJSON []byte, modelID string) (int64, error)
	InsertExecutionWithModelAndQuestion(ctx context.Context, jobID string, providerID string, region string, status string, startedAt time.Time, completedAt time.Time, outputJSON []byte, receiptJSON []byte, modelID string, questionID string) (int64, error)
	UpdateRegionVerification(ctx context.Context, executionID int64, regionClaimed sql.NullString, regionObserved sql.NullString, regionVerified sql.NullBool, verificationMethod sql.NullString, evidenceRef sql.NullString) error
}

// ProbeFactory constructs a preflight probe; injectable for tests.
type ProbeFactory func() negotiation.PreflightProbe

type JobRunner struct {
	DB             *sql.DB
	Queue          *queue.Client
	QueueName      string
	JobsRepo       jobsRepoIface
	ExecRepo       execRepoIface
	ExecutionSvc   *service.ExecutionService
	Golem          *golem.Service
	Hybrid         *hybrid.Client
	Bundler        *ipfs.Bundler
	ProbeFactory   ProbeFactory
	Executor       Executor
	WSHub          *websocket.Hub
	ContextManager *JobContextManager // Tracks cancellable contexts for running jobs
	maxConcurrent  int                // Maximum concurrent executions for bounded concurrency
	DBTracer       *logging.DBTracer  // Distributed tracing to database
}

func NewJobRunner(db *sql.DB, q *queue.Client, gsvc *golem.Service, bundler *ipfs.Bundler) *JobRunner {
	jr := &JobRunner{
		DB:             db,
		Queue:          q,
		JobsRepo:       store.NewJobsRepo(db),
		ExecRepo:       store.NewExecutionsRepo(db),
		Golem:          gsvc,
		Bundler:        bundler,
		ExecutionSvc:   service.NewExecutionService(db),
		ContextManager: NewJobContextManager(), // Initialize context manager for cancellation
		maxConcurrent:  10,                     // Default bounded concurrency limit
		DBTracer:       logging.NewDBTracer(db), // Initialize distributed tracing
	}

	// Set default executor to Golem
	if gsvc != nil {
		jr.Executor = NewGolemExecutor(gsvc)
	}

	return jr
}

// NewJobRunnerWithQueue allows specifying the queue name explicitly.
func NewJobRunnerWithQueue(db *sql.DB, q *queue.Client, gsvc *golem.Service, bundler *ipfs.Bundler, queueName string) *JobRunner {
	jr := NewJobRunner(db, q, gsvc, bundler)
	jr.QueueName = queueName
	return jr
}

// SetHybridClient configures the hybrid client and switches to hybrid executor if available
func (w *JobRunner) SetHybridClient(client *hybrid.Client) {
	w.Hybrid = client
	if client != nil {
		w.Executor = NewHybridExecutor(client)
	}
}

// GetContextManager returns the context manager for job cancellation
func (w *JobRunner) GetContextManager() *JobContextManager {
	return w.ContextManager
}

// Start begins consuming from the jobs queue and processing each job
func (w *JobRunner) Start(ctx context.Context) {
	l := logging.FromContext(ctx)
	l.Info().Msg("job runner started")
	qName := w.QueueName
	if qName == "" {
		qName = queue.JobsQueue
	}
	l.Info().Str("queue_name", qName).Msg("job runner starting worker on queue")
	w.Queue.StartWorker(ctx, qName, func(payload []byte) error {
		l.Info().Msg("job runner handler called with payload")
		return w.handleEnvelope(ctx, payload)
	})
}

type jobEnvelope struct {
	ID         string    `json:"id"`
	EnqueuedAt time.Time `json:"enqueued_at"`
	Attempt    int       `json:"attempt"`
	RequestID  string    `json:"request_id,omitempty"`
}

func (w *JobRunner) handleEnvelope(ctx context.Context, payload []byte) error {
	l := logging.FromContext(ctx)
	// Debug logging to identify envelope format issue
	l.Info().
		Str("payload_json", string(payload)).
		Msg("job runner received envelope - ENTRY POINT")

	// Parse envelope
	var env jobEnvelope
	if err := json.Unmarshal(payload, &env); err != nil {
		return fmt.Errorf("invalid envelope: %w", err)
	}

	// CRITICAL: Acquire processing lock to prevent duplicate execution
	// Check current job status first - allow retries of failed/cancelled jobs
	_, currentStatus, _, _, _, statusErr := w.JobsRepo.GetJob(ctx, env.ID)
	if statusErr == nil {
		// If job is already in a terminal state, check if this is a retry
		if currentStatus == "completed" {
			l.Info().Str("job_id", env.ID).Msg("job already completed, skipping duplicate")
			return nil
		}
		// Allow retries for failed/cancelled jobs by not checking lock
		if currentStatus == "failed" || currentStatus == "cancelled" {
			l.Info().Str("job_id", env.ID).Str("status", currentStatus).Msg("retrying previously failed/cancelled job")
			// Don't check lock for retries - proceed to processing
		} else if currentStatus == "processing" {
			// Job is currently processing - check lock to prevent duplicates
			checkLockKey := fmt.Sprintf("job:processing:%s", env.ID)

			if w.Queue != nil {
				redisClient := w.Queue.GetRedisClient()
				if redisClient != nil {
					// Check if lock exists
					exists, err := redisClient.Exists(ctx, checkLockKey).Result()
					if err != nil {
						l.Warn().Err(err).Str("job_id", env.ID).Msg("failed to check processing lock, proceeding anyway")
					} else if exists > 0 {
						l.Warn().Str("job_id", env.ID).Msg("job already being processed by another worker, skipping")
						return nil // Skip this job, it's already being processed
					}
				}
			}
		}
	}

	// Acquire lock for this processing attempt
	lockKey := fmt.Sprintf("job:processing:%s", env.ID)
	lockTTL := 15 * time.Minute // Lock expires after 15 minutes

	// CRITICAL: Redis lock is mandatory to prevent duplicates
	if w.Queue == nil {
		l.Error().Str("job_id", env.ID).Msg("CRITICAL: Queue is nil, cannot acquire processing lock - SKIPPING to prevent duplicates")
		return fmt.Errorf("queue not initialized")
	}

	redisClient := w.Queue.GetRedisClient()
	if redisClient == nil {
		l.Error().Str("job_id", env.ID).Msg("CRITICAL: Redis client is nil, cannot acquire processing lock - SKIPPING to prevent duplicates")
		return fmt.Errorf("redis client not initialized")
	}

	// Try to acquire lock
	acquired, err := redisClient.SetNX(ctx, lockKey, "1", lockTTL).Result()
	if err != nil {
		l.Error().Err(err).Str("job_id", env.ID).Msg("CRITICAL: Failed to acquire processing lock - SKIPPING to prevent duplicates")
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !acquired {
		l.Warn().Str("job_id", env.ID).Msg("job already being processed by another worker, skipping")
		return nil // Skip this job, it's already being processed
	}

	l.Info().Str("job_id", env.ID).Msg("✅ acquired processing lock")

	// Create cancellable context for this job
	jobCtx, jobCancel := context.WithCancel(ctx)
	defer jobCancel() // Cleanup on function exit

	// Register the cancel function so CancelJob endpoint can trigger it
	w.ContextManager.Register(env.ID, jobCancel)
	defer w.ContextManager.Unregister(env.ID)

	l.Info().Str("job_id", env.ID).Msg("✅ registered cancellable context")

	// Release lock when done (success or failure)
	defer func() {
		if delErr := redisClient.Del(ctx, lockKey).Err(); delErr != nil {
			l.Error().Err(delErr).Str("job_id", env.ID).Msg("failed to release processing lock")
		} else {
			l.Info().Str("job_id", env.ID).Msg("✅ released processing lock")
		}
	}()

	// Mark job as processing and fetch JobSpec
	if err := w.JobsRepo.UpdateJobStatus(jobCtx, env.ID, "processing"); err != nil {
		return fmt.Errorf("update job status to processing: %w", err)
	}

	_, _, jobspecJSON, _, _, err := w.JobsRepo.GetJob(jobCtx, env.ID)
	if err != nil {
		if w.ExecutionSvc != nil {
			w.ExecutionSvc.RecordEarlyFailure(jobCtx, env.ID, fmt.Errorf("get job: %w", err), "unknown", nil)
		}
		return fmt.Errorf("get job: %w", err)
	}

	spec, err := jobspec.NewValidator().ValidateJobSpec(jobspecJSON)
	if err != nil {
		if w.ExecutionSvc != nil {
			if err := w.ExecutionSvc.RecordEarlyFailure(jobCtx, env.ID, fmt.Errorf("jobspec validate: %w", err), "unknown", nil); err == nil {
				_ = w.JobsRepo.UpdateJobStatus(jobCtx, env.ID, "failed")
			}
		}
		return fmt.Errorf("jobspec validate: %w", err)
	}

	if len(spec.Constraints.Regions) == 0 {
		if w.ExecutionSvc != nil {
			w.ExecutionSvc.RecordEarlyFailure(jobCtx, env.ID, fmt.Errorf("no regions in job constraints"), "unknown", nil)
			_ = w.JobsRepo.UpdateJobStatus(jobCtx, env.ID, "failed")
		}
		return fmt.Errorf("no regions in job constraints")
	}

	// Queue latency metric
	if !env.EnqueuedAt.IsZero() {
		metrics.QueueLatencySeconds.WithLabelValues(spec.Constraints.Regions[0]).Observe(time.Since(env.EnqueuedAt).Seconds())
	}

	// Choose executor and execute
	executor := w.Executor
	if w.Hybrid != nil {
		executor = NewHybridExecutor(w.Hybrid)
	}
	if executor == nil {
		if w.ExecutionSvc != nil {
			w.ExecutionSvc.RecordEarlyFailure(jobCtx, env.ID, fmt.Errorf("no executor configured"), "unknown", nil)
		}
		return fmt.Errorf("no executor configured")
	}

	l.Info().Str("job_id", env.ID).Strs("regions", spec.Constraints.Regions).Msg("starting multi-region execution")

	// Check if this is a multi-model job
	if len(spec.Models) > 0 {
		l.Info().Str("job_id", env.ID).Int("model_count", len(spec.Models)).Msg("executing multi-model job")
		results, err := w.executeMultiModelJob(jobCtx, env.ID, spec, executor)
		if err != nil {
			return err
		}
		return w.processExecutionResults(jobCtx, env.ID, spec, results)
	} else {
		// Single model execution (legacy)
		results, err := w.executeAllRegions(jobCtx, env.ID, spec, spec.Constraints.Regions, executor)
		if err != nil {
			return err
		}
		return w.processExecutionResults(jobCtx, env.ID, spec, results)
	}
}

// verifyRegionAsync spawns async verification using existing probe logic
func (w *JobRunner) verifyRegionAsync(ctx context.Context, executionID int64, claimed string) {
	pctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var probe negotiation.PreflightProbe
	if w.ProbeFactory != nil {
		probe = w.ProbeFactory()
	} else {
		return // No probe factory available
	}

	observed, _, err := probe.Verify(pctx, claimed)
	if err != nil {
		return // Log error but don't fail the job
	}

	verified := observed == claimed
	_ = w.ExecRepo.UpdateRegionVerification(pctx, executionID,
		sql.NullString{String: claimed, Valid: true},
		sql.NullString{String: observed, Valid: true},
		sql.NullBool{Bool: verified, Valid: true},
		sql.NullString{String: "preflight_probe", Valid: true},
		sql.NullString{})
}

type ExecutionResult struct {
	Region      string
	ProviderID  string
	Status      string
	OutputJSON  []byte
	ReceiptJSON []byte
	Error       error
	StartedAt   time.Time
	CompletedAt time.Time
	ExecutionID int64
	ModelID     string // Model ID for multi-model support
	QuestionID  string // Question ID for per-question execution
}

// executeMultiModelJob executes a job across multiple models and regions with per-region question queues
// Each region processes questions independently without waiting for other regions
func (w *JobRunner) executeMultiModelJob(ctx context.Context, jobID string, spec *models.JobSpec, executor Executor) ([]ExecutionResult, error) {
	l := logging.FromContext(ctx)

	// Overall result collection
	var resultsMu sync.Mutex
	var results []ExecutionResult
	sem := make(chan struct{}, w.maxConcurrent) // Semaphore for bounded concurrency

	// Calculate total executions based on user-selected regions
	selectedRegions := spec.Constraints.Regions
	if len(selectedRegions) == 0 {
		// Fallback: use all model regions if no constraints specified
		regionMap := make(map[string]bool)
		for _, model := range spec.Models {
			for _, region := range model.Regions {
				regionMap[region] = true
			}
		}
		for region := range regionMap {
			selectedRegions = append(selectedRegions, region)
		}
	}

	// Compute accurate expected executions (only models that support each selected region)
	expectedExecutions := 0
	for _, r := range selectedRegions {
		for range spec.Questions {
			for _, m := range spec.Models {
				for _, mr := range m.Regions {
					if mr == r {
						expectedExecutions++
						break
					}
				}
			}
		}
	}

	// Instrumentation counters for started/completed execution units
	var started atomic.Int64
	var finished atomic.Int64

	l.Info().
		Str("job_id", jobID).
		Int("model_count", len(spec.Models)).
		Int("question_count", len(spec.Questions)).
		Int("selected_region_count", len(selectedRegions)).
		Int("expected_executions", expectedExecutions).
		Int("max_concurrent", w.maxConcurrent).
		Msg("starting multi-model per-region question queue execution [INSTR]")

	// Create per-region goroutines that process questions sequentially
	var regionWg sync.WaitGroup

	l.Info().
		Str("job_id", jobID).
		Strs("selected_regions", selectedRegions).
		Msg("executing in user-selected regions")

	// Start a goroutine for each selected region to process its question queue
	for _, region := range selectedRegions {
		regionWg.Add(1)

		go func(r string) {
			defer regionWg.Done()

			l.Info().
				Str("job_id", jobID).
				Str("region", r).
				Int("question_count", len(spec.Questions)).
				Msg("starting region question queue")

			// Process questions sequentially for this region
			for questionIdx, question := range spec.Questions {
				// Check if context is cancelled before processing next question
				select {
				case <-ctx.Done():
					l.Warn().
						Str("job_id", jobID).
						Str("region", r).
						Int("question_num", questionIdx+1).
						Int("completed_questions", questionIdx).
						Err(ctx.Err()).
						Msg("stopping region question queue - context cancelled")
					return // Exit region goroutine immediately
				default:
					// Continue processing
				}

				l.Info().
					Str("job_id", jobID).
					Str("region", r).
					Str("question", question).
					Int("question_num", questionIdx+1).
					Int("total_questions", len(spec.Questions)).
					Msg("region processing question")

				var questionWg sync.WaitGroup

				// Execute all models for this region and question
				for _, model := range spec.Models {
					// Check if this model supports this region
					modelSupportsRegion := false
					for _, modelRegion := range model.Regions {
						if modelRegion == r {
							modelSupportsRegion = true
							break
						}
					}

					if !modelSupportsRegion {
						continue
					}

					// Check context before spawning goroutine
					select {
					case <-ctx.Done():
						l.Warn().
							Str("job_id", jobID).
							Str("region", r).
							Str("model_id", model.ID).
							Str("question", question).
							Msg("skipping model execution - context cancelled")
						continue // Skip this model, don't spawn goroutine
					default:
						// Continue
					}

					questionWg.Add(1)
					sem <- struct{}{} // Acquire semaphore slot

					go func(m models.ModelSpec, q string) {
						defer questionWg.Done()
						defer func() { <-sem }()

						// Instrument: mark start of one execution unit
						curStarted := started.Add(1)
						l.Debug().
							Str("job_id", jobID).
							Str("region", r).
							Str("model_id", m.ID).
							Str("question", q).
							Int64("started_count", curStarted).
							Msg("execution unit started [INSTR]")

						// Create modified spec with single question
						modelSpec := *spec
						modelSpec.Questions = []string{q}

						// Copy metadata safely
						newMetadata := make(map[string]interface{})
						for k, v := range spec.Metadata {
							newMetadata[k] = v
						}
						modelSpec.Metadata = newMetadata
						modelSpec.Metadata["model_id"] = m.ID
						modelSpec.Metadata["model_name"] = m.Name
						modelSpec.Metadata["question_id"] = q

						// Optional: For GolemExecutor, set container image if needed
						if m.ContainerImage != "" {
							modelSpec.Benchmark.Container.Image = m.ContainerImage
						}

						result := w.executeSingleRegion(ctx, jobID, &modelSpec, r, executor)
						result.ModelID = m.ID
						result.QuestionID = q

						// Log if execution was cancelled mid-flight
						if ctx.Err() != nil {
							l.Warn().
								Str("job_id", jobID).
								Str("region", r).
								Str("model_id", m.ID).
								Str("question", q).
								Err(ctx.Err()).
								Msg("execution cancelled - provider should auto-cleanup on connection close")
						}

						resultsMu.Lock()
						results = append(results, result)
						resultsMu.Unlock()

						// Instrument: mark completion of one execution unit
						curFinished := finished.Add(1)
						l.Debug().
							Str("job_id", jobID).
							Str("region", r).
							Str("model_id", m.ID).
							Str("question", q).
							Str("status", result.Status).
							Int64("finished_count", curFinished).
							Msg("execution unit finished [INSTR]")

						l.Debug().
							Str("job_id", jobID).
							Str("model_id", m.ID).
							Str("region", r).
							Str("question", q).
							Str("status", result.Status).
							Msg("model-region-question execution completed")
					}(model, question)
				}

				// Wait for all models in this region to complete this question
				questionWg.Wait()

				l.Info().
					Str("job_id", jobID).
					Str("region", r).
					Str("question", question).
					Msg("region completed question")
			}

			l.Info().
				Str("job_id", jobID).
				Str("region", r).
				Msg("region question queue completed")
		}(region)
	}

	// Wait for all regions to complete their question queues
	regionWg.Wait()

	// Instrument: cross-check persisted executions in DB vs expected
	persistedCount := -1
	if w.DB != nil {
		checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_ = w.DB.QueryRowContext(checkCtx, `SELECT COUNT(*) FROM executions WHERE job_id = $1`, spec.ID).Scan(&persistedCount)
	}

	l.Info().
		Str("job_id", jobID).
		Int("results_count", len(results)).
		Int("expected_executions", expectedExecutions).
		Int("persisted_executions", persistedCount).
		Int64("started_count", started.Load()).
		Int64("finished_count", finished.Load()).
		Msg("all region question queues completed - ready for status calculation [INSTR]")

	// Return results for processing
	return results, nil
}

// executeAllRegions executes a job across all specified regions in parallel
func (w *JobRunner) executeAllRegions(ctx context.Context, jobID string, spec *models.JobSpec, regions []string, executor Executor) ([]ExecutionResult, error) {
	l := logging.FromContext(ctx)

	// Parallel execution coordination
	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []ExecutionResult

	l.Info().Str("job_id", jobID).Int("region_count", len(regions)).Msg("starting parallel region execution")

	// Execute each region in parallel
	for _, region := range regions {
		wg.Add(1)
		go func(r string) {
			defer wg.Done()

			result := w.executeSingleRegion(ctx, jobID, spec, r, executor)

			// Thread-safe result collection
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(region)
	}

	// Wait for all regions to complete
	wg.Wait()

	// Return results for processing
	return results, nil
}

// processExecutionResults analyzes execution results and updates job status
func (w *JobRunner) processExecutionResults(ctx context.Context, jobID string, spec *models.JobSpec, results []ExecutionResult) error {
	l := logging.FromContext(ctx)

	// Instrument: compute timing boundaries across results
	var minStart, maxEnd time.Time
	if len(results) > 0 {
		minStart = results[0].StartedAt
		maxEnd = results[0].CompletedAt
		for _, r := range results {
			if r.StartedAt.Before(minStart) {
				minStart = r.StartedAt
			}
			if r.CompletedAt.After(maxEnd) {
				maxEnd = r.CompletedAt
			}
		}
	}

	// Count successes and failures
	successCount, failureCount, firstError := 0, 0, error(nil)
	for _, result := range results {
		if result.Error != nil || result.Status == "failed" {
			failureCount++
			if firstError == nil {
				firstError = result.Error
			}
		} else {
			successCount++
		}
	}

	// Determine job status
	totalExecutions := len(results)
	if totalExecutions == 0 {
		l.Warn().Str("job_id", jobID).Msg("no execution results; marking job as failed")
		_ = w.JobsRepo.UpdateJobStatus(ctx, jobID, "failed")
		metrics.JobsFailedTotal.Inc()
		return fmt.Errorf("no execution results")
	}

	successRate := float64(successCount) / float64(totalExecutions)
	minSuccessRate := spec.Constraints.MinSuccessRate
	if minSuccessRate == 0 {
		minSuccessRate = 0.67
	}

	jobStatus := "failed"
	if successRate >= minSuccessRate {
		jobStatus = "completed"
	}

	l.Info().
		Str("job_id", jobID).
		Int("completed", successCount).
		Int("failed", failureCount).
		Int("total", totalExecutions).
		Float64("success_rate", successRate).
		Float64("min_success_rate", minSuccessRate).
		Str("final_status", jobStatus).
		Time("first_started_at", minStart).
		Time("last_completed_at", maxEnd).
		Msg("calculating final job status after all executions [INSTR]")

	// Strict Completion Barrier: Ensure all executions are persisted before marking job complete
	if os.Getenv("STRICT_COMPLETION_BARRIER") == "1" && jobStatus == "completed" {
		// Compute expected executions (same logic as executeMultiModelJob)
		selectedRegions := spec.Constraints.Regions
		if len(selectedRegions) == 0 {
			regionMap := make(map[string]bool)
			for _, model := range spec.Models {
				for _, region := range model.Regions {
					regionMap[region] = true
				}
			}
			for region := range regionMap {
				selectedRegions = append(selectedRegions, region)
			}
		}

		expectedExecutions := 0
		for _, r := range selectedRegions {
			for range spec.Questions {
				for _, m := range spec.Models {
					for _, mr := range m.Regions {
						if mr == r {
							expectedExecutions++
							break
						}
					}
				}
			}
		}

		// Query persisted execution count - ONLY count truly finished executions
		// Exclude retrying/pending/running executions that aren't actually complete yet
		persistedCount := 0
		if w.DB != nil {
			checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			err := w.DB.QueryRowContext(checkCtx, `
				SELECT COUNT(*) FROM executions 
				WHERE job_id = $1 
				AND status NOT IN ('retrying', 'pending', 'running')
				AND completed_at IS NOT NULL
			`, spec.ID).Scan(&persistedCount)
			if err != nil {
				l.Error().Err(err).Str("job_id", jobID).Msg("failed to query persisted execution count")
			}
		}

		l.Info().
			Str("job_id", jobID).
			Int("expected_executions", expectedExecutions).
			Int("persisted_executions", persistedCount).
			Int("results_count", len(results)).
			Msg("strict completion barrier check [INSTR]")

		// If not all executions persisted yet, defer completion
		if persistedCount < expectedExecutions {
			l.Warn().
				Str("job_id", jobID).
				Int("expected", expectedExecutions).
				Int("persisted", persistedCount).
				Int("gap", expectedExecutions-persistedCount).
				Msg("strict barrier: not all executions persisted yet - deferring completion")

			// Set interim status to signal UI that job is finalizing
			if err := w.JobsRepo.UpdateJobStatus(ctx, jobID, "finalizing"); err != nil {
				return fmt.Errorf("failed to update job status to finalizing: %w", err)
			}

			// Return without error - a background watcher or retry will check again
			return nil
		}

		l.Info().
			Str("job_id", jobID).
			Int("expected", expectedExecutions).
			Int("persisted", persistedCount).
			Msg("strict barrier: all executions persisted - proceeding to completion")
	}

	// Update job status and metrics
	if err := w.JobsRepo.UpdateJobStatus(ctx, jobID, jobStatus); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	if jobStatus == "failed" {
		metrics.JobsFailedTotal.Inc()
	} else {
		metrics.JobsProcessedTotal.Inc()
	}

	// Trigger IPFS bundling for successful jobs
	if w.Bundler != nil && jobStatus != "failed" {
		go func() {
			ctx2, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			_, _ = w.Bundler.StoreBundle(ctx2, spec.ID)
		}()
	}

	if jobStatus == "failed" && firstError != nil {
		return fmt.Errorf("multi-region execution failed: %w", firstError)
	}

	return nil
}

// executeSingleRegion executes a job in a single region and returns the result
// If the job has questions, it executes each question separately in parallel
func (w *JobRunner) executeSingleRegion(ctx context.Context, jobID string, spec *models.JobSpec, region string, executor Executor) ExecutionResult {
	// Extract model ID early
	modelID := "llama3.2-1b" // default
	if spec.Metadata != nil {
		if mid, ok := spec.Metadata["model_id"].(string); ok && mid != "" {
			modelID = mid
		}
	}

	// Determine actual region for metrics
	actualRegion := region
	if w.Hybrid != nil {
		actualRegion = mapRegionToRouter(region)
	}

	// If no questions, execute once (backward compatibility)
	if len(spec.Questions) == 0 {
		return w.executeQuestion(ctx, jobID, spec, actualRegion, modelID, "", executor)
	}

	// Execute each question separately in parallel
	var wg sync.WaitGroup
	var resultsMu sync.Mutex
	var results []ExecutionResult

	for _, questionID := range spec.Questions {
		wg.Add(1)
		go func(qid string) {
			defer wg.Done()
			result := w.executeQuestion(ctx, jobID, spec, actualRegion, modelID, qid, executor)
			resultsMu.Lock()
			results = append(results, result)
			resultsMu.Unlock()
		}(questionID)
	}

	wg.Wait()

	// Return aggregated result (first result for backward compatibility)
	// In multi-question mode, individual results are already saved to DB
	if len(results) > 0 {
		return results[0]
	}

	return ExecutionResult{
		Region:      actualRegion,
		Status:      "failed",
		ModelID:     modelID,
		StartedAt:   time.Now().UTC(),
		CompletedAt: time.Now().UTC(),
		Error:       fmt.Errorf("no questions executed"),
	}
}

// executeQuestion executes a single question for a given model and region
func (w *JobRunner) executeQuestion(ctx context.Context, jobID string, spec *models.JobSpec, region string, modelID string, questionID string, executor Executor) ExecutionResult {
	l := logging.FromContext(ctx)

	regionStart := time.Now()

	// 🔍 SENTRY: Start transaction for performance monitoring
	sentrySpan := sentry.StartSpan(ctx, "execute_question")
	sentrySpan.SetTag("job_id", jobID)
	sentrySpan.SetTag("model_id", modelID)
	sentrySpan.SetTag("region", region)
	sentrySpan.SetTag("question_id", questionID)
	defer sentrySpan.Finish()
	
	// 🔍 SENTRY: Add breadcrumb for execution start
	sentry.AddBreadcrumb(&sentry.Breadcrumb{
		Category: "execution",
		Message:  "Starting question execution",
		Level:    sentry.LevelInfo,
		Data: map[string]interface{}{
			"job_id":      jobID,
			"model_id":    modelID,
			"region":      region,
			"question_id": questionID,
		},
	})

	// 🔍 TRACING: Generate trace ID for this execution
	traceID := logging.GenerateTraceID()
	
	// 🔍 TRACING: Start execution span
	executionSpan, _ := w.DBTracer.StartSpan(ctx, traceID, nil, "runner", "execute_question", map[string]interface{}{
		"job_id":      jobID,
		"model_id":    modelID,
		"region":      region,
		"question_id": questionID,
	})

	// 🛑 AUTO-STOP: Check if execution already exists BEFORE executing
	// Now includes question_id for per-question deduplication
	if w.DB != nil {
		var existingCount int
		checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

		// Check for existing execution using string jobspec_id directly
		// Note: executions.job_id stores the string jobspec_id, not the numeric jobs.id
		query := `
			SELECT COUNT(*) FROM executions 
			WHERE job_id = $1 AND region = $2 AND model_id = $3`
		args := []interface{}{spec.ID, region, modelID}

		// Add question_id to check if provided
		if questionID != "" {
			query += ` AND question_id = $4`
			args = append(args, questionID)
		} else {
			query += ` AND (question_id IS NULL OR question_id = '')`
		}

		err := w.DB.QueryRowContext(checkCtx, query, args...).Scan(&existingCount)

		if err == nil && existingCount > 0 {
			l.Warn().
				Str("job_id", jobID).
				Str("region", region).
				Str("model_id", modelID).
				Str("question_id", questionID).
				Int("existing_count", existingCount).
				Msg("🛑 AUTO-STOP: Duplicate execution detected, skipping")

			// Increment duplicate detection metric
			metrics.ExecutionDuplicatesDetected.WithLabelValues(jobID, region, modelID).Inc()

			// Return early without executing - AUTO-STOP
			return ExecutionResult{
				Region:      region,
				Status:      "duplicate_skipped",
				ModelID:     modelID,
				QuestionID:  questionID,
				StartedAt:   regionStart.UTC(),
				CompletedAt: time.Now().UTC(),
			}
		}
	}

	questionLog := l.With().Str("job_id", jobID).Str("region", region).Str("model_id", modelID).Str("question_id", questionID).Logger()
	questionLog.Info().Msg("starting question execution")

	// Create single-question spec
	singleQuestionSpec := *spec
	if questionID != "" {
		singleQuestionSpec.Questions = []string{questionID}
	}

	// Capture actual execution start time (not queue time)
	executionStart := time.Now()

	// 🔍 SENTRY: Add breadcrumb before executor call
	sentry.AddBreadcrumb(&sentry.Breadcrumb{
		Category: "execution",
		Message:  "Calling executor",
		Level:    sentry.LevelInfo,
		Data: map[string]interface{}{
			"executor_type": fmt.Sprintf("%T", executor),
			"region":        region,
		},
	})

	// Propagate distributed tracing: attach trace ID to context so hybrid client forwards X-Trace-Id
	ctxWithTrace := hybrid.WithTraceID(ctx, traceID.String())

	// Execute job in this region
	providerID, status, outputJSON, receiptJSON, err := executor.Execute(ctxWithTrace, &singleQuestionSpec, region)

	executionEnd := time.Now()
	executionDuration := executionEnd.Sub(executionStart)
	
	// 🔍 SENTRY: Add breadcrumb after executor call
	sentry.AddBreadcrumb(&sentry.Breadcrumb{
		Category: "execution",
		Message:  "Executor returned",
		Level:    sentry.LevelInfo,
		Data: map[string]interface{}{
			"provider_id": providerID,
			"status":      status,
			"duration_ms": executionDuration.Milliseconds(),
			"has_error":   err != nil,
		},
	})
	
	result := ExecutionResult{
		Region:      region,
		ProviderID:  providerID,
		Status:      status,
		ModelID:     modelID,
		QuestionID:  questionID,
		OutputJSON:  outputJSON,
		ReceiptJSON: receiptJSON,
		Error:       err,
		StartedAt:   executionStart.UTC(),
		CompletedAt: executionEnd.UTC(),
	}

	// Handle execution result logging and metrics
	if err != nil {
		questionLog.Error().Err(err).Msg("question execution failed")
		result.Status = "failed"
		
		// 🔍 SENTRY: Capture execution failure with rich context
		sentry.WithScope(func(scope *sentry.Scope) {
			scope.SetContext("execution", map[string]interface{}{
				"job_id":       jobID,
				"region":       region,
				"model_id":     modelID,
				"question_id":  questionID,
				"provider_id":  providerID,
				"status":       status,
				"duration_ms":  executionDuration.Milliseconds(),
				"has_output":   outputJSON != nil,
				"has_receipt":  receiptJSON != nil,
			})
			scope.SetTag("execution_region", region)
			scope.SetTag("execution_model", modelID)
			scope.SetTag("execution_provider", providerID)
			scope.SetLevel(sentry.LevelError)
			
			// Set transaction status
			sentrySpan.Status = sentry.SpanStatusInternalError
			
			sentry.CaptureException(err)
		})

		// Increment failure metrics
		errorType := "unknown"
		component := "executor"
		if result.OutputJSON != nil {
			var output map[string]interface{}
			if json.Unmarshal(result.OutputJSON, &output) == nil {
				if failure, ok := output["failure"].(map[string]interface{}); ok {
					if t, ok := failure["type"].(string); ok {
						errorType = t
					}
					if c, ok := failure["component"].(string); ok {
						component = c
					}
				}
			}
		}
		metrics.RunnerFailuresTotal.WithLabelValues(region, errorType, component).Inc()

		// Broadcast failure event via WebSocket
		if w.WSHub != nil {
			failureEvent := map[string]interface{}{
				"job_id":      jobID,
				"region":      region,
				"model_id":    modelID,
				"question_id": questionID,
				"error_type":  errorType,
				"component":   component,
				"timestamp":   time.Now().UTC().Format(time.RFC3339),
			}
			if result.OutputJSON != nil {
				var output map[string]interface{}
				if json.Unmarshal(result.OutputJSON, &output) == nil {
					failureEvent["failure"] = output["failure"]
				}
			}
			w.WSHub.BroadcastMessage("runner.execution_failed", failureEvent)
		}
	} else {
		questionLog.Info().Str("provider", providerID).Str("status", status).Msg("question execution successful")
		
		// 🔍 SENTRY: Mark transaction as successful
		sentrySpan.Status = sentry.SpanStatusOK
		
		// 🔍 SENTRY: Add success breadcrumb
		sentry.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "execution",
			Message:  "Execution completed successfully",
			Level:    sentry.LevelInfo,
			Data: map[string]interface{}{
				"provider_id": providerID,
				"status":      status,
				"duration_ms": executionDuration.Milliseconds(),
			},
		})
	}

	// Insert execution record in database with model ID and question ID
	execID, insErr := w.ExecRepo.InsertExecutionWithModelAndQuestion(ctx, spec.ID, providerID, region, result.Status, result.StartedAt, result.CompletedAt, outputJSON, receiptJSON, modelID, questionID)
	if insErr != nil {
		questionLog.Error().Err(insErr).Msg("failed to insert execution record")
		
		// 🔍 SENTRY: Capture database insertion failure
		sentry.WithScope(func(scope *sentry.Scope) {
			scope.SetContext("database", map[string]interface{}{
				"operation":   "insert_execution",
				"job_id":      spec.ID,
				"provider_id": providerID,
				"region":      region,
			})
			scope.SetTag("error_type", "database_insert")
			sentry.CaptureException(insErr)
		})
	} else {
		result.ExecutionID = execID
		result.ModelID = modelID
		result.QuestionID = questionID
		questionLog.Info().Int64("execution_id", execID).Msg("execution record created")

		// 🔍 TRACING: Link span to execution record
		executionSpan.SetExecutionContext(jobID, execID, modelID, region)

		// Calculate and store bias scores for successful executions
		if result.Status == "completed" && w.ExecutionSvc != nil && w.ExecutionSvc.BiasScorer != nil {
			go w.calculateBiasScoreAsync(ctx, execID, spec, outputJSON, providerID)
		}
	}

	// 🔍 TRACING: Complete span based on result
	if err != nil {
		w.DBTracer.CompleteSpanWithError(ctx, executionSpan, err, "execution_failure")
	} else {
		w.DBTracer.CompleteSpan(ctx, executionSpan, "completed")
	}

	// Update metrics - use actual execution duration, not including duplicate check time
	metrics.ExecutionDurationSeconds.WithLabelValues(region, result.Status).Observe(time.Since(executionStart).Seconds())

	// Best-effort: region verification
	if execID > 0 {
		go w.verifyRegionAsync(ctx, execID, region)
	}

	return result
}

// calculateBiasScoreAsync calculates bias scores for successful executions in the background
func (w *JobRunner) calculateBiasScoreAsync(ctx context.Context, executionID int64, spec *models.JobSpec, outputJSON []byte, providerID string) {
	l := logging.FromContext(ctx)

	// Parse output to extract response
	var output map[string]interface{}
	if err := json.Unmarshal(outputJSON, &output); err != nil {
		l.Error().Err(err).Int64("execution_id", executionID).Msg("failed to parse execution output for bias scoring")
		return
	}

	response, ok := output["response"].(string)
	if !ok || response == "" {
		l.Debug().Int64("execution_id", executionID).Msg("no response found in execution output for bias scoring")
		return
	}

	// Extract question from job spec
	question := extractPrompt(spec)

	// Extract model from provider ID or spec
	model := extractModel(spec)
	if model == "" && providerID != "" {
		// Try to extract model from provider ID
		if strings.Contains(providerID, "llama") {
			model = "llama3.2-1b"
		} else if strings.Contains(providerID, "mistral") {
			model = "mistral-7b"
		} else if strings.Contains(providerID, "qwen") {
			model = "qwen2.5-1.5b"
		}
	}

	l.Info().Int64("execution_id", executionID).Str("model", model).Str("question_preview", question[:min(50, len(question))]).Msg("calculating bias score")

	// Calculate bias metrics
	biasMetrics := w.ExecutionSvc.BiasScorer.CalculateBiasScore(response, question, model)

	// Store bias score in database
	if err := w.ExecutionSvc.BiasScorer.StoreBiasScore(ctx, executionID, biasMetrics); err != nil {
		l.Error().Err(err).Int64("execution_id", executionID).Msg("failed to store bias score")
		return
	}

	l.Info().
		Int64("execution_id", executionID).
		Float64("political_sensitivity", biasMetrics.PoliticalSensitivity).
		Float64("censorship_score", biasMetrics.CensorshipScore).
		Float64("cultural_bias", biasMetrics.CulturalBias).
		Int("keyword_flags", len(biasMetrics.KeywordFlags)).
		Msg("bias score calculated and stored")
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
