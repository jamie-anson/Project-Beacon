# Phase 1 Implementation Guide
**Date**: 2025-10-02
**Estimated Time**: 4.5-5.5 hours

---

## 🎯 Implementation Order

### Phase 1A: Model Cleanup (30 min) - START HERE

#### Step 1: Remove Mistral from Modal Deployments

**Files to edit**:
1. `/modal-deployment/modal_hf_us.py`
2. `/modal-deployment/modal_hf_eu.py`
3. `/modal-deployment/modal_hf_apac.py`

**Change in each file** (around line 43-65):

**REMOVE this block**:
```python
"mistral-7b": {
    "hf_model": "mistralai/Mistral-7B-Instruct-v0.3", 
    "gpu": "T4",
    "memory_gb": 12,
    "context_length": 32768,
    "description": "Strong 7B parameter general-purpose model (8-bit on T4)"
},
```

**Keep only**:
```python
MODEL_REGISTRY = {
    "llama3.2-1b": {
        "hf_model": "meta-llama/Llama-3.2-1B-Instruct",
        "gpu": "T4",
        "memory_gb": 8,
        "context_length": 128000,
        "description": "Fast 1B parameter model for quick inference"
    },
    "qwen2.5-1.5b": {
        "hf_model": "Qwen/Qwen2.5-1.5B-Instruct",
        "gpu": "T4", 
        "memory_gb": 8,
        "context_length": 32768,
        "description": "Efficient 1.5B parameter model"
    }
}
```

#### Step 2: Deploy Modal Apps

```bash
cd /Users/Jammie/Desktop/Project\ Beacon/Website/modal-deployment

# Deploy US region
modal deploy modal_hf_us.py

# Deploy EU region
modal deploy modal_hf_eu.py

# Deploy APAC region
modal deploy modal_hf_apac.py
```

#### Step 3: Verify Deployments

```bash
# Test US endpoint
curl https://jamie-anson--project-beacon-hf-us-inference.modal.run/models

# Test EU endpoint
curl https://jamie-anson--project-beacon-hf-eu-inference.modal.run/models

# Test APAC endpoint
curl https://jamie-anson--project-beacon-hf-apac-inference.modal.run/models

# Expected response (each):
# {"models_available": ["llama3.2-1b", "qwen2.5-1.5b"], ...}
```

---

### Phase 1B: Sequential Question Batching (3-4 hours)

#### Step 1: Modify executeMultiModelJob()

**File**: `/runner-app/internal/worker/job_runner.go`

**Find** (around line 318):
```go
// Execute each model in each of its regions with bounded concurrency
for _, model := range spec.Models {
    for _, region := range model.Regions {
        wg.Add(1)
        sem <- struct{}{} // Acquire semaphore slot
        go func(m models.ModelSpec, r string) {
            defer wg.Done()
            defer func() { <-sem }()
            
            // ... existing code ...
            
            result := w.executeSingleRegion(ctx, jobID, &modelSpec, r, executor)
            result.ModelID = m.ID
            
            resultsMu.Lock()
            results = append(results, result)
            resultsMu.Unlock()
        }(model, region)
    }
}
```

