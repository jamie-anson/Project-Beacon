package service

import (
	"context"
	"database/sql"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/logging"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
)

// JobRecoveryService handles recovery of stale processing jobs on startup
type JobRecoveryService struct {
	DB       *sql.DB
	JobsRepo *store.JobsRepo
}

// NewJobRecoveryService creates a new job recovery service
func NewJobRecoveryService(db *sql.DB) *JobRecoveryService {
	return &JobRecoveryService{
		DB:       db,
		JobsRepo: store.NewJobsRepo(db),
	}
}

// RecoverStaleJobs finds jobs stuck in "processing" status and requeues them
// This handles crash recovery where jobs were being processed but never completed
func (s *JobRecoveryService) RecoverStaleJobs(ctx context.Context, staleThreshold time.Duration) error {
	l := logging.FromContext(ctx)
	
	// Find jobs in "processing" status that are older than threshold
	rows, err := s.DB.QueryContext(ctx, `
		SELECT jobspec_id, updated_at 
		FROM jobs 
		WHERE status = 'processing' 
		AND updated_at < NOW() - $1::INTERVAL
		ORDER BY updated_at ASC
	`, staleThreshold)
	
	if err != nil {
		return err
	}
	defer rows.Close()
	
	var recoveredCount int
	for rows.Next() {
		var jobID string
		var updatedAt time.Time
		
		if err := rows.Scan(&jobID, &updatedAt); err != nil {
			l.Error().Err(err).Msg("failed to scan stale job row")
			continue
		}
		
		staleDuration := time.Since(updatedAt)
		l.Info().
			Str("job_id", jobID).
			Dur("stale_duration", staleDuration).
			Msg("recovering stale processing job")
		
		// Reset job status to "created" so it can be reprocessed
		if err := s.JobsRepo.UpdateJobStatus(ctx, jobID, "created"); err != nil {
			l.Error().Err(err).Str("job_id", jobID).Msg("failed to reset stale job status")
			continue
		}
		
		recoveredCount++
		l.Info().Str("job_id", jobID).Msg("stale job reset to created status")
	}
	
	if recoveredCount > 0 {
		l.Info().Int("recovered_count", recoveredCount).Msg("stale job recovery completed")
	} else {
		l.Info().Msg("no stale processing jobs found")
	}
	
	return nil
}

// GetStaleJobsCount returns the number of jobs stuck in processing status
func (s *JobRecoveryService) GetStaleJobsCount(ctx context.Context, staleThreshold time.Duration) (int, error) {
	var count int
	err := s.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) 
		FROM jobs 
		WHERE status = 'processing' 
		AND updated_at < NOW() - $1::INTERVAL
	`, staleThreshold).Scan(&count)
	
	return count, err
}
