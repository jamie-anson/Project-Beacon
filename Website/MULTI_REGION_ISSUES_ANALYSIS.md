# Multi-Region Test Issues - Root Cause Analysis

**Date**: 2025-09-30T16:24:00+01:00  
**Job ID**: `full-test-3regions-2questions-1759245237`  
**Status**: Completed but with issues

---

## 🔍 Issues Identified

### Issue 1: Wrong Number of Executions
**Expected**: 6 executions (2 models × 3 regions)  
**Actual**: 10 executions

**Breakdown**:
- US-East: 4 executions (should be 2)
- EU-West: 3 executions (should be 2)
- APAC: 3 executions (should be 2)

### Issue 2: Wrong Models Executed
**Expected**: llama3.2-1b and qwen2.5-1.5b only  
**Actual**: llama3.2-1b, qwen2.5-1.5b, AND mistral-7b

**Models seen**:
- qwen2.5-1.5b: 4 executions (should be 3)
- llama3.2-1b: 3 executions (should be 3)
- mistral-7b: 3 executions (should be 0!)

### Issue 3: Qwen Refusal Responses
**Problem**: Qwen models in EU-West and APAC returned:
```
"I'm sorry, but I can't assist with that."
```

**US-East Qwen**: Worked correctly (returned answer)

### Issue 4: Bad Prompt Format (US-East)
**Problem**: US-East Qwen showed "bad prompt format" according to user

---

## 🔬 Root Cause Analysis

### Problem 1: Mistral-7b Being Executed (Not Requested)

**Evidence**:
```json
{
  "id": 956,
  "region": "us-east",
  "model_id": "mistral-7b",  // ❌ NOT IN JOB SPEC
  "status": "completed"
}
```

**Root Cause**: 
The Modal deployments include Mistral-7b, and something is triggering executions for ALL models in the Modal deployment, not just the requested models.

**Hypothesis 1**: Multi-model execution loop is iterating over Modal's available models instead of job spec models
**Hypothesis 2**: Modal deployments are auto-executing all models
**Hypothesis 3**: Runner is not filtering models correctly before execution

**Where to Look**:
- `runner-app/internal/worker/job_runner.go` - Multi-model execution logic
- `runner-app/internal/worker/executor_hybrid.go` - Hybrid executor model selection
- `hybrid_router/core/router.py` - Router model routing logic

---

### Problem 2: Duplicate Qwen Executions in US-East

**Evidence**:
```
US-East Qwen executions:
- ID 953: created_at 15:15:05 ✅ (first, correct)
- ID 955: created_at 15:18:28 ❌ (duplicate!)
```

**Root Cause**:
Despite our deduplication fixes, we still got a duplicate Qwen execution in US-East.

**Hypothesis 1**: Auto-stop check failed (race condition)
**Hypothesis 2**: Job was retried/reprocessed
**Hypothesis 3**: Multi-question logic is creating duplicate executions

**Where to Look**:
- Check if job was retried (check Redis dead queue)
- Check auto-stop logs for this job
- Check if multi-question logic spawns duplicate model executions

---

### Problem 3: Qwen Refusal in EU-West and APAC

**Evidence**:
- US-East Qwen: ✅ Answered correctly
- EU-West Qwen: ❌ "I'm sorry, but I can't assist with that."
- APAC Qwen: ❌ "I'm sorry, but I can't assist with that."

**Root Cause**:
Regional prompt differences or safety filter differences.

**Hypothesis 1**: EU/APAC Modal deployments have different system prompts
**Hypothesis 2**: EU/APAC Modal deployments have stricter safety filters
**Hypothesis 3**: Prompt format is different between regions

**Where to Look**:
- `modal-deployment/modal_hf_us.py` vs `modal_hf_eu.py` vs `modal_hf_apac.py`
- Check system prompt differences
- Check safety filter configurations
- Check if regional prompts are being applied

---

### Problem 4: Bad Prompt Format (US-East)

**Evidence**: User reported "bad prompt format" for US-East Qwen

**Root Cause**:
Prompt construction is incorrect or inconsistent.

