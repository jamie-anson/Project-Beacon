package service

import (
    "context"
    "crypto/ed25519"
    "regexp"
    "testing"
    "time"

    sqlmock "github.com/DATA-DOG/go-sqlmock"
    "github.com/jamie-anson/project-beacon-runner/internal/store"
    "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func buildSignedReceipt(t *testing.T, status string) *models.Receipt {
    t.Helper()
    _, priv, err := ed25519.GenerateKey(nil)
    if err != nil { t.Fatalf("gen key: %v", err) }

    r := models.NewReceipt(
        "job-xyz",
        models.ExecutionDetails{
            TaskID: "task1", ProviderID: "prov1", Region: "EU",
            StartedAt: time.Now().Add(-time.Minute), CompletedAt: time.Now(),
            Duration: time.Second, Status: status,
        },
        models.ExecutionOutput{Data: map[string]any{"out":"ok"}, Hash: "h", Metadata: map[string]any{}},
        models.ProvenanceInfo{BenchmarkHash: "bh", ProviderInfo: map[string]any{}, ExecutionEnv: map[string]any{}},
    )
    if err := r.Sign(priv); err != nil {
        t.Fatalf("sign receipt: %v", err)
    }
    return r
}

func expectCreateExecutionInsert(mock sqlmock.Sqlmock, jobspecID string, r *models.Receipt, retID int64) {
    // We don't need exact JSON, accept any args types; ensure RETURNING id yields retID
    mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO executions (job_id, provider_id, region, status, started_at, completed_at, output_data, receipt_data)
        VALUES ((SELECT id FROM jobs WHERE jobspec_id = $1), $2, $3, $4, $5, $6, $7, $8)
        RETURNING id`)).
        WithArgs(jobspecID, r.ExecutionDetails.ProviderID, r.ExecutionDetails.Region, r.ExecutionDetails.Status, r.ExecutionDetails.StartedAt, r.ExecutionDetails.CompletedAt, sqlmock.AnyArg(), sqlmock.AnyArg()).
        WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(retID))
}

func expectUpdateJobStatus(mock sqlmock.Sqlmock, jobspecID, status string) {
    mock.ExpectExec(regexp.QuoteMeta(`
        UPDATE jobs 
        SET status = $1, updated_at = NOW() 
        WHERE jobspec_id = $2`)).
        WithArgs(status, jobspecID).
        WillReturnResult(sqlmock.NewResult(0, 1))
}

func TestRecordExecution_Happy_CompletesAndUpdatesRunning(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()
    svc := NewJobsService(db)
    // Override repos to ensure they use the same db
    svc.ExecutionsRepo = store.NewExecutionsRepo(db)
    svc.JobsRepo = store.NewJobsRepo(db)

    r := buildSignedReceipt(t, "completed")

    expectCreateExecutionInsert(mock, "job-xyz", r, 1)
    // completed maps to running
    expectUpdateJobStatus(mock, "job-xyz", "running")

    if err := svc.RecordExecution(context.Background(), "job-xyz", r); err != nil {
        t.Fatalf("unexpected err: %v", err)
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestRecordExecution_StatusFailed_UpdatesFailed(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()
    svc := NewJobsService(db)
    svc.ExecutionsRepo = store.NewExecutionsRepo(db)
    svc.JobsRepo = store.NewJobsRepo(db)

    r := buildSignedReceipt(t, "failed")

    expectCreateExecutionInsert(mock, "job-xyz", r, 2)
    expectUpdateJobStatus(mock, "job-xyz", "failed")

    if err := svc.RecordExecution(context.Background(), "job-xyz", r); err != nil {
        t.Fatalf("unexpected err: %v", err)
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestRecordExecution_DBErrorFromInsert_Propagates(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()
    svc := NewJobsService(db)
    svc.ExecutionsRepo = store.NewExecutionsRepo(db)
    svc.JobsRepo = store.NewJobsRepo(db)

    r := buildSignedReceipt(t, "completed")

    // Cause insert to fail
    mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO executions (job_id, provider_id, region, status, started_at, completed_at, output_data, receipt_data)
        VALUES ((SELECT id FROM jobs WHERE jobspec_id = $1), $2, $3, $4, $5, $6, $7, $8)
        RETURNING id`)).
        WithArgs("job-xyz", r.ExecutionDetails.ProviderID, r.ExecutionDetails.Region, r.ExecutionDetails.Status, r.ExecutionDetails.StartedAt, r.ExecutionDetails.CompletedAt, sqlmock.AnyArg(), sqlmock.AnyArg()).
        WillReturnError(context.DeadlineExceeded)

    if err := svc.RecordExecution(context.Background(), "job-xyz", r); err == nil {
        t.Fatalf("expected error, got nil")
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}
