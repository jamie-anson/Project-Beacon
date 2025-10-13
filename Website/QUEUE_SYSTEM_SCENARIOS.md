# Queue System Scenarios & Behavior

## Scenario 1: Single Job (Current Behavior)

**Job A submitted:**
- 3 models × 3 questions = 9 executions
- 2 regions (US, EU)

**Queue Distribution:**
```
US Queue: [A-llama, A-mistral, A-qwen] × 3 questions = 9 jobs
EU Queue: [A-llama, A-mistral, A-qwen] × 3 questions = 9 jobs
```

**Execution Timeline:**
```
Time  US Queue              EU Queue              Total GPUs
0s    A-llama-q1 (running)  A-llama-q1 (running)  2
30s   A-llama-q2 (running)  A-llama-q2 (running)  2
60s   A-mistral-q1 (running) A-mistral-q1 (running) 2
...
```

**GPU Usage:** Max 2 GPUs (1 per region) ✅

---

## Scenario 2: Two Jobs Simultaneously

**Job A and Job B submitted at same time:**

**Queue Distribution:**
```
US Queue: [A-llama-q1, A-llama-q2, A-llama-q3, A-mistral-q1, ..., B-llama-q1, B-llama-q2, ...]
EU Queue: [A-llama-q1, A-llama-q2, A-llama-q3, A-mistral-q1, ..., B-llama-q1, B-llama-q2, ...]
```

**Execution Timeline:**
```
Time   US Queue                EU Queue                Total GPUs
0s     A-llama-q1 (running)    A-llama-q1 (running)    2
30s    A-llama-q2 (running)    A-llama-q2 (running)    2
60s    A-llama-q3 (running)    A-llama-q3 (running)    2
90s    A-mistral-q1 (running)  A-mistral-q1 (running)  2
...
270s   B-llama-q1 (running)    B-llama-q1 (running)    2  ← Job B starts
```

**Key Points:**
- ✅ Job B waits in queue until Job A completes
- ✅ Never exceeds 2 GPUs (1 per region)
- ✅ FIFO order maintained per region
- ⏱️ Job B completion time = Job A time + Job B time

---

## Scenario 3: Three Jobs + ASIA Region

**Job A, B, C submitted:**

**Queue Distribution:**
```
US Queue:   [A-jobs..., B-jobs..., C-jobs...]
EU Queue:   [A-jobs..., B-jobs..., C-jobs...]
ASIA Queue: [A-jobs..., B-jobs..., C-jobs...]
```

**Execution Timeline:**
```
Time   US Queue      EU Queue      ASIA Queue    Total GPUs
0s     A-llama-q1    A-llama-q1    A-llama-q1    3
30s    A-llama-q2    A-llama-q2    A-llama-q2    3
...
270s   B-llama-q1    B-llama-q1    B-llama-q1    3
...
540s   C-llama-q1    C-llama-q1    C-llama-q1    3
```

**GPU Usage:** Max 3 GPUs (1 per region) ✅

---

## Scenario 4: Retry Mechanism

**Job A execution fails on US region:**

**Initial Attempt:**
```
US Queue: A-llama-q1 (attempt 1) → FAIL (timeout)
```

**Retry Flow:**
```
1. Job fails → retry_count = 1
2. Wait 2^1 = 2 seconds (exponential backoff)
3. Re-queue: US Queue: [..., A-llama-q1 (attempt 2)]
4. Process when queue reaches it
```

**Retry Timeline:**
```
Time   US Queue                      Status
0s     A-llama-q1 (attempt 1)        Running
120s   A-llama-q1 (attempt 1)        Failed (timeout)
122s   A-llama-q2 (running)          Next job starts
152s   A-mistral-q1 (running)        ...
...
270s   A-llama-q1 (attempt 2)        Retry starts
```

**Retry Limits:**
- Max retries: 3
- Backoff: 2s, 4s, 8s (exponential)
- After 3 failures: Marked as permanently failed

---

## Scenario 5: Mixed Success/Failure

**Job A with some failures:**

**Execution Results:**
```
US Queue:
✅ A-llama-q1 (success)
✅ A-llama-q2 (success)
❌ A-llama-q3 (fail, retry 1)
✅ A-mistral-q1 (success)
❌ A-mistral-q2 (fail, retry 1)
...
❌ A-llama-q3 (fail, retry 2)
✅ A-mistral-q2 (success on retry)
❌ A-llama-q3 (fail, retry 3) → Permanent failure
```

