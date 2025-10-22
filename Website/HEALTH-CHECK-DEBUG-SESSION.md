# 🔍 Health Check Debug Session

**Date**: 2025-10-22  
**Issue**: Jobs still failing after health check fix  
**Status**: 🔄 Debugging in progress

---

## 🎯 Current Situation

### **Symptoms**
- ❌ Jobs failing instantly with "No healthy providers available"
- ✅ `/providers` endpoint shows `healthy: true`
- ❌ Actual inference requests fail
- ⚠️ `last_health_check: 0` (health check hasn't run or timestamp not set)

### **Test Results**

**Provider Status**:
```bash
$ curl https://project-beacon-production.up.railway.app/providers | jq '.providers[] | {name, healthy}'
{
  "name": "modal-us-east",
  "healthy": true  # Shows healthy
}
{
  "name": "modal-eu-west",
  "healthy": true  # Shows healthy
}
```

**Inference Test**:
```bash
$ curl -X POST https://project-beacon-production.up.railway.app/inference \
  -d '{"model":"llama3.2-1b","prompt":"test","region_preference":"us-east"}'
{
  "success": false,
  "error": "No healthy providers available"  # ❌ Fails!
}
```

**Direct Modal Test**:
```bash
$ curl -X POST https://jamie-anson--project-beacon-hf-us-inference.modal.run \
  -d '{"model":"llama3.2-1b","prompt":"test"}'
{
  "success": true  # ✅ Modal endpoints work!
}
```

---

## 🔍 Hypothesis

### **Possible Causes**

1. **Health check not running**: Background task on line 66 of `main.py` might be failing silently
2. **Race condition**: Providers initialized as `healthy: true` by default, but health check runs and marks them `false`
3. **Health check failing**: New health check code has a bug
4. **Deployment issue**: Railway hasn't actually deployed the new code yet

### **Evidence**

- ✅ Modal endpoints work directly
- ✅ `/providers` shows `healthy: true`
- ❌ `select_provider()` sees no healthy providers
- ⚠️ `last_health_check: 0` suggests health check hasn't completed

---

## 🔧 Debug Actions Taken

### **1. Added Health Check Logging**

Added detailed logging to `hybrid_router/core/router.py`:

```python
async def _check_provider_health(self, provider: Provider):
    logger.info(f"🔍 [HEALTH_CHECK] Starting health check for {provider.name}")
    # ... health check logic ...
    if response.status_code == 200:
        logger.info(f"✅ [HEALTH_CHECK] {provider.name} is HEALTHY")
    else:
        logger.warning(f"❌ [HEALTH_CHECK] {provider.name} returned status {response.status_code}")
```

### **2. Added Provider Selection Logging**

```python
def select_provider(self, request: InferenceRequest):
    logger.error(
        f"[SELECT_PROVIDER] Total providers: {len(self.providers)}, "
        f"Healthy: {len(healthy_providers)}, "
        f"Provider health: {[(p.name, p.healthy, p.last_health_check) for p in self.providers]}"
    )
```

### **3. Deployed Debug Build**

- Commit: `579c564`
- Message: "debug: Add detailed logging to provider health checks and selection"
- Pushed: 2025-10-22 15:55 UTC
- Railway: Auto-deploying now

---

## 📊 What We'll See in Logs

After Railway deploys (2-3 minutes), we'll see:

### **On Startup**:
```
🔍 [HEALTH_CHECK] Starting health check for modal-us-east
🔍 [HEALTH_CHECK] Starting health check for modal-eu-west
✅ [HEALTH_CHECK] modal-us-east is HEALTHY (success=True)
✅ [HEALTH_CHECK] modal-eu-west is HEALTHY (success=True)
```

### **On Inference Request**:
```
[SELECT_PROVIDER] Total providers: 2, Healthy: 2, Provider health: [('modal-us-east', True, 1761148xxx), ('modal-eu-west', True, 1761148xxx)]
```

### **If Health Check Fails**:
```
❌ [HEALTH_CHECK] modal-us-east failed: <error details>
[SELECT_PROVIDER] Total providers: 2, Healthy: 0, Provider health: [('modal-us-east', False, 0), ('modal-eu-west', False, 0)]
```

---

## 🎯 Next Steps

### **1. Wait for Railway Deployment** (~2-3 minutes)

Check Railway dashboard or wait for GitHub webhook confirmation.

### **2. Check Railway Logs**

```bash
# Look for health check logs
railway logs | grep "HEALTH_CHECK"

# Look for provider selection logs
railway logs | grep "SELECT_PROVIDER"
```

### **3. Submit Test Job**

Once logs show healthy providers, submit a test job:
```bash
# Via portal or API
# Job ID: bias-detection-<timestamp>
```

### **4. Analyze Results**

Based on what we see in logs:

**Scenario A: Health checks succeed**
- Logs show: `✅ [HEALTH_CHECK] modal-us-east is HEALTHY`
- But `[SELECT_PROVIDER]` shows `Healthy: 0`
- **Diagnosis**: Race condition or provider state not persisting

**Scenario B: Health checks fail**
- Logs show: `❌ [HEALTH_CHECK] modal-us-east failed: <error>`
- **Diagnosis**: Health check code has a bug (timeout, wrong endpoint, etc.)

**Scenario C: No health check logs**
- No `[HEALTH_CHECK]` logs appear
- **Diagnosis**: Background task not running or failing silently

**Scenario D: Health checks succeed AND select_provider sees them**
- Logs show healthy providers in both places
- **Diagnosis**: Issue is elsewhere (region matching, model filtering, etc.)

---

## 🔬 Debugging Tools

### **Check Railway Logs**:
```bash
# Via Railway CLI (if installed)
railway logs --tail 100

# Via Railway dashboard
https://railway.app/project/<project-id>/service/<service-id>/logs
```

### **Test Inference**:
```bash
curl -X POST https://project-beacon-production.up.railway.app/inference \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.2-1b",
    "prompt": "test",
    "temperature": 0.1,
    "max_tokens": 10,
    "region_preference": "us-east"
  }' | jq '.'
```

### **Check Provider Status**:
```bash
curl https://project-beacon-production.up.railway.app/providers | jq '.providers[] | {name, healthy, last_health_check}'
```

---

## 📝 Related Files

- **Router Core**: `/hybrid_router/core/router.py` (lines 181-224, 226-244)
- **Main App**: `/hybrid_router/main.py` (line 66 - health check startup)
- **Provider Model**: `/hybrid_router/models/provider.py`
- **Previous Fix**: Commit `75a8ef6` (health check endpoint fix)
- **Debug Build**: Commit `579c564` (added logging)

---

## 💡 Potential Root Causes

### **Most Likely** (ordered by probability):

1. **Health check timing issue** (60%): Health check runs but takes too long, providers marked unhealthy before first request
2. **Health check exception** (20%): Health check code throws exception, caught by `return_exceptions=True`, providers stay unhealthy
3. **HTTP client issue** (10%): `self.client.post()` not working correctly in Railway environment
4. **Deployment lag** (10%): Railway hasn't deployed new code yet, still using old broken health check

---

## ✅ Success Criteria

We'll know the issue is fixed when:

1. ✅ Railway logs show: `✅ [HEALTH_CHECK] modal-us-east is HEALTHY`
2. ✅ Railway logs show: `[SELECT_PROVIDER] ... Healthy: 2`
3. ✅ Test inference request succeeds
4. ✅ Job execution completes successfully

---

**Status**: 🔄 WAITING FOR RAILWAY DEPLOYMENT  
**ETA**: 2-3 minutes  
**Next Action**: Check Railway logs for `[HEALTH_CHECK]` and `[SELECT_PROVIDER]` messages
