package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"

	// OpenTelemetry
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// JobsRepo provides persistence operations for jobs
type JobsRepo struct {
	DB *sql.DB
}

func NewJobsRepo(db *sql.DB) *JobsRepo {
	return &JobsRepo{DB: db}
}

// CreateJob inserts a new job with validation
func (r *JobsRepo) CreateJob(ctx context.Context, jobspec *models.JobSpec) error {
	tracer := otel.Tracer("runner/store/jobs")
	ctx, span := tracer.Start(ctx, "JobsRepo.CreateJob", oteltrace.WithAttributes(
		attribute.String("job.id", jobspec.ID),
	))
	defer span.End()
	if r.DB == nil {
		return errors.New("database connection is nil")
	}

	// Serialize JobSpec to JSON
	jobspecData, err := json.Marshal(jobspec)
	if err != nil {
		return fmt.Errorf("failed to marshal jobspec: %w", err)
	}

	_, err = r.DB.ExecContext(ctx, `
		INSERT INTO jobs (jobspec_id, jobspec_data, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`, jobspec.ID, jobspecData, "queued", jobspec.CreatedAt, time.Now())

	if err != nil {
		return fmt.Errorf("failed to insert job: %w", err)
	}
	return nil
}

// UpsertJobTx inserts/updates a job row inside an existing transaction
func (r *JobsRepo) UpsertJobTx(ctx context.Context, tx *sql.Tx, jobspecID string, jobspecData []byte, status string) error {
	tracer := otel.Tracer("runner/store/jobs")
	ctx, span := tracer.Start(ctx, "JobsRepo.UpsertJobTx", oteltrace.WithAttributes(
		attribute.String("job.id", jobspecID),
		attribute.String("job.status", status),
	))
	defer span.End()
	if tx == nil {
		return errors.New("nil tx in UpsertJobTx")
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO jobs (jobspec_id, jobspec_data, status)
		VALUES ($1, $2, $3)
		ON CONFLICT (jobspec_id)
		DO UPDATE SET jobspec_data = EXCLUDED.jobspec_data, status = EXCLUDED.status, updated_at = NOW()
	`, jobspecID, jobspecData, status)
	return err
}

// GetJobByID returns a complete JobSpec by ID
func (r *JobsRepo) GetJobByID(ctx context.Context, jobspecID string) (*models.JobSpec, string, error) {
	tracer := otel.Tracer("runner/store/jobs")
	ctx, span := tracer.Start(ctx, "JobsRepo.GetJobByID", oteltrace.WithAttributes(
		attribute.String("job.id", jobspecID),
	))
	defer span.End()
	if r.DB == nil {
		return nil, "", errors.New("database connection is nil")
	}

	var jobspecData []byte
	var status string
	var createdAt, updatedAt time.Time

	row := r.DB.QueryRowContext(ctx, `
		SELECT jobspec_data, status, created_at, updated_at 
		FROM jobs 
		WHERE jobspec_id = $1
	`, jobspecID)

	err := row.Scan(&jobspecData, &status, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, "", fmt.Errorf("job not found: %s", jobspecID)
		}
		return nil, "", fmt.Errorf("failed to get job: %w", err)
	}

	// Deserialize JobSpec
	var jobspec models.JobSpec
	if err := json.Unmarshal(jobspecData, &jobspec); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal jobspec: %w", err)
	}

	return &jobspec, status, nil
}

// GetJob returns the stored job fields (legacy method)
func (r *JobsRepo) GetJob(ctx context.Context, jobspecID string) (id string, status string, data []byte, createdAt, updatedAt sql.NullTime, err error) {
	row := r.DB.QueryRowContext(ctx, `SELECT jobspec_id, status, jobspec_data, created_at, updated_at FROM jobs WHERE jobspec_id = $1`, jobspecID)
	err = row.Scan(&id, &status, &data, &createdAt, &updatedAt)
	return
}

// ListRecentJobs lists recent jobs (simple placeholder, limit 50)
func (r *JobsRepo) ListRecentJobs(ctx context.Context, limit int) (*sql.Rows, error) {
	tracer := otel.Tracer("runner/store/jobs")
	ctx, span := tracer.Start(ctx, "JobsRepo.ListRecentJobs", oteltrace.WithAttributes(
		attribute.Int("limit", limit),
	))
	defer span.End()
	if limit <= 0 {
		limit = 50
	}
	return r.DB.QueryContext(ctx, `SELECT jobspec_id, status, created_at FROM jobs ORDER BY created_at DESC LIMIT $1`, limit)
}

// UpdateJobStatus updates the status of a job
func (r *JobsRepo) UpdateJobStatus(ctx context.Context, jobspecID string, status string) error {
	tracer := otel.Tracer("runner/store/jobs")
	ctx, span := tracer.Start(ctx, "JobsRepo.UpdateJobStatus", oteltrace.WithAttributes(
		attribute.String("job.id", jobspecID),
		attribute.String("job.status", status),
	))
	defer span.End()
	if r.DB == nil {
		return errors.New("database connection is nil")
	}

	result, err := r.DB.ExecContext(ctx, `
		UPDATE jobs 
		SET status = $1, updated_at = NOW() 
		WHERE jobspec_id = $2
	`, status, jobspecID)

	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("job not found: %s", jobspecID)
	}

	return nil
}

// ListJobsByStatus returns jobs with a specific status
func (r *JobsRepo) ListJobsByStatus(ctx context.Context, status string, limit int) ([]*models.JobSpec, error) {
	tracer := otel.Tracer("runner/store/jobs")
	ctx, span := tracer.Start(ctx, "JobsRepo.ListJobsByStatus", oteltrace.WithAttributes(
		attribute.String("job.status", status),
		attribute.Int("limit", limit),
	))
	defer span.End()
	if r.DB == nil {
		return nil, errors.New("database connection is nil")
	}

	if limit <= 0 {
		limit = 50
	}

	rows, err := r.DB.QueryContext(ctx, `
		SELECT jobspec_data 
		FROM jobs 
		WHERE status = $1 
		ORDER BY created_at DESC 
		LIMIT $2
	`, status, limit)

	if err != nil {
		return nil, fmt.Errorf("failed to query jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*models.JobSpec
	for rows.Next() {
		var jobspecData []byte
		if err := rows.Scan(&jobspecData); err != nil {
			return nil, fmt.Errorf("failed to scan job row: %w", err)
		}

		var jobspec models.JobSpec
		if err := json.Unmarshal(jobspecData, &jobspec); err != nil {
			return nil, fmt.Errorf("failed to unmarshal jobspec: %w", err)
		}

		jobs = append(jobs, &jobspec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating job rows: %w", err)
	}

	return jobs, nil
}

// DeleteJob deletes a job by jobspec_id
func (r *JobsRepo) DeleteJob(ctx context.Context, jobspecID string) error {
	tracer := otel.Tracer("runner/store/jobs")
	ctx, span := tracer.Start(ctx, "JobsRepo.DeleteJob", oteltrace.WithAttributes(
		attribute.String("job.id", jobspecID),
	))
	defer span.End()
	_, err := r.DB.ExecContext(ctx, `DELETE FROM jobs WHERE jobspec_id = $1`, jobspecID)
	return err
}
