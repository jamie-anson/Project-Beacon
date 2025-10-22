import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import BiasDetectionResults from '../BiasDetectionResults';
import * as api from '../../lib/api';

jest.mock('../../lib/api');

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useParams: () => ({ jobId: 'test-job-123' }),
}));

describe('BiasDetectionResults - LLM Summary Rendering', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('renders LLM summary when analysis includes summary field', async () => {
    const mockAnalysis = {
      job_id: 'test-job-123',
      analysis: {
        bias_variance: 0.68,
        censorship_rate: 0.42,
        summary: 'Cross-region analysis completed with significant findings. High censorship detected in 67% of regions.',
      },
      region_scores: {
        us_east: { bias_score: 0.15 },
        asia_pacific: { bias_score: 0.78 },
      },
    };

    api.getBiasAnalysis = jest.fn().mockResolvedValue(mockAnalysis);

    render(
      <BrowserRouter>
        <BiasDetectionResults />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText(/Cross-region analysis completed/i)).toBeInTheDocument();
    });

    expect(screen.getByText(/High censorship detected/i)).toBeInTheDocument();
    expect(api.getBiasAnalysis).toHaveBeenCalledWith('test-job-123');
  });

  test('renders fallback text when summary is missing', async () => {
    const mockAnalysis = {
      job_id: 'test-job-123',
      analysis: {
        bias_variance: 0.68,
        censorship_rate: 0.42,
      },
      region_scores: {
        us_east: { bias_score: 0.15 },
      },
    };

    api.getBiasAnalysis = jest.fn().mockResolvedValue(mockAnalysis);

    render(
      <BrowserRouter>
        <BiasDetectionResults />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.queryByText(/Cross-region analysis completed/i)).not.toBeInTheDocument();
    });

    expect(screen.getByText(/No summary available/i) || screen.getByText(/Analysis in progress/i)).toBeInTheDocument();
  });

  test('handles API error gracefully', async () => {
    api.getBiasAnalysis = jest.fn().mockRejectedValue(new Error('API error'));

    render(
      <BrowserRouter>
        <BiasDetectionResults />
      </BrowserRouter>
    );

    await waitFor(() => {
      expect(screen.getByText(/error/i) || screen.getByText(/failed/i)).toBeInTheDocument();
    });
  });

  test('displays loading state before data arrives', () => {
    api.getBiasAnalysis = jest.fn().mockImplementation(() => new Promise(() => {}));

    render(
      <BrowserRouter>
        <BiasDetectionResults />
      </BrowserRouter>
    );

    expect(screen.getByText(/loading/i) || screen.getByRole('status')).toBeInTheDocument();
  });
});
