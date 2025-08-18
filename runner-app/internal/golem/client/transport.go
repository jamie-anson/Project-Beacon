package client

import (
	"context"
	"net/http"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// YagnaClient abstracts Yagna REST interactions for probing, market, agreements and execution.
type YagnaClient interface {
	// Probe checks connectivity/auth and returns the endpoint path that succeeded and optional version payload.
	Probe(ctx context.Context) (hitPath string, version map[string]any, err error)
	// CreateDemand creates a market demand and returns its ID.
	CreateDemand(ctx context.Context, spec DemandSpec) (string, error)
	// NegotiateAgreement negotiates an agreement for the given demand and returns agreement ID.
	NegotiateAgreement(ctx context.Context, demandID string) (string, error)
	// CreateActivity creates an activity for a given agreement and returns activity ID.
	CreateActivity(ctx context.Context, agreementID string, jobspec *models.JobSpec) (string, error)
	// Exec runs the specified container command, returning stdio and exit code.
	Exec(ctx context.Context, activityID string, jobspec *models.JobSpec) (stdout string, stderr string, exitCode int, err error)
	// StopActivity stops/releases the specified activity.
	StopActivity(ctx context.Context, activityID string) error
	// GetWalletInfo retrieves wallet information for payments.
	GetWalletInfo(ctx context.Context) (*WalletInfo, error)
	// GetPaymentPlatforms retrieves available payment platforms.
	GetPaymentPlatforms(ctx context.Context) ([]PaymentPlatform, error)
}

// YagnaRESTClient is a concrete implementation of YagnaClient using raw HTTP.
type YagnaRESTClient struct {
	BaseURL    string
	AppKey     string
	HTTPClient *http.Client
	Timeout    time.Duration
	// Discovered base prefixes for Market and Activity APIs
	MarketBase   string
	ActivityBase string
}

// DemandSpec is the typed demand payload (extracted from sdk.go scaffolding).
type DemandSpec struct {
	Constraints string         `json:"constraints"`
	Properties  map[string]any `json:"properties"`
	Metadata    map[string]any `json:"metadata"`
}

// WalletInfo and PaymentPlatform mirror types from the golem package that the client needs.
type WalletInfo struct {
	Address   string  `json:"address"`
	BalanceGLM float64 `json:"balance_glm"`
}

type PaymentPlatform struct {
	Name string `json:"name"`
}
