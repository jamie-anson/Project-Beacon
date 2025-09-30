package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/logging"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
)

// JobTimeoutService handles automatic timeout of stuck processing jobs
type JobTimeoutService struct {
	DB       *sql.DB
	JobsRepo *store.JobsRepo
}

// NewJobTimeoutService creates a new job timeout service
func NewJobTimeoutService(db *sql.DB) *JobTimeoutService {
	return &JobTimeoutService{
		DB:       db,
		JobsRepo: store.NewJobsRepo(db),
	}
}

// TimeoutStuckJobs finds jobs stuck in "processing" status and marks them as failed
// This prevents jobs from being stuck indefinitely and provides better UX
func (s *JobTimeoutService) TimeoutStuckJobs(ctx context.Context, timeoutThreshold time.Duration) error {
	l := logging.FromContext(ctx)
	
	// Find jobs in "processing" status that are older than threshold
	rows, err := s.DB.QueryContext(ctx, `
		SELECT jobspec_id, updated_at, created_at
		FROM jobs 
		WHERE status = 'processing' 
		AND updated_at < NOW() - $1::INTERVAL
		ORDER BY updated_at ASC
	`, fmt.Sprintf("%d seconds", int(timeoutThreshold.Seconds())))
	
	if err != nil {
		return err
	}
	defer rows.Close()
	
	var timedOutCount int
	for rows.Next() {
		var jobID string
		var updatedAt, createdAt time.Time
		
		if err := rows.Scan(&jobID, &updatedAt, &createdAt); err != nil {
			l.Error().Err(err).Msg("failed to scan stuck job row")
			continue
		}
		
		processingDuration := time.Since(updatedAt)
		totalDuration := time.Since(createdAt)
		
		l.Warn().
			Str("job_id", jobID).
			Dur("processing_duration", processingDuration).
			Dur("total_duration", totalDuration).
			Time("started_processing", updatedAt).
			Msg("timing out stuck processing job")
		
		// Mark job as failed with timeout reason
		if err := s.JobsRepo.UpdateJobStatus(ctx, jobID, "failed"); err != nil {
			l.Error().Err(err).Str("job_id", jobID).Msg("failed to timeout stuck job")
			continue
		}
		
		timedOutCount++
		l.Info().
			Str("job_id", jobID).
			Dur("processing_duration", processingDuration).
			Msg("stuck job marked as failed due to timeout")
	}
	
	if timedOutCount > 0 {
		l.Info().
			Int("timed_out_count", timedOutCount).
			Dur("timeout_threshold", timeoutThreshold).
			Msg("job timeout cleanup completed")
	} else {
		l.Debug().
			Dur("timeout_threshold", timeoutThreshold).
			Msg("no stuck processing jobs found")
	}
	
	return nil
}

// GetStuckJobsCount returns the number of jobs stuck in processing status
func (s *JobTimeoutService) GetStuckJobsCount(ctx context.Context, timeoutThreshold time.Duration) (int, error) {
	var count int
	err := s.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) 
		FROM jobs 
		WHERE status = 'processing' 
		AND updated_at < NOW() - $1::INTERVAL
	`, fmt.Sprintf("%d seconds", int(timeoutThreshold.Seconds()))).Scan(&count)
	
	return count, err
}

// GetStuckJobsDetails returns details of jobs stuck in processing status
func (s *JobTimeoutService) GetStuckJobsDetails(ctx context.Context, timeoutThreshold time.Duration) ([]StuckJobInfo, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT jobspec_id, updated_at, created_at
		FROM jobs 
		WHERE status = 'processing' 
		AND updated_at < NOW() - $1::INTERVAL
		ORDER BY updated_at ASC
	`, fmt.Sprintf("%d seconds", int(timeoutThreshold.Seconds())))
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var stuckJobs []StuckJobInfo
	for rows.Next() {
		var job StuckJobInfo
		if err := rows.Scan(&job.JobID, &job.UpdatedAt, &job.CreatedAt); err != nil {
			continue
		}
		job.ProcessingDuration = time.Since(job.UpdatedAt)
		job.TotalDuration = time.Since(job.CreatedAt)
		stuckJobs = append(stuckJobs, job)
	}
	
	return stuckJobs, nil
}

// StuckJobInfo contains information about a stuck job
type StuckJobInfo struct {
	JobID              string        `json:"job_id"`
	UpdatedAt          time.Time     `json:"updated_at"`
	CreatedAt          time.Time     `json:"created_at"`
	ProcessingDuration time.Duration `json:"processing_duration"`
	TotalDuration      time.Duration `json:"total_duration"`
}
