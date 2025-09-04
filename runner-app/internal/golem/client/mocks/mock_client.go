package mocks

import (
	"context"

	"github.com/jamie-anson/project-beacon-runner/internal/golem/client"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// MockYagnaClient is a minimal test double for client.YagnaClient
// Fields can be set per-test to control behavior.
type MockYagnaClient struct {
	ProbeFn              func(ctx context.Context) (string, map[string]any, error)
	CreateDemandFn       func(ctx context.Context, spec client.DemandSpec) (string, error)
	NegotiateAgreementFn func(ctx context.Context, demandID string) (string, error)
	CreateActivityFn     func(ctx context.Context, agreementID string, jobspec *models.JobSpec) (string, error)
	ExecFn               func(ctx context.Context, activityID string, jobspec *models.JobSpec) (string, string, int, error)
	StopActivityFn       func(ctx context.Context, activityID string) error
	GetWalletInfoFn      func(ctx context.Context) (*client.WalletInfo, error)
	GetPaymentPlatformsFn func(ctx context.Context) ([]client.PaymentPlatform, error)
}

func (m *MockYagnaClient) Probe(ctx context.Context) (string, map[string]any, error) {
	return m.ProbeFn(ctx)
}
func (m *MockYagnaClient) CreateDemand(ctx context.Context, spec client.DemandSpec) (string, error) {
	return m.CreateDemandFn(ctx, spec)
}
func (m *MockYagnaClient) NegotiateAgreement(ctx context.Context, demandID string) (string, error) {
	return m.NegotiateAgreementFn(ctx, demandID)
}
func (m *MockYagnaClient) CreateActivity(ctx context.Context, agreementID string, jobspec *models.JobSpec) (string, error) {
	return m.CreateActivityFn(ctx, agreementID, jobspec)
}
func (m *MockYagnaClient) Exec(ctx context.Context, activityID string, jobspec *models.JobSpec) (string, string, int, error) {
	return m.ExecFn(ctx, activityID, jobspec)
}
func (m *MockYagnaClient) StopActivity(ctx context.Context, activityID string) error {
	return m.StopActivityFn(ctx, activityID)
}
func (m *MockYagnaClient) GetWalletInfo(ctx context.Context) (*client.WalletInfo, error) {
	return m.GetWalletInfoFn(ctx)
}
func (m *MockYagnaClient) GetPaymentPlatforms(ctx context.Context) ([]client.PaymentPlatform, error) {
	return m.GetPaymentPlatformsFn(ctx)
}
