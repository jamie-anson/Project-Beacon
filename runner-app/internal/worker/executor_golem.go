package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jamie-anson/project-beacon-runner/internal/golem"
	"github.com/jamie-anson/project-beacon-runner/internal/logging"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// GolemExecutor implements the Executor interface using the Golem network
type GolemExecutor struct {
	Service *golem.Service
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
		return "", "failed", nil, nil, fmt.Errorf("golem service not configured")
	}

	l.Info().Str("job_id", spec.ID).Str("region", region).Msg("executing job on Golem network")
	
	res, err := golem.ExecuteSingleRegion(ctx, g.Service, spec, region)
	if err != nil {
		l.Error().Err(err).Str("job_id", spec.ID).Str("region", region).Msg("Golem execution error")
		
		// Persist failed execution row with error details in output
		out := map[string]any{"error": err.Error()}
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
