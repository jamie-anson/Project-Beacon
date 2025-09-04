//go:build store_legacy_tests

package store

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/mattn/go-sqlite3"
)

func TestExecutionsRepo_Insert_Success(t *testing.T) {
	db := setupExecutionsTestDB(t)
	defer db.Close()

	repo := NewExecutionsRepo(db)

	execution := &Execution{
		ID:        "exec-1",
		JobSpecID: "job-1",
		Region:    "US",
		Status:    "completed",
		Output:    `{"result": "hello world"}`,
		StartedAt: time.Now(),
		CompletedAt: sql.NullTime{
			Time:  time.Now().Add(time.Minute),
			Valid: true,
		},
		CreatedAt: time.Now(),
	}

	err := repo.Insert(execution)
	require.NoError(t, err)

	// Verify execution was inserted
	retrieved, err := repo.GetByID(execution.ID)
	require.NoError(t, err)
	assert.Equal(t, execution.ID, retrieved.ID)
	assert.Equal(t, execution.JobSpecID, retrieved.JobSpecID)
	assert.Equal(t, execution.Region, retrieved.Region)
	assert.Equal(t, execution.Status, retrieved.Status)
}

func TestExecutionsRepo_GetByID_NotFound(t *testing.T) {
	db := setupExecutionsTestDB(t)
	defer db.Close()

	repo := NewExecutionsRepo(db)

	_, err := repo.GetByID("non-existent-exec")
	require.Error(t, err)
	assert.Equal(t, sql.ErrNoRows, err)
}

func TestExecutionsRepo_GetExecutionsByJobSpecID(t *testing.T) {
	db := setupExecutionsTestDB(t)
	defer db.Close()

	repo := NewExecutionsRepo(db)

	jobSpecID := "job-multi-exec"

	// Insert multiple executions for the same job
	regions := []string{"US", "EU", "APAC"}
	for i, region := range regions {
		execution := &Execution{
			ID:        fmt.Sprintf("exec-%d", i+1),
			JobSpecID: jobSpecID,
			Region:    region,
			Status:    "completed",
			Output:    fmt.Sprintf(`{"result": "output from %s"}`, region),
			StartedAt: time.Now(),
			CompletedAt: sql.NullTime{
				Time:  time.Now().Add(time.Minute),
				Valid: true,
			},
			CreatedAt: time.Now(),
		}
		err := repo.Insert(execution)
		require.NoError(t, err)
	}

	// Get all executions for the job
	executions, err := repo.GetExecutionsByJobSpecID(jobSpecID)
	require.NoError(t, err)
	assert.Len(t, executions, 3)

	// Verify all regions are represented
	regionSet := make(map[string]bool)
	for _, exec := range executions {
		regionSet[exec.Region] = true
		assert.Equal(t, jobSpecID, exec.JobSpecID)
	}
	assert.True(t, regionSet["US"])
	assert.True(t, regionSet["EU"])
	assert.True(t, regionSet["APAC"])
}

