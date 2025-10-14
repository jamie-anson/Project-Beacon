import { useCallback, useEffect, useMemo, useState } from 'react';
import { getCrossRegionDiff } from '../lib/api/diffs/index.js';
import { getJob } from '../lib/api/runner/jobs.js';
import { transformCrossRegionDiff } from '../lib/diffs/transform.js';
import { generateMockDiffAnalysis } from '../lib/diffs/mock.js';
import { AVAILABLE_MODELS } from '../lib/diffs/constants.js';

const MOCK_ENABLED_KEY = 'beacon:enable_diff_mock';

export function useCrossRegionDiff(jobId, { pollInterval = 0, enableMock = false } = {}) {
  const [state, setState] = useState({
    loading: true,
    error: null,
    data: null,
    job: null,
    usingMock: false
  });

  const mockToggle = useMemo(() => {
    if (typeof window === 'undefined') return enableMock;
    try {
      const stored = window.localStorage.getItem(MOCK_ENABLED_KEY);
      if (stored === null) return enableMock;
      return stored === 'true';
    } catch {
      return enableMock;
    }
  }, [enableMock]);

  const fetchData = useCallback(async () => {
    if (!jobId) {
      setState({ loading: false, error: null, data: null, job: null, usingMock: false });
      return;
    }

    setState((prev) => ({ ...prev, loading: true, error: null }));

    try {
      console.log('🔍 useCrossRegionDiff: Fetching data for job:', jobId);
      const [job, diff] = await Promise.all([
        getJob({ id: jobId, include: 'executions' }),
        getCrossRegionDiff(jobId)
      ]);

      console.log('✅ useCrossRegionDiff: Data fetched successfully', {
        hasJob: !!job,
        hasDiff: !!diff,
        executionCount: job?.executions?.length
      });

      const transformed = transformCrossRegionDiff(diff, job, AVAILABLE_MODELS);
      console.log('✅ useCrossRegionDiff: Data transformed', {
        hasModels: !!transformed?.models,
        modelCount: transformed?.models?.length
      });

      setState({
        loading: false,
        error: null,
        data: transformed,
        job,
        usingMock: false
      });
    } catch (error) {
      console.error('❌ useCrossRegionDiff: Error fetching data:', error);
      if (mockToggle) {
        try {
          const job = await getJob({ id: jobId, include: 'executions' });
          const mock = generateMockDiffAnalysis(job, AVAILABLE_MODELS);
          setState({
            loading: false,
            error: null,
            data: mock,
            job,
            usingMock: true
          });
          return;
        } catch (mockError) {
          setState({
            loading: false,
            error: mockError,
            data: null,
            job: null,
            usingMock: false
          });
          return;
        }
      }

      setState({ loading: false, error, data: null, job: null, usingMock: false });
    }
  }, [jobId, mockToggle]);

  useEffect(() => {
    fetchData();
    if (!pollInterval) return undefined;
    const timer = setInterval(fetchData, pollInterval);
    return () => clearInterval(timer);
  }, [fetchData, pollInterval]);

  const retry = useCallback(() => {
    fetchData();
  }, [fetchData]);

  return useMemo(
    () => ({
      loading: state.loading,
      error: state.error,
      data: state.data,
      job: state.job,
      usingMock: state.usingMock,
      refetch: fetchData,
      retry
    }),
    [state, fetchData, retry]
  );
}
