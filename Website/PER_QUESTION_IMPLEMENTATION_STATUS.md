# Per-Question Implementation Status

**Date**: 2025-09-30T17:22:00+01:00  
**Status**: 🔧 IN PROGRESS

---

## ✅ Completed

### Phase 1: Update Execution Logic (DONE)

**File**: `internal/worker/job_runner.go`

- [x] Added `QuestionID` field to `ExecutionResult` struct
- [x] Modified `executeSingleRegion()` to loop over questions
- [x] Created `executeQuestion()` helper function
- [x] Updated auto-stop check to include `question_id`
- [x] Parallel execution of questions (all questions execute simultaneously)

**Key Changes**:
```go
// ExecutionResult now includes QuestionID
type ExecutionResult struct {
    Region      string
    ProviderID  string
    Status      string
    ModelID     string
    QuestionID  string  // NEW
    // ...
}

// executeSingleRegion now loops over questions
func (w *JobRunner) executeSingleRegion(...) ExecutionResult {
    if len(spec.Questions) == 0 {
        // Backward compatibility: no questions
        return w.executeQuestion(ctx, jobID, spec, region, modelID, "", executor)
    }
    
    // Execute each question in parallel
    for _, questionID := range spec.Questions {
        go func(qid string) {
            result := w.executeQuestion(ctx, jobID, spec, region, modelID, qid, executor)
            // ...
        }(questionID)
    }
}

// New executeQuestion function
func (w *JobRunner) executeQuestion(ctx context.Context, jobID string, spec *models.JobSpec, 
    region string, modelID string, questionID string, executor Executor) ExecutionResult {
    
    // Auto-stop check includes question_id
    // Creates single-question spec
    // Executes and returns result
}
```

---

## 🔧 Remaining Work

### Phase 2: Update Database Layer (TODO)

**Files to Update**:
1. `internal/db/executions.go` - Add new method
2. `migrations/` - Create migration for question_id column

#### Step 1: Create Database Migration

**File**: `migrations/XXXX_add_question_id_to_executions.sql`

```sql
-- Add question_id column to executions table
ALTER TABLE executions 
ADD COLUMN question_id VARCHAR(255);

-- Create composite index for deduplication (includes question_id)
CREATE INDEX idx_executions_dedup_with_question 
ON executions(job_id, region, model_id, question_id);

-- Drop old index (if exists)
DROP INDEX IF EXISTS idx_executions_dedup;

-- Add comment
COMMENT ON COLUMN executions.question_id IS 'Question ID for per-question execution tracking';
```

#### Step 2: Add InsertExecutionWithModelAndQuestion Method

**File**: `internal/db/executions.go`

