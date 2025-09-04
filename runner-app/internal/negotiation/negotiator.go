package negotiation

import (
    "context"
    "errors"
    "time"
    metrics "github.com/jamie-anson/project-beacon-runner/internal/metrics"
)

// Negotiator orchestrates strict/relax acquisition and verification.
type Negotiator interface {
    Acquire(ctx context.Context, req RegionRequest) (Agreement, RegionVerification, error)
}

type negotiator struct{}

// OfferSource yields offers observed during negotiation.
type OfferSource interface {
    Next(ctx context.Context) (OfferView, error)
}

// Telemetry counters (lightweight; to be wired to metrics logger later).
type telemetry struct {
    offersSeen   int
    offersP0P2   int
    offersP3     int
    probesPassed int
    probesFailed int
}

type negotiatorImpl struct {
    src  OfferSource
    filt OfferFilter
    probe PreflightProbe // optional
    tlm  telemetry
}

func NewNegotiator(src OfferSource, filt OfferFilter) Negotiator {
    return &negotiatorImpl{src: src, filt: filt}
}

// NewNegotiatorWithProbe allows providing an optional PreflightProbe used in relax window.
func NewNegotiatorWithProbe(src OfferSource, filt OfferFilter, probe PreflightProbe) Negotiator {
    return &negotiatorImpl{src: src, filt: filt, probe: probe}
}

var (
    ErrNoOfferSource = errors.New("no offer source configured")
    ErrNoMatch       = errors.New("no matching offers found within time window")
    ErrNeedsProbe    = errors.New("offer requires preflight probe")
)

func (n *negotiatorImpl) Acquire(ctx context.Context, req RegionRequest) (Agreement, RegionVerification, error) {
    if n.src == nil || n.filt == nil {
        return nil, RegionVerification{}, ErrNoOfferSource
    }

    start := time.Now()
    outcome := "no_match"
    defer func() {
        metrics.NegotiationDurationSeconds.WithLabelValues(req.Region, outcome).Observe(time.Since(start).Seconds())
    }()

    strictUntil := time.Now().Add(req.StrictTimeout)
    relaxUntil := strictUntil.Add(req.RelaxTimeout)

    var bestLevel MatchLevel = P3NeedsProbe
    var bestClaimed string

    // Helper to consider an offer and update best selection.
    consider := func(ov OfferView) {
        n.tlm.offersSeen++
        metrics.OffersSeenTotal.WithLabelValues(req.Region).Inc()
        lvl, claimed := n.filt.Classify(ov, req.Region)
        switch lvl {
        case P0Explicit, P1Generic, P2Tags:
            n.tlm.offersP0P2++
            metrics.OffersP0P2Total.WithLabelValues(req.Region).Inc()
        case P3NeedsProbe:
            n.tlm.offersP3++
            metrics.OffersP3Total.WithLabelValues(req.Region).Inc()
        }
        if lvl < bestLevel { // lower enum value is higher priority
            bestLevel, bestClaimed = lvl, claimed
        }
    }

    // Strict window: only accept P0/P1/P2.
    for time.Now().Before(strictUntil) {
        ov, err := n.src.Next(ctx)
        if err != nil {
            // Ignore transient errors; continue until timeout or ctx done.
            select {
            case <-ctx.Done():
                break
            default:
            }
            continue
        }
        consider(ov)
        if bestLevel == P0Explicit || bestLevel == P1Generic || bestLevel == P2Tags {
            // Early exit with highest priority seen so far.
            rv := RegionVerification{Claimed: bestClaimed}
            outcome = "strict_match"
            return nil, rv, nil
        }
    }

    // Relax window: allow P3; if probe configured, verify and return on success; otherwise signal needs probe.
    for time.Now().Before(relaxUntil) {
        ov, err := n.src.Next(ctx)
        if err != nil {
            select {
            case <-ctx.Done():
                break
            default:
            }
            continue
        }
        consider(ov)
        if bestLevel != P3NeedsProbe {
            rv := RegionVerification{Claimed: bestClaimed}
            outcome = "relax_match"
            return nil, rv, nil
        }
        // If P3 is the best so far, try probe if available.
        if bestLevel == P3NeedsProbe {
            rv := RegionVerification{Claimed: bestClaimed}
            if n.probe == nil {
                outcome = "needs_probe"
                return nil, rv, ErrNeedsProbe
            }
            // AgreementID not yet established; pass empty or a placeholder until integrated with engine.
            observed, evidence, perr := n.probe.Verify(ctx, "")
            if perr != nil {
                n.tlm.probesFailed++
                metrics.ProbesFailedTotal.WithLabelValues(req.Region).Inc()
                continue
            }
            rv.Observed = observed
            rv.Method = "preflight-geoip"
            rv.Evidence = evidence
            if observed == req.Region {
                rv.Verified = true
                n.tlm.probesPassed++
                metrics.ProbesPassedTotal.WithLabelValues(req.Region).Inc()
                outcome = "probe_verified"
                return nil, rv, nil
            }
            // Mismatch; keep searching.
            n.tlm.probesFailed++
            metrics.ProbesFailedTotal.WithLabelValues(req.Region).Inc()
            continue
        }
    }

    outcome = "no_match"
    return nil, RegionVerification{}, ErrNoMatch
}
