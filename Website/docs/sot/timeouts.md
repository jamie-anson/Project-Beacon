# Timeout Configuration - Source of Truth

**Last Updated**: 2025-10-02 01:21 UTC  
**Status**: 🔴 UNDER INVESTIGATION - Multiple timeout issues discovered

---

## 🎯 Overview

Project Beacon has **multiple timeout layers** that can cause executions to fail. This document tracks ALL timeout configurations across the stack.

---

## 📊 Current Timeout Values

### Layer 1: Runner → Hybrid Router (HTTP Client)

**Location**: `/runner-app/internal/hybrid/client.go` (lines 28-29)

```go
// Default timeout
timeoutSec := 300  // 300 seconds (5 minutes)

// Overridable via environment variables (checked in order):
1. HYBRID_ROUTER_TIMEOUT  (primary)
2. HYBRID_TIMEOUT         (fallback)
```

**Current Configuration** (as of 2025-10-15):
- **Code Default**: 300 seconds (5 minutes) ✅ Updated from 120s
- **Fly.io Secret**: `HYBRID_ROUTER_TIMEOUT` = Not set (using code default)
- **Observed Behavior**: 
  - Historical: ~151 seconds (Railway timeout issue)
  - Current: 300 seconds (working as expected)

**Issue**: 
- ❌ Secret value unknown (can only see digest)
- ❌ Observed timeout (151s) doesn't match any expected value
- ❌ Attempted to set to 300s but didn't take effect

---

### Layer 2: Hybrid Router → Modal (HTTP Client)

**Location**: `/Website/hybrid_router/core/router.py` (line 21)

```python
self.client = httpx.AsyncClient(timeout=600.0)  # 10 minutes
```

**Current Configuration**:
- **Hardcoded**: 600 seconds (10 minutes)
- **Configurable**: No (hardcoded in Python)

**Status**: ✅ Should be sufficient (10 minutes > Modal execution time)

---

### Layer 3: Modal Function Timeouts

#### 3A. Modal Inference Functions (Multi-Region)

**Location**: `/Website/modal-deployment/modal_hf_multiregion.py`

```python
# US Region (lines 420-426)
@app.function(
    timeout=900,              # 15 minutes
    startup_timeout=1800,     # 30 minutes
    scaledown_window=600,     # 10 minutes
)

# EU Region (lines 444-450)
@app.function(
    timeout=900,              # 15 minutes
    startup_timeout=1800,     # 30 minutes
    scaledown_window=600,     # 10 minutes
)

# APAC Region (lines 468-474)
@app.function(
    timeout=900,              # 15 minutes
    startup_timeout=1800,     # 30 minutes
    scaledown_window=600,     # 10 minutes
)
```

**Current Configuration**:
- **Execution Timeout**: 900 seconds (15 minutes)
- **Startup Timeout**: 1800 seconds (30 minutes for first load)
- **Container Idle**: 600 seconds (10 minutes keep-warm)

**Status**: ✅ Should be sufficient for inference

---

#### 3B. Modal Web Endpoints

**Location**: `/Website/modal-deployment/modal_hf_multiregion.py` (line 547)

```python
@app.function(
    timeout=600,  # 10 minutes for web endpoint
)
@modal.fastapi_endpoint(method="POST")
def inference_endpoint(item: dict):
```

**Current Configuration**:
- **Web Endpoint Timeout**: 600 seconds (10 minutes)

**Status**: ✅ Should be sufficient

---

#### 3C. Modal Regional Deployments (Separate Files)

**US Region** (`modal_hf_us.py` line 315-320):
```python
timeout=900                  # 15 minutes
container_idle_timeout=120   # 2 minutes
startup_timeout=1800         # 30 minutes
```

**EU Region** (`modal_hf_eu.py` line 370-375):
```python
timeout=900                  # 15 minutes
container_idle_timeout=120   # 2 minutes
startup_timeout=1800         # 30 minutes
```

**APAC Region** (`modal_hf_apac.py` line 370-375):
```python
timeout=900                  # 15 minutes
container_idle_timeout=120   # 2 minutes
startup_timeout=1800         # 30 minutes
```

**Status**: ✅ Consistent across regions

---

### Layer 4: Job Timeout Monitor

**Location**: `/runner-app/cmd/runner/main.go` (line 165)

