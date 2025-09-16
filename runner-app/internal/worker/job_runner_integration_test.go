//go:build !ci
// +build !ci

package worker

import (
    "context"
    "database/sql"
    "encoding/json"
    "testing"
    "time"

    "github.com/jamie-anson/project-beacon-runner/internal/golem"
    "github.com/jamie-anson/project-beacon-runner/internal/metrics"
    "github.com/jamie-anson/project-beacon-runner/internal/negotiation"
    "github.com/jamie-anson/project-beacon-runner/pkg/models"
    "github.com/stretchr/testify/require"
)

// --- Fakes ---
type fakeJobsRepo struct{ data map[string][]byte }

func (f *fakeJobsRepo) GetJob(ctx context.Context, id string) (string, string, []byte, sql.NullTime, sql.NullTime, error) {
    // Return jobspec row with queued status
    return id, "queued", f.data[id], sql.NullTime{}, sql.NullTime{}, nil
}

func (f *fakeJobsRepo) UpdateJobStatus(ctx context.Context, jobID string, status string) error {
    // Mock implementation for job status updates
    return nil
}

type fakeExecRepo struct{
    inserted struct{
        jobID string
        region string
        status string
    }
    execID int64
    verifications chan struct{
        executionID int64
        claimed string
        observed string
        verified bool
        method string
    }
}

func (f *fakeExecRepo) InsertExecution(ctx context.Context, jobID string, providerID string, region string, status string, startedAt time.Time, completedAt time.Time, outputJSON []byte, receiptJSON []byte) (int64, error) {
    f.inserted.jobID = jobID
    f.inserted.region = region
    f.inserted.status = status
    if f.execID == 0 { f.execID = 42 }
    return f.execID, nil
}

func (f *fakeExecRepo) UpdateRegionVerification(ctx context.Context, executionID int64, regionClaimed sql.NullString, regionObserved sql.NullString, regionVerified sql.NullBool, verificationMethod sql.NullString, evidenceRef sql.NullString) error {
    if f.verifications != nil {
        f.verifications <- struct{
            executionID int64
            claimed string
            observed string
            verified bool
            method string
        }{executionID, regionClaimed.String, regionObserved.String, regionVerified.Bool, verificationMethod.String}
    }
    return nil
}

type fakeProbe struct{ observed string }
func (p *fakeProbe) Verify(ctx context.Context, agreementID string) (string, negotiation.Evidence, error) {
    return p.observed, negotiation.Evidence{}, nil
}

type fakeProbeErr struct{ err error }
func (p *fakeProbeErr) Verify(ctx context.Context, agreementID string) (string, negotiation.Evidence, error) {
    return "", negotiation.Evidence{}, p.err
}

// --- Helpers ---
func mkJobSpecJSON(jobID, region string) []byte {
    js := &models.JobSpec{
        ID: jobID,
        Version: "1.0.0",
        Benchmark: models.BenchmarkSpec{
            Name: "echo",
            Container: models.ContainerSpec{Image: "alpine:latest", Command: []string{"echo","ok"}},
            Input: models.InputSpec{
                Type: "prompt",
                Data: map[string]interface{}{"prompt": "test"},
                Hash: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
            },
        },
        Constraints: models.ExecutionConstraints{
            Regions: []string{region},
            MinRegions: 1,
            Timeout:  30 * time.Second,
        },
        CreatedAt: time.Now(),
    }
    b, _ := json.Marshal(js)
    return b
}

// --- Tests ---
func TestJobRunner_Preflight_Success(t *testing.T) {
    ctx := context.Background()
    // Skip signature verification in validator for test fixtures
    t.Setenv("VALIDATION_SKIP_SIGNATURE", "true")

    jobID := "job-1"
    claimed := "US"

    before, err := metrics.Summary()
    require.NoError(t, err)

    jr := &JobRunner{
        DB:        nil,
        JobsRepo:  &fakeJobsRepo{data: map[string][]byte{jobID: mkJobSpecJSON(jobID, claimed)}},
        ExecRepo:  &fakeExecRepo{execID: 101, verifications: make(chan struct{executionID int64; claimed, observed string; verified bool; method string}, 1)},
        Golem:     golem.NewService("", "testnet"),
        Bundler:   nil,
        ProbeFactory: func() negotiation.PreflightProbe { return &fakeProbe{observed: claimed} },
    }

    // Envelope for the job
    payload, _ := json.Marshal(jobEnvelope{ID: jobID, EnqueuedAt: time.Now().Add(-time.Second)})

    // Execute
    require.NoError(t, jr.handleEnvelope(ctx, payload))

    // Wait for verification call
    select {
    case v := <-jr.ExecRepo.(*fakeExecRepo).verifications:
        require.Equal(t, int64(101), v.executionID)
        require.Equal(t, claimed, v.claimed)
        require.Equal(t, claimed, v.observed)
        require.True(t, v.verified)
        require.Equal(t, "preflight-geoip", v.method)
    case <-time.After(2 * time.Second):
        t.Fatal("timeout waiting for UpdateRegionVerification")
    }

    // Telemetry: jobs_processed_total should have increased by 1
    after, err := metrics.Summary()
    require.NoError(t, err)
    require.Equal(t, before["jobs_processed_total"]+1, after["jobs_processed_total"])
}

