import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { BrowserRouter, MemoryRouter } from 'react-router-dom';
import '@testing-library/jest-dom';
import Executions from '../Executions.jsx';

// Mock the API
jest.mock('../../lib/api/runner/executions.js', () => ({
  getExecutions: jest.fn()
}));

jest.mock('../../hooks/usePageTitle.js', () => ({
  usePageTitle: jest.fn()
}));

const { getExecutions } = require('../../lib/api/runner/executions.js');

const mockExecutions = [
  {
    id: 801,
    job_id: 'bias-detection-1759077571504',
    status: 'completed',
    region: 'us-east',
    model_id: 'qwen2.5-1.5b',
    created_at: '2025-09-28T16:39:53Z'
  },
  {
    id: 802,
    job_id: 'bias-detection-1759077571504',
    status: 'completed',
    region: 'eu-west',
    model_id: 'llama3.2-1b',
    created_at: '2025-09-28T16:40:13Z'
  },
  {
    id: 803,
    job_id: 'bias-detection-1759077571504',
    status: 'completed',
    region: 'asia-pacific',
    model_id: 'mistral-7b',
    created_at: '2025-09-28T16:40:08Z'
  },
  {
    id: 804,
    job_id: 'other-job-123',
    status: 'failed',
    region: 'us-east',
    model_id: 'qwen2.5-1.5b',
    created_at: '2025-09-28T15:30:00Z'
  }
];

const renderWithRouter = (component, initialEntries = ['/']) => {
  return render(
    <MemoryRouter initialEntries={initialEntries}>
      {component}
    </MemoryRouter>
  );
};

