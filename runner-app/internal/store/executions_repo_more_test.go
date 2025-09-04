package store

import (
    "context"
    "database/sql"
    "errors"
    "time"
    "regexp"
    "testing"

    "github.com/DATA-DOG/go-sqlmock"
    "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func TestListExecutionsByJobSpecID_ScanError(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock.New: %v", err) }
    defer db.Close()

    repo := &ExecutionsRepo{DB: db}

    rows := sqlmock.NewRows([]string{"receipt_data"}).
        AddRow("dummy").
        RowError(0, errors.New("scan boom"))

    mock.ExpectQuery("SELECT e.receipt_data\\s+FROM executions e\\s+JOIN jobs j ON e.job_id = j.id\\s+WHERE j.jobspec_id = ").
        WithArgs("job-1").
        WillReturnRows(rows)

    _, err = repo.ListExecutionsByJobSpecID(context.Background(), "job-1")
    if err == nil { t.Fatalf("expected error, got nil") }
    if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet expectations: %v", err) }
}

func TestInsertExecution_QueryError(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock.New: %v", err) }
    defer db.Close()

    repo := &ExecutionsRepo{DB: db}

    mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO executions (job_id, provider_id, region, status, started_at, completed_at, output_data, receipt_data)
        VALUES ((SELECT id FROM jobs WHERE jobspec_id = $1), $2, $3, $4, $5, $6, $7, $8)
        RETURNING id
    `)).
        WithArgs("job-1", "prov", "us", "completed", sqlmock.AnyArg(), sqlmock.AnyArg(), []byte("{}"), []byte("{}")).
        WillReturnError(sql.ErrConnDone)

    // Directly call legacy InsertExecution to avoid JSON marshal variability
    _, err = repo.InsertExecution(context.Background(), "job-1", "prov", "us", "completed", time.Now(), time.Now(), []byte("{}"), []byte("{}"))
    if err == nil {
        t.Fatalf("expected query error")
    }
    if e := mock.ExpectationsWereMet(); e != nil { t.Fatalf("unmet expectations: %v", e) }
}

func TestInsertExecution_ScanIDError(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock.New: %v", err) }
    defer db.Close()

    repo := &ExecutionsRepo{DB: db}

    rows := sqlmock.NewRows([]string{"id"}).AddRow("not-an-int64")
    mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO executions (job_id, provider_id, region, status, started_at, completed_at, output_data, receipt_data)
        VALUES ((SELECT id FROM jobs WHERE jobspec_id = $1), $2, $3, $4, $5, $6, $7, $8)
        RETURNING id
    `)).
        WithArgs("job-1", "prov", "us", "completed", sqlmock.AnyArg(), sqlmock.AnyArg(), []byte("{}"), []byte("{}")).
        WillReturnRows(rows)

    _, err = repo.InsertExecution(context.Background(), "job-1", "prov", "us", "completed", time.Now(), time.Now(), []byte("{}"), []byte("{}"))
    if err == nil {
        t.Fatalf("expected scan id error, got nil")
    }
    if e := mock.ExpectationsWereMet(); e != nil { t.Fatalf("unmet expectations: %v", e) }
}

