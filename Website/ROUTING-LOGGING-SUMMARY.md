# Regional Routing Logging & Monitoring

**Created**: 2025-10-22 23:20 UTC+01:00  
**Status**: ✅ COMPREHENSIVE LOGGING AVAILABLE

---

## 📋 Summary

**Question**: Do we have logs (custom or Sentry) for checking if routing was done correctly?

**Answer**: ✅ **YES** - We have comprehensive logging at multiple levels:

1. **Custom Application Logs** (via Python logging)
2. **Sentry Integration** (distributed tracing + error tracking)
3. **Debug API Endpoints** (real-time routing inspection)

---

## 🔍 Custom Application Logs

### Location
`hybrid_router/core/router.py` - Provider selection logic

### Log Levels

#### INFO Level - Normal Routing
```python
# When region preference is specified
logger.info(
    f"Region-locked provider selection: {request.region_preference} "
    f"(provider_count={len(region_providers)})"
)

# When provider is selected
logger.info(
    f"Provider selected: {provider.name} (region={provider.region}, "
    f"type={provider.type.value}, region_locked={bool(request.region_preference)})"
)

# Fallback provider selection
logger.info(
    f"Provider selected (fallback): {selected.name} (region={selected.region}, "
    f"type={selected.type.value}, region_locked={bool(request.region_preference)})"
)
```

**Example Log Output**:
```
INFO: Region-locked provider selection: us-east (provider_count=1)
INFO: Provider selected: modal-us-east (region=us-east, type=modal, region_locked=True)
```

#### ERROR Level - Routing Failures
```python
# When no providers available for requested region
logger.error(
    f"No healthy providers available for region {request.region_preference}. "
    f"Available regions: {available_regions}"
)

# When no healthy providers at all
logger.error(
    f"[SELECT_PROVIDER] NO HEALTHY PROVIDERS! "
    f"All providers: {[(p.name, p.healthy) for p in self.providers]}"
)

# Provider health status debug
logger.error(
    f"[SELECT_PROVIDER] Total providers: {len(self.providers)}, "
    f"Healthy: {len(healthy_providers)}, "
    f"Provider health: {[(p.name, p.healthy, p.last_health_check) for p in self.providers]}"
)
```

**Example Error Log**:
```
ERROR: No healthy providers available for region eu-west. Available regions: ['us-east']
ERROR: [SELECT_PROVIDER] NO HEALTHY PROVIDERS! All providers: [('modal-us-east', True), ('modal-eu-west', False)]
```

### Log Fields Captured

| Field | Description | Example |
|-------|-------------|---------|
| `provider.name` | Provider identifier | `modal-us-east` |
| `provider.region` | Geographic region | `us-east` |
| `provider.type` | Provider type | `modal` |
| `region_locked` | Whether region was explicitly requested | `True` |
| `provider_count` | Number of available providers in region | `1` |
| `request.region_preference` | User's requested region | `us-east` |

---

## 🎯 Sentry Integration

### Location
`hybrid_router/api/inference.py` - Inference endpoint

### Transaction Tracking

```python
# Start distributed transaction
transaction = sentry_sdk.start_transaction(op="inference", name="router.inference")

# Tag with routing metadata
transaction.set_tag("model", inference_request.model)
transaction.set_tag("region", inference_request.region_preference or "auto")

# Add breadcrumbs for routing decisions
sentry_sdk.add_breadcrumb(
    category="inference",
    message="Inference request received",
    level="info",
    data={
        "model": inference_request.model,
        "region_preference": inference_request.region_preference
    }
)
```

### What Sentry Captures

1. **Transaction Tags**:
   - `model`: Which model was requested
   - `region`: Which region was requested (or "auto")
   - `service`: Always "router"

2. **Breadcrumbs**:
   - Inference request received
   - Provider selection
   - Inference execution
   - Response returned

3. **Error Context**:
   - Full stack trace
   - Request parameters
   - Provider states
   - Routing decision chain

4. **Performance Metrics**:
   - Transaction duration
   - Provider selection time
   - Inference execution time
   - Total request time

### Sentry Dashboard Views

**Performance → Transactions**:
- Filter by `transaction:router.inference`
- Group by `region` tag to see routing distribution
- View P50/P95/P99 latencies per region

**Issues → Search**:
- `transaction:router.inference region:us-east` - US routing issues
- `transaction:router.inference region:eu-west` - EU routing issues
- `message:"No healthy providers"` - Provider availability issues

---

## 🛠️ Debug API Endpoints

### Real-Time Routing Inspection

#### 1. Test Inference with Diagnostics
```bash
curl -X POST https://project-beacon-production.up.railway.app/debug/test-inference \
  -H "Content-Type: application/json" \
  -d '{"model":"llama3.2-1b","prompt":"test","region_preference":"us-east","max_tokens":5}'
```

**Response**:
```json
{
  "success": true,
  "provider_selected": "modal-us-east",
  "provider_used": "modal-us-east",
  "duration_seconds": 0.76,
  "error": null,
  "provider_states": {
    "modal-us-east": {"healthy": true, "region": "us-east"},
    "modal-eu-west": {"healthy": true, "region": "eu-west"}
  },
  "healthy_count": 2
}
```

**Key Fields**:
- `provider_selected`: Which provider was chosen by routing logic
- `provider_used`: Which provider actually handled the request
- `provider_states`: Health status of all providers
- `healthy_count`: Number of healthy providers available

#### 2. Provider Status
```bash
curl https://project-beacon-production.up.railway.app/debug/providers
```

