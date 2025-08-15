package store

import (
	"context"
	"database/sql"
	"errors"
)

// JobsRepo provides persistence operations for jobs
 type JobsRepo struct {
	DB *sql.DB
 }

func NewJobsRepo(db *sql.DB) *JobsRepo {
	return &JobsRepo{DB: db}
}

// UpsertJobTx inserts/updates a job row inside an existing transaction
func (r *JobsRepo) UpsertJobTx(ctx context.Context, tx *sql.Tx, jobspecID string, jobspecData []byte, status string) error {
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

// GetJob returns the stored job fields
func (r *JobsRepo) GetJob(ctx context.Context, jobspecID string) (id string, status string, data []byte, createdAt, updatedAt sql.NullTime, err error) {
	row := r.DB.QueryRowContext(ctx, `SELECT jobspec_id, status, jobspec_data, created_at, updated_at FROM jobs WHERE jobspec_id = $1`, jobspecID)
	err = row.Scan(&id, &status, &data, &createdAt, &updatedAt)
	return
}

// ListRecentJobs lists recent jobs (simple placeholder, limit 50)
func (r *JobsRepo) ListRecentJobs(ctx context.Context, limit int) (*sql.Rows, error) {
	if limit <= 0 {
		limit = 50
	}
	return r.DB.QueryContext(ctx, `SELECT jobspec_id, status, created_at FROM jobs ORDER BY created_at DESC LIMIT $1`, limit)
}

// DeleteJob deletes a job by jobspec_id
func (r *JobsRepo) DeleteJob(ctx context.Context, jobspecID string) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM jobs WHERE jobspec_id = $1`, jobspecID)
	return err
}
