# Bias Analysis 404 Fix Plan

**Date:** 2025-10-08  
**Issue:** Portal Bias Detection results page returns 404 for completed jobs  
**Root Cause:** Workflow mismatch between job submission and analysis retrieval

---

## Problem Analysis

### What's Happening
1. ✅ Portal submits bias detection job via `/api/v1/jobs` (standard endpoint)
2. ✅ Job executes successfully (12 executions completed)
3. ✅ Executions stored in `executions` table
4. ❌ Portal requests analysis via `/api/v2/jobs/{jobId}/bias-analysis`
5. ❌ Endpoint looks for data in `cross_region_executions` table
6. ❌ Table is empty → 404 error

### Root Cause
**Two separate job submission workflows exist:**

#### Workflow A: Standard Jobs (Currently Used by Portal)
- Endpoint: `POST /api/v1/jobs`
- Handler: `internal/api/handlers_jobs.go`
- Creates: `jobs` table entry
- Creates: `executions` table entries
- Does NOT create: `cross_region_executions` entry

#### Workflow B: Cross-Region Jobs (Not Used by Portal)
- Endpoint: `POST /api/v1/jobs/cross-region`
- Handler: `internal/handlers/cross_region_handlers.go:SubmitCrossRegionJob`
- Creates: `jobs` table entry
- Creates: `cross_region_executions` entry
- Creates: `region_results` entries
- Creates: `cross_region_analyses` entry

### The Mismatch
- Portal uses **Workflow A** to submit jobs
- Bias analysis endpoint expects data from **Workflow B**
- Result: Data exists in wrong tables

---

## Solution Options

### Option 1: Portal Uses Cross-Region Endpoint ⭐ RECOMMENDED
**Change portal to submit via `/api/v1/jobs/cross-region`**

**Pros:**
- Uses existing, tested infrastructure
- Proper separation of concerns
- Analysis data automatically generated
- No backend changes needed

**Cons:**
- Requires portal code changes
- Need to verify endpoint compatibility with current job format

**Implementation:**
1. Update portal job submission to use `/api/v1/jobs/cross-region`
2. Verify request format matches expected schema
3. Test end-to-end flow

**Files to Change:**
- `Website/portal/src/lib/api/runner/jobs.js` - Change endpoint URL
- `Website/portal/src/pages/BiasDetection.jsx` - Verify payload format

---

### Option 2: Backfill Cross-Region Data
**Add background job to populate `cross_region_executions` from existing jobs**

**Pros:**
- Fixes historical data
- No portal changes needed
- Can run as one-time migration

**Cons:**
- Doesn't prevent future issues
- Complex data transformation
- May miss nuances of proper workflow

**Implementation:**
1. Create migration script
2. Query jobs with multiple regions
3. Create corresponding `cross_region_executions` entries
4. Generate analysis data from executions

---

### Option 3: Make Bias Analysis Work with Standard Jobs
**Modify bias-analysis endpoint to work with `executions` table directly**

**Pros:**
- No portal changes
- Works with existing data
- Backward compatible

**Cons:**
- Duplicates logic
- Bypasses proper workflow
- May miss cross-region specific features
- Technical debt

**Implementation:**
1. Modify `GetJobBiasAnalysis` handler
2. Add fallback to query `executions` table
3. Generate analysis on-the-fly if needed

---

### Option 4: Merge Workflows
**Make standard job endpoint create cross-region data for multi-region jobs**

**Pros:**
- Single workflow for all jobs
- Automatic analysis generation
- Clean architecture

**Cons:**
- Significant backend refactoring
- Risk of breaking existing functionality
- Need to handle both single and multi-region cases

**Implementation:**
1. Detect multi-region jobs in standard endpoint
2. Call cross-region creation logic
3. Ensure backward compatibility

---

## Recommended Approach: Option 1

**Use the cross-region endpoint** - it's the cleanest solution that uses existing, tested infrastructure.

### Implementation Steps

#### Phase 1: Investigate Current Endpoint ✅
- [x] Confirm `/api/v1/jobs/cross-region` exists
- [x] Review expected request format
- [x] Compare with portal's current payload
- [ ] Test endpoint manually with portal-like payload

#### Phase 2: Update Portal Job Submission
- [ ] Locate job submission code in portal
- [ ] Change endpoint from `/api/v1/jobs` to `/api/v1/jobs/cross-region`
- [ ] Verify payload format matches expected schema
- [ ] Add any missing required fields

#### Phase 3: Test End-to-End
- [ ] Submit test job from portal
- [ ] Verify `cross_region_executions` entry created
- [ ] Wait for job completion
- [ ] Verify analysis data generated
- [ ] Test bias analysis page loads correctly

