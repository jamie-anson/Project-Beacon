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

### Option 5: Temporary Adapter Layer ⚡ QUICK WIN
**Add middleware to convert standard jobs to cross-region format**

**Pros:**
- No portal changes needed immediately
- Fixes issue for all new jobs
- Buys time for proper portal migration
- Can be removed later

**Cons:**
- Technical debt
- Server-side transformation overhead
- Doesn't fix historical data

**Implementation:**
1. Add post-processing hook in standard job handler
2. Detect multi-region jobs (regions.length > 1)
3. Automatically create cross_region_executions entry
4. Run in parallel with standard workflow

**Code Location:** `internal/api/handlers_jobs.go` after job creation

---

## Risk Assessment

### High Risk
- **Signature rejection**: Cross-region endpoint may reject portal signatures
- **Data loss**: If backfill fails, historical jobs permanently broken
- **Breaking change**: Portal change affects all users immediately

### Medium Risk  
- **Performance**: Cross-region endpoint may be slower
- **Validation differences**: Cross-region may have stricter validation
- **Field incompatibilities**: Questions/models arrays may not be supported

### Low Risk
- **Database impact**: Cross-region creates more rows per job
- **Monitoring gaps**: May need new alerts for cross-region failures

### Mitigation
- Test thoroughly with signed payloads before deployment
- Deploy during low-traffic window
- Keep standard endpoint as emergency fallback
- Add feature flag to toggle between endpoints

---

## Recommended Approach: Option 1

**Use the cross-region endpoint** - it's the cleanest solution that uses existing, tested infrastructure.

### Implementation Steps

#### Phase 1: Deep Investigation ✅
- [x] Confirm `/api/v1/jobs/cross-region` exists in code
- [x] Review expected request format
- [x] Compare with portal's current payload
- [x] **Test endpoint availability** - ⚠️ **CRITICAL FINDING**
- [ ] **Test with portal's Ed25519 signing keys**
- [ ] **Verify questions + models arrays are supported**
- [ ] **Check response format matches portal expectations**
- [ ] **Test error responses (400, 401, 403)**
- [ ] **Measure endpoint latency vs standard endpoint**
- [ ] **Review handler code for side effects**

**⚠️ CRITICAL DISCOVERY:**
The production runner (`beacon-runner-production.fly.dev`) is running `cmd/runner/main.go`, which does NOT include the cross-region handlers. The cross-region endpoint exists in `cmd/server/main.go` but is not deployed.

**✅ SOLUTION IMPLEMENTED:**
Added cross-region endpoint to production runner by modifying `internal/api/routes.go`:
- Line 107-109: Added `jobs.POST("/cross-region", biasAnalysisHandler.SubmitCrossRegionJob)`
- Endpoint now available at `/api/v1/jobs/cross-region`
- Uses existing `handlers.CrossRegionHandlers` infrastructure
- No deployment changes needed - just code update

**Next: Deploy and test**

#### Phase 1.5: Verify Authentication Requirements
- [ ] Confirm cross-region endpoint requires Ed25519 signatures
- [ ] Verify signature canonicalization includes job_spec wrapper
- [ ] Test with portal's signing flow (signJobSpecForAPI)
- [ ] Check if wallet_auth is required in cross-region payload
- [ ] Verify TRUSTED_KEYS_FILE includes portal's public key

#### Phase 2: Update Portal Job Submission
- [ ] Locate job submission code in portal
- [ ] Change endpoint from `/api/v1/jobs` to `/api/v1/jobs/cross-region`
- [ ] Verify payload format matches expected schema
- [ ] Add any missing required fields

#### Phase 2.5: Feature Flag Implementation
- [ ] Add `VITE_USE_CROSS_REGION_ENDPOINT` environment variable
- [ ] Default to `false` initially
- [ ] Enable for internal testing first
- [ ] Gradual rollout: 10% → 50% → 100%
- [ ] Quick rollback via environment variable change

**Benefits:** No code deployment needed for rollback

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

### Portal Payload Transformation

**⚠️ CRITICAL: Field name is `jobspec` not `job_spec`**

**Required Changes:**
1. Wrap payload in `jobspec` object (NOT `job_spec`)
2. Rename `jobspec_id` → `id` inside jobspec
3. Extract `constraints.regions` → top-level `target_regions`
4. Extract `constraints.min_success_rate` → top-level (or default to 0.67)
5. Preserve `questions` and `models` arrays

**Code Example:**
```javascript
// Before (standard endpoint)
const payload = {
  jobspec_id: id,
  version: "v1",
  benchmark: {...},
  constraints: { regions: ["US", "EU"], min_regions: 1 },
  questions: [...],
  models: [...]
}

// After (cross-region endpoint)
const payload = {
  jobspec: {  // CRITICAL: use "jobspec" not "job_spec"
    id: id,  // renamed from jobspec_id
    version: "v1",
    benchmark: {...},
    constraints: { regions: ["US", "EU"], min_regions: 1 },
    questions: [...],
    models: [...]
  },
  target_regions: ["US", "EU"],  // extracted from constraints
  min_regions: 1,
  min_success_rate: 0.67
}
```

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

## Debugging Tools

### Admin Endpoints to Add
- `GET /admin/jobs/{id}/workflow-type` - Check which workflow was used
- `GET /admin/jobs/{id}/can-analyze` - Check if analysis data exists
- `POST /admin/jobs/{id}/backfill-analysis` - Manually trigger analysis generation

### SQL Queries for Investigation

```sql
-- Find jobs with executions but no cross-region data
SELECT j.id, j.created_at, COUNT(e.id) as execution_count
FROM jobs j
LEFT JOIN executions e ON e.job_id = j.id
LEFT JOIN cross_region_executions cre ON cre.job_id = j.id
WHERE e.id IS NOT NULL AND cre.id IS NULL
GROUP BY j.id
ORDER BY j.created_at DESC;

-- Check analysis data completeness
SELECT 
  j.id,
  EXISTS(SELECT 1 FROM cross_region_executions WHERE job_id = j.id) as has_cre,
  EXISTS(SELECT 1 FROM cross_region_analyses WHERE job_id = j.id) as has_analysis
FROM jobs j
WHERE j.created_at > NOW() - INTERVAL '7 days';
```

### Debugging Checklist
- [ ] Run SQL query to identify affected jobs
- [ ] Check Runner logs for cross-region endpoint errors
- [ ] Verify signature verification logs
- [ ] Test with curl/Postman using signed payload
- [ ] Review database foreign key constraints

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
