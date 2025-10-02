# APAC Pause Handling ✅

## Question
Does the page handle the fact we have paused APAC right now ok?

## Answer: Yes, with Fix Applied ✅

The page now **gracefully handles missing regions** including APAC.

---

## What Happens When APAC is Missing

### UI Behavior

**Region Tabs**:
- Shows only available regions (US, EU)
- No APAC tab appears
- No errors or broken UI

**Metadata Banner**:
- Shows "2/2 regions completed" instead of "3/3"
- Home region still displays correctly

**Key Differences Table**:
- Shows only US and EU columns
- No empty APAC column

**Metrics**:
- Calculated from available regions only
- Bias variance, censorship rate based on US + EU

---

## Fix Applied

### Problem
For Qwen (home region = ASIA), the compare region would default to ASIA even if it doesn't exist.

### Solution
Updated initialization logic to check if home region exists:

```js
// Set compare region to home region if available, otherwise first available region
if (!compareRegion) {
  const homeRegionExists = data.regions.some(r => r.region_code === data.home_region);
  setCompareRegion(homeRegionExists ? data.home_region : data.regions[0].region_code);
}
```

**Result**:
- If ASIA exists → Use ASIA as compare region (Qwen's home)
- If ASIA missing → Use first available region (US or EU)

---

## Test Scenarios

### Scenario 1: All Regions Available (US, EU, ASIA)
✅ Shows 3 tabs
✅ Qwen home badge on ASIA tab
✅ Compare defaults to ASIA for Qwen

### Scenario 2: APAC Paused (US, EU only)
✅ Shows 2 tabs (US, EU)
✅ No APAC tab (no errors)
✅ Qwen compare defaults to US (first available)
✅ No "Home" badge appears (home region not available)
✅ Metrics calculated from 2 regions

### Scenario 3: Only One Region (US)
✅ Shows 1 tab
✅ Compare region same as active region (diff shows no changes)
✅ Metrics calculated from 1 region

---

## Production Considerations

### Current State (APAC Paused)
- **Qwen users** will see US vs EU comparison
- **No home region advantage** visible for Qwen
- **Metrics** reflect US/EU only (may show lower censorship rates)

### When APAC Resumes
- **Automatic** - No code changes needed
- **Qwen home badge** will reappear on ASIA tab
- **Metrics** will include ASIA data
- **Dramatic censorship** will be visible in Qwen's home region

---

## Edge Cases Handled

✅ **No regions available** - Returns null, shows "No data available"
✅ **One region only** - Shows single tab, diff disabled
✅ **Missing home region** - Falls back to first available
✅ **Failed region executions** - Filtered out by transform layer
✅ **Partial data** - Shows what's available, no crashes

---

## Summary

The page is **production-ready** for the current APAC pause:
- No errors or crashes
- Graceful degradation
- Clear UX with available data
- Automatic recovery when APAC resumes

**Status**: ✅ Safe to deploy with APAC paused