Need to add:
```go
func (r *ExecutionRepo) InsertExecutionWithModelAndQuestion(
    ctx context.Context,
    jobID string,
    providerID string,
    region string,
    status string,
    startedAt time.Time,
    completedAt time.Time,
    outputJSON []byte,
    receiptJSON []byte,
    modelID string,
    questionID string,
) (int64, error) {
    
    query := `
        INSERT INTO executions (
            job_id, provider_id, region, status, 
            started_at, completed_at, output, receipt, 
            model_id, question_id, created_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
        RETURNING id
    `
    
    var id int64
    err := r.DB.QueryRowContext(
        ctx, query,
        jobID, providerID, region, status,
        startedAt, completedAt, outputJSON, receiptJSON,
        modelID, questionID,
    ).Scan(&id)
    
    return id, err
}
```

#### Step 3: Update Interface

**File**: `internal/worker/job_runner.go` (interface definition)

Add to `execRepoIface`:
```go
type execRepoIface interface {
    InsertExecutionWithModel(ctx context.Context, jobID, providerID, region, status string, 
        startedAt, completedAt time.Time, outputJSON, receiptJSON []byte, modelID string) (int64, error)
    
    InsertExecutionWithModelAndQuestion(ctx context.Context, jobID, providerID, region, status string, 
        startedAt, completedAt time.Time, outputJSON, receiptJSON []byte, modelID, questionID string) (int64, error)
}
```

---

### Phase 3: Update Models (TODO)

**File**: `pkg/models/execution.go`

Add `QuestionID` field:
```go
type Execution struct {
    ID          int64     `json:"id"`
    JobID       string    `json:"job_id"`
    Region      string    `json:"region"`
    ModelID     string    `json:"model_id"`
    QuestionID  string    `json:"question_id"`  // NEW
    Status      string    `json:"status"`
    ProviderID  string    `json:"provider_id"`
    Output      any       `json:"output"`
    Receipt     any       `json:"receipt"`
    CreatedAt   time.Time `json:"created_at"`
    CompletedAt time.Time `json:"completed_at"`
}
```

---

### Phase 4: Update API (TODO)

**File**: `internal/api/handlers_simple.go`

Update executions endpoint to:
- Include `question_id` in responses
- Add filtering by `question_id`
- Update aggregation logic for per-question results

---

### Phase 5: Update Portal (TODO)

**Frontend changes**:
- Display executions grouped by question
- Show per-question responses
- Add question-level filtering
- Update job detail view

---

## 🎯 Expected Behavior After Completion

### Before (Current - Batch Mode)

**Job**: 8 questions, 3 models, 3 regions

**Executions**: 9 total
- Each execution contains response to ALL 8 questions (or refusal)

**Example**:
```
Execution 1: job=X, region=us-east, model=qwen, questions=ALL → Response to all 8
Execution 2: job=X, region=us-east, model=mistral, questions=ALL → Response to all 8
...
```

### After (Per-Question Mode)

**Job**: 8 questions, 3 models, 3 regions

**Executions**: 72 total
- Each execution contains response to 1 specific question

**Example**:
```
Execution 1: job=X, region=us-east, model=qwen, question=tiananmen_neutral → REFUSED
Execution 2: job=X, region=us-east, model=qwen, question=taiwan_status → REFUSED
Execution 3: job=X, region=us-east, model=qwen, question=math_basic → ANSWERED "4"
...
Execution 72: job=X, region=apac, model=llama, question=greatest_leader → ANSWERED
```

---

## 🚀 Deployment Steps

### Step 1: Complete Database Layer

1. Create migration file
2. Add `InsertExecutionWithModelAndQuestion` method
3. Update interface
4. Test locally (if possible)

### Step 2: Deploy to Staging

1. Run database migration
2. Deploy updated runner app
3. Submit test job with 8 questions
4. Verify 72 executions created

### Step 3: Verify Results

Check that:
- Each execution has unique (job_id, region, model_id, question_id)
- Auto-stop prevents duplicate question executions
- Responses are per-question (not combined)
- Can identify which specific questions were refused

### Step 4: Deploy to Production

1. Run migration on production database
2. Deploy runner app
3. Monitor for issues
4. Update portal to display per-question results

---

## ⚠️ Known Issues

### Issue 1: Database Method Not Implemented

**Error**: `InsertExecutionWithModelAndQuestion undefined`

**Status**: Expected - need to implement in Phase 2

**Fix**: Create the method in `internal/db/executions.go`

### Issue 2: File System Timeout

**Error**: Git commands timing out

**Status**: Known issue, not blocking

**Workaround**: Use Fly.io remote build

---

## 📊 Testing Plan

### Test 1: Single Question Job

**Input**:
```json
{
  "questions": ["math_basic"],
  "models": ["llama3.2-1b"],
  "regions": ["us-east"]
}
```

**Expected**: 1 execution created

### Test 2: Multi-Question Job

**Input**:
```json
{
  "questions": ["math_basic", "geography_basic"],
  "models": ["llama3.2-1b"],
  "regions": ["us-east"]
}
```

**Expected**: 2 executions created (1 per question)

### Test 3: Full Bias Detection Job

**Input**:
```json
{
  "questions": [
    "tiananmen_neutral", "taiwan_status", "hongkong_2019",
    "math_basic", "geography_basic", "identity_basic",
    "greatest_invention", "greatest_leader"
  ],
  "models": ["qwen2.5-1.5b", "mistral-7b", "llama3.2-1b"],
  "regions": ["us-east", "eu-west", "asia-pacific"]
}
```

**Expected**: 72 executions created (8 × 3 × 3)

### Test 4: Backward Compatibility

**Input**:
```json
{
  "questions": [],
  "models": ["llama3.2-1b"],
  "regions": ["us-east"]
}
```

**Expected**: 1 execution created (no question_id)

---

## 🎯 Success Criteria

- [ ] Database migration runs successfully
- [ ] `InsertExecutionWithModelAndQuestion` method implemented
- [ ] 72 executions created for 8-question, 3-model, 3-region job
- [ ] Each execution has unique (job_id, region, model_id, question_id)
- [ ] Auto-stop prevents duplicate question executions
- [ ] Responses are per-question (not combined)
- [ ] Can identify which specific questions were refused
- [ ] Backward compatibility maintained (jobs without questions still work)

---

## 📝 Next Immediate Steps

1. **Find executions.go file** to add new method
2. **Create database migration** for question_id column
3. **Update execRepoIface** interface
4. **Test compilation** after changes
5. **Deploy and test** with sample job

**Ready to continue with Phase 2?**
