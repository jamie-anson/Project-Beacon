# Database Persistence Bug in Cross-Region Execution

## Executive Summary

**Status**: P0 - Production data loss issue  
**Impact**: All cross-region job results lost, jobs stuck in "queued" status  
**Root Cause**: Likely async goroutine context lifecycle or missing job status update  
**Priority**: Fix after queue tests (separate workstream)

---

## Problem Statement

Cross-region job executions complete successfully but results are not persisted to the database. Jobs remain in "queued" status and no execution records are created, despite the execution completing without errors.

**Key Observation**: Repository methods execute without errors, but data doesn't persist. This suggests either:
1. Transaction not being committed
2. Context being cancelled before commit
3. Writing to wrong tables (cross_region_* vs executions)
4. Job status never updated from "queued" to "completed"

## Context

**Project**: Project Beacon Runner (Go backend)
**Component**: Cross-region execution handler (`internal/handlers/cross_region_handlers.go`)
**Database**: PostgreSQL (accessed via `crossRegionRepo`)

## Symptoms

1. **Job Status**: Jobs remain in `status: "queued"` after successful completion
2. **Execution Records**: `/api/v1/jobs/{job_id}/executions/all` returns `{"executions": null}`
3. **Cross-Region Records**: `/api/v1/executions/{execution_id}/cross-region` returns 0 region_results
4. **No Errors**: Database operations complete without logging any errors
5. **Execution Success**: Logs confirm successful execution:
   - `Cross-region execution completed status=completed success_count=2 total_regions=2`
   - Both US and EU providers complete successfully

## Test Case

**Job ID**: `bias-detection-1760288315095`
**Cross-Region Execution ID**: `ecf59daf-4e84-4f09-8391-fd7e792d8b5f`

**Timeline**:
- 16:58:35 - Job submitted, signature verified
- 16:58:35 - Cross-region execution started on US + EU providers
- 16:59:55 - EU provider completed successfully
- 17:00:27 - US provider completed successfully
- 17:00:27 - `Cross-region execution completed status=completed success_count=2`

**Current State**:
```bash
curl -s "https://beacon-runner-production.fly.dev/api/v1/jobs/bias-detection-1760288315095" | jq ".status"
# Returns: "queued"

curl -s "https://beacon-runner-production.fly.dev/api/v1/jobs/bias-detection-1760288315095/executions/all" | jq ".executions | length"
# Returns: 0

curl -s "https://beacon-runner-production.fly.dev/api/v1/executions/ecf59daf-4e84-4f09-8391-fd7e792d8b5f/cross-region" | jq ".region_results | length"
# Returns: 0
```

## Code Location

**File**: `/Users/Jammie/Desktop/Project Beacon/runner-app/internal/handlers/cross_region_handlers.go`

**Relevant Section**: Lines 167-266 (async goroutine in `SubmitCrossRegionJob`)

The handler:
1. Creates a cross-region execution record (line 150-165)
2. Starts async goroutine (line 168)
3. Executes across regions (line 174)
4. Updates cross-region execution status (line 195-205)
5. Creates and updates region results (line 207-259)

**Error Logging Added** (lines 204, 216, 258):
- No errors appear in production logs
- Database operations complete silently without persisting data

## Recent Changes

1. **Context Fix** (commit `d77c422`):
   - Changed from `c.Request.Context()` to `execCtx` (background context)
   - Prevents context cancellation after HTTP response
   - Applied to lines 179, 196, 208, 244

2. **Error Logging** (commit `1bcc65d`):
   - Added error logging to all database operations
   - No errors logged in production

## Database Schema

**Tables Involved**:
- `jobs` - Main job records (status should update from "queued" to "completed")
- `executions` - Standard execution records for region/model runs
- `outbox` - Outbox rows for reliable publish to Redis
- `idempotency_keys` - Prevent duplicate job creation
- Note: If `cross_region_executions` / `region_results` are present, confirm mapping and decide to either (a) continue using them and bridge to standard endpoints, or (b) unify persistence on `executions` with a single source of truth.

