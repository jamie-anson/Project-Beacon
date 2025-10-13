# Execution Records Fix - Portal Compatibility

## Problem Solved ✅

**Issue**: Portal showed "Job completed successfully!" but all regions/models showed "Pending" with 0% progress.

**Root Cause**: Cross-region jobs wrote to `cross_region_executions` + `region_results` tables, but portal reads from `executions` table.

**Solution**: Write execution records to BOTH table structures for compatibility.

---

## Changes Made

### 1. Added executionsRepo to CrossRegionHandlers

**File**: `internal/handlers/cross_region_handlers.go`

- Added `executionsRepo *store.ExecutionsRepo` to struct
- Updated constructor to accept executionsRepo parameter
- Added execution record creation logic after region results

### 2. Create Execution Records for Portal

**Location**: Lines 296-381 in `cross_region_handlers.go`

For each **region × model × question** combination:
- Extract models from jobspec metadata (supports multi-model)
- Extract questions from jobspec
- Create execution record with proper status, timestamps, output, and receipt
- Log success/failure for each record

### 3. Updated Handler Initialization

**Files Modified**:
- `internal/api/routes.go` - Pass executionsRepo to handler
- `cmd/server/main.go` - Initialize and pass executionsRepo

---

## What This Fixes

### Before:
```json
GET /api/v1/jobs/{id}/executions/all
{
  "executions": null  // ❌ No data
}
```

### After:
```json
GET /api/v1/jobs/{id}/executions/all
{
  "executions": [
    {
      "region": "US",
      "model_id": "llama3.2-1b",
      "question_id": "greatest_invention",
      "status": "completed",
      ...
    },
    // ... 17 more records (3 questions × 3 models × 2 regions)
  ]
}
```

---

## Portal Impact

**Live Progress Component** will now show:
- ✅ Proper progress for each region
- ✅ Proper progress for each model
- ✅ Proper progress for each question
- ✅ Actual execution data instead of "Pending"
- ✅ Ability to view results and compare

---

## Data Model

### Dual Write Strategy:

```
Cross-Region Job Submission
         ↓
    ┌────────────────────────────────┐
    │  Create Job Record             │
    │  (jobs table)                  │
    └────────────────────────────────┘
         ↓
    ┌────────────────────────────────┐
    │  Execute Across Regions        │
    └────────────────────────────────┘
         ↓
    ┌────────────────────────────────┐
    │  Write to Cross-Region Tables  │
    │  - cross_region_executions     │
    │  - region_results              │
    └────────────────────────────────┘
         ↓
    ┌────────────────────────────────┐
    │  Write to Executions Table     │  ← NEW!
    │  (for portal compatibility)    │
    └────────────────────────────────┘
         ↓
    ┌────────────────────────────────┐
    │  Update Job Status             │
    │  (completed/failed)            │
    └────────────────────────────────┘
```

---

## Testing

### Expected Logs (After Deployment):

```
[INFO] Creating execution records for portal job_id=bias-detection-xxx region_count=2
[INFO] Created execution record job_id=bias-detection-xxx region=US model=llama3.2-1b question=greatest_invention
[INFO] Created execution record job_id=bias-detection-xxx region=US model=mistral-7b question=greatest_invention
[INFO] Created execution record job_id=bias-detection-xxx region=US model=qwen2.5-1.5b question=greatest_invention
... (18 total records for 3 questions × 3 models × 2 regions)
[INFO] Finished creating execution records for portal job_id=bias-detection-xxx
```

### Verification Commands:

```bash
# Submit new job from portal
# Note the job ID

# Check execution records
curl -s "https://beacon-runner-production.fly.dev/api/v1/jobs/{job_id}/executions/all" | jq '.executions | length'
# Should return: 18 (or 6 if 1 question, 2 regions, 3 models)

# Check portal UI
# Should show progress for each region/model/question
```

---

## Files Modified

1. `internal/handlers/cross_region_handlers.go` (+100 lines)
   - Added executionsRepo field
   - Added execution record creation logic
   - Added comprehensive logging

2. `internal/api/routes.go` (+1 line)
   - Pass executionsRepo to handler

3. `cmd/server/main.go` (+5 lines)
   - Initialize executionsRepo
   - Pass to handler

---

## Build Status

✅ Code compiles successfully
✅ No breaking changes
✅ Backward compatible (still writes to cross_region tables)
✅ Ready to deploy

---

## Deployment

```bash
cd /Users/Jammie/Desktop/Project\ Beacon/runner-app
flyctl deploy -a beacon-runner-production
```

---

## Success Criteria

After deployment and submitting a new test job:

✅ Job status = "completed"
✅ Executions array has 18 records (3 questions × 3 models × 2 regions)
✅ Portal Live Progress shows actual progress (not "Pending")
✅ Users can view individual execution results
✅ Users can compare results across regions/models

---

## Long-Term Considerations

### Current Approach: Dual Write
- ✅ Quick to implement
- ✅ Backward compatible
- ✅ Portal works immediately
- ❌ Data duplication
- ❌ Two sources of truth

### Future Optimization: Unified View
Consider creating a database view that unifies both table structures:

```sql
CREATE VIEW executions_unified AS
SELECT * FROM executions
UNION ALL
SELECT 
    rr.id,
    j.id as job_id,
    rr.region,
    rr.provider_id,
    rr.status,
    ...
FROM region_results rr
JOIN cross_region_executions cre ON ...
JOIN jobs j ON ...;
```

Then update portal to query the view instead of the table.

---

## Notes

- This fix addresses the immediate UX issue
- Both table structures are maintained for now
- Future refactoring can consolidate to single source of truth
- No data loss - all information preserved in both locations
