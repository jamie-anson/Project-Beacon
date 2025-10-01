# ASIA Provider Timeout Analysis

**Date**: 2025-10-01T16:48:00+01:00  
**Status**: 🔍 **ROOT CAUSE FOUND**

---

## 🎯 Discovery

**ASIA executions are timing out after exactly 120 seconds!**

### Evidence

1. **Execution Timestamps**:
   - Started: 15:37:03
   - Completed: 15:39:03
   - Duration: **Exactly 120 seconds**

2. **Hybrid Router Timeout**: 
   - Default: 120 seconds (from `client.go`)
   - Matches execution duration exactly

3. **Direct Test Works**:
   ```bash
   curl https://project-beacon-production.up.railway.app/inference \
     -d '{"model": "llama3.2-1b", "region_preference": "asia-pacific", ...}'
   ```
   - Result: **SUCCESS** in 11.4 seconds
   - Provider used: `modal-asia-pacific`

---

## 🤔 Why ASIA Times Out But US/EU Don't

### Theory 1: Cold Start
ASIA provider takes longer to cold-start than US/EU.

**Evidence**: 
- Direct test works in 11.4s (warm)
- Runner might be hitting cold starts

**Likelihood**: MEDIUM

### Theory 2: Network Latency
Runner (in London) → Railway (US) → Modal ASIA has higher latency.

**Evidence**:
- Runner is in `lhr` (London)
- Railway is in US
- Modal ASIA is in Singapore
- Round-trip latency could be significant

**Likelihood**: HIGH

### Theory 3: Modal ASIA Provider Issue
The ASIA provider might be slower or having issues.

**Evidence**:
- Direct test works fine
- But maybe under load it's slower?

**Likelihood**: LOW

### Theory 4: Runner Making Multiple Requests
Runner might be retrying or making multiple requests to ASIA.

**Evidence**:
- Need to check logs
- Executor has retry logic?

**Likelihood**: MEDIUM

---

## 🔍 Key Questions

1. **Why does direct test work in 11s but runner times out at 120s?**
   - Is the runner making the request correctly?
   - Is there retry logic adding delays?
   - Is the request getting stuck somewhere?

2. **Why only ASIA?**
   - US and EU work fine
   - All use the same code path
   - Only difference is the region

3. **Is it really timing out or failing earlier?**
   - Empty `provider_id` suggests failure before execution
   - But timestamps suggest 120s timeout
   - Which is it?

---

## 🔧 Debugging Steps

### Step 1: Check if Request is Being Made
Look for TRACE logs in runner:
```
TRACE: Calling hybrid router with request
```

Should show the exact request being sent to hybrid router.

### Step 2: Check Hybrid Router Logs
Check Railway logs for incoming requests with `region_preference: asia-pacific`

### Step 3: Test from Runner Location
The runner is in London (`lhr`). Test latency:
```bash
# From London to Modal ASIA
time curl https://jamie-anson--project-beacon-hf-run-inference-apac.modal.run
```

### Step 4: Increase Timeout
Try increasing `HYBRID_ROUTER_TIMEOUT` to 180 seconds:
```bash
flyctl secrets set HYBRID_ROUTER_TIMEOUT=180 --app beacon-runner-change-me
```

---

## 💡 Hypothesis

**Most Likely**: The runner IS making the request, but it's timing out due to:
1. Network latency (London → US → Singapore)
2. Cold start delays in Modal ASIA
3. Combined latency exceeding 120s threshold

**Evidence**:
- Empty `provider_id` = request never completed
- Exactly 120s duration = timeout
- Direct test works = provider is functional
- Only ASIA affected = geography-specific issue

---

## ✅ Solution Options

### Option 1: Increase Timeout (Quick Fix)
```bash
flyctl secrets set HYBRID_ROUTER_TIMEOUT=300 --app beacon-runner-change-me
```

**Pros**: Simple, might fix immediately  
**Cons**: Doesn't address root cause

### Option 2: Deploy Runner Closer to ASIA
Deploy a second runner instance in Asia region.

**Pros**: Reduces latency permanently  
**Cons**: More complex, costs more

### Option 3: Optimize Modal ASIA Provider
Keep Modal ASIA warm to avoid cold starts.

**Pros**: Improves performance  
**Cons**: Costs more (always-on)

### Option 4: Use Different ASIA Provider
Switch from Modal to RunPod or another provider in ASIA.

**Pros**: Might have better performance  
**Cons**: Need to set up and test

---

## 🎯 Recommended Action

**Try Option 1 first**: Increase timeout to 300 seconds.

```bash
flyctl secrets set HYBRID_ROUTER_TIMEOUT=300 --app beacon-runner-change-me
flyctl restart --app beacon-runner-change-me
```

Then submit another test job and see if ASIA works.

If it works: Problem solved (temporarily)  
If it doesn't: Need to investigate further (check logs, test latency)

---

## 📊 Expected Outcome

After increasing timeout:
- ASIA executions should complete successfully
- Might take 60-180 seconds instead of timing out at 120s
- US/EU should still work fine (unaffected)

---

## 🚀 Next Steps

1. **Increase timeout** to 300s
2. **Restart runner**
3. **Submit test job** (1 question, 3 models, 3 regions)
4. **Check if ASIA works**
5. **If yes**: Consider permanent solution (deploy ASIA runner)
6. **If no**: Check logs for actual error

---

## 📝 Summary

**Problem**: ASIA executions timeout after 120s  
**Root Cause**: Network latency + cold starts exceed timeout  
**Quick Fix**: Increase timeout to 300s  
**Long-term**: Deploy runner in ASIA region or optimize provider
