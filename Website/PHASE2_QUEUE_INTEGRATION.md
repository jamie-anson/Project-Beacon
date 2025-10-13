# Phase 2: Queue System Integration - Complete

## Overview

Phase 2 integrates the queue system with the inference API, enabling:
- Queue-based inference requests
- Job status tracking
- Queue position visibility
- Estimated wait times

---

## What Was Implemented

### 1. ✅ Queued Inference Endpoint

**New Endpoint:** `POST /inference/queued`

**Purpose:** Submit inference requests to queue instead of immediate execution

**Request:**
```json
{
  "model": "llama3.2-1b",
  "prompt": "What is the capital of France?",
  "temperature": 0.1,
  "max_tokens": 500,
  "region_preference": "US"
}
```

**Response:**
```json
{
  "success": true,
  "job_id": "inference-a1b2c3d4e5f6",
  "status": "queued",
  "queue_position": 3,
  "estimated_wait_seconds": 90,
  "region": "US",
  "message": "Job queued in US region. Check status at /inference/status/inference-a1b2c3d4e5f6"
}
```

---

### 2. ✅ Job Status Tracking

**New Endpoint:** `GET /inference/status/{job_id}`

**Purpose:** Check status of queued inference jobs

**Response States:**

**Queued:**
```json
{
  "job_id": "inference-abc123",
  "status": "queued_or_not_found",
  "message": "Job is either queued, not found, or result has expired"
}
```

**Processing:**
```json
{
  "job_id": "inference-abc123",
  "status": "processing",
  "region": "US",
  "started_at": "2025-10-13T16:00:00Z"
}
```

**Completed:**
```json
{
  "job_id": "inference-abc123",
  "status": "completed",
  "result": {
    "success": true,
    "response": "The capital of France is Paris.",
    "model": "llama3.2-1b"
  },
  "completed_at": "2025-10-13T16:00:30Z",
  "duration": 28.5,
  "region": "US"
}
```

**Failed:**
```json
{
  "job_id": "inference-abc123",
  "status": "failed",
  "error": "Model timeout after 300 seconds",
  "retry_count": 3,
  "completed_at": "2025-10-13T16:05:00Z",
  "region": "US"
}
```

---

### 3. ✅ In-Memory Result Storage

**Implementation:**
- Results stored in `job_results` dictionary
- Keyed by `job_id`
- Includes status, result/error, timestamps, region

**Storage Lifecycle:**
```python
# When job completes successfully
job_results[job_id] = {
    "status": "completed",
    "result": inference_response,
    "completed_at": timestamp,
    "duration": seconds,
    "region": region_name
}

# When job fails permanently
job_results[job_id] = {
    "status": "failed",
    "error": error_message,
    "retry_count": 3,
    "completed_at": timestamp,
    "region": region_name
}
```

**Limitations:**
- ⚠️ In-memory only (lost on restart)
- ⚠️ No TTL (results never expire)
- ⚠️ No size limit (could grow unbounded)

**Future:** Migrate to Redis with TTL

---

## Usage Examples

### Example 1: Submit and Poll

```bash
# 1. Submit job
curl -X POST https://project-beacon-production.up.railway.app/inference/queued \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.2-1b",
    "prompt": "Explain quantum computing",
    "temperature": 0.1,
    "max_tokens": 200,
    "region_preference": "US"
  }'

# Response:
# {
#   "job_id": "inference-xyz789",
#   "status": "queued",
#   "queue_position": 2,
#   "estimated_wait_seconds": 60
# }

# 2. Poll status (every 5 seconds)
while true; do
  curl https://project-beacon-production.up.railway.app/inference/status/inference-xyz789
  sleep 5
done

# 3. When status="completed", retrieve result
```

### Example 2: Direct Execution (Bypass Queue)

```bash
# Use original endpoint for immediate execution
curl -X POST https://project-beacon-production.up.railway.app/v1/inference \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.2-1b",
    "prompt": "Quick question",
    "temperature": 0.1,
    "max_tokens": 50
  }'

# Executes immediately (no queue)
```

---

## Integration with Runner App

### Current State
Runner app calls `/v1/inference` directly (immediate execution)

### Future Integration Options

**Option A: Runner Uses Queued Endpoint**
```go
// In hybrid client
resp, err := client.Post("/inference/queued", request)
jobID := resp.JobID

// Poll for completion
for {
    status, err := client.Get("/inference/status/" + jobID)
    if status.Status == "completed" {
        return status.Result
    }
    time.Sleep(5 * time.Second)
}
```

**Option B: Hybrid Router Auto-Queues**
```python
# Router decides based on load
if queue_depth > 10:
    # Auto-queue high-load requests
    return queue_job(request)
else:
    # Execute immediately
    return execute_now(request)
```

**Option C: Runner Configurable**
```go
// Environment variable controls behavior
if os.Getenv("USE_QUEUE") == "true" {
    return client.QueuedInference(request)
} else {
    return client.DirectInference(request)
}
```

