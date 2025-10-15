# Stuck Jobs Analysis - Where Jobs Can Get Stuck

## Overview
Jobs can get stuck in various states throughout the execution pipeline. This document identifies all accumulation points and mitigation strategies.

---

## 1. Database Level - Job Status States

### **Stuck in "created" Status**
**Location:** `jobs` table, `status = 'created'`

**How it happens:**
- Job inserted into database
- Never picked up by worker
- Worker crashed before processing
- Queue message lost

**Detection:**
```sql
SELECT jobspec_id, created_at, NOW() - created_at as age
FROM jobs 
WHERE status = 'created' 
AND created_at < NOW() - INTERVAL '5 minutes'
ORDER BY created_at ASC;
```

**Mitigation:** 
- `internal/service/job_repair.go` - `RepairStuckJobs()`
- Resets jobs older than 5 minutes to "created"
- Runs periodically via cron/scheduler

---

### **Stuck in "processing" Status**
**Location:** `jobs` table, `status = 'processing'`

**How it happens:**
- Worker picked up job, set status to "processing"
- Worker crashed mid-execution
- Context cancelled before completion
- Database write failed for final status

**Detection:**
```sql
SELECT jobspec_id, updated_at, NOW() - updated_at as age
FROM jobs 
WHERE status = 'processing' 
AND updated_at < NOW() - INTERVAL '10 minutes'
ORDER BY updated_at ASC;
```

**Mitigation:**
- `internal/service/job_recovery.go` - `RecoverStaleJobs()`
- Resets jobs stuck in "processing" for >10 minutes
- Resets to "created" for retry

---

### **Stuck in "running" Status**
**Location:** `jobs` table, `status = 'running'`

**How it happens:**
- Job marked as "running" but never completes
- All executions complete but job status never updated
- Worker crashed after executions but before final status update
- `processExecutionResults()` failed

**Detection:**
```sql
-- Running jobs with no started_at
SELECT jobspec_id, created_at
FROM jobs 
WHERE status = 'running' 
AND started_at IS NULL;

-- Running jobs over 30 minutes old
SELECT jobspec_id, started_at, NOW() - started_at as age
FROM jobs 
WHERE status = 'running' 
AND started_at < NOW() - INTERVAL '30 minutes';
```

**Mitigation:**
- `internal/service/job_repair.go` - Checks for stale "running" jobs
- `internal/service/job_timeout.go` - Times out long-running jobs
- Admin API: `/admin/jobs/{id}/cancel` - Manual cancellation

---

### **Stuck in "queued" Status**
**Location:** `jobs` table, `status = 'queued'`

**How it happens:**
- Job created but never transitioned to "processing"
- Worker not consuming from queue
- Redis queue connection lost
- Message stuck in dead-letter queue

**Detection:**
```sql
SELECT jobspec_id, created_at, NOW() - created_at as age
FROM jobs 
WHERE status = 'queued' 
AND created_at < NOW() - INTERVAL '5 minutes'
ORDER BY created_at ASC;
```

**Mitigation:**
- Queue monitoring
- Dead-letter queue processing
- Worker health checks

---

## 2. Queue Level - Redis Accumulation

### **Jobs Queue**
**Location:** Redis key `jobs` (or `constants.JobsQueueName`)

**How it happens:**
- Messages pushed but never consumed
- Worker not running
- Worker crashed
- Consumer lag

**Detection:**
```bash
# Check queue length
redis-cli LLEN jobs

# Peek at oldest message
redis-cli LINDEX jobs -1
```

**Mitigation:**
- Worker auto-restart
- Queue length monitoring/alerting
- Multiple worker instances for redundancy

---

### **Dead-Letter Queue**
**Location:** Redis key `jobs:dead`

**How it happens:**
- Job failed max retry attempts
- Permanent failure (validation error, etc.)
- Moved from main queue after retries exhausted

**Detection:**
```bash
# Check dead-letter queue length
redis-cli LLEN jobs:dead

# View dead jobs
redis-cli LRANGE jobs:dead 0 -1
```

**Mitigation:**
- `internal/queue/redis_queue.go` - Retry logic with exponential backoff
- Manual inspection and requeue if needed
- Alerting on dead-letter queue growth

---

## 3. Execution Level - Incomplete Executions

### **Executions Never Written**
**Location:** Missing from `executions` table

**How it happens:**
- Goroutine cancelled before DB write
- Context timeout during execution
- Database connection lost
- `InsertExecutionWithModelAndQuestion()` failed silently

**Detection:**
```sql
-- Jobs with fewer executions than expected
SELECT 
    j.jobspec_id,
    j.status,
    COUNT(e.id) as actual_executions,
    -- Expected: num_regions * num_models * num_questions
    (SELECT COUNT(*) FROM jsonb_array_elements(j.jobspec_data->'constraints'->'regions')) * 
    (SELECT COUNT(*) FROM jsonb_array_elements(j.jobspec_data->'metadata'->'models')) *
    (SELECT COUNT(*) FROM jsonb_array_elements(j.jobspec_data->'questions')) as expected_executions
FROM jobs j
LEFT JOIN executions e ON e.job_id = j.id
WHERE j.status IN ('completed', 'failed')
GROUP BY j.id, j.jobspec_id, j.jobspec_data
HAVING COUNT(e.id) < (
    (SELECT COUNT(*) FROM jsonb_array_elements(j.jobspec_data->'constraints'->'regions')) * 
    (SELECT COUNT(*) FROM jsonb_array_elements(j.jobspec_data->'metadata'->'models')) *
    (SELECT COUNT(*) FROM jsonb_array_elements(j.jobspec_data->'questions'))
);
```

