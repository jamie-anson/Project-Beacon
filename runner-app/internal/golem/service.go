package golem

import (
	"crypto/ed25519"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
	"github.com/jamie-anson/project-beacon-runner/pkg/crypto"
    clientpkg "github.com/jamie-anson/project-beacon-runner/internal/golem/client"
)

// Service handles Golem Network integration
type Service struct {
	apiKey    string
	network   string // mainnet, testnet
	timeout   time.Duration
	providers map[string]*Provider
	providersOnce sync.Once
	signingKey ed25519.PrivateKey // optional, for receipt signing
	backend   string // "mock" (default) or "sdk"
	// Yagna REST SDK configuration
	yagnaURL   string
	yagnaKey   string
	httpClient *http.Client
	// Feature flag: enable real demand/agreement/activity execution path
	enableRealExec bool
	// Transport client for Yagna REST (used when backend=="sdk")
	client clientpkg.YagnaClient
	// Wallet information for payments
	walletInfo *WalletInfo
}

// Provider type is defined in client.go to avoid duplication

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

// Config holds configuration for the Service
type Config struct {
	APIKey       string
	Network      string
	Timeout      time.Duration
	SigningKey   ed25519.PrivateKey
	Backend      string
	YagnaURL     string
	YagnaKey     string
	HTTPClient   *http.Client
	EnableRealExec bool
	MarketBase   string
	ActivityBase string
}

// NewService creates a new Golem service instance using environment-derived config.
func NewService(apiKey, network string) *Service {
	cfg := LoadConfigFromEnv(apiKey, network)
	return NewServiceWithConfig(cfg)
}

// NewServiceWithConfig constructs Service from a provided Config without reading environment.
func NewServiceWithConfig(cfg Config) *Service {
	svc := &Service{
		apiKey:       cfg.APIKey,
		network:      cfg.Network,
		timeout:      cfg.Timeout,
		providers:    make(map[string]*Provider),
		signingKey:   cfg.SigningKey,
		backend:      cfg.Backend,
		yagnaURL:     cfg.YagnaURL,
		yagnaKey:     cfg.YagnaKey,
		httpClient:   cfg.HTTPClient,
		enableRealExec: cfg.EnableRealExec,
	}

	if svc.httpClient == nil {
		svc.httpClient = &http.Client{Timeout: 15 * time.Second}
	}

	// Ensure we have a signing key; generate ephemeral if none provided
	if svc.signingKey == nil {
		if kp, err := crypto.GenerateKeyPair(); err == nil {
			svc.signingKey = kp.PrivateKey
		}
	}

	// Initialize transport client for SDK backend usage
	svc.client = clientpkg.NewYagnaRESTClient(svc.yagnaURL, svc.yagnaKey, svc.httpClient)
	if yc, ok := svc.client.(*clientpkg.YagnaRESTClient); ok {
		if cfg.MarketBase != "" {
			yc.MarketBase = cfg.MarketBase
		}
		if cfg.ActivityBase != "" {
			yc.ActivityBase = cfg.ActivityBase
		}
	}

	return svc
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
    if estimatedHours <= 0 {
        // Default to 30 minutes if not specified
        estimatedHours = 0.5
    }

    rate := provider.Pricing.CPUPerHour
    if rate <= 0 {
        // Fallback to legacy Price field used by mocks/tests
        rate = provider.Price
    }
    cost := rate * estimatedHours

    return cost, nil
}

// Network returns the configured network (e.g., mainnet, testnet)
// Accessors moved to service_config.go
