package golem

import (
	"fmt"
	"strings"
	"time"

	"context"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// executeTaskSDKReal is the scaffold for the real demand/agreement/activity/exec flow.
// TODO: Implement using Yagna REST API:
// 1) Create demand with constraints (region, image, resources)
// 2) Negotiate agreement with a provider
// 3) Create activity (deploy container image)
// 4) Start execution with command and environment
// 5) Stream/capture stdout, stderr, exit code
// 6) Stop and release activity; handle errors and timeouts; retries
func (s *Service) executeTaskSDKReal(ctx context.Context, provider *Provider, jobspec *models.JobSpec) (*TaskExecution, error) {
	start := time.Now()
	meta := map[string]any{
		"region":    provider.Region,
		"image":     jobspec.Benchmark.Container.Image,
		"cmd":       jobspec.Benchmark.Container.Command,
		"env":       jobspec.Benchmark.Container.Environment,
		"yagna_url": strings.TrimRight(s.yagnaURL, "/"),
	}

	// 0) Ensure wallet is ready for payments
	if err := s.EnsureWalletReady(ctx); err != nil {
		return s.execFailure("wallet_check", start, provider, jobspec, meta, err), nil
	}

	// 1) Create demand
	dSpec := s.buildDemandSpec(provider, jobspec)
	demandID, err := s.client.CreateDemand(ctx, dSpec)
	if err != nil {
		return s.execFailure("create_demand", start, provider, jobspec, meta, err), nil
	}
	meta["demand_id"] = demandID

	// 2) Negotiate agreement
	agreeID, err := s.client.NegotiateAgreement(ctx, demandID)
	if err != nil {
		return s.execFailure("negotiate_agreement", start, provider, jobspec, meta, err), nil
	}
	meta["agreement_id"] = agreeID

	// 3) Create activity
	actID, err := s.client.CreateActivity(ctx, agreeID, jobspec)
	if err != nil {
		return s.execFailure("create_activity", start, provider, jobspec, meta, err), nil
	}
	meta["activity_id"] = actID

	// 4) Execute container
	stdout, stderr, exitCode, err := s.client.Exec(ctx, actID, jobspec)
	if err != nil {
		meta["exit_code"] = exitCode
		return s.execFailure("exec_container", start, provider, jobspec, meta, err), nil
	}

	// 5) Cleanup (best-effort)
	_ = s.client.StopActivity(ctx, actID)

	// Success
	end := time.Now()
	exec := &TaskExecution{
		ID:          fmt.Sprintf("sdk-exec-%d", end.UnixNano()),
		JobSpecID:   jobspec.ID,
		ProviderID:  provider.ID,
		Status:      "completed",
		StartedAt:   start,
		CompletedAt: end,
		Output: map[string]any{
			"stdout":    stdout,
			"stderr":    stderr,
			"exit_code": exitCode,
		},
		Metadata: meta,
	}
	return exec, nil
}