**Repository**: `internal/store/cross_region_repo.go`

**Methods Called**:
- `UpdateCrossRegionExecutionStatus(ctx, id, status, successCount, failureCount, completedAt, durationMs)`
- `CreateRegionResult(ctx, executionID, region, startedAt)`
- `UpdateRegionResult(ctx, id, status, completedAt, durationMs, providerID, output, error, scoring, metadata)`

## Investigation Questions (Priority Order)

### 🔴 CRITICAL: Job Status Update Missing?
**Hypothesis**: Handler never updates `jobs.status` from "queued" to "completed"

**Check**:
```bash
grep -n "UPDATE jobs SET status" internal/handlers/cross_region_handlers.go
grep -n "UpdateJobStatus" internal/handlers/cross_region_handlers.go
```

**Expected**: Should find code that updates job status after execution completes  
**If Missing**: This is the smoking gun - add job status update after line 205

---

### 🟡 HIGH: Transaction Commit Issue?
**Hypothesis**: Repo methods use transactions but don't commit

**Check**:
```bash
grep -A 10 "func.*UpdateCrossRegionExecutionStatus" internal/store/cross_region_repo.go | grep -E "Begin|Commit|Rollback"
```

**Expected**: Either auto-commit (no explicit transaction) or explicit `tx.Commit()`  
**If Using Transactions**: Verify commit is called and not deferred rollback

---

### 🟡 HIGH: Context Cancellation?
**Hypothesis**: `context.Background()` in goroutine gets cancelled somehow

**Check**:
```go
// In cross_region_handlers.go, verify:
execCtx := context.Background()
// Should NOT be:
execCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel() // This would cancel before DB writes complete
```

**Test**: Add `time.Sleep(5 * time.Second)` before DB writes to ensure goroutine completes

---

### 🟢 MEDIUM: Table Mismatch?
**Hypothesis**: Writing to `cross_region_executions` but portal reads from `executions`

**Check**:
```sql
-- Verify both tables exist
SELECT table_name FROM information_schema.tables 
WHERE table_name IN ('executions', 'cross_region_executions', 'region_results');

-- Check if data is in cross_region tables but not exposed via API
SELECT COUNT(*) FROM cross_region_executions;
SELECT COUNT(*) FROM region_results;
SELECT COUNT(*) FROM executions;
```

**If Data Exists in cross_region_***: Need to bridge to standard API endpoints

---

### 🟢 LOW: Repository Implementation
**Hypothesis**: Repo methods are no-ops

**Check**: We already verified repo methods ARE implemented (lines 82-213 in cross_region_repo.go)  
**Status**: ✅ Methods exist and have proper SQL queries

## Expected Behavior

After successful cross-region execution:

1. **Job Status**: `jobs.status` should update from "queued" to "completed"
2. **Cross-Region Execution**: `cross_region_executions` should show:
   - `status: "completed"`
   - `success_count: 2`
   - `failure_count: 0`
   - `completed_at: <timestamp>`
   - `duration_ms: <duration>`

3. **Region Results**: `region_results` should contain 2 records (US + EU):
   - `region: "US"` and `region: "EU"`
   - `status: "success"`
   - `provider_id: "modal-us-east"` / `"modal-eu-west"`
   - `execution_output: <receipt data>`
   - `completed_at: <timestamp>`

4. **API Responses**:
   - `/api/v1/jobs/{job_id}` should return `status: "completed"`
   - `/api/v1/jobs/{job_id}/executions/all` should return execution records
   - `/api/v1/executions/{execution_id}/cross-region` should return region results

## Files to Investigate

