# Layout Update - Restructured Page ✅

## Changes Made

Updated the Layer 2 page layout to match the wireframe design.

---

## What Changed

### 1. Question Header - No Container ✅
**Before**: Header was in a gray container box
**After**: Header is standalone text (no background box)

```jsx
// Before
<div className="bg-gray-800 border border-gray-700 rounded-lg p-6">
  <h1>Question</h1>
</div>

// After
<div>
  <h1>Question</h1>
</div>
```

**Rationale**: Matches other pages - headers are not in containers

---

### 2. Metrics Section Moved Below Responses ✅
**Before**: Metrics grid appeared before region tabs
**After**: Metrics section appears after response viewer

**New Order**:
1. Breadcrumb
2. Metadata banner
3. **Question header** (no container)
4. **Region tabs & response viewer**
5. **Analysis Metrics section** (new container)
6. Key differences table
7. Visualizations
8. Model navigation
9. Provenance footer

---

### 3. Metrics Section - Same Styling ✅
**Before**: Metrics cards had special styling
**After**: Metrics section uses standard container styling

```jsx
<div className="bg-gray-800 border border-gray-700 rounded-lg p-6">
  <h3>Analysis Metrics</h3>
  {/* Risk assessment banner */}
  {/* Metrics grid */}
</div>
```

**Styling**: Same `bg-gray-800 border border-gray-700` as all other sections

---

## Visual Hierarchy

### Before
```
Metadata Banner
Question Header (in box)
Risk Banner
Metrics Grid (4 cards)
Region Tabs
Response Viewer
...
```

### After
```
Metadata Banner
Question Header (no box) ← Cleaner
Region Tabs              ← Priority
Response Viewer          ← Priority
Analysis Metrics         ← Moved down, in container
  - Risk Banner
  - Metrics Grid
Key Differences
Visualizations
...
```

---

## Benefits

✅ **Response takes priority** - Users see content first
✅ **Cleaner header** - No unnecessary container
✅ **Consistent styling** - All sections use same container style
✅ **Better flow** - Read response → see metrics → analyze differences
✅ **Matches wireframe** - Follows design exactly

---

## Updated Loading Skeleton

Also updated the loading skeleton to match new structure:
- Question header (no container)
- Region tabs skeleton
- Metrics section skeleton (after response)

---

## Files Updated

- `portal/src/pages/ModelRegionDiffPage.jsx` - Restructured layout

---

## Test It

Visit any model URL:
```
http://localhost:5173/results/test-job-123/model/qwen2.5-1.5b/question/what-happened-at-tiananmen-square
```

You'll see:
1. Clean question header (no box)
2. Region tabs immediately below
3. Response viewer (priority)
4. Metrics section below response (in standard container)
5. Rest of analysis follows

**Layout now matches wireframe perfectly!** ✅
