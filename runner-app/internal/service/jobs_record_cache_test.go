package service

import (
    "context"
    "crypto/ed25519"
    "regexp"
    "testing"
    "time"

    "github.com/DATA-DOG/go-sqlmock"
    "github.com/jamie-anson/project-beacon-runner/internal/store"
    "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

type countingCache struct{ sets []struct{ key string; ttl time.Duration; val []byte } }
func (c *countingCache) Get(ctx context.Context, key string) ([]byte, bool, error) { return nil, false, nil }
func (c *countingCache) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error {
    c.sets = append(c.sets, struct{ key string; ttl time.Duration; val []byte }{key, ttl, val})
    return nil
}

func TestRecordExecution_CacheInvalidation(t *testing.T) {
    // sqlmock-backed DB for repos
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock.New: %v", err) }
    defer db.Close()

    s := &JobsService{}
    s.ExecutionsRepo = store.NewExecutionsRepo(db)
    s.JobsRepo = store.NewJobsRepo(db)

    cc := &countingCache{}
    s.SetCache(cc)

    // Prepare a valid signed receipt
    _, priv, err := ed25519.GenerateKey(nil)
    if err != nil { t.Fatalf("gen key: %v", err) }
    rec := models.NewReceipt("job-123", models.ExecutionDetails{Status: "completed"}, models.ExecutionOutput{Hash: "h"}, models.ProvenanceInfo{})
    if err := rec.Sign(priv); err != nil { t.Fatalf("sign receipt: %v", err) }

    // Expectations for ExecutionsRepo.CreateExecution (INSERT ... RETURNING id)
    mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO executions (job_id, provider_id, region, status, started_at, completed_at, output_data, receipt_data)
        VALUES ((SELECT id FROM jobs WHERE jobspec_id = $1), $2, $3, $4, $5, $6, $7, $8)
        RETURNING id`)).
        WithArgs("job-123", sqlmock.AnyArg(), sqlmock.AnyArg(), "completed", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
        WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

    // Expectations for JobsRepo.UpdateJobStatus (UPDATE jobs ... WHERE jobspec_id = $2)
    mock.ExpectExec(regexp.QuoteMeta(`UPDATE jobs 
        SET status = $1, updated_at = NOW() 
        WHERE jobspec_id = $2`)).
        WithArgs("running", "job-123").
        WillReturnResult(sqlmock.NewResult(0, 1))

    if err := s.RecordExecution(context.Background(), "job-123", rec); err != nil {
        t.Fatalf("RecordExecution err: %v", err)
    }
    if len(cc.sets) != 2 {
        t.Fatalf("expected 2 cache Set calls, got %d", len(cc.sets))
    }
    if cc.sets[0].key != "job:job-123" || cc.sets[1].key != "job:job-123:latest_receipt" {
        t.Fatalf("unexpected keys: %+v", cc.sets)
    }
    if cc.sets[0].ttl <= 0 || cc.sets[1].ttl <= 0 {
        t.Fatalf("expected positive TTLs: %+v", cc.sets)
    }

    if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet expectations: %v", err) }
}
