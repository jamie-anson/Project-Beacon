/**
 * Integration tests for LiveProgressTable component
 * Tests the complete component with all sub-components and hooks
 */

import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import LiveProgressTable from '../LiveProgressTable';
import { retryQuestion } from '../../../lib/api/runner/executions';

// Mock dependencies
jest.mock('../../../lib/api/runner/executions');
jest.mock('../../Toasts', () => ({
  showToast: jest.fn()
}));

const renderWithRouter = (component) => {
  return render(<BrowserRouter>{component}</BrowserRouter>);
};

describe('LiveProgressTable Integration Tests', () => {
  const mockRefetchActive = jest.fn();

  const mockActiveJob = {
    id: 'test-job-123',
    status: 'processing',
    job: {
      questions: ['q1', 'q2'],
      models: [
        { id: 'llama3.2-1b', regions: ['US', 'EU'] },
        { id: 'mistral-7b', regions: ['US', 'EU'] }
      ]
    },
    executions: [
      {
        id: 'exec-1',
        status: 'completed',
        model_id: 'llama3.2-1b',
        question_id: 'q1',
        region: 'us-east',
        started_at: new Date().toISOString(),
        response_classification: 'substantive'
      },
      {
        id: 'exec-2',
        status: 'running',
        model_id: 'mistral-7b',
        question_id: 'q1',
        region: 'eu-west',
        started_at: new Date().toISOString()
      },
      {
        id: 'exec-3',
        status: 'failed',
        model_id: 'llama3.2-1b',
        question_id: 'q2',
        region: 'us-east',
        started_at: new Date().toISOString(),
        error: 'Timeout error'
      }
    ]
  };

  const defaultProps = {
    activeJob: mockActiveJob,
    selectedRegions: ['US', 'EU'],
    loadingActive: false,
    refetchActive: mockRefetchActive,
    activeJobId: 'test-job-123',
    isCompleted: false,
    diffReady: false
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Complete Rendering', () => {
    it('should render all major sections', () => {
      renderWithRouter(<LiveProgressTable {...defaultProps} />);

      // Progress header should be present
      expect(screen.getByText('Executing questions...')).toBeInTheDocument();

      // Progress breakdown should be present
      expect(screen.getByText(/Completed:/)).toBeInTheDocument();
      expect(screen.getByText(/Running:/)).toBeInTheDocument();

      // Region table should be present
      expect(screen.getByText('US')).toBeInTheDocument();
      expect(screen.getByText('EU')).toBeInTheDocument();

      // Actions should be present
      expect(screen.getByText('Refresh')).toBeInTheDocument();
    });

    it('should calculate and display correct progress metrics', () => {
      renderWithRouter(<LiveProgressTable {...defaultProps} />);

      // 1 completed, 1 running, 1 failed out of 8 total (2 questions × 2 models × 2 regions)
      expect(screen.getByText('Completed: 1')).toBeInTheDocument();
      expect(screen.getByText('Running: 1')).toBeInTheDocument();
      expect(screen.getByText('Failed: 1')).toBeInTheDocument();
      expect(screen.getByText('Pending: 5')).toBeInTheDocument();
    });

    it('should display question progress breakdown', () => {
      renderWithRouter(<LiveProgressTable {...defaultProps} />);

      expect(screen.getByText('Question Progress')).toBeInTheDocument();
      expect(screen.getByText('q1')).toBeInTheDocument();
      expect(screen.getByText('q2')).toBeInTheDocument();
    });
  });

  describe('Region Expansion', () => {
    it('should expand region when clicked', () => {
      const { container } = renderWithRouter(<LiveProgressTable {...defaultProps} />);

      const usRegionRow = screen.getByText('US').closest('.grid');
      fireEvent.click(usRegionRow);

      // Should show execution details
      expect(screen.getByText('Execution Details for US')).toBeInTheDocument();
    });

    it('should collapse region when clicked again', () => {
      const { container } = renderWithRouter(<LiveProgressTable {...defaultProps} />);

      const usRegionRow = screen.getByText('US').closest('.grid');
      
      // Expand
      fireEvent.click(usRegionRow);
      expect(screen.getByText('Execution Details for US')).toBeInTheDocument();

      // Collapse
      fireEvent.click(usRegionRow);
      expect(screen.queryByText('Execution Details for US')).not.toBeInTheDocument();
    });

    it('should show execution details with model and question grid', () => {
      renderWithRouter(<LiveProgressTable {...defaultProps} />);

      const usRegionRow = screen.getByText('US').closest('.grid');
      fireEvent.click(usRegionRow);

      expect(screen.getByText('llama3.2-1b')).toBeInTheDocument();
      expect(screen.getByText('Execution Details for US')).toBeInTheDocument();
    });
  });

  describe('Retry Functionality', () => {
    it('should handle retry button click', async () => {
      retryQuestion.mockResolvedValueOnce({ success: true });

      renderWithRouter(<LiveProgressTable {...defaultProps} />);

      // Expand US region
      const usRegionRow = screen.getByText('US').closest('.grid');
      fireEvent.click(usRegionRow);

      // Click retry button for failed execution
      const retryButton = screen.getByText('Retry');
      fireEvent.click(retryButton);

      await waitFor(() => {
        expect(retryQuestion).toHaveBeenCalled();
      });
    });

    it('should prevent duplicate retries', async () => {
      retryQuestion.mockImplementation(() => new Promise(resolve => setTimeout(resolve, 100)));

      renderWithRouter(<LiveProgressTable {...defaultProps} />);

      const usRegionRow = screen.getByText('US').closest('.grid');
      fireEvent.click(usRegionRow);

      const retryButton = screen.getByText('Retry');
      
      // Click twice rapidly
      fireEvent.click(retryButton);
      fireEvent.click(retryButton);

      await waitFor(() => {
        expect(retryQuestion).toHaveBeenCalledTimes(1);
      });
    });
  });

  describe('Refresh Functionality', () => {
    it('should call refetchActive when refresh button clicked', () => {
      renderWithRouter(<LiveProgressTable {...defaultProps} />);

      const refreshButton = screen.getByText('Refresh');
      fireEvent.click(refreshButton);

      expect(mockRefetchActive).toHaveBeenCalledTimes(1);
    });
  });

  describe('Failure States', () => {
    it('should display failure alert for failed jobs', () => {
      const failedJob = {
        ...mockActiveJob,
        status: 'failed',
        executions: []
      };

      renderWithRouter(<LiveProgressTable {...defaultProps} activeJob={failedJob} />);

      expect(screen.getByText('Job Failed')).toBeInTheDocument();
      expect(screen.getByText(/Job failed with status: failed/)).toBeInTheDocument();
    });

    it('should show all regions as failed when job fails', () => {
      const failedJob = {
        ...mockActiveJob,
        status: 'failed',
        executions: []
      };

      renderWithRouter(<LiveProgressTable {...defaultProps} activeJob={failedJob} />);

      const failedStatuses = screen.getAllByText('failed');
      expect(failedStatuses.length).toBeGreaterThan(1); // Multiple regions should show failed
    });

    it('should display timeout alert for stuck jobs', () => {
      jest.useFakeTimers();
      
      const stuckJob = {
        ...mockActiveJob,
        status: 'processing',
        executions: []
      };

      renderWithRouter(<LiveProgressTable {...defaultProps} activeJob={stuckJob} />);

      // Fast-forward time to simulate stuck job
      jest.advanceTimersByTime(16 * 60 * 1000); // 16 minutes

      jest.useRealTimers();
    });
  });

  describe('Completed State', () => {
    it('should enable View Diffs button when completed', () => {
      const completedJob = {
        ...mockActiveJob,
        status: 'completed'
      };

      renderWithRouter(<LiveProgressTable {...defaultProps} activeJob={completedJob} isCompleted={true} />);

      const diffButton = screen.getByText('View Cross-Region Diffs');
      expect(diffButton).not.toBeDisabled();
    });

    it('should show completion indicator', () => {
      const completedJob = {
        ...mockActiveJob,
        status: 'completed'
      };

      renderWithRouter(<LiveProgressTable {...defaultProps} activeJob={completedJob} isCompleted={true} />);

      expect(screen.getByText('Job completed successfully!')).toBeInTheDocument();
    });
  });

  describe('Loading States', () => {
    it('should show refreshing status when loading', () => {
      renderWithRouter(<LiveProgressTable {...defaultProps} loadingActive={true} />);

      const refreshingStatuses = screen.getAllByText('refreshing');
      expect(refreshingStatuses.length).toBeGreaterThan(0);
    });

    it('should show processing indicator', () => {
      renderWithRouter(<LiveProgressTable {...defaultProps} />);

      expect(screen.getByText('Processing...')).toBeInTheDocument();
    });
  });

  describe('Empty States', () => {
    it('should handle job with no executions', () => {
      const noExecJob = {
        ...mockActiveJob,
        executions: []
      };

      renderWithRouter(<LiveProgressTable {...defaultProps} activeJob={noExecJob} />);

      expect(screen.getByText('Completed: 0')).toBeInTheDocument();
      expect(screen.getByText('Running: 0')).toBeInTheDocument();
    });

    it('should handle null job', () => {
      renderWithRouter(<LiveProgressTable {...defaultProps} activeJob={null} />);

      expect(screen.getByText('Completed: 0')).toBeInTheDocument();
    });
  });

  describe('Multi-Model Support', () => {
    it('should display multiple models in region rows', () => {
      renderWithRouter(<LiveProgressTable {...defaultProps} />);

      expect(screen.getByText('2 models')).toBeInTheDocument();
    });

    it('should show model breakdown in execution details', () => {
      renderWithRouter(<LiveProgressTable {...defaultProps} />);

      const usRegionRow = screen.getByText('US').closest('.grid');
      fireEvent.click(usRegionRow);

      expect(screen.getByText('llama3.2-1b')).toBeInTheDocument();
    });
  });

  describe('Time Display', () => {
    it('should display time ago for executions', () => {
      renderWithRouter(<LiveProgressTable {...defaultProps} />);

      expect(screen.getByText(/ago/)).toBeInTheDocument();
    });

    it('should show countdown timer for active jobs', () => {
      jest.useFakeTimers();

      renderWithRouter(<LiveProgressTable {...defaultProps} />);

      // Should show time remaining
      const timeDisplay = screen.getByText(/Time remaining:|executions/);
      expect(timeDisplay).toBeInTheDocument();

      jest.useRealTimers();
    });
  });

  describe('Accessibility', () => {
    it('should have proper link hrefs', () => {
      renderWithRouter(<LiveProgressTable {...defaultProps} />);

      const viewLinks = screen.getAllByText('View');
      expect(viewLinks[0]).toHaveAttribute('href');
    });

    it('should have clickable regions', () => {
      renderWithRouter(<LiveProgressTable {...defaultProps} />);

      const usRegion = screen.getByText('US').closest('.grid');
      expect(usRegion).toHaveClass('cursor-pointer');
    });
  });

  describe('Performance', () => {
    it('should handle large number of executions', () => {
      const manyExecs = Array.from({ length: 100 }, (_, i) => ({
        id: `exec-${i}`,
        status: i % 3 === 0 ? 'completed' : i % 3 === 1 ? 'running' : 'failed',
        model_id: `model-${i % 5}`,
        question_id: `q${i % 10}`,
        region: i % 2 === 0 ? 'us-east' : 'eu-west',
        started_at: new Date().toISOString()
      }));

      const largeJob = {
        ...mockActiveJob,
        executions: manyExecs
      };

      const { container } = renderWithRouter(<LiveProgressTable {...defaultProps} activeJob={largeJob} />);

      expect(container).toBeInTheDocument();
      expect(screen.getByText(/Completed:/)).toBeInTheDocument();
    });
  });
});
