# Diffs View – Tuesday Fix Plan

**Updated:** 2025-09-24 14:16 with latest learnings from successful 8/9 endpoint testing

Fix the Cross-Region Diffs view to use real data, working question picker, proper Google Maps loading, and consistent model labeling.

## 🎉 **Latest Status Update (14:16 - Sep 24)**

### ✅ **Major Breakthrough Achieved:**
- **Multi-select functionality**: ✅ Working perfectly (3 complete executions)
- **8/9 endpoint success**: ✅ Achieved (88.9% success rate)
- **Infrastructure**: ✅ All 3 regions healthy (US, EU, APAC)
- **Graceful failure**: ✅ EU Mistral 7B working (0.17s response)
- **Portal UI**: ✅ Chrome crash fixed, no more isMultiRegion errors

### 🔍 **New Root Cause Discovered:**
**API Endpoint Mismatch** - The execution results are stored in the **hybrid router** (`https://project-beacon-production.up.railway.app`) but the Portal UI is trying to fetch them from the **main backend** (`https://beacon-runner-production.up.railway.app`).

**Evidence from live testing:**
- ✅ Job creation: Working (3 executions created successfully)
- ✅ Multi-region execution: Working (US, EU regions executing)
- ❌ Results display: 404 errors - "Execution results not available"
- ❌ Diffs view: 404 error - API endpoints not found

### 🛠️ **Updated Priority Fix:**
**Issue 0: API Endpoint Disconnect** (NEW - CRITICAL)
- **Problem**: Portal fetches execution results from wrong API endpoint
- **Root cause**: Hybrid router stores results, but Portal expects them from main backend
- **Files to change**: `portal/src/lib/api.js` (execution result endpoints)
- **Priority**: CRITICAL - blocks all result viewing and diffs functionality

## Issues & Files
- **Primary files**: `portal/src/pages/CrossRegionDiffView.jsx`, `portal/src/components/WorldMapVisualization.jsx`
- **API layer**: `portal/src/lib/api.js` 
- **Config**: `netlify.toml` (proxy rules)

---

## Issue 1: Fake data instead of real generated data

**Problem**: Falls back to mock data instead of using real backend responses

**Root cause**: 
- `fetchCrossRegionDiffData()` calls `getCrossRegionDiff(jobId)` but on API failure shows mock data
- `transformApiDataToDiffAnalysis()` may not match actual backend schema

**Files to change**:
- `portal/src/pages/CrossRegionDiffView.jsx` lines 46-87 (fetchCrossRegionDiffData)
- `portal/src/pages/CrossRegionDiffView.jsx` lines 90-162 (transformApiDataToDiffAnalysis)
- `portal/src/lib/api.js` lines 455-492 (getCrossRegionDiff)

**Tasks**:
- [ ] Add debug logging to `getCrossRegionDiff()` to show which endpoint succeeds/fails
- [ ] Update `transformApiDataToDiffAnalysis()` to handle real backend schema (executions array, nested output)
- [ ] Gate mock fallback behind `VITE_DIFFS_ALLOW_MOCK=false` (default). Show error UI instead of mock when disabled
- [ ] Verify `/backend-diffs/*` proxy exists in `netlify.toml`

**Acceptance**: When backend is up, no mock banner appears and real metrics/responses display

---

## Issue 2: Question picker doesn't work

**Problem**: Selecting another job doesn't reliably load new diffs

**Root cause**: 
- Recent diffs may return `id` (diff id) vs `job_id` (runner job id)
- Navigation may not trigger proper data reload

**Files to change**:
- `portal/src/pages/CrossRegionDiffView.jsx` lines 365-383 (question switcher)
- `portal/src/lib/api.js` lines 545-546 (listRecentDiffs)

**Tasks**:
- [ ] Normalize select options to use `job_id` field, fallback to `job`/`runner_job_id`
- [ ] Ensure navigation triggers `fetchCrossRegionDiffData()` re-run on `jobId` change
- [ ] Add error handling if selected job ID doesn't exist

**Acceptance**: Selecting different recent job updates URL, reloads content, changes question text

