package recovery

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/errors"
)

// JobRecoveryStrategy defines how to handle failed jobs
type JobRecoveryStrategy int

const (
	// RetryStrategy retries the job with exponential backoff
	RetryStrategy JobRecoveryStrategy = iota

	// DeadLetterStrategy sends job to dead letter queue
	DeadLetterStrategy

	// FallbackStrategy uses alternative execution path
	FallbackStrategy

	// SkipStrategy marks job as failed and continues
	SkipStrategy
)

// JobRecoveryConfig defines job-specific recovery behavior
type JobRecoveryConfig struct {
	MaxRetries      int
	RetryDelay      time.Duration
	MaxRetryDelay   time.Duration
	Strategy        JobRecoveryStrategy
	FallbackRegions []string
	DeadLetterTTL   time.Duration
}

// DefaultJobRecoveryConfig returns sensible defaults for job recovery
func DefaultJobRecoveryConfig() JobRecoveryConfig {
	return JobRecoveryConfig{
		MaxRetries:      3,
		RetryDelay:      30 * time.Second,
		MaxRetryDelay:   10 * time.Minute,
		Strategy:        RetryStrategy,
		FallbackRegions: []string{"US", "EU"},
		DeadLetterTTL:   24 * time.Hour,
	}
}

// JobRecoveryManager handles job-specific error recovery
type JobRecoveryManager struct {
	recoveryManager *RecoveryManager
	logger          *slog.Logger
}

// NewJobRecoveryManager creates a new job recovery manager
func NewJobRecoveryManager(rm *RecoveryManager, logger *slog.Logger) *JobRecoveryManager {
	return &JobRecoveryManager{
		recoveryManager: rm,
		logger:          logger,
	}
}

// RecoverFailedJob attempts to recover a failed job execution
func (jrm *JobRecoveryManager) RecoverFailedJob(
	ctx context.Context,
	jobID string,
	originalError error,
	config JobRecoveryConfig,
	executeFunc func(context.Context, string) error,
) error {
	jrm.logger.Info("attempting job recovery",
		"job_id", jobID,
		"strategy", config.Strategy,
		"original_error", originalError)

	switch config.Strategy {
	case RetryStrategy:
		return jrm.retryJob(ctx, jobID, config, executeFunc)

	case DeadLetterStrategy:
		return jrm.sendToDeadLetter(ctx, jobID, originalError, config)

	case FallbackStrategy:
		return jrm.executeFallback(ctx, jobID, config, executeFunc)

	case SkipStrategy:
		return jrm.skipJob(ctx, jobID, originalError)

	default:
		return errors.Newf(errors.InternalError, "unknown recovery strategy: %v", config.Strategy)
	}
}

// retryJob implements retry logic with exponential backoff
func (jrm *JobRecoveryManager) retryJob(
	ctx context.Context,
	jobID string,
	config JobRecoveryConfig,
	executeFunc func(context.Context, string) error,
) error {
	retryConfig := RetryConfig{
		MaxAttempts:  config.MaxRetries,
		InitialDelay: config.RetryDelay,
		MaxDelay:     config.MaxRetryDelay,
		Multiplier:   2.0,
		Jitter:       true,
	}

	return jrm.recoveryManager.ExecuteWithRecovery(
		ctx,
		fmt.Sprintf("job_retry_%s", jobID),
		func(ctx context.Context) error {
			return executeFunc(ctx, jobID)
		},
		retryConfig,
	)
}

// sendToDeadLetter moves the job to dead letter queue for manual inspection
func (jrm *JobRecoveryManager) sendToDeadLetter(
	ctx context.Context,
	jobID string,
	originalError error,
	config JobRecoveryConfig,
) error {
	jrm.logger.Error("sending job to dead letter queue",
		"job_id", jobID,
		"original_error", originalError,
		"ttl", config.DeadLetterTTL)

	// In a real implementation, this would interact with your queue system
	// For now, we'll just log the action
	return errors.Newf(errors.InternalError,
		"job %s sent to dead letter queue due to: %v", jobID, originalError)
}

// executeFallback attempts execution with fallback regions
func (jrm *JobRecoveryManager) executeFallback(
	ctx context.Context,
	jobID string,
	config JobRecoveryConfig,
	executeFunc func(context.Context, string) error,
) error {
	jrm.logger.Info("attempting fallback execution",
		"job_id", jobID,
		"fallback_regions", config.FallbackRegions)

	// Try fallback regions one by one
	for _, region := range config.FallbackRegions {
		jrm.logger.Info("trying fallback region",
			"job_id", jobID,
			"region", region)

		err := jrm.recoveryManager.ExecuteWithRecovery(
			ctx,
			fmt.Sprintf("job_fallback_%s_%s", jobID, region),
			func(ctx context.Context) error {
				// In a real implementation, this would set the region context
				// and then execute the job
				return executeFunc(ctx, jobID)
			},
			DefaultRetryConfig(),
		)

		if err == nil {
			jrm.logger.Info("fallback execution succeeded",
				"job_id", jobID,
				"region", region)
			return nil
		}

		jrm.logger.Warn("fallback region failed",
			"job_id", jobID,
			"region", region,
			"error", err)
	}

	return errors.Newf(errors.ExternalServiceError,
		"all fallback regions failed for job %s", jobID)
}

// skipJob marks the job as failed and continues processing
func (jrm *JobRecoveryManager) skipJob(
	ctx context.Context,
	jobID string,
	originalError error,
) error {
	jrm.logger.Warn("skipping failed job",
		"job_id", jobID,
		"original_error", originalError)

	// In a real implementation, this would update the job status to failed
	return nil
}

// RecoverStuckJobs identifies and recovers jobs that have been stuck for too long
func (jrm *JobRecoveryManager) RecoverStuckJobs(
	ctx context.Context,
	stuckThreshold time.Duration,
	getStuckJobsFunc func(context.Context, time.Duration) ([]string, error),
	executeFunc func(context.Context, string) error,
) error {
	stuckJobs, err := getStuckJobsFunc(ctx, stuckThreshold)
	if err != nil {
		return errors.Wrap(err, errors.DatabaseError, "failed to get stuck jobs")
	}

	if len(stuckJobs) == 0 {
		return nil
	}

	jrm.logger.Info("found stuck jobs for recovery",
		"count", len(stuckJobs),
		"threshold", stuckThreshold)

	config := DefaultJobRecoveryConfig()
	config.Strategy = RetryStrategy
	config.MaxRetries = 1 // Only retry once for stuck jobs

	for _, jobID := range stuckJobs {
		err := jrm.RecoverFailedJob(ctx, jobID,
			errors.NewTimeoutError("job stuck"), config, executeFunc)

		if err != nil {
			jrm.logger.Error("failed to recover stuck job",
				"job_id", jobID,
				"error", err)
		}
	}

	return nil
}

// GetRecoveryMetrics returns metrics about job recovery operations
func (jrm *JobRecoveryManager) GetRecoveryMetrics() map[string]interface{} {
	circuitStats := jrm.recoveryManager.GetCircuitBreakerStats()

	metrics := map[string]interface{}{
		"circuit_breakers": make(map[string]interface{}),
		"timestamp":        time.Now(),
	}

	cbMetrics := make(map[string]interface{})
	for _, stat := range circuitStats {
		cbMetrics[stat.Name] = map[string]interface{}{
			"state":          stat.State.String(),
			"failures":       stat.Failures,
			"successes":      stat.Successes,
			"requests":       stat.Requests,
			"last_fail_time": stat.LastFailTime,
		}
	}
	metrics["circuit_breakers"] = cbMetrics

	return metrics
}