**Replace with**:
```go
// Execute questions sequentially to avoid overwhelming Modal
for questionIdx, question := range spec.Questions {
    l.Info().
        Str("job_id", jobID).
        Str("question", question).
        Int("question_num", questionIdx+1).
        Int("total_questions", len(spec.Questions)).
        Msg("starting question batch")
    
    var questionWg sync.WaitGroup
    var questionResults []ExecutionResult
    
    // Execute all models × regions for this question in parallel
    // NOTE: Only 2 models (llama3.2-1b, qwen2.5-1.5b) - mistral dropped
    for _, model := range spec.Models {
        for _, region := range model.Regions {
            questionWg.Add(1)
            sem <- struct{}{} // Acquire semaphore slot
            
            go func(m models.ModelSpec, r string, q string) {
                defer questionWg.Done()
                defer func() { <-sem }()
                
                // Create modified spec with single question
                modelSpec := *spec
                modelSpec.Questions = []string{q}
                
                // Copy metadata safely
                newMetadata := make(map[string]interface{})
                for k, v := range spec.Metadata {
                    newMetadata[k] = v
                }
                modelSpec.Metadata = newMetadata
                modelSpec.Metadata["model_id"] = m.ID
                modelSpec.Metadata["model_name"] = m.Name
                modelSpec.Metadata["question_id"] = q
                
                result := w.executeSingleRegion(ctx, jobID, &modelSpec, r, executor)
                result.ModelID = m.ID
                result.QuestionID = q
                
                resultsMu.Lock()
                questionResults = append(questionResults, result)
                resultsMu.Unlock()
                
                l.Debug().
                    Str("job_id", jobID).
                    Str("model_id", m.ID).
                    Str("region", r).
                    Str("question", q).
                    Str("status", result.Status).
                    Msg("model-region-question execution completed")
            }(model, region, question)
        }
    }
    
    // Wait for this question batch to complete before moving to next
    questionWg.Wait()
    
    // Append question results to overall results
    resultsMu.Lock()
    results = append(results, questionResults...)
    resultsMu.Unlock()
    
    l.Info().
        Str("job_id", jobID).
        Str("question", question).
        Int("executions", len(questionResults)).
        Msg("question batch completed")
}
```

#### Step 2: Build and Test

```bash
cd /Users/Jammie/Desktop/Project\ Beacon/runner-app

# Build
go build ./...

# Run tests
go test ./internal/worker -v

# Check for errors
echo $?  # Should be 0
```

---

### Phase 1C: Modal Cancellation (1 hour)

#### Step 1: Add Cancellation to executeSingleRegion()

**File**: `/runner-app/internal/worker/job_runner.go`

**Find** (around line 585):
```go
providerID, status, outputJSON, receiptJSON, err := executor.Execute(ctx, &singleQuestionSpec, region)
```

**Add before this line**:
```go
// Check if context is already cancelled before executing
select {
case <-ctx.Done():
    l.Warn().
        Str("job_id", jobID).
        Str("region", region).
        Str("model_id", modelID).
        Str("question_id", questionID).
        Msg("execution cancelled before start - context deadline exceeded")
    return ExecutionResult{
        Region:      region,
        Status:      "cancelled",
        ModelID:     modelID,
        QuestionID:  questionID,
        Error:       ctx.Err(),
        StartedAt:   time.Now(),
        CompletedAt: time.Now(),
    }
default:
    // Continue with execution
}
```

**After the Execute() call, add**:
```go
// Log if execution was cancelled mid-flight
if ctx.Err() != nil {
    l.Warn().
        Str("job_id", jobID).
        Str("region", region).
        Str("model_id", modelID).
        Str("question_id", questionID).
        Err(ctx.Err()).
        Msg("execution cancelled - Modal should auto-cleanup on connection close")
}
```

#### Step 2: Verify Hybrid Client Context Handling

**File**: `/runner-app/internal/hybrid/client.go`

**Check that HTTP requests use the context**:
```go
req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
```

If not using `NewRequestWithContext`, update to use it.

---

### Phase 1D: Database Migration (30 min)

#### Step 1: Check if question_id column exists

```bash
# Connect to database
psql $DATABASE_URL

# Check schema
\d executions

# Look for question_id column
```

**If column missing**, create migration:

**File**: `/runner-app/migrations/0009_add_question_id.up.sql`
```sql
-- Add question_id column for per-question execution tracking
ALTER TABLE executions ADD COLUMN IF NOT EXISTS question_id VARCHAR(255);

-- Add index for fast queries
CREATE INDEX IF NOT EXISTS idx_executions_question_id ON executions(question_id);
```

