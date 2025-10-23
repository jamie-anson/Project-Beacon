# 🔬 Router Diagnostic Tools - Complete Guide

**Date**: 2025-10-22  
**Purpose**: Make router health issues easy to diagnose  
**Status**: ✅ Deployed to Railway

---

## 🎯 Problem Solved

**Before**: Hard to diagnose why providers show as healthy but inference fails
- Had to manually check logs
- No visibility into health check timing
- Couldn't test individual components
- No way to force health checks

**After**: Comprehensive diagnostic endpoints and automated testing
- ✅ See exact provider states in real-time
- ✅ Test individual providers
- ✅ Force health checks manually
- ✅ Test inference with detailed diagnostics
- ✅ Automated diagnostic script

---

## 🛠️ New Debug Endpoints

### **1. GET /debug/providers**
**Purpose**: Detailed provider status with timing information

```bash
curl https://project-beacon-production.up.railway.app/debug/providers | jq '.'
```

**Returns**:
```json
{
  "total_providers": 2,
  "healthy_count": 2,
  "providers": [
    {
      "name": "modal-us-east",
      "type": "modal",
      "endpoint": "https://jamie-anson--project-beacon-hf-us-inference.modal.run",
      "region": "us-east",
      "healthy": true,
      "last_health_check": 1761148500.123,
      "last_health_check_ago_seconds": 45.2,
      "avg_latency": 0.0,
      "success_rate": 1.0,
      "cost_per_second": 0.00005,
      "max_concurrent": 10
    }
  ],
  "timestamp": 1761148545.345
}
```

**Use Case**: See if providers are actually healthy and when they were last checked

---

### **2. POST /debug/force-health-check**
**Purpose**: Manually trigger health check for all providers

```bash
curl -X POST https://project-beacon-production.up.railway.app/debug/force-health-check | jq '.'
```

**Returns**:
```json
{
  "success": true,
  "duration_seconds": 2.45,
  "before": {
    "modal-us-east": true,
    "modal-eu-west": false
  },
  "after": {
    "modal-us-east": true,
    "modal-eu-west": true
  },
  "changes": {
    "modal-eu-west": {
      "from": false,
      "to": true
    }
  },
  "timestamp": 1761148545.345
}
```

**Use Case**: Test if health checks can succeed, see state changes

---

### **3. POST /debug/test-provider/{provider_name}**
**Purpose**: Test a specific provider's health check

```bash
curl -X POST https://project-beacon-production.up.railway.app/debug/test-provider/modal-us-east | jq '.'
```

**Returns**:
```json
{
  "success": true,
  "error": null,
  "provider": "modal-us-east",
  "duration_seconds": 1.23,
  "before": {
    "healthy": false,
    "last_health_check": 0
  },
  "after": {
    "healthy": true,
    "last_health_check": 1761148545.345
  },
  "changed": true,
  "timestamp": 1761148545.345
}
```

**Use Case**: Isolate which provider is failing health checks

---

### **4. POST /debug/test-inference**
**Purpose**: Test inference with detailed diagnostics

```bash
curl -X POST https://project-beacon-production.up.railway.app/debug/test-inference | jq '.'
```

**Returns**:
```json
{
  "success": true,
  "provider_selected": "modal-us-east",
  "provider_used": "modal-us-east",
  "duration_seconds": 1.45,
  "error": null,
  "provider_states": {
    "modal-us-east": {
      "healthy": true,
      "region": "us-east"
    },
    "modal-eu-west": {
      "healthy": true,
      "region": "eu-west"
    }
  }
}
```

**Use Case**: Test if inference works and see provider states at time of request

---

### **5. GET /debug/health-check-history**
**Purpose**: Show when health checks last ran

```bash
curl https://project-beacon-production.up.railway.app/debug/health-check-history | jq '.'
```

**Returns**:
```json
{
  "providers": [
    {
      "provider": "modal-us-east",
      "healthy": true,
      "last_check_timestamp": 1761148500.123,
      "last_check_ago_seconds": 45.2,
      "last_check_ago_human": "45s ago"
    },
    {
      "provider": "modal-eu-west",
      "healthy": false,
      "last_check_timestamp": 0,
      "last_check_ago_seconds": null,
      "last_check_ago_human": "Never"
    }
  ],
  "current_timestamp": 1761148545.345
}
```

**Use Case**: See if health checks are running regularly

---

### **6. GET /debug/startup-status**
**Purpose**: Check if startup health checks completed

```bash
curl https://project-beacon-production.up.railway.app/debug/startup-status | jq '.'
```

**Returns**:
```json
{
  "startup_health_checks_completed": true,
  "some_health_checks_completed": true,
  "providers_checked": 2,
  "providers_total": 2,
  "providers_healthy": 2,
  "providers_never_checked": []
}
```

