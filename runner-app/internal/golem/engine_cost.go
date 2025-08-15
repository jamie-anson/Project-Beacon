package golem

import (
	"context"
	"fmt"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// EstimateExecutionCost estimates the total cost for multi-region execution.
func (e *ExecutionEngine) EstimateExecutionCost(ctx context.Context, jobspec *models.JobSpec) (float64, error) {
	providers, err := e.service.DiscoverProviders(ctx, jobspec.Constraints)
	if err != nil {
		return 0, fmt.Errorf("provider discovery failed: %w", err)
	}

	grouped := groupProvidersByRegion(providers)
	bestPerRegion := selectBestPerRegion(grouped)

	var total float64
	for _, provider := range bestPerRegion {
		cost, err := e.service.EstimateTaskCost(provider, jobspec)
		if err != nil {
			continue // Skip providers with cost estimation errors
		}
		total += cost
	}
	return total, nil
}
