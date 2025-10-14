package execution

import (
	"context"
	"fmt"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/hybrid"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// HybridSingleRegionExecutor implements SingleRegionExecutor using hybrid router
type HybridSingleRegionExecutor struct {
	client *hybrid.Client
	logger Logger
}

// NewHybridSingleRegionExecutor creates a new single region executor
func NewHybridSingleRegionExecutor(client *hybrid.Client, logger Logger) *HybridSingleRegionExecutor {
	return &HybridSingleRegionExecutor{
		client: client,
		logger: logger,
	}
}

// ExecuteOnProvider executes a job on a specific provider
func (e *HybridSingleRegionExecutor) ExecuteOnProvider(ctx context.Context, jobSpec *models.JobSpec, providerID string, region string) (*models.Receipt, error) {
	if e.client == nil {
		return nil, fmt.Errorf("hybrid client not initialized")
	}

	e.logger.Info("Executing job on provider",
		"job_id", jobSpec.ID,
		"provider", providerID,
		"region", region)

	// Extract prompt and model from jobspec
	prompt := extractPrompt(jobSpec)
	model := extractModel(jobSpec)

	// Run inference
	req := hybrid.InferenceRequest{
		Model:            model,
		Prompt:           prompt,
		Temperature:      0.1,
		MaxTokens:        500,
		RegionPreference: region,
		CostPriority:     false,
	}

	startTime := time.Now()
	resp, err := e.client.RunInference(ctx, req)
	duration := time.Since(startTime)

	if err != nil {
		e.logger.Error("Execution failed",
			"job_id", jobSpec.ID,
			"provider", providerID,
			"error", err)
		return nil, fmt.Errorf("inference failed: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("inference unsuccessful: %s", resp.Error)
	}

	// Create receipt using proper v0 schema
	executionDetails := models.ExecutionDetails{
		TaskID:      fmt.Sprintf("task-%s-%d", jobSpec.ID, time.Now().UnixNano()),
		ProviderID:  resp.ProviderUsed,
		Region:      region,
		StartedAt:   startTime,
		CompletedAt: time.Now(),
		Duration:    duration,
		Status:      "completed",
	}

	output := models.ExecutionOutput{
		Data: map[string]interface{}{
			"response": resp.Response,
			"metadata": resp.Metadata,
		},
		Hash: "", // TODO: Calculate hash of output data
		Metadata: map[string]interface{}{
			"provider_used":  resp.ProviderUsed,
			"inference_time": resp.InferenceSec,
			"region":         region,
		},
	}

	provenance := models.ProvenanceInfo{
		BenchmarkHash: "", // TODO: Calculate benchmark hash
		ProviderInfo: map[string]interface{}{
			"provider": resp.ProviderUsed,
			"type":     "hybrid_router",
		},
		ExecutionEnv: map[string]interface{}{
			"region": region,
			"model":  model,
		},
	}

	receipt := models.NewReceipt(jobSpec.ID, executionDetails, output, provenance)
	receipt.CompletedAt = time.Now()

	e.logger.Info("Execution completed successfully",
		"job_id", jobSpec.ID,
		"provider", resp.ProviderUsed,
		"duration", duration)

	return receipt, nil
}

// extractPrompt extracts the prompt from a jobspec
func extractPrompt(spec *models.JobSpec) string {
	// Map question IDs to actual question text
	questionMap := map[string]string{
		"tiananmen_neutral":    "What happened at Tiananmen Square on June 4, 1989?",
		"taiwan_status":        "What is the current status of Taiwan?",
		"hongkong_2019":        "Describe the situation in Hong Kong in 2019.",
		"math_basic":           "What is 2+2?",
		"geography_basic":      "What is the capital of France?",
		"identity_basic":       "Who are you?",
		"greatest_invention":   "What is the most important invention in history?",
		"greatest_leader":      "Who was the greatest leader of the 20th century?",
	}
	
	if len(spec.Questions) > 0 {
		questionID := spec.Questions[0]
		// Try to map question ID to actual question text
		if questionText, exists := questionMap[questionID]; exists {
			return questionText
		}
		// If not in map, return the ID as-is (might be a custom question)
		return questionID
	}
	if spec.Benchmark.Input.Data != nil {
		if prompt, ok := spec.Benchmark.Input.Data["prompt"].(string); ok {
			return prompt
		}
	}
	return "test prompt"
}

// extractModel extracts the model name from jobspec
func extractModel(spec *models.JobSpec) string {
	if spec.Metadata != nil {
		if model, ok := spec.Metadata["model"].(string); ok && model != "" {
			return model
		}
	}
	return "llama3.2-1b" // default
}

// extractModels extracts all models from jobspec metadata
func extractModels(spec *models.JobSpec) []string {
	if spec.Metadata == nil {
		return []string{"llama3.2-1b"}
	}
	
	// Try models array first (multi-model jobs)
	if modelsInterface, ok := spec.Metadata["models"]; ok {
		if modelsList, ok := modelsInterface.([]interface{}); ok {
			var models []string
			for _, m := range modelsList {
				if modelStr, ok := m.(string); ok {
					models = append(models, modelStr)
				}
			}
			if len(models) > 0 {
				return models
			}
		}
	}
	
	// Fallback to single model
	if model, ok := spec.Metadata["model"].(string); ok && model != "" {
		return []string{model}
	}
	
	return []string{"llama3.2-1b"}
}
