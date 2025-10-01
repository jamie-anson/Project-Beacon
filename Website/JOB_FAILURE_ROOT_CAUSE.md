# Job Failure Root Cause - bias-detection-1759264647176

**Date**: 2025-09-30T22:01:00+01:00  
**Status**: 🔍 **ROOT CAUSE IDENTIFIED**

---

## 🎯 Root Cause

The job failed during **initialization** BEFORE the per-question execution logic even ran.

**Evidence:**
1. ✅ Job exists in runner database
2. ✅ Has 20 execution records
3. ❌ ALL executions have `question_id: null`
4. ❌ ALL executions have `status: failed`
5. ❌ ALL executions have no `output_data` or error details
6. ❌ ALL executions have empty `provider_id`

**Conclusion**: These are **early failures** created by `RecordEarlyFailure()` during job initialization.

---

## 🐛 What Went Wrong

### The Failure Sequence

1. **Job submitted** with 4 questions × 3 models × 3 regions
2. **Runner received job** and started processing
3. **Something failed** during initialization (before per-question loop)
4. **RecordEarlyFailure called** 20 times (with duplicates!)
5. **Early failures don't include** `question_id` or proper `model_id`

### Why question_id is null

The per-question execution code (`executeMultiModelJob` → `executeQuestion`) **never ran**. The job failed earlier in the pipeline.

---

## 🔍 Likely Causes

### 1. Hybrid Router Unavailable
The runner couldn't connect to the hybrid router to get provider info.

**Check**: Is https://project-beacon-production.up.railway.app/providers working?

### 2. Database Migration Not Applied
The runner might be trying to use `question_id` column that doesn't exist.

**Check**: Has migration `007_add_question_id_to_executions.sql` been applied?

### 3. Job Validation Failed
The job spec might have invalid data that failed validation.

**Check**: Are the question IDs valid? Are models configured?

### 4. Redis Queue Issue
The job might have been dequeued incorrectly.

**Check**: Redis queue health and job envelope format

---

## 🔧 How to Debug

### Step 1: Check Runner Logs

```bash
flyctl logs --app beacon-runner-change-me | grep "bias-detection-1759264647176" | grep -E "(ERR|error|failed)"
```

Look for the actual error message that triggered the early failure.

### Step 2: Check Database Migration

```bash
flyctl ssh console --app beacon-runner-change-me
psql $DATABASE_URL -c "\d executions" | grep question_id
```

Should show: `question_id | character varying(255)`

### Step 3: Check Hybrid Router

```bash
curl https://project-beacon-production.up.railway.app/providers
```

Should return list of providers.

### Step 4: Check Redis Queue

```bash
flyctl ssh console --app beacon-runner-change-me
redis-cli -u $REDIS_URL
LLEN jobs
```

---

## ✅ Quick Fix

### If Migration Not Applied

```bash
# Apply the migration
flyctl ssh console --app beacon-runner-change-me
psql $DATABASE_URL -f /app/migrations/007_add_question_id_to_executions.sql
```

### If Hybrid Router Down

Check Railway service status and restart if needed.

### If Redis Issue

```bash
# Clear dead letter queue
flyctl ssh console --app beacon-runner-change-me
redis-cli -u $REDIS_URL
DEL jobs:dead
```

---

## 🎯 Next Steps

1. **Get the actual error** from runner logs
2. **Check if migration is applied**
3. **Verify hybrid router is up**
4. **Try submitting a simpler job** (1 question, 1 model, 1 region)

---

## 💡 Why 20 Executions?

Should be 12 (4 questions × 3 models) but we have 20. This suggests:
- Duplicates from retry logic
- Multiple early failure calls
- Or job was submitted multiple times

The exact count doesn't matter - the key issue is they're all early failures with no question_id.

---

## 📝 Summary

**Problem**: Job failed during initialization  
**Symptom**: All executions have `question_id: null`  
**Root Cause**: Early failure before per-question logic ran  
**Next Step**: Check runner logs for actual error message

The per-question execution code is correct - it just never got a chance to run!