```go
timeoutThreshold := 15 * time.Minute  // Jobs processing for >15min are timed out
checkInterval := 2 * time.Minute      // Check every 2 minutes
```

**Current Configuration**:
- **Job Timeout**: 15 minutes (900 seconds)
- **Check Interval**: 2 minutes (120 seconds)

**Purpose**: Background worker that marks jobs stuck in "processing" status as "failed"

**Status**: ✅ Appropriate for multi-question jobs

---

## 🔍 The Mystery: Why 151 Seconds? **SOLVED!**

### Observed Behavior:
- APAC executions consistently timeout at **~150 seconds**
- EU executions complete at **~148 seconds** (just under timeout)
- US executions complete at **~45 seconds** (well under timeout)

### 🎯 ROOT CAUSE FOUND: Railway Platform Timeout

**Location**: Railway.app platform (hosting Hybrid Router)

**Evidence**:
- Job bias-detection-1759401587711:
  - ASIA Q1: Timeout at 10:42:19 (150s from 10:39:49)
  - EU Q1: Completed at 10:42:17 (148s from 10:39:49)
  - **2 second difference!**

**If this was a runner timeout, both would timeout at the same time.**

### Railway HTTP Timeout History:

**Historical Limits**:
- **Original**: 5 minutes (300 seconds) - documented in Railway docs
- **Updated (4 months ago)**: 15 minutes (900 seconds) - per Railway staff
- **Observed**: ~150 seconds (2.5 minutes) ❌

**Sources**:
- Railway Docs: https://docs.railway.app/reference/public-networking#technical-specifications
- Railway Staff Update: https://station.railway.com/feedback/increase-max-platform-timeout-beyond-5-m-9d15d4ee
  - "We have now increased our HTTP Timeout to 15 minutes!" - 4 months ago

### The Mystery Deepens:

**Railway should support 15 minutes (900s), but we're seeing 150s timeouts!**

Possible explanations:
1. **Old Railway proxy version** - deployment not on latest infrastructure
2. **Regional differences** - different timeout for different Railway regions
3. **Load balancer timeout** - separate timeout at Railway's edge
4. **Undocumented limit** - additional timeout layer not in docs

### Why This Explains the Pattern:

1. **Runner → Railway Router**: 301s timeout (we set this)
2. **Railway Proxy → Hybrid Router**: ??? (should be 900s, seeing 150s)
3. **Hybrid Router → Modal**: 600s timeout (httpx client)
4. **Modal execution**: Takes 150-200s for APAC
5. **Result**: Something at Railway kills connection at 150s

### Investigation Findings:

**Code Analysis**:
1. ✅ **Uvicorn Configuration**: No request timeout set in `main.py`
   - Only uses `timeout-keep-alive` (default 5s for idle connections)
   - No `--timeout` or similar flags
   
2. ✅ **Railway Configuration**: `railway.json` has no HTTP timeout setting
   - Only `healthcheckTimeout: 300` (for health checks, not requests)
   
3. ✅ **Hybrid Router Client**: `httpx.AsyncClient(timeout=600.0)` 
   - Router → Modal timeout is 600s (10 minutes)
   - This is NOT the bottleneck

4. ✅ **Dockerfile**: Standard Python 3.11-slim, no timeout configs

**Conclusion**: The 150s timeout is NOT coming from the application code!

### Where is the 150s Timeout Coming From?

**Hypothesis**: Railway's load balancer or proxy layer has a 150s timeout that:
1. Is not documented in their public docs
2. Is separate from the "15 minute" HTTP timeout they announced
3. May be a per-region or per-tier limitation

**Evidence Supporting This**:
- Application code has no 150s timeout configured
- Railway docs say 15 minutes (900s)
- We're seeing exactly 150s (2.5 minutes)
- The timeout is consistent across all APAC requests

### Next Steps to Investigate:

1. **Check Railway deployment details in UI**:
   - Deployment region
   - Service tier/plan
   - Any proxy or load balancer settings
   - Network configuration

2. **Contact Railway support** with specific question:
   - "Why are HTTP requests timing out at 150s when docs say 900s?"
   - Provide job ID and timestamps as evidence
   
3. **Test direct Modal connection** from runner:
   - Bypass Railway router entirely
   - Hit Modal APAC endpoint directly
   - Confirm if it completes successfully

4. **Deploy to Fly.io** (recommended):
   - Eliminates Railway as variable
   - Fly.io has configurable timeouts
   - Can set to 600s+ to match Modal

