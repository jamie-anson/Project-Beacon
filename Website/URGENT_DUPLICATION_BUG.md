# URGENT: Critical Duplication Bug in Per-Question Execution

**Date**: 2025-09-29T23:40:43+01:00  
**Severity**: 🔴 **CRITICAL**  
**Status**: ❌ **PRODUCTION BROKEN**

---

## 🚨 Critical Issue

**Job**: bias-detection-1759185378516  
**Created**: 2025-09-29T22:36:19Z (AFTER deployment)  
**Expected Executions**: 72 (8 questions × 3 models × 3 regions)  
**Actual Executions**: 100+ with massive duplication

---

## 📊 Duplication Analysis

### Expected vs Actual:

| Model-Region | Expected | Actual | Extra |
|--------------|----------|--------|-------|
| eu-west \| qwen2.5-1.5b | 8 | **14** | +6 |
| asia-pacific \| qwen2.5-1.5b | 8 | **14** | +6 |
| us-east \| qwen2.5-1.5b | 8 | **13** | +5 |
| us-east \| mistral-7b | 8 | **13** | +5 |
| us-east \| llama3.2-1b | 8 | **13** | +5 |
| eu-west \| mistral-7b | 8 | **13** | +5 |
| asia-pacific \| mistral-7b | 8 | **9** | +1 |
| eu-west \| llama3.2-1b | 8 | **6** | -2 |
| asia-pacific \| llama3.2-1b | 8 | **5** | -3 |

**Total**: 100 executions instead of 72 (**+28 duplicates, -5 missing**)

---

## 🔍 Root Cause

### The Bug is in the New Code:

**File**: `internal/worker/job_runner.go`  
**Function**: `executeMultiModelJob()`  
**Lines**: 259-322

**Problem**: The triple nested loop with goroutines is creating race conditions:

```go
for _, model := range spec.Models {
    for _, region := range model.Regions {
        for questionIdx, questionID := range questions {
            wg.Add(1)
            sem <- struct{}{} // Acquire semaphore
            go func(m models.ModelSpec, r string, qID string, qIdx int) {
                // Execute...
            }(model, region, questionID, questionIdx)
        }
    }
}
```

**Issue**: Multiple goroutines are being spawned simultaneously, and there's likely:
1. Job envelope being processed multiple times from Redis
2. Race condition in the goroutine spawning
3. No deduplication check before creating executions

---

## 🎯 Immediate Actions Required

### 1. ROLLBACK DEPLOYMENT (URGENT)

**Revert to previous commit before per-question execution:**

```bash
cd /Users/Jammie/Desktop/Project\ Beacon/runner-app
git revert b9e22ac 1b3236b --no-commit
git commit -m "URGENT: Rollback per-question execution due to critical duplication bug"
git push origin main
```

Then redeploy to Fly.io:
```bash
flyctl deploy -a beacon-runner-production
```

---

### 2. FIX THE BUG

**Root Cause**: The issue is likely that the job is being dequeued multiple times from Redis, or there's no idempotency check.

**Solution Options**:

#### Option A: Add Execution Deduplication (Recommended)

Add a check before inserting executions:

```go
// In executeSingleRegion() before InsertExecutionWithModel()
questionID := ""
if spec.Metadata != nil {
    if qid, ok := spec.Metadata["question_id"].(string); ok {
        questionID = qid
    }
}

// Check if execution already exists
exists, err := w.ExecRepo.ExecutionExists(ctx, spec.ID, actualRegion, modelID, questionID)
if err != nil {
    l.Error().Err(err).Msg("failed to check execution existence")
} else if exists {
    l.Warn().
        Str("job_id", spec.ID).
        Str("region", actualRegion).
        Str("model_id", modelID).
        Str("question_id", questionID).
        Msg("execution already exists, skipping duplicate")
    return result // Skip duplicate
}
```

Add the database method:

