package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/mattn/go-sqlite3"
)

func TestGetCrossRegionExecutions_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewExecutionsRepo(db)
	ctx := context.Background()

	// Create test job
	_, err := db.ExecContext(ctx, `
		INSERT INTO jobs (id, jobspec_id, status, created_at)
		VALUES (1, 'test-job-123', 'completed', datetime('now'))
	`)
	require.NoError(t, err)

	// Create test executions for different regions
	outputUS := map[string]interface{}{
		"response": "US response text",
	}
	outputUSJSON, _ := json.Marshal(outputUS)

	outputEU := map[string]interface{}{
		"response": "EU response text",
	}
	outputEUJSON, _ := json.Marshal(outputEU)

	// Insert US execution
	_, err = db.ExecContext(ctx, `
		INSERT INTO executions (
			job_id, region, status, provider_id, model_id, question_id,
			output_data, started_at, created_at,
			response_length, is_substantive
		)
		VALUES (1, 'us-east', 'completed', 'modal-us-001', 'llama3.2-1b', 'identity_basic', ?, datetime('now'), datetime('now'), 100, 1)
	`, outputUSJSON)
	require.NoError(t, err)

	// Insert EU execution
	_, err = db.ExecContext(ctx, `
		INSERT INTO executions (
			job_id, region, status, provider_id, model_id, question_id,
			output_data, started_at, created_at,
			response_length, is_substantive
		)
		VALUES (1, 'eu-west', 'completed', 'modal-eu-001', 'llama3.2-1b', 'identity_basic', ?, datetime('now'), datetime('now'), 80, 1)
	`, outputEUJSON)
	require.NoError(t, err)

	// Test GetCrossRegionExecutions
	executions, err := repo.GetCrossRegionExecutions(ctx, "test-job-123", "llama3.2-1b", "identity_basic")
	require.NoError(t, err)
	assert.Equal(t, 2, len(executions))

	// Verify executions are ordered by region
	assert.Equal(t, "eu-west", executions[0]["region"]) // Alphabetically first
	assert.Equal(t, "us-east", executions[1]["region"])

	// Verify execution data
	usExec := executions[1]
	assert.Equal(t, "test-job-123", usExec["job_id"])
	assert.Equal(t, "llama3.2-1b", usExec["model_id"])
	assert.Equal(t, "completed", usExec["status"])
	assert.Equal(t, "modal-us-001", usExec["provider_id"])
	assert.Equal(t, int64(100), usExec["response_length"])
	assert.Equal(t, true, usExec["is_substantive"])

	// Verify output data
	output := usExec["output"].(map[string]interface{})
	assert.Equal(t, "US response text", output["response"])
}

func TestGetCrossRegionExecutions_NoResults(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewExecutionsRepo(db)
	ctx := context.Background()

	executions, err := repo.GetCrossRegionExecutions(ctx, "nonexistent-job", "llama3.2-1b", "identity_basic")
	require.NoError(t, err)
	assert.Equal(t, 0, len(executions))
}

func TestGetCrossRegionExecutions_FiltersByModel(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewExecutionsRepo(db)
	ctx := context.Background()

	// Create test job
	_, err := db.ExecContext(ctx, `
		INSERT INTO jobs (id, jobspec_id, status, created_at)
		VALUES (1, 'test-job-456', 'completed', datetime('now'))
	`)
	require.NoError(t, err)

	outputJSON, _ := json.Marshal(map[string]interface{}{"response": "test"})

	// Insert executions for different models
	_, err = db.ExecContext(ctx, `
		INSERT INTO executions (job_id, region, status, provider_id, model_id, question_id, output_data, started_at, created_at)
		VALUES 
			(1, 'us-east', 'completed', 'modal-us-001', 'llama3.2-1b', 'identity_basic', ?, datetime('now'), datetime('now')),
			(1, 'us-east', 'completed', 'modal-us-001', 'mistral-7b', 'identity_basic', ?, datetime('now'), datetime('now'))
	`, outputJSON, outputJSON)
	require.NoError(t, err)

	// Query for llama3.2-1b only
	executions, err := repo.GetCrossRegionExecutions(ctx, "test-job-456", "llama3.2-1b", "identity_basic")
	require.NoError(t, err)
	assert.Equal(t, 1, len(executions))
	assert.Equal(t, "llama3.2-1b", executions[0]["model_id"])

	// Query for mistral-7b only
	executions, err = repo.GetCrossRegionExecutions(ctx, "test-job-456", "mistral-7b", "identity_basic")
	require.NoError(t, err)
	assert.Equal(t, 1, len(executions))
	assert.Equal(t, "mistral-7b", executions[0]["model_id"])
}

func TestGetCrossRegionExecutions_HandlesNullFields(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewExecutionsRepo(db)
	ctx := context.Background()

	// Create test job
	_, err := db.ExecContext(ctx, `
		INSERT INTO jobs (id, jobspec_id, status, created_at)
		VALUES (1, 'test-job-789', 'completed', datetime('now'))
	`)
	require.NoError(t, err)

	outputJSON, _ := json.Marshal(map[string]interface{}{"response": "test"})

	// Insert execution with NULL optional fields
	_, err = db.ExecContext(ctx, `
		INSERT INTO executions (
			job_id, region, status, provider_id, model_id, question_id,
			output_data, started_at, created_at
		)
		VALUES (1, 'us-east', 'completed', 'modal-us-001', 'llama3.2-1b', NULL, ?, datetime('now'), datetime('now'))
	`, outputJSON)
	require.NoError(t, err)

	executions, err := repo.GetCrossRegionExecutions(ctx, "test-job-789", "llama3.2-1b", "")
	require.NoError(t, err)
	assert.Equal(t, 1, len(executions))

	exec := executions[0]
	// Verify NULL fields are not present in the map
	_, hasQuestionID := exec["question_id"]
	assert.False(t, hasQuestionID)

	_, hasCompletedAt := exec["completed_at"]
	assert.False(t, hasCompletedAt)

	_, hasResponseLength := exec["response_length"]
	assert.False(t, hasResponseLength)
}

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	// Create jobs table
	_, err = db.Exec(`
		CREATE TABLE jobs (
			id INTEGER PRIMARY KEY,
			jobspec_id TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at DATETIME NOT NULL
		)
	`)
	require.NoError(t, err)

	// Create executions table
	_, err = db.Exec(`
		CREATE TABLE executions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			job_id INTEGER NOT NULL,
			region TEXT NOT NULL,
			status TEXT NOT NULL,
			provider_id TEXT NOT NULL,
			model_id TEXT NOT NULL,
			question_id TEXT,
			output_data TEXT,
			started_at DATETIME NOT NULL,
			completed_at DATETIME,
			created_at DATETIME NOT NULL,
			response_length INTEGER,
			response_classification TEXT,
			is_substantive BOOLEAN,
			is_content_refusal BOOLEAN,
			is_technical_error BOOLEAN,
			FOREIGN KEY (job_id) REFERENCES jobs(id)
		)
	`)
	require.NoError(t, err)

	return db
}
