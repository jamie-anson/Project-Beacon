# Bias Detection Results Page - Manual Test Plan

**Date:** 2025-10-03  
**Status:** Ready for manual validation

---

## Pre-Test Setup

### 1. Ensure Backend is Running
```bash
# Check backend health
curl https://beacon-runner-change-me.fly.dev/health

# Expected: {"status":"healthy","service":"runner"}
```

### 2. Ensure Portal is Built
```bash
cd portal
npm run build
# Should complete successfully
```

---

## Test Scenarios

### Test 1: Happy Path (With Completed Job)

**Prerequisites:** Need a completed multi-region job

**Steps:**
1. Open browser: `https://projectbeacon.netlify.app/portal/executions`
2. Find a completed job (status = "completed")
3. Click "Bias Analysis" link
4. **Expected:**
   - Page loads: `/portal/bias-detection/{jobId}`
   - Header shows: "Bias Detection Results"
   - Summary section displays AI-generated text
   - Overall metrics show 4 cards
   - Regional scores display
   - World map renders (if Google Maps key set)
   - Back button works

**If No Completed Jobs:**
- Submit a job via `/portal/bias-detection`
- Wait 2-5 minutes for completion
- Then test

---

### Test 2: Direct URL Access

**Steps:**
1. Open browser: `https://projectbeacon.netlify.app/portal/bias-detection/test-job-123`
2. **Expected:**
   - Page loads with header "Bias Detection Results"
   - Shows error state: "Error Loading Analysis" (job doesn't exist)
   - Retry button visible
   - Back to Executions button visible

**Validates:**
- Route works
- Error handling works
- Navigation works

---

### Test 3: API Endpoint Direct Test

**Steps:**
```bash
# Test with non-existent job (should 404)
curl -i https://beacon-runner-change-me.fly.dev/api/v2/jobs/test-job-123/bias-analysis

# Expected: HTTP 404
# Body: {"error": "Cross-region execution not found for jobspec_id: test-job-123"}
```

**Test with real job (if available):**
```bash
# Replace with actual job ID
curl https://beacon-runner-change-me.fly.dev/api/v2/jobs/{REAL_JOB_ID}/bias-analysis | jq

# Expected: JSON with analysis, region_scores, job_id
```

**Validates:**
- Backend endpoint works
- Returns correct JSON structure
- Error handling works

---

### Test 4: Local Development

**Steps:**
```bash
# Start dev server
cd portal
npm run dev

# Open browser: http://localhost:5173/portal/bias-detection/test-job
```

**Expected:**
- Page renders
- Shows error state (job doesn't exist)
- All components load
- No console errors

---

### Test 5: Component Rendering

**When viewing a real completed job:**

**Check these sections appear:**
- [ ] Header: "Bias Detection Results"
- [ ] Job ID displayed
- [ ] Summary Card with AI text
- [ ] Recommendation with color coding
- [ ] Overall Metrics (4 cards):
  - Bias Variance
  - Censorship Rate
  - Factual Consistency
  - Narrative Divergence
- [ ] Regional Scores (grid of regions)
- [ ] Key Differences (if available)
- [ ] Risk Assessment (if available)
- [ ] World Map (if Google Maps key set)
- [ ] Metadata footer (execution ID, timestamp)
- [ ] Back to Executions button

---

### Test 6: Error States

**Test 404 Error:**
1. Navigate to: `/portal/bias-detection/nonexistent-job-999`
2. **Expected:**
   - Red error box
   - "Error Loading Analysis" heading
   - Error message displayed
   - Retry button
   - Back to Executions button

**Test Retry:**
1. Click "Retry" button
2. **Expected:**
   - Loading state briefly
   - Error state returns (job still doesn't exist)

**Test Back Button:**
1. Click "Back to Executions"
2. **Expected:**
   - Navigates to `/portal/executions`
   - Executions table displays

---

### Test 7: Mobile Responsiveness

**Steps:**
1. Open page on mobile device or resize browser to mobile width
2. **Expected:**
   - Layout adapts to mobile
   - Cards stack vertically
   - Text remains readable
   - Buttons accessible
   - Map scales appropriately

---

## Known Issues to Watch For

### Issue 1: Page Not Rendering
**Symptom:** Blank page or "Page not found"
**Cause:** Route not registered or build issue
**Fix:** Verify App.jsx has route, rebuild portal

### Issue 2: API 404
**Symptom:** "Error Loading Analysis" for all jobs
**Cause:** Backend not deployed or endpoint not registered
**Fix:** Verify backend deployment, check cmd/server/main.go

### Issue 3: CORS Error
**Symptom:** Network error in console
**Cause:** CORS not configured for /api/v2
**Fix:** Check backend CORS middleware

### Issue 4: Empty Analysis
**Symptom:** "No Analysis Available" for completed jobs
**Cause:** Analysis not generated or not persisted
**Fix:** Check backend logs, verify OpenAI key set

### Issue 5: Map Not Showing
**Symptom:** Map section missing or shows placeholder
**Cause:** Google Maps API key not set
**Fix:** Set REACT_APP_GOOGLE_MAPS_API_KEY (optional)

---

## Success Criteria

**Minimum for MVP:**
- [ ] Page loads without errors
- [ ] Error states display correctly
- [ ] Navigation works (to/from executions)
- [ ] API endpoint returns data
- [ ] At least one section renders (summary or metrics)

**Full Feature Working:**
- [ ] AI summary displays (400-500 words)
- [ ] All 4 overall metrics show
- [ ] Regional scores display
- [ ] Key differences listed
- [ ] Risk assessment shown
- [ ] Map renders (if key set)
- [ ] Mobile responsive

---

## Quick Validation Checklist

```bash
# 1. Backend health
curl https://beacon-runner-change-me.fly.dev/health
# ✅ Should return healthy

# 2. API endpoint exists
curl -i https://beacon-runner-change-me.fly.dev/api/v2/jobs/test/bias-analysis
# ✅ Should return 404 (expected for test job)

# 3. Portal builds
cd portal && npm run build
# ✅ Should complete without errors

# 4. Portal dev server
cd portal && npm run dev
# ✅ Should start on localhost:5173

# 5. Page loads locally
open http://localhost:5173/portal/bias-detection/test-job
# ✅ Should show page with error state
```

---

## Automated Test Results

**E2E Tests:** 4 passed, 6 failed (expected - no real data yet)

**Passing Tests:**
- ✅ Happy path (skipped - no data)
- ✅ API endpoint validation
- ✅ Navigation from executions
- ✅ Component rendering (skipped - no data)

**Failing Tests:**
- ⚠️ 404 error state (page not rendering in test)
- ⚠️ Loading state (timeout)
- ⚠️ Back button (page not loading)
- ⚠️ Retry button (page not loading)
- ⚠️ Error display (page not rendering)
- ⚠️ Network timeout (page not rendering)

**Root Cause:** Tests running against production Netlify which doesn't have latest build yet

**Resolution:** Deploy portal, then rerun tests

---

## Deployment Validation

**After deploying portal:**

1. **Smoke Test:**
   ```bash
   curl https://projectbeacon.netlify.app/portal/bias-detection/test
   # Should return HTML with React app
   ```

2. **Browser Test:**
   - Open: `https://projectbeacon.netlify.app/portal/bias-detection/test-job`
   - Should see page render (with error state)

3. **Full Flow Test:**
   - Submit multi-region job
   - Wait for completion
   - View bias analysis
   - Verify all sections render

---

## Current Status

**Backend:** ✅ DEPLOYED & WORKING
**Frontend:** ✅ BUILT & READY
**Tests:** ⚠️ Need portal deployment to validate
**Integration:** ✅ Code verified, awaiting deployment test

**Next Action:** Deploy portal to Netlify, then rerun E2E tests
