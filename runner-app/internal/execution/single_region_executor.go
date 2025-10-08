package execution

import (
	"context"
	"encoding/json"
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

	// Create receipt
	receipt := &models.Receipt{
		ID:         fmt.Sprintf("receipt-%s-%d", jobSpec.ID, time.Now().UnixNano()),
		JobID:      jobSpec.ID,
		ProviderID: resp.ProviderUsed,
		Status:     "completed",
		CreatedAt:  startTime,
		CompletedAt: time.Now(),
		Duration:   duration,
		Output: models.Output{
			Data: map[string]interface{}{
				"response": resp.Response,
				"metadata": resp.Metadata,
			},
			Format: "json",
		},
		Metadata: map[string]interface{}{
			"provider_used":  resp.ProviderUsed,
			"inference_time": resp.InferenceSec,
			"region":         region,
		},
	}

	e.logger.Info("Execution completed successfully",
		"job_id", jobSpec.ID,
		"provider", resp.ProviderUsed,
		"duration", duration)

	return receipt, nil
}

// extractPrompt extracts the prompt from a jobspec
func extractPrompt(spec *models.JobSpec) string {
	if len(spec.Questions) > 0 {
		return spec.Questions[0]
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
