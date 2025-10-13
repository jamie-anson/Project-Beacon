# Queue Retry Priority System

## Problem: Retry Starvation

**Without Priority:**
```
Initial Queue: [A-q1, A-q2, A-q3, B-q1, B-q2, B-q3]

0s    A-q1 fails → re-queued to END
      Queue: [A-q2, A-q3, B-q1, B-q2, B-q3, A-q1(retry)]
      
      A-q1 retry waits behind ALL of Job B! ❌
```

**Issues:**
- ❌ Job A's retry gets stuck behind unrelated Job B
- ❌ Unfair delay (152+ seconds)
- ❌ Poor user experience
- ❌ Job A appears slower than it should be

---

## Solution: Two-Tier Queue System

### Architecture

```
┌─────────────────────────────────────┐
│         Region Queue                │
├─────────────────────────────────────┤
│  Priority Queue (Retries)           │
│  [A-q1(retry), C-q2(retry)]         │ ← Checked FIRST
├─────────────────────────────────────┤
│  Regular Queue (New Jobs)           │
│  [A-q2, A-q3, B-q1, B-q2, B-q3]     │ ← Checked SECOND
└─────────────────────────────────────┘
```

### Processing Logic

```python
while True:
    # 1. Check retry queue first (priority)
    if retry_queue.not_empty():
        job = retry_queue.get()  # Process retry immediately
    else:
        job = regular_queue.get()  # Process new job
    
    # 2. Execute job
    try:
        execute(job)
    except:
        # 3. Failed? Re-queue to PRIORITY queue
        retry_queue.put(job)  # Jumps ahead of regular queue
```

---

## Example: With Priority System

**Scenario:**
```
Initial State:
Regular Queue: [A-q1, A-q2, A-q3, B-q1, B-q2, B-q3]
Retry Queue:   []

Time  Event
0s    A-q1 starts
30s   A-q1 fails → retry_count=1
32s   A-q1 added to RETRY QUEUE (after 2s backoff)
      Regular Queue: [A-q2, A-q3, B-q1, B-q2, B-q3]
      Retry Queue:   [A-q1(retry)]
      
32s   A-q1(retry) starts IMMEDIATELY ← Priority! ✅
      (Skips ahead of A-q2, A-q3, B-q1, etc.)
```

**Timeline Comparison:**

| Time | Without Priority | With Priority |
|------|------------------|---------------|
| 0s   | A-q1 starts      | A-q1 starts   |
| 30s  | A-q1 fails       | A-q1 fails    |
| 32s  | A-q2 starts      | **A-q1 retry starts** ✅ |
| 62s  | A-q3 starts      | A-q2 starts   |
| 92s  | B-q1 starts      | A-q3 starts   |
| 122s | B-q2 starts      | B-q1 starts   |
| 152s | B-q3 starts      | B-q2 starts   |
| 182s | **A-q1 retry** ❌ | B-q3 starts   |

**Improvement:** 182s → 32s (150 second reduction!) 🚀

---

## Benefits

### 1. **Fairness** ✅
- Retries don't get stuck behind unrelated jobs
- Job A's retry doesn't wait for Job B to complete
- Each job's retries are processed promptly

### 2. **Faster Recovery** ✅
- Failed jobs retry immediately after backoff
- No waiting for other jobs to finish
- Transient failures resolved quickly

### 3. **Better UX** ✅
- Users see faster retry attempts
- Jobs complete sooner
- More predictable timing

### 4. **Prevents Cascading Delays** ✅
- One job's failures don't delay others
- Queue congestion doesn't amplify retry delays
- System remains responsive

---

## Edge Cases

### Case 1: Multiple Retries

```
Regular Queue: [A-q2, A-q3, B-q1, B-q2]
Retry Queue:   [A-q1(retry), C-q5(retry)]

Processing Order:
1. A-q1(retry)  ← First retry
2. C-q5(retry)  ← Second retry
3. A-q2         ← Regular queue
4. A-q3
5. B-q1
...
```

**Behavior:** Retries processed in FIFO order, but ALL retries before regular jobs.

---

### Case 2: Retry Fails Again

```
Time  Event
0s    A-q1(retry) starts from priority queue
30s   A-q1(retry) fails AGAIN → retry_count=2
34s   A-q1(retry2) added to priority queue (4s backoff)
34s   A-q1(retry2) starts IMMEDIATELY
```

**Behavior:** Subsequent retries also get priority.

---

### Case 3: Max Retries Exhausted

```
Time  Event
0s    A-q1(retry3) starts from priority queue
30s   A-q1(retry3) fails → retry_count=3 (max reached)
      A-q1 marked as PERMANENTLY FAILED
      NOT re-queued
```

**Behavior:** After max retries, job is removed from queue entirely.

---

## Implementation Details

### Queue Structure

```python
class RegionQueue:
    def __init__(self):
        self.queue = asyncio.Queue()        # Regular jobs
        self.retry_queue = asyncio.Queue()  # Priority retries
        
    async def enqueue(self, job, is_retry=False):
        if is_retry:
            await self.retry_queue.put(job)  # Priority
        else:
            await self.queue.put(job)        # Regular
```

### Processing Loop

```python
async def process_queue(self):
    while True:
        # Priority: Check retry queue first
        if not self.retry_queue.empty():
            job = await self.retry_queue.get()
            logger.info("Processing RETRY job (priority)")
        else:
            job = await self.queue.get()
            logger.info("Processing regular job")
        
        try:
            await execute(job)
        except Exception as e:
            if job.retry_count < max_retries:
                # Re-queue to PRIORITY queue
                await self.enqueue(job, is_retry=True)
```

---

## Monitoring

**Queue Status Response:**
```json
{
  "region": "US",
  "queue_size": 10,
  "retry_queue_size": 2,  ← New field
  "processing": true,
  "current_job": {
    "job_id": "A-q1",
    "retry_count": 1,
    "is_retry": true
  }
}
```

**Metrics to Track:**
- Retry queue depth
- Average retry wait time
- Retry success rate
- Time saved by priority system

---

## Alternative Approaches Considered

### 1. **Insert at Front of Regular Queue** ❌
```
Queue: [A-q1(retry), A-q2, A-q3, B-q1, ...]
```
**Problem:** Mixes retries with regular jobs, hard to track

### 2. **Separate Queue Per Job** ❌
```
Job-A Queue: [A-q1, A-q2, A-q3]
Job-B Queue: [B-q1, B-q2, B-q3]
```
**Problem:** Complex coordination, unfair resource allocation

### 3. **Two-Tier System** ✅ (Chosen)
```
Retry Queue:   [retries...]
Regular Queue: [new jobs...]
```
**Benefits:** Simple, fair, efficient

---

## Future Enhancements

### Priority Levels
```python
class QueuedJob:
    priority: int = 0  # 0=normal, 1=retry, 2=urgent
```

### Adaptive Backoff
```python
# Longer backoff if queue is congested
backoff = min(2 ** retry_count * (1 + queue_size/10), 60)
```

### Retry Budget
```python
# Limit total retries per time window
if retries_last_minute > 100:
    use_longer_backoff()
```

---

## Summary

**Problem:** Retries getting stuck behind unrelated jobs
**Solution:** Two-tier queue system (priority + regular)
**Result:** 5x faster retry processing, better fairness, improved UX

✅ Retries processed immediately after backoff
✅ No interference between different jobs
✅ Fair resource allocation
✅ Better user experience
