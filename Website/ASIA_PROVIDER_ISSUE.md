# ASIA Provider Issue - Root Cause

**Date**: 2025-10-01T16:41:00+01:00  
**Status**: 🔍 **IDENTIFIED**

---

## 🎯 Test Results

### Job: bias-detection-1759333011707
**Config**: 1 question × 3 models × 3 regions = 9 executions

**Results**:
- ✅ **US**: 3/3 completed (100%)
- ✅ **EU**: 3/3 completed (100%)
- ❌ **ASIA**: 0/3 completed (0% - all failed)

---

## 🔍 Root Cause

**ASIA provider exists in hybrid router but runner can't use it!**

### Evidence

1. **Hybrid Router Has Provider** ✅
   ```bash
   curl https://project-beacon-production.up.railway.app/providers
   ```
   Returns: `modal-asia-pacific` (healthy)

2. **Runner Can't Find Provider** ❌
   All ASIA executions have `provider_id: ""`

3. **US/EU Work Fine** ✅
   - US executions use `provider_id: "modal-us-east"`
   - EU executions use `provider_id: "modal-eu-west"`

---

## 💡 Why This Happens

The runner's hybrid client is failing to discover or select the ASIA provider even though:
- ✅ Provider exists in hybrid router
- ✅ Provider is marked as healthy
- ✅ Region mapping is correct (`"ASIA"` → `"asia-pacific"`)

**Possible causes**:
1. Hybrid client filters providers by some criteria that ASIA doesn't meet
2. Provider discovery API call fails for ASIA region
3. Provider selection logic has a bug for ASIA
4. Environment variable configuration missing for ASIA

---

## 🔧 Previous Fix Attempts

From memory, we've tried setting multiple environment variables for ASIA:
- `PROVIDER_ASIA_001`
- `PROVIDER_APAC_001`
- `ASIA_PROVIDER_ENDPOINT`
- `PROVIDER_ASIA_PACIFIC_001`

But the issue persists!

---

## 🎯 Next Steps

### Step 1: Check Hybrid Client Code
Look at how the hybrid client discovers and selects providers:
- `internal/hybrid/client.go`
- Provider discovery logic
- Provider filtering logic

### Step 2: Check Runner Logs for ASIA
Search for logs when ASIA execution attempts:
```bash
flyctl logs --app beacon-runner-production | grep -i "asia"
```

### Step 3: Test Hybrid Router Directly
```bash
curl "https://project-beacon-production.up.railway.app/providers?region=asia-pacific"
```

### Step 4: Check Environment Variables
Verify ASIA provider configuration in runner secrets

---

## 📊 Impact

**Good News**:
- ✅ Per-question execution working (9 executions created)
- ✅ Portal UI showing correct counts
- ✅ US and EU providers working perfectly
- ✅ question_id in API responses

**Bad News**:
- ❌ ASIA region completely broken
- ❌ 33% of global coverage lost
- ❌ Can't test cross-region bias detection properly

---

## 🚀 Workaround

For now, users can:
1. Only select US and EU regions
2. Avoid ASIA until this is fixed
3. Still get cross-region comparison (US vs EU)

---

## 📝 Summary

**Problem**: ASIA provider exists but runner can't use it  
**Symptom**: All ASIA executions fail with empty `provider_id`  
**Impact**: 0% success rate for ASIA region  
**Status**: Need to debug hybrid client provider discovery logic

**The good news**: Everything else is working! US and EU are 100% successful.