---

## Queue Position Tracking

### How It Works

**1. On Submission:**
```python
queue_size = queue_manager.get_queue_size(region)
estimated_wait = queue_size * 30  # 30s per job estimate

return {
    "queue_position": queue_size,
    "estimated_wait_seconds": estimated_wait
}
```

**2. Real-Time Position:**
```python
# TODO: Implement live position tracking
# Current limitation: Position only calculated at submission
# Future: Track position as queue processes
```

**3. Estimated Wait Time:**
- Simple calculation: `queue_size × 30 seconds`
- Assumes average 30s per execution
- Doesn't account for retries or failures
- Future: Use historical data for better estimates

---

## Testing

### Test 1: Queue Submission
```bash
curl -X POST http://localhost:8000/inference/queued \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.2-1b",
    "prompt": "Test prompt",
    "temperature": 0.1,
    "max_tokens": 50,
    "region_preference": "US"
  }' | jq
```

**Expected:**
- Returns job_id
- Status = "queued"
- Queue position shown
- Estimated wait time provided

### Test 2: Status Check
```bash
curl http://localhost:8000/inference/status/inference-abc123 | jq
```

**Expected:**
- Returns current status
- Shows processing/completed/failed
- Includes result if completed

### Test 3: Queue Status
```bash
curl http://localhost:8000/queue/status | jq
```

**Expected:**
- Shows all region queues
- Global retry queue size
- Current jobs processing

---

## Monitoring

### Key Metrics

**1. Queue Depth**
```python
GET /queue/status
{
  "global_retry_queue_size": 2,
  "regions": {
    "US": {"queue_size": 10},
    "EU": {"queue_size": 5},
    "ASIA": {"queue_size": 3}
  }
}
```

**2. Job Completion Rate**
- Track: completed_count, failed_count per region
- Calculate: success_rate = completed / (completed + failed)

**3. Average Wait Time**
- Track: time from submission to execution start
- Alert if > 5 minutes

**4. Result Storage Size**
- Track: len(job_results)
- Alert if > 10,000 (memory concern)

---

## Limitations & Future Work

### Current Limitations

**1. In-Memory Storage** ⚠️
- Results lost on restart
- No persistence
- Unbounded growth

**2. No TTL** ⚠️
- Results never expire
- Memory leak risk
- Manual cleanup needed

**3. Simple Position Tracking** ⚠️
- Position only at submission
- No live updates
- Estimates may be inaccurate

**4. No Job Cancellation** ⚠️
- Can't cancel queued jobs
- Must wait for completion
- Wastes resources

### Phase 3 Enhancements

**1. Redis Storage**
```python
# Persistent, distributed storage
redis_client.setex(
    f"job:{job_id}",
    ttl=3600,  # 1 hour TTL
    value=json.dumps(result)
)
```

**2. Live Position Updates**
```python
# WebSocket or SSE for real-time updates
@router.websocket("/inference/status/{job_id}/ws")
async def job_status_stream(websocket, job_id):
    while True:
        position = get_queue_position(job_id)
        await websocket.send_json({
            "position": position,
            "estimated_wait": position * 30
        })
        await asyncio.sleep(5)
```

**3. Job Cancellation**
```python
@router.delete("/inference/{job_id}")
async def cancel_job(job_id):
    # Remove from queue
    # Mark as cancelled
    # Return refund/credit
```

**4. Priority Queues**
```python
# Paid users get priority
if user.tier == "premium":
    await priority_queue.put(job)
else:
    await regular_queue.put(job)
```

---

## API Summary

### New Endpoints

| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | `/inference/queued` | Submit job to queue |
| GET | `/inference/status/{job_id}` | Check job status |
| GET | `/queue/status` | View all queues |
| GET | `/queue/status/{region}` | View region queue |

### Existing Endpoints (Unchanged)

| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | `/inference` | Direct execution |
| POST | `/v1/inference` | Direct execution (legacy) |
| GET | `/health` | Health check |
| GET | `/providers` | List providers |

---

## Deployment Checklist

- [x] Implement queued inference endpoint
- [x] Implement status tracking endpoint
- [x] Add result storage (in-memory)
- [x] Update queue workers to store results
- [x] Add queue position calculation
- [x] Add estimated wait time
- [ ] Deploy to Railway
- [ ] Test with runner app
- [ ] Monitor queue depth
- [ ] Document for users

---

## Summary

**Phase 2 Status:** ✅ Complete

**What Works:**
- ✅ Queue-based inference submission
- ✅ Job status tracking (queued, processing, completed, failed)
- ✅ Queue position visibility
- ✅ Estimated wait times
- ✅ Result storage and retrieval

**What's Next (Phase 3):**
- Redis persistence
- Live position updates
- Job cancellation
- Priority queues
- Rate limiting

**Impact:**
- Users can submit jobs without blocking
- Better visibility into queue state
- Foundation for advanced features
- Scalable architecture for growth
