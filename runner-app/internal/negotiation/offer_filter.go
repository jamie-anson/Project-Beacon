package negotiation

import "strings"

// OfferFilter classifies offers by region match strength.
type OfferFilter interface {
    Classify(offer OfferView, targetRegion string) (level MatchLevel, claimed string)
}

type offerFilter struct{}

func NewOfferFilter() OfferFilter { return &offerFilter{} }

func (f *offerFilter) Classify(offer OfferView, targetRegion string) (MatchLevel, string) {
    tr := normalizeRegion(targetRegion)
    if tr == "" {
        return P3NeedsProbe, ""
    }
    // P0: beacon.region exact match
    if r, ok := offer.Properties["beacon.region"]; ok && normalizeRegion(r) == tr {
        return P0Explicit, r
    }
    // P1: region/geo.region exact match
    if r, ok := offer.Properties["region"]; ok && normalizeRegion(r) == tr {
        return P1Generic, r
    }
    if r, ok := offer.Properties["geo.region"]; ok && normalizeRegion(r) == tr {
        return P1Generic, r
    }
    // P2: tags contain target label
    for _, t := range offer.Tags {
        if normalizeRegion(t) == tr {
            return P2Tags, t
        }
    }
    // P3: needs probe
    return P3NeedsProbe, ""
}

// normalizeRegion maps variations to canonical US/EU/ASIA strings.
func normalizeRegion(s string) string {
    switch val := strings.TrimSpace(strings.ToUpper(s)); val {
    case "US", "USA", "UNITED STATES":
        return "US"
    case "EU", "EUROPE", "EUROPEAN UNION":
        return "EU"
    case "ASIA", "APAC", "AS", "ASIAN":
        return "ASIA"
    default:
        return ""
    }
}
