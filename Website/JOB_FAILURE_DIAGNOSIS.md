# Job Failure Diagnosis - bias-detection-1759330371966

**Date**: 2025-10-01T16:13:00+01:00  
**Status**: 🔍 **INVESTIGATING**

---

## ✅ What's Working

1. **API Fix Deployed** ✅
   - `question_id` now appears in API responses
   - Portal shows per-question breakdown correctly
   - UI displays "4 questions × 3 models × 3 regions"

2. **Hybrid Router Healthy** ✅
   - 3 providers available (US, EU, ASIA)
   - All providers showing as healthy
   - Router responding to health checks

3. **Runner Configuration** ✅
   - `HYBRID_BASE` and `HYBRID_ROUTER_URL` configured
   - Runner has access to hybrid router

---

## ❌ What's Failing

**All 20 executions failed with:**
- `status: "failed"`
- `output_data: null` (no error details)
- Empty `provider_id`
- All have correct `question_id` and `model_id`

**This indicates early failures** - jobs are failing before execution even starts.

---

## 🔍 Investigation Results

### Check 1: API Response
```bash
curl "https://beacon-runner-change-me.fly.dev/api/v1/jobs/bias-detection-1759330371966?include=executions"
```

**Result**: 20 executions, all failed, all with `question_id` ✅

### Check 2: Hybrid Router
```bash
curl "https://project-beacon-production.up.railway.app/health"
```

**Result**:
```json
{
  "status": "healthy",
  "providers_total": 3,
  "providers_healthy": 3,
  "regions": ["asia-pacific", "eu-west", "us-east"]
}
```

### Check 3: Provider List
```bash
curl "https://project-beacon-production.up.railway.app/providers"
```

**Result**: 3 Modal providers, all healthy, all regions covered

### Check 4: Runner Secrets
```bash
flyctl secrets list --app beacon-runner-change-me
```

**Result**: `HYBRID_BASE`, `HYBRID_ROUTER_URL`, and provider endpoints configured

---

## 🤔 Possible Causes

### 1. Runner Can't Connect to Hybrid Router
Even though router is healthy, runner might not be able to reach it.

**Test**: Check runner logs for connection errors

### 2. Provider Discovery Failing
Runner might be failing to discover providers from the hybrid router.

**Test**: Check if runner is calling `/providers` endpoint

### 3. Execution Logic Error
The per-question execution code might have a bug causing early failures.

**Test**: Check runner logs for execution errors

### 4. Database Migration Issue
The `question_id` column might not have the right constraints.

**Test**: Check database schema

---

## 🔧 Next Steps

### Step 1: Check Runner Logs (Need to do this)
```bash
# Look for errors around job creation time (14:53:00 UTC)
flyctl logs --app beacon-runner-change-me | grep -A 5 "14:5[3-5]"
```

### Step 2: Check if Runner is Using Hybrid Router
```bash
# Look for hybrid router calls
flyctl logs --app beacon-runner-change-me | grep "hybrid"
```

### Step 3: Check Early Failure Reason
```bash
# Look for RecordEarlyFailure calls
flyctl logs --app beacon-runner-change-me | grep "early failure"
```

### Step 4: Test Provider Connectivity
```bash
# SSH into runner and test
flyctl ssh console --app beacon-runner-change-me
curl https://project-beacon-production.up.railway.app/providers
```

---

## 📊 Execution Pattern

**Expected**: 4 questions × 3 models × 3 regions = 36 executions  
**Actual**: 20 executions (all failed)

**Why 20 instead of 36?**
- Possible duplicates from retries
- Or job stopped after 20 failures
- Or some executions weren't created

---

## 💡 Hypothesis

**Most Likely**: Runner is failing to connect to hybrid router or discover providers, causing all executions to fail early before they can even start.

**Evidence**:
- All executions have empty `provider_id`
- All executions have no `output_data`
- All executions failed at similar times (within 2 minutes)

**Next**: Need to check runner logs to confirm this hypothesis.

---

## 🎯 Action Items

1. ⏳ **Check runner logs** for connection errors
2. ⏳ **Verify hybrid router connectivity** from runner
3. ⏳ **Check provider discovery** logic
4. ⏳ **Test a simple 1-question job** to isolate the issue

---

## 📝 Summary

**Good News**: 
- ✅ API fix deployed and working
- ✅ Portal UI showing per-question breakdown
- ✅ Hybrid router healthy with 3 providers

**Bad News**:
- ❌ All executions failing early (before execution starts)
- ❌ No error details in output_data
- ❌ Empty provider_id suggests provider discovery failing

**Next**: Need to check runner logs to find the actual error message.
