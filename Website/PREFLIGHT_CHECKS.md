# Pre-Flight Checks - Phase 1 Implementation
**Date**: 2025-10-02
**Task**: Sequential Question Batching + Modal Cancellation

---

## ✅ STEP 0: Pre-Flight Verification Results

### 1. ExecutionResult Struct - ✅ READY
**Location**: `/runner-app/internal/worker/job_runner.go` (lines 287-299)

```go
type ExecutionResult struct {
    Region      string
    ProviderID  string
    Status      string
    OutputJSON  []byte
    ReceiptJSON []byte
    Error       error
    StartedAt   time.Time
    CompletedAt time.Time
    ExecutionID int64
    ModelID     string     // ✅ Already present
    QuestionID  string     // ✅ Already present
}
```

**Status**: ✅ **Both fields already exist!** No changes needed.

---

### 2. Semaphore Size - ⚠️ NEEDS UPDATE
**Location**: `/runner-app/internal/worker/job_runner.go` (line 69)

**Current**:
```go
maxConcurrent: 10, // Default bounded concurrency limit
```

**Current semaphore initialization** (line 309):
```go
sem := make(chan struct{}, w.maxConcurrent) // Semaphore for bounded concurrency
```

**Analysis**:
- Current size: **10** ✅
- Required for 2-model batching: **≥6**
- Recommended: **8** (33% buffer)
- **Decision**: Keep at 10 (provides 67% buffer, even better)

**Status**: ✅ **Current size (10) is sufficient!** No changes needed.

---

### 3. executeSingleRegion() Signature - ✅ VERIFIED
**Location**: `/runner-app/internal/worker/job_runner.go` (line 467)

```go
func (w *JobRunner) executeSingleRegion(
    ctx context.Context, 
    jobID string, 
    spec *models.JobSpec, 
    region string, 
    executor Executor
) ExecutionResult
```

**Status**: ✅ **Signature confirmed.** Returns `ExecutionResult` as expected.

---

### 4. Database Support - ✅ READY
**Location**: `/runner-app/internal/store/executions_repo.go`

**Method exists**:
```go
func (r *ExecutionsRepo) InsertExecutionWithModelAndQuestion(
    ctx context.Context,
    jobspecID string,
    providerID string,
    region string,
    status string,
    startedAt time.Time,
    completedAt time.Time,
    outputJSON []byte,
    receiptJSON []byte,
    modelID string,
    questionID string,  // ✅ Already supports question_id
) (int64, error)
```

**Status**: ✅ **Database method already supports question_id!**

---

### 5. Current Multi-Model Execution - 📍 NEEDS MODIFICATION
**Location**: `/runner-app/internal/worker/job_runner.go` (lines 318-372)

**Current implementation**:
```go
// Execute each model in each of its regions with bounded concurrency
for _, model := range spec.Models {
    for _, region := range model.Regions {
        wg.Add(1)
        sem <- struct{}{} // Acquire semaphore slot
        go func(m models.ModelSpec, r string) {
            defer wg.Done()
            defer func() { <-sem }() // Release semaphore slot
            
            // Execute model-region combination
            result := w.executeSingleRegion(ctx, jobID, &modelSpec, r, executor)
            result.ModelID = m.ID
            
            resultsMu.Lock()
            results = append(results, result)
            resultsMu.Unlock()
        }(model, region)
    }
}
```

**Issue**: All model-region combinations execute in parallel (12 concurrent for 2 models × 3 regions × 2 questions)

**Required change**: Add outer question loop to batch by question

---

### 6. Modal Deployments - ⚠️ NEEDS CLEANUP
**Files to modify**:
- `/modal-deployment/modal_hf_us.py`
- `/modal-deployment/modal_hf_eu.py`
- `/modal-deployment/modal_hf_apac.py`

**Current MODEL_REGISTRY** (from modal_hf_eu.py lines 43-65):
```python
MODEL_REGISTRY = {
    "llama3.2-1b": {...},
    "mistral-7b": {...},      # ❌ REMOVE
    "qwen2.5-1.5b": {...}
}
```

**Required**: Remove `mistral-7b` from all 3 regional deployments

---

## 📋 Implementation Checklist

### Phase 1A: Model Cleanup (30 min)
- [ ] Remove Mistral 7B from `modal_hf_us.py` MODEL_REGISTRY
- [ ] Remove Mistral 7B from `modal_hf_eu.py` MODEL_REGISTRY  
- [ ] Remove Mistral 7B from `modal_hf_apac.py` MODEL_REGISTRY
- [ ] Deploy all 3 Modal apps: `modal deploy modal_hf_*.py`
- [ ] Verify only 2 models available via `/models` endpoint

### Phase 1B: Sequential Question Batching (3-4 hours)
- [ ] Modify `executeMultiModelJob()` in job_runner.go
- [ ] Add outer question loop with sequential execution
- [ ] Add inner model×region parallel execution per question
- [ ] Ensure `result.QuestionID = q` is set (already in struct ✅)
- [ ] Add question batch logging
- [ ] Test locally with mock executor

### Phase 1C: Modal Cancellation (1 hour)
- [ ] Add context cancellation handling in `executeSingleRegion()`
- [ ] Verify hybrid client respects context cancellation
- [ ] Add timeout cancellation logging
- [ ] Test with 1s timeout job

### Phase 1D: Build & Test (30 min)
- [ ] Run: `go build ./...`
- [ ] Run: `go test ./internal/worker -v`
- [ ] Fix any compilation errors
- [ ] Verify all tests pass

---

## 🎯 Key Findings Summary

| Item | Status | Action Required |
|------|--------|-----------------|
| ExecutionResult.QuestionID | ✅ Ready | None - already exists |
| ExecutionResult.ModelID | ✅ Ready | None - already exists |
| Semaphore size | ✅ Ready | None - 10 is sufficient |
| executeSingleRegion() | ✅ Ready | None - signature correct |
| Database support | ✅ Ready | None - method exists |
| Modal deployments | ⚠️ Needs work | Remove Mistral 7B |
| executeMultiModelJob() | ⚠️ Needs work | Add question batching |
| Modal cancellation | ⚠️ Needs work | Add context handling |

---

## 🚀 Ready to Proceed

**Good news**: Most infrastructure already in place!
- ✅ Struct fields exist
- ✅ Database supports question_id
- ✅ Semaphore size sufficient
- ✅ Method signatures correct

**Work required**:
1. Remove Mistral 7B from Modal deployments (30 min)
2. Add sequential question loop to executeMultiModelJob() (3-4 hours)
3. Add Modal cancellation handling (1 hour)

**Total estimated time**: 4.5-5.5 hours

---

**Next step**: Start with Phase 1A (Model Cleanup) tomorrow morning! 🎯
