# Global Retry Queue - Cross-Region Priority System

## Problem: Slow Retry Recovery

**User Perspective:**
> "I don't want to wait 10 minutes for my job to complete. If a question fails, I want it retried IMMEDIATELY, not stuck behind other jobs."

**Previous System:**
```
US Queue: [A-q1, A-q2, A-q3, B-q1, B-q2, B-q3]

A-q1 fails on US → re-queued to US retry queue
US Retry Queue: [A-q1(retry)]

But US is busy processing A-q2, A-q3, B-q1...
A-q1 retry waits 5+ minutes ❌
```

---

## Solution: Global Retry Queue with Cross-Region Failover

### Architecture

```
┌─────────────────────────────────────────────────────┐
│         GLOBAL RETRY QUEUE (Highest Priority)       │
│  [A-q1(retry-US), B-q2(retry-EU), C-q3(retry-ASIA)] │
│                                                      │
│  ANY region can pick up ANY retry                   │
│  Fastest possible recovery                          │
└─────────────────────────────────────────────────────┘
         ↓ Priority 1 (checked first by all workers)
┌─────────────────────────────────────────────────────┐
│  US Queue          EU Queue          ASIA Queue     │
│  [A-q2, A-q3]      [A-q2, A-q3]      [A-q2, A-q3]   │
└─────────────────────────────────────────────────────┘
         ↓ Priority 2 (checked if global queue empty)
```

### Processing Priority

**Each region worker checks in order:**
1. **Global Retry Queue** (highest priority, cross-region)
2. **Region Retry Queue** (region-specific retries)
3. **Region Regular Queue** (new jobs)

---

## Example: Fast Recovery

### Scenario: US Fails, EU Picks Up Retry

```
Time  Event
0s    US worker: A-q1 starts
30s   US worker: A-q1 fails (timeout)
32s   A-q1 added to GLOBAL retry queue (after 2s backoff)
      
      Global Retry Queue: [A-q1(retry)]
      US Queue: [A-q2, A-q3, B-q1] (busy)
      EU Queue: [A-q2] (idle)
      
32s   EU worker: Checks global queue → finds A-q1(retry)
32s   EU worker: Starts A-q1(retry) ← Cross-region retry! ✅
62s   EU worker: A-q1(retry) completes successfully
```

**Result:** 
- Retry completed in 32 seconds (not 5+ minutes)
- EU picked up US's failed job (cross-region failover)
- User sees fast recovery

---

## Benefits

### 1. **Instant Retry** ⚡
- Retries don't wait for original region to be free
- ANY idle region can pick up the retry
- Fastest possible recovery time

### 2. **Cross-Region Failover** 🌍
- If US is congested, EU can handle US retries
- Automatic load balancing
- Better resource utilization

### 3. **Fair to Users** ✅
- Retries don't get stuck behind new jobs
- Failed executions get immediate attention
- Better user experience

### 4. **Reduced Wait Time** ⏱️
```
Before: Retry waits 5-10 minutes (stuck in region queue)
After:  Retry starts in 2-30 seconds (global priority)

Improvement: 10x-150x faster retry! 🚀
```

---

## Detailed Example: 3 Regions, Multiple Failures

### Initial State
```
US Queue:   [A-q1, A-q2, A-q3, B-q1, B-q2]
EU Queue:   [A-q1, A-q2, A-q3]
ASIA Queue: [A-q1, A-q2]
```

### Timeline

```
Time  US Worker         EU Worker         ASIA Worker       Global Retry Queue
0s    A-q1 starts       A-q1 starts       A-q1 starts       []
30s   A-q1 fails        A-q1 completes    A-q1 fails        
32s   (processing A-q2) A-q2 starts       (processing A-q2) [A-q1(US), A-q1(ASIA)]
      
32s   (A-q2 running)    A-q2 completes    (A-q2 running)    [A-q1(US), A-q1(ASIA)]
34s   (A-q2 running)    Checks global →   (A-q2 running)    [A-q1(ASIA)]
                        A-q1(US) starts! ✅
                        
64s   A-q2 completes    A-q1(US) done ✅  A-q2 completes    [A-q1(ASIA)]
64s   Checks global →                     Checks global →   []
      A-q1(ASIA) starts! ✅               (queue empty)
      
94s   A-q1(ASIA) done ✅                  A-q3 starts       []
```

**Result:**
- US's failed A-q1 was retried by EU (32s after failure)
- ASIA's failed A-q1 was retried by US (64s after failure)
- Both retries completed quickly without waiting for original region

---

## Edge Cases

### Case 1: All Regions Busy

```
Global Retry Queue: [A-q1(retry), B-q2(retry), C-q3(retry)]
US Queue:   [processing D-q1]
EU Queue:   [processing E-q1]
ASIA Queue: [processing F-q1]

Behavior:
- Retries wait in global queue
- First region to finish picks up first retry
- Still faster than region-specific retry queue
```

### Case 2: Retry Fails Again (Cross-Region)

