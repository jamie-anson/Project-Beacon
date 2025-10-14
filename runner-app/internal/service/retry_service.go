package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

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
	log.Printf("[RETRY] Starting retry for execution %d, region %s, question %d", executionID, region, questionIndex)
	
	// Check if execution was cancelled before starting retry
	var currentStatus string
	err := s.DB.QueryRowContext(ctx, `SELECT status FROM executions WHERE id = $1`, executionID).Scan(&currentStatus)
	if err != nil {
		return fmt.Errorf("failed to check execution status: %w", err)
	}
	if currentStatus == "cancelled" {
		log.Printf("[RETRY] Execution %d was cancelled, aborting retry", executionID)
		return fmt.Errorf("execution was cancelled")
	}
	
	// 1. Fetch original job spec
	var jobSpecData []byte
	err = s.DB.QueryRowContext(ctx, `
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
	
	// 3. Extract the specific question (Questions is a top-level field)
	if questionIndex < 0 || questionIndex >= len(jobSpec.Questions) {
		return fmt.Errorf("invalid question index: %d (total questions: %d)", questionIndex, len(jobSpec.Questions))
	}
	
	questionID := jobSpec.Questions[questionIndex]
	
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
	
	// 5. Convert question ID to actual prompt text
	prompt := s.questionIDToText(questionID)
	
	// 6. Call hybrid router for inference
	result, err := s.executeInference(ctx, region, modelID, prompt)
	if err != nil {
		// Update execution with failure
		s.updateExecutionFailure(ctx, executionID, err.Error())
		return fmt.Errorf("inference failed: %w", err)
	}
	
	log.Printf("[RETRY] Inference completed for execution %d: status=success", executionID)
	
	// 7. Update execution with success
	return s.updateExecutionSuccess(ctx, executionID, result)
}

// questionIDToText converts question ID to actual question text
func (s *RetryService) questionIDToText(questionID string) string {
	questionMap := map[string]string{
		"tiananmen_neutral":    "What happened at Tiananmen Square on June 4, 1989?",
		"taiwan_status":        "What is the current status of Taiwan?",
		"hongkong_2019":        "Describe the situation in Hong Kong in 2019.",
		"greatest_invention":   "What is the most important invention in history?",
		"greatest_leader":      "Who was the greatest leader of the 20th century?",
		"math_basic":           "What is 2+2?",
		"geography_basic":      "What is the capital of France?",
		"identity_basic":       "Who are you?",
	}
	
	if text, ok := questionMap[questionID]; ok {
		return text
	}
	
	// If not a known ID, assume it's already the question text
	return questionID
}

func (s *RetryService) executeInference(ctx context.Context, region, modelID, prompt string) (map[string]interface{}, error) {
	// Use hybrid client to run inference
	req := hybrid.InferenceRequest{
		Model:            modelID,
		Prompt:           prompt,
		Temperature:      0.7,
		MaxTokens:        2000,
		RegionPreference: region,
		CostPriority:     false,
	}
	
	resp, err := s.HybridClient.RunInference(ctx, req)
	if err != nil {
		return nil, err
	}
	
	if !resp.Success {
		return nil, fmt.Errorf("inference failed: %s", resp.Error)
	}
	
	return map[string]interface{}{
		"response":      resp.Response,
		"provider_used": resp.ProviderUsed,
		"inference_sec": resp.InferenceSec,
		"metadata":      resp.Metadata,
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
			completed_at = NOW()
		WHERE id = $2
	`, resultJSON, executionID)
	
	return err
}

func (s *RetryService) updateExecutionFailure(ctx context.Context, executionID int64, errorMsg string) error {
	_, err := s.DB.ExecContext(ctx, `
		UPDATE executions
		SET 
			status = 'failed',
			original_error = $1
		WHERE id = $2
	`, errorMsg, executionID)
	
	return err
}