**Use Case**: Verify startup health checks ran successfully

---

## 🤖 Automated Diagnostic Script

### **Usage**

```bash
cd /Users/Jammie/Desktop/Project\ Beacon/Website
./scripts/diagnose-router-health.sh
```

### **What It Does**

Runs 8 comprehensive tests:

1. ✅ **Basic Connectivity** - Can we reach the router?
2. ✅ **Startup Status** - Did startup health checks complete?
3. ✅ **Provider Details** - What's the state of each provider?
4. ✅ **Health Check History** - When were providers last checked?
5. ✅ **Force Health Check** - Can we manually trigger checks?
6. ✅ **Test Individual Providers** - Test each provider separately
7. ✅ **Test Inference** - Does inference work end-to-end?
8. ✅ **Direct Modal Test** - Are Modal endpoints accessible?

### **Example Output**

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🔬 Router Health Comprehensive Diagnostics
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📋 Test 1: Basic Connectivity
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ Router is reachable

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📋 Test 2: Startup Health Check Status
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ Startup health checks completed (2/2 providers checked)
ℹ️  Healthy providers: 2/2

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📋 Test 3: Provider Detailed Status
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  modal-us-east:
    Region: us-east
    Healthy: true
    Endpoint: https://jamie-anson--project-beacon-hf-us-inference.modal.run
    Last check: 45s ago

  modal-eu-west:
    Region: eu-west
    Healthy: true
    Endpoint: https://jamie-anson--project-beacon-hf-eu-inference.modal.run
    Last check: 45s ago

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📋 Test 7: Test Inference Request
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
ℹ️  Testing inference with diagnostics...
✅ Inference succeeded via modal-us-east (1.45s)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📋 Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

ℹ️  Diagnostic complete. Key findings:

✅ Router is healthy and inference is working

✅ No issues detected. System is operational.
```

---

## 🔍 Troubleshooting Workflows

### **Scenario 1: Jobs Failing with "No healthy providers"**

**Steps**:
```bash
# 1. Run diagnostic script
./scripts/diagnose-router-health.sh

# 2. Check startup status
curl https://project-beacon-production.up.railway.app/debug/startup-status | jq '.'

# 3. If startup checks didn't complete, force a health check
curl -X POST https://project-beacon-production.up.railway.app/debug/force-health-check | jq '.'

# 4. Test inference
curl -X POST https://project-beacon-production.up.railway.app/debug/test-inference | jq '.'
```

**What to look for**:
- ❌ `startup_health_checks_completed: false` → Health checks didn't run
- ❌ `last_health_check: 0` → Provider never checked
- ❌ `healthy: false` after force check → Health check is failing
- ✅ `healthy: true` but inference fails → Issue in select_provider logic

---

### **Scenario 2: Providers Show Healthy But Inference Fails**

**Steps**:
```bash
# 1. Check provider states
curl https://project-beacon-production.up.railway.app/debug/providers | jq '.providers[] | {name, healthy, last_check_ago_seconds}'

# 2. Test inference with diagnostics
curl -X POST https://project-beacon-production.up.railway.app/debug/test-inference | jq '.'

# 3. Check Railway logs for SELECT_PROVIDER messages
railway logs | grep "SELECT_PROVIDER"
```

**What to look for**:
- Provider states in `/debug/providers` vs `/debug/test-inference`
- `[SELECT_PROVIDER]` logs showing `Healthy: 0` when providers show healthy
- Race condition: providers healthy in API but not in select_provider

---

### **Scenario 3: Health Checks Taking Too Long**

**Steps**:
```bash
# 1. Test individual providers
curl -X POST https://project-beacon-production.up.railway.app/debug/test-provider/modal-us-east | jq '.duration_seconds'
curl -X POST https://project-beacon-production.up.railway.app/debug/test-provider/modal-eu-west | jq '.duration_seconds'

# 2. Force health check and measure total time
time curl -X POST https://project-beacon-production.up.railway.app/debug/force-health-check | jq '.duration_seconds'
```

**What to look for**:
- Duration > 10s → Modal cold start or network issues
- One provider slow, others fast → Specific provider issue
- All providers slow → Network or Railway issue

---

### **Scenario 4: Providers Never Checked**

**Steps**:
```bash
# 1. Check startup status
curl https://project-beacon-production.up.railway.app/debug/startup-status | jq '.providers_never_checked'

# 2. Check Railway logs for startup errors
railway logs | grep -E "(HEALTH_CHECK|Starting Project Beacon)"

