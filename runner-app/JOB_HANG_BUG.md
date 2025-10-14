# Critical Bug: Jobs Hanging Indefinitely

## Problem

Jobs are stuck in "queued" status indefinitely even though all executions have completed or failed.

## Example

**Job:** `bias-detection-1760445014840`
- Status: "queued" (should be "completed" or "failed")
- Executions: 11 total (should be 18)
  - US: 6 executions (4 completed, 2 failed)
  - EU: 5 executions (5 completed)
- Missing: 7 executions never created

## Root Cause

1. **Removed overall job timeout** (to prevent queue wait timeouts)
2. **Sequential region execution** waits for all regions to finish
3. **Job runner never completes** if not all executions are created
4. **No timeout to detect hung jobs**

## Why It Happens

```go
// In cross_region_executor.go
execCtx := ctx  // No timeout!

for _, plan := range plans {
    regionResult := cre.executeRegion(execCtx, jobSpec, plan)
    // If executeRegion hangs or doesn't create all executions,
    // the loop never completes
}

// This code is NEVER reached if loop hangs
UpdateJobStatus(ctx, jobID, "completed")
```

## Immediate Fix Options

### Option 1: Add Per-Job Timeout (Recommended)
```go
// Add reasonable timeout for entire job
jobTimeout := time.Duration(len(plans)) * 15 * time.Minute  // 15 min per region
jobCtx, cancel := context.WithTimeout(ctx, jobTimeout)
defer cancel()

for _, plan := range plans {
    regionResult := cre.executeRegion(jobCtx, jobSpec, plan)
}
```

### Option 2: Detect Completion
```go
// After all regions, check if we're done
if allExecutionsComplete() {
    UpdateJobStatus(ctx, jobID, calculateStatus())
    return
}
```

### Option 3: Background Cleanup Job
```go
// Periodic job to find hung jobs
func cleanupHungJobs() {
    jobs := findJobsInStatus("queued", olderThan(30*time.Minute))
    for _, job := range jobs {
        if allExecutionsComplete(job) {
            UpdateJobStatus(job.ID, calculateFinalStatus(job))
        }
    }
}
```

## Recommended Solution

**Combination of Option 1 + Option 3:**

1. Add per-job timeout (30 minutes max)
2. Add cleanup job to catch edge cases
3. Log warnings when approaching timeout

## Impact

**Current State:**
- Jobs hang forever in "queued" status
- Portal shows negative time remaining
- Users confused about job status
- Database fills with incomplete jobs

**After Fix:**
- Jobs complete or timeout within 30 minutes
- Clear status updates
- Cleanup catches edge cases
- Better user experience

## Files to Modify

1. `internal/execution/cross_region_executor.go` - Add job timeout
2. `internal/worker/job_runner.go` - Add completion detection
3. `cmd/runner/main.go` - Add cleanup worker (optional)

## Testing

1. Submit job with 2 regions
2. Verify job completes within expected time
3. Test timeout scenario (kill Modal during execution)
4. Verify cleanup job fixes hung jobs