func TestExecutionsRepo_UpdateStatus_Success(t *testing.T) {
	db := setupExecutionsTestDB(t)
	defer db.Close()

	repo := NewExecutionsRepo(db)

	execution := &Execution{
		ID:        "exec-status-update",
		JobSpecID: "job-status",
		Region:    "US",
		Status:    "running",
		StartedAt: time.Now(),
		CreatedAt: time.Now(),
	}

	err := repo.Insert(execution)
	require.NoError(t, err)

	// Update status to completed
	err = repo.UpdateStatus(execution.ID, "completed")
	require.NoError(t, err)

	// Verify status was updated
	retrieved, err := repo.GetByID(execution.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", retrieved.Status)
}

func TestExecutionsRepo_UpdateIPFSCID_Success(t *testing.T) {
	db := setupExecutionsTestDB(t)
	defer db.Close()

	repo := NewExecutionsRepo(db)

	execution := &Execution{
		ID:        "exec-ipfs-update",
		JobSpecID: "job-ipfs",
		Region:    "US",
		Status:    "completed",
		StartedAt: time.Now(),
		CreatedAt: time.Now(),
	}

	err := repo.Insert(execution)
	require.NoError(t, err)

	// Update IPFS CID
	testCID := "QmTestCID123"
	err = repo.UpdateIPFSCID(execution.ID, testCID)
	require.NoError(t, err)

	// Verify CID was updated
	retrieved, err := repo.GetByID(execution.ID)
	require.NoError(t, err)
	assert.Equal(t, testCID, retrieved.IPFSCID.String)
	assert.True(t, retrieved.IPFSCID.Valid)
}

func TestExecutionsRepo_GetExecutionsByStatus(t *testing.T) {
	db := setupExecutionsTestDB(t)
	defer db.Close()

	repo := NewExecutionsRepo(db)

	// Insert executions with different statuses
	statuses := []string{"pending", "running", "completed", "failed"}
	for i, status := range statuses {
		execution := &Execution{
			ID:        fmt.Sprintf("exec-status-%d", i),
			JobSpecID: fmt.Sprintf("job-%d", i),
			Region:    "US",
			Status:    status,
			StartedAt: time.Now(),
			CreatedAt: time.Now(),
		}
		err := repo.Insert(execution)
		require.NoError(t, err)
	}

	// Test filtering by status
	runningExecs, err := repo.GetExecutionsByStatus("running")
	require.NoError(t, err)
	assert.Len(t, runningExecs, 1)
	assert.Equal(t, "running", runningExecs[0].Status)

	completedExecs, err := repo.GetExecutionsByStatus("completed")
	require.NoError(t, err)
	assert.Len(t, completedExecs, 1)
	assert.Equal(t, "completed", completedExecs[0].Status)
}

func TestExecutionsRepo_GetExecutionsByRegion(t *testing.T) {
	db := setupExecutionsTestDB(t)
	defer db.Close()

	repo := NewExecutionsRepo(db)

	// Insert executions in different regions
	regions := []string{"US", "EU", "APAC"}
	for i, region := range regions {
		execution := &Execution{
			ID:        fmt.Sprintf("exec-region-%d", i),
			JobSpecID: "job-region-test",
			Region:    region,
			Status:    "completed",
			StartedAt: time.Now(),
			CreatedAt: time.Now(),
		}
		err := repo.Insert(execution)
		require.NoError(t, err)
	}

	// Test filtering by region
	usExecs, err := repo.GetExecutionsByRegion("US")
	require.NoError(t, err)
	assert.Len(t, usExecs, 1)
	assert.Equal(t, "US", usExecs[0].Region)

	euExecs, err := repo.GetExecutionsByRegion("EU")
	require.NoError(t, err)
	assert.Len(t, euExecs, 1)
	assert.Equal(t, "EU", euExecs[0].Region)
}

func TestExecutionsRepo_GetExecutionsWithIPFSCID(t *testing.T) {
	db := setupExecutionsTestDB(t)
	defer db.Close()

	repo := NewExecutionsRepo(db)

	// Insert executions with and without IPFS CIDs
	for i := 0; i < 3; i++ {
		execution := &Execution{
			ID:        fmt.Sprintf("exec-ipfs-%d", i),
			JobSpecID: "job-ipfs-test",
			Region:    "US",
			Status:    "completed",
			StartedAt: time.Now(),
			CreatedAt: time.Now(),
		}
		
		if i < 2 {
			execution.IPFSCID = sql.NullString{
				String: fmt.Sprintf("QmTestCID%d", i),
				Valid:  true,
			}
		}
		
		err := repo.Insert(execution)
		require.NoError(t, err)
	}

	// Get executions with IPFS CIDs
	execsWithCID, err := repo.GetExecutionsWithIPFSCID()
	require.NoError(t, err)
	assert.Len(t, execsWithCID, 2)
	
	for _, exec := range execsWithCID {
		assert.True(t, exec.IPFSCID.Valid)
		assert.NotEmpty(t, exec.IPFSCID.String)
	}
}

func TestExecutionsRepo_Delete_Success(t *testing.T) {
	db := setupExecutionsTestDB(t)
	defer db.Close()

	repo := NewExecutionsRepo(db)

	execution := &Execution{
		ID:        "exec-delete",
		JobSpecID: "job-delete",
		Region:    "US",
		Status:    "completed",
		StartedAt: time.Now(),
		CreatedAt: time.Now(),
	}

	err := repo.Insert(execution)
	require.NoError(t, err)

	// Delete execution
	err = repo.Delete(execution.ID)
	require.NoError(t, err)

	// Verify execution was deleted
	_, err = repo.GetByID(execution.ID)
	require.Error(t, err)
	assert.Equal(t, sql.ErrNoRows, err)
}

func TestExecutionsRepo_GetExecutionStats(t *testing.T) {
	db := setupExecutionsTestDB(t)
	defer db.Close()

	repo := NewExecutionsRepo(db)

	jobSpecID := "job-stats"

	// Insert executions with different statuses
	statuses := []string{"completed", "completed", "failed", "running"}
	for i, status := range statuses {
		execution := &Execution{
			ID:        fmt.Sprintf("exec-stats-%d", i),
			JobSpecID: jobSpecID,
			Region:    "US",
			Status:    status,
			StartedAt: time.Now(),
			CreatedAt: time.Now(),
		}
		err := repo.Insert(execution)
		require.NoError(t, err)
	}

	// Get execution statistics
	stats, err := repo.GetExecutionStats(jobSpecID)
	require.NoError(t, err)
	
	assert.Equal(t, 4, stats.Total)
	assert.Equal(t, 2, stats.Completed)
	assert.Equal(t, 1, stats.Failed)
	assert.Equal(t, 1, stats.Running)
	assert.Equal(t, 0, stats.Pending)
}

func TestExecutionsRepo_ErrorsOnNilDB(t *testing.T) {
	repo := NewExecutionsRepo(nil)

	// Insert
	err := repo.Insert(&Execution{})
	assert.Error(t, err)

	// GetByID
	_, err = repo.GetByID("exec-1")
	assert.Error(t, err)

	// GetExecutionsByJobSpecID
	_, err = repo.GetExecutionsByJobSpecID("job-1")
	assert.Error(t, err)

	// UpdateStatus
	err = repo.UpdateStatus("exec-1", "completed")
	assert.Error(t, err)

	// UpdateIPFSCID
	err = repo.UpdateIPFSCID("exec-1", "QmTestCID123")
	assert.Error(t, err)

	// GetExecutionsByStatus
	_, err = repo.GetExecutionsByStatus("running")
	assert.Error(t, err)

	// GetExecutionsByRegion
	_, err = repo.GetExecutionsByRegion("US")
	assert.Error(t, err)

	// GetExecutionsWithIPFSCID
	_, err = repo.GetExecutionsWithIPFSCID()
	assert.Error(t, err)

	// Delete
	err = repo.Delete("exec-1")
	assert.Error(t, err)

	// GetExecutionStats
	_, err = repo.GetExecutionStats("job-1")
	assert.Error(t, err)
}

func TestExecutionsRepo_New_WithDB(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	repo := NewExecutionsRepo(db)
	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.DB)
}

// Test database setup helper for executions
func setupExecutionsTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	// Create executions table schema
	schema := `
	CREATE TABLE executions (
		id TEXT PRIMARY KEY,
		jobspec_id TEXT NOT NULL,
		region TEXT NOT NULL,
		status TEXT NOT NULL,
		output TEXT,
		error_message TEXT,
		ipfs_cid TEXT,
		started_at DATETIME NOT NULL,
		completed_at DATETIME,
		created_at DATETIME NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX idx_executions_jobspec_id ON executions(jobspec_id);
	CREATE INDEX idx_executions_status ON executions(status);
	CREATE INDEX idx_executions_region ON executions(region);
	CREATE INDEX idx_executions_ipfs_cid ON executions(ipfs_cid);
	`

	_, err = db.Exec(schema)
	require.NoError(t, err)

	return db
}
