# Retry Re-execution Logic Implementation Guide

## Overview
This document provides the implementation details for adding actual inference re-execution to the `RetryQuestion()` handler.

---

## Current State

The `RetryQuestion()` handler in `/runner-app/internal/api/executions_handler.go` currently:
- ✅ Validates execution exists
- ✅ Checks retry eligibility (failed/timeout/error)
- ✅ Enforces max retry limit (3 attempts)
- ✅ Updates database with retry count and history
- ❌ **Missing:** Actual inference re-execution

---

## Implementation Steps

### Step 1: Understand Existing Execution Flow

First, examine how jobs are currently executed:

**Files to Review:**
- `/runner-app/internal/worker/job_runner.go` - Main job execution logic
- `/runner-app/internal/worker/executor.go` - Executor interface
- `/runner-app/internal/worker/executor_hybrid.go` - Hybrid executor implementation
- `/runner-app/internal/hybrid/client.go` - Hybrid router HTTP client

**Key Functions:**
- `JobRunner.handleEnvelope()` - Processes job from queue
- `HybridExecutor.Execute()` - Runs inference via hybrid router
- `extractPrompt()` and `extractModel()` - Helper functions

### Step 2: Add Re-execution Service

Create a new service to handle single question re-execution:

**File:** `/runner-app/internal/service/retry_service.go`

```go
package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/hybrid"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

type RetryService struct {
	DB           *sql.DB
	HybridClient *hybrid.Client
}

func NewRetryService(db *sql.DB, hybridClient *hybrid.Client) *RetryService {
	return &RetryService{
		DB:           db,
		HybridClient: hybridClient,
	}
}

// RetryQuestionExecution re-runs inference for a specific question
func (s *RetryService) RetryQuestionExecution(ctx context.Context, executionID int64, region string, questionIndex int) error {
	// 1. Fetch original job spec
	var jobSpecData []byte
	err := s.DB.QueryRowContext(ctx, `
		SELECT j.jobspec_data
		FROM executions e
		JOIN jobs j ON e.job_id = j.id
		WHERE e.id = $1
	`, executionID).Scan(&jobSpecData)
	
	if err != nil {
		return fmt.Errorf("failed to fetch job spec: %w", err)
	}
	
	// 2. Parse job spec
	var jobSpec models.JobSpec
	if err := json.Unmarshal(jobSpecData, &jobSpec); err != nil {
		return fmt.Errorf("failed to parse job spec: %w", err)
	}
	
	// 3. Extract the specific question
	if questionIndex < 0 || questionIndex >= len(jobSpec.Input.Questions) {
		return fmt.Errorf("invalid question index: %d", questionIndex)
	}
	
	question := jobSpec.Input.Questions[questionIndex]
	
	// 4. Get model ID from execution
	var modelID string
	err = s.DB.QueryRowContext(ctx, `
		SELECT COALESCE(model_id, 'llama3.2-1b')
		FROM executions
		WHERE id = $1
	`, executionID).Scan(&modelID)
	
	if err != nil {
		return fmt.Errorf("failed to fetch model ID: %w", err)
	}
	
	// 5. Call hybrid router for inference
	result, err := s.executeInference(ctx, region, modelID, question.Text, question.SystemPrompt)
	if err != nil {
		// Update execution with failure
		s.updateExecutionFailure(ctx, executionID, err.Error())
		return fmt.Errorf("inference failed: %w", err)
	}
	
	// 6. Update execution with success
	return s.updateExecutionSuccess(ctx, executionID, result)
}

func (s *RetryService) executeInference(ctx context.Context, region, modelID, prompt, systemPrompt string) (map[string]interface{}, error) {
	// Use hybrid client to run inference
	req := &hybrid.InferenceRequest{
		Region:       region,
		Model:        modelID,
		Prompt:       prompt,
		SystemPrompt: systemPrompt,
		Temperature:  0.7,
		MaxTokens:    2000,
	}
	
	resp, err := s.HybridClient.RunInference(ctx, req)
	if err != nil {
		return nil, err
	}
	
	return map[string]interface{}{
		"response":    resp.Response,
		"provider_id": resp.ProviderID,
		"duration_ms": resp.DurationMs,
		"tokens":      resp.Tokens,
	}, nil
}

func (s *RetryService) updateExecutionSuccess(ctx context.Context, executionID int64, result map[string]interface{}) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return err
	}
	
	_, err = s.DB.ExecContext(ctx, `
		UPDATE executions
		SET 
			status = 'completed',
			output_data = $1,
			completed_at = NOW(),
			updated_at = NOW()
		WHERE id = $2
	`, resultJSON, executionID)
	
	return err
}

func (s *RetryService) updateExecutionFailure(ctx context.Context, executionID int64, errorMsg string) error {
	_, err := s.DB.ExecContext(ctx, `
		UPDATE executions
		SET 
			status = 'failed',
			original_error = $1,
			updated_at = NOW()
		WHERE id = $2
	`, errorMsg, executionID)
	
	return err
}
```

