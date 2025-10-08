# Bias Analysis 404 Fix - Implementation Status

**Date:** 2025-10-08  
**Status:** Phase 1 Complete - Ready for Deployment

---

## Summary

Successfully identified and fixed the root cause of bias analysis 404 errors. The cross-region endpoint exists in code but wasn't registered in the production runner's routes.

---

## Changes Made

### 1. Backend Route Registration
**File:** `/Users/Jammie/Desktop/Project Beacon/runner-app/internal/api/routes.go`

**Change:** Added cross-region job submission endpoint (lines 107-109)

```go
// Cross-region job submission endpoint
if biasAnalysisHandler != nil {
    jobs.POST("/cross-region", biasAnalysisHandler.SubmitCrossRegionJob)
}
```

**Impact:**
- Endpoint now available at `POST /api/v1/jobs/cross-region`
- Uses existing `handlers.CrossRegionHandlers` infrastructure
- Automatically creates `cross_region_executions` and `cross_region_analyses` entries
- No breaking changes to existing endpoints

---

## Testing Script Created

**File:** `/Users/Jammie/Desktop/Project Beacon/Website/scripts/test-cross-region-endpoint.js`

**Purpose:**
- Tests cross-region endpoint with portal-like payload
- Validates payload transformation
- Checks signature requirements
- Provides detailed analysis of responses

**Usage:**
```bash
cd /Users/Jammie/Desktop/Project\ Beacon/Website
RUNNER_URL=https://beacon-runner-production.fly.dev node scripts/test-cross-region-endpoint.js
```

---

## Deployment Steps

### 1. Deploy Runner App
```bash
cd /Users/Jammie/Desktop/Project\ Beacon/runner-app

# Verify changes
git diff internal/api/routes.go

# Commit changes
git add internal/api/routes.go
git commit -m "feat: add cross-region job submission endpoint

- Register POST /api/v1/jobs/cross-region in routes
- Fixes bias analysis 404 errors
- Uses existing CrossRegionHandlers infrastructure
- Ref: BIAS_ANALYSIS_404_FIX_PLAN.md"

# Push to trigger Fly.io deployment
git push origin main
```

### 2. Wait for Deployment
Monitor Fly.io deployment:
```bash
fly logs -a beacon-runner-production
```

Look for:
- Successful build completion
- Server startup on port 8090
- No route registration errors

### 3. Test Endpoint
```bash
# Test endpoint is available
curl -X POST https://beacon-runner-production.fly.dev/api/v1/jobs/cross-region \
  -H "Content-Type: application/json" \
  -d '{"job_spec":{"id":"test"},"target_regions":["US"]}'

# Should return 400 (validation error) instead of 404
# 404 = endpoint not found
# 400 = endpoint found, payload invalid
```

### 4. Run Full Test
```bash
cd /Users/Jammie/Desktop/Project\ Beacon/Website
RUNNER_URL=https://beacon-runner-production.fly.dev node scripts/test-cross-region-endpoint.js
```

---

## Next Steps After Deployment

### Phase 1.5: Authentication Testing
- [ ] Test with portal's Ed25519 signing keys
- [ ] Verify signature canonicalization with `job_spec` wrapper
- [ ] Check if `wallet_auth` is required
- [ ] Confirm TRUSTED_KEYS_FILE includes portal key

### Phase 2: Portal Integration
- [ ] Update `portal/src/lib/api/runner/jobs.js`
- [ ] Change `createJob()` to use `/jobs/cross-region` endpoint
- [ ] Transform payload format (add `job_spec` wrapper)
- [ ] Extract `target_regions` from `constraints.regions`
- [ ] Test end-to-end job submission

### Phase 3: Verification
- [ ] Submit test job from portal
- [ ] Verify `cross_region_executions` entry created
- [ ] Wait for job completion
- [ ] Check bias analysis page loads without 404
- [ ] Verify analysis data is populated

---

## Payload Transformation Reference

### Current Portal Payload (Standard Endpoint)
```json
{
  "jobspec_id": "bias-detection-1234",
  "version": "v1",
  "benchmark": {...},
  "constraints": {
    "regions": ["US", "EU"],
    "min_regions": 1,
    "min_success_rate": 0.67
  },
  "questions": [...],
  "models": [...]
}
```

### Required Cross-Region Payload
```json
{
  "job_spec": {
    "id": "bias-detection-1234",  // renamed from jobspec_id
    "version": "v1",
    "benchmark": {...},
    "constraints": {
      "regions": ["US", "EU"],
      "min_regions": 1,
      "min_success_rate": 0.67
    },
    "questions": [...],
    "models": [...]
  },
  "target_regions": ["US", "EU"],  // extracted from constraints
  "min_regions": 1,
  "min_success_rate": 0.67,
  "enable_analysis": true
}
```

---

## Expected Behavior After Fix

### Before (Current State)
1. Portal submits job via `/api/v1/jobs` ✅
2. Job executes successfully ✅
3. Data stored in `executions` table ✅
4. Portal requests `/api/v2/jobs/{id}/bias-analysis` ❌
5. Endpoint returns 404 (no data in `cross_region_executions`) ❌

### After (Fixed State)
1. Portal submits job via `/api/v1/jobs/cross-region` ✅
2. Job executes successfully ✅
3. Data stored in `cross_region_executions` table ✅
4. Analysis generated in `cross_region_analyses` table ✅
5. Portal requests `/api/v2/jobs/{id}/bias-analysis` ✅
6. Endpoint returns analysis data ✅

---

## Rollback Plan

If issues arise after deployment:

1. **Immediate:** Revert commit and redeploy
   ```bash
   git revert HEAD
   git push origin main
   ```

2. **Portal stays on standard endpoint** (no portal changes yet)
   - No user impact since portal hasn't switched endpoints
   - Gives time to debug issues

3. **Investigate and fix**
   - Check Fly.io logs for errors
   - Test locally with proper payload
   - Fix issues and redeploy

---

## Success Criteria

- [ ] Endpoint returns 400 (not 404) for invalid payloads
- [ ] Endpoint accepts valid cross-region payloads
- [ ] Creates `cross_region_executions` entries
- [ ] Generates `cross_region_analyses` data
- [ ] Bias analysis endpoint returns data (not 404)
- [ ] No regression in existing `/api/v1/jobs` endpoint

---

## Files Modified

### Backend
- `runner-app/internal/api/routes.go` (lines 107-109)

### Testing
- `Website/scripts/test-cross-region-endpoint.js` (new file)

### Documentation
- `BIAS_ANALYSIS_404_FIX_PLAN.md` (updated with findings)
- `BIAS_ANALYSIS_IMPLEMENTATION_STATUS.md` (this file)

---

## Notes

- The cross-region handlers already exist in `internal/handlers/cross_region_handlers.go`
- The endpoint was just not registered in the production runner's routes
- This is a minimal, low-risk change (3 lines of code)
- No database migrations needed
- No breaking changes to existing functionality
- Portal changes can be done separately after endpoint is verified working
