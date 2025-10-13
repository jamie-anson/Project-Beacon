# Deploy Diagnostic Logging - Ready to Deploy

## Changes Made ✅

### File: `internal/handlers/cross_region_handlers.go`

#### 1. Added Panic Recovery & Goroutine Completion Logging (Line 169-174)
```go
defer func() {
    if r := recover(); r != nil {
        logger.Error().Interface("panic", r).Str("job_id", req.JobSpec.ID).Msg("PANIC in cross-region goroutine")
    }
    logger.Info().Str("job_id", req.JobSpec.ID).Msg("Cross-region goroutine completed")
}()
```

**Purpose**: Catch any panics and confirm goroutine completes

#### 2. Added Job Status Update Logging (Line 278-290)
```go
logger.Info().
    Str("job_id", req.JobSpec.ID).
    Str("old_status", "queued").
    Str("new_status", newStatus).
    Msg("Attempting to update job status")

if err := h.jobsRepo.UpdateJobStatus(execCtx, req.JobSpec.ID, newStatus); err != nil {
    logger.Error().Err(err).Str("job_id", req.JobSpec.ID).Str("status", newStatus).Msg("FAILED to update job status")
} else {
    logger.Info().Str("job_id", req.JobSpec.ID).Str("status", newStatus).Msg("Successfully updated job status")
}
```

**Purpose**: Track status update attempts and results

#### 3. Added Nil Check Warning (Line 289-291)
```go
} else {
    logger.Warn().Str("job_id", req.JobSpec.ID).Msg("jobsRepo is nil, cannot update job status")
}
```

**Purpose**: Detect if jobsRepo is unexpectedly nil

---

## What These Logs Will Tell Us

### Scenario 1: Goroutine Not Completing
**Logs Missing**:
- ❌ "Cross-region goroutine completed"

**Diagnosis**: Goroutine is hanging or being killed
**Fix**: Investigate execution timeout or context cancellation

---

### Scenario 2: Goroutine Panicking
**Logs Present**:
- ✅ "PANIC in cross-region goroutine"
- ✅ "Cross-region goroutine completed"

**Diagnosis**: Code is crashing before reaching status update
**Fix**: Fix the panic based on error details

---

### Scenario 3: Status Update Not Called
**Logs Present**:
- ✅ "Cross-region goroutine completed"
- ❌ "Attempting to update job status"

**Diagnosis**: Code path exits before reaching status update
**Fix**: Check if result.Status is unexpected value

---

### Scenario 4: Status Update Failing
**Logs Present**:
- ✅ "Cross-region goroutine completed"
- ✅ "Attempting to update job status"
- ✅ "FAILED to update job status" (with error)

**Diagnosis**: UpdateJobStatus is failing (job not found, DB error, etc.)
**Fix**: Based on error message (likely "job not found")

---

### Scenario 5: Status Update Succeeding (Expected!)
**Logs Present**:
- ✅ "Cross-region goroutine completed"
- ✅ "Attempting to update job status"
- ✅ "Successfully updated job status"

**Diagnosis**: Everything working! Issue might be elsewhere
**Fix**: Check if job is actually in database, or API not returning updated data

---

### Scenario 6: jobsRepo is Nil
**Logs Present**:
- ✅ "Cross-region goroutine completed"
- ✅ "jobsRepo is nil, cannot update job status"

**Diagnosis**: Handler not initialized properly
**Fix**: Check routes.go initialization

---

## Deployment Steps

### 1. Build and Test Locally
```bash
cd /Users/Jammie/Desktop/Project\ Beacon/runner-app

# Verify it compiles
go build ./...

# Run tests
go test ./internal/handlers -v
```

### 2. Deploy to Production
```bash
# Deploy to Fly.io
flyctl deploy -a beacon-runner-production

# Watch logs in real-time
flyctl logs -a beacon-runner-production
```

### 3. Submit Test Job
```bash
# From portal, submit a new bias-detection job
# Note the job ID (e.g., bias-detection-1760365xxx)
```

### 4. Monitor Logs
```bash
# Watch for the diagnostic logs
flyctl logs -a beacon-runner-production | grep -E "Cross-region goroutine|Attempting to update|Successfully updated|FAILED to update|PANIC"
```

### 5. Expected Log Sequence (Success Case)
```
[INFO] starting cross-region execution job_id=bias-detection-xxx regions=2
[INFO] Cross-region execution completed status=completed success_count=2 total_regions=2
[INFO] Attempting to update job status job_id=bias-detection-xxx old_status=queued new_status=completed
[INFO] Successfully updated job status job_id=bias-detection-xxx status=completed
[INFO] Cross-region goroutine completed job_id=bias-detection-xxx
```

### 6. Verify Job Status
```bash
# Check if status updated via API
curl -s "https://beacon-runner-production.fly.dev/api/v1/jobs/bias-detection-xxx" | jq .status
# Should return: "completed"
```

---

## Next Steps Based on Findings

### If Logs Show Success But Job Still "queued"
- **Issue**: Job not in database, or API reading wrong data
- **Action**: Query database directly to verify job exists and status is updated

### If Logs Show "job not found" Error
- **Issue**: Job creation failing or using wrong ID
- **Action**: Check job creation logs, verify ID matches

### If Logs Show Panic
- **Issue**: Code bug in execution path
- **Action**: Fix the panic based on stack trace

### If No Logs Appear
- **Issue**: Endpoint not being called, or logs not working
- **Action**: Check if portal is actually calling the endpoint, verify logging configuration

---

## Rollback Plan

If deployment causes issues:

```bash
# Rollback to previous version
flyctl releases -a beacon-runner-production
flyctl releases rollback <previous-version> -a beacon-runner-production
```

Changes are non-breaking (only adding logs), so rollback should not be necessary.

---

## Success Criteria

✅ Logs appear for test job submission
✅ Can identify which scenario is occurring
✅ Have clear next steps based on log output
✅ Job status updates to "completed" (ultimate goal)

---

## Files Modified

- `internal/handlers/cross_region_handlers.go` (3 changes, ~20 lines added)

## Build Status

✅ Code compiles successfully
✅ No breaking changes
✅ Ready to deploy
