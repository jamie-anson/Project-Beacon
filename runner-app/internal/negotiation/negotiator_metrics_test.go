package negotiation

import (
    "context"
    "testing"
    "time"

    "github.com/prometheus/client_golang/prometheus/testutil"
    metrics "github.com/jamie-anson/project-beacon-runner/internal/metrics"
)

type fakeSource struct{ offers []OfferView }
func (f *fakeSource) Next(ctx context.Context) (OfferView, error) {
    if len(f.offers) == 0 {
        // wait a tiny bit to allow time windows to elapse
        select {
        case <-ctx.Done():
            return OfferView{}, ctx.Err()
        case <-time.After(2 * time.Millisecond):
            return OfferView{}, context.DeadlineExceeded
        }
    }
    ov := f.offers[0]
    f.offers = f.offers[1:]
    return ov, nil
}

type fakeFilter struct{ lvl MatchLevel; claimed string }
func (f fakeFilter) Classify(ov OfferView, target string) (MatchLevel, string) { return f.lvl, f.claimed }

type fakeProbe struct{ observed string; err error }
func (p fakeProbe) Verify(ctx context.Context, agreementID string) (string, Evidence, error) {
    if p.err != nil { return "", Evidence{}, p.err }
    return p.observed, Evidence{PublicIP: "1.2.3.4"}, nil
}

func resetNegotiationMetrics(t *testing.T) {
    // Recreate vectors by re-registering on a fresh registry isn't trivial since metrics.RegisterAll uses Default.
    // For these tests we only read per-label child counters via testutil.ToFloat64 on the child, which doesn't require resetting.
    // To keep tests independent, read current values and assert deltas.
}

func TestStrictMatchMetrics(t *testing.T) {
    resetNegotiationMetrics(t)
    region := "US"
    src := &fakeSource{offers: []OfferView{{}}}
    filt := fakeFilter{lvl: P0Explicit, claimed: region}
    n := NewNegotiator(src, filt).(*negotiatorImpl)

    beforeSeen := testutil.ToFloat64(metrics.OffersSeenTotal.WithLabelValues(region))
    beforeP0P2 := testutil.ToFloat64(metrics.OffersP0P2Total.WithLabelValues(region))

    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()
    _, _, _ = n.Acquire(ctx, RegionRequest{Region: region, StrictTimeout: 50 * time.Millisecond, RelaxTimeout: 10 * time.Millisecond})

    if got := testutil.ToFloat64(metrics.OffersSeenTotal.WithLabelValues(region)); got < beforeSeen+1 {
        t.Fatalf("offers seen: got %v want >= %v", got, beforeSeen+1)
    }
    if got := testutil.ToFloat64(metrics.OffersP0P2Total.WithLabelValues(region)); got < beforeP0P2+1 {
        t.Fatalf("offers p0p2: got %v want >= %v", got, beforeP0P2+1)
    }
    // Negotiation duration observation count should increase (cannot easily assert by label without registry); sanity check collector not empty.
    if c := testutil.CollectAndCount(metrics.NegotiationDurationSeconds); c == 0 {
        t.Fatalf("expected negotiation_duration_seconds to have observations")
    }
}

func TestProbeSuccessMetrics(t *testing.T) {
    resetNegotiationMetrics(t)
    region := "EU"
    src := &fakeSource{offers: []OfferView{{}}}
    filt := fakeFilter{lvl: P3NeedsProbe, claimed: ""}
    probe := fakeProbe{observed: region}
    n := NewNegotiatorWithProbe(src, filt, probe).(*negotiatorImpl)

    beforeP3 := testutil.ToFloat64(metrics.OffersP3Total.WithLabelValues(region))
    beforePass := testutil.ToFloat64(metrics.ProbesPassedTotal.WithLabelValues(region))

    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()
    _, _, _ = n.Acquire(ctx, RegionRequest{Region: region, StrictTimeout: 0, RelaxTimeout: 50 * time.Millisecond})

    if got := testutil.ToFloat64(metrics.OffersP3Total.WithLabelValues(region)); got < beforeP3+1 {
        t.Fatalf("offers p3: got %v want >= %v", got, beforeP3+1)
    }
    if got := testutil.ToFloat64(metrics.ProbesPassedTotal.WithLabelValues(region)); got < beforePass+1 {
        t.Fatalf("probes passed: got %v want >= %v", got, beforePass+1)
    }
}

func TestProbeFailureMetrics(t *testing.T) {
    resetNegotiationMetrics(t)
    region := "ASIA"
    src := &fakeSource{offers: []OfferView{{}}}
    filt := fakeFilter{lvl: P3NeedsProbe, claimed: ""}
    probe := fakeProbe{observed: "OTHER"} // mismatch
    n := NewNegotiatorWithProbe(src, filt, probe).(*negotiatorImpl)

    beforeFail := testutil.ToFloat64(metrics.ProbesFailedTotal.WithLabelValues(region))

    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
    defer cancel()
    _, _, _ = n.Acquire(ctx, RegionRequest{Region: region, StrictTimeout: 0, RelaxTimeout: 20 * time.Millisecond})

    if got := testutil.ToFloat64(metrics.ProbesFailedTotal.WithLabelValues(region)); got < beforeFail+1 {
        t.Fatalf("probes failed: got %v want >= %v", got, beforeFail+1)
    }
}
