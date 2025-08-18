package golem

import (
    "context"
    "fmt"
    "strings"
    "time"

    "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// discoverProvidersSDK is the SDK-backed provider discovery
func (s *Service) discoverProvidersSDK(ctx context.Context, constraints models.ExecutionConstraints) ([]*Provider, error) {
    // Probe Yagna via transport client
    hitPath, version, err := s.client.Probe(ctx)
    if err != nil {
        return nil, err
    }

    // If real execution is enabled, create real SDK providers for each region
    if s.enableRealExec {
        var providers []*Provider
        for _, region := range constraints.Regions {
            provider := &Provider{
                ID:     fmt.Sprintf("yagna-sdk-%s", strings.ToLower(region)),
                Name:   fmt.Sprintf("Yagna SDK Provider (%s)", region),
                Region: region,
                Status: "online",
                Score:  0.95,
                Price:  0.01, // 0.01 GLM per hour
                Resources: ProviderResources{
                    CPU:    2,
                    Memory: 2048,
                    Disk:   10000,
                    GPU:    false,
                    Uptime: 99.0,
                },
                Metadata: map[string]interface{}{
                    "yagna_url":  strings.TrimRight(s.yagnaURL, "/"),
                    "probe_path": hitPath,
                    "version":    version,
                    "real_exec":  true,
                },
            }
            providers = append(providers, provider)
        }
        return providers, nil
    }

    // Fallback: Return single probe provider for testing
    region := "unknown"
    if len(constraints.Regions) > 0 {
        region = constraints.Regions[0]
    }

    providers := []*Provider{
        {
            ID:     "yagna-sdk-probe",
            Name:   "Yagna Daemon",
            Region: region,
            Status: "online",
            Score:  1.0,
            Price:  0.0,
            Resources: ProviderResources{
                CPU:    0,
                Memory: 0,
                Disk:   0,
                GPU:    false,
                Uptime: 100.0,
            },
            Metadata: map[string]interface{}{
                "yagna_url": strings.TrimRight(s.yagnaURL, "/"),
                "probe_path": hitPath,
                "version":   version,
            },
        },
    }

    return providers, nil
}

// executeTaskSDK executes the task using the real Golem SDK (stub for now)
func (s *Service) executeTaskSDK(ctx context.Context, provider *Provider, jobspec *models.JobSpec) (*TaskExecution, error) {
    // If feature flag is enabled, route to real implementation (to be filled)
    if s.enableRealExec {
        return s.executeTaskSDKReal(ctx, provider, jobspec)
    }
    // Minimal execution stub: probe Yagna via transport and synthesize a completed TaskExecution.
    // This validates the end-to-end Execute path (worker -> execute -> persist)
    // before wiring full demand/agree/execute.
    hitPath, version, err := s.client.Probe(ctx)
    if err != nil {
        return nil, fmt.Errorf("yagna exec probe failed: %w", err)
    }

    now := time.Now()
    exec := &TaskExecution{
        ID:         fmt.Sprintf("sdk-probe-%d", now.UnixNano()),
        JobSpecID:  jobspec.ID,
        ProviderID: "yagna-sdk",
        Status:     "completed",
        StartedAt:  now.Add(-1 * time.Second),
        CompletedAt: now,
        Output: map[string]any{
            "stdout": "SDK path OK (probe only)",
            "stderr": "",
            "exit_code": 0,
        },
        Metadata: map[string]any{
            "yagna_url": strings.TrimRight(s.yagnaURL, "/"),
            "probe_path": hitPath,
            "version":   version,
            "region":    provider.Region,
            "image":     jobspec.Benchmark.Container.Image,
            "cmd":       jobspec.Benchmark.Container.Command,
            "env":       jobspec.Benchmark.Container.Environment,
            "note":      "placeholder execution; demand/agree/execute not yet implemented",
        },
    }
    return exec, nil
}

// executeTaskSDKReal is the scaffold for the real demand/agreement/activity/exec flow.
// TODO: Implement using Yagna REST API:
// 1) Create demand with constraints (region, image, resources)
// 2) Negotiate agreement with a provider
// 3) Create activity (deploy container image)
// 4) Start execution with command and environment
// 5) Stream/capture stdout, stderr, exit code
// 6) Stop and release activity; handle errors and timeouts; retries
// executeTaskSDKReal is implemented in execute_real.go

// --- Real exec scaffolding helpers (Yagna REST wiring to be implemented) ---

// buildDemandSpec is implemented in demand.go
