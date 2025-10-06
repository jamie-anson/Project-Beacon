/**
 * Region Utilities
 * Pure functions for region code normalization and mapping
 */

/**
 * Normalize execution region into standard region code (US/EU/ASIA)
 * @param {Object} exec - The execution object
 * @returns {string} Normalized region code
 */
export function regionCodeFromExec(exec) {
  try {
    const raw = String(exec?.region || exec?.region_claimed || '').toLowerCase();
    if (!raw) return '';
    if (raw.includes('us') || raw.includes('united states')) return 'US';
    if (raw.includes('eu') || raw.includes('europe')) return 'EU';
    if (raw.includes('asia') || raw.includes('apac') || raw.includes('pacific')) return 'ASIA';
    return raw.toUpperCase();
  } catch {
    return '';
  }
}

/**
 * Map display region codes to database region names
 * @param {string} displayRegion - Display region code (US/EU/ASIA)
 * @returns {string} Database region name
 */
export function mapRegionToDatabase(displayRegion) {
  switch (displayRegion) {
    case 'US': return 'us-east';
    case 'EU': return 'eu-west';
    case 'ASIA': return 'asia-pacific';
    default: return displayRegion.toLowerCase();
  }
}

/**
 * Normalize region string to database format
 * @param {string} region - Region string
 * @returns {string} Normalized database region name
 */
export function normalizeRegion(region) {
  const v = String(region || '').toUpperCase();
  if (v === 'US') return 'us-east';
  if (v === 'EU') return 'eu-west';
  if (v === 'ASIA') return 'asia-pacific';
  return 'us-east';
}

/**
 * Group executions by region
 * @param {Array} executions - Array of execution objects
 * @param {Array} selectedRegions - Array of selected region codes
 * @returns {Object} Map of region code to executions
 */
export function groupExecutionsByRegion(executions = [], selectedRegions = []) {
  const grouped = {};
  
  // Initialize with selected regions
  selectedRegions.forEach(region => {
    grouped[region] = [];
  });
  
  // Group executions by region
  executions.forEach(exec => {
    const regionCode = regionCodeFromExec(exec);
    if (regionCode && selectedRegions.includes(regionCode)) {
      if (!grouped[regionCode]) {
        grouped[regionCode] = [];
      }
      grouped[regionCode].push(exec);
    }
  });
  
  return grouped;
}

/**
 * Filter regions that have executions or are selected
 * @param {Array} allRegions - All possible regions
 * @param {Array} executions - Array of execution objects
 * @param {Array} selectedRegions - Array of selected region codes
 * @returns {Array} Filtered region codes
 */
export function filterVisibleRegions(allRegions, executions = [], selectedRegions = []) {
  return allRegions.filter(region => {
    const regionExecs = executions.filter(x => regionCodeFromExec(x) === region);
    return regionExecs.length > 0 || selectedRegions.includes(region);
  });
}
