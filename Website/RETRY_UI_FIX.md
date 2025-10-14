gpus# Retry Mechanism UI Fix

## Problem
The retry mechanism was not appearing in the UI after execution failures. Users saw greyed out "Answer" text instead of a functional "Retry" button.

## Root Cause
The API endpoint `/api/v1/jobs/{id}/executions` was **not querying or returning retry tracking columns** from the database:
- `retry_count`
- `max_retries`
- `last_retry_at`
- `retry_history`
- `original_error`

These columns exist in the database (migration `0010_add_retry_tracking.up.sql`) but were not being included in the SQL SELECT statement or the response JSON.

## Solution

### Backend Changes (`runner-app/internal/api/executions_handler.go`)

1. **Updated SQL Query** (lines 55-83)
   - Added retry tracking columns to SELECT statement:
     ```sql
     COALESCE(e.retry_count, 0) AS retry_count,
     COALESCE(e.max_retries, 3) AS max_retries,
     e.last_retry_at,
     COALESCE(e.retry_history, '[]'::jsonb) AS retry_history,
     e.original_error
     ```

2. **Updated Response Struct** (lines 85-113)
   - Added retry fields to `Exec` struct:
     ```go
     RetryCount    int             `json:"retry_count"`
     MaxRetries    int             `json:"max_retries"`
     LastRetryAt   *string         `json:"last_retry_at,omitempty"`
     RetryHistory  json.RawMessage `json:"retry_history,omitempty"`
     OriginalError *string         `json:"original_error,omitempty"`
     ```

3. **Updated Row Scanning** (lines 115-149)
   - Added proper scanning and formatting for retry fields
   - Handle nullable fields (`last_retry_at`, `original_error`)
   - Include retry history JSON if available

### Frontend Changes (`Website/portal/src/components/bias-detection/RegionRow.jsx`)

1. **Retry State Detection** (lines 24-28)
   - Extract `retry_count` and `max_retries` from execution data
   - Determine if retry is available: `canRetry = failed/timeout/error && retry_count < max_retries`
   - Detect exhausted retries: `retriesExhausted = failed/timeout/error && retry_count >= max_retries`

2. **Retry Handler** (lines 30-52)
   - POST request to `/api/v1/executions/{executionId}/retry`
   - Reload page on success to show updated status
   - Show error alert on failure

3. **UI Updates**
   - **Status Display** (lines 62-71): Show retry count badge next to status (e.g., "Failed (Retry 1/3)")
   - **Action Button** (lines 74-99):
     - ✅ **Completed**: Show "Answer" link (green)
     - 🔄 **Failed + Can Retry**: Show "Retry" button (yellow)
     - ❌ **Max Retries Reached**: Show "Max retries reached" (red)
     - ⏳ **Pending**: Show greyed out "Answer" text

## Visual Changes

### Before
```
Region          Status    View Result
United States   Failed    Answer (greyed out)
Europe          Complete  Answer (clickable)
```

### After
```
Region          Status              View Result
United States   Failed (Retry 1/3)  Retry (clickable, yellow)
Europe          Complete            Answer (clickable, green)
```

### After Max Retries
```
Region          Status              View Result
United States   Failed (Retry 3/3)  Max retries reached (red)
Europe          Complete            Answer (clickable, green)
```

## Testing Checklist

- [ ] Backend: Verify retry fields appear in `/api/v1/jobs/{id}/executions` response
- [ ] Frontend: Failed executions show "Retry" button instead of greyed "Answer"
- [ ] Frontend: Retry count badge appears next to status (e.g., "Retry 1/3")
- [ ] Frontend: Clicking "Retry" triggers POST to `/api/v1/executions/{id}/retry`
- [ ] Frontend: After max retries, show "Max retries reached" message
- [ ] Frontend: Completed executions still show "Answer" link
- [ ] Frontend: Pending executions show greyed "Answer" text

## Deployment Notes

1. **Database**: Migration `0010_add_retry_tracking.up.sql` must be applied
2. **Backend**: Deploy updated `executions_handler.go` 
3. **Frontend**: Deploy updated `RegionRow.jsx`
4. **API Compatibility**: Backward compatible - retry fields default to 0/3/null if not set

## Related Files

- `/Users/Jammie/Desktop/Project Beacon/runner-app/internal/api/executions_handler.go`
- `/Users/Jammie/Desktop/Project Beacon/Website/portal/src/components/bias-detection/RegionRow.jsx`
- `/Users/Jammie/Desktop/Project Beacon/runner-app/migrations/0010_add_retry_tracking.up.sql`
- `/Users/Jammie/Desktop/Project Beacon/runner-app/internal/queue/retry_handler.go`

## Status
✅ Backend changes complete and building successfully
✅ Frontend changes complete
⏳ Ready for deployment and testing
