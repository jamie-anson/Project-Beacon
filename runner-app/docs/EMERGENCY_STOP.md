# Emergency Stop Endpoint

**Endpoint**: `POST /admin/jobs/:id/emergency-stop`  
**Authentication**: Admin token required  
**Purpose**: Immediately stop a running job and all its executions

---

## Usage

### Basic Usage:

```bash
curl -X POST https://beacon-runner-change-me.fly.dev/admin/jobs/JOB_ID/emergency-stop \
  -H "X-Admin-Token: $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'
```

### With Environment Variable:

```bash
export ADMIN_TOKEN="your-admin-token"
export JOB_ID="bias-detection-1759185378516"

curl -X POST "https://beacon-runner-change-me.fly.dev/admin/jobs/${JOB_ID}/emergency-stop" \
  -H "X-Admin-Token: ${ADMIN_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{}'
```

---

## Response

### Success Response (Job Stopped):

```json
{
  "ok": true,
  "message": "Job emergency stopped",
  "job_id": "bias-detection-1759185378516",
  "previous_status": "processing",
  "new_status": "failed",
  "executions_stopped": 42,
  "stopped_at": "2025-09-29T23:48:00Z"
}
```

### Success Response (Already Stopped):

```json
{
  "ok": true,
  "message": "Job already in terminal state",
  "job_id": "bias-detection-1759185378516",
  "status": "failed",
  "action": "none"
}
```

### Error Response (Job Not Found):

```json
{
  "error": "job not found",
  "job_id": "invalid-job-id"
}
```

---

## What It Does

### 1. Updates Job Status:

- Changes job status from `processing`/`pending`/`created` to `failed`
- Adds metadata to job record:
  ```json
  {
    "emergency_stop": true,
    "stopped_at": "2025-09-29T23:48:00Z",
    "stopped_by": "admin"
  }
  ```

### 2. Stops All Running Executions:

- Finds all executions with status: `pending`, `running`, or `processing`
- Updates their status to `failed`
- Sets `completed_at` timestamp
- Adds error message to execution output:
  ```json
  {
    "error": "Emergency stop by admin",
    "emergency_stop": true
  }
  ```

### 3. Returns Summary:

- Previous job status
- New job status
- Number of executions stopped
- Timestamp of stop action

---

## Use Cases

### 1. Runaway Job (Duplication Bug):

**Scenario**: Job creating 100+ duplicate executions

```bash
# Stop the job immediately
curl -X POST "https://beacon-runner-change-me.fly.dev/admin/jobs/bias-detection-1759185378516/emergency-stop" \
  -H "X-Admin-Token: $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'
```

**Result**: All pending executions stopped, preventing more duplicates

---

### 2. Infinite Loop:

**Scenario**: Job stuck in processing state for hours

```bash
# Check job status first
curl -s "https://beacon-runner-change-me.fly.dev/api/v1/jobs/stuck-job-123" | jq '.status'

# Emergency stop
curl -X POST "https://beacon-runner-change-me.fly.dev/admin/jobs/stuck-job-123/emergency-stop" \
  -H "X-Admin-Token: $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'
```

---

### 3. Cost Control:

**Scenario**: Job consuming too many resources

```bash
# Stop expensive job
curl -X POST "https://beacon-runner-change-me.fly.dev/admin/jobs/expensive-job-456/emergency-stop" \
  -H "X-Admin-Token: $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'
```

---

## Safety Features

### 1. Idempotent:

- Safe to call multiple times
- Returns success if job already stopped
- No error if job already in terminal state

### 2. Atomic:

- Updates job and executions in single transaction
- Either all updates succeed or none do

### 3. Auditable:

- Adds metadata to job record
- Tracks who stopped the job (admin)
- Records timestamp of stop action

---

## Verification

### Check Job Was Stopped:

```bash
# Get job status
curl -s "https://beacon-runner-change-me.fly.dev/api/v1/jobs/JOB_ID" | jq '{id: .job.id, status: .status, metadata: .job.metadata}'
```

**Expected**:
```json
{
  "id": "bias-detection-1759185378516",
  "status": "failed",
  "metadata": {
    "emergency_stop": true,
    "stopped_at": "2025-09-29T23:48:00Z",
    "stopped_by": "admin"
  }
}
```

### Check Executions Were Stopped:

```bash
# Count stopped executions
curl -s "https://beacon-runner-change-me.fly.dev/api/v1/executions?job_id=JOB_ID" | \
  jq '.executions | group_by(.status) | map({status: .[0].status, count: length})'
```

**Expected**:
```json
[
  {"status": "completed", "count": 10},
  {"status": "failed", "count": 42}
]
```

---

## Comparison with Other Endpoints