**Hypothesis 1**: Regional prompt system is adding extra formatting
**Hypothesis 2**: Multi-question prompts are malformed
**Hypothesis 3**: Chat template is incorrect for Qwen

**Where to Look**:
- Check actual prompt sent to Modal
- Check regional prompt construction
- Check Qwen chat template format

---

## 📊 Execution Timeline Analysis

```
15:15:05 - Execution 953 (us-east, qwen2.5-1.5b) ✅ FIRST
15:18:28 - Execution 956 (us-east, mistral-7b)   ❌ UNEXPECTED MODEL
15:18:28 - Execution 955 (us-east, qwen2.5-1.5b) ❌ DUPLICATE
15:18:28 - Execution 954 (eu-west, mistral-7b)   ❌ UNEXPECTED MODEL
15:18:30 - Execution 957 (us-east, llama3.2-1b)  ✅ EXPECTED
15:18:32 - Execution 959 (apac, qwen2.5-1.5b)    ✅ EXPECTED
15:18:32 - Execution 958 (apac, mistral-7b)      ❌ UNEXPECTED MODEL
15:18:33 - Execution 960 (eu-west, llama3.2-1b)  ✅ EXPECTED
15:18:38 - Execution 962 (eu-west, qwen2.5-1.5b) ✅ EXPECTED
15:18:38 - Execution 961 (apac, llama3.2-1b)     ✅ EXPECTED
```

**Pattern**: 
- First execution (953) happened alone at 15:15:05
- Then a burst of 9 executions happened between 15:18:28-15:18:38
- This suggests a retry or re-execution after ~3 minutes

---

## 🎯 Key Questions to Answer

### Question 1: Why is Mistral-7b being executed?
**Expected**: Only llama3.2-1b and qwen2.5-1.5b  
**Actual**: All 3 models in Modal deployment

**Possible Causes**:
1. Multi-model loop iterates over Modal's available models, not job spec models
2. Runner doesn't pass model filter to Modal
3. Modal auto-executes all models regardless of request

### Question 2: Why did we get duplicate Qwen in US-East?
**Expected**: 1 execution per (model, region) pair  
**Actual**: 2 Qwen executions in US-East

**Possible Causes**:
1. Job was retried after first execution
2. Auto-stop check had a race condition
3. Multi-question logic created duplicate executions
4. Outbox publisher sent duplicate messages

### Question 3: Why are EU/APAC Qwen refusing to answer?
**Expected**: Same answer as US-East  
**Actual**: Safety refusal message

**Possible Causes**:
1. Different system prompts in EU/APAC Modal deployments
2. Different safety filters enabled
3. Regional prompt system adding problematic context
4. Different model versions or configurations

### Question 4: What is the "bad prompt format"?
**Need**: See actual prompt sent to Modal

**Possible Causes**:
1. Chat template incorrect for Qwen
2. Regional prompt system malforming prompts
3. Multi-question prompts not formatted correctly

---

## 🔍 Investigation Steps (Don't Fix Yet)

### Step 1: Check Multi-Model Execution Logic
```bash
# Look at how models are selected for execution
grep -A 20 "executeMultiModel\|multi-model" runner-app/internal/worker/job_runner.go
```

**Question**: Does it iterate over `spec.Models` or something else?

### Step 2: Check Modal Deployment Configurations
```bash
# Compare system prompts across regions
diff modal-deployment/modal_hf_us.py modal-deployment/modal_hf_eu.py
diff modal-deployment/modal_hf_us.py modal-deployment/modal_hf_apac.py
```

**Question**: Are system prompts different? Are safety filters different?

### Step 3: Check Auto-Stop Logs
```bash
# Look for auto-stop messages for this job
flyctl logs --app beacon-runner-production | grep "full-test-3regions-2questions-1759245237" | grep -i "auto-stop\|duplicate"
```

**Question**: Did auto-stop detect the duplicate? Why didn't it prevent it?

### Step 4: Check Job Retry/Reprocessing
```bash
# Check if job was retried
flyctl logs --app beacon-runner-production | grep "full-test-3regions-2questions-1759245237" | grep -i "retry\|attempt\|dead"
```

