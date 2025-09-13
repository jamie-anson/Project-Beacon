package db

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
)

type DB struct {
	*sql.DB
}

// runWithGolangMigrate runs migrations from the given path using golang-migrate.
// path should be a directory containing versioned *.up.sql and *.down.sql files.
func runWithGolangMigrate(dbURL, path string) error {
    src := "file://" + path
    m, err := migrate.New(src, dbURL)
    if err != nil {
        return fmt.Errorf("migrate init: %w", err)
    }
    if err := m.Up(); err != nil && err.Error() != "no change" {
        return err
    }
    return nil
}

func Initialize(dbURL string) (*DB, error) {
	if dbURL == "" {
		dbURL = "postgres://postgres:password@localhost:5433/beacon_runner?sslmode=disable"
	}

	// Use pgx stdlib driver for better perf/features while keeping database/sql API
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		fmt.Printf("Warning: Failed to open database: %v\n", err)
		fmt.Println("Running in database-less mode for testing...")
		return &DB{nil}, nil // Return with nil DB for testing
	}

	if err := db.Ping(); err != nil {
		fmt.Printf("Warning: Failed to ping database: %v\n", err)
		fmt.Println("Running in database-less mode for testing...")
		return &DB{nil}, nil // Return with nil DB for testing
	}

	// Run migrations: prefer golang-migrate if enabled, otherwise fallback to inline
	useM := strings.ToLower(os.Getenv("USE_MIGRATIONS"))
	if useM == "1" || useM == "true" || useM == "yes" || useM == "" {
		path := os.Getenv("MIGRATIONS_PATH")
		if path == "" {
			path = "migrations" // default relative directory
		}
		if err := runWithGolangMigrate(dbURL, path); err != nil {
			fmt.Printf("Warning: golang-migrate failed: %v\n", err)
			fmt.Println("Falling back to inline migrations...")
			if err2 := runMigrations(db); err2 != nil {
				fmt.Printf("Warning: Failed to run inline migrations: %v\n", err2)
				fmt.Println("Running in database-less mode for testing...")
				return &DB{nil}, nil
			}
		}
	} else {
		if err := runMigrations(db); err != nil {
			fmt.Printf("Warning: Failed to run migrations: %v\n", err)
			fmt.Println("Running in database-less mode for testing...")
			return &DB{nil}, nil // Return with nil DB for testing
		}
	}

	fmt.Println("Database connected successfully!")
	return &DB{db}, nil
}

func runMigrations(db *sql.DB) error {
	// Create jobs table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS jobs (
			id SERIAL PRIMARY KEY,
			jobspec_id VARCHAR(255) UNIQUE NOT NULL,
			jobspec_data JSONB NOT NULL,
			status VARCHAR(50) DEFAULT 'queued',
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create jobs table: %w", err)
	}

	// Create executions table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS executions (
			id SERIAL PRIMARY KEY,
			job_id INTEGER REFERENCES jobs(id),
			provider_id VARCHAR(255) NOT NULL,
			region VARCHAR(100) NOT NULL,
			status VARCHAR(50) DEFAULT 'pending',
			started_at TIMESTAMP,
			completed_at TIMESTAMP,
			output_data JSONB,
			receipt_data JSONB,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create executions table: %w", err)
	}

	// Add IPFS columns to executions (idempotent)
	_, err = db.Exec(`
		ALTER TABLE executions
		ADD COLUMN IF NOT EXISTS ipfs_cid VARCHAR(100),
		ADD COLUMN IF NOT EXISTS ipfs_pinned_at TIMESTAMP
	`)
	if err != nil {
		return fmt.Errorf("failed to add IPFS columns to executions: %w", err)
	}

	// Add region verification columns to executions (idempotent)
	_, err = db.Exec(`
		ALTER TABLE executions
		ADD COLUMN IF NOT EXISTS region_claimed TEXT,
		ADD COLUMN IF NOT EXISTS region_observed TEXT,
		ADD COLUMN IF NOT EXISTS region_verified BOOLEAN,
		ADD COLUMN IF NOT EXISTS verification_method TEXT,
		ADD COLUMN IF NOT EXISTS preflight_evidence_ref TEXT
	`)
	if err != nil {
		return fmt.Errorf("failed to add region verification columns to executions: %w", err)
	}

	// Create ipfs_bundles table (idempotent)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS ipfs_bundles (
			id SERIAL PRIMARY KEY,
			job_id VARCHAR(100) NOT NULL,
			cid VARCHAR(100) NOT NULL UNIQUE,
			bundle_size BIGINT,
			execution_count INTEGER NOT NULL,
			regions TEXT[] NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			pinned_at TIMESTAMP,
			gateway_url TEXT,
			FOREIGN KEY (job_id) REFERENCES jobs(jobspec_id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create ipfs_bundles table: %w", err)
	}

	// Helpful indexes (idempotent)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_executions_ipfs_cid ON executions(ipfs_cid) WHERE ipfs_cid IS NOT NULL`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_ipfs_bundles_job_id ON ipfs_bundles(job_id)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_ipfs_bundles_cid ON ipfs_bundles(cid)`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_ipfs_bundles_created_at ON ipfs_bundles(created_at)`)

	// Create diffs table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS diffs (
			id SERIAL PRIMARY KEY,
			job_id INTEGER REFERENCES jobs(id),
			region_a VARCHAR(100) NOT NULL,
			region_b VARCHAR(100) NOT NULL,
			similarity_score DECIMAL(5,4),
			diff_data JSONB,
			classification VARCHAR(50),
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create diffs table: %w", err)
	}

	// Create outbox table (for atomic DB + queue pattern)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS outbox (
			id BIGSERIAL PRIMARY KEY,
			topic TEXT NOT NULL,
			payload JSONB NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			published_at TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create outbox table: %w", err)
	}

	return nil
}

// Job represents a stored job in the database
type Job struct {
	ID          int    `json:"id"`
	JobSpecID   string `json:"jobspec_id"`
	JobSpecJSON []byte `json:"jobspec_data"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// GetJob retrieves a job by its JobSpec ID
func (db *DB) GetJob(jobSpecID string) (*Job, error) {
	if db.DB == nil {
		return nil, fmt.Errorf("database not available")
	}

	query := `
		SELECT id, jobspec_id, jobspec_data, status, created_at, updated_at 
		FROM jobs 
		WHERE jobspec_id = $1
	`
	
	var job Job
	err := db.QueryRow(query, jobSpecID).Scan(
		&job.ID,
		&job.JobSpecID,
		&job.JobSpecJSON,
		&job.Status,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job not found: %s", jobSpecID)
		}
		return nil, fmt.Errorf("failed to retrieve job: %w", err)
	}
	
	return &job, nil
}
