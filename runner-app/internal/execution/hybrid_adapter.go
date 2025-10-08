package execution

import (
	"context"

	"github.com/jamie-anson/project-beacon-runner/internal/hybrid"
)

// HybridRouterAdapter adapts hybrid.Client to HybridRouterClient interface
type HybridRouterAdapter struct {
	client *hybrid.Client
}

// NewHybridRouterAdapter creates a new adapter
func NewHybridRouterAdapter(client *hybrid.Client) *HybridRouterAdapter {
	return &HybridRouterAdapter{client: client}
}

// GetProviders retrieves providers and converts them to execution.Provider format
func (a *HybridRouterAdapter) GetProviders(ctx context.Context) ([]Provider, error) {
	hybridProviders, err := a.client.GetProviders(ctx)
	if err != nil {
		return nil, err
	}

	providers := make([]Provider, len(hybridProviders))
	for i, hp := range hybridProviders {
		providers[i] = Provider{
			ID:              hp.Name, // Use name as ID
			Name:            hp.Name,
			Region:          hp.Region,
			Type:            hp.Type,
			Healthy:         hp.Healthy,
			CostPerSecond:   hp.CostPerSecond,
			AvgLatency:      hp.AvgLatency,
			SuccessRate:     hp.SuccessRate,
			LastHealthCheck: hp.LastHealthCheck,
		}
	}

	return providers, nil
}
