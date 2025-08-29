package negotiation

import "time"

// RegionRequest captures acquisition requirements for a target region.
type RegionRequest struct {
    Region            string        // "US" | "EU" | "ASIA"
    MinVCPU           int
    MinMemGiB         int
    NetworkEgress     bool
    PricePerMinuteMax float64
    TotalPriceCap     float64
    StrictTimeout     time.Duration // e.g., 60s
    RelaxTimeout      time.Duration // e.g., 30s
}

// MatchLevel expresses how confidently an offer matches the target region.
type MatchLevel int

const (
    P0Explicit  MatchLevel = iota // beacon.region exact match
    P1Generic                     // region/geo.region exact match
    P2Tags                        // tags contain target label
    P3NeedsProbe                  // no explicit region; requires preflight
)

// DemandSpec captures hard constraints we send to the market.
type DemandSpec struct {
    Runtime           string  // e.g., "docker"
    MinVCPU           int
    MinMemGiB         int
    NetworkEgress     bool
    PricePerMinuteMax float64
    TotalPriceCap     float64
    Region            string // target region (soft; used by requestor filtering)
}

// OfferView is a normalized view of an offer's properties and tags used for filtering.
type OfferView struct {
    Properties map[string]string // e.g., {"beacon.region":"US", "region":"US"}
    Tags       []string          // e.g., ["US", "gpu", "fast"]
}

// Agreement is a placeholder for the acquired agreement reference.
type Agreement interface{}

// Evidence holds proof material from the preflight probe.
type Evidence struct {
    PublicIP    string
    GeoSource   string // e.g., "geolite2-city@2025-08-01"
    Country     string
    Timestamp   time.Time
    EvidenceRef string // optional: CID/URL to persisted probe payload
}

// RegionVerification summarizes claimed vs observed region.
type RegionVerification struct {
    Claimed  string
    Observed string
    Verified bool
    Method   string // e.g., "preflight-geoip"
    Evidence Evidence
}
