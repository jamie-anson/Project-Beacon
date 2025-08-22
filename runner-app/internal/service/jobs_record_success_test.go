package service

import (
    "context"
    "crypto/ed25519"
    "regexp"
    "testing"
    "time"

    sqlmock "github.com/DATA-DOG/go-sqlmock"
    "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func TestRecordExecution_HappyPath_InsertsExecutionAndUpdatesStatus(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()
    svc := NewJobsService(db)

    // Build a signed receipt so VerifySignature passes
    _, priv, err := ed25519.GenerateKey(nil)
    if err != nil {
        t.Fatalf("failed to generate key: %v", err)
    }

    receipt := models.NewReceipt("job-rc", models.ExecutionDetails{
        TaskID:      "task-1",
        ProviderID:  "prov-1",
        Region:      "US",
        StartedAt:   time.Now().Add(-1 * time.Minute),
        CompletedAt: time.Now(),
        Duration:    1 * time.Minute,
        Status:      "completed",
    }, models.ExecutionOutput{Data: map[string]any{"msg": "ok"}}, models.ProvenanceInfo{})

    // Sign the receipt
    if err := receipt.Sign(priv); err != nil {
        t.Fatalf("failed to sign receipt: %v", err)
    }

    // Expect INSERT into executions ... RETURNING id
    mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO executions (job_id, provider_id, region, status, started_at, completed_at, output_data, receipt_data)
        VALUES ((SELECT id FROM jobs WHERE jobspec_id = $1), $2, $3, $4, $5, $6, $7, $8)
        RETURNING id`)).
        WithArgs("job-rc", receipt.ExecutionDetails.ProviderID, receipt.ExecutionDetails.Region, receipt.ExecutionDetails.Status,
            receipt.ExecutionDetails.StartedAt, receipt.ExecutionDetails.CompletedAt, sqlmock.AnyArg(), sqlmock.AnyArg()).
        WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))

    // Expect UPDATE jobs SET status = 'running' WHERE jobspec_id = $2
    mock.ExpectExec(regexp.QuoteMeta(`
        UPDATE jobs 
        SET status = $1, updated_at = NOW() 
        WHERE jobspec_id = $2`)).
        WithArgs("running", "job-rc").
        WillReturnResult(sqlmock.NewResult(1, 1))

    if err := svc.RecordExecution(context.Background(), "job-rc", receipt); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}
