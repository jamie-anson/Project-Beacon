# Compare Page Blank - Debug Plan

## Problem

**URL**: https://projectbeacon.netlify.app/portal/results/bias-detection-1760469397144/diffs  
**Issue**: Page is blank (white screen)  
**Job ID**: `bias-detection-1760469397144` (new job submitted after fix)

---

## Diagnostic Steps

### Step 1: Check Browser Console

Open browser console (F12) and look for:

**Expected Logs**:
```
🔍 Getting cross-region diff for job: bias-detection-1760469397144
```

**Possible Error Messages**:
- API 404 errors
- API 500 errors
- JavaScript errors
- React rendering errors

### Step 2: Check Network Tab

1. Open Network tab in DevTools
2. Filter for "XHR" or "Fetch"
3. Look for API calls to:
   - `/executions/bias-detection-1760469397144/cross-region-diff`
   - `/executions/bias-detection-1760469397144/regions`
   - `/executions/bias-detection-1760469397144/cross-region`
   - `/jobs/bias-detection-1760469397144/executions/all`

**Check**:
- [ ] Which endpoint is being called?
- [ ] What's the HTTP status code?
- [ ] What's the response body?

### Step 3: Check React DevTools

1. Install React DevTools extension
2. Open Components tab
3. Find `CrossRegionDiffPage` component
4. Check props and state:
   - `loading`: should be `false` after load
   - `error`: should be `null` if no error
   - `diffAnalysis`: should have data
   - `job`: should have job data

---

## Root Cause Hypotheses

### Hypothesis A: API Endpoint Missing ⚠️ LIKELY

**Problem**: Cross-region jobs don't have the `/executions/:id/cross-region` endpoint

**Evidence**:
- New jobs use `/jobs/cross-region` submission endpoint
- But results page expects `/executions/:id/cross-region` data endpoint
- These are different endpoints!

**Fix**: Need to check which endpoint the backend actually provides for cross-region results

### Hypothesis B: Data Structure Mismatch

**Problem**: API returns data but in unexpected format

**Evidence**:
- Page expects specific structure from `transformCrossRegionDiff()`
- New cross-region endpoint may return different structure

**Fix**: Update transform function or page to handle new structure

### Hypothesis C: Job ID vs Execution ID Confusion

**Problem**: Page is looking for execution ID but receiving job ID

**Evidence**:
- Cross-region jobs have both `job_id` and `cross_region_execution_id`
- Page may be using wrong ID

**Fix**: Update button to pass correct ID

### Hypothesis D: Silent JavaScript Error

**Problem**: React error boundary not catching error

**Evidence**:
- Blank page suggests rendering failure
- No error message displayed

**Fix**: Add error boundary or check console for errors

---

## Quick Tests

### Test 1: Check if Job Exists

```bash
curl https://beacon-runner-production.fly.dev/api/v1/jobs/bias-detection-1760469397144
```

**Expected**: Job data with status "completed"

### Test 2: Check Executions

```bash
curl https://beacon-runner-production.fly.dev/api/v1/jobs/bias-detection-1760469397144/executions/all
```

**Expected**: Array of execution records

### Test 3: Check Cross-Region Endpoint

```bash
curl https://beacon-runner-production.fly.dev/api/v1/executions/bias-detection-1760469397144/cross-region
```

**Expected**: Cross-region analysis data OR 404

### Test 4: Check Alternative Endpoint

```bash
curl https://beacon-runner-production.fly.dev/api/v1/executions/bias-detection-1760469397144/regions
```

**Expected**: Region results data OR 404

---

## Likely Issue

Based on the code analysis, the **most likely issue** is:

**The Compare button is passing the job ID, but the page expects data from cross-region-specific endpoints that may not exist or may require the cross_region_execution_id instead.**

### Why This Happens

1. **Job Submission** (fixed):
   - ✅ Uses `POST /api/v1/jobs/cross-region`
   - ✅ Creates `cross_region_executions` table record
   - ✅ Returns both `job_id` and `cross_region_execution_id`

2. **Compare Button** (just fixed):
   - ✅ Navigates to `/portal/results/:jobId/diffs`
   - ✅ Passes job ID correctly

3. **Compare Page** (ISSUE):
   - ❌ Calls `getCrossRegionDiff(jobId)`
   - ❌ Tries endpoints like `/executions/:jobId/cross-region`
   - ❌ These endpoints may not exist or may need `cross_region_execution_id`

---

## Potential Fixes

### Option 1: Backend - Add Missing Endpoint

**Add endpoint**: `GET /api/v1/executions/:jobId/cross-region`

**Returns**: Cross-region analysis data for the job

**Pros**: Matches frontend expectations  
**Cons**: Requires backend changes

### Option 2: Frontend - Use Fallback Construction

**Current behavior**: Page already has fallback to construct from executions

**Issue**: Fallback may be failing silently

**Fix**: Add better error handling and logging

### Option 3: Frontend - Use Different Endpoint

**Change**: Update `getCrossRegionDiff()` to call correct endpoint

**Example**: `/api/v2/jobs/:jobId/bias-analysis` (if that has cross-region data)

**Pros**: Uses existing endpoint  
**Cons**: May not have all needed data

### Option 4: Update Button to Pass Execution ID

**Change**: Compare button passes `cross_region_execution_id` instead of `job_id`

**Route**: `/portal/results/:executionId/diffs`

**Pros**: Matches backend data model  
**Cons**: Requires button and route changes

---

## Immediate Actions

1. **Open browser console** on blank page
2. **Check for errors** (JavaScript or API)
3. **Check Network tab** for API calls and responses
4. **Share findings** so we can determine correct fix

---

## Expected Console Output (If Working)

```
🔍 Getting cross-region diff for job: bias-detection-1760469397144
✅ Main backend succeeded
🎯 Auto-selecting first available model: llama3.2-1b
🔍 Model Selection Debug: { selectedModel: "llama3.2-1b", ... }
```

## Expected Console Output (If Failing)

```
🔍 Getting cross-region diff for job: bias-detection-1760469397144
⚠️  All endpoints failed, constructing from execution data...
✅ Successfully constructed cross-region diff from execution data
📊 Analysis: Cross-region analysis for 2 regions with 6 total executions (100% success rate)
```

OR

```
🔍 Getting cross-region diff for job: bias-detection-1760469397144
❌ Fallback construction failed: [error message]
```

---

**Next Step**: Check browser console and share the actual error messages!
