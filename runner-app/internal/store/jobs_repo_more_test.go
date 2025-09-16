//go:build !ci
// +build !ci

package store

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func TestJobsRepo_GetJobByID_NotFound(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock new: %v", err) }
    defer db.Close()

    repo := NewJobsRepo(db)
    ctx := context.Background()
    jobID := "job-404"

    mock.ExpectQuery(regexp.QuoteMeta(`SELECT jobspec_data, status, created_at, updated_at 
        FROM jobs 
        WHERE jobspec_id = $1`)).
        WithArgs(jobID).
        WillReturnError(sql.ErrNoRows)

    js, status, err := repo.GetJobByID(ctx, jobID)
    if js != nil || status != "" || err == nil {
        t.Fatalf("expected not found error, got js=%v status=%q err=%v", js, status, err)
    }
    if !regexp.MustCompile(`job not found`).MatchString(err.Error()) {
        t.Fatalf("expected error to mention job not found, got %v", err)
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("expectations: %v", err)
    }
}

func TestJobsRepo_CreateJob_MarshalError(t *testing.T) {
    db, _, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock new: %v", err) }
    defer db.Close()

    repo := NewJobsRepo(db)
    ctx := context.Background()

    // Insert an unsupported type (func) into Metadata to force json.Marshal error
    js := &models.JobSpec{
        ID:        "job-marshal-bad",
        Version:   "1.0.0",
        CreatedAt: time.Now(),
        Benchmark: models.BenchmarkSpec{Container: models.ContainerSpec{Image: "img"}, Input: models.InputSpec{Hash: "h"}},
        Constraints: models.ExecutionConstraints{Regions: []string{"us"}},
        Metadata: map[string]interface{}{
            "bad": func() {},
        },
    }

    err = repo.CreateJob(ctx, js)
    if err == nil || !regexp.MustCompile(`failed to marshal jobspec`).MatchString(err.Error()) {
        t.Fatalf("expected marshal error, got %v", err)
    }
}

func TestJobsRepo_CreateJob_ExecError(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock new: %v", err) }
    defer db.Close()

    repo := NewJobsRepo(db)
    ctx := context.Background()

    js := &models.JobSpec{
        ID:        "job-insert-fails",
        Version:   "1.0.0",
        CreatedAt: time.Now(),
        Benchmark: models.BenchmarkSpec{Container: models.ContainerSpec{Image: "img"}, Input: models.InputSpec{Hash: "h"}},
        Constraints: models.ExecutionConstraints{Regions: []string{"us"}},
    }

    mock.ExpectExec(regexp.QuoteMeta(`
        INSERT INTO jobs (jobspec_id, jobspec_data, status, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5)
    `)).
        WithArgs(js.ID, sqlmock.AnyArg(), "queued", js.CreatedAt, sqlmock.AnyArg()).
        WillReturnError(errors.New("insert boom"))

    err = repo.CreateJob(ctx, js)
    if err == nil || !regexp.MustCompile(`failed to insert job`).MatchString(err.Error()) {
        t.Fatalf("expected wrapped exec error, got %v", err)
    }
    if e := mock.ExpectationsWereMet(); e != nil { t.Fatalf("expectations: %v", e) }
}

func TestJobsRepo_UpdateJobStatus_RowsAffectedError(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock new: %v", err) }
    defer db.Close()

    repo := NewJobsRepo(db)
    ctx := context.Background()

    mock.ExpectExec(regexp.QuoteMeta(`UPDATE jobs 
		SET status = $1, updated_at = NOW() 
		WHERE jobspec_id = $2`)).
        WithArgs("running", "job-rows-err").
        WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected err")))

    err = repo.UpdateJobStatus(ctx, "job-rows-err", "running")
    if err == nil || !regexp.MustCompile(`failed to get rows affected`).MatchString(err.Error()) {
        t.Fatalf("expected rows affected error, got %v", err)
    }
    if e := mock.ExpectationsWereMet(); e != nil { t.Fatalf("expectations: %v", e) }
}

func TestJobsRepo_ListRecentJobs_RowsErr(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock new: %v", err) }
    defer db.Close()

    repo := NewJobsRepo(db)
    ctx := context.Background()

    cols := []string{"jobspec_id", "status", "created_at"}
    rows := sqlmock.NewRows(cols).
        AddRow("job-1", "queued", time.Now()).
        RowError(0, errors.New("scan boom"))

    mock.ExpectQuery(regexp.QuoteMeta(`SELECT jobspec_id, status, created_at FROM jobs ORDER BY created_at DESC LIMIT $1`)).
        WithArgs(50).
        WillReturnRows(rows)

    // Iterate to trigger rows.Err()
    rs, err := repo.ListRecentJobs(ctx, 50)
    if err != nil { t.Fatalf("query err: %v", err) }
    defer rs.Close()
    for rs.Next() {
        var id, status string
        var createdAt sql.NullTime
        _ = rs.Scan(&id, &status, &createdAt)
    }
    if e := rs.Err(); e == nil {
        t.Fatalf("expected rows.Err to be non-nil")
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("expectations: %v", err)
    }
}

