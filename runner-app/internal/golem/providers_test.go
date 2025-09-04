package golem

import (
    "context"
    "testing"
    "time"

    "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func TestProviderMatching_ByRegionScorePriceLists(t *testing.T) {
    s := NewService("test", "testnet")
    // use generated mock providers
    providers := s.generateMockProviders()

    // pick one provider id from US and one from EU for list tests
    var usID, euID string
    for _, p := range providers {
        if p.Region == "US" && usID == "" { usID = p.ID }
        if p.Region == "EU" && euID == "" { euID = p.ID }
    }
    if usID == "" || euID == "" {
        t.Fatalf("expected mock providers to include US and EU regions")
    }

    // Build region filters
    filters := []models.ProviderFilter{
        { Region: "US", MinScore: 0.9 },
        { Region: "EU", MaxPrice: 0.6 },
        { Region: "US", Whitelist: []string{usID} },
        { Region: "EU", Blacklist: []string{euID} },
    }

    rf := buildRegionFilters(filters)

    // Validate that US/EU regions have entries
    if len(rf["US"]) == 0 || len(rf["EU"]) == 0 {
        t.Fatalf("expected region filters for US and EU")
    }

    // Ensure providerMatchesRegionFilters honors filters
    for _, p := range providers {
        ok := providerMatchesRegionFilters(s, p, rf)
        if p.Region == "US" {
            // Must be either whitelisted or meet min score
            if p.ID != usID && p.Score < 0.9 && ok {
                t.Errorf("US provider %s should have been filtered out (score=%.2f)", p.ID, p.Score)
            }
        }
        if p.Region == "EU" {
            if p.ID == euID && ok {
                t.Errorf("EU provider %s is blacklisted but was accepted", p.ID)
            }
            if p.Price > 0.6 && ok {
                t.Errorf("EU provider %s price=%.2f exceeds max 0.6 but was accepted", p.ID, p.Price)
            }
        }
    }
}

func TestDiscoverProviders_MinRegionsAndConstraints(t *testing.T) {
    s := NewService("test", "testnet")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    cons := models.ExecutionConstraints{
        Regions:    []string{"US", "EU", "APAC"},
        MinRegions: 3,
        Providers: []models.ProviderFilter{
            { Region: "US", MinScore: 0.5 },
            { Region: "EU", MaxPrice: 1.0 },
            { Region: "APAC" },
        },
    }

    provs, err := s.DiscoverProviders(ctx, cons)
    if err != nil {
        t.Fatalf("DiscoverProviders failed: %v", err)
    }

    if len(provs) == 0 {
        t.Fatalf("expected at least one provider after filtering")
    }

    // Ensure only requested regions are included
    for _, p := range provs {
        if p.Region != "US" && p.Region != "EU" && p.Region != "APAC" {
            t.Errorf("unexpected region returned: %s", p.Region)
        }
    }
}

func TestDiscoverProviders_InsufficientRegions(t *testing.T) {
    s := NewService("test", "testnet")
    ctx := context.Background()

    cons := models.ExecutionConstraints{
        Regions:    []string{"MARS", "MOON"},
        MinRegions: 2,
    }

    _, err := s.DiscoverProviders(ctx, cons)
    if err == nil {
        t.Fatalf("expected error for insufficient providers by region, got nil")
    }
}
