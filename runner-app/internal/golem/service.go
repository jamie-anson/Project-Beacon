package golem

import (
	"crypto/ed25519"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
	pcrypto "github.com/jamie-anson/project-beacon-runner/pkg/crypto"
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
	client YagnaClient
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
	svc := &Service{
		apiKey:    apiKey,
		network:   network,
		timeout:   30 * time.Minute,
		providers: make(map[string]*Provider),
	}

	// Optionally load receipt signing private key from env (base64)
	if keyB64 := os.Getenv("RECEIPT_PRIVATE_KEY"); keyB64 != "" {
		if pk, err := pcrypto.PrivateKeyFromBase64(keyB64); err == nil {
			svc.signingKey = pk
		}
	} else {
		// Dev/test fallback: generate an ephemeral signing key so receipts are signed
		if kp, err := pcrypto.GenerateKeyPair(); err == nil {
			svc.signingKey = kp.PrivateKey
		}
	}

	// Backend selection
	if b := os.Getenv("GOLEM_BACKEND"); b != "" {
		svc.backend = b
	} else {
		svc.backend = "mock"
	}

	// Yagna REST configuration (used when backend=="sdk")
	if url := os.Getenv("YAGNA_API_URL"); url != "" {
		svc.yagnaURL = url
	} else {
		svc.yagnaURL = "http://127.0.0.1:7465"
	}
	svc.yagnaKey = os.Getenv("YAGNA_APPKEY")
	svc.httpClient = &http.Client{Timeout: 15 * time.Second}

	// Initialize transport client for SDK backend usage
	svc.client = NewYagnaRESTClient(svc.yagnaURL, svc.yagnaKey, svc.httpClient)

	// Feature flag for real SDK execution path
	if os.Getenv("GOLEM_ENABLE_REAL_EXEC") == "true" {
		svc.enableRealExec = true
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
	cost := provider.Price * estimatedHours
	
	return cost, nil
}

// Network returns the configured network (e.g., mainnet, testnet)
func (s *Service) Network() string {
    return s.network
}

// SigningKey returns the configured receipt signing private key (may be nil)
func (s *Service) SigningKey() ed25519.PrivateKey {
    return s.signingKey
}