func TestJobsRepo_UpdateJobStatus_NotFound(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock new: %v", err) }
    defer db.Close()

    repo := NewJobsRepo(db)
    ctx := context.Background()
    jobID := "job-missing"

    mock.ExpectExec(regexp.QuoteMeta(`UPDATE jobs 
		SET status = $1, updated_at = NOW() 
		WHERE jobspec_id = $2`)).
        WithArgs("done", jobID).
        WillReturnResult(sqlmock.NewResult(0, 0))

    err = repo.UpdateJobStatus(ctx, jobID, "done")
    if err == nil || !regexp.MustCompile(`job not found`).MatchString(err.Error()) {
        t.Fatalf("expected job not found error, got %v", err)
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("expectations: %v", err)
    }
}

func TestJobsRepo_UpdateJobStatus_ExecError(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock new: %v", err) }
    defer db.Close()

    repo := NewJobsRepo(db)
    ctx := context.Background()
    jobID := "job-err"

    mock.ExpectExec(regexp.QuoteMeta(`UPDATE jobs 
		SET status = $1, updated_at = NOW() 
		WHERE jobspec_id = $2`)).
        WithArgs("running", jobID).
        WillReturnError(errors.New("db down"))

    err = repo.UpdateJobStatus(ctx, jobID, "running")
    if err == nil || !regexp.MustCompile(`failed to update job status`).MatchString(err.Error()) {
        t.Fatalf("expected wrapped exec error, got %v", err)
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("expectations: %v", err)
    }
}

func TestJobsRepo_GetJobByID_InvalidJSON(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil { t.Fatalf("sqlmock new: %v", err) }
	defer db.Close()

	repo := NewJobsRepo(db)
	ctx := context.Background()
	jobID := "job-badjson"

	cols := []string{"jobspec_data", "status", "created_at", "updated_at"}
	rows := sqlmock.NewRows(cols).
		AddRow([]byte(`{"id":`), "queued", time.Now(), time.Now()) // malformed JSON

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT jobspec_data, status, created_at, updated_at 
		FROM jobs 
		WHERE jobspec_id = $1`)).
		WithArgs(jobID).
		WillReturnRows(rows)

	js, status, err := repo.GetJobByID(ctx, jobID)
	if js != nil || status != "" || err == nil {
		t.Fatalf("expected unmarshal error, got js=%v status=%q err=%v", js, status, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestJobsRepo_ListJobsByStatus_RowsErr(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock new: %v", err) }
    defer db.Close()

    repo := NewJobsRepo(db)
    ctx := context.Background()

    // Provide one valid JSON row, but make rows.Err() return an error after iteration
    validJSON := []byte(`{"id":"job-1","version":"1.0.0","created_at":"2025-08-21T00:00:00Z"}`)
    cols := []string{"jobspec_data"}
    rows := sqlmock.NewRows(cols).
        AddRow(validJSON).
        RowError(0, errors.New("iter boom"))

    mock.ExpectQuery(regexp.QuoteMeta(`SELECT jobspec_data 
        FROM jobs 
        WHERE status = $1 
        ORDER BY created_at DESC 
        LIMIT $2`)).
        WithArgs("queued", 50).
        WillReturnRows(rows)

    list, err := repo.ListJobsByStatus(ctx, "queued", 0)
    if err == nil || !regexp.MustCompile(`error iterating job rows`).MatchString(err.Error()) {
        t.Fatalf("expected rows.Err wrapped error, got list=%v err=%v", list, err)
    }
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("expectations: %v", err)
    }
}

func TestJobsRepo_ListJobsByStatus_UnmarshalError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil { t.Fatalf("sqlmock new: %v", err) }
	defer db.Close()

	repo := NewJobsRepo(db)
	ctx := context.Background()
	status := "queued"

	cols := []string{"jobspec_data"}
	rows := sqlmock.NewRows(cols).
		AddRow([]byte(`{"id":`)) // malformed JSON

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT jobspec_data 
		FROM jobs 
		WHERE status = $1 
		ORDER BY created_at DESC 
		LIMIT $2`)).
		WithArgs(status, 50).
		WillReturnRows(rows)

	list, err := repo.ListJobsByStatus(ctx, status, 0)
	if list != nil || err == nil {
		t.Fatalf("expected unmarshal error, got list=%v err=%v", list, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
