package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

func Initialize() (*DB, error) {
	// Get database URL from environment or use default
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:password@localhost:5432/beacon_runner?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
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

	// Run migrations
	if err := runMigrations(db); err != nil {
		fmt.Printf("Warning: Failed to run migrations: %v\n", err)
		fmt.Println("Running in database-less mode for testing...")
		return &DB{nil}, nil // Return with nil DB for testing
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

	return nil
}
