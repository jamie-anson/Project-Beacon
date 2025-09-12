package golem

import (
    "context"
    "fmt"

    "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// DiscoverProviders finds suitable providers based on execution constraints
func (s *Service) DiscoverProviders(ctx context.Context, constraints models.ExecutionConstraints) ([]*Provider, error) {
    // Get all available providers
    allProviders, err := s.getAvailableProviders(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get providers: %w", err)
    }

    // Edge case: No providers available
    if len(allProviders) == 0 {
        return nil, fmt.Errorf("no providers available in the network")
    }

    // Edge case: No regions specified
    if len(constraints.Regions) == 0 {
        return nil, fmt.Errorf("no target regions specified in constraints")
    }

    // Build region-specific filters for efficient matching
    regionFilters := buildRegionFilters(constraints.Providers)

    // Filter providers based on constraints
    var filteredProviders []*Provider
    regionCount := make(map[string]int)
    regionErrors := make(map[string][]string)

    for _, provider := range allProviders {
        // Skip providers not in required regions
        if !containsString(constraints.Regions, provider.Region) {
            continue
        }

        // Apply provider filters with detailed error tracking
        matches, reason := providerMatchesRegionFiltersDetailed(s, provider, regionFilters)
        if !matches {
            regionErrors[provider.Region] = append(regionErrors[provider.Region], 
                fmt.Sprintf("Provider %s: %s", provider.ID, reason))
            continue
        }

        filteredProviders = append(filteredProviders, provider)
        regionCount[provider.Region]++
    }

    // Edge case: No providers found in any region
    if len(filteredProviders) == 0 {
        var errorDetails []string
        for region, errors := range regionErrors {
            errorDetails = append(errorDetails, fmt.Sprintf("Region %s: %v", region, errors))
        }
        return nil, fmt.Errorf("no suitable providers found in any region. Details: %v", errorDetails)
    }

    // Edge case: Insufficient regions with providers
    if len(regionCount) < constraints.MinRegions {
        var availableRegions []string
        for region := range regionCount {
            availableRegions = append(availableRegions, region)
        }
        return nil, fmt.Errorf("insufficient providers: found %d regions %v, need %d", 
            len(regionCount), availableRegions, constraints.MinRegions)
    }

    // Edge case: Some regions have no providers after filtering
    var regionsWithoutProviders []string
    for _, region := range constraints.Regions {
        if regionCount[region] == 0 {
            regionsWithoutProviders = append(regionsWithoutProviders, region)
        }
    }
    
    // Log warning for regions without providers but don't fail if we have enough
    if len(regionsWithoutProviders) > 0 && len(regionCount) >= constraints.MinRegions {
        // This is acceptable - we have minimum required regions
        // Could add logging here if needed
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

// getAvailableProviders retrieves all available providers from the network
func (s *Service) getAvailableProviders(ctx context.Context) ([]*Provider, error) {
    // For now, return mock providers - this would integrate with actual Golem network discovery
    return s.getMockProviders(), nil
}

// getMockProviders returns mock providers for testing/development
func (s *Service) getMockProviders() []*Provider {
    // Use the same providers as generateMockProviders to ensure consistency
    return s.generateMockProviders()
}

// containsString checks if a string slice contains a specific string
func containsString(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}

// providerMatchesRegionFiltersDetailed checks if provider matches filters with detailed error reporting
func providerMatchesRegionFiltersDetailed(s *Service, provider *Provider, regionFilters map[string][]models.ProviderFilter) (bool, string) {
    filters, ok := regionFilters[provider.Region]
    if !ok || len(filters) == 0 {
        return true, ""
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
        return false, "provider is blacklisted"
    }

    // If region whitelist exists and includes provider, accept immediately
    if len(wl) > 0 {
        if _, ok := wl[provider.ID]; ok {
            return true, ""
        }
        // If whitelist exists but provider not in it, still allow other filters to match
    }

    // Check if provider matches at least one filter's constraints
    for _, filter := range filters {
        matches, reason := matchesProviderFilterDetailed(s, provider, filter)
        if matches {
            return true, ""
        }
        if reason != "" {
            return false, reason
        }
    }
    
    return false, "provider does not match any filter constraints"
}

// matchesProviderFilterDetailed checks if a provider matches the given filter with detailed error reporting
func matchesProviderFilterDetailed(s *Service, provider *Provider, filter models.ProviderFilter) (bool, string) {
    // Check region
    if filter.Region != "" && provider.Region != filter.Region {
        return false, fmt.Sprintf("region mismatch: expected %s, got %s", filter.Region, provider.Region)
    }
    
    // Check minimum score
    if filter.MinScore > 0 && provider.Score < filter.MinScore {
        return false, fmt.Sprintf("score too low: %.2f < %.2f", provider.Score, filter.MinScore)
    }
    
    // Check maximum price
    if filter.MaxPrice > 0 && provider.Price > filter.MaxPrice {
        return false, fmt.Sprintf("price too high: %.4f > %.4f", provider.Price, filter.MaxPrice)
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
            return false, "provider not in whitelist"
        }
    }
    
    // Check blacklist
    for _, blacklisted := range filter.Blacklist {
        if provider.ID == blacklisted {
            return false, "provider is blacklisted"
        }
    }
    
    return true, ""
}
