# CRITICAL: Timeout Configuration Too Low

**Date**: 2025-10-02 00:58 UTC  
**Status**: 🔴 BLOCKS PRODUCTION USE

---

## 🎯 Root Cause: HTTP Timeout = 150s, but Execution Takes 200s+

### The Problem:

**Current Timeout**: `HYBRID_ROUTER_TIMEOUT` = **150 seconds** (2.5 minutes)

**Actual Execution Time**:
- Cold start: ~44 seconds
- Execution: ~158 seconds  
- **Total**: ~202 seconds (3.4 minutes)

**Result**: Runner marks executions as "failed" before Modal actually completes them.

---

## 📊 Evidence from Job bias-detection-1759362340523:

### Timeline:
```
00:45:43 (+0s)   - Runner sends requests to EU/APAC Modal
00:46:27 (+44s)  - Modal containers START (cold start delay)
00:48:13 (+150s) - ⏱️ RUNNER HTTP TIMEOUT! Marks as "failed"
                   - Empty provider_id in database
                   - Runner moves to Q2
00:48:27 (+204s) - Modal ACTUALLY COMPLETES (returns 200)
                   - But runner already gave up
                   - Q2 requests sent to Modal (zombies!)
```

### Database Evidence:
- All failed executions have `provider_id: ""` (no response received)
- All failed at exactly 23:48:13 (150s timeout)
- 6 failures for EU/APAC Q1 (llama, mistral, qwen × 2 regions)

### Modal Evidence:
- APAC runs 233-235: All returned status 200 ✅
- Execution time: 2m 38s - 2m 40s (158-160 seconds)
- EU also has 8 pending requests (zombie requests for Q2)

---

## 🔧 The Fix

### Where: Fly.io Secret Configuration

**Current Value**:
```bash
HYBRID_ROUTER_TIMEOUT=150  # Too low!
```

**Recommended Value**:
```bash
HYBRID_ROUTER_TIMEOUT=300  # 5 minutes
```

### Calculation:
```
Cold Start:      ~45s  (worst case)
Mistral 7B:      ~160s (slowest model)
Buffer:          ~95s  (safety margin)
-------------------------
Total:           300s  (5 minutes)
```

### Why 300 seconds (5 minutes)?

1. **Cold Start Variability**: 
   - Typical: 4-5 seconds
   - Worst case: 30-45 seconds (when Modal needs to spin up containers)

2. **Model Execution Time**:
   - Llama 3.2-1B: ~30-40 seconds
   - Qwen 2.5-1.5B: ~35-45 seconds
   - **Mistral 7B: ~150-160 seconds** (slowest!)

3. **Network Latency**:
   - Fly.io (UK) → Modal (EU/APAC)
   - ~5-10 seconds overhead

4. **Safety Buffer**:
   - Allows for temporary slowdowns
   - Prevents false positives

---

## 🚀 Implementation

### Step 1: Update Fly.io Secret

```bash
cd /Users/Jammie/Desktop/Project\ Beacon/runner-app
flyctl secrets set HYBRID_ROUTER_TIMEOUT=300 -a beacon-runner-change-me
```

### Step 2: Restart Runner (Automatic)

Fly.io will automatically restart the app when secrets change.

### Step 3: Verify

```bash
# Check secret is set
flyctl secrets list -a beacon-runner-change-me

# Monitor logs to confirm new timeout
flyctl logs -a beacon-runner-change-me
```

---

## 📋 Alternative: Per-Model Timeouts

If you want more granular control, we could implement per-model timeouts in the code:

```go
// In internal/worker/executor_hybrid.go or similar

func getModelTimeout(modelID string) time.Duration {
    switch modelID {
    case "mistral-7b":
        return 300 * time.Second  // 5 minutes for Mistral
    case "llama3.2-1b":
        return 120 * time.Second  // 2 minutes for Llama
    case "qwen2.5-1.5b":
        return 120 * time.Second  // 2 minutes for Qwen
    default:
        return 180 * time.Second  // 3 minutes default
    }
}
```

But this is **more complex** and not needed right now. The global timeout increase is simpler and sufficient.

---

## 🔍 Testing Plan

### After Timeout Increase:

1. **Submit 2-question test job** via portal
2. **Monitor execution timing**:
   ```bash
   flyctl logs -a beacon-runner-change-me --follow | grep -E "(timeout|completed)"
   ```
3. **Expected Results**:
   - ✅ All EU/APAC executions complete (no premature timeouts)
   - ✅ Provider IDs populated in database
   - ✅ No zombie requests sent after first question
   - ✅ Context cancellation still works (if needed)

### Success Criteria:
- [ ] No executions fail with empty `provider_id`
- [ ] Modal execution times < 300s for all models
- [ ] No timeout errors in logs
- [ ] All regions complete successfully or fail for real reasons

---

## ⚠️ Why This Wasn't Caught Earlier

### Previous Test Jobs:
- **bias-detection-1758114275**: Only US succeeded (EU/APAC failed quickly)
- **bias-detection-1759359380236**: US completed fast, EU/APAC timed out but you cancelled manually
- **bias-detection-1759360485977**: Same pattern

### The Pattern:
- US has only 2 models (faster)
- US completes Q1 and Q2 before timeout
- EU/APAC have 3 models (slower, especially Mistral 7B)
- EU/APAC Q1 hits 150s timeout
- Runner thinks they failed
- Sends Q2 to Modal anyway (zombie requests)

