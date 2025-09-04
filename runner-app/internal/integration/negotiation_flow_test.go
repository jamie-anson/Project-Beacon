package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jamie-anson/project-beacon-runner/internal/golem"
	"github.com/jamie-anson/project-beacon-runner/internal/negotiation"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// fakeOfferSource yields a single offer with no region properties (P3 needs probe)
type fakeOfferSource struct{ yielded bool }

func (f *fakeOfferSource) Next(ctx context.Context) (negotiation.OfferView, error) {
	if f.yielded {
		// Wait briefly to allow relax window to proceed
		select {
		case <-ctx.Done():
			return negotiation.OfferView{}, ctx.Err()
		case <-time.After(2 * time.Millisecond):
			return negotiation.OfferView{}, context.DeadlineExceeded
		}
	}
	f.yielded = true
	return negotiation.OfferView{Properties: map[string]string{}}, nil
}

// stub ipFetcher always returns a fixed IP
func stubFetcher(ip string) func(ctx context.Context, _ string) (string, error) {
	return func(ctx context.Context, _ string) (string, error) { return ip, nil }
}

// stub geoResolver maps any IP to desired bucket
type stubGeo struct{ country, bucket, source string; err error }

func (s stubGeo) LookupIP(ip string) (string, string, string, error) {
	return s.country, s.bucket, s.source, s.err
}

// fakeExecRepo captures persisted verification calls
type fakeExecRepo struct{ called int; last struct{ claimed, observed string; verified bool; method string } }

func (f *fakeExecRepo) UpdateRegionVerification(claimed, observed string, verified bool, method string) {
	f.called++
	f.last.claimed = claimed
	f.last.observed = observed
	f.last.verified = verified
	f.last.method = method
}

func TestNegotiationToExecutionToPersistence(t *testing.T) {
	ctx := context.Background()
	region := "EU"

	// 1) Negotiation: P3 offer -> probe returns EU -> verified
	src := &fakeOfferSource{}
	filt := negotiation.NewOfferFilter()
	probe := negotiation.NewPreflightProbe(stubFetcher("203.0.113.10"), stubGeo{country: "DE", bucket: "EU", source: "test-db"})
	n := negotiation.NewNegotiatorWithProbe(src, filt, probe)
	_, rv, err := n.Acquire(ctx, negotiation.RegionRequest{Region: region, StrictTimeout: 0, RelaxTimeout: 80 * time.Millisecond})
	if err != nil {
		t.Fatalf("negotiation failed: %v", err)
	}
	if !rv.Verified || rv.Observed != region {
		t.Fatalf("expected verified EU, got verified=%v observed=%q", rv.Verified, rv.Observed)
	}

	// 2) Execution: run single region harness (uses mock provider list in golem.Service)
	js := &models.JobSpec{
		ID: "job-int-1",
		Version: "1.0.0",
		Benchmark: models.BenchmarkSpec{
			Name: "echo",
			Container: models.ContainerSpec{Image: "alpine:latest", Command: []string{"echo", "ok"}},
			Input: models.InputSpec{Type: "prompt", Data: map[string]interface{}{"p": "x"}, Hash: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"},
		},
		Constraints: models.ExecutionConstraints{Regions: []string{region}, MinRegions: 1, Timeout: 15 * time.Second},
		CreatedAt: time.Now(),
	}
	b, _ := json.Marshal(js)
	_ = b // ensure no unused warnings; js passed directly below
	svc := golem.NewService("", "testnet")
	if _, err := golem.ExecuteSingleRegion(ctx, svc, js, region); err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	// 3) Persistence: persist region verification
	repo := &fakeExecRepo{}
	repo.UpdateRegionVerification(rv.Claimed, rv.Observed, rv.Verified, rv.Method)
	if repo.called != 1 {
		t.Fatalf("expected 1 persistence call, got %d", repo.called)
	}
	if repo.last.observed != region || !repo.last.verified || repo.last.method != "preflight-geoip" {
		t.Fatalf("unexpected persistence payload: %+v", repo.last)
	}
}