1. **Handler**: `internal/handlers/cross_region_handlers.go` (lines 167-266)
2. **Repository**: `internal/store/cross_region_repo.go`
3. **Database Schema**: `migrations/` (look for cross_region_executions, region_results tables)
4. **Executor**: `internal/execution/cross_region_executor.go` (returns results but doesn't persist)

## Debugging Steps (Execute in Order)

### Step 1: Enable DB_AUDIT Logging 🔍
**Goal**: See exactly what SQL is executing and if it's succeeding

```bash
# In production (Fly.io)
flyctl secrets set DB_AUDIT=1 -a beacon-runner-production
flyctl deploy -a beacon-runner-production

# Watch logs for DB operations
flyctl logs -a beacon-runner-production | grep "DB_AUDIT"
```

**Expected Output**:
```
DB_AUDIT update_cross_region_execution_status id=xxx status=completed rows=1 elapsed_ms=45
DB_AUDIT create_region_result id=xxx exec_id=xxx region=US rows=1 elapsed_ms=23
DB_AUDIT update_region_result id=xxx rows=1 elapsed_ms=31
```

**If rows=0**: SQL is executing but not matching any records (wrong ID?)  
**If no output**: DB operations not being called at all

---

### Step 2: Check Job Status Update Code 🎯
**Goal**: Verify handler updates job status after execution

```bash
cd /Users/Jammie/Desktop/Project\ Beacon/runner-app
grep -n "jobs.status" internal/handlers/cross_region_handlers.go
grep -n "UpdateJob" internal/handlers/cross_region_handlers.go
grep -n "status.*completed" internal/handlers/cross_region_handlers.go
```

**If No Results**: **THIS IS THE BUG** - handler never updates job status

**Fix Required**:
```go
// After line 205 in cross_region_handlers.go, add:
if err := h.jobsRepo.UpdateJobStatus(execCtx, jobID, "completed"); err != nil {
    log.Printf("ERROR: Failed to update job status: %v", err)
}
```

---

### Step 3: Verify Data in Database 📊
**Goal**: Check if data exists but isn't exposed via API

```sql
-- Connect to production DB
psql $DATABASE_URL

-- Check all tables for the test job
SELECT 'jobs' as table_name, COUNT(*) as count 
FROM jobs WHERE id = 'bias-detection-1760288315095'
UNION ALL
SELECT 'cross_region_executions', COUNT(*) 
FROM cross_region_executions WHERE jobspec_id = 'bias-detection-1760288315095'
UNION ALL
SELECT 'region_results', COUNT(*) 
FROM region_results WHERE cross_region_execution_id IN (
    SELECT id FROM cross_region_executions WHERE jobspec_id = 'bias-detection-1760288315095'
)
UNION ALL
SELECT 'executions', COUNT(*) 
FROM executions WHERE job_id = 'bias-detection-1760288315095';
```

**Interpretation**:
- `jobs = 1, cross_region_executions = 0`: Job created but execution never recorded
- `jobs = 1, cross_region_executions = 1, region_results = 0`: Execution created but regions not recorded
- `jobs = 1, cross_region_executions = 1, region_results = 2`: Data exists! API not bridging to standard endpoints
- `All = 0`: Complete data loss (most severe)

---

### Step 4: Test Repository Methods Directly 🧪
**Goal**: Verify repo methods work outside async goroutine

```go
// Create test file: internal/store/cross_region_repo_test.go
func TestCrossRegionRepo_Integration(t *testing.T) {
    // Setup DB connection
    db := setupTestDB(t)
    repo := NewCrossRegionRepo(db)
    
    ctx := context.Background()
    
    // Test create
    exec, err := repo.CreateCrossRegionExecution(ctx, "test-job", 2, 1, 0.67)
    require.NoError(t, err)
    require.NotEmpty(t, exec.ID)
    
    // Test update
    now := time.Now()
    duration := int64(1000)
    err = repo.UpdateCrossRegionExecutionStatus(ctx, exec.ID, "completed", 2, 0, &now, &duration)
    require.NoError(t, err)
    
    // Verify data persisted
    var status string
    err = db.QueryRow("SELECT status FROM cross_region_executions WHERE id = $1", exec.ID).Scan(&status)
    require.NoError(t, err)
    assert.Equal(t, "completed", status)
}
```

**Run**: `go test ./internal/store -run TestCrossRegionRepo_Integration -v`

**If Test Passes**: Repo works fine, issue is in handler goroutine  
**If Test Fails**: Repo has fundamental issue

---

### Step 5: Add Goroutine Completion Logging 📝
**Goal**: Verify goroutine completes and doesn't panic

```go
// In cross_region_handlers.go, line 168:
go func() {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("PANIC in cross-region goroutine: %v", r)
        }
        log.Printf("Cross-region goroutine completed for job %s", jobID)
    }()
    
    // ... existing goroutine code ...
}()
```

**Expected Log**: `Cross-region goroutine completed for job bias-detection-xxx`  
**If Missing**: Goroutine is panicking or hanging

## Success Criteria

After fixing:
- Submit a new cross-region job
- Verify job status updates to "completed"
- Verify execution records appear in `/api/v1/jobs/{job_id}/executions/all`
- Verify region results appear in `/api/v1/executions/{execution_id}/cross-region`
- Verify no errors in logs


## Root-Cause Hypotheses (Ranked by Likelihood)

### 🔴 MOST LIKELY: Missing Job Status Update
**Probability**: 90%  
**Evidence**: 
- Logs show "Cross-region execution completed" but job stays "queued"
- Handler updates `cross_region_executions` but likely never touches `jobs.status`
- This is a common oversight in async handlers

**Verification**: Search for job status update in handler (Step 2 above)  
**Fix**: Add `UpdateJobStatus()` call after execution completes

---

### 🟡 LIKELY: Table Mismatch (Data Exists, Not Exposed)
**Probability**: 60%  
**Evidence**:
- Repo methods execute without errors
- Data might be in `cross_region_executions` / `region_results` tables
- Portal queries `/api/v1/jobs/{id}/executions/all` which reads from `executions` table

**Verification**: Check database directly (Step 3 above)  
**Fix**: Bridge cross_region tables to standard API or consolidate on `executions`

---

### 🟡 POSSIBLE: Transaction Not Committed
**Probability**: 40%  
**Evidence**:
- Repo methods use `db.ExecContext()` which should auto-commit
- But if wrapped in transaction elsewhere, might not commit

**Verification**: Check for `BEGIN` / `COMMIT` in repo methods  
**Fix**: Ensure explicit `tx.Commit()` if using transactions

---

### 🟢 UNLIKELY: Context Cancellation
**Probability**: 20%  
**Evidence**:
- Handler uses `context.Background()` which never cancels
- Recent fix specifically changed from request context to background

**Verification**: Check for `WithTimeout` or `WithCancel` wrapping  
**Fix**: Remove timeout/cancel or increase timeout

---

### 🟢 UNLIKELY: Neon Connection Issues
**Probability**: 10%  
**Evidence**:
- Would see errors in logs (none reported)
- Other DB operations work fine

**Verification**: Enable `DB_AUDIT=1` to see connection errors  
**Fix**: Add retry logic for transient failures

## Quick Fix (Immediate Action) ⚡

**If Step 2 confirms missing job status update**:

1. **Add Job Status Update** in `internal/handlers/cross_region_handlers.go`:

```go
// After line 205 (after UpdateCrossRegionExecutionStatus):
if finalStatus == "completed" || finalStatus == "failed" {
    if err := h.jobsRepo.UpdateJobStatus(execCtx, jobID, finalStatus); err != nil {
        log.Printf("ERROR: Failed to update job status to %s: %v", finalStatus, err)
    } else {
        log.Printf("Updated job %s status to %s", jobID, finalStatus)
    }
}
```

2. **Verify JobsRepo has UpdateJobStatus method**:

```bash
grep -n "UpdateJobStatus" internal/store/jobs_repo.go
```

**If method doesn't exist**, add it:

```go
// In internal/store/jobs_repo.go:
func (r *JobsRepo) UpdateJobStatus(ctx context.Context, jobID string, status string) error {
    query := `UPDATE jobs SET status = $1, updated_at = NOW() WHERE id = $2`
    result, err := r.db.ExecContext(ctx, query, status, jobID)
    if err != nil {
        return fmt.Errorf("failed to update job status: %w", err)
    }
    
    rows, _ := result.RowsAffected()
    if rows == 0 {
        return fmt.Errorf("job %s not found", jobID)
    }
    
    return nil
}
```

3. **Deploy and Test**:

```bash
go test ./internal/store -v
flyctl deploy -a beacon-runner-production

# Submit test job and verify status updates
curl -s "https://beacon-runner-production.fly.dev/api/v1/jobs/{job_id}" | jq .status
```

---

## Remediation Plan (Production Grade)

- **[A. Instrumentation]**
  - Add `DB_AUDIT=1` toggle to log write paths: operation, rows_affected, SQLSTATE, elapsed ms.
  - Enhance readiness: startup DB ping + simple write/read round-trip.
  - Scope: `internal/store/*_repo.go`, `internal/service/*` write paths.

- **[B. Atomic Transactions]**
  - Implement `CreateJobAtomic(ctx, spec, idempotencyKey)` in `internal/service/jobs.go`:
    - Insert job (RETURNING id).
    - Insert idempotency key (ON CONFLICT DO NOTHING), verify idempotent behavior.
    - Insert outbox row carrying envelope referencing the job.
    - All in a single transaction; rollback on any error.
  - Ensure cross-region completion updates (job status, execution records, analytics) are grouped in coherent transactions where applicable.

- **[C. Uniqueness & Bridging]**
  - Enforce `UNIQUE (job_id, region, model_id)` on `executions` and treat conflicts as idempotent success.
  - If `cross_region_executions/region_results` are used, write bridging logic to expose results via standard `/api/v1/jobs/{job_id}/executions/all` endpoint, or consolidate persistence on `executions` to match the portal contract.

- **[D. Resilience]**
  - Add safe retries for transient SQLSTATEs: `57P01`, `08006`, `40001` with bounded backoff.
  - Use request-scoped contexts (derived from parent) with sufficient per-write timeout (5–8s) instead of `context.Background()`.
  - Pool settings: `MaxOpenConns=10`, `MaxIdleConns=10`, `ConnMaxIdleTime=5m`, `sslmode=require`.

- **[E. Outbox Publisher Semantics]**
  - Claim rows with `FOR UPDATE SKIP LOCKED LIMIT N` and mark `processed_at` within the same transaction that publishes to Redis.
  - Keep envelope format exactly as `internal/worker/job_runner.go` expects (prior fix: no extra wrapping).

## Implementation Tasks (code map)

- **[service]** `internal/service/jobs.go`: add `CreateJobAtomic(...)` orchestration.
- **[repos]**
  - `internal/store/jobs_repo.go`: `InsertJobTx(ctx, tx, spec) RETURNING id`.
  - `internal/store/idempotency_repo.go`: `InsertKeyTx(ctx, tx, key) ON CONFLICT DO NOTHING`.
  - `internal/store/outbox_repo.go`: `InsertOutboxTx(ctx, tx, payload)` and claim/mark helpers.
  - `internal/store/executions_repo.go`: idempotent `InsertExecutionWithModel(...)` honoring uniqueness.
- **[publisher]** `internal/worker/outbox_publisher.go`: atomic claim → publish → mark processed.
- **[handlers]** `internal/handlers/cross_region_handlers.go`: ensure goroutine uses parent-derived `ctx` with sane timeouts and invokes repo methods that commit.
- **[metrics]** Export `db_write_errors_total{sqlstate}`, `db_write_retries_total`, `db_write_latency_ms`.

## SQL Migrations (DDL)

```sql
-- Executions uniqueness to prevent duplicates (safe idempotency)
CREATE UNIQUE INDEX IF NOT EXISTS executions_job_region_model_uidx
  ON executions (job_id, region, model_id);

-- Idempotency key uniqueness
CREATE UNIQUE INDEX IF NOT EXISTS idempotency_keys_key_uidx
  ON idempotency_keys (key);

-- Outbox claiming efficiency
CREATE INDEX IF NOT EXISTS outbox_unprocessed_idx
  ON outbox (processed_at, id);
```

## Verification & Rollout

- **[local repro]** Submit signed job; verify `jobs`, `executions`, `outbox`, `idempotency_keys` rows created; runner consumes outbox and updates status to `completed`.
- **[staging canary]** Enable `DB_AUDIT=1`; run scripted canary; confirm metrics/logs.
- **[prod canary]** Off-hours deploy; keep `DB_AUDIT=1` for 30–60m; rollback on elevated error-rate.

### Terminal quick refs (local)

- **[Terminal D]** docker compose up (Postgres, Redis)
- **[Terminal B]** `go run cmd/runner/main.go` (port 8090)
- **[Terminal C]** smoke:
```bash
curl -s http://localhost:8090/health | jq .service
curl -s http://localhost:8090/api/v1/jobs/{job_id} | jq .status
```

## Additional Context

- **Signature Verification**: ✅ Working correctly (separate issue, now fixed)
- **Job Execution**: ✅ Completes successfully on providers
- **Only Issue**: Database persistence in cross-region path
- **Environment**: Production (Fly.io), PostgreSQL (Neon)
- **Go Version**: 1.24
- **Framework**: Gin (HTTP), GORM or pgx (database - check which one)

## Related Files

- `internal/api/routes.go` (lines 70-101) - Handler initialization
- `internal/execution/cross_region_executor.go` - Execution logic
- `internal/store/executions_repo.go` - Standard executions (for comparison)
- `pkg/models/jobspec.go` - Job model

## ✅ Investigation Complete - Code Review

**Portal Endpoint**: ✅ Correctly calls `/jobs/cross-region`  
**Handler Code**: ✅ Job status update exists (line 270)  
**UpdateJobStatus Method**: ✅ Implemented correctly  
**Handler Initialization**: ✅ jobsRepo properly passed in  

**Conclusion**: Code is correct, issue is runtime behavior.

---

## Action Plan Summary

### Phase 1: Investigation (30 minutes) ✅ COMPLETE
- [x] Search for job status update code → FOUND at line 270
- [x] Verify UpdateJobStatus method exists → CONFIRMED
- [x] Check portal endpoint → CORRECT `/jobs/cross-region`
- [ ] Enable `DB_AUDIT=1` logging → NEXT STEP
- [ ] Query database for existing data → BLOCKED (need DB access)

### Phase 2: Quick Fix (1 hour)
- [ ] Add missing job status update (if confirmed)
- [ ] Add `UpdateJobStatus()` method to JobsRepo (if missing)
- [ ] Add goroutine completion logging
- [ ] Deploy to production

### Phase 3: Verification (30 minutes)
- [ ] Submit new test job
- [ ] Verify job status updates to "completed"
- [ ] Verify execution records appear in API
- [ ] Check logs for DB_AUDIT output

### Phase 4: Long-term Fixes (separate ticket)
- [ ] Implement atomic transactions (Section B)
- [ ] Add uniqueness constraints (Section C)
- [ ] Improve resilience with retries (Section D)
- [ ] Fix outbox publisher semantics (Section E)
- [ ] Bridge cross_region tables to standard API

---

## Notes

- This appears to be a pre-existing architectural issue, not introduced by recent changes
- **Most likely cause**: Missing job status update in async goroutine handler
- The cross-region execution path may have been designed to use separate tables (`cross_region_executions`, `region_results`) instead of the standard `executions` table
- The portal expects results in the standard `/api/v1/jobs/{job_id}/executions/all` endpoint
- May need to bridge the two data models or update the portal to use cross-region endpoints

---

## Success Metrics

**After Fix**:
- ✅ Jobs transition from "queued" → "completed" status
- ✅ Execution records visible in `/api/v1/jobs/{id}/executions/all`
- ✅ Cross-region results visible in `/api/v1/executions/{id}/cross-region`
- ✅ No errors in production logs
- ✅ `DB_AUDIT` logs show successful writes with `rows=1`

**Monitoring**:
- Track job status distribution (should see "completed" and "failed", not just "queued")
- Monitor `db_write_errors_total` metric (should be 0)
- Alert on jobs stuck in "queued" for >10 minutes