### Immediate Solutions:

**Option 1: Deploy Hybrid Router to Fly.io** (⚠️ NOT RECOMMENDED)
- ❌ **Fly.io has 60-second idle timeout** (even worse than Railway!)
- If no data sent/received for 60s, connection closes
- Can request custom timeout via support (paid plans)
- **This won't solve the problem without streaming responses**

**Fly.io Timeout Details**:
- **Default**: 60 seconds of idle time (no data sent/received)
- **Configurable**: Yes, but requires contacting support + paid plan
- **Workaround**: Send data every <60s to keep connection alive
- **Source**: https://community.fly.io/t/request-timeouts-on-fly-io/5653

**Comparison with Modal APAC Reality**:
```
Modal APAC Execution Timeline (from logs):
  Cold start wait:  367s (6m 7s) - waiting for container
  Cold start time:   10s (9.85s) - container starting  
  Execution time:   103s (1m 43s) - actual inference
  TOTAL:            480s (8 minutes!)

Railway timeout:    150s - fails at 2.5 minutes ❌
Fly.io timeout:      60s - fails at 1 minute ❌❌
```

**Fly.io runner would fail even FASTER than Railway!**
- Railway: Fails after 150s (during cold start wait)
- Fly.io: Fails after 60s (even earlier in cold start wait)
- **Neither platform can handle Modal APAC's 8-minute total time**

**Option 2: Keep-Alive Ping Strategy** (Creative workaround)
- Send periodic "ping" requests to keep connection alive
- Ping every 140s during long-running requests
- Prevents Railway from timing out idle connection
- **Pros**: No infrastructure changes needed
- **Cons**: Complex to implement, hacky solution

**Implementation approach**:
```python
# In Hybrid Router - wrap Modal requests with keep-alive
async def call_modal_with_keepalive(url, payload, timeout=600):
    # Start the actual request
    task = asyncio.create_task(httpx_client.post(url, json=payload))
    
    # Send keep-alive pings every 140s
    while not task.done():
        try:
            await asyncio.wait_for(asyncio.shield(task), timeout=140)
            break  # Request completed
        except asyncio.TimeoutError:
            # Send a lightweight ping to keep connection alive
            await httpx_client.get(f"{url}/health")
            continue
    
    return await task
```

**Concerns**:
- Railway might still timeout the original request
- Adds complexity and potential bugs
- Doesn't solve root cause
- May not work if Railway tracks per-request timeout (not per-connection)

**Option 3: Bypass Router for APAC**
- Use direct Modal APAC endpoint from runner
- Only for slow regions
- Keeps router for US/EU
- Quick fix but not ideal architecture

**Option 4: Optimize Modal APAC**
- Reduce cold start time
- Keep containers warm longer
- Get execution under 150s
- Doesn't solve root cause

---

## 🚫 APAC Region Status: TEMPORARILY DISABLED

**Date**: 2025-10-02  
**Status**: ❌ Disabled in production

### Reason:

Modal APAC has severe cold start issues that cause 100% failure rate:
- **Cold start wait**: 6+ minutes (waiting for container availability)
- **Cold start time**: ~10 seconds (container initialization)
- **Execution time**: ~2 minutes (actual inference)
- **Total time**: ~8 minutes (480 seconds)
- **Railway timeout**: 150 seconds (2.5 minutes)
- **Result**: All APAC requests timeout before Modal completes

### Cost Analysis for Fix:

To make APAC functional, we would need Modal keep-warm strategy:

**Option 1: `scaledown_window=600`** (Recommended)
- Cost: ~$87/month for APAC
- Keeps container warm for 10 minutes after last request
- Eliminates 6-minute cold start wait
- Total execution time: ~2 minutes (under Railway 150s limit)

**Option 2: `keep_warm=1`** (Too expensive)
- Cost: ~$630/month for APAC (24/7 warm container)
- Not cost-effective for current usage

### Current Production Regions:

- ✅ **US-EAST**: Working perfectly (completes in ~45s)
- ✅ **EU-WEST**: Working well (completes in ~148s, just under timeout)
- ❌ **ASIA-PACIFIC**: Disabled (would take 480s, times out at 150s)

### When Will APAC Be Re-enabled?