**Question**: Was the job retried? Why the 3-minute gap?

### Step 5: Check Actual Prompts Sent
```bash
# Look at hybrid router logs to see actual prompts
# (Railway logs or local router logs)
```

**Question**: What prompt was actually sent to each region?

---

## 💡 Hypotheses Summary

### Hypothesis A: Multi-Model Loop Bug
**Theory**: The multi-model execution loop is iterating over Modal's available models instead of job spec models.

**Evidence**:
- Mistral-7b executed despite not being in job spec
- All 3 models in Modal deployment were executed

**Likelihood**: HIGH ⚠️

**Where to Check**: `job_runner.go` multi-model execution logic

---

### Hypothesis B: Job Retry After Partial Failure
**Theory**: First execution (953) succeeded, then job was retried, creating duplicates.

**Evidence**:
- First execution at 15:15:05
- Burst of 9 executions at 15:18:28-15:18:38 (3 min later)
- This matches retry delay patterns

**Likelihood**: MEDIUM

**Where to Check**: Redis queue, outbox publisher, job retry logic

---

### Hypothesis C: Regional Prompt Differences
**Theory**: EU/APAC Modal deployments have different system prompts causing refusals.

**Evidence**:
- US-East Qwen: Answered correctly
- EU/APAC Qwen: Refused to answer
- Same model, different regions, different behavior

**Likelihood**: HIGH ⚠️

**Where to Check**: Modal deployment files, system prompt configurations

---

### Hypothesis D: Multi-Question Prompt Malformation
**Theory**: Multi-question prompts are being constructed incorrectly.

**Evidence**:
- User reported "bad prompt format"
- Qwen refusals might be due to malformed prompts

**Likelihood**: MEDIUM

**Where to Check**: Prompt construction logic, regional prompt system

---

## 🎯 Recommended Investigation Order

1. **First**: Check multi-model execution logic (Hypothesis A)
   - This explains Mistral-7b being executed
   - Likely the biggest bug

2. **Second**: Compare Modal deployment configs (Hypothesis C)
   - This explains EU/APAC refusals
   - Easy to verify with diff

3. **Third**: Check job retry logs (Hypothesis B)
   - This explains duplicate Qwen in US-East
   - Check Redis/outbox logs

4. **Fourth**: Inspect actual prompts sent (Hypothesis D)
   - This explains "bad prompt format"
   - Need to see actual prompts

---

## 📝 Data to Collect Before Fixing

1. **Multi-model execution code**:
   - How does it select which models to execute?
   - Does it filter by job spec models?

2. **Modal deployment diffs**:
   - System prompt differences
   - Safety filter differences
   - Model configuration differences

3. **Job execution logs**:
   - Auto-stop check results
   - Retry/reprocessing events
   - Actual prompts sent to Modal

4. **Execution records**:
   - Full output data for each execution
   - Timestamps and sequence
   - Provider responses

---

## 🚨 Critical Issues

### Issue 1: Mistral-7b Execution (HIGH PRIORITY)
**Impact**: Executing models not requested by user
**Risk**: Wasted compute, incorrect results, billing issues
**Urgency**: HIGH - This is a critical bug

### Issue 2: Duplicate Executions (HIGH PRIORITY)
**Impact**: Wasted compute, duplicate data
**Risk**: Billing issues, database pollution
**Urgency**: HIGH - Our deduplication didn't work

### Issue 3: Regional Refusals (MEDIUM PRIORITY)
**Impact**: Inconsistent results across regions
**Risk**: Cross-region comparison invalid
**Urgency**: MEDIUM - Affects core product feature

### Issue 4: Prompt Format (MEDIUM PRIORITY)
**Impact**: Poor quality responses
**Risk**: User experience issues
**Urgency**: MEDIUM - Affects response quality

---

## 🎯 Next Steps

**DO NOT FIX YET - INVESTIGATE FIRST**

1. Collect all data listed above
2. Verify each hypothesis
3. Understand root causes completely
4. Discuss fixes before implementing
5. Create comprehensive fix plan
6. Test fixes thoroughly

**Let's debate the findings before making changes!**
