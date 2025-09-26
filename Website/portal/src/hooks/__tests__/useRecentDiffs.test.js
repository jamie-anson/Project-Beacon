import { renderHook, waitFor } from '@testing-library/react';
import { useRecentDiffs } from '../useRecentDiffs.js';

jest.mock('../../lib/api/diffs/index.js', () => ({
  listRecentDiffs: jest.fn()
}));

const { listRecentDiffs } = await import('../../lib/api/diffs/index.js');

describe('useRecentDiffs', () => {
  beforeEach(() => {
    jest.resetAllMocks();
  });

  test('loads recent diffs on success', async () => {
    const payload = [
      {
        id: 'diff-1',
        created_at: '2025-09-25T00:00:00Z',
        similarity: 0.78,
        a: { region: 'US', text: 'Response A' },
        b: { region: 'EU', text: 'Response B' }
      }
    ];
    listRecentDiffs.mockResolvedValue(payload);

    const { result } = renderHook(() => useRecentDiffs({ limit: 5, pollInterval: 0 }));

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.error).toBeNull();
    expect(result.current.data).toEqual(payload);
    expect(listRecentDiffs).toHaveBeenCalledWith({ limit: 5 });
  });

  test('captures error state when API fails', async () => {
    const err = new Error('Network error');
    listRecentDiffs.mockRejectedValue(err);

    const { result } = renderHook(() => useRecentDiffs({ limit: 5, pollInterval: 0 }));

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.error).toBe(err);
    expect(result.current.data).toEqual([]);
  });
});
