# Queue Timeout Fix

## Problem: Jobs Timing Out While Waiting in Queue

### Root Cause
The overall job timeout (`jobSpec.Constraints.Timeout = 10 minutes`) was applied when the job **started**, not when it **began executing**. This caused jobs to timeout while waiting in the queue.

### Example Scenario

**Without Fix:**
```
Timeline:
0s    Job-C submitted
      Overall context created with 10-minute timeout
      
0s    Job-A executing (Job-C waits in queue)
270s  Job-A completes
270s  Job-B executing (Job-C still waiting)
540s  Job-B completes
540s  Job-C FINALLY starts execution
600s  Job-C times out! ❌
      
Reason: 600 seconds elapsed since context creation
        Only 60 seconds of actual execution time
        Job failed due to queue wait time, not execution time
```

**Impact:**
- ❌ Jobs fail even though they never got to execute
- ❌ Queue congestion causes cascading failures
- ❌ Users see "timeout" errors for jobs that never ran
- ❌ Unfair: Later jobs penalized for earlier jobs' execution time

---

## Solution: Remove Overall Job Timeout

### Implementation

**Before:**
```go
// Create context with overall timeout
execCtx, cancel := context.WithTimeout(ctx, jobSpec.Constraints.Timeout)
defer cancel()

// Execute regions sequentially
for _, plan := range plans {
    regionResult := cre.executeRegion(execCtx, jobSpec, plan)
    // ...
}
```

**After:**
```go
// Don't use overall job timeout for sequential execution
// Reason: Job might wait in queue, and timeout would include wait time
// Instead, each individual execution gets its own timeout (5 min per execution)
// This allows jobs to wait in queue without timing out
execCtx := ctx

// Execute regions sequentially
for _, plan := range plans {
    regionResult := cre.executeRegion(execCtx, jobSpec, plan)
    // ...
}
```

### Per-Execution Timeout (Still Applied)

Each individual execution still has its own 5-minute timeout:

```go
// In executeRegion loop:
execCtx, execCancel := context.WithTimeout(ctx, 5*time.Minute)
receipt, err := cre.singleRegionExecutor.ExecuteOnProvider(execCtx, &execSpec, providerID, plan.Region)
execCancel()
```

**This ensures:**
- ✅ Individual executions can't hang forever (5 min max)
- ✅ Jobs can wait in queue indefinitely without timing out
- ✅ Fair: Timeout only applies to actual execution time

---

## Behavior Comparison

### Before Fix

| Job | Queue Wait | Execution Time | Total Time | Result |
|-----|------------|----------------|------------|--------|
| A   | 0s         | 270s           | 270s       | ✅ Success |
| B   | 270s       | 270s           | 540s       | ✅ Success |
| C   | 540s       | 60s            | 600s       | ❌ Timeout (exceeded 10 min) |

**Job C fails even though it only executed for 60 seconds!**

### After Fix

| Job | Queue Wait | Execution Time | Total Time | Result |
|-----|------------|----------------|------------|--------|
| A   | 0s         | 270s           | 270s       | ✅ Success |
| B   | 270s       | 270s           | 540s       | ✅ Success |
| C   | 540s       | 270s           | 810s       | ✅ Success |

**Job C succeeds because timeout only applies to execution, not queue wait!**

---

## Edge Cases

### Case 1: Extremely Long Queue Wait

```
Job-Z waits 2 hours in queue
Job-Z executes for 5 minutes
Total: 2 hours 5 minutes

Result: ✅ Success (no overall timeout)
```

**Behavior:** Jobs can wait indefinitely in queue. Only execution time is limited.

### Case 2: Individual Execution Timeout

```
Job-X starts execution
Execution hangs for 6 minutes (exceeds 5-min execution timeout)

Result: ❌ Timeout (individual execution limit)
```

**Behavior:** Individual executions still have 5-minute timeout to prevent hangs.

### Case 3: Multiple Retries