# 3. Force health check
curl -X POST https://project-beacon-production.up.railway.app/debug/force-health-check | jq '.'
```

**What to look for**:
- Startup task failed silently
- Exception in health check code
- Background task not running

---

## 📊 Quick Reference

### **One-Line Health Check**
```bash
curl -s https://project-beacon-production.up.railway.app/debug/startup-status | jq '{completed: .startup_health_checks_completed, healthy: .providers_healthy, total: .providers_total}'
```

### **Provider Summary**
```bash
curl -s https://project-beacon-production.up.railway.app/debug/providers | jq '.providers[] | {name, healthy, last_check: .last_health_check_ago_seconds}'
```

### **Test Everything**
```bash
./scripts/diagnose-router-health.sh
```

### **Force Refresh**
```bash
curl -X POST https://project-beacon-production.up.railway.app/debug/force-health-check | jq '{success, duration_seconds, changes}'
```

---

## 🎯 Next Steps

### **After Railway Deploys** (~2-3 minutes)

1. **Run diagnostic script**:
   ```bash
   ./scripts/diagnose-router-health.sh
   ```

2. **If issues found**, use specific debug endpoints to investigate

3. **Check Railway logs** for detailed health check messages:
   ```bash
   railway logs | grep -E "(HEALTH_CHECK|SELECT_PROVIDER)"
   ```

4. **Submit test job** and verify it succeeds

---

## 📝 Files Created

- **Debug API**: `/hybrid_router/api/debug.py` (6 endpoints)
- **API Registration**: `/hybrid_router/api/__init__.py` (added debug_router)
- **Main App**: `/hybrid_router/main.py` (registered debug_router)
- **Diagnostic Script**: `/scripts/diagnose-router-health.sh` (8 tests)
- **This Guide**: `/ROUTER-DIAGNOSTIC-TOOLS.md`

---

## 💡 Key Benefits

### **Before**:
- ❌ Had to manually check logs
- ❌ No visibility into health check timing
- ❌ Couldn't test components individually
- ❌ Hard to diagnose race conditions
- ❌ No way to force health checks

### **After**:
- ✅ Real-time provider status with timing
- ✅ Manual health check triggering
- ✅ Individual component testing
- ✅ Detailed inference diagnostics
- ✅ Automated comprehensive testing
- ✅ Clear troubleshooting workflows

---

**Status**: ✅ DEPLOYED TO RAILWAY  
**Latest Commit**: `36817cf` (Fixed Modal endpoint paths)  
**Previous Commit**: `7af1ed1` (Initial diagnostic tools)  
**Ready**: After Railway deployment completes (~2-3 min)

---

## ⚠️ Known Issues & Fixes

### **Issue 1: Modal Endpoints Returning 404** (FIXED in `36817cf`)
**Symptom**: Health checks fail, providers show as unhealthy, but Modal dashboard shows functions running  
**Root Cause**: Router was posting to root URL instead of `/inference` endpoint  
**Fix**: Added `/inference` path to Modal endpoint URLs in both health checks and inference calls

**Before**:
```python
response = await self.client.post(provider.endpoint, ...)  # ❌ 404 Not Found
```

**After**:
```python
endpoint_url = f"{provider.endpoint}/inference" if provider.type == ProviderType.MODAL else provider.endpoint
response = await self.client.post(endpoint_url, ...)  # ✅ Works!
```

**Verification**:
```bash
# Test Modal endpoint directly
curl -X POST https://jamie-anson--project-beacon-hf-us-inference.modal.run/inference \
  -H "Content-Type: application/json" \
  -d '{"model":"llama3.2-1b","prompt":"test","max_tokens":5}' | jq '.'