describe('Executions Page - Region Filtering Tests', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    getExecutions.mockResolvedValue(mockExecutions);
  });

  describe('Region Filter Mapping', () => {
    test('filters executions by us-east when region=us-east in URL', async () => {
      renderWithRouter(
        <Executions />,
        ['/executions?job=bias-detection-1759077571504&region=us-east']
      );

      await waitFor(() => {
        expect(screen.getByText('1 of 4 shown')).toBeInTheDocument();
      });

      // Should show only the us-east execution
      expect(screen.getByText('801')).toBeInTheDocument();
      expect(screen.queryByText('802')).not.toBeInTheDocument();
      expect(screen.queryByText('803')).not.toBeInTheDocument();
    });

    test('filters executions by eu-west when region=eu-west in URL', async () => {
      renderWithRouter(
        <Executions />,
        ['/executions?job=bias-detection-1759077571504&region=eu-west']
      );

      await waitFor(() => {
        expect(screen.getByText('1 of 4 shown')).toBeInTheDocument();
      });

      // Should show only the eu-west execution
      expect(screen.getByText('802')).toBeInTheDocument();
      expect(screen.queryByText('801')).not.toBeInTheDocument();
      expect(screen.queryByText('803')).not.toBeInTheDocument();
    });

    test('filters executions by asia-pacific when region=asia-pacific in URL', async () => {
      renderWithRouter(
        <Executions />,
        ['/executions?job=bias-detection-1759077571504&region=asia-pacific']
      );

      await waitFor(() => {
        expect(screen.getByText('1 of 4 shown')).toBeInTheDocument();
      });

      // Should show only the asia-pacific execution
      expect(screen.getByText('803')).toBeInTheDocument();
      expect(screen.queryByText('801')).not.toBeInTheDocument();
      expect(screen.queryByText('802')).not.toBeInTheDocument();
    });

    test('shows no results when using incorrect display region names', async () => {
      // Test the old broken behavior to ensure it's fixed
      renderWithRouter(
        <Executions />,
        ['/executions?job=bias-detection-1759077571504&region=US']
      );

      await waitFor(() => {
        expect(screen.getByText('No executions match current filters.')).toBeInTheDocument();
      });

      // Should show filter info
      expect(screen.getByText('US')).toBeInTheDocument(); // In filter display
      expect(screen.getByText('Clear filters')).toBeInTheDocument();
    });

    test('shows no results for EU and ASIA display names', async () => {
      // Test EU
      renderWithRouter(
        <Executions />,
        ['/executions?job=bias-detection-1759077571504&region=EU']
      );

      await waitFor(() => {
        expect(screen.getByText('No executions match current filters.')).toBeInTheDocument();
      });

      // Test ASIA
      renderWithRouter(
        <Executions />,
        ['/executions?job=bias-detection-1759077571504&region=ASIA']
      );

      await waitFor(() => {
        expect(screen.getByText('No executions match current filters.')).toBeInTheDocument();
      });
    });
  });

  describe('Job Filtering', () => {
    test('filters executions by job ID correctly', async () => {
      renderWithRouter(
        <Executions />,
        ['/executions?job=bias-detection-1759077571504']
      );

      await waitFor(() => {
        expect(screen.getByText('3 of 4 shown')).toBeInTheDocument();
      });

      // Should show executions 801, 802, 803 but not 804
      expect(screen.getByText('801')).toBeInTheDocument();
      expect(screen.getByText('802')).toBeInTheDocument();
      expect(screen.getByText('803')).toBeInTheDocument();
      expect(screen.queryByText('804')).not.toBeInTheDocument();
    });

    test('combines job and region filters correctly', async () => {
      renderWithRouter(
        <Executions />,
        ['/executions?job=bias-detection-1759077571504&region=us-east']
      );

      await waitFor(() => {
        expect(screen.getByText('1 of 4 shown')).toBeInTheDocument();
      });

      // Should show only execution 801 (matches both job and region)
      expect(screen.getByText('801')).toBeInTheDocument();
      expect(screen.queryByText('802')).not.toBeInTheDocument();
      expect(screen.queryByText('803')).not.toBeInTheDocument();
      expect(screen.queryByText('804')).not.toBeInTheDocument();
    });
  });

  describe('Filter Display', () => {
    test('shows current filters in UI', async () => {
      renderWithRouter(
        <Executions />,
        ['/executions?job=test-job&region=us-east']
      );

      await waitFor(() => {
        // Should show filter chips
        expect(screen.getByText('job:')).toBeInTheDocument();
        expect(screen.getByText('test-job')).toBeInTheDocument();
        expect(screen.getByText('region:')).toBeInTheDocument();
        expect(screen.getByText('US-EAST')).toBeInTheDocument(); // Uppercase in display
        expect(screen.getByText('Clear filters')).toBeInTheDocument();
      });
    });

    test('shows no filter chips when no filters applied', async () => {
      renderWithRouter(<Executions />, ['/executions']);

      await waitFor(() => {
        expect(screen.getByText('4 shown')).toBeInTheDocument();
      });

      // Should not show filter chips
      expect(screen.queryByText('job:')).not.toBeInTheDocument();
      expect(screen.queryByText('region:')).not.toBeInTheDocument();
      expect(screen.queryByText('Clear filters')).not.toBeInTheDocument();
    });
  });

  describe('Case Sensitivity', () => {
    test('handles case-insensitive region matching', async () => {
      // The current implementation converts to uppercase for comparison
      renderWithRouter(
        <Executions />,
        ['/executions?region=US-EAST'] // Uppercase
      );

      await waitFor(() => {
        expect(screen.getByText('2 of 4 shown')).toBeInTheDocument();
      });

      // Should match both us-east executions (801 and 804)
      expect(screen.getByText('801')).toBeInTheDocument();
      expect(screen.getByText('804')).toBeInTheDocument();
    });

    test('handles mixed case region names', async () => {
      renderWithRouter(
        <Executions />,
        ['/executions?region=Us-East'] // Mixed case
      );

      await waitFor(() => {
        expect(screen.getByText('2 of 4 shown')).toBeInTheDocument();
      });

      // Should still match us-east executions
      expect(screen.getByText('801')).toBeInTheDocument();
      expect(screen.getByText('804')).toBeInTheDocument();
    });
  });

  describe('Edge Cases', () => {
    test('handles empty execution list', async () => {
      getExecutions.mockResolvedValue([]);

      renderWithRouter(<Executions />);

      await waitFor(() => {
        expect(screen.getByText('No executions yet.')).toBeInTheDocument();
      });
    });

    test('handles API errors gracefully', async () => {
      getExecutions.mockRejectedValue(new Error('API Error'));

      renderWithRouter(<Executions />);

      await waitFor(() => {
        expect(screen.getByText('Backend unavailable - Executions service offline')).toBeInTheDocument();
      });
    });

    test('handles malformed execution data', async () => {
      getExecutions.mockResolvedValue([
        { id: 'bad-1' }, // Missing required fields
        { job_id: 'test', region: null }, // Null region
        { id: 'good-1', job_id: 'test', region: 'us-east', status: 'completed' }
      ]);

      renderWithRouter(<Executions />);

      await waitFor(() => {
        expect(screen.getByText('3 shown')).toBeInTheDocument();
      });

      // Should handle malformed data gracefully
      expect(screen.getByText('good-1')).toBeInTheDocument();
    });
  });

  describe('URL Parameter Parsing', () => {
    test('handles URL-encoded parameters correctly', async () => {
      renderWithRouter(
        <Executions />,
        ['/executions?job=bias-detection-1759077571504&region=asia-pacific']
      );

      await waitFor(() => {
        expect(screen.getByText('1 of 4 shown')).toBeInTheDocument();
      });

      expect(screen.getByText('803')).toBeInTheDocument();
    });

    test('handles special characters in job IDs', async () => {
      const specialJobId = 'job-with-special-chars-123!@#';
      const executionsWithSpecialJob = [
        ...mockExecutions,
        {
          id: 999,
          job_id: specialJobId,
          status: 'completed',
          region: 'us-east',
          created_at: '2025-09-28T17:00:00Z'
        }
      ];

      getExecutions.mockResolvedValue(executionsWithSpecialJob);

      renderWithRouter(
        <Executions />,
        [`/executions?job=${encodeURIComponent(specialJobId)}`]
      );

      await waitFor(() => {
        expect(screen.getByText('1 of 5 shown')).toBeInTheDocument();
      });

      expect(screen.getByText('999')).toBeInTheDocument();
    });
  });
});
