package store

import (
    "context"
    "encoding/json"
    "regexp"
    "testing"
    "time"

    sqlmock "github.com/DATA-DOG/go-sqlmock"
    "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func TestListJobsByStatus_DefaultLimitAndOneRow(t *testing.T) {
    db, mock, _ := sqlmock.New()
    defer db.Close()

    repo := NewJobsRepo(db)

    // Prepare a minimal valid JobSpec JSON
    js := &models.JobSpec{ID: "job-1", Version: "1.0.0", CreatedAt: time.Now()}
    data, _ := json.Marshal(js)

    rows := sqlmock.NewRows([]string{"jobspec_data"}).AddRow(data)

    mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT jobspec_data 
        FROM jobs 
        WHERE status = $1 
        ORDER BY created_at DESC 
        LIMIT $2`)).
        WithArgs("queued", 50).
        WillReturnRows(rows)

    got, err := repo.ListJobsByStatus(context.Background(), "queued", 0)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(got) != 1 || got[0].ID != "job-1" {
        t.Fatalf("unexpected result: %+v", got)
    }

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}
