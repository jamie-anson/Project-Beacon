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

func buildHybridFailure(region, model string, hre *hybrid.InferenceResponse, herr error) (map[string]any, string) {
	failure := map[string]any{
		"stage":        "router_inference",
		"component":    "hybrid_router",
		"subcomponent": "inference",
		"code":         "ROUTER_UNKNOWN_ERROR",
		"type":         "unknown",
		"region":       region,
		"model":        model,
	}

	message := ""
	if herr != nil {
		message = herr.Error()
	}
	if hre != nil && hre.Error != "" {
		message = hre.Error
	}
	if message != "" {
		failure["message"] = message
	}

	errorCode := "ROUTER_UNKNOWN_ERROR"
	stage := "router_inference"
	failureType := failure["type"].(string)
	transient := false

	if herr != nil {
		if he, ok := herr.(*hybrid.HybridError); ok {
			failureType = string(he.Type)
			if he.StatusCode > 0 {
				failure["http_status"] = he.StatusCode
			}
			if he.URL != "" {
				failure["url"] = he.URL
			}

			switch he.Type {
			case hybrid.ErrorTypeNotFound:
				stage = "router_http_request"
				errorCode = "ROUTER_HTTP_404"
				failureType = "http_error"
			case hybrid.ErrorTypeHTTP:
				stage = "router_http_request"
				failureType = "http_error"
				status := he.StatusCode
				if status <= 0 {
					status = 500
				}
				errorCode = fmt.Sprintf("ROUTER_HTTP_%d", status)
				failure["http_status"] = status
				transient = status >= 500
			case hybrid.ErrorTypeTimeout:
				stage = "router_http_timeout"
				errorCode = "ROUTER_TIMEOUT"
				failureType = "timeout"
				transient = true
			case hybrid.ErrorTypeNetwork:
				stage = "router_network"
				errorCode = "ROUTER_NETWORK"
				failureType = "network"
				transient = true
			case hybrid.ErrorTypeJSON:
				stage = "router_response_parse"
				errorCode = "ROUTER_JSON_ERROR"
				failureType = "json_error"
			case hybrid.ErrorTypeRouter:
				stage = "router_inference"
				errorCode = "ROUTER_ERROR"
				failureType = "router_error"
			default:
				// keep defaults
			}
		}
	}

	if hre != nil {
		if hre.ProviderUsed != "" {
			failure["provider"] = hre.ProviderUsed
		}
		if hre.Metadata != nil {
			if providerType, ok := hre.Metadata["provider_type"]; ok {
				failure["provider_type"] = providerType
			}
			if routerFailure, ok := hre.Metadata["failure"]; ok {
				failure["router_failure"] = routerFailure
			}
			if metaRegion, ok := hre.Metadata["region"]; ok {
				if typed, okTyped := metaRegion.(string); okTyped && typed != "" {
					failure["region"] = typed
				}
			}
		}
	}

	if message != "" {
		failure["message"] = message
	}

	failure["stage"] = stage
	failure["code"] = errorCode
	failure["type"] = failureType
	failure["transient"] = transient

	return failure, errorCode
}

// NewHybridExecutor creates a new HybridExecutor
func NewHybridExecutor(client *hybrid.Client) *HybridExecutor {
	return &HybridExecutor{
		Client: client,
	}
}

// Execute runs a job using the hybrid router with regional prompts
func (h *HybridExecutor) Execute(ctx context.Context, spec *models.JobSpec, region string) (providerID, status string, outputJSON, receiptJSON []byte, err error) {
	l := logging.FromContext(ctx)

	if h.Client == nil {
		return "", "failed", nil, nil, fmt.Errorf("hybrid client not configured")
	}

	// Extract question and format with regional prompt
	question := extractPrompt(spec)
	model := extractModel(spec)
	regionPref := mapRegionToRouter(region)
	
	// Format prompt with regional system prompt
	prompt := formatRegionalPrompt(question, regionPref)

	l.Info().Str("job_id", spec.ID).Str("region", regionPref).Str("model", model).Str("prompt", prompt).Msg("calling hybrid router")

	req := hybrid.InferenceRequest{
		Model:            model,
		Prompt:           prompt,
		Temperature:      0.1,
		MaxTokens:        500,  // Increased from 128 for detailed bias detection responses
		RegionPreference: regionPref,
		CostPriority:     false, // Prefer quality over cost for bias detection
	}

	// TRACE: Log exact URLs and requests being made
	l.Error().
		Str("job_id", spec.ID).
		Interface("inference_request", req).
		Msg("TRACE: Calling hybrid router with request")

	hre, herr := h.Client.RunInference(ctx, req)

	// TRACE: Log response details and provider info
	l.Error().
		Str("job_id", spec.ID).
		Bool("success", hre != nil && hre.Success).
		Str("provider_used", func() string {
			if hre != nil { return hre.ProviderUsed }
			return "unknown"
		}()).
		Interface("response_metadata", func() interface{} {
			if hre != nil { return hre.Metadata }
			return nil
		}()).
		Err(herr).
		Msg("TRACE: Hybrid router response received")

	if herr != nil || hre == nil || !hre.Success {
		l.Error().Err(herr).Str("job_id", spec.ID).Str("region", regionPref).Msg("hybrid router inference error")

		failure, errorCode := buildHybridFailure(regionPref, model, hre, herr)
		errMsg := fmt.Sprintf("hybrid error: %v", herr)
		if failureMsg, ok := failure["message"].(string); ok && failureMsg != "" {
			errMsg = fmt.Sprintf("hybrid error: %s", failureMsg)
		}

		out := map[string]any{
			"error":      errMsg,
			"error_code": errorCode,
			"failure":    failure,
		}
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

	//var receiptJSON []byte <= error
	if hre.Metadata != nil {
		if rec, ok := hre.Metadata["receipt"]; ok && rec != nil {
			switch typed := rec.(type) {
			case string:
				receiptJSON = []byte(typed)
			default:
				if recBytes, marshalErr := json.Marshal(typed); marshalErr == nil {
					receiptJSON = recBytes
				} else {
					l.Warn().Err(marshalErr).Str("job_id", spec.ID).Msg("failed to marshal receipt metadata")
				}
			}
		}
	}

	return hre.ProviderUsed, "completed", outJSON, receiptJSON, nil
}