**File**: `/runner-app/migrations/0009_add_question_id.down.sql`
```sql
-- Rollback: Remove question_id column
DROP INDEX IF EXISTS idx_executions_question_id;
ALTER TABLE executions DROP COLUMN IF EXISTS question_id;
```

#### Step 2: Run migration

```bash
# Run migration (adjust command based on your migration tool)
# Example with golang-migrate:
migrate -path ./migrations -database "$DATABASE_URL" up
```

---

## 🧪 Testing Plan

### Test 1: Local Build
```bash
cd runner-app
go build ./...
go test ./internal/worker -v
```

### Test 2: Deploy to Fly.io
```bash
cd runner-app
flyctl deploy
flyctl logs -a beacon-runner-change-me --follow
```

### Test 3: Submit Test Job
```bash
# Submit 2-question, 2-model job
# Expected: 12 executions (2 questions × 2 models × 3 regions)

# Monitor logs for:
# - "starting question batch" (should see 2 times)
# - "question batch completed" (should see 2 times)
# - Gap between Q1 complete and Q2 start (<2s)
```

### Test 4: Verify Database
```bash
psql $DATABASE_URL -c "
SELECT 
    job_id, 
    model_id, 
    question_id, 
    region, 
    status,
    created_at
FROM executions 
WHERE job_id = 'YOUR_TEST_JOB_ID'
ORDER BY created_at;
"

# Expected: 12 rows with question_id populated
```

### Test 5: Check Modal Dashboard
- Go to Modal dashboard
- Check "Containers" tab during Q1 execution
- **Expected**: Max 6 running containers (2 per region)
- **Expected**: Same 6 containers reused for Q2

### Test 6: Timeout Cancellation Test
```bash
# Submit job with 1s timeout (will fail)
# Verify Modal containers stop immediately
# Check logs for "execution cancelled" messages
```

---

## 📊 Success Criteria

- [ ] Modal deployments show only 2 models (llama, qwen)
- [ ] Code compiles without errors
- [ ] All tests pass
- [ ] Job logs show sequential question batching
- [ ] Gap between questions <2s
- [ ] All 12 executions complete successfully
- [ ] Modal dashboard shows max 6 concurrent containers
- [ ] Same containers reused for Q2 (check container IDs)
- [ ] Database has question_id populated
- [ ] Timeout cancellation working

---

## 🔧 Debug Commands

```bash
# Pre-flight verification
grep -n "ExecutionResult" runner-app/internal/worker/job_runner.go
grep -n "make(chan struct" runner-app/internal/worker/job_runner.go
grep -n "QuestionID" runner-app/pkg/models/

# Build verification
cd runner-app && go build ./...
cd runner-app && go test ./internal/worker -v

# Deployment
cd runner-app && flyctl deploy
flyctl logs -a beacon-runner-change-me --follow

# Database verification
psql $DATABASE_URL -c "\d executions"
psql $DATABASE_URL -c "SELECT job_id, model_id, question_id, status FROM executions WHERE job_id='test-job-id' ORDER BY created_at;"

# Modal verification
# Check Modal dashboard manually
# Expected: Max 6 running containers during Q1 (2 per region)
# Models: llama3.2-1b and qwen2.5-1.5b only (mistral removed)
```

---

## 🚨 Rollback Plan

If Phase 1 fails:

1. **Revert job_runner.go changes**:
   ```bash
   git checkout HEAD -- runner-app/internal/worker/job_runner.go
   ```

2. **Redeploy runner**:
   ```bash
   cd runner-app && flyctl deploy
   ```

3. **Re-add Mistral to Modal** (if needed):
   ```bash
   git checkout HEAD -- modal-deployment/modal_hf_*.py
   modal deploy modal-deployment/modal_hf_us.py
   modal deploy modal-deployment/modal_hf_eu.py
   modal deploy modal-deployment/modal_hf_apac.py
   ```

---

**Ready to implement! Start with Phase 1A tomorrow morning.** 🚀
