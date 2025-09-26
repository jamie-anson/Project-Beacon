import { useCallback, useEffect, useMemo, useState } from 'react';
import { listRecentDiffs } from '../lib/api/diffs/index.js';

export function useRecentDiffs({ limit = 10, pollInterval = 15000 } = {}) {
  const [state, setState] = useState({
    loading: true,
    error: null,
    data: []
  });

  const fetchDiffs = useCallback(async () => {
    try {
      setState((prev) => ({ ...prev, loading: true, error: null }));
      const data = await listRecentDiffs({ limit });
      setState({ loading: false, error: null, data });
    } catch (error) {
      setState({ loading: false, error, data: [] });
    }
  }, [limit]);

  useEffect(() => {
    fetchDiffs();
    if (!pollInterval) return undefined;
    const timer = setInterval(fetchDiffs, pollInterval);
    return () => clearInterval(timer);
  }, [fetchDiffs, pollInterval]);

  const retry = useCallback(() => {
    fetchDiffs();
  }, [fetchDiffs]);

  return useMemo(
    () => ({
      loading: state.loading,
      error: state.error,
      data: state.data,
      refetch: fetchDiffs,
      retry
    }),
    [state, fetchDiffs, retry]
  );
}
