package integration

import (
	"context"
	"database/sql"
	"testing"
	"time"

	intdb "github.com/jamie-anson/project-beacon-runner/internal/db"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func TestDB_RegionVerificationPersistence(t *testing.T) {
	ctx := context.Background()
	// Initialize DB (uses default DSN or env). Will return nil DB in tests if not reachable.
	db, err := intdb.Initialize("")
	if err != nil {
		// If init returns error, skip to avoid flakiness on environments without DB
		t.Skipf("db init error: %v", err)
	}
	if db.DB == nil {
		t.Skip("database not available; skipping DB-backed test")
	}
	defer db.Close()

	// Insert a job row to satisfy FK
	jobspecID := "int-db-rv-1"
	jobspecJSON := []byte(`{"id":"int-db-rv-1"}`)
	if _, err := db.ExecContext(ctx, `INSERT INTO jobs (jobspec_id, jobspec_data, status) VALUES ($1,$2,'queued') ON CONFLICT (jobspec_id) DO NOTHING`, jobspecID, jobspecJSON); err != nil {
		t.Fatalf("insert job failed: %v", err)
	}

	repo := store.NewExecutionsRepo(db.DB)

	// Create a minimal receipt to insert an execution
	receipt := &models.Receipt{
		ExecutionDetails: models.ExecutionDetails{
			ProviderID: "prov-test",
			Region:     "EU",
			Status:     "completed",
			StartedAt:  time.Now().Add(-1 * time.Minute),
			CompletedAt: time.Now(),
		},
	}
	execID, err := repo.CreateExecution(ctx, jobspecID, receipt)
	if err != nil {
		t.Fatalf("create execution failed: %v", err)
	}

	// Update region verification
	claimed := sql.NullString{String: "EU", Valid: true}
	observed := sql.NullString{String: "EU", Valid: true}
	verified := sql.NullBool{Bool: true, Valid: true}
	method := sql.NullString{String: "preflight-geoip", Valid: true}
	evidence := sql.NullString{String: "evidence://test", Valid: true}
	if err := repo.UpdateRegionVerification(ctx, execID, claimed, observed, verified, method, evidence); err != nil {
		t.Fatalf("update verification failed: %v", err)
	}

	// Read back columns directly
	var gotClaimed, gotObserved, gotMethod, gotEvidence sql.NullString
	var gotVerified sql.NullBool
	row := db.QueryRowContext(ctx, `SELECT region_claimed, region_observed, region_verified, verification_method, preflight_evidence_ref FROM executions WHERE id=$1`, execID)
	if err := row.Scan(&gotClaimed, &gotObserved, &gotVerified, &gotMethod, &gotEvidence); err != nil {
		t.Fatalf("scan verification columns failed: %v", err)
	}
	if !gotClaimed.Valid || gotClaimed.String != "EU" || !gotObserved.Valid || gotObserved.String != "EU" || !gotVerified.Valid || !gotVerified.Bool || !gotMethod.Valid || gotMethod.String != "preflight-geoip" || !gotEvidence.Valid {
		t.Fatalf("unexpected verification values: claimed=%v observed=%v verified=%v method=%v evidence=%v", gotClaimed, gotObserved, gotVerified, gotMethod, gotEvidence)
	}
}

func TestDB_RegionVerificationDefaultsNull(t *testing.T) {
	ctx := context.Background()
	db, err := intdb.Initialize("")
	if err != nil {
		t.Skipf("db init error: %v", err)
	}
	if db.DB == nil {
		t.Skip("database not available; skipping DB-backed test")
	}
	defer db.Close()

	// Ensure job exists
	jobspecID := "int-db-rv-null"
	jobspecJSON := []byte(`{"id":"int-db-rv-null"}`)
	if _, err := db.ExecContext(ctx, `INSERT INTO jobs (jobspec_id, jobspec_data, status) VALUES ($1,$2,'queued') ON CONFLICT (jobspec_id) DO NOTHING`, jobspecID, jobspecJSON); err != nil {
		t.Fatalf("insert job failed: %v", err)
	}

	// Create execution without updating verification columns
	repo := store.NewExecutionsRepo(db.DB)
	receipt := &models.Receipt{ExecutionDetails: models.ExecutionDetails{ProviderID: "prov-null", Region: "US", Status: "completed", StartedAt: time.Now().Add(-2 * time.Minute), CompletedAt: time.Now()}}
	execID, err := repo.CreateExecution(ctx, jobspecID, receipt)
	if err != nil {
		t.Fatalf("create execution failed: %v", err)
	}

	var claimed, observed, method, evidence sql.NullString
	var verified sql.NullBool
	row := db.QueryRowContext(ctx, `SELECT region_claimed, region_observed, region_verified, verification_method, preflight_evidence_ref FROM executions WHERE id=$1`, execID)
	if err := row.Scan(&claimed, &observed, &verified, &method, &evidence); err != nil {
		t.Fatalf("scan verification columns failed: %v", err)
	}
	if claimed.Valid || observed.Valid || verified.Valid || method.Valid || evidence.Valid {
		t.Fatalf("expected all verification fields to be NULL by default, got claimed=%v observed=%v verified=%v method=%v evidence=%v", claimed, observed, verified, method, evidence)
	}
}
