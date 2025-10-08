# Live Progress UI Fixes - Processing Animation & Retry Button

## Issues Fixed

### 1. Missing Processing Animation in Progress Bar
**Problem**: Progress bar was completely static with no visual indication when jobs were running/processing.

**Solution**: Added animated shimmer effect to progress bars when status is 'running' or 'processing':
- Added `animate-pulse` class to the progress bar fill
- Added shimmer overlay with gradient animation
- Configured Tailwind with shimmer keyframes

**Files Modified**:
- `portal/src/components/bias-detection/progress/RegionRow.jsx` (lines 82-93)
- `portal/tailwind.config.js` (added animation & keyframes)

**Visual Effect**:
- Progress bar now pulses when running
- Shimmer animation sweeps across the bar
- Clear visual feedback that job is actively processing

### 2. Missing Job-Level Retry Button
**Problem**: Retry functionality only existed for individual questions (hidden in expanded view), no way to retry entire failed jobs.

**Solution**: Added prominent "Retry Job" button that appears when jobs fail:
- Button appears in ProgressActions component when job fails or times out
- Yellow color scheme with rotating arrows icon
- Positioned before the Refresh button for visibility
- Reloads page to allow user to resubmit

**Files Modified**:
- `portal/src/components/bias-detection/progress/ProgressActions.jsx` (added retry button)
- `portal/src/components/bias-detection/LiveProgressTable.jsx` (passed failure state & retry handler)

**User Flow**:
1. Job fails or times out
2. "Retry Job" button appears prominently
3. User clicks to reload page
4. User can resubmit the job with same or different parameters

## Technical Details

### Processing Animation Implementation
```jsx
<div className="flex-1 h-2 bg-gray-700 rounded overflow-hidden min-w-[40px] relative">
  <div 
    className={`h-full bg-green-500 transition-all duration-300 ${
      status === 'running' || status === 'processing' 
        ? 'animate-pulse' 
        : ''
    }`} 
    style={{ width: `${progress.percentage}%` }} 
  />
  {(status === 'running' || status === 'processing') && (
    <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/20 to-transparent animate-shimmer" />
  )}
</div>
```

### Retry Button Implementation
```jsx
{isFailed && onRetryJob && (
  <button 
    onClick={onRetryJob} 
    className="px-3 py-1.5 bg-yellow-600 text-white rounded text-sm hover:bg-yellow-700 flex items-center gap-2"
  >
    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
    </svg>
    Retry Job
  </button>
)}
```

## Test Coverage

### Unit Tests Created

#### RegionRow Processing Animation Tests (10 tests)
- ✅ Shows pulse animation when status is running
- ✅ Shows pulse animation when status is processing
- ✅ Shows shimmer overlay when status is running
- ✅ Shows shimmer overlay when status is processing
- ✅ Does NOT show pulse animation when completed
- ✅ Does NOT show shimmer overlay when completed
- ✅ Does NOT show animations when failed
- ✅ Has relative positioning on progress container
- ✅ Has absolute positioning on shimmer overlay
- ✅ Proper CSS classes applied

**File**: `portal/src/components/bias-detection/progress/__tests__/RegionRow.test.jsx`

#### ProgressActions Retry Button Tests (11 tests)
- ✅ Renders retry button when job failed
- ✅ Does NOT render retry button when job succeeded
- ✅ Does NOT render retry button when job is in progress
- ✅ Does NOT render retry button when onRetryJob not provided
- ✅ Calls onRetryJob when retry button clicked
- ✅ Styles retry button with yellow color scheme
- ✅ Displays rotating arrows icon in retry button
- ✅ Positions retry button before refresh button
- ✅ Handles isFailed=undefined gracefully
- ✅ Shows retry button for timeout failures
- ✅ Maintains functionality after state changes

**File**: `portal/src/components/bias-detection/progress/__tests__/ProgressActions.test.jsx`

### Storybook Stories Created

#### RegionRow Stories (11 scenarios)
- Processing with animation
- Processing status
- Completed (no animation)
- Failed (no animation)
- Timeout (no animation)
- Mixed progress with running animation
- Pending
- Refreshing
- Expanded view
- Single model running
- All models running (maximum animation)

**File**: `portal/src/components/bias-detection/progress/RegionRow.stories.jsx`

#### ProgressActions Stories (10 scenarios)
- Job in progress (no retry button)
- Job failed (shows retry button)
- Job completed successfully (no retry button)
- Job timeout (shows retry button)
- No job ID (minimal buttons)
- Failed without retry handler (no retry button)
- All buttons visible
- Long job ID (test overflow)
- Mobile view simulation
- Interactive demo (state transitions)

**File**: `portal/src/components/bias-detection/progress/ProgressActions.stories.jsx`

## Testing Checklist

### Automated Tests
- [x] Unit tests for processing animation (10 tests)
- [x] Unit tests for retry button (11 tests)
- [x] Storybook visual tests (21 scenarios)

### Manual Testing
- [ ] Verify shimmer animation appears on running jobs
- [ ] Verify shimmer stops when job completes
- [ ] Verify retry button appears on failed jobs
- [ ] Verify retry button appears on timeout jobs
- [ ] Verify retry button does NOT appear on successful jobs
- [ ] Verify retry button reloads page correctly
- [ ] Test on different browsers (Chrome, Firefox, Safari)
- [ ] Test on mobile devices

### Run Tests
```bash
# Run unit tests
npm test -- RegionRow.test.jsx
npm test -- ProgressActions.test.jsx

# Run Storybook for visual testing
npm run storybook
```

## Status
✅ **Implementation Complete** - Ready for testing and deployment
✅ **Test Suite Complete** - 21 unit tests + 21 Storybook scenarios