# Should return: {"success": true, "response": "..."}
```

---

## 🔍 Troubleshooting Common Issues

### **Providers Show as Unhealthy**

1. **Check if Modal endpoints are actually running**:
   ```bash
   # Test US endpoint
   curl -X POST https://jamie-anson--project-beacon-hf-us-inference.modal.run/inference \
     -H "Content-Type: application/json" \
     -d '{"model":"llama3.2-1b","prompt":"test","max_tokens":5}'
   
   # Test EU endpoint  
   curl -X POST https://jamie-anson--project-beacon-hf-eu-inference.modal.run/inference \
     -H "Content-Type: application/json" \
     -d '{"model":"llama3.2-1b","prompt":"test","max_tokens":5}'
   ```

2. **Force a health check**:
   ```bash
   curl -X POST https://project-beacon-production.up.railway.app/debug/force-health-check | jq '.'
   ```

3. **Check health check duration**:
   - If `duration_seconds` ≈ 30: Health checks are timing out (bad)
   - If `duration_seconds` < 5: Health checks are working (good)

4. **Check Railway logs**:
   ```bash
   # Look for health check errors
   railway logs | grep "HEALTH_CHECK"
   ```

### **Jobs Fail Immediately**

1. **Check provider health first**:
   ```bash
   curl https://project-beacon-production.up.railway.app/debug/providers | jq '.healthy_count'
   ```
   - If `healthy_count` = 0: Providers are down, check health checks
   - If `healthy_count` > 0: Issue is elsewhere

2. **Test inference directly**:
   ```bash
   curl -X POST https://project-beacon-production.up.railway.app/debug/test-inference \
     -H "Content-Type: application/json" \
     -d '{"model":"llama3.2-1b","prompt":"test"}' | jq '.'
   ```

3. **Check for "No healthy providers" error**:
   - If present: Health checks are failing
   - If absent: Issue is in job submission/execution

### **Health Checks Timeout**

**Symptom**: `force-health-check` takes 30 seconds, providers stay unhealthy

**Possible Causes**:
1. Modal endpoints are down (check Modal dashboard)
2. Wrong endpoint URL (missing `/inference` path)
3. Network issues between Railway and Modal
4. Modal cold start taking too long

**Debug Steps**:
```bash
# 1. Test Modal directly (bypass router)
curl -X POST https://jamie-anson--project-beacon-hf-us-inference.modal.run/inference \
  -H "Content-Type: application/json" \
  -d '{"model":"llama3.2-1b","prompt":"test","max_tokens":5}'

# 2. Check if it's a cold start issue
# Run the curl command 2-3 times - should get faster

# 3. Check Railway → Modal connectivity
# Look for network errors in Railway logs
```

---

## 🆕 **Known Issues & Resolutions** (Updated 2025-10-23)

### **Issue: "No Healthy Providers" Despite Healthy Status**

**Discovered**: 2025-10-23 00:29 UTC  
**Status**: 🔴 ACTIVE

**Symptoms**:
- `/debug/providers` shows: `healthy_count: 2`, both providers `healthy: true`
- `/health` shows: `status: "healthy"`, `providers_healthy: 2`
- Force health check: Completes in 0.5s, all providers pass
- **BUT**: Runner gets `"No healthy providers available"` error

**Evidence**:
```bash
# Router reports healthy
curl /debug/providers
→ {"healthy_count": 2, "providers": [{"healthy": true}, ...]}

# Runner logs show failure
ERR Execution failed error="No healthy providers available"
```

**Root Cause**: Provider selection logic bug (under investigation)

**Possible Causes**:
1. Race condition in provider health state
2. Region-specific filtering broken
3. Provider capacity check failing incorrectly
4. Stale provider state during request

**Diagnostic Commands**:
```bash
# 1. Check provider state during failure
curl /debug/providers | jq '.providers[] | {name, healthy, last_check_ago}'

# 2. Test direct inference (bypass runner)
curl -X POST https://project-beacon-production.up.railway.app/inference \
  -H "Content-Type: application/json" \
  -d '{"model":"llama3.2-1b","prompt":"test","max_tokens":5}' | jq '.'

# 3. Check Railway logs for provider selection
# Look for: "[SELECT_PROVIDER]" log messages

# 4. Force health check and immediately test
curl -X POST /debug/force-health-check && \
curl -X POST /inference -d '{"model":"llama3.2-1b","prompt":"test","max_tokens":5}'
```

**Workaround**: None currently - investigating root cause

---

### **Issue: Database Connection Timeout** ✅ FIXED

**Discovered**: 2025-10-23 00:18 UTC  
**Status**: ✅ RESOLVED

**Symptoms**:
- All jobs fail within 11ms
- Error: `"failed to connect to database: dial tcp [IP]:5432: operation was canceled"`
- 6 connection attempts (3 IPv6 + 3 IPv4) all timeout

**Root Cause**: 
- Default `DB_TIMEOUT_MS`: 4000ms (4 seconds)
- Network path Fly.io London → Neon eu-west-2 takes >4s

**Fix Applied**:
```bash
fly secrets set DB_TIMEOUT_MS=30000 -a beacon-runner-production
```

**Result**: Database connections now succeed

---

### **Issue: Modal Health Check Path** ✅ FIXED

**Discovered**: 2025-10-22 17:00 UTC  
**Status**: ✅ RESOLVED

**Symptoms**:
- Health checks timing out after 30s
- Modal endpoints working but router reports unhealthy
- Health checks hitting `/inference` path instead of root

**Root Cause**: 
- Router was appending `/inference` to Modal health check URLs
- Modal health endpoint is at root path, not `/inference`

**Fix Applied**: 
- Use Modal's dedicated `/health` endpoint
- URL: `https://jamie-anson--project-beacon-hf-{region}-health.modal.run`

**Result**: Health checks now complete in 0.5s

---