```
0s    A-q1 fails on US → Global retry queue
2s    EU picks up A-q1(retry) from global queue
32s   A-q1(retry) fails on EU → Back to global queue
36s   ASIA picks up A-q1(retry2) from global queue
66s   A-q1(retry2) succeeds on ASIA ✅

Result: Job tried on 3 different regions, succeeded on 3rd attempt
```

### Case 3: Region-Specific Issue

```
US has network issue (all US jobs failing)

0s    A-q1 fails on US → Global retry queue
2s    EU picks up A-q1(retry) → Succeeds ✅
      
Result: Automatic failover to healthy region
```

---

## Monitoring

### Queue Status API Response

```json
{
  "global_retry_queue_size": 3,
  "regions": {
    "US": {
      "queue_size": 10,
      "retry_queue_size": 0,
      "processing": true,
      "current_job": {
        "job_id": "A-q2",
        "retry_count": 0
      }
    },
    "EU": {
      "queue_size": 5,
      "retry_queue_size": 0,
      "processing": true,
      "current_job": {
        "job_id": "A-q1",
        "retry_count": 1,
        "original_region": "US"
      }
    }
  }
}
```

### Metrics to Track

1. **Cross-Region Retry Rate**
   - % of retries picked up by different region
   - Indicates load balancing effectiveness

2. **Retry Wait Time**
   - Time from failure to retry start
   - Target: < 30 seconds

3. **Retry Success by Region**
   - Which regions have best retry success rate
   - Identify problematic regions

4. **Global Queue Depth**
   - Number of retries waiting
   - Alert if > 10 (indicates system-wide issues)

---

## Comparison: Before vs After

### Before (Region-Specific Retry)

```
US Queue: [A-q1, A-q2, A-q3, B-q1, B-q2, B-q3]

0s    A-q1 fails
2s    A-q1 added to US retry queue
      US Retry Queue: [A-q1(retry)]
      US Regular Queue: [A-q2, A-q3, B-q1, B-q2, B-q3]
      
2s    A-q2 starts (A-q1 retry waits)
32s   A-q3 starts (A-q1 retry waits)
62s   B-q1 starts (A-q1 retry waits)
92s   B-q2 starts (A-q1 retry waits)
122s  B-q3 starts (A-q1 retry waits)
152s  A-q1(retry) FINALLY starts ❌

Wait time: 150 seconds
```

### After (Global Retry Queue)

```
US Queue: [A-q1, A-q2, A-q3, B-q1, B-q2, B-q3]
EU Queue: [A-q2] (idle)

0s    A-q1 fails on US
2s    A-q1 added to GLOBAL retry queue
      Global Retry Queue: [A-q1(retry)]
      
2s    EU checks global queue → finds A-q1(retry)
2s    EU starts A-q1(retry) ✅

Wait time: 2 seconds
Improvement: 75x faster! 🚀
```

---

## Implementation Details

### Queue Structure

```python
class RegionQueueManager:
    def __init__(self):
        # Global retry queue (highest priority)
        self.global_retry_queue = asyncio.Queue()
        
        # Region-specific queues
        self.queues = {
            "US": RegionQueue("US"),
            "EU": RegionQueue("EU"),
            "ASIA": RegionQueue("ASIA"),
        }
```

### Worker Processing Loop

```python
async def process_queue(self, region: str):
    while True:
        # Priority 1: Global retry queue
        if not self.global_retry_queue.empty():
            job = await self.global_retry_queue.get()
            logger.info(f"[{region}] Processing GLOBAL retry")
            
        # Priority 2: Region retry queue
        elif not queue.retry_queue.empty():
            job = await queue.retry_queue.get()
            
        # Priority 3: Regular queue
        else:
            job = await queue.queue.get()
        
        # Execute job
        try:
            await execute(job)
        except Exception as e:
            # Failed? Add to GLOBAL retry queue
            await self.global_retry_queue.put(job)
```

---

## User Experience

### Before
```
User: Submits job
User: Sees "Question 1 failed"
User: Waits... (5-10 minutes)
User: "Why is it taking so long?" 😤
User: Sees "Question 1 completed (retry)"
```

### After
```
User: Submits job
User: Sees "Question 1 failed"
User: Waits... (2-30 seconds)
User: Sees "Question 1 completed (retry)" ✅
User: "Wow, that was fast!" 😊
```

---

## Future Enhancements

### 1. Smart Region Selection
```python
# Prefer regions with better success rate
if retry_count > 1:
    prefer_region = get_most_reliable_region()
```

### 2. Retry Budget
```python
# Limit retries per time window
if global_retries_per_minute > 100:
    apply_rate_limiting()
```

### 3. Predictive Retry
```python
# If region is having issues, proactively retry elsewhere
if region_failure_rate > 0.5:
    route_to_different_region()
```

---

## Summary

**Problem:** Retries waiting 5-10 minutes in region-specific queues
**Solution:** Global retry queue with cross-region failover
**Result:** 75x faster retry, better UX, automatic load balancing

✅ Retries start in 2-30 seconds (not 5-10 minutes)
✅ Cross-region failover (any region can handle any retry)
✅ Better resource utilization (idle regions help busy ones)
✅ Improved user experience (fast recovery from failures)

**User Impact:** "I don't want to wait 10 minutes" → Problem solved! ⚡