**Option 1**: When keep-warm strategy is implemented (~$87/month budget approved)  
**Option 2**: When Railway timeout is increased to 900s (requires Railway support)  
**Option 3**: When Modal improves APAC cold start times (unlikely)  
**Option 4**: When we migrate to a platform without strict timeout limits

### Implementation Details:

**Portal UI**: APAC option greyed out with explanation  
**Hybrid Router**: APAC endpoint commented out  
**Dashboard**: APAC shown as "Disabled" status  
**Documentation**: This section added to explain status

---

## 🎯 Recommended Timeout Values

Based on actual execution times from Modal logs:

### Execution Time Requirements:
- **Cold Start**: 30-45 seconds (worst case)
- **Llama 3.2-1B**: 30-40 seconds
- **Qwen 2.5-1.5B**: 35-45 seconds
- **Mistral 7B**: 150-160 seconds (slowest!)
- **Network Latency**: 5-10 seconds
- **Buffer**: 50-100 seconds (safety margin)

### Recommended Configuration:

```
Runner → Hybrid Router:  300 seconds (5 minutes)
  Rationale: 45s cold start + 160s Mistral + 95s buffer

Hybrid Router → Modal:   600 seconds (10 minutes) ✅ Already set
  Rationale: Double the runner timeout for safety

Modal Function Timeout:  900 seconds (15 minutes) ✅ Already set
  Rationale: Allows for unexpected delays

Job Monitor Timeout:     900 seconds (15 minutes) ✅ Already set
  Rationale: Multi-question jobs need time
```

---

## 🚨 Known Issues

### Issue 1: Runner HTTP Client Timeout
- **Status**: 🔴 BROKEN
- **Expected**: 300 seconds
- **Actual**: ~151 seconds
- **Impact**: False failures on EU/APAC regions
- **Fix Attempted**: Set `HYBRID_ROUTER_TIMEOUT=300` via Fly.io secrets
- **Result**: No change (still timing out at 151s)

### Issue 2: Secret Value Unknown
- **Status**: 🔴 BLOCKED
- **Problem**: Can't verify actual secret value
- **Digest**: `70f17482d5716502` (same before/after update)
- **Impact**: Can't confirm if secret update worked

### Issue 3: Context Cancellation Not Working
- **Status**: 🔴 BROKEN  
- **Expected**: Stop all regions when one fails
- **Actual**: Regions continue to Q2 after Q1 timeout
- **Impact**: Zombie requests sent to Modal
- **Related**: Phase 1B fix deployed but ineffective

---

## 📝 Investigation TODO

- [ ] **Determine actual HYBRID_ROUTER_TIMEOUT value**
  - Check app logs for HTTP client initialization
  - Add logging to hybrid client constructor
  - Verify environment variable is being read

- [ ] **Find source of 151-second timeout**
  - Not in code (120s default)
  - Not from attempted fix (300s)
  - Check Fly.io platform limits
  - Check Railway router timeout

- [ ] **Verify secret application**
  - Confirm secrets are loaded at startup
  - Check if machine restart needed
  - Verify environment variables in running container

- [ ] **Test timeout fix**
  - Deploy logging to show actual timeout value
  - Submit test job after verification
  - Monitor for 151s vs 300s timeout

---

## 📊 Test Results

### Job: bias-detection-1759363701760 (2025-10-02 00:08)

**US Region**: ✅ 6/6 completed
**EU Region**: ✅ 6/6 completed  
**APAC Region**: ❌ 6/6 failed (timeout)

**APAC Timeline**:
```
Q1 Started:   00:08:29
Q1 Timeout:   00:11:00  (151 seconds)
Q2 Started:   00:11:00  (sent anyway!)
Q2 Timeout:   00:14:27  (207 seconds)
```

**Evidence**:
- All APAC failures have `provider_id: ""` (no response received)
- Timeout at exactly 151 seconds (not 120s, not 300s)
- Context cancellation didn't prevent Q2 from being sent

---

## 🎉 Recent Updates (2025-10-15)

### Goroutine Coordination Bug - FIXED ✅

**Issue**: Jobs were being marked "completed" before all executions finished, causing:
- Portal showing "Job completed!" while executions still running
- EU executions completing 30-80 seconds AFTER job marked done
- Database writes happening after job status finalized

**Root Cause**: 
1. `StartedAt` timestamp captured BEFORE hybrid router execution (included queue time)
2. Barrier query counted all execution rows, including in-progress retries

