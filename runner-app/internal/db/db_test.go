package db

import (
    "database/sql"
    "errors"
    "regexp"
    "testing"

    sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestGetJob_DBUnavailable(t *testing.T) {
    d := &DB{DB: nil}
    if _, err := d.GetJob("id-1"); err == nil {
        t.Fatalf("expected error when DB is nil")
    }
}

func TestGetJob_NotFound(t *testing.T) {
    mockDB, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock.New error: %v", err) }
    defer mockDB.Close()

    d := &DB{DB: mockDB}

    // Expect query with jobspec_id parameter
    mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, jobspec_id, jobspec_data, status, created_at, updated_at FROM jobs WHERE jobspec_id = $1`)).
        WithArgs("js-123").
        WillReturnError(sql.ErrNoRows)

    _, err = d.GetJob("js-123")
    if err == nil || err.Error() != "job not found: js-123" {
        t.Fatalf("expected not found error, got %v", err)
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestGetJob_ScanOtherError(t *testing.T) {
    mockDB, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock.New error: %v", err) }
    defer mockDB.Close()

    d := &DB{DB: mockDB}

    mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, jobspec_id, jobspec_data, status, created_at, updated_at FROM jobs WHERE jobspec_id = $1`)).
        WithArgs("js-err").
        WillReturnError(errors.New("boom"))

    _, err = d.GetJob("js-err")
    if err == nil || !regexp.MustCompile(`failed to retrieve job`).MatchString(err.Error()) {
        t.Fatalf("expected wrapped error, got %v", err)
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestGetJob_Success(t *testing.T) {
    mockDB, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock.New error: %v", err) }
    defer mockDB.Close()

    d := &DB{DB: mockDB}

    rows := sqlmock.NewRows([]string{"id", "jobspec_id", "jobspec_data", "status", "created_at", "updated_at"}).
        AddRow(1, "js-ok", []byte(`{"a":1}`), "queued", "now", "now")

    mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, jobspec_id, jobspec_data, status, created_at, updated_at FROM jobs WHERE jobspec_id = $1`)).
        WithArgs("js-ok").
        WillReturnRows(rows)

    job, err := d.GetJob("js-ok")
    if err != nil { t.Fatalf("unexpected err: %v", err) }
    if job == nil || job.JobSpecID != "js-ok" || job.Status != "queued" {
        t.Fatalf("unexpected job: %#v", job)
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}
