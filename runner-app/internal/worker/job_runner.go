package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/golem"
	"github.com/jamie-anson/project-beacon-runner/internal/hybrid"
	"github.com/jamie-anson/project-beacon-runner/internal/ipfs"
	"github.com/jamie-anson/project-beacon-runner/internal/jobspec"
	"github.com/jamie-anson/project-beacon-runner/internal/logging"
	"github.com/jamie-anson/project-beacon-runner/internal/queue"
	"github.com/jamie-anson/project-beacon-runner/internal/service"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
	"github.com/jamie-anson/project-beacon-runner/internal/metrics"
	"github.com/jamie-anson/project-beacon-runner/internal/negotiation"
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
	UpdateRegionVerification(ctx context.Context, executionID int64, regionClaimed sql.NullString, regionObserved sql.NullString, regionVerified sql.NullBool, verificationMethod sql.NullString, evidenceRef sql.NullString) error
}

// ProbeFactory constructs a preflight probe; injectable for tests.
type ProbeFactory func() negotiation.PreflightProbe

type JobRunner struct {
	DB           *sql.DB
	Queue        *queue.Client
	QueueName    string
	JobsRepo     jobsRepoIface
	ExecRepo     execRepoIface
	ExecutionSvc *service.ExecutionService
	Golem        *golem.Service
	Hybrid       *hybrid.Client
	Bundler      *ipfs.Bundler
	ProbeFactory ProbeFactory
	Executor     Executor
}

func NewJobRunner(db *sql.DB, q *queue.Client, gsvc *golem.Service, bundler *ipfs.Bundler) *JobRunner {
	jr := &JobRunner{
		DB:           db,
		Queue:        q,
		JobsRepo:     store.NewJobsRepo(db),
		ExecRepo:     store.NewExecutionsRepo(db),
		Golem:        gsvc,
		Bundler:      bundler,
		ExecutionSvc: service.NewExecutionService(db),
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
	ID         string     `json:"id"`
	EnqueuedAt time.Time  `json:"enqueued_at"`
	Attempt    int        `json:"attempt"`
	RequestID  string     `json:"request_id,omitempty"`
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
	

	// Mark job as processing and fetch JobSpec
	if err := w.JobsRepo.UpdateJobStatus(ctx, env.ID, "processing"); err != nil {
		return fmt.Errorf("update job status to processing: %w", err)
	}

	_, _, jobspecJSON, _, _, err := w.JobsRepo.GetJob(ctx, env.ID)
	if err != nil {
		w.ExecutionSvc.RecordEarlyFailure(ctx, env.ID, fmt.Errorf("get job: %w", err), "unknown", nil)
		return fmt.Errorf("get job: %w", err)
	}

	spec, err := jobspec.NewValidator().ValidateJobSpec(jobspecJSON)
	if err != nil {
		w.ExecutionSvc.RecordEarlyFailure(ctx, env.ID, fmt.Errorf("jobspec validate: %w", err), "unknown", nil)
		_ = w.JobsRepo.UpdateJobStatus(ctx, env.ID, "failed")
		return fmt.Errorf("jobspec validate: %w", err)
	}

	if len(spec.Constraints.Regions) == 0 {
		w.ExecutionSvc.RecordEarlyFailure(ctx, env.ID, fmt.Errorf("no regions in job constraints"), "unknown", nil)
		_ = w.JobsRepo.UpdateJobStatus(ctx, env.ID, "failed")
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
		w.ExecutionSvc.RecordEarlyFailure(ctx, env.ID, fmt.Errorf("no executor configured"), "unknown", nil)
		return fmt.Errorf("no executor configured")
	}

	l.Info().Str("job_id", env.ID).Strs("regions", spec.Constraints.Regions).Msg("starting multi-region execution")
	
	results, err := w.executeAllRegions(ctx, env.ID, spec, spec.Constraints.Regions, executor)
	if err != nil {
		return err
	}
	
	return w.processExecutionResults(ctx, env.ID, spec, results)
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
	successRate := float64(successCount) / float64(len(results))
	minSuccessRate := spec.Constraints.MinSuccessRate
	if minSuccessRate == 0 {
		minSuccessRate = 0.67
	}
	
	jobStatus := "failed"
	if successRate >= minSuccessRate {
		jobStatus = "completed"
	}
	
	l.Info().Str("job_id", jobID).Int("successful", successCount).Int("failed", failureCount).Float64("success_rate", successRate).Msg("multi-region execution completed")
	
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
func (w *JobRunner) executeSingleRegion(ctx context.Context, jobID string, spec *models.JobSpec, region string, executor Executor) ExecutionResult {
	l := logging.FromContext(ctx)
	
	regionStart := time.Now()
	l.Info().Str("job_id", jobID).Str("region", region).Msg("starting region execution")
	
	// Execute job in this region
	providerID, status, outputJSON, receiptJSON, err := executor.Execute(ctx, spec, region)
	
	// Determine actual region for metrics (hybrid executor may map regions)
	actualRegion := region
	if w.Hybrid != nil {
		actualRegion = mapRegionToRouter(region)
	}
	
	regionEnd := time.Now()
	result := ExecutionResult{
		Region:      actualRegion,
		ProviderID:  providerID,
		Status:      status,
		OutputJSON:  outputJSON,
		ReceiptJSON: receiptJSON,
		Error:       err,
		StartedAt:   regionStart.UTC(),
		CompletedAt: regionEnd.UTC(),
	}
	
	// Handle execution result logging
	if err != nil {
		l.Error().Err(err).Str("job_id", jobID).Str("region", actualRegion).Msg("region execution failed")
		result.Status = "failed"
	} else {
		l.Info().Str("job_id", jobID).Str("provider", providerID).Str("region", actualRegion).Str("status", status).Msg("region execution successful")
	}
	
	// Insert execution record in database
	execID, insErr := w.ExecRepo.InsertExecution(ctx, spec.ID, providerID, actualRegion, result.Status, result.StartedAt, result.CompletedAt, outputJSON, receiptJSON)
	if insErr != nil {
		l.Error().Err(insErr).Str("job_id", jobID).Str("region", actualRegion).Msg("failed to insert execution record")
	} else {
		result.ExecutionID = execID
		l.Info().Str("job_id", jobID).Int64("execution_id", execID).Str("region", actualRegion).Msg("execution record created")
	}
	
	// Update metrics
	metrics.ExecutionDurationSeconds.WithLabelValues(actualRegion, result.Status).Observe(time.Since(regionStart).Seconds())
	
	// Best-effort: region verification
	if execID > 0 {
		go w.verifyRegionAsync(ctx, execID, actualRegion)
	}
	
	return result
}

