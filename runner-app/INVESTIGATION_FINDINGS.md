# Database Persistence Investigation - Findings

## Status: Code Review Complete ✅

### What I Found

#### ✅ Job Status Update Code EXISTS
**Location**: `internal/handlers/cross_region_handlers.go:263-273`

```go
if h.jobsRepo != nil {
    newStatus := "running"
    if result.Status == "completed" {
        newStatus = "completed"
    } else if result.Status == "failed" {
        newStatus = "failed"
    }
    if err := h.jobsRepo.UpdateJobStatus(execCtx, req.JobSpec.ID, newStatus); err != nil {
        logger.Error().Err(err).Str("job_id", req.JobSpec.ID).Msg("failed to update job status")
    }
}
```

#### ✅ UpdateJobStatus Method EXISTS
**Location**: `internal/store/jobs_repo.go:138-167`

```go
func (r *JobsRepo) UpdateJobStatus(ctx context.Context, jobspecID string, status string) error {
    result, err := r.DB.ExecContext(ctx, `
        UPDATE jobs 
        SET status = $1, updated_at = NOW() 
        WHERE jobspec_id = $2
    `, status, jobspecID)
    
    if err != nil {
        return fmt.Errorf("failed to update job status: %w", err)
    }
    
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }
    
    if rowsAffected == 0 {
        return fmt.Errorf("job not found: %s", jobspecID)
    }
    
    return nil
}
```

#### ✅ Handler Properly Initialized
**Location**: `internal/api/routes.go:101`

```go
biasAnalysisHandler = handlers.NewCrossRegionHandlers(
    crossRegionExecutor, 
    crossRegionRepo, 
    diffEngine, 
    jobsRepo  // ← jobsRepo is passed in
)
```

#### ✅ Route Registered
**Location**: `internal/api/routes.go:139`

```go
jobs.POST("/cross-region", biasAnalysisHandler.SubmitCrossRegionJob)
```

---

## 🤔 So Why Isn't It Working?

### Hypothesis 1: Goroutine Not Completing (Most Likely)
The async goroutine might be:
- Panicking silently
- Hanging on execution
- Getting cancelled before reaching the status update

**Evidence Needed**:
- Check logs for "Cross-region goroutine completed"
- Check logs for "failed to update job status"
- Check logs for panics

### Hypothesis 2: Job Not Created in Database
The job might not be created in the `jobs` table at all, so the UPDATE finds nothing.

**Evidence Needed**:
- Check if `rowsAffected = 0` errors are logged
- Query database directly for job existence

### Hypothesis 3: Different Code Path
Portal might be calling a different endpoint that doesn't use the cross-region handler.

**Evidence Needed**:
- Check portal code for which endpoint it calls
- Check if there's a legacy `/api/v1/jobs` endpoint being used instead

### Hypothesis 4: Database Connection Issue
The `h.jobsRepo.DB` might be nil or disconnected in the goroutine.

**Evidence Needed**:
- Check logs for "database connection is nil"
- Enable DB_AUDIT=1 to see SQL execution

---

## 🎯 Immediate Action Items

### 1. Add Goroutine Completion Logging
**File**: `internal/handlers/cross_region_handlers.go`
**Line**: 168 (start of goroutine)

```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            logger.Error().Interface("panic", r).Str("job_id", req.JobSpec.ID).Msg("PANIC in cross-region goroutine")
        }
        logger.Info().Str("job_id", req.JobSpec.ID).Msg("Cross-region goroutine completed")
    }()
    
    // ... existing code ...
}()
```

### 2. Add More Detailed Status Update Logging
**File**: `internal/handlers/cross_region_handlers.go`
**Line**: 270

```go
logger.Info().
    Str("job_id", req.JobSpec.ID).
    Str("old_status", "queued").
    Str("new_status", newStatus).
    Msg("Attempting to update job status")

if err := h.jobsRepo.UpdateJobStatus(execCtx, req.JobSpec.ID, newStatus); err != nil {
    logger.Error().Err(err).Str("job_id", req.JobSpec.ID).Msg("failed to update job status")
} else {
    logger.Info().Str("job_id", req.JobSpec.ID).Str("status", newStatus).Msg("Successfully updated job status")
}
```

### 3. Enable DB_AUDIT Logging
**Command**:
```bash
flyctl secrets set DB_AUDIT=1 -a beacon-runner-production
flyctl deploy -a beacon-runner-production
```

### 4. Check Portal Endpoint
**File**: Check portal code to see which endpoint it's calling

---

## 📊 What We Know vs What We Need

### ✅ What We Know
- Code exists and looks correct
- Handler is initialized properly
- Route is registered
- SQL query is correct (uses `jobspec_id`)

### ❓ What We Don't Know
- Is the goroutine completing?
- Is the UpdateJobStatus being called?
- Is it returning an error?
- Is the job even in the database?
- Which endpoint is the portal actually calling?

---

## 🚀 Next Steps (In Order)

1. **Add logging** (changes above) and deploy
2. **Submit a new test job** from portal
3. **Watch logs** for:
   - "Cross-region goroutine completed"
   - "Attempting to update job status"
   - "Successfully updated job status" OR error
4. **Based on logs**, determine root cause:
   - If goroutine not completing → investigate execution hanging
   - If status update not called → goroutine exiting early
   - If status update called but failing → check error message
   - If no logs at all → wrong endpoint being called

---

## 💡 Quick Test Without Deploy

Check if the portal is using the correct endpoint:

```bash
# Check portal source for API calls
grep -r "cross-region\|/jobs" Website/portal/src/
```

Check recent production logs for which endpoint is being hit:

```bash
# This would show which endpoint received the request
flyctl logs -a beacon-runner-production | grep "POST /api/v1/jobs"
```
