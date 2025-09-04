package golem

// groupProvidersByRegion groups providers by their region.
func groupProvidersByRegion(providers []*Provider) map[string][]*Provider {
	m := make(map[string][]*Provider)
	for _, p := range providers {
		m[p.Region] = append(m[p.Region], p)
	}
	return m
}

// selectBestProviderInRegion selects the provider with the highest score in the slice.
func selectBestProviderInRegion(providers []*Provider) *Provider {
	var best *Provider
	for _, p := range providers {
		if best == nil || p.Score > best.Score {
			best = p
		}
	}
	return best
}

// selectBestPerRegion returns best provider per region from a grouped map.
func selectBestPerRegion(grouped map[string][]*Provider) map[string]*Provider {
	out := make(map[string]*Provider, len(grouped))
	for region, provs := range grouped {
		out[region] = selectBestProviderInRegion(provs)
	}
	return out
}
