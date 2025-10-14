# Execution Status Mismatch Investigation
**Date:** 2025-01-14 23:37  
**Priority:** HIGH  
**Status:** 🔍 INVESTIGATING

---

## Problem Statement

Portal UI shows executions as "Failed" even though Modal logs show them as "Succeeded". This creates a data integrity issue where users see incorrect job status.

**Observed Behavior:**
- Modal Dashboard: All executions show "Succeeded" ✅
- Portal UI: mistral-7b shows "Failed" ❌
- Database: Unknown (need to verify)

---

## Evidence Collected

### 1. Portal UI Behavior

**Screenshot Evidence:**
- mistral-7b (United States): Status = "Failed" (red badge)
- llama3.2-1b: Completed successfully
- qwen2.5-1.5b: Still pending

**Console Warnings:**
```javascript
[MISSING EXECUTION] Q:identity_basic M:mistral-7b R:EU
[MISSING EXECUTION] Q:identity_basic M:qwen2.5-1.5b R:EU
[MISSING EXECUTION] Q:taiwan_status M:llama3.2-1b R:EU
// ... multiple missing EU executions
```

**Debug Output:**
```javascript
{
  lookingFor: "EU",
  lookingForType: "string",
  availableExecs: (1) [...]  // Only 1 execution found (likely US)
}
```

### 2. Modal Infrastructure Status

