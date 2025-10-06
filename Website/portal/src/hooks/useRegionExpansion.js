/**
 * useRegionExpansion Hook
 * Manages expanded/collapsed state for region rows
 */

import { useState } from 'react';

/**
 * Custom hook for managing region expansion state
 * @returns {Object} Expansion state and functions
 */
export function useRegionExpansion() {
  const [expandedRegions, setExpandedRegions] = useState(new Set());
  
  /**
   * Toggle expansion state for a region
   * @param {string} region - The region code to toggle
   */
  const toggleRegion = (region) => {
    setExpandedRegions(prev => {
      const newExpanded = new Set(prev);
      if (newExpanded.has(region)) {
        newExpanded.delete(region);
      } else {
        newExpanded.add(region);
      }
      return newExpanded;
    });
  };
  
  /**
   * Check if a region is expanded
   * @param {string} region - The region code to check
   * @returns {boolean}
   */
  const isExpanded = (region) => {
    return expandedRegions.has(region);
  };
  
  /**
   * Expand a specific region
   * @param {string} region - The region code to expand
   */
  const expandRegion = (region) => {
    setExpandedRegions(prev => new Set(prev).add(region));
  };
  
  /**
   * Collapse a specific region
   * @param {string} region - The region code to collapse
   */
  const collapseRegion = (region) => {
    setExpandedRegions(prev => {
      const newExpanded = new Set(prev);
      newExpanded.delete(region);
      return newExpanded;
    });
  };
  
  /**
   * Expand all regions
   * @param {Array} regions - Array of region codes
   */
  const expandAll = (regions) => {
    setExpandedRegions(new Set(regions));
  };
  
  /**
   * Collapse all regions
   */
  const collapseAll = () => {
    setExpandedRegions(new Set());
  };
  
  return {
    expandedRegions,
    toggleRegion,
    isExpanded,
    expandRegion,
    collapseRegion,
    expandAll,
    collapseAll
  };
}
