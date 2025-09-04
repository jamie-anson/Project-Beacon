package service

import (
    "context"
    "encoding/json"
    "regexp"
    "testing"
    "time"

    sqlmock "github.com/DATA-DOG/go-sqlmock"
    "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func TestRecordExecution_SignatureFailure_NoDBCalls(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()
    svc := NewJobsService(db)

    // No expectations on DB: VerifySignature should fail first and short-circuit
    receipt := &models.Receipt{ // missing Signature/PublicKey triggers validation error
        JobSpecID: "job-x",
        ExecutionDetails: models.ExecutionDetails{Status: "completed"},
    }

    err := svc.RecordExecution(context.Background(), "job-x", receipt)
    if err == nil {
        t.Fatalf("expected signature verification error, got nil")
    }

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unexpected DB interactions: %v", err)
    }
}

func TestGetJob_HappyPath_ReturnsSpecAndStatus(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()
    svc := NewJobsService(db)

    // Prepare a minimal jobspec JSON stored in DB
    spec := &models.JobSpec{
        ID: "job-123",
        Version: "1.0.0",
        Benchmark: models.BenchmarkSpec{Name: "who-are-you"},
        Constraints: models.ExecutionConstraints{Regions: []string{"US"}},
        CreatedAt: time.Now(),
    }
    data, _ := json.Marshal(spec)

    rows := sqlmock.NewRows([]string{"jobspec_data", "status", "created_at", "updated_at"}).
        AddRow(data, "created", time.Now(), time.Now())

    mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT jobspec_data, status, created_at, updated_at 
        FROM jobs 
        WHERE jobspec_id = $1`)).
        WithArgs("job-123").
        WillReturnRows(rows)

    got, status, err := svc.GetJob(context.Background(), "job-123")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if status != "created" {
        t.Fatalf("unexpected status: %s", status)
    }
    if got == nil || got.ID != "job-123" || got.Benchmark.Name != "who-are-you" {
        t.Fatalf("unexpected jobspec: %+v", got)
    }

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}
