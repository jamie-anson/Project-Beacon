package store

import (
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestUpdateExecutionCID_UsesPrimaryKeyID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewIPFSRepo(db)

	// Expect the correct UPDATE with WHERE id = $2
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE executions 
        SET ipfs_cid = $1, ipfs_pinned_at = CURRENT_TIMESTAMP
        WHERE id = $2`)).
		WithArgs("bafyCID", "123").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.UpdateExecutionCID("123", "bafyCID"); err != nil {
		t.Fatalf("UpdateExecutionCID error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetExecutionsByJobSpecID_JoinsJobs(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewIPFSRepo(db)

	rows := sqlmock.NewRows([]string{
		"id", "job_id", "region", "provider_id", "status", "started_at", "completed_at",
		"created_at", "output_data", "receipt_data", "ipfs_cid", "ipfs_pinned_at",
	}).
		AddRow(1, 10, "eu", "prov-1", "completed", nil, nil, time.Now(), nil, nil, nil, nil)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT e.id, e.job_id, e.region, e.provider_id, e.status, e.started_at, e.completed_at,
               e.created_at, e.output_data, e.receipt_data, e.ipfs_cid, e.ipfs_pinned_at
        FROM executions e
        JOIN jobs j ON e.job_id = j.id
        WHERE j.jobspec_id = $1
        ORDER BY e.created_at DESC`)).
		WithArgs("job-001").
		WillReturnRows(rows)

	execs, err := repo.GetExecutionsByJobSpecID("job-001")
	if err != nil {
		t.Fatalf("GetExecutionsByJobSpecID error: %v", err)
	}
	if len(execs) != 1 {
		t.Fatalf("expected 1 exec, got %d", len(execs))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
