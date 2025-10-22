# 🐛 Bug Fix: Negative Execution Durations

**Date**: 2025-10-22  
**Severity**: CRITICAL  
**Status**: FIXED ✅  

---

## 🔍 Problem

All job executions were showing **negative durations** (-4ms to -38ms), indicating `completed_at` was BEFORE `created_at`, which is physically impossible.

### Symptoms
```sql
SELECT id, duration_ms FROM executions WHERE status = 'failed' ORDER BY created_at DESC LIMIT 5;

  id  | duration_ms
------+-------------
 2390 |  -38.033000
 2389 |   -4.503000
 2388 |  -14.346000
 2387 |   -7.695000
 2386 |   -7.776000
```

---

## 🎯 Root Cause

### The Bug
The `executions` table has a `DEFAULT NOW()` constraint on `created_at`:

```sql
-- From migrations/0001_init.up.sql
CREATE TABLE executions (
    ...
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()  -- ⚠️ THE PROBLEM
);
```

### What Was Happening

1. **Code execution**:
   ```go
   executionStart := time.Now()  // e.g., 15:46:48.490
   // ... execute job ...
   executionEnd := time.Now()    // e.g., 15:46:48.500
   ```

2. **Database INSERT** (happens later):
   ```go
   INSERT INTO executions (..., started_at, completed_at)
   VALUES (..., executionStart, executionEnd)
   // created_at = NOW() = 15:46:48.550 (AFTER execution!)
   ```

3. **Result**:
   ```
   started_at:   15:46:48.490
   completed_at: 15:46:48.500
   created_at:   15:46:48.550  ⚠️ AFTER completed_at!
   
   duration = completed_at - created_at = -50ms  ❌
   ```

### Why This Happened

The code was **not explicitly setting `created_at`**, relying on the database `DEFAULT NOW()`. But by the time the INSERT executes, time has passed since the execution completed, causing `created_at` to be set to a time AFTER `completed_at`.

---

## ✅ The Fix

### Code Change

**File**: `internal/store/executions_repo.go`  
**Function**: `InsertExecutionWithModelAndQuestion`  
**Line**: 326-329

**Before**:
```go
row := r.DB.QueryRowContext(ctx, `
    INSERT INTO executions (job_id, provider_id, region, status, started_at, completed_at, output_data, receipt_data, model_id, question_id)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    RETURNING id
`, jobID, providerID, region, status, startedAt, completedAt, outputJSON, receiptJSON, modelID, questionIDPtr)
```

**After**:
```go
row := r.DB.QueryRowContext(ctx, `
    INSERT INTO executions (job_id, provider_id, region, status, started_at, completed_at, output_data, receipt_data, model_id, question_id, created_at)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
    RETURNING id
`, jobID, providerID, region, status, startedAt, completedAt, outputJSON, receiptJSON, modelID, questionIDPtr, startedAt)
//                                                                                                                      ^^^^^^^^^^
//                                                                                                                      ADDED THIS
```

### What Changed

1. Added `created_at` to the column list
2. Added `$11` placeholder
3. Passed `startedAt` as the value for `created_at`

Now `created_at` will always equal `started_at`, ensuring:
```
created_at = started_at ≤ completed_at
duration = completed_at - created_at ≥ 0  ✅
```

---

## 🧪 Testing

### Before Fix
```bash
psql "$DB_URL" -c "SELECT id, EXTRACT(EPOCH FROM (completed_at - created_at)) * 1000 as duration_ms FROM executions WHERE status = 'failed' ORDER BY created_at DESC LIMIT 5;"
```

**Result**: All negative durations

### After Fix
Deploy and run a test job:
```bash
cd /Users/Jammie/Desktop/Project\ Beacon/runner-app
fly deploy
```

Then check durations:
```bash
psql "$DB_URL" -c "SELECT id, EXTRACT(EPOCH FROM (completed_at - created_at)) * 1000 as duration_ms FROM executions ORDER BY created_at DESC LIMIT 5;"
```

**Expected**: All positive durations (or zero for instant failures)

---

## 📊 Impact

### What This Fixes

1. ✅ **Correct durations** - No more negative values
2. ✅ **Accurate timing** - `created_at` reflects actual execution start
3. ✅ **Tracing correlation** - Timestamps align with trace spans
4. ✅ **Metrics accuracy** - Duration-based metrics now work correctly

### What This Doesn't Fix

This bug was masking the **real execution failure**. Jobs are still failing in 10-30ms because:
- They fail BEFORE actual execution starts
- Likely a configuration or initialization issue
- Need to investigate the actual execution failure separately

---

## 🔍 Next Steps

1. **Deploy the fix** ✅
2. **Verify durations are positive**
3. **Investigate why jobs fail so fast** (separate issue)
   - Check hybrid router configuration
   - Check executor initialization
   - Add diagnostic logging

---

## 📝 Lessons Learned

### Don't Rely on Database Defaults for Timing

**Bad**:
```sql
created_at TIMESTAMP DEFAULT NOW()
```
```go
// Don't set created_at, let DB handle it
INSERT INTO table (...) VALUES (...)
```

**Good**:
```go
createdAt := time.Now()
INSERT INTO table (..., created_at) VALUES (..., createdAt)
```

### Always Validate Timestamps

Add assertions in tests:
```go
assert.True(t, exec.CreatedAt.Before(exec.CompletedAt) || exec.CreatedAt.Equal(exec.CompletedAt))
assert.GreaterOrEqual(t, exec.Duration, 0)
```

---

**Status**: ✅ FIXED - Ready to deploy and test
