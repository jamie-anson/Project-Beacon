# Region Queue System for GPU Resource Management

## Problem
Running multiple jobs simultaneously caused Modal GPU limit exhaustion (10 GPU free tier):
- Each job executes 3 models × 2 regions = 6 GPUs peak
- 2 jobs simultaneously = 12 GPUs → **exceeds limit** ❌

## Solution: Per-Region Queue System

### Architecture
```
Job Submission
      ↓
Hybrid Router
      ↓
Region Queues (Sequential Processing)
├─ US Queue   → Modal US   (max 3 GPUs)
├─ EU Queue   → Modal EU   (max 3 GPUs)  
└─ ASIA Queue → Modal ASIA (max 3 GPUs)
```

### Key Features

**1. Sequential Processing Per Region**
- Each region processes one job at a time
- Maximum 3 GPUs per region (3 models)
- Total system: max 9 GPUs (3 regions × 3 models)

**2. Parallel Regions**
- US, EU, and ASIA queues run in parallel
- Each queue is independent
- No cross-region GPU contention

**3. Queue Management**
- FIFO (First In, First Out) per region
- Automatic retry on failure
- Queue status monitoring

### Components

#### 1. `RegionQueue` Class
- Manages single region's job queue
- Tracks current job, completed count, failed count
- Sequential processing with asyncio.Queue

#### 2. `RegionQueueManager` Class
- Manages all 3 region queues
- Starts worker tasks for each region
- Provides status endpoints

#### 3. Queue API Endpoints
- `GET /queue/status` - All queues status
- `GET /queue/status/{region}` - Specific region status

### Usage

**Queue Status Response:**
```json
{
  "queues": {
    "US": {
      "region": "US",
      "queue_size": 2,
      "processing": true,
      "current_job": {
        "job_id": "bias-detection-123",
        "model": "mistral-7b",
        "region": "US",
        "queued_at": "2025-10-13T16:00:00Z",
        "started_at": "2025-10-13T16:01:00Z"
      },
      "completed": 15,
      "failed": 2
    },
    "EU": {...},
    "ASIA": {...}
  }
}
```

### Benefits

✅ **GPU Limit Compliance**
- Never exceeds 9 GPUs (3 per region)
- Safe for Modal free tier (10 GPU limit)

✅ **Fair Resource Allocation**
- FIFO ensures fair job processing
- No job starvation

✅ **Better Observability**
- Real-time queue status
- Per-region metrics
- Job tracking

✅ **Graceful Degradation**
- If one region fails, others continue
- Failed jobs tracked separately

### Implementation Status

**Phase 1: Core Queue System** ✅
- [x] RegionQueue class
- [x] RegionQueueManager class
- [x] Queue worker tasks
- [x] Status endpoints

**Phase 2: Integration** (Next)
- [ ] Modify inference endpoint to use queues
- [ ] Update runner app to handle queued responses
- [ ] Add queue position tracking

**Phase 3: Advanced Features** (Future)
- [ ] Priority queues
- [ ] Job cancellation
- [ ] Queue persistence (Redis)
- [ ] Rate limiting per user

### Files Created

- `hybrid_router/core/region_queue.py` - Queue implementation
- `hybrid_router/api/queue.py` - Queue API endpoints
- `hybrid_router/main.py` - Updated to start queue workers

### Testing

**Check Queue Status:**
```bash
curl https://project-beacon-production.up.railway.app/queue/status | jq
```

**Check Specific Region:**
```bash
curl https://project-beacon-production.up.railway.app/queue/status/US | jq
```

### Migration Path

**Current (Immediate Fix):**
- Runner executes regions sequentially
- Prevents parallel region execution
- Max 3 GPUs at a time

**Future (Queue System):**
- Router manages queues
- Runner submits to queues
- Better resource management

### Notes

- Queue system is **additive** - doesn't break existing functionality
- Can be enabled gradually per region
- Compatible with current runner implementation
- Provides foundation for future scaling
