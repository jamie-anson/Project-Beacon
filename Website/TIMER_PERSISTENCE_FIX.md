# Live Progress Timer Persistence Fix

## Problem
The "Time Remaining" countdown in the Live Progress bar was resetting to 60 seconds whenever users navigated away from the Bias Detection page and returned. This created a poor UX where the timer didn't accurately reflect the actual job runtime.

## Root Cause
The `jobStartTime` was stored in React component state (`useState`) in the `useJobProgress` hook. When the component unmounted (user navigated away) and remounted (user returned), the state was reset and `Date.now()` was used as the new start time, causing the countdown to restart.

## Solution
Implemented localStorage persistence for the job start time:

### Changes Made

1. **`useJobProgress.js`** - Added localStorage persistence:
   - Initialize state from localStorage on mount
   - Persist to localStorage whenever job ID changes
   - Graceful error handling for localStorage failures

2. **`BiasDetection.jsx`** - Added cleanup:
   - Clear localStorage entry when user dismisses completed job

3. **`useBiasDetection.js`** - Added cleanup:
   - Clear localStorage entry when resetting Live Progress state (wallet changes, etc.)

4. **Tests** - Added comprehensive test coverage:
   - Test localStorage persistence on job start
   - Test restoration from localStorage on mount
   - Test localStorage update when job ID changes

## How It Works

1. **Job Start**: When a new job starts, `useJobProgress` creates a `jobStartTime` object with `{ jobId, startTime }` and saves it to `localStorage` under key `beacon:job_start_time`

2. **Navigation Away**: User navigates to another page, component unmounts, but localStorage persists

3. **Navigation Back**: User returns to Bias Detection page, component mounts, `useJobProgress` reads from localStorage and restores the original start time

4. **Timer Calculation**: The countdown timer uses the persisted start time, so it shows the correct remaining time based on when the job actually started

5. **Cleanup**: When job completes or user dismisses it, localStorage entry is cleared to prevent stale data

## Benefits

- ✅ Timer persists across page navigation
- ✅ Accurate time remaining display
- ✅ Better user experience
- ✅ No breaking changes to existing functionality
- ✅ Graceful fallback if localStorage is unavailable
- ✅ Comprehensive test coverage

## Testing

Run the portal tests:
```bash
cd portal
npm test -- useJobProgress.test.js
```

All 21 tests should pass, including the 3 new localStorage persistence tests.

## User Impact

Users can now:
- Navigate away from the Bias Detection page while a job is running
- Return to the page and see the accurate time remaining
- Trust that the timer reflects the actual job runtime, not when they last viewed the page
