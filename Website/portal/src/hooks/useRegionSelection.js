import React from 'react';

/**
 * Encapsulates region selection logic for the diff page
 * - Initializes active and compare regions when data arrives
 * - Defaults compare to home region if present, else first region
 */
export function useRegionSelection(regions, homeRegion) {
  const [activeRegion, setActiveRegion] = React.useState(null);
  const [compareRegion, setCompareRegion] = React.useState(null);

  React.useEffect(() => {
    if (!regions || regions.length === 0) return;

    if (!activeRegion) {
      setActiveRegion(regions[0].region_code);
    }

    if (!compareRegion) {
      const homeExists = regions.some(r => r.region_code === homeRegion);
      setCompareRegion(homeExists ? homeRegion : regions[0].region_code);
    }
  }, [regions, homeRegion, activeRegion, compareRegion]);

  return { activeRegion, setActiveRegion, compareRegion, setCompareRegion };
}
