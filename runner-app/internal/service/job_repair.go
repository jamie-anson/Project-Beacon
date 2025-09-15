package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
	
	// OpenTelemetry
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Job represents a job record for repair operations
type Job struct {
	ID           string     `json:"id"`
	JobSpecID    string     `json:"jobspec_id"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
}

// JobRepairService handles job status inconsistencies and repairs
type JobRepairService struct {
	jobsService *JobsService
}

// NewJobRepairService creates a new job repair service
func NewJobRepairService(jobsService *JobsService) *JobRepairService {
	return &JobRepairService{
		jobsService: jobsService,
	}
}

// RepairResult represents the result of a job repair operation
type RepairResult struct {
	JobID          string    `json:"job_id"`
	PreviousStatus string    `json:"previous_status"`
	NewStatus      string    `json:"new_status"`
	Action         string    `json:"action"`
	Reason         string    `json:"reason"`
	RepairedAt     time.Time `json:"repaired_at"`
}

// RepairSummary summarizes the results of a repair operation
type RepairSummary struct {
	TotalJobs        int           `json:"total_jobs"`
	RepairedJobs     int           `json:"repaired_jobs"`
	SkippedJobs      int           `json:"skipped_jobs"`
	ErrorJobs        int           `json:"error_jobs"`
	RepairResults    []RepairResult `json:"repair_results"`
	StartTime        time.Time     `json:"start_time"`
	EndTime          time.Time     `json:"end_time"`
	Duration         time.Duration `json:"duration"`
}

// RepairStuckJobs identifies and repairs jobs stuck in inconsistent states
func (r *JobRepairService) RepairStuckJobs(ctx context.Context, maxAge time.Duration) (*RepairSummary, error) {
	tracer := otel.Tracer("runner/service/job-repair")
	ctx, span := tracer.Start(ctx, "JobRepairService.RepairStuckJobs", oteltrace.WithAttributes(
		attribute.String("max_age", maxAge.String()),
	))
	defer span.End()

	startTime := time.Now()
	summary := &RepairSummary{
		StartTime:     startTime,
		RepairResults: make([]RepairResult, 0),
	}

	// Find jobs that are potentially stuck
	stuckJobs, err := r.findStuckJobs(ctx, maxAge)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to find stuck jobs: %w", err)
	}

	summary.TotalJobs = len(stuckJobs)
	log.Printf("Found %d potentially stuck jobs for repair", len(stuckJobs))

	for _, job := range stuckJobs {
		result, err := r.repairJob(ctx, job)
		if err != nil {
			log.Printf("Failed to repair job %s: %v", job.ID, err)
			summary.ErrorJobs++
			continue
		}

		if result != nil {
			summary.RepairResults = append(summary.RepairResults, *result)
			summary.RepairedJobs++
		} else {
			summary.SkippedJobs++
		}
	}

	summary.EndTime = time.Now()
	summary.Duration = summary.EndTime.Sub(summary.StartTime)

	span.SetAttributes(
		attribute.Int("total_jobs", summary.TotalJobs),
		attribute.Int("repaired_jobs", summary.RepairedJobs),
		attribute.Int("skipped_jobs", summary.SkippedJobs),
		attribute.Int("error_jobs", summary.ErrorJobs),
	)

	log.Printf("Job repair completed: %d total, %d repaired, %d skipped, %d errors in %v",
		summary.TotalJobs, summary.RepairedJobs, summary.SkippedJobs, summary.ErrorJobs, summary.Duration)

	return summary, nil
}

// findStuckJobs identifies jobs that are potentially stuck in inconsistent states
func (r *JobRepairService) findStuckJobs(ctx context.Context, maxAge time.Duration) ([]*Job, error) {
	cutoffTime := time.Now().Add(-maxAge)
	
	// Find jobs that are:
	// 1. In "created" status for too long (should have been processed)
	// 2. In "running" status for too long (might be stuck)
	// 3. Have inconsistent outbox entries
	
	query := `
		SELECT id, jobspec_id, status, created_at, updated_at, started_at, completed_at, error_message
		FROM jobs 
		WHERE (
			(status = 'created' AND created_at < $1) OR
			(status = 'running' AND started_at < $2) OR
			(status = 'running' AND started_at IS NULL)
		)
		ORDER BY created_at ASC
		LIMIT 100
	`
	
	rows, err := r.jobsService.DB.QueryContext(ctx, query, cutoffTime, cutoffTime.Add(-30*time.Minute))
	if err != nil {
		return nil, fmt.Errorf("failed to query stuck jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		job := &Job{}
		var startedAt, completedAt sql.NullTime
		var errorMessage sql.NullString

		err := rows.Scan(
			&job.ID,
			&job.JobSpecID,
			&job.Status,
			&job.CreatedAt,
			&job.UpdatedAt,
			&startedAt,
			&completedAt,
			&errorMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job row: %w", err)
		}

		if startedAt.Valid {
			job.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			job.CompletedAt = &completedAt.Time
		}
		if errorMessage.Valid {
			job.ErrorMessage = errorMessage.String
		}

		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

// repairJob attempts to repair a single job's inconsistent state
func (r *JobRepairService) repairJob(ctx context.Context, job *Job) (*RepairResult, error) {
	tracer := otel.Tracer("runner/service/job-repair")
	ctx, span := tracer.Start(ctx, "JobRepairService.repairJob", oteltrace.WithAttributes(
		attribute.String("job.id", job.ID),
		attribute.String("job.status", job.Status),
	))
	defer span.End()

	previousStatus := job.Status
	now := time.Now()

	// Determine the appropriate repair action based on job state
	var action, reason, newStatus string

	switch job.Status {
	case "created":
		// Job stuck in created state - likely missing outbox entry
		if time.Since(job.CreatedAt) > 10*time.Minute {
			action = "republish"
			reason = "Job stuck in created state, republishing to outbox"
			newStatus = "created" // Status stays the same, but we republish
			
			// Republish the job to the outbox
			if err := r.jobsService.RepublishJob(ctx, job.ID); err != nil {
				span.RecordError(err)
				return nil, fmt.Errorf("failed to republish job %s: %w", job.ID, err)
			}
		} else {
			// Job is recent, skip repair
			return nil, nil
		}

	case "running":
		if job.StartedAt == nil {
			// Job marked as running but no start time - inconsistent state
			action = "fix_status"
			reason = "Job marked as running but missing start time, resetting to created"
			newStatus = "created"
			
			// Reset job to created status and republish
			if err := r.resetJobToCreated(ctx, job.ID); err != nil {
				span.RecordError(err)
				return nil, fmt.Errorf("failed to reset job %s to created: %w", job.ID, err)
			}
			
			// Republish to ensure it gets processed
			if err := r.jobsService.RepublishJob(ctx, job.ID); err != nil {
				span.RecordError(err)
				return nil, fmt.Errorf("failed to republish reset job %s: %w", job.ID, err)
			}
			
		} else if time.Since(*job.StartedAt) > 30*time.Minute {
			// Job running for too long - likely stuck or crashed
			action = "timeout_reset"
			reason = "Job running for too long, resetting to created for retry"
			newStatus = "created"
			
			// Reset job to created status and republish
			if err := r.resetJobToCreated(ctx, job.ID); err != nil {
				span.RecordError(err)
				return nil, fmt.Errorf("failed to reset timed out job %s: %w", job.ID, err)
			}
			
			// Republish to ensure it gets processed
			if err := r.jobsService.RepublishJob(ctx, job.ID); err != nil {
				span.RecordError(err)
				return nil, fmt.Errorf("failed to republish timed out job %s: %w", job.ID, err)
			}
		} else {
			// Job is running normally, skip repair
			return nil, nil
		}

	default:
		// Other statuses (completed, failed) don't need repair
		return nil, nil
	}

	result := &RepairResult{
		JobID:          job.ID,
		PreviousStatus: previousStatus,
		NewStatus:      newStatus,
		Action:         action,
		Reason:         reason,
		RepairedAt:     now,
	}

	span.SetAttributes(
		attribute.String("repair.action", action),
		attribute.String("repair.reason", reason),
		attribute.String("repair.new_status", newStatus),
	)

	log.Printf("Repaired job %s: %s -> %s (%s)", job.ID, previousStatus, newStatus, reason)
	return result, nil
}

// resetJobToCreated resets a job's status to created and clears runtime fields
func (r *JobRepairService) resetJobToCreated(ctx context.Context, jobID string) error {
	query := `
		UPDATE jobs 
		SET status = 'created', 
		    started_at = NULL, 
		    completed_at = NULL, 
		    error_message = NULL,
		    updated_at = NOW()
		WHERE id = $1
	`
	
	_, err := r.jobsService.DB.ExecContext(ctx, query, jobID)
	if err != nil {
		return fmt.Errorf("failed to reset job %s to created: %w", jobID, err)
	}
	
	return nil
}

// GetStuckJobsStats returns statistics about potentially stuck jobs
func (r *JobRepairService) GetStuckJobsStats(ctx context.Context) (map[string]interface{}, error) {
	tracer := otel.Tracer("runner/service/job-repair")
	ctx, span := tracer.Start(ctx, "JobRepairService.GetStuckJobsStats")
	defer span.End()

	stats := make(map[string]interface{})
	
	// Count jobs by status and age
	queries := map[string]string{
		"created_over_10min": `
			SELECT COUNT(*) FROM jobs 
			WHERE status = 'created' AND created_at < NOW() - INTERVAL '10 minutes'
		`,
		"running_over_30min": `
			SELECT COUNT(*) FROM jobs 
			WHERE status = 'running' AND started_at < NOW() - INTERVAL '30 minutes'
		`,
		"running_no_start_time": `
			SELECT COUNT(*) FROM jobs 
			WHERE status = 'running' AND started_at IS NULL
		`,
		"total_created": `
			SELECT COUNT(*) FROM jobs WHERE status = 'created'
		`,
		"total_running": `
			SELECT COUNT(*) FROM jobs WHERE status = 'running'
		`,
	}

	for key, query := range queries {
		var count int
		err := r.jobsService.DB.QueryRowContext(ctx, query).Scan(&count)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("failed to get %s count: %w", key, err)
		}
		stats[key] = count
	}

	return stats, nil
}