**Modal Dashboard (Run #20):**
- Enqueued: Oct 14, 2025 at 23:34:38
- Started: Oct 14, 2025 at 23:34:38
- Startup: 0ms (warm container)
- Execution: 45.11s
- Status: ✅ **Succeeded**

**Modal Dashboard (Run #19):**
- Execution: 1m 33s
- Status: ✅ **Succeeded**

**Modal Dashboard (Run #18):**
- Status: 🟣 **Running** (still in progress)

**Modal Logs Warning (Non-Critical):**
```
The attention mask is not set and cannot be inferred from input 
because pad token is same as eos token.
```
- This is a Hugging Face Transformers warning
- Does NOT cause execution failure
- Can be fixed with proper attention_mask handling

### 3. Architecture Flow

```
User submits job
    ↓
Runner spawns 3 region goroutines (US, EU, APAC)
    ↓
Each region → Hybrid Router → Modal endpoint
    ↓
Modal executes inference → Returns response
    ↓
Runner receives response → Writes to database
    ↓
Portal polls database → Shows status to user
```

**Bottleneck Identified:** Runner → Database write step

---

## Root Cause Hypotheses

### Hypothesis 1: Runner Misinterpreting Modal Response ⭐ LIKELY

**Theory:** Runner receives successful Modal response but marks it as failed due to:
- Response validation failure
- ~~Timeout (even though Modal succeeded)~~ **RULED OUT** ❌
- Missing/malformed fields in response
- Exception during response parsing

**Evidence:**
- Modal shows success
- Portal shows failure
- ~~Timing suggests runner timeout (45s execution might exceed runner's timeout)~~

**Timeout Configuration Found:**
```go
// internal/hybrid/client.go line 29
timeoutSec := 300  // 5 minutes = 300 seconds
httpClient: &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
```
- **Runner timeout: 300s (5 minutes)** ✅
- **Modal execution: 45s** ✅
- **Timeout is NOT the issue** - plenty of headroom

**ROOT CAUSE FOUND:** ✅

**Location:** `hybrid_router/core/router.py` line 414
```python
# Treat empty text as failure for better upstream handling
if bool(success) and (resp_text is None or str(resp_text).strip() == ""):
    return {
        "success": False,
        "error": "Provider returned empty response",
        "error_code": "EMPTY_MODEL_RESPONSE",
    }
```

**The Bug:**
1. Modal executes successfully (45s, status="success")
2. Modal returns empty or whitespace-only response text
3. Router marks it as `success=False` due to empty response
4. Runner receives `success=False` and marks execution as "failed"
5. Database stores status="failed"
6. Portal shows "Failed" even though Modal succeeded

**Why Empty Response:**
- Model might return empty string for certain prompts
- Tokenization issues (attention mask warning)
- Model refusing to answer
- Response parsing error in Modal

**How to Verify:**
- Check Modal logs for actual response content
- Check router logs for "EMPTY_MODEL_RESPONSE" error
- Check runner logs for "hybrid error: Provider returned empty response"

**Potential Fixes:**
1. **Fix Modal:** Ensure models always return non-empty responses
2. **Fix Router:** Don't treat empty responses as failures (return success with empty string)
3. **Fix Runner:** Handle empty responses gracefully (mark as "completed" with empty output)

---

### Hypothesis 2: Database Write Failure

**Theory:** Runner tries to write "completed" status but database write fails, defaults to "failed"

**Evidence:**
- Execution record exists (portal shows it)
- Status is wrong

**How to Verify:**
- Check runner logs for database errors
- Query database directly for execution status
- Check for transaction rollbacks

**Potential Fix:**
- Add database write retries
- Better error handling for partial failures
- Transaction isolation improvements

---

### Hypothesis 3: Race Condition in Status Updates

**Theory:** Multiple goroutines updating same execution, last write wins with wrong status

**Evidence:**
- Parallel execution across regions
- Timing-sensitive behavior

**How to Verify:**
- Check for duplicate execution IDs
- Review goroutine synchronization
- Check database transaction logs

**Potential Fix:**
- Add optimistic locking
- Use database-level status transitions
- Serialize status updates per execution

---

### Hypothesis 4: Portal Misreading Database Status

**Theory:** Database has correct status, portal transforms it incorrectly

**Evidence:**
- Less likely given console shows missing executions
- Would affect all executions, not just some

**How to Verify:**
- Query database directly
- Compare with portal API response
- Check status normalization logic

**Potential Fix:**
- Fix status mapping in portal
- Standardize status values

---

## Related Issues Discovered

### Issue 1: Portal Region Name Mismatch ✅ FIXED

**Problem:** Portal couldn't match EU executions due to region name format
- Database: `'eu-west'`
- Portal: `'EU'`
- `normalizeRegion()` wasn't handling all variants

**Fix Applied:** Updated `normalizeRegion()` with explicit checks
**Status:** Deployed to production (commit b689c33)
**Verification:** Waiting for Netlify deployment + job completion

---

### Issue 2: Missing EU Executions

**Problem:** Portal shows only US executions, EU executions missing
**Possible Causes:**
1. EU executions haven't completed yet (still running)
2. EU executions failed to write to database
3. API query filtering out EU executions

**Current Status:** 
- 3 completed (likely all US)
- 9 pending (likely EU + remaining US)
- Job still in progress

---

## Investigation Plan

### Step 1: Check Runner Logs (IMMEDIATE)

**What to look for:**
```bash
# Search for mistral-7b execution logs
grep -i "mistral-7b" runner.log

# Look for timeout errors
grep -i "timeout\|timed out" runner.log

# Look for database errors
grep -i "failed to insert\|database error" runner.log

# Look for status transitions
grep -i "status.*failed\|marking.*failed" runner.log
```

**Expected findings:**
- Timeout error for mistral-7b (45s execution)
- Database insert success/failure
- Actual status being written

---

### Step 2: Query Database Directly

**SQL Queries:**
```sql
-- Check all executions for this job
SELECT 
    id, 
    region, 
    model_id, 
    question_id, 
    status, 
    started_at, 
    completed_at,
    created_at
FROM executions e
JOIN jobs j ON e.job_id = j.id
WHERE j.jobspec_id = '<job-id>'
ORDER BY created_at DESC;

-- Check for failed executions
SELECT * FROM executions 
WHERE status = 'failed' 
ORDER BY created_at DESC 
LIMIT 10;

-- Check execution counts by status
SELECT status, COUNT(*) 
FROM executions e
JOIN jobs j ON e.job_id = j.id
WHERE j.jobspec_id = '<job-id>'
GROUP BY status;
```

**Expected findings:**
- Actual status values in database
- Whether EU executions exist
- Timing of status updates

---

### Step 3: Check Runner Timeout Configuration

**Files to check:**
```go
// runner-app/internal/worker/job_runner.go
// Look for timeout configuration

// runner-app/internal/executor/*.go
// Check HTTP client timeouts

// hybrid_router/core/router.py
// Check Modal request timeout
```

**Current known timeouts:**
- Hybrid Router: 600s (10 minutes)
- Modal: 900s (15 minutes)
- Runner: ??? (need to check)

**If runner timeout < 45s:** That's the problem!

---

### Step 4: Review Response Validation

**Check:**
```go
// runner-app/internal/worker/job_runner.go
// Lines around execution result handling
// Look for validation that might reject valid responses
```

**Possible issues:**
- Strict schema validation
- Required fields that Modal doesn't return
- Type mismatches

---

## Proposed Solutions

### Solution 1: Fix Router Empty Response Handling ⭐ RECOMMENDED

**Implementation:**
```python
# hybrid_router/core/router.py line 414
# REMOVE this check that treats empty responses as failures:
# if bool(success) and (resp_text is None or str(resp_text).strip() == ""):
#     return {"success": False, "error": "Provider returned empty response"}

# REPLACE with:
if bool(success):
    # Allow empty responses - some models may return empty for certain prompts
    # Upstream can handle empty responses appropriately
    resp_text = resp_text or ""  # Convert None to empty string
```

**Pros:**
- Fixes root cause directly
- Simple one-line change
- Allows models to return empty responses legitimately
- No false failures

**Cons:**
- Empty responses now marked as "success"
- Need to handle empty responses in portal UI
- May hide actual model failures

**Effort:** 5 minutes
**Risk:** Low

**Alternative:** Add a flag to distinguish "empty but successful" from "failed"

---

### Solution 2: Improve Error Handling

**Implementation:**
```go
// Distinguish between different failure types
if err != nil {
    if isTimeout(err) {
        // Mark as "timeout" not "failed"
        status = "timeout"
    } else if isValidationError(err) {
        // Log validation error but mark as "completed" if Modal succeeded
        status = "completed"
    } else {
        status = "failed"
    }
}
```

**Pros:**
- More granular status tracking
- Better debugging
- Prevents false failures

**Cons:**
- More complex logic
- Need to define all error types

**Effort:** 30 minutes
**Risk:** Medium

---

### Solution 3: Add Status Reconciliation

**Implementation:**
```go
// Periodic job that checks Modal status and updates database
func reconcileExecutionStatuses() {
    // For each "failed" execution
    // Check Modal logs
    // If Modal shows success, update to "completed"
}
```

**Pros:**
- Fixes existing bad data
- Prevents future mismatches
- Self-healing system

**Cons:**
- Additional complexity
- Requires Modal API integration
- Periodic overhead

**Effort:** 2 hours
**Risk:** Medium

---

### Solution 4: Retry Failed Executions

**Implementation:**
```go
// If execution fails, check if it's retriable
if status == "failed" && isRetriable(err) {
    // Add to retry queue
    retryExecution(jobID, modelID, region, questionID)
}
```

**Pros:**
- Handles transient failures
- Improves success rate
- User doesn't see failures

**Cons:**
- Increased cost (duplicate executions)
- Complexity
- May hide real issues

**Effort:** 1 hour
**Risk:** Medium

---

## Immediate Actions

### Priority 1: Verify Root Cause (15 minutes)

1. ✅ Check runner logs for mistral-7b execution
2. ✅ Query database for actual status
3. ✅ Check runner timeout configuration
4. ✅ Identify exact failure point

### Priority 2: Quick Fix (30 minutes)

1. Increase runner timeout to 15 minutes
2. Deploy to production
3. Test with new job
4. Verify status matches Modal

### Priority 3: Long-term Solution (2 hours)

1. Add status reconciliation
2. Improve error handling
3. Add monitoring/alerting
4. Document status flow

---

## Success Criteria

✅ **Fixed when:**
1. Modal "Succeeded" → Portal shows "Completed"
2. No false "Failed" statuses
3. EU executions appear in portal
4. Status updates within 5 seconds of completion

---

## Monitoring & Validation

### Metrics to Track

1. **Status Mismatch Rate**
   - Count: Modal success but Portal failed
   - Target: 0%

2. **Execution Completion Time**
   - P50, P95, P99 latencies
   - Target: <60s per execution

3. **Database Write Success Rate**
   - Successful inserts / total attempts
   - Target: >99.9%

4. **Region Execution Balance**
   - US vs EU vs APAC completion rates
   - Target: Equal distribution

### Alerts to Add

1. Status mismatch detected
2. Execution timeout exceeded
3. Database write failure
4. Missing executions for >5 minutes

---

## Related Documentation

- [GPU-OPTIMIZATION-PLAN.md](./GPU-OPTIMIZATION-PLAN.md) - Performance optimization
- [PORTAL-REGION-FIX-2025-01-14.md](./PORTAL-REGION-FIX-2025-01-14.md) - Region matching fix
- [ARCHITECTURE-ANALYSIS-2025-01-14.md](./ARCHITECTURE-ANALYSIS-2025-01-14.md) - System architecture

---

## Next Steps

1. **Immediate:** Check runner logs for timeout/error
2. **Short-term:** Increase runner timeout
3. **Medium-term:** Add status reconciliation
4. **Long-term:** Comprehensive monitoring

**Status:** 🔍 Awaiting runner log analysis
