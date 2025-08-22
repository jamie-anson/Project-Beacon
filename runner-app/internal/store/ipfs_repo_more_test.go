package store

import (
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
)

func TestGetBundleByJobID_NoRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil { t.Fatalf("sqlmock.New: %v", err) }
	defer db.Close()

	repo := NewIPFSRepo(db)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, job_id, cid, bundle_size, execution_count, regions, created_at, pinned_at, gateway_url
        FROM ipfs_bundles 
        WHERE job_id = $1
        ORDER BY created_at DESC
        LIMIT 1`)).
		WithArgs("job-x").
		WillReturnError(sql.ErrNoRows)

	b, err := repo.GetBundleByJobID("job-x")
	if err != nil { t.Fatalf("unexpected err: %v", err) }
	if b != nil { t.Fatalf("expected nil bundle on no rows, got %#v", b) }
	if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}

func TestGetBundleByCID_NoRows(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock.New: %v", err) }
    defer db.Close()

    repo := NewIPFSRepo(db)

    mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, job_id, cid, bundle_size, execution_count, regions, created_at, pinned_at, gateway_url
        FROM ipfs_bundles 
        WHERE cid = $1`)).
        WithArgs("notfound").
        WillReturnError(sql.ErrNoRows)

    b, err := repo.GetBundleByCID("notfound")
    if err != nil { t.Fatalf("unexpected err: %v", err) }
    if b != nil { t.Fatalf("expected nil on no rows, got %#v", b) }
    if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}

func TestCreateBundle_Success(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock.New: %v", err) }
    defer db.Close()

    repo := NewIPFSRepo(db)
    now := time.Now()
    gw := "https://gw/ipfs/cid"
    size := int64(123)
    b := &IPFSBundle{JobID: "job-7", CID: "bafyX", BundleSize: &size, ExecutionCount: 2, Regions: []string{"us","eu"}, GatewayURL: &gw}

    mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO ipfs_bundles (job_id, cid, bundle_size, execution_count, regions, pinned_at, gateway_url)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id, created_at`)).
        WithArgs("job-7", "bafyX", &size, 2, sqlmock.AnyArg(), (*time.Time)(nil), &gw).
        WillReturnRows(sqlmock.NewRows([]string{"id","created_at"}).AddRow(99, now))

    if err := repo.CreateBundle(b); err != nil { t.Fatalf("CreateBundle err: %v", err) }
    if b.ID != 99 || b.CreatedAt.IsZero() == true { t.Fatalf("unexpected bundle after insert: %#v", b) }
    if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}

func TestUpdateExecutionCIDByID_Success(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock.New: %v", err) }
    defer db.Close()

    repo := NewIPFSRepo(db)

    mock.ExpectExec(regexp.QuoteMeta(`UPDATE executions 
        SET ipfs_cid = $1, ipfs_pinned_at = CURRENT_TIMESTAMP
        WHERE id = $2`)).
        WithArgs("bafyCID", 123).
        WillReturnResult(sqlmock.NewResult(0, 1))

    if err := repo.UpdateExecutionCIDByID(123, "bafyCID"); err != nil {
        t.Fatalf("UpdateExecutionCIDByID error: %v", err)
    }
    if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}

func TestGetBundleByJobID_Success(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock.New: %v", err) }
    defer db.Close()

    repo := NewIPFSRepo(db)

    now := time.Now()
    gw := "http://gw/ipfs/cid"
    rows := sqlmock.NewRows([]string{"id","job_id","cid","bundle_size","execution_count","regions","created_at","pinned_at","gateway_url"}).
        AddRow(2, "job-9", "bafyZ", nil, 3, pq.Array([]string{"us"}), now, now, gw)

    mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, job_id, cid, bundle_size, execution_count, regions, created_at, pinned_at, gateway_url
        FROM ipfs_bundles 
        WHERE job_id = $1
        ORDER BY created_at DESC
        LIMIT 1`)).
        WithArgs("job-9").
        WillReturnRows(rows)

    b, err := repo.GetBundleByJobID("job-9")
    if err != nil { t.Fatalf("unexpected err: %v", err) }
    if b == nil || b.JobID != "job-9" || b.CID != "bafyZ" || b.ExecutionCount != 3 {
        t.Fatalf("unexpected bundle: %#v", b)
    }
    if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}

