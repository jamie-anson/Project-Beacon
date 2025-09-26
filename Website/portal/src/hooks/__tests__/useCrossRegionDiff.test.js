import { renderHook, waitFor } from '@testing-library/react';
import { useCrossRegionDiff } from '../useCrossRegionDiff.js';
import { AVAILABLE_MODELS } from '../../lib/diffs/constants.js';

jest.mock('../../lib/api/runner/jobs.js', () => ({
  getJob: jest.fn()
}));

jest.mock('../../lib/api/diffs/index.js', () => ({
  getCrossRegionDiff: jest.fn()
}));

const { getJob } = await import('../../lib/api/runner/jobs.js');
const { getCrossRegionDiff } = await import('../../lib/api/diffs/index.js');

const JOB_FIXTURE = {
  id: 'job-123',
  jobspec: {
    questions: [
      { question: 'What happened at Tiananmen Square?' }
    ]
  },
  executions: [
    { region: 'US', status: 'completed', provider_id: 'provider-us' }
  ]
};

const DIFF_FIXTURE = {
  job_id: 'job-123',
  question: { text: 'What happened at Tiananmen Square?' },
  analysis: {
    bias_variance: 0.42,
    censorship_rate: 0.12,
    factual_consistency: 0.9,
    narrative_divergence: 0.33
  },
  executions: [
    {
      region: 'US',
      status: 'completed',
      provider_id: 'provider-us',
      output: {
        responses: [{ response: 'Detailed response text.' }]
      }
    }
  ]
};

function clearMockToggle() {
  try {
    window.localStorage.removeItem('beacon:enable_diff_mock');
  } catch {
    // ignore
  }
}

describe('useCrossRegionDiff', () => {
  beforeEach(() => {
    jest.resetAllMocks();
    clearMockToggle();
  });

  test('loads diff analysis data and job details on success', async () => {
    getJob.mockResolvedValue(JOB_FIXTURE);
    getCrossRegionDiff.mockResolvedValue(DIFF_FIXTURE);

    const { result } = renderHook(() => useCrossRegionDiff('job-123'));

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.error).toBeNull();
    expect(result.current.job).toEqual(JOB_FIXTURE);
    expect(result.current.usingMock).toBe(false);
    expect(result.current.data?.models).toHaveLength(AVAILABLE_MODELS.length);
    expect(result.current.data?.question).toBe('What happened at Tiananmen Square?');
  });

  test('falls back to mock data when API fails and mock toggle enabled', async () => {
    window.localStorage.setItem('beacon:enable_diff_mock', 'true');
    getJob.mockResolvedValue(JOB_FIXTURE);
    getCrossRegionDiff.mockRejectedValue(new Error('Diff API down'));

    const { result } = renderHook(() => useCrossRegionDiff('job-123'));

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.error).toBeNull();
    expect(result.current.usingMock).toBe(true);
    expect(result.current.data).not.toBeNull();
    expect(result.current.data?.models).toHaveLength(AVAILABLE_MODELS.length);
  });

  test('surfaces error when API fails and mock toggle disabled', async () => {
    getJob.mockResolvedValue(JOB_FIXTURE);
    const networkError = new Error('Diff API unavailable');
    getCrossRegionDiff.mockRejectedValue(networkError);

    const { result } = renderHook(() => useCrossRegionDiff('job-123'));

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.error).toBe(networkError);
    expect(result.current.data).toBeNull();
    expect(result.current.job).toBeNull();
    expect(result.current.usingMock).toBe(false);
  });
});