**Mitigation:**
- Proper error handling in `executeMultiRegion()`
- Ensure goroutines complete before job marked done
- Use `sync.WaitGroup` correctly
- Log execution write failures

---

### **Executions Stuck in "running" Status**
**Location:** `executions` table, `status = 'running'`

**How it happens:**
- Execution started but never updated to final status
- Modal/provider returned success but status update failed
- Context cancelled before status update

**Detection:**
```sql
SELECT 
    e.id,
    e.job_id,
    e.region,
    e.model_id,
    e.status,
    e.started_at,
    NOW() - e.started_at as age
FROM executions e
WHERE e.status = 'running'
AND e.started_at < NOW() - INTERVAL '15 minutes'
ORDER BY e.started_at ASC;
```

**Mitigation:**
- Execution timeout logic
- Status update retry
- Admin cleanup script

---

## 4. Worker Level - Process Failures

### **Worker Crashes**
**How it happens:**
- OOM (out of memory)
- Panic/unhandled error
- Container restart
- Deployment

**Impact:**
- Jobs in "processing" get stuck
- Queue messages not acknowledged
- Executions incomplete

**Mitigation:**
- Graceful shutdown handling
- Context cancellation propagation
- Job status reset on worker restart
- Health checks and auto-restart

---

### **Context Cancellation**
**Location:** `internal/worker/job_runner.go` - `jobCtx` with timeout

**How it happens:**
```go
jobCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
defer cancel()
```
- Job exceeds 15-minute timeout
- Parent context cancelled
- Goroutines don't respect context

**Impact:**
- Job marked as "failed" or stuck in "processing"
- Partial executions written
- Some regions complete, others don't

**Mitigation:**
- Increase timeout for multi-region jobs
- Ensure all goroutines check context
- Proper cleanup on cancellation

---

## 5. Portal Level - Display Issues

### **Stale Data Display**
**Location:** Portal UI showing old job status

**How it happens:**
- Polling lag (15s interval)
- Caching issues
- Job status updated but executions not yet written
- Race condition between job completion and execution writes

**Impact:**
- User sees "running" when job is done
- Missing executions shown as errors
- Confusing UX

**Mitigation:**
- Faster polling during execution (2-5s) ✅ Already implemented
- WebSocket real-time updates (future)
- Better "pending" vs "missing" messaging ✅ Fixed

---

## 6. Network Level - External Service Failures

### **Modal/Provider Timeouts**
**How it happens:**
- Modal cold start >150s
- Network timeout
- Provider unavailable

**Impact:**
- Execution marked as "failed"
- Job might fail if success rate not met
- Retry logic may exhaust attempts

**Mitigation:**
- Retry with exponential backoff
- Multiple provider fallback
- Increase timeout for cold starts

---

### **Database Connection Loss**
**How it happens:**
- Railway/Fly.io network issue
- PostgreSQL restart
- Connection pool exhausted

**Impact:**
- Job status updates fail
- Execution writes fail
- Jobs stuck in intermediate states

**Mitigation:**
- Connection retry logic
- Transaction rollback
- Health checks and reconnection

---

## Summary of Stuck Job States

| State | Location | Detection Query | Mitigation |
|-------|----------|----------------|------------|
| **created** | `jobs` table | `status='created' AND age>5min` | `job_repair.go` |
| **processing** | `jobs` table | `status='processing' AND age>10min` | `job_recovery.go` |
| **running** | `jobs` table | `status='running' AND age>30min` | `job_timeout.go` |
| **queued** | `jobs` table | `status='queued' AND age>5min` | Queue monitoring |
| **Queue backlog** | Redis `jobs` | `LLEN jobs` | Worker scaling |
| **Dead-letter** | Redis `jobs:dead` | `LLEN jobs:dead` | Manual review |
| **Missing executions** | `executions` table | Count mismatch | Goroutine fixes |
| **Stuck executions** | `executions` table | `status='running' AND age>15min` | Cleanup script |

---

## Recommended Actions

### **Immediate:**
1. ✅ Implement stuck job detection queries
2. ✅ Add monitoring/alerting for each accumulation point
3. ✅ Create admin cleanup scripts

### **Short-Term:**
4. 🔄 Add Sentry alerts for stuck job patterns
5. 🔄 Dashboard showing stuck job metrics
6. 🔄 Automated cleanup cron jobs

### **Long-Term:**
7. 🔌 Distributed tracing for job lifecycle
8. 📊 Job execution analytics
9. 🔧 Self-healing job recovery system
