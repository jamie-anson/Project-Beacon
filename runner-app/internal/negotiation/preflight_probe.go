package negotiation

import (
    "context"
    "errors"
    "fmt"
    "time"
)

// PreflightProbe verifies provider region via egress IP + GeoIP.
type PreflightProbe interface {
    Verify(ctx context.Context, agreementID string) (observedRegion string, evidence Evidence, err error)
}

// ipFetcher returns the public IP address observed when contacting the provider.
// agreementID can be used to address a specific endpoint when applicable.
type ipFetcher func(ctx context.Context, agreementID string) (string, error)

type geoResolver interface {
    LookupIP(ip string) (country string, regionBucket string, source string, err error)
}

type preflightProbe struct {
    fetch ipFetcher
    geo   geoResolver
    now   func() time.Time
}

// NewPreflightProbe creates a probe with provided dependencies.
func NewPreflightProbe(fetch ipFetcher, geo geoResolver) PreflightProbe {
    if fetch == nil {
        fetch = func(ctx context.Context, _ string) (string, error) {
            return "", errors.New("no ip fetcher configured")
        }
    }
    if geo == nil {
        geo = nil
    }
    return &preflightProbe{fetch: fetch, geo: geo, now: time.Now}
}

func (p *preflightProbe) Verify(ctx context.Context, agreementID string) (string, Evidence, error) {
    if p.fetch == nil || p.geo == nil {
        return "", Evidence{Timestamp: p.now(), GeoSource: "uninitialized"}, errors.New("preflight probe not configured")
    }
    ip, err := p.fetch(ctx, agreementID)
    if err != nil {
        return "", Evidence{Timestamp: p.now(), GeoSource: "uninitialized"}, fmt.Errorf("failed to fetch egress ip: %w", err)
    }
    country, bucket, source, err := p.geo.LookupIP(ip)
    if err != nil {
        return "", Evidence{PublicIP: ip, Timestamp: p.now(), GeoSource: source}, fmt.Errorf("geoip lookup failed: %w", err)
    }
    ev := Evidence{
        PublicIP:  ip,
        GeoSource: source,
        Country:   country,
        Timestamp: p.now(),
    }
    return bucket, ev, nil
}