### Step 3: Update ExecutionsHandler

Modify `/runner-app/internal/api/executions_handler.go`:

```go
// Add RetryService field to ExecutionsHandler
type ExecutionsHandler struct {
	ExecutionsRepo *store.ExecutionsRepo
	RetryService   *service.RetryService  // ADD THIS
}

// Update RetryQuestion handler (replace TODO section)
func (h *ExecutionsHandler) RetryQuestion(c *gin.Context) {
	// ... existing validation code ...
	
	// Update execution record with retry attempt
	_, err = h.ExecutionsRepo.DB.ExecContext(ctx, `
		UPDATE executions 
		SET 
			retry_count = $1,
			last_retry_at = NOW(),
			retry_history = retry_history || $2::jsonb,
			status = 'retrying',
			updated_at = NOW()
		WHERE id = $3
	`, newRetryCount, fmt.Sprintf("[%s]", mustMarshalJSON(retryHistoryEntry)), executionID)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update retry count",
		})
		return
	}
	
	// REPLACE TODO WITH ACTUAL RE-EXECUTION
	if h.RetryService != nil {
		// Run re-execution asynchronously
		go func() {
			retryCtx := context.Background()
			if err := h.RetryService.RetryQuestionExecution(retryCtx, executionID, req.Region, req.QuestionIndex); err != nil {
				// Log error but don't fail the HTTP response
				// The execution status will be updated in the database
				fmt.Printf("Retry execution failed for execution %d: %v\n", executionID, err)
			}
		}()
	}
	
	c.JSON(http.StatusOK, RetryQuestionResponse{
		ExecutionID:   fmt.Sprintf("%d", executionID),
		Region:        req.Region,
		QuestionIndex: req.QuestionIndex,
		Status:        "retrying",
		RetryAttempt:  newRetryCount,
		UpdatedAt:     time.Now().Format(time.RFC3339),
		Result: map[string]interface{}{
			"message": "Retry queued successfully",
			"job_id":  jobSpecID,
			"model_id": modelID,
			"question_id": questionID,
		},
	})
}
```

### Step 4: Wire Up Dependencies

Update `/runner-app/internal/api/routes.go`:

```go
func SetupRoutes(jobsService *service.JobsService, cfg *config.Config, redisClient *redis.Client, queueClient ...interface{ GetCircuitBreakerStats() string }) *gin.Engine {
	// ... existing code ...
	
	// Initialize RetryService
	var retryService *service.RetryService
	if jobsService != nil && jobsService.ExecutionsRepo != nil {
		// Get hybrid client from somewhere (you may need to pass it as parameter)
		hybridClient := hybrid.NewClient(cfg.HybridRouterURL)
		retryService = service.NewRetryService(jobsService.ExecutionsRepo.DB, hybridClient)
	}
	
	if jobsService != nil {
		// ... existing handler initialization ...
		executionsHandler = NewExecutionsHandler(jobsService.ExecutionsRepo)
		executionsHandler.RetryService = retryService  // ADD THIS
		// ... rest of code ...
	}
	
	// ... rest of routes ...
}
```

