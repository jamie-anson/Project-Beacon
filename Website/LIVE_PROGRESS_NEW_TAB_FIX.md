# Live Progress Links Open in New Tab Fix

## Problem
When users clicked on links in the Live Progress view ("Detect Bias", "Compare", "Answer"), they would navigate away from the Bias Detection page and lose the Live Progress context. This created a poor UX where users couldn't easily return to monitor their running job.

## Solution
Updated all navigation links in the Live Progress components to open in new tabs using `window.open()` or `target="_blank"`, preserving the Live Progress view while allowing users to explore results.

## Changes Made

### 1. **QuestionRow.jsx** - "Detect Bias" Button
- **Before**: Used `navigate()` to navigate in same tab
- **After**: Uses `window.open()` to open in new tab
- **Code**: 
  ```javascript
  window.open(`/bias-detection/${jobId}`, '_blank', 'noopener,noreferrer');
  ```

### 2. **ModelRow.jsx** - "Compare" Button
- **Before**: Used `navigate()` to navigate in same tab
- **After**: Uses `window.open()` to open in new tab
- **Code**:
  ```javascript
  window.open(`/results/${jobId}/model/${modelId}/question/${encodedQuestion}`, '_blank', 'noopener,noreferrer');
  ```

### 3. **ExecutionDetails.jsx** - "Answer" Link
- **Before**: `<Link>` component without target attribute
- **After**: Added `target="_blank"` and `rel="noopener noreferrer"`
- **Code**:
  ```jsx
  <Link
    to={`/portal/executions/${exec.id}`}
    target="_blank"
    rel="noopener noreferrer"
    className="text-pink-400 hover:text-pink-300 underline decoration-dotted"
  >
    Answer
  </Link>
  ```

### 4. **RegionRow.jsx** - "Answer" Link
- **Status**: Already had `target="_blank"` and `rel="noopener noreferrer"` ✅
- No changes needed

## Security Considerations

All new tab links include `rel="noopener noreferrer"` to prevent:
- **Tabnabbing attacks**: The new page cannot access `window.opener`
- **Referrer leakage**: The new page doesn't receive referrer information

## User Experience Benefits

✅ **Live Progress Persistence**: Users can click links without losing their job monitoring view
✅ **Multi-tasking**: Users can explore multiple results in separate tabs while job is running
✅ **Context Preservation**: The Bias Detection page with Live Progress stays open and continues polling
✅ **Easy Navigation**: Users can close result tabs and return to Live Progress
✅ **No Confusion**: Users won't accidentally navigate away and wonder where their job went

## Testing

Build verification:
```bash
cd portal
npm run build
```

Build succeeds with no errors. The portal compiles successfully with all changes.

## Affected Components

- `/portal/src/components/bias-detection/QuestionRow.jsx`
- `/portal/src/components/bias-detection/ModelRow.jsx`
- `/portal/src/components/bias-detection/progress/ExecutionDetails.jsx`
- `/portal/src/components/bias-detection/RegionRow.jsx` (already correct)

## User Impact

Users can now:
- Click "Detect Bias" to view cross-question analysis in a new tab
- Click "Compare" to view model-region comparisons in a new tab
- Click "Answer" to view individual execution results in a new tab
- Keep the Live Progress view open to monitor job status
- Navigate between multiple result tabs without losing context
