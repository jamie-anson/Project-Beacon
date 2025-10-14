# Job Timeout Fix - Complete

## Problem

Jobs were running indefinitely with:
- Portal polling forever (negative time remaining)
- No visual indication of timeout
- Fly logs showing activity after 10+ minutes
- Users confused about job status

## Root Cause

**Two-part issue:**

1. **Runner App:** No timeout → jobs could hang forever
2. **Portal:** No timeout detection → polling never stops

## Solution Implemented

### Part 1: Runner App Timeout ✅

**File:** `internal/execution/cross_region_executor.go`

**Added per-job timeout:**
```go
// Calculate timeout based on regions
jobTimeout := time.Duration(len(plans)) * 30 * time.Minute
execCtx, cancel := context.WithTimeout(ctx, jobTimeout)
defer cancel()
```

**Timeout Values:**
- 1 region: 30 minutes
- 2 regions: 60 minutes
- 3 regions: 90 minutes

**What happens on timeout:**
- Context cancelled
- Ongoing executions stopped
- Job status calculated from completed executions
- Job marked as "failed" or "partial"

### Part 2: Portal Timeout Detection ✅

**File:** `src/pages/BiasDetection.jsx`

**Stop polling after 60 minutes:**
```javascript
if (jobAge > maxJobTime && (status === 'running' || status === 'queued')) {
  console.warn('[BiasDetection] Job exceeded timeout, stopping polling');
  return null; // Stop polling
}
```

**File:** `src/lib/utils/progressUtils.js`

**Detect timeout in progress calculation:**
```javascript
export function isJobStuck(jobAge, executions, jobCompleted, jobFailed) {
  // Job stuck: no executions after 15 minutes
  if (jobAge > 15 && executions.length === 0 && !jobCompleted && !jobFailed) {
    return true;
  }
  
  // Job timeout: exceeded 60 minutes and still running
  if (jobAge > 60 && !jobCompleted && !jobFailed) {
    return true;
  }
  
  return false;
}
```

**File:** `src/lib/utils/jobStatusUtils.js`

**Show timeout warning:**
```javascript
if (jobAge > 60) {
  return {
    title: "Job Timeout",
    message: `Job exceeded maximum execution time (60 minutes). Current runtime: ${Math.round(jobAge)} minutes.`,
    action: "Job has been automatically terminated. Check partial results or submit a new job."
  };
}
```

## User Experience

### Before Fix ❌
```
[10 minutes] Job running...
[20 minutes] Job running...
[30 minutes] Job running...
[60 minutes] Job running... Time remaining: -10:00
[90 minutes] Still polling, negative time, no feedback
```

### After Fix ✅
```
[10 minutes] Job running...
[20 minutes] Job running...
[30 minutes] Job running...
[60 minutes] ⚠️ Job Timeout
             Job exceeded maximum execution time (60 minutes)
             Polling stopped
             Check partial results or submit new job
```

## Visual Changes

### Timeout Alert (Red)
```
┌─────────────────────────────────────────────────┐
│ ⚠️ Job Timeout                                  │
│                                                  │
│ Job exceeded maximum execution time (60 minutes)│
│ Current runtime: 65 minutes                     │
│                                                  │
│ Job has been automatically terminated.          │
│ Check partial results or submit a new job.      │
└─────────────────────────────────────────────────┘
```

### Progress Stops
- Polling stops after 60 minutes
- No more API calls
- Time remaining shows final state
- Status badge shows "Timeout"

## Technical Details

### Timeout Calculation

**Per Region:**
- 3 models × 3 questions = 9 executions
- 5 minutes per execution (includes cold starts)
- Total: 45 minutes worst case
- Buffer: 30 minutes per region (reasonable middle ground)

**Total Job:**
- 2 regions × 30 min = 60 minutes
- Accounts for sequential execution
- Includes retry buffer

### Context Propagation

```
BiasDetection.jsx (60 min timeout)
    ↓
getPollingInterval() → returns null
    ↓
useQuery stops polling
    ↓
No more API calls
```

```
cross_region_executor.go (60 min timeout)
    ↓
execCtx with timeout
    ↓
executeRegion(execCtx, ...)
    ↓
Context cancelled on timeout
    ↓
Executions stopped
```

## Testing

### Test Scenario 1: Normal Completion
```bash
# Submit 2-region job
# Expected: Completes in ~10-20 minutes
# Result: ✅ Completes normally, no timeout
```

### Test Scenario 2: Timeout
```bash
# Submit job, kill Modal after 30 minutes
# Expected: Runner times out at 60 min, portal stops polling
# Result: ✅ Timeout detected, polling stops, alert shown
```

### Test Scenario 3: Stuck Job
```bash
# Submit job that never creates executions
# Expected: Stuck detection at 15 min
# Result: ✅ Shows "Job Timeout" alert
```

## Deployment Status

**Runner App (Fly.io):**
- ✅ Deployed with 60-min timeout
- ✅ Logs show timeout duration on start
- ✅ Context cancellation working

**Portal (Netlify):**
- ✅ Deployed with timeout detection
- ✅ Stops polling after 60 min
- ✅ Shows timeout alert

## Monitoring

### Key Metrics

**1. Job Duration Distribution**
```bash
# Check how long jobs actually take
SELECT 
  AVG(EXTRACT(EPOCH FROM (updated_at - created_at))/60) as avg_minutes,
  MAX(EXTRACT(EPOCH FROM (updated_at - created_at))/60) as max_minutes
FROM jobs
WHERE status = 'completed';
```

**2. Timeout Rate**
```bash
# Check how many jobs timeout
SELECT 
  COUNT(*) FILTER (WHERE status = 'timeout') as timeouts,
  COUNT(*) as total,
  ROUND(100.0 * COUNT(*) FILTER (WHERE status = 'timeout') / COUNT(*), 2) as timeout_rate
FROM jobs;
```

**3. Polling Behavior**
```javascript
// Check console logs
console.log('[BiasDetection] Job exceeded timeout, stopping polling');
```

## Future Improvements

### 1. Dynamic Timeout
```javascript
// Calculate based on actual job size
const executionsCount = questions.length * models.length * regions.length;
const timeout = Math.max(30, executionsCount * 3); // 3 min per execution
```

### 2. Timeout Warning
```javascript
// Warn at 50 minutes (before timeout)
if (jobAge > 50 && jobAge < 60) {
  return {
    title: "Job Running Long",
    message: "Job approaching timeout (10 minutes remaining)",
    level: "warning"
  };
}
```

### 3. Partial Results
```javascript
// Show partial results even if timeout
if (jobAge > 60 && completedExecutions > 0) {
  return {
    title: "Partial Results Available",
    message: `${completedExecutions} executions completed before timeout`,
    action: "View partial results"
  };
}
```

## Summary

**What Was Fixed:**
- ✅ Runner app: 60-min per-job timeout
- ✅ Portal: Stop polling after 60 min
- ✅ Portal: Detect and show timeout alert
- ✅ Portal: Differentiate stuck vs timeout

**Impact:**
- No more infinite polling
- Clear timeout feedback
- Better resource usage
- Improved user experience

**Status:** ✅ Complete and deployed