### Step 5: Handle Edge Cases

**Important Considerations:**

1. **Timeout Handling:**
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
   defer cancel()
   ```

2. **Concurrent Retries:**
   - Database update is atomic (retry_count increment)
   - Multiple simultaneous retries will fail at validation step

3. **Provider Selection:**
   - Hybrid router should handle provider selection
   - May want to exclude provider that previously failed

4. **Partial Results:**
   - If retry succeeds, update only that execution
   - Don't affect other regions/questions

5. **Audit Trail:**
   - Update retry_history with final status:
   ```go
   retry_history = retry_history || '[{"attempt": X, "status": "success/failed", "timestamp": "..."}]'::jsonb
   ```

---

## Testing Plan

### Unit Tests

**File:** `/runner-app/internal/service/retry_service_test.go`

```go
func TestRetryService_RetryQuestionExecution(t *testing.T) {
	// Test cases:
	// 1. Successful retry
	// 2. Invalid question index
	// 3. Inference failure
	// 4. Database update failure
}
```

### Integration Tests

**File:** `/runner-app/internal/api/executions_handler_test.go`

```go
func TestExecutionsHandler_RetryQuestion_EndToEnd(t *testing.T) {
	// 1. Create failed execution
	// 2. Call retry endpoint
	// 3. Verify status updated to 'retrying'
	// 4. Wait for async execution
	// 5. Verify final status is 'completed'
}
```

### Manual Testing

```bash
# 1. Create a job that will timeout
curl -X POST http://localhost:8090/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d @test-job.json

# 2. Wait for execution to fail/timeout

# 3. Get execution ID from failed execution
curl http://localhost:8090/api/v1/jobs/{job_id}/executions/all

# 4. Retry the failed question
curl -X POST http://localhost:8090/api/v1/executions/{execution_id}/retry-question \
  -H "Content-Type: application/json" \
  -d '{"region": "United States", "question_index": 0}'

# 5. Check execution status
curl http://localhost:8090/api/v1/executions/{execution_id}/details
```

---

## Deployment Checklist

- [ ] Run database migration `0010_add_retry_tracking.up.sql`
- [ ] Deploy updated runner app with RetryService
- [ ] Verify hybrid router is accessible
- [ ] Test retry endpoint with failed execution
- [ ] Monitor logs for async execution errors
- [ ] Update API documentation

---

## Monitoring & Observability

Add logging at key points:

```go
log.Printf("[RETRY] Starting retry for execution %d, region %s, question %d", executionID, region, questionIndex)
log.Printf("[RETRY] Inference completed for execution %d: status=%s, duration=%dms", executionID, status, duration)
log.Printf("[RETRY] Failed to retry execution %d: %v", executionID, err)
```

Add metrics:
- `retry_attempts_total` - Counter of retry attempts
- `retry_success_total` - Counter of successful retries
- `retry_failure_total` - Counter of failed retries
- `retry_duration_seconds` - Histogram of retry durations

---

## Alternative: Queue-Based Approach

Instead of async goroutine, consider using Redis queue:

```go
// Enqueue retry job
retryJob := map[string]interface{}{
	"execution_id":   executionID,
	"region":         region,
	"question_index": questionIndex,
	"retry_attempt":  newRetryCount,
}
redisClient.LPush(ctx, "retry_queue", mustMarshalJSON(retryJob))
```

**Pros:**
- Better reliability (survives app restarts)
- Can monitor queue depth
- Easier to implement rate limiting

**Cons:**
- More complex setup
- Requires separate worker process

---

## Next Steps

1. Implement `RetryService` in `/runner-app/internal/service/retry_service.go`
2. Update `ExecutionsHandler` to use `RetryService`
3. Wire up dependencies in `routes.go`
4. Write unit tests
5. Run integration tests
6. Deploy and test in staging environment
7. Update retry-plan.md checklist
