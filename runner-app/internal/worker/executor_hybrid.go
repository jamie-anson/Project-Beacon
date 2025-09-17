package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jamie-anson/project-beacon-runner/internal/hybrid"
	"github.com/jamie-anson/project-beacon-runner/internal/logging"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// HybridExecutor implements the Executor interface using the hybrid router
type HybridExecutor struct {
	Client *hybrid.Client
}

// NewHybridExecutor creates a new HybridExecutor
func NewHybridExecutor(client *hybrid.Client) *HybridExecutor {
	return &HybridExecutor{
		Client: client,
	}
}

// Execute runs a job using the hybrid router
func (h *HybridExecutor) Execute(ctx context.Context, spec *models.JobSpec, region string) (providerID, status string, outputJSON, receiptJSON []byte, err error) {
	l := logging.FromContext(ctx)
	
	if h.Client == nil {
		return "", "failed", nil, nil, fmt.Errorf("hybrid client not configured")
	}

	prompt := extractPrompt(spec)
	model := extractModel(spec)
	regionPref := mapRegionToRouter(region)
	
	l.Info().Str("job_id", spec.ID).Str("region", regionPref).Str("model", model).Str("prompt", prompt).Msg("calling hybrid router")
	
	req := hybrid.InferenceRequest{
		Model:            model,
		Prompt:           prompt,
		Temperature:      0.1,
		MaxTokens:        128,
		RegionPreference: regionPref,
		CostPriority:     true,
	}
	
	l.Debug().Str("job_id", spec.ID).Interface("request", req).Msg("hybrid router request details")
	
	hre, herr := h.Client.RunInference(ctx, req)
	
	l.Info().Str("job_id", spec.ID).Bool("success", hre != nil && hre.Success).Err(herr).Msg("hybrid router response received")
	
	if herr != nil || hre == nil || !hre.Success {
		l.Error().Err(herr).Str("job_id", spec.ID).Str("region", regionPref).Msg("hybrid router inference error")
		
		out := map[string]any{"error": fmt.Sprintf("hybrid error: %v", herr)}
		if hre != nil && hre.Error != "" {
			out["router_error"] = hre.Error
		}
		outJSON, _ := json.Marshal(out)
		
		return "", "failed", outJSON, nil, fmt.Errorf("hybrid execution failed: %w", herr)
	}
	
	// Success via Hybrid
	l.Info().Str("job_id", spec.ID).Str("provider", hre.ProviderUsed).Str("region", regionPref).Msg("hybrid router execution successful")
	
	out := map[string]any{
		"response": hre.Response,
		"provider": hre.ProviderUsed,
		"metadata": hre.Metadata,
	}
	outJSON, _ := json.Marshal(out)
	
	return hre.ProviderUsed, "completed", outJSON, nil, nil
}
