package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jamie-anson/project-beacon-runner/internal/golem"
	"github.com/jamie-anson/project-beacon-runner/internal/logging"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// GolemExecutor implements the Executor interface using the Golem network
type GolemExecutor struct {
	Service *golem.Service
}

func buildGolemFailure(region string, goErr error) map[string]any {
	message := "golem execution failed"
	if goErr != nil {
		message = goErr.Error()
	}

	failure := map[string]any{
		"code":          "GOLEM_EXECUTION_FAILED",
		"stage":         "provider_execution",
		"component":     "golem_executor",
		"subcomponent":  "execution",
		"message":       message,
		"provider_type": "golem",
		"region":        region,
		"transient":     false,
	}

	switch {
	case goErr == nil:
		failure["code"] = "GOLEM_EXECUTION_FAILED"
	case errors.Is(goErr, golem.ErrTimeout):
		failure["code"] = "GOLEM_TIMEOUT"
		failure["transient"] = true
	case errors.Is(goErr, golem.ErrUnavailable):
		failure["code"] = "GOLEM_UNAVAILABLE"
		failure["transient"] = true
	case errors.Is(goErr, golem.ErrNotFound):
		failure["code"] = "GOLEM_NOT_FOUND"
	case errors.Is(goErr, golem.ErrCanceled):
		failure["code"] = "GOLEM_CANCELED"
	}

	return failure
}

// NewGolemExecutor creates a new GolemExecutor
func NewGolemExecutor(service *golem.Service) *GolemExecutor {
	return &GolemExecutor{
		Service: service,
	}
}

// Execute runs a job using the Golem network
func (g *GolemExecutor) Execute(ctx context.Context, spec *models.JobSpec, region string) (providerID, status string, outputJSON, receiptJSON []byte, err error) {
	l := logging.FromContext(ctx)

	if g.Service == nil {
		failure := buildGolemFailure(region, fmt.Errorf("service not configured"))
		out := map[string]any{
			"error":      "golem error: service not configured",
			"error_code": failure["code"],
			"failure":    failure,
		}
		outJSON, _ := json.Marshal(out)
		return "", "failed", outJSON, nil, fmt.Errorf("golem execution failed: service not configured")
	}

	l.Info().Str("job_id", spec.ID).Str("region", region).Msg("executing job on Golem network")

	res, err := golem.ExecuteSingleRegion(ctx, g.Service, spec, region)
	if err != nil {
		l.Error().Err(err).Str("job_id", spec.ID).Str("region", region).Msg("Golem execution error")

		failure := buildGolemFailure(region, err)

		// Persist failed execution row with error details in output
		out := map[string]any{
			"error":      fmt.Sprintf("golem error: %v", err),
			"error_code": failure["code"],
			"failure":    failure,
		}
		outJSON, _ := json.Marshal(out)

		return "", "failed", outJSON, nil, fmt.Errorf("golem execution failed: %w", err)
	}

	l.Info().Str("job_id", spec.ID).Str("provider", res.ProviderID).Str("region", region).Msg("Golem execution successful")

	// Marshal output and receipt
	outJSON, _ := json.Marshal(res.Execution.Output)
	recJSON, _ := json.Marshal(res.Receipt)

	status = "completed"
	if res.Execution != nil {
		status = res.Execution.Status
	}

	return res.ProviderID, status, outJSON, recJSON, nil
}