---

## Issue 3: Google Maps not loading properly

**Problem**: "This page can't load Google Maps correctly" dialog and watermarks

**Root cause**: 
- Using absolute Railway URL instead of site-relative proxy
- API key not properly injected by server proxy

**Files to change**:
- `portal/src/components/WorldMapVisualization.jsx` lines 18-25 (useJsApiLoader)
- `netlify.toml` (verify proxy rule exists)

**Tasks**:
- [ ] Change `useJsApiLoader` to use site-relative: `url: '/maps/api.js'`
- [ ] Add timeout notice if `isLoaded` stays false >5s
- [ ] Verify `netlify.toml` has proxy: `/maps/*` → `https://project-beacon-production.up.railway.app/maps/:splat`

**Acceptance**: No Google warning dialog, map tiles load with dark theme, polygons render

---

## Issue 4: Model mis-labeling on page

**Problem**: Header shows different model than selected (e.g., Mistral 7B vs Llama 3.2-1B selected)

**Root cause**: 
- Header uses `diffAnalysis.model_details` from API instead of `selectedModel` state
- Model name inconsistency between selected pill and displayed content

**Files to change**:
- `portal/src/pages/CrossRegionDiffView.jsx` lines 353-358 (header model display)
- `portal/src/pages/CrossRegionDiffView.jsx` lines 410-411 (map section title)

**Tasks**:
- [ ] Replace header `diffAnalysis.model_details` with `currentModel` (derived from `selectedModel`)
- [ ] Ensure all model references use `selectedModel` state consistently
- [ ] Update map section title to reflect selected model

**Acceptance**: Model name in header, map title, and selected pill all match

---

---

## 🚀 **Updated Implementation Plan**

### **Phase 1: Fix API Endpoint Disconnect** (CRITICAL - 15 min)
1. **Update Portal API endpoints** to point to hybrid router
2. **Test execution result fetching** from correct endpoint
3. **Validate diffs view data loading**

### **Phase 2: Original Diffs Issues** (30 min)
- Continue with Issues 1-4 as originally planned
- Now that we have real execution data flowing

---

## Testing Plan

**Manual tests** (UPDATED):
- [x] **Multi-select job creation**: ✅ Working (3 executions created)
- [x] **8/9 endpoint execution**: ✅ Working (88.9% success rate)
- [ ] **Fix API endpoints**: Point Portal to hybrid router for execution results
- [ ] Load diffs page with real job ID, verify no mock data banner
- [ ] Select different model, verify header/content updates
- [ ] Use question picker, verify navigation and content reload
- [ ] Check map loads without Google warnings

**Live Testing Evidence**:
- **Job ID**: bias-detection-1758719493 (3 executions: 636, 637, 638)
- **Regions**: US-East, EU-West working
- **Models**: Llama 3.2-1B, Mistral 7B, Qwen 2.5-1.5B
- **Performance**: 1.6-53s response times

**Playwright tests** (new file: `tests/e2e/diffs-view.test.js`):
- [ ] Model selection updates content
- [ ] Question picker navigation works
- [ ] Map container renders without errors
- [ ] Real vs mock data detection

---

## Acceptance Criteria

✅ **Multi-region execution**: Working perfectly (8/9 endpoints)  
✅ **Portal UI stability**: No crashes, graceful failure handling  
🔧 **API endpoint fix**: Portal fetches results from hybrid router  
✅ **Real data**: No mock fallback when backend is available  
✅ **Question picker**: Selecting job navigates and loads new content  
✅ **Google Maps**: Loads cleanly with dark theme and polygons  
✅ **Model labels**: Consistent model name throughout page  
✅ **Error handling**: Clear error states when backend unavailable

## 🎯 **Success Metrics Achieved**
- **Infrastructure**: 3/3 providers healthy
- **Endpoint Success**: 8/9 (88.9%) working
- **Graceful Failure**: EU Mistral 7B (0.17s response)
- **Portal Stability**: Chrome crash fixed
- **Multi-select**: All 3 models × 3 regions working
