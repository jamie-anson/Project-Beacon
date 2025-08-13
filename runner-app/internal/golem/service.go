package golem

import (
	"fmt"
	"sync"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// Service handles Golem Network integration
type Service struct {
	apiKey    string
	network   string // mainnet, testnet
	timeout   time.Duration
	providers map[string]*Provider
	providersOnce sync.Once
}

// Provider represents a Golem compute provider
type Provider struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Region   string            `json:"region"`
	Status   string            `json:"status"` // online, offline, busy
	Score    float64           `json:"score"`  // reputation score 0-1
	Price    float64           `json:"price"`  // GLM per hour
	Resources ProviderResources `json:"resources"`
	Metadata map[string]interface{} `json:"metadata"`
}

// ProviderResources represents available compute resources
type ProviderResources struct {
	CPU    int     `json:"cpu"`    // CPU cores
	Memory int64   `json:"memory"` // Memory in MB
	Disk   int64   `json:"disk"`   // Disk space in MB
	GPU    bool    `json:"gpu"`    // GPU availability
	Uptime float64 `json:"uptime"` // Uptime percentage
}

// TaskExecution represents a running task on Golem
type TaskExecution struct {
	ID          string                 `json:"id"`
	JobSpecID   string                 `json:"jobspec_id"`
	ProviderID  string                 `json:"provider_id"`
	Status      string                 `json:"status"` // pending, running, completed, failed
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt time.Time              `json:"completed_at"`
	Output      interface{}            `json:"output"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// NewService creates a new Golem service instance
func NewService(apiKey, network string) *Service {
	return &Service{
		apiKey:    apiKey,
		network:   network,
		timeout:   30 * time.Minute,
		providers: make(map[string]*Provider),
	}
}

// GetProvider retrieves a provider by ID
func (s *Service) GetProvider(id string) (*Provider, error) {
	provider, exists := s.providers[id]
	if !exists {
		return nil, fmt.Errorf("provider not found: %s", id)
	}
	return provider, nil
}

// EstimateTaskCost estimates the cost of running a task
func (s *Service) EstimateTaskCost(provider *Provider, jobspec *models.JobSpec) (float64, error) {
	// Simple cost estimation: price per hour * estimated execution time
	estimatedHours := float64(jobspec.Constraints.Timeout) / float64(time.Hour)
	cost := provider.Price * estimatedHours
	
	return cost, nil
}