```go
// In internal/store/executions.go
func (r *ExecutionsRepo) ExecutionExists(ctx context.Context, jobID, region, modelID, questionID string) (bool, error) {
    query := `
        SELECT EXISTS(
            SELECT 1 FROM executions 
            WHERE job_id = $1 
            AND region = $2 
            AND model_id = $3 
            AND (
                ($4 = '' AND question_id IS NULL) OR 
                question_id = $4
            )
        )`
    
    var exists bool
    err := r.db.QueryRowContext(ctx, query, jobID, region, modelID, questionID).Scan(&exists)
    return exists, err
}
```

#### Option B: Add Job Processing Lock

Prevent job from being processed multiple times:

```go
// In handleEnvelope()
func (w *JobRunner) handleEnvelope(ctx context.Context, envelope *queue.JobEnvelope) error {
    // Try to acquire processing lock
    lockKey := fmt.Sprintf("job:processing:%s", envelope.ID)
    acquired, err := w.Redis.SetNX(ctx, lockKey, "1", 10*time.Minute).Result()
    if err != nil {
        return fmt.Errorf("failed to acquire lock: %w", err)
    }
    if !acquired {
        l.Warn().Str("job_id", envelope.ID).Msg("job already being processed, skipping")
        return nil
    }
    defer w.Redis.Del(ctx, lockKey)
    
    // Process job...
}
```

---

### 3. ADD DATABASE CONSTRAINT

Prevent duplicates at database level:

```sql
-- Add unique constraint (after adding question_id column if not exists)
ALTER TABLE executions 
ADD COLUMN IF NOT EXISTS question_id VARCHAR(255);

CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_execution 
ON executions(job_id, region, model_id, COALESCE(question_id, ''));
```

---

## 📋 Recovery Steps

### Immediate (Next 30 minutes):

1. ✅ Document the bug
2. ⏳ Rollback deployment to stable version
3. ⏳ Notify users (if any active jobs)
4. ⏳ Clean up duplicate executions in database

### Short-term (Next 2 hours):

1. ⏳ Implement deduplication fix
2. ⏳ Add database constraint
3. ⏳ Test fix locally with multi-question job
4. ⏳ Deploy fix with monitoring

### Long-term (Next 24 hours):

1. ⏳ Add comprehensive integration tests
2. ⏳ Add execution count validation
3. ⏳ Add alerting for duplicate detection
4. ⏳ Review all goroutine usage for race conditions

---

## 🔬 Why This Happened

### Deployment Timeline:

1. **22:10**: Committed per-question execution code (1b3236b)
2. **22:26**: Fixed missing dependencies (b9e22ac)
3. **22:31**: Deployment completed to Fly.io
4. **22:36**: First job submitted with new code
5. **22:38**: Massive duplication started

### Testing Gap:

- ✅ Code compiled successfully
- ✅ Dependencies resolved
- ❌ **NO INTEGRATION TESTING** with real job submission
- ❌ **NO VALIDATION** of execution count
- ❌ **NO DEDUPLICATION** logic

---

## 🎯 Lessons Learned

1. **Always test with real jobs before deploying**
2. **Add deduplication at multiple levels** (application + database)
3. **Validate execution counts** match expectations
4. **Add integration tests** for multi-model, multi-question jobs
5. **Monitor for anomalies** immediately after deployment

---

## 📊 Impact Assessment

**Affected Jobs**: 1 (bias-detection-1759185378516)  
**Duplicate Executions**: ~28  
**Cost Impact**: Minimal (failed executions)  
**Data Integrity**: Compromised (duplicate records)  
**User Impact**: High (incorrect results, confusion)

---

## ✅ Action Items

- [ ] Rollback deployment immediately
- [ ] Implement deduplication fix
- [ ] Add database constraint
- [ ] Test fix locally
- [ ] Deploy with monitoring
- [ ] Clean up duplicate data
- [ ] Add integration tests
- [ ] Document incident

---

**Status**: 🔴 **CRITICAL - IMMEDIATE ROLLBACK REQUIRED**  
**Priority**: P0  
**ETA for Fix**: 2-4 hours
