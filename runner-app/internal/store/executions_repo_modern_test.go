package store

import (
    "context"
    "database/sql"
    "encoding/json"
    "regexp"
    "testing"
    "time"

    "github.com/DATA-DOG/go-sqlmock"
    "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func TestGetReceiptByJobSpecID_NilDB(t *testing.T) {
    r := &ExecutionsRepo{DB: nil}
    if _, err := r.GetReceiptByJobSpecID(context.Background(), "job-1"); err == nil {
        t.Fatalf("expected error for nil DB")
    }
}

func TestGetReceiptByJobSpecID_NoRows(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()
    r := &ExecutionsRepo{DB: db}

    mock.ExpectQuery(`SELECT`).
        WithArgs("job-404").
        WillReturnRows(sqlmock.NewRows([]string{"receipt_data"}))

    _, err := r.GetReceiptByJobSpecID(context.Background(), "job-404")
    if err == nil {
        t.Fatalf("expected error for not found receipt")
    }
    if e := mock.ExpectationsWereMet(); e != nil {
        t.Fatalf("unmet expectations: %v", e)
    }
}

func TestGetReceiptByJobSpecID_Success(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()
    r := &ExecutionsRepo{DB: db}

    rec := models.NewReceipt("job-1", models.ExecutionDetails{ProviderID: "p", Region: "us", Status: "completed", StartedAt: time.Now(), CompletedAt: time.Now()}, models.ExecutionOutput{Data: map[string]interface{}{"ok": true}, Hash: "h"}, models.ProvenanceInfo{})
    b, _ := json.Marshal(rec)
    rows := sqlmock.NewRows([]string{"receipt_data"}).AddRow(b)

    mock.ExpectQuery(`SELECT`).
        WithArgs("job-1").
        WillReturnRows(rows)

    out, err := r.GetReceiptByJobSpecID(context.Background(), "job-1")
    if err != nil || out == nil || out.ExecutionDetails.ProviderID != "p" {
        t.Fatalf("unexpected result: out=%v err=%v", out, err)
    }
    if e := mock.ExpectationsWereMet(); e != nil {
        t.Fatalf("unmet expectations: %v", e)
    }
}

func TestListExecutionsByJobSpecIDPaginated_DefaultsAndSuccess(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()
    r := &ExecutionsRepo{DB: db}

    rec := models.NewReceipt("job-1", models.ExecutionDetails{ProviderID: "p1", Region: "us", Status: "completed", StartedAt: time.Now(), CompletedAt: time.Now()}, models.ExecutionOutput{Data: map[string]interface{}{"n": 1}, Hash: "h1"}, models.ProvenanceInfo{})
    b, _ := json.Marshal(rec)

    mock.ExpectQuery(`SELECT`).
        WithArgs("job-1", 20, 0).
        WillReturnRows(sqlmock.NewRows([]string{"receipt_data"}).AddRow(b))

    out, err := r.ListExecutionsByJobSpecIDPaginated(context.Background(), "job-1", 0, -1)
    if err != nil || len(out) != 1 {
        t.Fatalf("unexpected: len=%d err=%v", len(out), err)
    }
    if e := mock.ExpectationsWereMet(); e != nil {
        t.Fatalf("unmet expectations: %v", e)
    }
}

func TestListExecutionsByJobSpecIDPaginated_UnmarshalError(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()
    r := &ExecutionsRepo{DB: db}

    mock.ExpectQuery(`SELECT`).
        WithArgs("job-1", 5, 2).
        WillReturnRows(sqlmock.NewRows([]string{"receipt_data"}).AddRow([]byte("{ not-json }")))

    if _, err := r.ListExecutionsByJobSpecIDPaginated(context.Background(), "job-1", 5, 2); err == nil {
        t.Fatalf("expected unmarshal error")
    }
}

func TestListExecutionsByJobSpecID_QueryError(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()
    r := &ExecutionsRepo{DB: db}

    mock.ExpectQuery(`SELECT`).
        WithArgs("job-2").
        WillReturnError(sql.ErrConnDone)

    if _, err := r.ListExecutionsByJobSpecID(context.Background(), "job-2"); err == nil {
        t.Fatalf("expected query error")
    }
}

func TestUpdateExecutionStatus_Success(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()
    r := &ExecutionsRepo{DB: db}

    mock.ExpectExec(regexp.QuoteMeta(`UPDATE executions 
        SET status = $1 
        WHERE id = $2`)).
        WithArgs("completed", int64(42)).
        WillReturnResult(sqlmock.NewResult(0, 1))

    if err := r.UpdateExecutionStatus(context.Background(), 42, "completed"); err != nil {
        t.Fatalf("unexpected err: %v", err)
    }
    if e := mock.ExpectationsWereMet(); e != nil {
        t.Fatalf("unmet expectations: %v", e)
    }
}

func TestCreateExecution_MarshalError(t *testing.T) {
    db, _, _ := sqlmock.New()
    defer db.Close()
    r := &ExecutionsRepo{DB: db}

    rec := &models.Receipt{
        ExecutionDetails: models.ExecutionDetails{ProviderID: "p", Region: "us", Status: "completed", StartedAt: time.Now(), CompletedAt: time.Now()},
        // Put a non-JSON-marshalable value in Output.Data
        Output: models.ExecutionOutput{Data: map[string]interface{}{"bad": func() {}}, Hash: "h"},
    }
    if _, err := r.CreateExecution(context.Background(), "job-1", rec); err == nil {
        t.Fatalf("expected marshal error")
    }
}