**Fix Deployed**: `registry.fly.io/beacon-runner-production:deployment-01K7MAN7ZG53EGEPQ3JTJ2EMRY`
- Moved `StartedAt` capture to immediately before `executor.Execute()` call
- Updated barrier query to exclude `status IN ('retrying', 'pending', 'running')`
- Only counts executions with `completed_at IS NOT NULL`

**Validation**: Job `bias-detection-1760545271810` (job_id=434)
- Job marked `completed` at 16:26:47
- Last execution finished at 16:26:46
- **Gap: 1 second** (vs 83 seconds before fix) ✅

---

### New Mystery: 238-Second Timeout (2025-10-15)

**Observed**: Execution 2193 (qwen2.5-1.5b US) failed after 238 seconds
- **Started**: 16:21:12
- **Completed**: 16:25:11  
- **Duration**: 238 seconds (~4 minutes)
- **Modal logs**: Show successful execution in ~1m22s
- **Output data**: `null` (no response received by runner)

**Analysis**:
- Not the 300s hybrid client timeout (would be longer)
- Not the 151s Railway timeout (different value)
- Modal succeeded, so failure happened in network layer
- Runner waited 238s before giving up

**Hypothesis**: Intermediate timeout layer (load balancer, proxy, or network) between runner and hybrid router

**Status**: 🟡 Under investigation - does not affect barrier fix success

---

**Status**: 🟢 IMPROVED - Goroutine coordination fixed, timeout mysteries remain

---

## 🔴 CRITICAL ISSUE: Database Connection Timeout (2025-10-23)

**Date Discovered**: 2025-10-23 00:18 UTC  
**Status**: 🔴 ACTIVE - All jobs failing immediately

### Symptom:
All job submissions fail within 11ms with error:
```
Failed to create execution record in real-time
error="failed to lookup job: failed to connect to `user=neondb_owner database=neondb`:
  dial tcp [IP]:5432: operation was canceled"
```

### Root Cause:
**Runner cannot connect to Neon database** - all connection attempts timeout.

**Error Details**:
- 6 connection attempts (3 IPv6 + 3 IPv4) all fail
- Error: `dial tcp [IP]:5432: operation was canceled`
- Database: Neon PostgreSQL (eu-west-2)
- Runner: Fly.io (London region)

### Current Configuration:

**Location**: `/runner-app/internal/config/config.go` (line 104)

```go
DBTimeout: time.Duration(getInt("DB_TIMEOUT_MS", 4000)) * time.Millisecond
```

**Default**: 4000ms (4 seconds)  
**Environment Variable**: `DB_TIMEOUT_MS`

### Issue:
**4 seconds is too short** for Fly.io (London) → Neon (eu-west-2) connection establishment.

Network path includes:
1. Fly.io London → Internet
2. Internet → AWS eu-west-2
3. AWS → Neon connection pooler
4. Pooler → Actual database

With network latency + connection pooler overhead, 4s is insufficient.

### Fix:

**Increase database timeout to 30 seconds:**

```bash
fly secrets set DB_TIMEOUT_MS=30000 -a beacon-runner-production
```

**Rationale**:
- Allows time for network latency (Fly.io → Neon)
- Handles connection pooler warm-up
- Matches industry standard for remote database connections
- Still fails fast enough (30s) to not block job queue

### Related Timeouts:

**Redis Timeout** (also in config.go):
```go
RedisTimeout: time.Duration(getInt("REDIS_TIMEOUT_MS", 2000)) * time.Millisecond
```

**Default**: 2000ms (2 seconds)  
**Status**: May also need increase if Redis connection issues occur

### Impact:
- ❌ **All jobs fail immediately** (0 executions created)
- ❌ **Portal shows "Job Failed" instantly**
- ❌ **No execution records in database**
- ❌ **Sentry shows database connection errors**

### Timeline:
- **00:08 UTC**: Job `bias-detection-1761174483524` failed
- **00:18 UTC**: Job `bias-detection-1761175099849` failed
- **00:18 UTC**: Root cause identified via Fly logs
- **00:22 UTC**: Neon console verified - database active
- **00:25 UTC**: Fix documented, awaiting deployment

### Verification After Fix:
1. Submit test job through Portal
2. Check Fly logs for successful database connection
3. Verify execution records created in database
4. Confirm job completes successfully

---