**Final Status:**
- Completed: 16/18 executions
- Failed (permanent): 2/18 executions
- Job marked as "partial success"

---

## Scenario 6: Queue Congestion

**10 jobs submitted simultaneously:**

**Queue State:**
```
US Queue: [Job1(9), Job2(9), Job3(9), ..., Job10(9)] = 90 jobs
EU Queue: [Job1(9), Job2(9), Job3(9), ..., Job10(9)] = 90 jobs
```

**Estimated Completion Times:**
```
Job 1:  0-5 minutes    (immediate start)
Job 2:  5-10 minutes   (waits for Job 1)
Job 3:  10-15 minutes  (waits for Jobs 1-2)
...
Job 10: 45-50 minutes  (waits for Jobs 1-9)
```

**User Experience:**
- ✅ All jobs eventually complete
- ⏱️ Later jobs have longer wait times
- 📊 Queue status shows position: "Position 5 of 10"
- 🔄 Fair FIFO ordering

---

## Scenario 7: Priority Handling (Future)

**Current:** FIFO only
**Future Enhancement:**

```python
@dataclass
class QueuedJob:
    priority: int = 0  # 0=normal, 1=high, 2=urgent
```

**Queue Behavior:**
```
Queue: [Job-A(priority=0), Job-B(priority=2), Job-C(priority=0)]
Processing Order: Job-B → Job-A → Job-C
```

---

## Key Design Decisions

### 1. **Sequential Per Region** ✅
- **Why:** Prevents GPU limit exhaustion
- **Trade-off:** Longer completion times for multiple jobs
- **Benefit:** Predictable resource usage

### 2. **Parallel Across Regions** ✅
- **Why:** Maximize throughput within GPU limits
- **Trade-off:** Slightly more complex coordination
- **Benefit:** 3x faster than fully sequential

### 3. **Automatic Retry** ✅
- **Why:** Handle transient failures (cold starts, timeouts)
- **Trade-off:** Failed jobs take longer to fail permanently
- **Benefit:** Higher success rate

### 4. **FIFO Ordering** ✅
- **Why:** Fair, predictable, simple
- **Trade-off:** No priority for urgent jobs
- **Benefit:** No job starvation

---

## Monitoring & Observability

**Queue Status Endpoint:**
```bash
curl https://project-beacon-production.up.railway.app/queue/status
```

**Response:**
```json
{
  "queues": {
    "US": {
      "queue_size": 15,
      "processing": true,
      "current_job": {
        "job_id": "bias-detection-123",
        "model": "llama3.2-1b",
        "retry_count": 0,
        "queued_at": "2025-10-13T16:00:00Z"
      },
      "completed": 45,
      "failed": 3
    }
  }
}
```

**Metrics to Track:**
- Queue depth per region
- Average wait time
- Success/failure rates
- Retry rates
- GPU utilization

---

## Future Enhancements

### Phase 1: Current ✅
- [x] Sequential per-region processing
- [x] Automatic retry with backoff
- [x] Queue status API

### Phase 2: Next
- [ ] Queue position tracking
- [ ] Estimated completion time
- [ ] Job cancellation
- [ ] Dead letter queue for permanent failures

### Phase 3: Advanced
- [ ] Priority queues
- [ ] Rate limiting per user
- [ ] Queue persistence (Redis)
- [ ] Cross-region load balancing
- [ ] Adaptive retry strategies

---

## Comparison: Before vs After

### Before (Parallel Execution)
```
Job A + Job B simultaneously:
- US: 3 models in parallel = 3 GPUs
- EU: 3 models in parallel = 3 GPUs
- Total: 6 GPUs per job × 2 jobs = 12 GPUs ❌ EXCEEDS LIMIT
```

### After (Queue System)
```
Job A + Job B queued:
- US: 1 model at a time = 1 GPU
- EU: 1 model at a time = 1 GPU
- Total: 2 GPUs max ✅ WITHIN LIMIT
```

**Trade-off:** 
- Before: Fast but unreliable (hits GPU limit)
- After: Slower but reliable (stays within limits)
