package db

import (
	"context"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool holds a shared pgx connection pool. Opt-in; existing code can continue using database/sql via db.DB.
var Pool *pgxpool.Pool

// InitPool initializes a pgxpool.Pool if DATABASE_URL is set. Safe to call multiple times.
func InitPool(ctx context.Context) (*pgxpool.Pool, error) {
	if Pool != nil {
		return Pool, nil
	}
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return nil, nil
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	// Sensible defaults; can be env-driven later
	cfg.MaxConns = 10
	cfg.MinConns = 0
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 30 * time.Second
	p, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	// Ping to verify connectivity
	ctxPing, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := p.Ping(ctxPing); err != nil {
		p.Close()
		return nil, err
	}
	Pool = p
	return Pool, nil
}
