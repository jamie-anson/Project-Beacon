import { useCallback, useEffect, useMemo, useState } from 'react';
import { runnerFetch } from '../lib/api/http.js';
import { transformModelRegionDiff } from '../lib/diffs/modelDiffTransform.js';
import { generateMockModelDiff } from '../lib/diffs/mockModelDiff.js';
import { decodeQuestionId } from '../lib/diffs/questionId.js';

const MOCK_ENABLED_KEY = 'beacon:enable_model_diff_mock';

/**
 * Fetches and transforms cross-region diff data for a single model and question
 * @param {string} jobId - Job ID
 * @param {string} modelId - Model ID (e.g., 'llama3.2-1b')
 * @param {string} questionId - URL-encoded question text
 * @param {Object} options - Hook options
 * @param {number} options.pollInterval - Polling interval in ms (0 = no polling)
 * @param {boolean} options.enableMock - Enable mock data fallback
 * @returns {Object} { loading, error, data, job, usingMock, refetch, retry }
 */
export function useModelRegionDiff(jobId, modelId, questionId, { pollInterval = 0, enableMock = false } = {}) {
  const [state, setState] = useState({
    loading: true,
    error: null,
    data: null,
    job: null,
    usingMock: false
  });

  // Check if mock mode is enabled via localStorage
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
    if (!jobId || !modelId || !questionId) {
      setState({ loading: false, error: null, data: null, job: null, usingMock: false });
      return;
    }

    setState((prev) => ({ ...prev, loading: true, error: null }));

    try {
      // Decode question ID from hyphenated format
      const decodedQuestion = decodeQuestionId(questionId);
      
      // Fetch cross-region execution data for specific model and question
      const crossRegionData = await runnerFetch(`/executions/${jobId}/cross-region?model_id=${modelId}&question_id=${decodedQuestion}`);

      // Transform data for this specific model and question
      const transformed = transformModelRegionDiff(crossRegionData, modelId, decodedQuestion);

      if (!transformed) {
        throw new Error(`No data found for model ${modelId} and question "${decodedQuestion}"`);
      }

      setState({
        loading: false,
        error: null,
        data: transformed,
        job: crossRegionData.cross_region_execution,
        usingMock: false
      });
    } catch (error) {
      console.error('Failed to fetch model region diff:', error);

      // Try mock data if enabled
      if (mockToggle) {
        try {
          const decodedQuestion = decodeQuestionId(questionId);
          const mock = generateMockModelDiff(modelId, decodedQuestion);
          
          setState({
            loading: false,
            error: null,
            data: mock,
            job: { id: jobId, status: 'completed' },
            usingMock: true
          });
          
          console.warn('Using mock data for model region diff');
          return;
        } catch (mockError) {
          console.error('Mock data generation failed:', mockError);
        }
      }

      setState({ 
        loading: false, 
        error, 
        data: null, 
        job: null, 
        usingMock: false 
      });
    }
  }, [jobId, modelId, questionId, mockToggle]);

  // Initial fetch and polling
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
