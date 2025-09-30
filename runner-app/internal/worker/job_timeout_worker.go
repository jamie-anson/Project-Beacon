package worker

import (
	"context"
	"database/sql"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/logging"
	"github.com/jamie-anson/project-beacon-runner/internal/service"
)

// JobTimeoutWorker runs periodic cleanup of stuck processing jobs
type JobTimeoutWorker struct {
	timeoutService   *service.JobTimeoutService
	timeoutThreshold time.Duration
	checkInterval    time.Duration
	stopCh           chan struct{}
}

// NewJobTimeoutWorker creates a new job timeout worker
func NewJobTimeoutWorker(db *sql.DB, timeoutThreshold, checkInterval time.Duration) *JobTimeoutWorker {
	return &JobTimeoutWorker{
		timeoutService:   service.NewJobTimeoutService(db),
		timeoutThreshold: timeoutThreshold,
		checkInterval:    checkInterval,
		stopCh:           make(chan struct{}),
	}
}

// Start begins the periodic timeout checking
func (w *JobTimeoutWorker) Start(ctx context.Context) {
	l := logging.FromContext(ctx)
	
	l.Info().
		Dur("timeout_threshold", w.timeoutThreshold).
		Dur("check_interval", w.checkInterval).
		Msg("starting job timeout worker")
	
	ticker := time.NewTicker(w.checkInterval)
	defer ticker.Stop()
	
	// Run initial check
	w.checkTimeouts(ctx)
	
	for {
		select {
		case <-ticker.C:
			w.checkTimeouts(ctx)
		case <-w.stopCh:
			l.Info().Msg("job timeout worker stopped")
			return
		case <-ctx.Done():
			l.Info().Msg("job timeout worker stopped due to context cancellation")
			return
		}
	}
}

// Stop stops the timeout worker
func (w *JobTimeoutWorker) Stop() {
	close(w.stopCh)
}

// checkTimeouts performs the actual timeout check and cleanup
func (w *JobTimeoutWorker) checkTimeouts(ctx context.Context) {
	l := logging.FromContext(ctx)
	
	// First, get count of stuck jobs for logging
	count, err := w.timeoutService.GetStuckJobsCount(ctx, w.timeoutThreshold)
	if err != nil {
		l.Error().Err(err).Msg("failed to get stuck jobs count")
		return
	}
	
	if count == 0 {
		l.Debug().Msg("no stuck jobs found during timeout check")
		return
	}
	
	l.Info().
		Int("stuck_jobs_count", count).
		Dur("timeout_threshold", w.timeoutThreshold).
		Msg("found stuck jobs, starting timeout cleanup")
	
	// Get details for logging before cleanup
	stuckJobs, err := w.timeoutService.GetStuckJobsDetails(ctx, w.timeoutThreshold)
	if err != nil {
		l.Error().Err(err).Msg("failed to get stuck jobs details")
	} else {
		for _, job := range stuckJobs {
			l.Warn().
				Str("job_id", job.JobID).
				Dur("processing_duration", job.ProcessingDuration).
				Dur("total_duration", job.TotalDuration).
				Time("last_updated", job.UpdatedAt).
				Msg("job will be timed out")
		}
	}
	
	// Perform timeout cleanup
	if err := w.timeoutService.TimeoutStuckJobs(ctx, w.timeoutThreshold); err != nil {
		l.Error().Err(err).Msg("failed to timeout stuck jobs")
		return
	}
	
	l.Info().Msg("job timeout cleanup completed successfully")
}
