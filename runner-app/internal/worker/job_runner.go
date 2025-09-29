package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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
	UpdateRegionVerification(ctx context.Context, executionID int64, regionClaimed sql.NullString, regionObserved sql.NullString, regionVerified sql.NullBool, verificationMethod sql.NullString, evidenceRef sql.NullString) error
}

// ProbeFactory constructs a preflight probe; injectable for tests.
type ProbeFactory func() negotiation.PreflightProbe

type JobRunner struct {
	DB            *sql.DB
	Queue         *queue.Client
	QueueName     string
	JobsRepo      jobsRepoIface
	ExecRepo      execRepoIface
	ExecutionSvc  *service.ExecutionService
	Golem         *golem.Service
	Hybrid        *hybrid.Client
	Bundler       *ipfs.Bundler
	ProbeFactory  ProbeFactory
	Executor      Executor
	WSHub         *websocket.Hub
	maxConcurrent int // Maximum concurrent executions for bounded concurrency
}

func NewJobRunner(db *sql.DB, q *queue.Client, gsvc *golem.Service, bundler *ipfs.Bundler) *JobRunner {
	jr := &JobRunner{
		DB:            db,
		Queue:         q,
		JobsRepo:      store.NewJobsRepo(db),
		ExecRepo:      store.NewExecutionsRepo(db),
		Golem:         gsvc,
		Bundler:       bundler,
		ExecutionSvc:  service.NewExecutionService(db),
		maxConcurrent: 10, // Default bounded concurrency limit
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
	
	// Check if this is a multi-model job
	if len(spec.Models) > 0 {
		l.Info().Str("job_id", env.ID).Int("model_count", len(spec.Models)).Msg("executing multi-model job")
		results, err := w.executeMultiModelJob(ctx, env.ID, spec, executor)
		if err != nil {
			return err
		}
		return w.processExecutionResults(ctx, env.ID, spec, results)
	} else {
		// Single model execution (legacy)
		results, err := w.executeAllRegions(ctx, env.ID, spec, spec.Constraints.Regions, executor)
		if err != nil {
			return err
		}
		return w.processExecutionResults(ctx, env.ID, spec, results)
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
	ModelID     string // NEW: Model ID for multi-model support
}

// executeMultiModelJob executes a job across multiple models and regions with bounded concurrency
func (w *JobRunner) executeMultiModelJob(ctx context.Context, jobID string, spec *models.JobSpec, executor Executor) ([]ExecutionResult, error) {
	l := logging.FromContext(ctx)
	
	// Parallel execution coordination with bounded concurrency
	var wg sync.WaitGroup
	var resultsMu sync.Mutex
	var results []ExecutionResult
	sem := make(chan struct{}, w.maxConcurrent) // Semaphore for bounded concurrency
	
	totalExecutions := 0
	for _, model := range spec.Models {
		totalExecutions += len(model.Regions)
	}
	
	l.Info().Str("job_id", jobID).Int("model_count", len(spec.Models)).Int("total_executions", totalExecutions).Int("max_concurrent", w.maxConcurrent).Msg("starting multi-model parallel execution")
	
	// Execute each model in each of its regions with bounded concurrency
	for _, model := range spec.Models {
		for _, region := range model.Regions {
			wg.Add(1)
			sem <- struct{}{} // Acquire semaphore slot
			go func(m models.ModelSpec, r string) {
				defer wg.Done()
				defer func() { <-sem }() // Release semaphore slot
				
				// Create a modified spec for this specific model
				// Use shallow copy and be careful with shared pointers
				modelSpec := *spec
				
				// Ensure metadata map is not shared between goroutines
				if modelSpec.Metadata == nil {
					modelSpec.Metadata = make(map[string]interface{})
				} else {
					// Create a new map to avoid concurrent map writes
					newMetadata := make(map[string]interface{})
					for k, v := range spec.Metadata {
						newMetadata[k] = v
					}
					modelSpec.Metadata = newMetadata
				}
				
				// For HybridExecutor, set metadata only (router will route by model_id)
				modelSpec.Metadata["model_id"] = m.ID
				modelSpec.Metadata["model_name"] = m.Name
				
				// Optional: For GolemExecutor, set container image if needed
				if m.ContainerImage != "" {
					modelSpec.Benchmark.Container.Image = m.ContainerImage
				}
				
				result := w.executeSingleRegion(ctx, jobID, &modelSpec, r, executor)
				result.ModelID = m.ID // Add model ID to result
				
				// Thread-safe result collection
				resultsMu.Lock()
				results = append(results, result)
				resultsMu.Unlock()
				
				l.Debug().Str("job_id", jobID).Str("model_id", m.ID).Str("region", r).Str("status", result.Status).Msg("model-region execution completed")
			}(model, region)
		}
	}
	
	// Wait for all model-region combinations to complete
	wg.Wait()
	
	l.Info().Str("job_id", jobID).Int("results_count", len(results)).Msg("multi-model execution completed")
	
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
	
	// Handle execution result logging and metrics
	if err != nil {
		l.Error().Err(err).Str("job_id", jobID).Str("region", actualRegion).Msg("region execution failed")
		result.Status = "failed"
		
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
		metrics.RunnerFailuresTotal.WithLabelValues(actualRegion, errorType, component).Inc()
		
		// Broadcast failure event via WebSocket
		if w.WSHub != nil {
			failureEvent := map[string]interface{}{
				"job_id": jobID,
				"region": actualRegion,
				"error_type": errorType,
				"component": component,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
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
		l.Info().Str("job_id", jobID).Str("provider", providerID).Str("region", actualRegion).Str("status", status).Msg("region execution successful")
	}
	
	// Extract model ID from spec metadata (for multi-model jobs) or use default
	modelID := "llama3.2-1b" // default
	if spec.Metadata != nil {
		if mid, ok := spec.Metadata["model_id"].(string); ok && mid != "" {
			modelID = mid
		}
	}
	
	// Insert execution record in database with model ID
	execID, insErr := w.ExecRepo.InsertExecutionWithModel(ctx, spec.ID, providerID, actualRegion, result.Status, result.StartedAt, result.CompletedAt, outputJSON, receiptJSON, modelID)
	if insErr != nil {
		l.Error().Err(insErr).Str("job_id", jobID).Str("region", actualRegion).Str("model_id", modelID).Msg("failed to insert execution record")
	} else {
		result.ExecutionID = execID
		result.ModelID = modelID // Set model ID in result
		l.Info().Str("job_id", jobID).Int64("execution_id", execID).Str("region", actualRegion).Str("model_id", modelID).Msg("execution record created")
		
		// NEW: Calculate and store bias scores for successful executions
		if result.Status == "completed" && w.ExecutionSvc != nil && w.ExecutionSvc.BiasScorer != nil {
			go w.calculateBiasScoreAsync(ctx, execID, spec, outputJSON, providerID)
		}
	}
	
	// Update metrics
	metrics.ExecutionDurationSeconds.WithLabelValues(actualRegion, result.Status).Observe(time.Since(regionStart).Seconds())
	
	// Best-effort: region verification
	if execID > 0 {
		go w.verifyRegionAsync(ctx, execID, actualRegion)
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

