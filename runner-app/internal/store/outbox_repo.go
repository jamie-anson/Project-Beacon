package store

import (
	"context"
	"database/sql"
)

type OutboxRepo struct {
	DB *sql.DB
}

func NewOutboxRepo(db *sql.DB) *OutboxRepo {
	return &OutboxRepo{DB: db}
}

func (r *OutboxRepo) InsertTx(ctx context.Context, tx *sql.Tx, topic string, payload []byte) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO outbox (topic, payload) VALUES ($1, $2)`, topic, payload)
	return err
}

// FetchUnpublished returns id, topic, payload for rows not yet published
func (r *OutboxRepo) FetchUnpublished(ctx context.Context, limit int) (*sql.Rows, error) {
	if limit <= 0 {
		limit = 50
	}
	return r.DB.QueryContext(ctx, `
		SELECT id, topic, payload
		FROM outbox
		WHERE published_at IS NULL
		ORDER BY id ASC
		LIMIT $1
	`, limit)
}

func (r *OutboxRepo) MarkPublished(ctx context.Context, id int64) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE outbox SET published_at = NOW() WHERE id = $1`, id)
	return err
}

// GetUnpublishedStats returns count and oldest age of unpublished messages
func (r *OutboxRepo) GetUnpublishedStats(ctx context.Context) (count int, oldestAgeSeconds float64, err error) {
	row := r.DB.QueryRowContext(ctx, `
		SELECT 
			COUNT(*) as count,
			COALESCE(EXTRACT(EPOCH FROM (NOW() - MIN(created_at))), 0) as oldest_age_seconds
		FROM outbox 
		WHERE published_at IS NULL
	`)
	err = row.Scan(&count, &oldestAgeSeconds)
	return
}
