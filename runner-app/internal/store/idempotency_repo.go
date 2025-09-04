package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	// OpenTelemetry
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// IdempotencyRepo manages idempotency key mappings
type IdempotencyRepo struct {
	DB *sql.DB
}

func NewIdempotencyRepo(db *sql.DB) *IdempotencyRepo {
	return &IdempotencyRepo{DB: db}
}

// GetByKey returns the jobspec_id mapped to idemKey, if any
func (r *IdempotencyRepo) GetByKey(ctx context.Context, idemKey string) (string, bool, error) {
	tracer := otel.Tracer("runner/store/idempotency")
	ctx, span := tracer.Start(ctx, "IdempotencyRepo.GetByKey", oteltrace.WithAttributes(
		attribute.String("idempotency.key", idemKey),
	))
	defer span.End()
	if r.DB == nil {
		return "", false, errors.New("database connection is nil")
	}
	var jobID string
	err := r.DB.QueryRowContext(ctx, `SELECT jobspec_id FROM idempotency_keys WHERE idem_key = $1`, idemKey).Scan(&jobID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, fmt.Errorf("query idempotency key: %w", err)
	}
	return jobID, true, nil
}

// PutTx records a mapping from idemKey to jobspecID inside an existing transaction.
// Relies on a unique index on idem_key for safety.
func (r *IdempotencyRepo) PutTx(ctx context.Context, tx *sql.Tx, idemKey, jobspecID string) error {
	tracer := otel.Tracer("runner/store/idempotency")
	ctx, span := tracer.Start(ctx, "IdempotencyRepo.PutTx", oteltrace.WithAttributes(
		attribute.String("idempotency.key", idemKey),
		attribute.String("job.id", jobspecID),
	))
	defer span.End()
	if tx == nil {
		return errors.New("nil tx in PutTx")
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO idempotency_keys (idem_key, jobspec_id)
		VALUES ($1, $2)
		ON CONFLICT (idem_key) DO NOTHING
	`, idemKey, jobspecID)
	return err
}
