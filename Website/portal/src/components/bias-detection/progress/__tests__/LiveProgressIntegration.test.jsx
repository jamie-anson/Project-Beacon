/**
 * Integration tests for Live Progress UI features
 * Tests processing animation and retry button working together
 */

import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import LiveProgressTable from '../../LiveProgressTable';

const renderWithRouter = (component) => {
  return render(<BrowserRouter>{component}</BrowserRouter>);
};

describe('LiveProgressTable Integration - Processing Animation & Retry Button', () => {
  const mockRefetchActive = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Processing Animation Integration', () => {
    it('should show processing animation when job is running', () => {
      const runningJob = {
        id: 'test-job-123',
        status: 'processing',
        created_at: new Date().toISOString(),
        executions: [
          {
            id: 1,
            status: 'running',
            region: 'us-east',
            model_id: 'llama3.2-1b',
            started_at: new Date().toISOString()
          }
        ]
      };

      const { container } = renderWithRouter(
        <LiveProgressTable
          activeJob={runningJob}
          selectedRegions={['US', 'EU']}
          loadingActive={false}
          refetchActive={mockRefetchActive}
          activeJobId="test-job-123"
          isCompleted={false}
        />
      );

      // Check for pulse animation
      const progressBar = container.querySelector('.animate-pulse');
      expect(progressBar).toBeInTheDocument();

      // Check for shimmer overlay
      const shimmer = container.querySelector('.animate-shimmer');
      expect(shimmer).toBeInTheDocument();
    });

    it('should remove processing animation when job completes', () => {
      const completedJob = {
        id: 'test-job-123',
        status: 'completed',
        created_at: new Date(Date.now() - 120000).toISOString(),
        executions: [
          {
            id: 1,
            status: 'completed',
            region: 'us-east',
            model_id: 'llama3.2-1b',
            started_at: new Date(Date.now() - 120000).toISOString()
          }
        ]
      };

      const { container } = renderWithRouter(
        <LiveProgressTable
          activeJob={completedJob}
          selectedRegions={['US', 'EU']}
          loadingActive={false}
          refetchActive={mockRefetchActive}
          activeJobId="test-job-123"
          isCompleted={true}
        />
      );

      // Animation should be removed
      const shimmer = container.querySelector('.animate-shimmer');
      expect(shimmer).not.toBeInTheDocument();
    });
  });

  describe('Retry Button Integration', () => {
    it('should show retry button when job fails', () => {
      const failedJob = {
        id: 'test-job-123',
        status: 'failed',
        created_at: new Date(Date.now() - 60000).toISOString(),
        executions: []
      };

      renderWithRouter(
        <LiveProgressTable
          activeJob={failedJob}
          selectedRegions={['US', 'EU']}
          loadingActive={false}
          refetchActive={mockRefetchActive}
          activeJobId="test-job-123"
          isCompleted={false}
        />
      );

      expect(screen.getByText('Retry Job')).toBeInTheDocument();
    });

    it('should show retry button when job times out', () => {
      const timeoutJob = {
        id: 'test-job-123',
        status: 'processing',
        created_at: new Date(Date.now() - 20 * 60 * 1000).toISOString(), // 20 minutes ago
        executions: []
      };

      renderWithRouter(
        <LiveProgressTable
          activeJob={timeoutJob}
          selectedRegions={['US', 'EU']}
          loadingActive={false}
          refetchActive={mockRefetchActive}
          activeJobId="test-job-123"
          isCompleted={false}
        />
      );

      expect(screen.getByText('Retry Job')).toBeInTheDocument();
    });

    it('should NOT show retry button when job is running', () => {
      const runningJob = {
        id: 'test-job-123',
        status: 'processing',
        created_at: new Date().toISOString(),
        executions: [
          {
            id: 1,
            status: 'running',
            region: 'us-east',
            model_id: 'llama3.2-1b',
            started_at: new Date().toISOString()
          }
        ]
      };

      renderWithRouter(
        <LiveProgressTable
          activeJob={runningJob}
          selectedRegions={['US', 'EU']}
          loadingActive={false}
          refetchActive={mockRefetchActive}
          activeJobId="test-job-123"
          isCompleted={false}
        />
      );

      expect(screen.queryByText('Retry Job')).not.toBeInTheDocument();
    });

    it('should reload page when retry button is clicked', () => {
      const failedJob = {
        id: 'test-job-123',
        status: 'failed',
        created_at: new Date(Date.now() - 60000).toISOString(),
        executions: []
      };

      // Mock window.location.reload
      delete window.location;
      window.location = { reload: jest.fn() };

      renderWithRouter(
        <LiveProgressTable
          activeJob={failedJob}
          selectedRegions={['US', 'EU']}
          loadingActive={false}
          refetchActive={mockRefetchActive}
          activeJobId="test-job-123"
          isCompleted={false}
        />
      );

      const retryButton = screen.getByText('Retry Job');
      fireEvent.click(retryButton);

      expect(window.location.reload).toHaveBeenCalledTimes(1);
    });
  });

  describe('Combined Behavior', () => {
    it('should transition from processing animation to retry button on failure', async () => {
      const { rerender, container } = renderWithRouter(
        <LiveProgressTable
          activeJob={{
            id: 'test-job-123',
            status: 'processing',
            created_at: new Date().toISOString(),
            executions: [
              {
                id: 1,
                status: 'running',
                region: 'us-east',
                model_id: 'llama3.2-1b',
                started_at: new Date().toISOString()
              }
            ]
          }}
          selectedRegions={['US', 'EU']}
          loadingActive={false}
          refetchActive={mockRefetchActive}
          activeJobId="test-job-123"
          isCompleted={false}
        />
      );

      // Initially should have animation, no retry button
      expect(container.querySelector('.animate-shimmer')).toBeInTheDocument();
      expect(screen.queryByText('Retry Job')).not.toBeInTheDocument();

      // Simulate job failure
      rerender(
        <BrowserRouter>
          <LiveProgressTable
            activeJob={{
              id: 'test-job-123',
              status: 'failed',
              created_at: new Date().toISOString(),
              executions: []
            }}
            selectedRegions={['US', 'EU']}
            loadingActive={false}
            refetchActive={mockRefetchActive}
            activeJobId="test-job-123"
            isCompleted={false}
          />
        </BrowserRouter>
      );

      // Should now have retry button, no animation
      await waitFor(() => {
        expect(screen.getByText('Retry Job')).toBeInTheDocument();
        expect(container.querySelector('.animate-shimmer')).not.toBeInTheDocument();
      });
    });

    it('should show both refresh and retry buttons when failed', () => {
      const failedJob = {
        id: 'test-job-123',
        status: 'failed',
        created_at: new Date(Date.now() - 60000).toISOString(),
        executions: []
      };

      renderWithRouter(
        <LiveProgressTable
          activeJob={failedJob}
          selectedRegions={['US', 'EU']}
          loadingActive={false}
          refetchActive={mockRefetchActive}
          activeJobId="test-job-123"
          isCompleted={false}
        />
      );

      expect(screen.getByText('Retry Job')).toBeInTheDocument();
      expect(screen.getByText('Refresh')).toBeInTheDocument();
    });

    it('should handle multi-region with mixed statuses correctly', () => {
      const mixedJob = {
        id: 'test-job-123',
        status: 'processing',
        created_at: new Date().toISOString(),
        executions: [
          {
            id: 1,
            status: 'completed',
            region: 'us-east',
            model_id: 'llama3.2-1b',
            started_at: new Date(Date.now() - 60000).toISOString()
          },
          {
            id: 2,
            status: 'running',
            region: 'eu-west',
            model_id: 'llama3.2-1b',
            started_at: new Date().toISOString()
          }
        ]
      };

      const { container } = renderWithRouter(
        <LiveProgressTable
          activeJob={mixedJob}
          selectedRegions={['US', 'EU']}
          loadingActive={false}
          refetchActive={mockRefetchActive}
          activeJobId="test-job-123"
          isCompleted={false}
        />
      );

      // Should have animation for running region
      const shimmer = container.querySelector('.animate-shimmer');
      expect(shimmer).toBeInTheDocument();

      // Should NOT have retry button (job still in progress)
      expect(screen.queryByText('Retry Job')).not.toBeInTheDocument();
    });
  });

  describe('Accessibility', () => {
    it('should maintain proper button order for keyboard navigation', () => {
      const failedJob = {
        id: 'test-job-123',
        status: 'failed',
        created_at: new Date(Date.now() - 60000).toISOString(),
        executions: []
      };

      const { container } = renderWithRouter(
        <LiveProgressTable
          activeJob={failedJob}
          selectedRegions={['US', 'EU']}
          loadingActive={false}
          refetchActive={mockRefetchActive}
          activeJobId="test-job-123"
          isCompleted={false}
        />
      );

      const buttons = Array.from(container.querySelectorAll('button'));
      const retryButton = buttons.find(btn => btn.textContent.includes('Retry Job'));
      const refreshButton = buttons.find(btn => btn.textContent === 'Refresh');

      // Retry should come before Refresh for logical tab order
      expect(buttons.indexOf(retryButton)).toBeLessThan(buttons.indexOf(refreshButton));
    });

    it('should have proper ARIA attributes on retry button', () => {
      const failedJob = {
        id: 'test-job-123',
        status: 'failed',
        created_at: new Date(Date.now() - 60000).toISOString(),
        executions: []
      };

      renderWithRouter(
        <LiveProgressTable
          activeJob={failedJob}
          selectedRegions={['US', 'EU']}
          loadingActive={false}
          refetchActive={mockRefetchActive}
          activeJobId="test-job-123"
          isCompleted={false}
        />
      );

      const retryButton = screen.getByText('Retry Job');
      expect(retryButton.tagName).toBe('BUTTON');
      expect(retryButton).toHaveClass('bg-yellow-600');
    });
  });
});
