# Queue Architecture - Project Beacon

## Overview
Project Beacon uses **two separate queue systems** that work together:
1. **Runner Queues** (Redis) - Job submission and processing
2. **Hybrid Router Queues** (In-memory) - GPU resource management and inference execution

## Runner Queues (Redis)

### Purpose
Manage job lifecycle from submission to execution coordination.

### 3 Queues

#### 1. Main Queue: `jobs`
- **Type**: Redis LIST (LPUSH/BRPOP)
- **Purpose**: Primary job processing queue
- **Flow**: Job submission → Outbox table → OutboxPublisher → `jobs` queue → JobRunner
- **Consumer**: JobRunner worker

#### 2. Retry Queue: `jobs:retry`
- **Type**: Redis SORTED SET (scored by retry timestamp)
- **Purpose**: Failed jobs waiting to be retried
- **Delay**: Exponential backoff (1min, 2min, 3min)
- **Max Retries**: 3 attempts
- **Consumer**: RetryHandler polls for jobs ready to retry

#### 3. Dead Letter Queue: `jobs:dead`
- **Type**: Redis LIST
- **Purpose**: Permanently failed jobs (max retries exhausted)
- **Admin Access**: `GET /admin/queue-dead`, `POST /admin/queue-dead/purge`

### Flow Diagram
```
Job Submission
  ↓
Postgres Outbox Table (transactional)
  ↓
OutboxPublisher → Redis "jobs" queue
  ↓
JobRunner processes job
  ↓
Success? → Done
  ↓
Failure? → Redis "jobs:retry" (sorted set with timestamp)
  ↓
RetryHandler polls → back to "jobs" queue
  ↓
Max retries? → Redis "jobs:dead" (permanent failure)
```

## Hybrid Router Queues (In-Memory)

### Purpose
Prevent GPU limit exhaustion on Modal (10 GPU concurrent limit) by processing jobs sequentially per region.

### 4 Queues

#### 1. Global Retry Queue (Highest Priority)
- **Type**: Python asyncio.Queue
- **Purpose**: Cross-region retries - ANY region can pick up the job
- **Priority**: Processed FIRST
- **Use Case**: When a job fails in one region, it goes here so the fastest available region can retry it
- **Max Retries**: 3 attempts with exponential backoff

#### 2. US Region Queue
- **Type**: Python asyncio.Queue
- **Purpose**: Jobs specifically for US region
- **Contains**:
  - Main queue: Regular US jobs
  - Region-specific retry queue: US-only retries

#### 3. EU Region Queue
- **Type**: Python asyncio.Queue
- **Purpose**: Jobs specifically for EU region
- **Contains**:
  - Main queue: Regular EU jobs
  - Region-specific retry queue: EU-only retries

#### 4. ASIA Region Queue
- **Type**: Python asyncio.Queue
- **Purpose**: Jobs specifically for ASIA region
- **Contains**:
  - Main queue: Regular ASIA jobs
  - Region-specific retry queue: ASIA-only retries

### Queue Priority Order

Each region worker processes jobs in this priority order:

```python
# Priority 1: Global retry queue (cross-region, highest priority)
if not self.global_retry_queue.empty():
    job = await self.global_retry_queue.get()

# Priority 2: Region-specific retry queue
elif not queue.retry_queue.empty():
    job = await queue.retry_queue.get()

# Priority 3: Regular region queue
else:
    job = await queue.queue.get()
```

### Flow Diagram
```
Inference Request arrives at Router
  ↓
Enqueue in region queue (US/EU/ASIA)
  ↓
Queue worker picks up job (Priority: global retry > region retry > regular)
  ↓
Execute inference on Modal GPU
  ↓
Success? → Return result
  ↓
Failure? → Increment retry_count
  ↓
retry_count < max_retries? → Global retry queue (with exponential backoff)
  ↓
retry_count >= max_retries? → Return permanent failure
```

## How the Two Systems Interact

### New Job Submission
```
1. User submits job via Portal
2. Runner API → Postgres jobs table + outbox table (transactional)
3. OutboxPublisher → Redis "jobs" queue
4. JobRunner dequeues job
5. JobRunner → HTTP POST to Hybrid Router /inference
6. Router → Enqueues in region queue (US/EU/ASIA)
7. Router queue worker → Executes on Modal GPU
8. Result → Returns to JobRunner
9. JobRunner → Updates execution record in Postgres
```

### User-Initiated Retry (UI Button)
```
1. User clicks "Retry" button in Portal
2. Portal → POST /api/v1/executions/{id}/retry-question
3. Runner API → RetryService.executeInference()
4. RetryService → HTTP POST to Hybrid Router /inference
5. Router → Enqueues in region queue (US/EU/ASIA)
6. Router queue worker → Executes on Modal GPU
7. If fails → Router's auto-retry (global_retry_queue)
8. Result → Returns to Runner
9. Runner → Updates execution record (increments retry_count)
```

### Automatic Retry (Router-Level)
```
1. Inference fails in Router queue worker
2. Router → Increments job.retry_count
3. retry_count < max_retries?
   → Sleep with exponential backoff (2^retry_count seconds, max 60s)
   → Enqueue in global_retry_queue (ANY region can pick it up)
4. retry_count >= max_retries?
   → Return permanent failure to Runner
```

## Key Differences

| Feature | Runner Queues | Hybrid Router Queues |
|---------|--------------|---------------------|
| **Storage** | Redis (persistent) | In-memory (ephemeral) |
| **Purpose** | Job lifecycle management | GPU resource control |
| **Scope** | Entire job (all questions/models/regions) | Single inference call |
| **Retry Logic** | Exponential backoff, dead letter queue | Exponential backoff, global retry queue |
| **Max Retries** | 3 attempts | 3 attempts |
| **Admin Access** | Yes (`/admin/queue-dead`) | Yes (`/queue/status`) |
| **Persistence** | Survives restarts | Lost on restart |

## Monitoring Endpoints

### Runner Queues
- `GET /admin/runtime/stats` - Queue sizes (main, retry, dead)
- `GET /admin/queue-dead` - View dead letter queue
- `POST /admin/queue-dead/purge` - Clear dead letter queue

### Hybrid Router Queues
- `GET /queue/status` - All region queue statuses
- `GET /queue/status/{region}` - Specific region queue status

## Summary

**Runner Queues**: Handle job submission, persistence, and coordination across the entire multi-question/multi-model/multi-region job lifecycle.

**Hybrid Router Queues**: Handle GPU resource management and inference execution, ensuring sequential processing per region to prevent exceeding Modal's 10 GPU limit.

Together, they provide:
- ✅ Reliable job processing with persistence
- ✅ Automatic retry with exponential backoff
- ✅ GPU resource control and optimization
- ✅ Cross-region failover for faster recovery
- ✅ Dead letter queue for permanently failed jobs
- ✅ Admin visibility and control