**Response**:
```json
{
  "providers": [
    {
      "name": "modal-us-east",
      "type": "modal",
      "endpoint": "https://jamie-anson--project-beacon-hf-us-inference.modal.run",
      "region": "us-east",
      "healthy": true,
      "last_health_check_ago_seconds": 10.5
    },
    {
      "name": "modal-eu-west",
      "type": "modal",
      "endpoint": "https://jamie-anson--project-beacon-hf-eu-inference.modal.run",
      "region": "eu-west",
      "healthy": false,
      "last_health_check_ago_seconds": 5.2
    }
  ]
}
```

---

## 📊 How to Verify Routing

### Method 1: Check Application Logs (Railway)

```bash
# Via Railway CLI
railway logs --tail 100 | grep -E "(Region-locked|Provider selected|region_preference)"

# Expected output:
# INFO: Region-locked provider selection: us-east (provider_count=1)
# INFO: Provider selected: modal-us-east (region=us-east, type=modal, region_locked=True)
```

### Method 2: Check Sentry Dashboard

1. Go to Sentry → Performance → Transactions
2. Filter: `transaction:router.inference`
3. Group by: `region` tag
4. View distribution:
   - `us-east`: X% of requests
   - `eu-west`: Y% of requests
   - `auto`: Z% of requests

### Method 3: Use Debug Endpoint

```bash
# Test specific region routing
curl -X POST https://project-beacon-production.up.railway.app/debug/test-inference \
  -H "Content-Type: application/json" \
  -d '{"model":"llama3.2-1b","prompt":"test","region_preference":"us-east","max_tokens":5}' \
  | jq '{provider_selected, provider_used, region: .provider_states["modal-us-east"].region}'
```

**Expected**:
```json
{
  "provider_selected": "modal-us-east",
  "provider_used": "modal-us-east",
  "region": "us-east"
}
```

### Method 4: Check Response Metadata

Every inference response includes routing metadata:

```bash
curl -X POST https://project-beacon-production.up.railway.app/inference \
  -H "Content-Type: application/json" \
  -d '{"model":"llama3.2-1b","prompt":"test","region_preference":"us-east","max_tokens":5}' \
  | jq '{provider_used, region: .metadata.region, receipt_region: .metadata.receipt.execution_details.region}'
```

**Expected**:
```json
{
  "provider_used": "modal-us-east",
  "region": "us-east",
  "receipt_region": "us-east"
}
```

---

## 🧪 Automated Test

**Location**: `tests/test_regional_routing.py`

**Run**:
```bash
python3 tests/test_regional_routing.py
```

**Output**:
```
🧪 Testing Regional Routing
============================================================

1️⃣ Testing US-East routing...
✅ US-East routing verified: modal-us-east

2️⃣ Testing EU-West routing...
✅ EU-West routing verified: modal-eu-west

3️⃣ Testing Asia-Pacific routing...
⚠️  APAC: No healthy providers available

4️⃣ Testing all regions in parallel...
✅ us-east: Routed to modal-us-east
✅ eu-west: Routed to modal-eu-west
⚠️  asia-pacific: No healthy providers available

5️⃣ Testing receipt consistency...
✅ Receipt data consistent with routing decision

============================================================
🎉 Regional routing tests completed!
```

---

## 🔍 Common Routing Issues & How to Debug

### Issue 1: Request Goes to Wrong Region

**Symptoms**:
- User requests `us-east` but gets `eu-west` provider
- `provider_used` doesn't match `region_preference`

**Debug Steps**:
1. Check logs for "Region-locked provider selection"
2. Verify `region_locked=True` in provider selection log
3. Check if requested region provider is healthy
4. Look for "No healthy providers available for region" error

**Sentry Query**:
```
transaction:router.inference region:us-east
```
Filter to failed transactions, check breadcrumbs for provider selection.

### Issue 2: No Providers Available for Region

**Symptoms**:
- Error: "No healthy providers available for region X"
- Job fails immediately

**Debug Steps**:
1. Check provider health: `curl .../debug/providers`
2. Check health check logs: `railway logs | grep HEALTH_CHECK`
3. Force health check: `curl -X POST .../debug/force-health-check`
4. Check Modal endpoint directly

**Sentry Query**:
```
message:"No healthy providers available for region"
```

### Issue 3: Routing Fallback Not Working

**Symptoms**:
- Request fails instead of falling back to another region
- Expected fallback behavior not occurring

**Debug Steps**:
1. Check if `region_preference` is set (strict mode)
2. Review router code - fallback is DISABLED when region is explicitly requested
3. This is by design - region preference is strict to ensure cross-region consistency

**Expected Behavior**:
- With `region_preference`: Strict routing, no fallback
- Without `region_preference`: Auto-select any healthy provider

---

## 📝 Summary

### ✅ What We Have

1. **Custom Logs**:
   - ✅ Provider selection decisions
   - ✅ Region-locked routing
   - ✅ Fallback behavior
   - ✅ Health check status

2. **Sentry Integration**:
   - ✅ Transaction tracking with region tags
   - ✅ Breadcrumbs for routing decisions
   - ✅ Error context with provider states
   - ✅ Performance metrics per region

3. **Debug Endpoints**:
   - ✅ Real-time provider status
   - ✅ Test inference with diagnostics
   - ✅ Provider health inspection
   - ✅ Routing decision visibility

4. **Automated Tests**:
   - ✅ Regional routing verification
   - ✅ Receipt consistency validation
   - ✅ Fallback behavior testing

### 🎯 Recommended Monitoring

**Daily**:
- Check Sentry for routing errors
- Review provider health status
- Monitor region distribution

**Per Deployment**:
- Run regional routing tests
- Verify all regions healthy
- Check debug endpoints

**On Issues**:
- Check Railway logs for routing decisions
- Use debug endpoints for real-time inspection
- Review Sentry transaction traces
- Run automated routing tests

---

**Status**: ✅ COMPREHENSIVE - We have excellent visibility into routing decisions at all levels