func TestCreateExecution_InsertError(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock.New: %v", err) }
    defer db.Close()

    repo := &ExecutionsRepo{DB: db}

    rec := models.NewReceipt("job-1", models.ExecutionDetails{ProviderID: "prov", Region: "us", Status: "completed", StartedAt: time.Now(), CompletedAt: time.Now()}, models.ExecutionOutput{Data: map[string]interface{}{"ok": true}, Hash: "h"}, models.ProvenanceInfo{})

    mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO executions (job_id, provider_id, region, status, started_at, completed_at, output_data, receipt_data)
        VALUES ((SELECT id FROM jobs WHERE jobspec_id = $1), $2, $3, $4, $5, $6, $7, $8)
        RETURNING id
    `)).
        WithArgs("job-1", "prov", "us", "completed", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
        WillReturnError(sql.ErrConnDone)

    _, err = repo.CreateExecution(context.Background(), "job-1", rec)
    if err == nil || !regexp.MustCompile(`failed to insert execution`).MatchString(err.Error()) {
        t.Fatalf("expected wrapped insert error, got %v", err)
    }
    if e := mock.ExpectationsWereMet(); e != nil { t.Fatalf("unmet expectations: %v", e) }
}

func TestListExecutionsByJobSpecIDPaginated_ScanError(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock.New: %v", err) }
    defer db.Close()

    repo := &ExecutionsRepo{DB: db}

    rows := sqlmock.NewRows([]string{"receipt_data"}).
        AddRow("dummy").
        RowError(0, errors.New("scan boom"))

    mock.ExpectQuery("SELECT e.receipt_data\\s+FROM executions e\\s+JOIN jobs j ON e.job_id = j.id\\s+WHERE j.jobspec_id = ").
        WithArgs("job-1", 10, 0).
        WillReturnRows(rows)

    _, err = repo.ListExecutionsByJobSpecIDPaginated(context.Background(), "job-1", 10, 0)
    if err == nil { t.Fatalf("expected error, got nil") }
    if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet expectations: %v", err) }
}

func TestListExecutionsByJobSpecID_UnmarshalError(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock.New: %v", err) }
    defer db.Close()

    repo := &ExecutionsRepo{DB: db}

    rows := sqlmock.NewRows([]string{"receipt_data"}).
        AddRow([]byte(`{"id":`)) // malformed JSON

    mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT e.receipt_data
        FROM executions e
        JOIN jobs j ON e.job_id = j.id
        WHERE j.jobspec_id = $1 AND e.receipt_data IS NOT NULL
        ORDER BY e.created_at DESC
    `)).
        WithArgs("job-1").
        WillReturnRows(rows)

    _, err = repo.ListExecutionsByJobSpecID(context.Background(), "job-1")
    if err == nil || !regexp.MustCompile(`failed to unmarshal receipt`).MatchString(err.Error()) {
        t.Fatalf("expected unmarshal error, got %v", err)
    }
    if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet expectations: %v", err) }
}

func TestListExecutionsByJobSpecIDPaginated_RowsErr(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock.New: %v", err) }
    defer db.Close()

    repo := &ExecutionsRepo{DB: db}

    // one valid JSON row, but rows.Err() will surface after iteration
    rows := sqlmock.NewRows([]string{"receipt_data"}).
        AddRow([]byte(`{"job_id":"job-1"}`)).
        RowError(0, errors.New("iter boom"))

    mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT e.receipt_data
        FROM executions e
        JOIN jobs j ON e.job_id = j.id
        WHERE j.jobspec_id = $1 AND e.receipt_data IS NOT NULL
        ORDER BY e.created_at DESC
        LIMIT $2 OFFSET $3
    `)).
        WithArgs("job-1", 10, 0).
        WillReturnRows(rows)

    _, err = repo.ListExecutionsByJobSpecIDPaginated(context.Background(), "job-1", 10, 0)
    if err == nil || !regexp.MustCompile(`error iterating execution rows`).MatchString(err.Error()) {
        t.Fatalf("expected rows.Err error, got %v", err)
    }
    if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet expectations: %v", err) }
}

func TestGetReceiptByJobSpecID_NotFound(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock.New: %v", err) }
    defer db.Close()

    repo := &ExecutionsRepo{DB: db}

    mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT e.receipt_data
        FROM executions e
        JOIN jobs j ON e.job_id = j.id
        WHERE j.jobspec_id = $1 AND e.receipt_data IS NOT NULL
        ORDER BY e.created_at DESC
        LIMIT 1
    `)).
        WithArgs("job-404").
        WillReturnError(sql.ErrNoRows)

    r, err := repo.GetReceiptByJobSpecID(context.Background(), "job-404")
    if r != nil || err == nil || !regexp.MustCompile(`no receipt found`).MatchString(err.Error()) {
        t.Fatalf("expected not found error, got r=%v err=%v", r, err)
    }
    if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet expectations: %v", err) }
}

func TestGetReceiptByJobSpecID_UnmarshalError(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock.New: %v", err) }
    defer db.Close()

    repo := &ExecutionsRepo{DB: db}

    rows := sqlmock.NewRows([]string{"receipt_data"}).
        AddRow([]byte(`{"id":`))

    mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT e.receipt_data
        FROM executions e
        JOIN jobs j ON e.job_id = j.id
        WHERE j.jobspec_id = $1 AND e.receipt_data IS NOT NULL
        ORDER BY e.created_at DESC
        LIMIT 1
    `)).
        WithArgs("job-1").
        WillReturnRows(rows)

    r, err := repo.GetReceiptByJobSpecID(context.Background(), "job-1")
    if r != nil || err == nil || !regexp.MustCompile(`failed to unmarshal receipt`).MatchString(err.Error()) {
        t.Fatalf("expected unmarshal error, got r=%v err=%v", r, err)
    }
    if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet expectations: %v", err) }
}