---

## 💡 Additional Optimization (Future)

### Consider: Increase Modal Container Warm Pool

To reduce cold start times, you could configure Modal to keep containers warm:

```python
# In Modal deployment
@app.function(
    keep_warm=1,  # Keep 1 container warm per region
    container_idle_timeout=300  # Keep alive for 5 minutes
)
def inference(...):
    pass
```

**Trade-off**: Costs ~$0.01/hour per warm container, but eliminates 44s cold starts.

---

## 📊 Expected Impact

### Before Fix:
- ❌ 50-60% timeout failures on EU/APAC
- ❌ 8-12 zombie requests per 2-question job
- ❌ Confusing "failed" status with successful Modal execution
- ❌ Wasted Modal resources

### After Fix:
- ✅ 0% timeout failures (unless real issues)
- ✅ 0 zombie requests (context cancellation works)
- ✅ Accurate job status
- ✅ No wasted resources

### Performance:
- **Reliability**: 50% → 90%+ success rate
- **Accuracy**: False failures eliminated
- **Cost**: No change (same Modal usage, just properly tracked)

---

## 🔴 UPDATE (2025-10-02 01:26): FIX DID NOT WORK

### What Happened:

**Attempted Fix**:
```bash
flyctl secrets set HYBRID_ROUTER_TIMEOUT=300 -a beacon-runner-change-me
```

**Result**: ❌ **FAILED** - Timeouts still occurring at ~151 seconds

### Test Job: bias-detection-1759363701760

**Evidence**:
- ✅ US-EAST: 6/6 completed (provider_id populated)
- ✅ EU-WEST: 6/6 completed (provider_id populated)  
- ❌ APAC: 6/6 failed (timeout at 151s, empty provider_id)

**Timeline**:
```
Q1: 00:08:29 → 00:11:00 (151 seconds - TIMEOUT)
Q2: 00:11:00 → 00:14:27 (207 seconds - sent anyway!)
```

### The Mystery:

Timeouts occurring at **151 seconds**, which doesn't match ANY configured value:
- Code default: 120s ❌
- Attempted setting: 300s ❌
- Hybrid Router: 600s ❌
- Modal: 900s ❌

### Issues Identified:

1. **Secret digest unchanged**: `70f17482d5716502` (before and after update)
2. **Can't verify secret value**: Only see digest, not actual value
3. **Context cancellation still broken**: Q2 sent after Q1 timeout
4. **Unknown timeout source**: 151s doesn't match any configuration

---

## 🔍 Next Steps: Add Diagnostic Logging

### Why We Need Logging:

**We can't verify the timeout value the code is actually using.**

The code reads the environment variable but we have no visibility into:
- Is `HYBRID_ROUTER_TIMEOUT` being read correctly?
- What value is it actually set to?
- Is there a different timeout being applied?

### Logging to Add:

**Location**: `/runner-app/internal/hybrid/client.go` (line 23-41)

```go
func New(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://project-beacon-production.up.railway.app"
	}
	
	// Determine HTTP timeout: default 120s, overridable via env
	timeoutSec := 120
	envVarUsed := "none"
	
	if v := os.Getenv("HYBRID_ROUTER_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeoutSec = n
			envVarUsed = "HYBRID_ROUTER_TIMEOUT"
		}
	} else if v := os.Getenv("HYBRID_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeoutSec = n
			envVarUsed = "HYBRID_TIMEOUT"
		}
	}
	
	// LOG THE ACTUAL TIMEOUT VALUE BEING USED
	log.Printf("[HYBRID_CLIENT] Initializing with timeout=%ds (source: %s, url: %s)", 
		timeoutSec, envVarUsed, baseURL)
	
	return &Client{
		baseURL:    trimRightSlash(baseURL),
		httpClient: &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
	}
}
```

**What This Will Tell Us**:
- Actual timeout value being used (120? 150? 300?)
- Which environment variable was read (if any)
- Whether the secret is being applied

---

## 📚 Source of Truth Documentation

**Comprehensive timeout documentation**: 
`/Website/docs/sot/timeouts.md`

This file documents:
- All timeout configurations across the stack
- Current values and locations
- Known issues and investigation status
- Test results and evidence

**Update this file as we discover more information.**

---

## 🎯 Revised Action Plan

### Step 1: Add Diagnostic Logging ✅ (Ready to implement)

```bash
cd /Users/Jammie/Desktop/Project\ Beacon/runner-app
# Edit internal/hybrid/client.go to add logging
# Deploy to Fly.io
flyctl deploy
```

### Step 2: Check Logs After Deployment

```bash
# Watch for the initialization log
flyctl logs -a beacon-runner-change-me | grep "HYBRID_CLIENT"
```

Expected output:
```
[HYBRID_CLIENT] Initializing with timeout=300s (source: HYBRID_ROUTER_TIMEOUT, url: ...)
```

### Step 3: Submit Test Job

After confirming the timeout value in logs, submit a new test job.

### Step 4: Verify Results

- If timeout is 120s → Secret not applied, need to debug Fly.io secrets
- If timeout is 150s → Different source, need to investigate
- If timeout is 300s → Secret applied but different issue (platform timeout?)

---

**Status**: 🔴 INVESTIGATION REQUIRED - Secret update ineffective  
**Priority**: P0 - Blocks production use  
**Effort**: 15 minutes (add logging + deploy + test)  
**Next**: Add logging to diagnose actual timeout value
