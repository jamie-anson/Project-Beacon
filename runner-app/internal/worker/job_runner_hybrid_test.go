package worker

import (
    "context"
    "database/sql"
    "testing"
    "time"

    "github.com/jamie-anson/project-beacon-runner/internal/hybrid"
    models "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

type fakeExecRepo2 struct{
    lastJobID string
    lastProvider string
    lastRegion string
    lastStatus string
}

func (f *fakeExecRepo2) InsertExecution(ctx context.Context, jobID, providerID, region, status string, startedAt, completedAt time.Time, outputJSON, receiptJSON []byte) (int64, error) {
    f.lastJobID = jobID
    f.lastProvider = providerID
    f.lastRegion = region
    f.lastStatus = status
    return 42, nil
}

func (f *fakeExecRepo2) UpdateRegionVerification(ctx context.Context, executionID int64, regionClaimed sql.NullString, regionObserved sql.NullString, regionVerified sql.NullBool, verificationMethod sql.NullString, evidenceRef sql.NullString) error {
    return nil
}

// Minimal executor that returns success with a provider id
type stubExecutor struct{}
func (s stubExecutor) Execute(ctx context.Context, spec *models.JobSpec, region string) (string, string, []byte, []byte, error) {
    return "modal-eu-west", "completed", []byte(`{"response":"ok"}`), nil, nil
}

func TestJobRunner_SetHybridClient_SetsHybridExecutor(t *testing.T) {
    jr := NewJobRunner(nil, nil, nil, nil)
    if _, ok := jr.Executor.(*GolemExecutor); !ok && jr.Executor != nil {
        t.Fatalf("precondition: expected default executor to be Golem or nil")
    }
    jr.SetHybridClient(hybrid.New("http://example.com"))
    if _, ok := jr.Executor.(*HybridExecutor); !ok {
        t.Fatalf("SetHybridClient did not set HybridExecutor")
    }
}

func TestJobRunner_ExecuteSingleRegion_UsesHybridRegionMapping(t *testing.T) {
    jr := NewJobRunner(nil, nil, nil, nil)
    fake := &fakeExecRepo2{}
    jr.ExecRepo = fake
    // Non-nil Hybrid triggers actualRegion mapping in executeSingleRegion
    jr.Hybrid = hybrid.New("http://example.com")

    spec := &models.JobSpec{ID: "job-1", Benchmark: models.BenchmarkSpec{Input: models.InputSpec{Type: "prompt", Data: map[string]any{"prompt": "hi"}}}}

    res := jr.executeSingleRegion(context.Background(), spec.ID, spec, "EU", stubExecutor{})

    if res.Region != "eu-west" {
        t.Fatalf("Region = %q, want eu-west (hybrid mapping)", res.Region)
    }
    if res.ProviderID != "modal-eu-west" || res.Status != "completed" {
        t.Fatalf("Unexpected provider/status: %q/%q", res.ProviderID, res.Status)
    }

    // Verify ExecRepo saw the mapped region
    if fake.lastRegion != "eu-west" {
        t.Fatalf("ExecRepo.InsertExecution region = %q, want eu-west", fake.lastRegion)
    }
}