| Endpoint | Purpose | When to Use |
|----------|---------|-------------|
| `/admin/jobs/:id/emergency-stop` | **Immediate stop** | Runaway jobs, duplicates, infinite loops |
| `/admin/timeout-stuck-jobs` | **Batch timeout** | Clean up old stuck jobs (15+ min) |
| `/admin/repair-stuck-jobs` | **Republish** | Retry failed jobs |

---

## Best Practices

### 1. Use for Emergencies Only:

- Don't use for normal job cancellation
- Reserve for critical situations
- Document reason for emergency stop

### 2. Verify Before Stopping:

```bash
# Check job status first
curl -s "https://beacon-runner-change-me.fly.dev/api/v1/jobs/JOB_ID" | jq '.status'

# Check execution count
curl -s "https://beacon-runner-change-me.fly.dev/api/v1/executions?job_id=JOB_ID" | jq '.executions | length'

# Then stop if needed
curl -X POST "https://beacon-runner-change-me.fly.dev/admin/jobs/JOB_ID/emergency-stop" ...
```

### 3. Monitor After Stopping:

```bash
# Verify job stopped
curl -s "https://beacon-runner-change-me.fly.dev/api/v1/jobs/JOB_ID" | jq '.status'

# Check no new executions created
curl -s "https://beacon-runner-change-me.fly.dev/api/v1/executions?job_id=JOB_ID" | jq '.executions | length'
```

---

## Troubleshooting

### Issue: Endpoint Returns 404

**Cause**: Endpoint not deployed yet

**Solution**: Wait for deployment to complete or redeploy:
```bash
flyctl deploy -a beacon-runner-change-me
```

### Issue: Endpoint Returns 403 Forbidden

**Cause**: Invalid or missing admin token

**Solution**: Check ADMIN_TOKEN environment variable:
```bash
echo $ADMIN_TOKEN
# Should output your admin token
```

### Issue: Job Still Running After Stop

**Cause**: Executions already started on workers

**Solution**: 
- Wait 1-2 minutes for workers to check job status
- Check execution status again
- If still running, contact support

---

## Implementation Details

### Database Updates:

```sql
-- Update job status
UPDATE jobs 
SET status = 'failed', 
    updated_at = NOW(),
    metadata = COALESCE(metadata, '{}'::jsonb) || 
               '{"emergency_stop": true, "stopped_at": "...", "stopped_by": "admin"}'::jsonb
WHERE id = $1;

-- Stop running executions
UPDATE executions 
SET status = 'failed',
    completed_at = NOW(),
    output = COALESCE(output, '{}'::jsonb) || 
             '{"error": "Emergency stop by admin", "emergency_stop": true}'::jsonb
WHERE job_id = $1 
AND status IN ('pending', 'running', 'processing');
```

### Code Location:

- **Handler**: `internal/handlers/admin.go` - `EmergencyStopJob()`
- **Route**: `internal/api/routes.go` - `POST /admin/jobs/:id/emergency-stop`

---

## Examples

### Example 1: Stop Duplication Bug Job

```bash
# The job that created 100+ duplicates
JOB_ID="bias-detection-1759185378516"

# Stop it
curl -X POST "https://beacon-runner-change-me.fly.dev/admin/jobs/${JOB_ID}/emergency-stop" \
  -H "X-Admin-Token: ${ADMIN_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{}'

# Response:
# {
#   "ok": true,
#   "message": "Job already in terminal state",
#   "job_id": "bias-detection-1759185378516",
#   "status": "failed",
#   "action": "none"
# }
```

### Example 2: Stop Processing Job

```bash
# Job stuck in processing
JOB_ID="stuck-job-123"

# Stop it
curl -X POST "https://beacon-runner-change-me.fly.dev/admin/jobs/${JOB_ID}/emergency-stop" \
  -H "X-Admin-Token: ${ADMIN_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{}'

# Response:
# {
#   "ok": true,
#   "message": "Job emergency stopped",
#   "job_id": "stuck-job-123",
#   "previous_status": "processing",
#   "new_status": "failed",
#   "executions_stopped": 15,
#   "stopped_at": "2025-09-29T23:48:00Z"
# }
```

---

## Security

### Authentication:

- Requires `X-Admin-Token` header
- Token must match `ADMIN_TOKEN` environment variable
- No other authentication methods accepted

### Authorization:

- Only admin role can access
- Rate limited to prevent abuse
- Logged for audit trail

### Audit Trail:

- All emergency stops logged
- Metadata includes who stopped job
- Timestamp recorded for compliance

---

**Status**: ✅ Implemented and deployed  
**Version**: 1.0.0  
**Last Updated**: 2025-09-29T23:48:00Z
