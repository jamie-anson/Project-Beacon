package service

import (
    "context"
    "crypto/ed25519"
    "errors"
    "regexp"
    "testing"
    "time"

    sqlmock "github.com/DATA-DOG/go-sqlmock"
    "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func TestRecordExecution_InsertExecutionError_Propagates(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()
    svc := NewJobsService(db)

    // Valid signed receipt
    _, priv, _ := ed25519.GenerateKey(nil)
    receipt := models.NewReceipt("job-xyz", models.ExecutionDetails{
        TaskID:      "t1",
        ProviderID:  "p1",
        Region:      "US",
        StartedAt:   time.Now().Add(-time.Minute),
        CompletedAt: time.Now(),
        Duration:    time.Minute,
        Status:      "completed",
    }, models.ExecutionOutput{}, models.ProvenanceInfo{})
    _ = receipt.Sign(priv)

    // Fail insert query
    mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO executions (job_id, provider_id, region, status, started_at, completed_at, output_data, receipt_data)
        VALUES ((SELECT id FROM jobs WHERE jobspec_id = $1), $2, $3, $4, $5, $6, $7, $8)
        RETURNING id`)).
        WithArgs("job-xyz", receipt.ExecutionDetails.ProviderID, receipt.ExecutionDetails.Region, receipt.ExecutionDetails.Status,
            receipt.ExecutionDetails.StartedAt, receipt.ExecutionDetails.CompletedAt, sqlmock.AnyArg(), sqlmock.AnyArg()).
        WillReturnError(errors.New("insert fail"))

    // No expectation for UpdateJobStatus, should not be called

    if err := svc.RecordExecution(context.Background(), "job-xyz", receipt); err == nil {
        t.Fatalf("expected error, got nil")
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unexpected DB interactions: %v", err)
    }
}

func TestRecordExecution_UpdateStatusZeroRows_ReturnsError(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()
    svc := NewJobsService(db)

    // Valid signed receipt
    _, priv, _ := ed25519.GenerateKey(nil)
    receipt := models.NewReceipt("job-missing", models.ExecutionDetails{
        TaskID:      "t2",
        ProviderID:  "p2",
        Region:      "EU",
        StartedAt:   time.Now().Add(-time.Minute),
        CompletedAt: time.Now(),
        Duration:    time.Minute,
        Status:      "failed",
    }, models.ExecutionOutput{}, models.ProvenanceInfo{})
    _ = receipt.Sign(priv)

    // Insert execution succeeds
    mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO executions (job_id, provider_id, region, status, started_at, completed_at, output_data, receipt_data)
        VALUES ((SELECT id FROM jobs WHERE jobspec_id = $1), $2, $3, $4, $5, $6, $7, $8)
        RETURNING id`)).
        WithArgs("job-missing", receipt.ExecutionDetails.ProviderID, receipt.ExecutionDetails.Region, receipt.ExecutionDetails.Status,
            receipt.ExecutionDetails.StartedAt, receipt.ExecutionDetails.CompletedAt, sqlmock.AnyArg(), sqlmock.AnyArg()).
        WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(99))

    // Update status affects 0 rows -> repo returns error
    mock.ExpectExec(regexp.QuoteMeta(`
        UPDATE jobs 
        SET status = $1, updated_at = NOW() 
        WHERE jobspec_id = $2`)).
        WithArgs("failed", "job-missing").
        WillReturnResult(sqlmock.NewResult(0, 0))

    if err := svc.RecordExecution(context.Background(), "job-missing", receipt); err == nil {
        t.Fatalf("expected error, got nil")
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}