func TestGetExecutionsByJobID_Success(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock.New: %v", err) }
    defer db.Close()

    repo := NewIPFSRepo(db)

    rows := sqlmock.NewRows([]string{
        "id", "job_id", "region", "provider_id", "status", "started_at", "completed_at",
        "created_at", "output_data", "receipt_data", "ipfs_cid", "ipfs_pinned_at",
    }).
        AddRow(7, 42, "eu", "prov-x", "done", nil, nil, time.Now(), nil, nil, nil, nil)

    mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, job_id, region, provider_id, status, started_at, completed_at, 
               created_at, output_data, receipt_data, ipfs_cid, ipfs_pinned_at
        FROM executions 
        WHERE job_id = $1
        ORDER BY created_at DESC`)).
        WithArgs("42").
        WillReturnRows(rows)

    execs, err := repo.GetExecutionsByJobID("42")
    if err != nil { t.Fatalf("unexpected err: %v", err) }
    if len(execs) != 1 || execs[0].Region != "eu" || execs[0].ProviderID != "prov-x" {
        t.Fatalf("unexpected execs: %#v", execs)
    }
    if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}

func TestGetExecutionsByCID_Success(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock.New: %v", err) }
    defer db.Close()

    repo := NewIPFSRepo(db)

    rows := sqlmock.NewRows([]string{
        "id", "job_id", "region", "provider_id", "status", "started_at", "completed_at",
        "created_at", "output_data", "receipt_data", "ipfs_cid", "ipfs_pinned_at",
    }).
        AddRow(9, 2, "us", "prov-y", "ok", nil, nil, time.Now(), nil, nil, "bafyCID", nil)

    mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, job_id, region, provider_id, status, started_at, completed_at, 
               created_at, output_data, receipt_data, ipfs_cid, ipfs_pinned_at
        FROM executions 
        WHERE ipfs_cid = $1
        ORDER BY created_at DESC`)).
        WithArgs("bafyCID").
        WillReturnRows(rows)

    execs, err := repo.GetExecutionsByCID("bafyCID")
    if err != nil { t.Fatalf("unexpected err: %v", err) }
    if len(execs) != 1 || execs[0].Region != "us" || execs[0].IPFSCid.String != "bafyCID" {
        t.Fatalf("unexpected execs: %#v", execs)
    }
    if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}

func TestGetBundleByCID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil { t.Fatalf("sqlmock.New: %v", err) }
	defer db.Close()

	repo := NewIPFSRepo(db)

	now := time.Now()
	gw := "http://gw/ipfs/cid"
	rows := sqlmock.NewRows([]string{"id","job_id","cid","bundle_size","execution_count","regions","created_at","pinned_at","gateway_url"}).
		AddRow(1, "job-1", "bafy..", nil, 2, pq.Array([]string{"us","eu"}), now, now, gw)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, job_id, cid, bundle_size, execution_count, regions, created_at, pinned_at, gateway_url
        FROM ipfs_bundles 
        WHERE cid = $1`)).
		WithArgs("bafy..").
		WillReturnRows(rows)

	b, err := repo.GetBundleByCID("bafy..")
	if err != nil { t.Fatalf("unexpected err: %v", err) }
	if b == nil || b.JobID != "job-1" || b.ExecutionCount != 2 || len(b.Regions) != 2 {
		t.Fatalf("unexpected bundle: %#v", b)
	}
	if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}

func TestListBundles_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil { t.Fatalf("sqlmock.New: %v", err) }
	defer db.Close()

	repo := NewIPFSRepo(db)

	rows := sqlmock.NewRows([]string{"id","job_id","cid","bundle_size","execution_count","regions","created_at","pinned_at","gateway_url"})
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, job_id, cid, bundle_size, execution_count, regions, created_at, pinned_at, gateway_url
        FROM ipfs_bundles 
        ORDER BY created_at DESC
        LIMIT $1 OFFSET $2`)).
		WithArgs(10, 0).
		WillReturnRows(rows)

	list, err := repo.ListBundles(10, 0)
	if err != nil { t.Fatalf("unexpected err: %v", err) }
	if len(list) != 0 { t.Fatalf("expected empty list, got %d", len(list)) }
	if err := mock.ExpectationsWereMet(); err != nil { t.Fatalf("unmet: %v", err) }
}
