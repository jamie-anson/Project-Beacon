package negotiation

import (
    "context"
    "testing"
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/testutil"
    metrics "github.com/jamie-anson/project-beacon-runner/internal/metrics"
)

// helper: count observations for a specific (region,outcome) in NegotiationDurationSeconds
func obsCountFor(region, outcome string) uint64 {
    fams, _ := prometheus.DefaultGatherer.Gather()
    var cnt uint64
    for _, mf := range fams {
        if mf.GetName() != "negotiation_duration_seconds" {
            continue
        }
        for _, m := range mf.Metric {
            // Must match both labels on the histogram metric
            okReg, okOut := false, false
            for _, lp := range m.Label {
                if lp.GetName() == "region" && lp.GetValue() == region { okReg = true }
                if lp.GetName() == "outcome" && lp.GetValue() == outcome { okOut = true }
            }
            if okReg && okOut && m.GetHistogram() != nil {
                cnt += m.GetHistogram().GetSampleCount()
            }
        }
    }
    return cnt
}

type fakeSrcOne struct{ emitted bool }
func (f *fakeSrcOne) Next(ctx context.Context) (OfferView, error) {
    if f.emitted {
        // trigger timeout pathway quickly
        select {
        case <-ctx.Done():
            return OfferView{}, ctx.Err()
        case <-time.After(2 * time.Millisecond):
            return OfferView{}, context.DeadlineExceeded
        }
    }
    f.emitted = true
    return OfferView{}, nil
}

type ff struct{ lvl MatchLevel; claimed string }
func (f ff) Classify(ov OfferView, target string) (MatchLevel, string) { return f.lvl, f.claimed }

type goodProbe struct{ observed string }
func (p goodProbe) Verify(ctx context.Context, _ string) (string, Evidence, error) {
    return p.observed, Evidence{PublicIP: "198.51.100.2"}, nil
}

func TestNegotiationDurationLabels_StrictMatch(t *testing.T) {
    region := "US"
    before := obsCountFor(region, "strict_match")

    n := NewNegotiator(&fakeSrcOne{}, ff{lvl: P0Explicit, claimed: region})
    ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
    defer cancel()
    _, _, _ = n.Acquire(ctx, RegionRequest{Region: region, StrictTimeout: 10 * time.Millisecond, RelaxTimeout: 10 * time.Millisecond})

    after := obsCountFor(region, "strict_match")
    if after <= before {
        t.Fatalf("expected strict_match count to increase, before=%d after=%d", before, after)
    }
}

func TestNegotiationDurationLabels_ProbeVerified(t *testing.T) {
    region := "EU"
    before := obsCountFor(region, "probe_verified")

    src := &fakeSrcOne{}
    filt := ff{lvl: P3NeedsProbe}
    probe := goodProbe{observed: region}
    n := NewNegotiatorWithProbe(src, filt, probe)

    ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
    defer cancel()
    _, _, _ = n.Acquire(ctx, RegionRequest{Region: region, StrictTimeout: 0, RelaxTimeout: 50 * time.Millisecond})

    after := obsCountFor(region, "probe_verified")
    if after <= before {
        t.Fatalf("expected probe_verified count to increase, before=%d after=%d", before, after)
    }
}

func TestNegotiationDurationLabels_NeedsProbe(t *testing.T) {
    region := "ASIA"
    before := obsCountFor(region, "needs_probe")

    src := &fakeSrcOne{}
    filt := ff{lvl: P3NeedsProbe}
    n := NewNegotiator(src, filt)

    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
    defer cancel()
    _, _, _ = n.Acquire(ctx, RegionRequest{Region: region, StrictTimeout: 0, RelaxTimeout: 20 * time.Millisecond})

    // We can't definitively ensure outcome due to retries; assert at least some observation recorded.
    // Also ensure other negotiation metrics registered (sanity)
    _ = testutil.CollectAndCount(metrics.NegotiationDurationSeconds)

    after := obsCountFor(region, "needs_probe")
    if after <= before {
        t.Fatalf("expected needs_probe count to increase, before=%d after=%d", before, after)
    }
}
