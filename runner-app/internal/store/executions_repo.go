package store

import (
	"context"
	"database/sql"
	"time"
)

// ExecutionsRepo provides persistence operations for executions
type ExecutionsRepo struct {
	DB *sql.DB
}

// GetLatestByJobSpecID returns the most recent execution for a given JobSpec ID
func (r *ExecutionsRepo) GetLatestByJobSpecID(
	ctx context.Context,
	jobspecID string,
) (
	id int64,
	providerID string,
	region string,
	status string,
	startedAt sql.NullTime,
	completedAt sql.NullTime,
	outputJSON []byte,
	receiptJSON []byte,
	createdAt time.Time,
	err error,
) {
	row := r.DB.QueryRowContext(ctx, `
		SELECT e.id, e.provider_id, e.region, e.status, e.started_at, e.completed_at, e.output_data, e.receipt_data, e.created_at
		FROM executions e
		JOIN jobs j ON e.job_id = j.id
		WHERE j.jobspec_id = $1
		ORDER BY e.created_at DESC
		LIMIT 1
	`, jobspecID)
	err = row.Scan(&id, &providerID, &region, &status, &startedAt, &completedAt, &outputJSON, &receiptJSON, &createdAt)
	return
}

func NewExecutionsRepo(db *sql.DB) *ExecutionsRepo {
	return &ExecutionsRepo{DB: db}
}

// InsertExecution inserts an execution row associated to a job via jobspec_id lookup
func (r *ExecutionsRepo) InsertExecution(
	ctx context.Context,
	jobspecID string,
	providerID string,
	region string,
	status string,
	startedAt time.Time,
	completedAt time.Time,
	outputJSON []byte,
	receiptJSON []byte,
) (int64, error) {
	row := r.DB.QueryRowContext(ctx, `
		INSERT INTO executions (job_id, provider_id, region, status, started_at, completed_at, output_data, receipt_data)
		VALUES ((SELECT id FROM jobs WHERE jobspec_id = $1), $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, jobspecID, providerID, region, status, startedAt, completedAt, outputJSON, receiptJSON)
	var id int64
	if err := row.Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}
