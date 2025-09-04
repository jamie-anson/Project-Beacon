package golem

import (
	"context"
	"crypto/ed25519"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// EngineService abstracts the operations ExecutionEngine needs from the Service.
// It allows decoupling the engine from concrete service implementation for testing.
type EngineService interface {
	DiscoverProviders(ctx context.Context, constraints models.ExecutionConstraints) ([]*Provider, error)
	ExecuteTask(ctx context.Context, provider *Provider, jobspec *models.JobSpec) (*TaskExecution, error)
	EstimateTaskCost(provider *Provider, jobspec *models.JobSpec) (float64, error)
	GetProvider(id string) (*Provider, error)

	// Accessors used in receipt generation
	Network() string
	SigningKey() ed25519.PrivateKey
}
