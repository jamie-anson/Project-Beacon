# Cross-Region Execution Bug - Root Cause Analysis

## Problem

Portal shows all 18 executions with identical responses (all answering "Greatest Invention" question).

## Root Cause

The cross-region executor only executes **once per region** instead of **9 times per region** (3 models × 3 questions).

### Current Flow (BROKEN)

```
Cross-Region Handler
  ↓
ExecuteAcrossRegions (2 regions: US, EU)
  ↓
executeRegion (called once per region)
  ↓
ExecuteOnProvider (called once per region)
  ↓
extractPrompt → Returns Questions[0] ONLY ❌
extractModel → Returns one model ONLY ❌
  ↓
Hybrid Router (executes 1 question with 1 model)
  ↓
Returns 1 receipt per region
  ↓
Handler creates 18 execution records with same receipt ❌
```

### Expected Flow (CORRECT)

```
Cross-Region Handler
  ↓
ExecuteAcrossRegions (2 regions: US, EU)
  ↓
executeRegion (called once per region)
  ↓
Loop: for each question (3 questions)
  ↓
  Loop: for each model (3 models)
    ↓
    ExecuteOnProvider (called 9 times per region)
      ↓
      Hybrid Router (executes 1 question with 1 model)
      ↓
      Returns 1 receipt
    ↓
  Store receipt with correct question/model
  ↓
Total: 18 receipts (2 regions × 3 models × 3 questions)
```

## Files Affected

### 1. `internal/execution/single_region_executor.go`
**Lines 116-118**: `extractPrompt()` only returns first question
**Lines 127-135**: `extractModel()` only returns one model

### 2. `internal/execution/cross_region_executor.go`
**Lines 258-322**: `executeRegion()` calls ExecuteOnProvider once instead of looping

### 3. `internal/handlers/cross_region_handlers.go`
**Lines 328-384**: Creates fake execution records by looping over questions/models that weren't actually executed

## Solution

### Option 1: Loop in Cross-Region Executor (Recommended)

Modify `executeRegion()` to loop through questions and models:

```go
func (cre *CrossRegionExecutor) executeRegion(ctx context.Context, jobSpec *models.JobSpec, plan RegionExecutionPlan) *RegionResult {
    // ... existing setup ...
    
    // Extract models and questions
    models := extractModels(jobSpec)
    questions := jobSpec.Questions
    
    // Execute each model × question combination
    var receipts []*models.Receipt
    for _, model := range models {
        for _, question := range questions {
            // Create modified jobspec for this specific execution
            execSpec := cloneJobSpec(jobSpec)
            execSpec.Metadata["model"] = model
            execSpec.Questions = []string{question}
            
            // Execute
            receipt, err := cre.singleRegionExecutor.ExecuteOnProvider(regionCtx, execSpec, providerID, plan.Region)
            if err != nil {
                // Handle error
                continue
            }
            
            receipts = append(receipts, receipt)
        }
    }
    
    // Return result with all receipts
    result.Receipts = receipts // Need to change RegionResult struct
    return result
}
```

### Option 2: Make Hybrid Router Handle Multi-Model/Question

Modify hybrid router to accept arrays of models/questions and return structured results.

**Pros**: Single API call per region
**Cons**: Requires hybrid router changes, more complex response parsing

## Impact

- **Current**: 2 actual executions, 18 fake records with duplicate data
- **After Fix**: 18 actual executions, 18 real records with unique data
- **Cost**: 9x increase in API calls to hybrid router (expected behavior)
- **Latency**: Longer job completion time (sequential execution per region)

## Next Steps

1. Implement Option 1 (loop in cross-region executor)
2. Modify `RegionResult` struct to store multiple receipts
3. Update handler to create execution records from actual receipts
4. Test with real job to verify 18 unique responses
5. Consider parallelizing model/question execution for performance

## Testing

After fix, verify:
- ✅ 18 unique execution records
- ✅ Each record has different question response
- ✅ Each record has correct model_id and question_id
- ✅ Receipts contain actual execution data
- ✅ Portal displays all results correctly
