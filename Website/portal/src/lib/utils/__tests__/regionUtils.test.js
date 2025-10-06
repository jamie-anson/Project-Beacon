/**
 * Unit tests for regionUtils
 */

import {
  regionCodeFromExec,
  mapRegionToDatabase,
  normalizeRegion,
  groupExecutionsByRegion,
  filterVisibleRegions
} from '../regionUtils';

describe('regionUtils', () => {
  describe('regionCodeFromExec', () => {
    it('should return US for us-east region', () => {
      expect(regionCodeFromExec({ region: 'us-east' })).toBe('US');
    });

    it('should return US for united states', () => {
      expect(regionCodeFromExec({ region: 'united states' })).toBe('US');
    });

    it('should return EU for eu-west region', () => {
      expect(regionCodeFromExec({ region: 'eu-west' })).toBe('EU');
    });

    it('should return EU for europe', () => {
      expect(regionCodeFromExec({ region: 'europe' })).toBe('EU');
    });

    it('should return ASIA for asia-pacific region', () => {
      expect(regionCodeFromExec({ region: 'asia-pacific' })).toBe('ASIA');
    });

    it('should return ASIA for apac', () => {
      expect(regionCodeFromExec({ region: 'apac' })).toBe('ASIA');
    });

    it('should use region_claimed if region is not available', () => {
      expect(regionCodeFromExec({ region_claimed: 'us-east' })).toBe('US');
    });

    it('should return empty string for null execution', () => {
      expect(regionCodeFromExec(null)).toBe('');
    });

    it('should return empty string for execution without region', () => {
      expect(regionCodeFromExec({})).toBe('');
    });

    it('should uppercase unknown regions', () => {
      expect(regionCodeFromExec({ region: 'canada' })).toBe('CANADA');
    });
  });

  describe('mapRegionToDatabase', () => {
    it('should map US to us-east', () => {
      expect(mapRegionToDatabase('US')).toBe('us-east');
    });

    it('should map EU to eu-west', () => {
      expect(mapRegionToDatabase('EU')).toBe('eu-west');
    });

    it('should map ASIA to asia-pacific', () => {
      expect(mapRegionToDatabase('ASIA')).toBe('asia-pacific');
    });

    it('should lowercase unknown regions', () => {
      expect(mapRegionToDatabase('CANADA')).toBe('canada');
    });
  });

  describe('normalizeRegion', () => {
    it('should normalize US to us-east', () => {
      expect(normalizeRegion('US')).toBe('us-east');
    });

    it('should normalize lowercase us to us-east', () => {
      expect(normalizeRegion('us')).toBe('us-east');
    });

    it('should normalize EU to eu-west', () => {
      expect(normalizeRegion('EU')).toBe('eu-west');
    });

    it('should normalize ASIA to asia-pacific', () => {
      expect(normalizeRegion('ASIA')).toBe('asia-pacific');
    });

    it('should default to us-east for unknown regions', () => {
      expect(normalizeRegion('UNKNOWN')).toBe('us-east');
    });

    it('should default to us-east for null', () => {
      expect(normalizeRegion(null)).toBe('us-east');
    });
  });

  describe('groupExecutionsByRegion', () => {
    it('should group executions by region', () => {
      const executions = [
        { id: 1, region: 'us-east' },
        { id: 2, region: 'eu-west' },
        { id: 3, region: 'us-east' }
      ];
      const selectedRegions = ['US', 'EU'];
      
      const result = groupExecutionsByRegion(executions, selectedRegions);
      
      expect(result.US).toHaveLength(2);
      expect(result.EU).toHaveLength(1);
    });

    it('should initialize empty arrays for selected regions', () => {
      const executions = [];
      const selectedRegions = ['US', 'EU', 'ASIA'];
      
      const result = groupExecutionsByRegion(executions, selectedRegions);
      
      expect(result.US).toEqual([]);
      expect(result.EU).toEqual([]);
      expect(result.ASIA).toEqual([]);
    });

    it('should only include executions from selected regions', () => {
      const executions = [
        { id: 1, region: 'us-east' },
        { id: 2, region: 'eu-west' },
        { id: 3, region: 'asia-pacific' }
      ];
      const selectedRegions = ['US', 'EU'];
      
      const result = groupExecutionsByRegion(executions, selectedRegions);
      
      expect(result.US).toHaveLength(1);
      expect(result.EU).toHaveLength(1);
      expect(result.ASIA).toBeUndefined();
    });

    it('should handle empty executions array', () => {
      const result = groupExecutionsByRegion([], ['US']);
      expect(result.US).toEqual([]);
    });

    it('should handle empty selectedRegions array', () => {
      const executions = [{ id: 1, region: 'us-east' }];
      const result = groupExecutionsByRegion(executions, []);
      expect(Object.keys(result)).toHaveLength(0);
    });
  });

  describe('filterVisibleRegions', () => {
    it('should include regions with executions', () => {
      const allRegions = ['US', 'EU', 'ASIA'];
      const executions = [{ region: 'us-east' }];
      const selectedRegions = [];
      
      const result = filterVisibleRegions(allRegions, executions, selectedRegions);
      
      expect(result).toContain('US');
    });

    it('should include selected regions even without executions', () => {
      const allRegions = ['US', 'EU', 'ASIA'];
      const executions = [];
      const selectedRegions = ['EU'];
      
      const result = filterVisibleRegions(allRegions, executions, selectedRegions);
      
      expect(result).toContain('EU');
    });

    it('should exclude regions without executions and not selected', () => {
      const allRegions = ['US', 'EU', 'ASIA'];
      const executions = [{ region: 'us-east' }];
      const selectedRegions = ['US'];
      
      const result = filterVisibleRegions(allRegions, executions, selectedRegions);
      
      expect(result).toContain('US');
      expect(result).not.toContain('EU');
      expect(result).not.toContain('ASIA');
    });

    it('should handle empty executions and selectedRegions', () => {
      const allRegions = ['US', 'EU', 'ASIA'];
      const result = filterVisibleRegions(allRegions, [], []);
      
      expect(result).toEqual([]);
    });

    it('should include regions with executions and selected', () => {
      const allRegions = ['US', 'EU'];
      const executions = [{ region: 'us-east' }];
      const selectedRegions = ['US', 'EU'];
      
      const result = filterVisibleRegions(allRegions, executions, selectedRegions);
      
      expect(result).toContain('US');
      expect(result).toContain('EU');
    });
  });
});
