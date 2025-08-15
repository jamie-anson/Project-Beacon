package golem

import (
	"context"
	"testing"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func TestExecuteSingleRegion(t *testing.T) {
	svc := NewService("", "testnet")
	jobspec := &models.JobSpec{
		ID:      "job-single-us",
		Version: "1.0",
		Benchmark: models.BenchmarkSpec{
			Name: "Who Are You?",
			Container: models.ContainerSpec{
				Image: "beacon/text-gen",
				Tag:   "latest",
				Resources: models.ResourceSpec{CPU: "1000m", Memory: "512Mi"},
			},
			Input: models.InputSpec{Type: "prompt", Data: map[string]any{"prompt": "hi"}, Hash: "sha256:abc"},
		},
		Constraints: models.ExecutionConstraints{Regions: []string{"US", "EU"}, MinRegions: 1, Timeout: 10 * time.Second},
	}

	ctx := context.Background()
	res, err := ExecuteSingleRegion(ctx, svc, jobspec, "US")
	if err != nil {
		t.Fatalf("ExecuteSingleRegion failed: %v", err)
	}
	if res == nil || res.Execution == nil {
		t.Fatalf("expected non-nil execution result")
	}
	if res.Region != "US" {
		t.Fatalf("expected region US, got %s", res.Region)
	}
	if res.Receipt == nil {
		t.Fatalf("expected receipt to be generated")
	}
}

func TestExecuteSingleRegion_InvalidRegion(t *testing.T) {
	svc := NewService("", "testnet")
	jobspec := &models.JobSpec{
		ID:      "job-single-us",
		Version: "1.0",
		Benchmark: models.BenchmarkSpec{
			Name: "Who Are You?",
			Container: models.ContainerSpec{Image: "beacon/text-gen", Tag: "latest", Resources: models.ResourceSpec{CPU: "1000m", Memory: "512Mi"}},
			Input:     models.InputSpec{Type: "prompt", Data: map[string]any{"prompt": "hi"}, Hash: "sha256:abc"},
		},
		Constraints: models.ExecutionConstraints{Regions: []string{"US"}, MinRegions: 1, Timeout: 10 * time.Second},
	}

	_, err := ExecuteSingleRegion(context.Background(), svc, jobspec, "EU")
	if err == nil {
		t.Fatalf("expected error for region not in jobspec constraints")
	}
}
