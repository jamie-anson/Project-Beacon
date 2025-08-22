package service

import (
    "database/sql"
    "context"
    "encoding/json"
    "regexp"
    "testing"
    "time"

    "github.com/DATA-DOG/go-sqlmock"
    "github.com/jamie-anson/project-beacon-runner/internal/store"
    "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

type setCountingCache struct{ sets int; lastKey string; lastTTL time.Duration; lastVal []byte }

func TestGetLatestReceiptCached_NoRows_NoCacheSet(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock.New: %v", err) }
    defer db.Close()

    s := &JobsService{}
    s.ExecutionsRepo = store.NewExecutionsRepo(db)

    mock.ExpectQuery(regexp.QuoteMeta(`SELECT e.receipt_data
        FROM executions e
        JOIN jobs j ON e.job_id = j.id
        WHERE j.jobspec_id = $1 AND e.receipt_data IS NOT NULL
        ORDER BY e.created_at DESC
        LIMIT 1`)).
        WithArgs("job-missing").
        WillReturnError(sql.ErrNoRows)

    cc := &setCountingCache{}
    s.SetCache(cc)

    got, err := s.GetLatestReceiptCached(context.Background(), "job-missing")
    if err == nil || got != nil {
        t.Fatalf("expected error and nil receipt, got rec=%v err=%v", got, err)
    }
    if cc.sets != 0 {
        t.Fatalf("expected no cache set on error, got %d sets", cc.sets)
    }
    if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet expectations: %v", err) }
}
func (c *setCountingCache) Get(ctx context.Context, key string) ([]byte, bool, error) { return nil, false, nil }
func (c *setCountingCache) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error {
    c.sets++
    c.lastKey, c.lastTTL, c.lastVal = key, ttl, val
    return nil
}

func TestGetLatestReceiptCached_CacheMissThenSet(t *testing.T) {
    // sqlmock-backed repos
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock.New: %v", err) }
    defer db.Close()

    s := &JobsService{}
    s.ExecutionsRepo = store.NewExecutionsRepo(db)

    // Prepare a receipt row the repo would return
    rec := &models.Receipt{ExecutionDetails: models.ExecutionDetails{Status: "completed"}}
    b, _ := json.Marshal(rec)

    mock.ExpectQuery(regexp.QuoteMeta(`SELECT e.receipt_data
        FROM executions e
        JOIN jobs j ON e.job_id = j.id
        WHERE j.jobspec_id = $1 AND e.receipt_data IS NOT NULL
        ORDER BY e.created_at DESC
        LIMIT 1`)).
        WithArgs("job-xyz").
        WillReturnRows(sqlmock.NewRows([]string{"receipt_data"}).AddRow(b))

    cc := &setCountingCache{}
    s.SetCache(cc)

    got, err := s.GetLatestReceiptCached(context.Background(), "job-xyz")
    if err != nil { t.Fatalf("err: %v", err) }
    if got == nil || got.ExecutionDetails.Status != "completed" {
        t.Fatalf("unexpected receipt: %#v", got)
    }
    if cc.sets != 1 {
        t.Fatalf("expected 1 cache set, got %d", cc.sets)
    }
    if cc.lastKey != "job:job-xyz:latest_receipt" || cc.lastTTL <= 0 || len(cc.lastVal) == 0 {
        t.Fatalf("unexpected cache set details: key=%s ttl=%v valLen=%d", cc.lastKey, cc.lastTTL, len(cc.lastVal))
    }

    if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet expectations: %v", err) }
}
