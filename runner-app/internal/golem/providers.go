package golem

import (
    "context"
    "fmt"

    "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// DiscoverProviders finds available providers matching the given constraints
func (s *Service) DiscoverProviders(ctx context.Context, constraints models.ExecutionConstraints) ([]*Provider, error) {
    // If using SDK backend, delegate to SDK implementation
    if s.backend == "sdk" {
        return s.discoverProvidersSDK(ctx, constraints)
    }

    // Default: simulate provider discovery (mock backend)
    providers := s.generateMockProviders()

    // Build region->filters map so that filters apply only to regions that specify them
    regionFilters := buildRegionFilters(constraints.Providers)

    // Filter providers based on constraints
    var filteredProviders []*Provider
    regionCount := make(map[string]int)

    for _, provider := range providers {
        // Check if provider's region is in required regions
        regionRequired := false
        for _, requiredRegion := range constraints.Regions {
            if provider.Region == requiredRegion {
                regionRequired = true
                break
            }
        }
        if !regionRequired {
            continue
        }

        // Apply filters only if this region has filters specified; otherwise accept provider
        if !providerMatchesRegionFilters(s, provider, regionFilters) {
            continue
        }

        filteredProviders = append(filteredProviders, provider)
        regionCount[provider.Region]++
    }

    // Ensure we have minimum required regions
    if len(regionCount) < constraints.MinRegions {
        return nil, fmt.Errorf("insufficient providers: found %d regions, need %d", len(regionCount), constraints.MinRegions)
    }

    return filteredProviders, nil
}

// buildRegionFilters groups provider filters by region for efficient matching
func buildRegionFilters(filters []models.ProviderFilter) map[string][]models.ProviderFilter {
    regionFilters := make(map[string][]models.ProviderFilter)
    for _, f := range filters {
        regionFilters[f.Region] = append(regionFilters[f.Region], f)
    }
    return regionFilters
}

// providerMatchesRegionFilters determines whether a provider should be included
// based on filters defined for its region. If no filters exist for the region,
// the provider is accepted by default.
func providerMatchesRegionFilters(s *Service, provider *Provider, regionFilters map[string][]models.ProviderFilter) bool {
    filters, ok := regionFilters[provider.Region]
    if !ok || len(filters) == 0 {
        return true
    }

    // Aggregate region-wide blacklists/whitelists
    bl := make(map[string]struct{})
    wl := make(map[string]struct{})
    for _, f := range filters {
        for _, id := range f.Blacklist { bl[id] = struct{}{} }
        for _, id := range f.Whitelist { wl[id] = struct{}{} }
    }

    // Region blacklist overrides everything
    if _, banned := bl[provider.ID]; banned {
        return false
    }

    // If region whitelist exists and includes provider, accept immediately
    if len(wl) > 0 {
        if _, ok := wl[provider.ID]; ok {
            return true
        }
        // If whitelist exists but provider not in it, still allow other filters to match
        // (some filters may not specify whitelist and still be valid constraints)
    }

    // Accept if provider matches at least one filter's constraints
    for _, filter := range filters {
        if s.matchesProviderFilter(provider, filter) {
            return true
        }
    }
    return false
}

// matchesProviderFilter checks if a provider matches the given filter
func (s *Service) matchesProviderFilter(provider *Provider, filter models.ProviderFilter) bool {
    // Check region
    if filter.Region != "" && provider.Region != filter.Region {
        return false
    }
    // Check minimum score
    if filter.MinScore > 0 && provider.Score < filter.MinScore {
        return false
    }
    // Check maximum price
    if filter.MaxPrice > 0 && provider.Price > filter.MaxPrice {
        return false
    }
    // Check whitelist
    if len(filter.Whitelist) > 0 {
        found := false
        for _, whitelisted := range filter.Whitelist {
            if provider.ID == whitelisted {
                found = true
                break
            }
        }
        if !found {
            return false
        }
    }
    // Check blacklist
    for _, blacklisted := range filter.Blacklist {
        if provider.ID == blacklisted {
            return false
        }
    }
    return true
}
