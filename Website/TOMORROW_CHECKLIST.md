# Tomorrow's Implementation Checklist
**Date**: 2025-10-02
**Goal**: Fix Modal HTTP 303 errors with sequential question batching

---

## ✅ Pre-Flight Status

**Good news - Most infrastructure already ready!**
- ✅ `ExecutionResult` struct has `QuestionID` field
- ✅ `ExecutionResult` struct has `ModelID` field  
- ✅ Semaphore size is 10 (sufficient for 6 concurrent)
- ✅ Database method `InsertExecutionWithModelAndQuestion()` exists
- ✅ `executeSingleRegion()` signature correct

**Work needed**:
- ⚠️ Remove Mistral 7B from Modal deployments
- ⚠️ Add sequential question loop to `executeMultiModelJob()`
- ⚠️ Add Modal cancellation handling

---

## 🎯 Morning Workflow (4.5-5.5 hours)

### ☕ Phase 1A: Model Cleanup (30 min) - ✅ COMPLETE

**Files edited**:
```
✅ /modal-deployment/modal_hf_us.py
✅ /modal-deployment/modal_hf_eu.py
✅ /modal-deployment/modal_hf_apac.py
```

**Task**: Remove `"mistral-7b": {...}` block from `MODEL_REGISTRY` in each file

**Deployed**:
```bash
✅ modal deploy modal_hf_us.py - SUCCESS
✅ modal deploy modal_hf_eu.py - SUCCESS
✅ modal deploy modal_hf_apac.py - SUCCESS
```

**Verified**: ✅ All endpoints return only 2 models (llama3.2-1b, qwen2.5-1.5b)
- US: https://jamie-anson--project-beacon-hf-us-models.modal.run
- EU: https://jamie-anson--project-beacon-hf-eu-health.modal.run
- APAC: https://jamie-anson--project-beacon-hf-apac-health.modal.run

---

### 🔧 Phase 1B: Sequential Question Batching (3-4 hours)

**File**: `/runner-app/internal/worker/job_runner.go`

**Find** (line ~318): The double for-loop executing models × regions

**Replace with**: Sequential question loop + parallel model×region execution per question

**Key changes**:
1. Add outer `for questionIdx, question := range spec.Questions`
2. Add `var questionWg sync.WaitGroup` for each question batch
3. Set `result.QuestionID = q` in goroutine
4. Add `questionWg.Wait()` between questions
5. Add logging for question batch start/complete

**Build & test**:
```bash
cd runner-app
go build ./...
go test ./internal/worker -v
```

---

### 🛡️ Phase 1C: Modal Cancellation (1 hour)

**File**: `/runner-app/internal/worker/job_runner.go`

**Add** (before `executor.Execute()` call):
- Context cancellation check with `select` statement
- Early return if context already cancelled

**Add** (after `executor.Execute()` call):
- Log warning if context was cancelled mid-execution

**Verify**: Hybrid client uses `NewRequestWithContext()`

---

### 🚀 Phase 1D: Deploy & Validate (1-2 hours)

**Deploy**:
```bash
cd runner-app
flyctl deploy
flyctl logs -a beacon-runner-change-me --follow
```

**Submit test job**: 2 questions × 2 models × 3 regions

**Watch for**:
- "starting question batch" (2 times)
- "question batch completed" (2 times)
- Gap between Q1→Q2 (<2 seconds)
- All 12 executions complete

**Check Modal dashboard**:
- Max 6 containers during Q1
- Same 6 containers reused for Q2

**Verify database**:
```sql
SELECT job_id, model_id, question_id, region, status 
FROM executions 
WHERE job_id = 'YOUR_TEST_JOB_ID'
ORDER BY created_at;
```

Expected: 12 rows with question_id populated

---

## 🎯 Success Indicators

1. ✅ Modal shows only 2 models per region
2. ✅ Code compiles without errors
3. ✅ Exactly 6 Modal containers during Q1 execution
4. ✅ Same 6 container IDs reused for Q2
5. ✅ Gap timing <2s between Q1 complete and Q2 start
6. ✅ All 12 executions complete (US: 4/4, EU: 4/4, ASIA: 4/4)
7. ✅ Database has question_id populated
8. ✅ No HTTP 303 errors

---

## 📚 Reference Documents

- **PREFLIGHT_CHECKS.md** - Detailed pre-flight verification results
- **IMPLEMENTATION_GUIDE.md** - Step-by-step code changes
- **MODAL_OPTIMIZATION_PLAN.md** - Full technical plan

---

## 🔥 Quick Start Commands

```bash
# 1. Remove Mistral from Modal
cd /Users/Jammie/Desktop/Project\ Beacon/Website/modal-deployment
# Edit modal_hf_us.py, modal_hf_eu.py, modal_hf_apac.py
modal deploy modal_hf_us.py
modal deploy modal_hf_eu.py
modal deploy modal_hf_apac.py

# 2. Modify runner code
cd /Users/Jammie/Desktop/Project\ Beacon/runner-app
# Edit internal/worker/job_runner.go (see IMPLEMENTATION_GUIDE.md)
go build ./...
go test ./internal/worker -v

# 3. Deploy runner
flyctl deploy
flyctl logs -a beacon-runner-change-me --follow

# 4. Test
# Submit 2-question job via portal
# Monitor logs and Modal dashboard
```

---

## 🚨 If Something Goes Wrong

**Code won't compile**:
- Check syntax in the new question loop
- Verify all variables are declared
- Run `go build ./...` to see specific errors

**Tests fail**:
- Check test expectations match new behavior
- Verify mock executor handles question batching
- Run `go test ./internal/worker -v` for details

**HTTP 303 errors persist**:
- Check Modal dashboard: Are there >6 containers?
- Verify question batching is actually sequential (check logs)
- May need to implement Phase 2 (unified routing) first

**Rollback**:
```bash
git checkout HEAD -- runner-app/internal/worker/job_runner.go
cd runner-app && flyctl deploy
```

---

**Let's ship this! 🚀**

**First thing tomorrow**: Remove Mistral from Modal deployments (30 min)