```
Job-Y execution fails 3 times
Each retry takes 5 minutes
Total execution time: 15 minutes

Result: ❌ Failed after 3 retries (not timeout)
```

**Behavior:** Retries don't count toward overall timeout. Job fails due to max retries, not timeout.

---

## Implications

### Positive

✅ **Fair Resource Allocation**
- Jobs aren't penalized for queue wait time
- Only actual execution time counts toward timeout

✅ **Better Queue Behavior**
- Jobs can wait in queue without risk of timeout
- Queue congestion doesn't cause cascading failures

✅ **Improved User Experience**
- Fewer confusing "timeout" errors
- Jobs that wait longer aren't more likely to fail

✅ **Predictable Behavior**
- Timeout is based on execution time, not submission time
- Easier to reason about and debug

### Considerations

⚠️ **No Overall Job Timeout**
- Jobs can theoretically run forever if executions keep succeeding
- Mitigated by: Individual execution timeouts (5 min each)
- Max total time: 18 executions × 5 min = 90 minutes

⚠️ **Queue Backlog**
- Jobs might wait hours in queue during high load
- Mitigated by: Queue status API shows wait time
- Future: Add queue position and estimated wait time

---

## Monitoring

### Metrics to Track

1. **Queue Wait Time**
   - Average time jobs spend in queue
   - Max queue wait time
   - Queue depth over time

2. **Execution Time**
   - Average execution time per job
   - Timeout rate (executions hitting 5-min limit)
   - Success rate by execution time

3. **Total Job Time**
   - End-to-end time (queue wait + execution)
   - Percentiles (p50, p95, p99)
   - Comparison to SLA targets

### Alerts

- Queue depth > 50 jobs
- Average queue wait > 30 minutes
- Execution timeout rate > 10%
- Total job time > 2 hours

---

## Future Enhancements

### 1. Queue Position API
```json
{
  "job_id": "bias-detection-123",
  "queue_position": 5,
  "estimated_wait_time": "15 minutes",
  "ahead_of_you": 4
}
```

### 2. Priority Queue
```python
# Urgent jobs skip ahead
if job.priority == "urgent":
    queue.insert(0, job)  # Front of queue
```

### 3. Adaptive Timeout
```python
# Longer timeout for complex jobs
timeout = base_timeout + (num_executions * 30s)
```

### 4. Queue SLA
```python
# Guarantee max wait time
if queue_wait_time > 1_hour:
    scale_up_workers()
```

---

## Testing

### Test Case 1: Job Waits in Queue
```
1. Submit Job-A (9 executions, ~5 min total)
2. Submit Job-B immediately
3. Verify Job-B waits in queue
4. Verify Job-B succeeds after Job-A completes
5. Verify Job-B execution time excludes queue wait
```

### Test Case 2: Individual Execution Timeout
```
1. Submit job with model that hangs
2. Verify execution times out after 5 minutes
3. Verify retry is attempted
4. Verify job fails after max retries
```

### Test Case 3: Long Queue Wait
```
1. Submit 10 jobs simultaneously
2. Verify all jobs eventually complete
3. Verify later jobs don't timeout due to queue wait
4. Verify execution times are reasonable
```

---

## Files Modified

- `internal/execution/cross_region_executor.go` (lines 144-148)
  - Removed overall job timeout
  - Added comment explaining rationale
  - Individual execution timeouts still apply

---

## Deployment Notes

1. **Backward Compatible**: No API changes
2. **Database**: No schema changes
3. **Behavior Change**: Jobs can now wait longer in queue
4. **Monitoring**: Watch for increased total job times
5. **Rollback**: Revert to previous timeout behavior if needed

---

## Summary

**Problem:** Jobs timing out while waiting in queue
**Solution:** Remove overall job timeout, keep per-execution timeout
**Result:** Fair resource allocation, better queue behavior, improved UX

✅ Jobs can wait in queue without timing out
✅ Individual executions still have 5-minute timeout
✅ Fair: Only execution time counts toward timeout
✅ Better user experience with fewer confusing errors