func TestJobRunner_Preflight_Mismatch(t *testing.T) {
    ctx := context.Background()
    t.Setenv("VALIDATION_SKIP_SIGNATURE", "true")

    jobID := "job-2"
    claimed := "US"
    observed := "EU"

    before, err := metrics.Summary()
    require.NoError(t, err)

    jr := &JobRunner{
        DB:        nil,
        JobsRepo:  &fakeJobsRepo{data: map[string][]byte{jobID: mkJobSpecJSON(jobID, claimed)}},
        ExecRepo:  &fakeExecRepo{execID: 202, verifications: make(chan struct{executionID int64; claimed, observed string; verified bool; method string}, 1)},
        Golem:     golem.NewService("", "testnet"),
        Bundler:   nil,
        ProbeFactory: func() negotiation.PreflightProbe { return &fakeProbe{observed: observed} },
    }

    payload, _ := json.Marshal(jobEnvelope{ID: jobID})
    require.NoError(t, jr.handleEnvelope(ctx, payload))

    select {
    case v := <-jr.ExecRepo.(*fakeExecRepo).verifications:
        require.Equal(t, int64(202), v.executionID)
        require.Equal(t, claimed, v.claimed)
        require.Equal(t, observed, v.observed)
        require.False(t, v.verified)
        require.Equal(t, "preflight-geoip", v.method)
    case <-time.After(2 * time.Second):
        t.Fatal("timeout waiting for UpdateRegionVerification")
    }

    // Telemetry: jobs_processed_total should have increased by 1 (execution still completes)
    after, err := metrics.Summary()
    require.NoError(t, err)
    require.Equal(t, before["jobs_processed_total"]+1, after["jobs_processed_total"])
}

// Preflight probe error path: ensure we log/skip and do not persist verification
func TestJobRunner_Preflight_ProbeError_NoPersistence(t *testing.T) {
    ctx := context.Background()
    t.Setenv("VALIDATION_SKIP_SIGNATURE", "true")

    jobID := "job-3"
    region := "US"

    before, err := metrics.Summary()
    require.NoError(t, err)

    verCh := make(chan struct{executionID int64; claimed, observed string; verified bool; method string}, 1)
    jr := &JobRunner{
        JobsRepo:  &fakeJobsRepo{data: map[string][]byte{jobID: mkJobSpecJSON(jobID, region)}},
        ExecRepo:  &fakeExecRepo{execID: 303, verifications: verCh},
        Golem:     golem.NewService("", "testnet"),
        ProbeFactory: func() negotiation.PreflightProbe { return &fakeProbeErr{err: context.DeadlineExceeded} },
    }

    payload, _ := json.Marshal(jobEnvelope{ID: jobID})
    require.NoError(t, jr.handleEnvelope(ctx, payload))

    select {
    case <-verCh:
        t.Fatal("expected no verification persistence on probe error")
    case <-time.After(600 * time.Millisecond):
        // ok: no persistence happened within window
    }

    // jobs_processed_total should have increased by 1 (execution still ran)
    after, err := metrics.Summary()
    require.NoError(t, err)
    require.Equal(t, before["jobs_processed_total"]+1, after["jobs_processed_total"])
}

// Multiple regions: ensure only chosen (first) region's verification is persisted exactly once
func TestJobRunner_Preflight_MultiRegion_OnlyChosenPersisted(t *testing.T) {
    ctx := context.Background()
    t.Setenv("VALIDATION_SKIP_SIGNATURE", "true")

    jobID := "job-4"
    regions := []string{"US", "EU"}

    // Build a custom JobSpec with multiple regions
    js := &models.JobSpec{
        ID: jobID,
        Version: "1.0.0",
        Benchmark: models.BenchmarkSpec{
            Name: "echo",
            Container: models.ContainerSpec{Image: "alpine:latest", Command: []string{"echo","ok"}},
            Input: models.InputSpec{Type: "prompt", Data: map[string]interface{}{"prompt": "x"}, Hash: "0123"},
        },
        Constraints: models.ExecutionConstraints{Regions: regions, MinRegions: 1, Timeout: 30 * time.Second},
        CreatedAt: time.Now(),
    }
    b, _ := json.Marshal(js)

    verCh := make(chan struct{executionID int64; claimed, observed string; verified bool; method string}, 1)
    jr := &JobRunner{
        JobsRepo:  &fakeJobsRepo{data: map[string][]byte{jobID: b}},
        ExecRepo:  &fakeExecRepo{execID: 404, verifications: verCh},
        Golem:     golem.NewService("", "testnet"),
        ProbeFactory: func() negotiation.PreflightProbe { return &fakeProbe{observed: regions[0]} },
    }

    payload, _ := json.Marshal(jobEnvelope{ID: jobID})
    require.NoError(t, jr.handleEnvelope(ctx, payload))

    select {
    case v := <-verCh:
        require.Equal(t, int64(404), v.executionID)
        require.Equal(t, regions[0], v.claimed)
        require.Equal(t, regions[0], v.observed)
        // Ensure only one persistence event
        select {
        case <-verCh:
            t.Fatal("expected exactly one verification persistence event")
        default:
        }
    case <-time.After(2 * time.Second):
        t.Fatal("timeout waiting for UpdateRegionVerification")
    }
}

// no queue needed in these tests

// Placeholder to keep the worker package's tests compiling.
// Proper integration tests will be added later.
func TestJobRunner_Placeholder(t *testing.T) {
    t.Skip("Integration tests need to be implemented with correct interfaces")
}