#### Phase 4: Handle Edge Cases
- [ ] Test with single-region jobs (if supported)
- [ ] Test with failed jobs
- [ ] Test with partial completions
- [ ] Verify error handling

#### Phase 5: Backfill Historical Data (Optional)
- [ ] Create migration script for existing jobs
- [ ] Test on staging data
- [ ] Run on production
- [ ] Verify historical jobs now have analysis

---

## Technical Details

### Cross-Region Endpoint Request Format
```json
POST /api/v1/jobs/cross-region
{
  "job_spec": {
    "id": "bias-detection-{timestamp}",
    "version": "v1",
    "benchmark": { ... },
    "constraints": {
      "regions": ["US", "EU"],
      "min_regions": 1,
      "min_success_rate": 0.67
    }
  },
  "target_regions": ["US", "EU"],
  "min_regions": 1,
  "min_success_rate": 0.67
}
```

### Current Portal Payload (Standard Endpoint)
```json
POST /api/v1/jobs
{
  "jobspec_id": "bias-detection-{timestamp}",
  "version": "v1",
  "benchmark": { ... },
  "constraints": {
    "regions": ["US", "EU"],
    "min_regions": 1
  },
  "questions": [...],
  "models": [...]
}
```

### Key Differences
1. Cross-region uses `job_spec` wrapper
2. Cross-region has top-level `target_regions`
3. Cross-region has top-level `min_success_rate`
4. Field naming: `jobspec_id` vs `id`

---

## Database Schema Reference

### Tables Involved

#### `jobs` (Used by both workflows)
- Stores job metadata
- Created by both endpoints

#### `executions` (Used by both workflows)
- Stores individual execution results
- Created by both endpoints
- Has optional `cross_region_execution_id` FK

#### `cross_region_executions` (Only Workflow B)
- Tracks multi-region job execution
- Links to multiple `region_results`
- Required for bias analysis endpoint

#### `region_results` (Only Workflow B)
- Per-region execution results
- Linked to `cross_region_executions`

#### `cross_region_analyses` (Only Workflow B)
- Bias analysis metrics
- Generated after job completion
- Required for bias analysis endpoint

---

## Testing Checklist

### Manual Testing
- [ ] Submit job via cross-region endpoint
- [ ] Verify database entries created
- [ ] Wait for completion
- [ ] Check bias analysis endpoint returns data
- [ ] Verify portal displays results correctly

### Automated Testing
- [ ] Add test for cross-region job submission
- [ ] Add test for bias analysis retrieval
- [ ] Add integration test for full workflow
- [ ] Add test for error cases

### Edge Cases
- [ ] Job with single region
- [ ] Job with 3+ regions
- [ ] Job with partial failures
- [ ] Job timeout scenarios
- [ ] Invalid region specifications

---

## Rollout Plan

### Stage 1: Development
1. Update portal code locally
2. Test against local Runner
3. Verify all tests pass

### Stage 2: Staging
1. Deploy portal changes to staging
2. Deploy Runner changes (if any) to staging
3. Run integration tests
4. Manual QA testing

### Stage 3: Production
1. Deploy Runner changes first (backward compatible)
2. Deploy portal changes
3. Monitor for errors
4. Run backfill script for historical data (optional)

### Stage 4: Verification
1. Submit test job from production portal
2. Verify analysis page loads
3. Check error logs
4. Monitor user feedback

---

## Rollback Plan

If issues arise:
1. Revert portal to use `/api/v1/jobs` endpoint
2. Keep Runner changes (backward compatible)
3. Investigate issues
4. Fix and redeploy

---

## Success Criteria

- [ ] Portal can submit bias detection jobs
- [ ] Jobs execute successfully
- [ ] Bias analysis data is generated
- [ ] Analysis page loads without 404 errors
- [ ] Historical jobs can be backfilled (optional)
- [ ] No regression in existing functionality

---

## Next Steps

1. **Immediate:** Test `/api/v1/jobs/cross-region` endpoint manually
2. **Next:** Update portal job submission code
3. **Then:** Test end-to-end flow
4. **Finally:** Deploy and monitor

---

## Related Files

### Backend (Runner)
- `internal/handlers/cross_region_handlers.go` - Cross-region job submission
- `internal/api/handlers_jobs.go` - Standard job submission
- `internal/store/cross_region_repo.go` - Database operations
- `migrations/0007_cross_region_executions.up.sql` - Schema

### Frontend (Portal)
- `portal/src/lib/api/runner/jobs.js` - Job submission API
- `portal/src/pages/BiasDetection.jsx` - Job submission UI
- `portal/src/pages/BiasDetectionResults.jsx` - Results display

---

## Notes

- The cross-region endpoint already exists and is tested
- No new backend functionality needed
- Main work is portal integration
- Consider adding better error messages for missing analysis data
- May want to add "Analysis pending" state in portal UI
