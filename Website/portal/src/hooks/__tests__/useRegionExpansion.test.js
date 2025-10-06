/**
 * Unit tests for useRegionExpansion hook
 */

import { renderHook, act } from '@testing-library/react';
import { useRegionExpansion } from '../useRegionExpansion';

describe('useRegionExpansion', () => {
  it('should initialize with no expanded regions', () => {
    const { result } = renderHook(() => useRegionExpansion());

    expect(result.current.expandedRegions.size).toBe(0);
  });

  it('should toggle region expansion', () => {
    const { result } = renderHook(() => useRegionExpansion());

    act(() => {
      result.current.toggleRegion('US');
    });

    expect(result.current.expandedRegions.has('US')).toBe(true);
    expect(result.current.isExpanded('US')).toBe(true);

    act(() => {
      result.current.toggleRegion('US');
    });

    expect(result.current.expandedRegions.has('US')).toBe(false);
    expect(result.current.isExpanded('US')).toBe(false);
  });

  it('should expand a region', () => {
    const { result } = renderHook(() => useRegionExpansion());

    act(() => {
      result.current.expandRegion('EU');
    });

    expect(result.current.isExpanded('EU')).toBe(true);
  });

  it('should collapse a region', () => {
    const { result } = renderHook(() => useRegionExpansion());

    act(() => {
      result.current.expandRegion('ASIA');
    });

    expect(result.current.isExpanded('ASIA')).toBe(true);

    act(() => {
      result.current.collapseRegion('ASIA');
    });

    expect(result.current.isExpanded('ASIA')).toBe(false);
  });

  it('should expand all regions', () => {
    const { result } = renderHook(() => useRegionExpansion());
    const regions = ['US', 'EU', 'ASIA'];

    act(() => {
      result.current.expandAll(regions);
    });

    expect(result.current.isExpanded('US')).toBe(true);
    expect(result.current.isExpanded('EU')).toBe(true);
    expect(result.current.isExpanded('ASIA')).toBe(true);
    expect(result.current.expandedRegions.size).toBe(3);
  });

  it('should collapse all regions', () => {
    const { result } = renderHook(() => useRegionExpansion());
    const regions = ['US', 'EU', 'ASIA'];

    act(() => {
      result.current.expandAll(regions);
    });

    expect(result.current.expandedRegions.size).toBe(3);

    act(() => {
      result.current.collapseAll();
    });

    expect(result.current.expandedRegions.size).toBe(0);
  });

  it('should handle multiple regions independently', () => {
    const { result } = renderHook(() => useRegionExpansion());

    act(() => {
      result.current.expandRegion('US');
      result.current.expandRegion('EU');
    });

    expect(result.current.isExpanded('US')).toBe(true);
    expect(result.current.isExpanded('EU')).toBe(true);
    expect(result.current.isExpanded('ASIA')).toBe(false);

    act(() => {
      result.current.collapseRegion('US');
    });

    expect(result.current.isExpanded('US')).toBe(false);
    expect(result.current.isExpanded('EU')).toBe(true);
  });

  it('should not duplicate regions when expanding already expanded region', () => {
    const { result } = renderHook(() => useRegionExpansion());

    act(() => {
      result.current.expandRegion('US');
      result.current.expandRegion('US');
    });

    expect(result.current.expandedRegions.size).toBe(1);
  });

  it('should handle collapsing non-expanded region gracefully', () => {
    const { result } = renderHook(() => useRegionExpansion());

    act(() => {
      result.current.collapseRegion('US');
    });

    expect(result.current.expandedRegions.size).toBe(0);
  });

  it('should check expansion state correctly', () => {
    const { result } = renderHook(() => useRegionExpansion());

    expect(result.current.isExpanded('US')).toBe(false);

    act(() => {
      result.current.expandRegion('US');
    });

    expect(result.current.isExpanded('US')).toBe(true);
    expect(result.current.isExpanded('EU')).toBe(false);
  });

  it('should maintain state across multiple operations', () => {
    const { result } = renderHook(() => useRegionExpansion());

    act(() => {
      result.current.toggleRegion('US');
      result.current.expandRegion('EU');
      result.current.toggleRegion('US');
      result.current.expandRegion('ASIA');
    });

    expect(result.current.isExpanded('US')).toBe(false);
    expect(result.current.isExpanded('EU')).toBe(true);
    expect(result.current.isExpanded('ASIA')).toBe(true);
  });

  it('should handle empty array in expandAll', () => {
    const { result } = renderHook(() => useRegionExpansion());

    act(() => {
      result.current.expandAll([]);
    });

    expect(result.current.expandedRegions.size).toBe(0);
  });

  it('should replace expanded regions when calling expandAll', () => {
    const { result } = renderHook(() => useRegionExpansion());

    act(() => {
      result.current.expandRegion('US');
      result.current.expandRegion('EU');
    });

    expect(result.current.expandedRegions.size).toBe(2);

    act(() => {
      result.current.expandAll(['ASIA']);
    });

    expect(result.current.expandedRegions.size).toBe(1);
    expect(result.current.isExpanded('ASIA')).toBe(true);
    expect(result.current.isExpanded('US')).toBe(false);
  });
});
