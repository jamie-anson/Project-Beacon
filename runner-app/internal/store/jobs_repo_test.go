package store

import (
    "context"
    "database/sql"
    "testing"

    sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestUpdateJobStatus_NoRowsAffected_ReturnsError(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    repo := NewJobsRepo(db)

    mock.ExpectExec("UPDATE jobs\\s+SET status = ").
        WithArgs("running", "missing-id").
        WillReturnResult(sqlmock.NewResult(0, 0))

    err := repo.UpdateJobStatus(context.Background(), "missing-id", "running")
    if err == nil {
        t.Fatalf("expected error when no rows affected")
    }

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestGetJobByID_NotFound(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    repo := NewJobsRepo(db)

    mock.ExpectQuery("SELECT jobspec_data, status, created_at, updated_at\\s+FROM jobs\\s+WHERE jobspec_id = ").
        WithArgs("nope").
        WillReturnError(sql.ErrNoRows)

    _, _, err := repo.GetJobByID(context.Background(), "nope")
    if err == nil {
        t.Fatalf("expected not found error")
    }

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}
