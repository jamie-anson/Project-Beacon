/**
 * Unit tests for RegionRow component
 */

import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import RegionRow from '../RegionRow';

const renderWithRouter = (component) => {
  return render(<BrowserRouter>{component}</BrowserRouter>);
};

describe('RegionRow', () => {
  const defaultProps = {
    region: 'US',
    regionExecs: [
      { id: 1, status: 'completed', started_at: new Date().toISOString() },
      { id: 2, status: 'running', started_at: new Date().toISOString() }
    ],
    isExpanded: false,
    onToggle: jest.fn(),
    jobCompleted: false,
    jobFailed: false,
    jobStuckTimeout: false,
    loadingActive: false,
    uniqueModels: ['llama3.2-1b'],
    hasQuestions: true,
    jobId: 'test-job-123',
    failureInfo: null,
    jobAge: 5,
    statusStr: 'processing'
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should render region name', () => {
    renderWithRouter(<RegionRow {...defaultProps} />);

    expect(screen.getByText('US')).toBeInTheDocument();
  });

  it('should call onToggle when clicked', () => {
    renderWithRouter(<RegionRow {...defaultProps} />);

    const row = screen.getByText('US').closest('.grid');
    fireEvent.click(row);

    expect(defaultProps.onToggle).toHaveBeenCalledTimes(1);
  });

  it('should display progress indicator', () => {
    renderWithRouter(<RegionRow {...defaultProps} />);

    expect(screen.getByText('1/2')).toBeInTheDocument(); // 1 completed out of 2 total
  });

  it('should show progress bar with correct width', () => {
    const { container } = renderWithRouter(<RegionRow {...defaultProps} />);

    const progressBar = container.querySelector('.bg-green-500');
    expect(progressBar).toHaveStyle({ width: '50%' }); // 1/2 = 50%
  });

  it('should display status badge', () => {
    renderWithRouter(<RegionRow {...defaultProps} />);

    // Should show running status since there's a running execution
    const statusBadge = screen.getByText('running');
    expect(statusBadge).toBeInTheDocument();
  });

  it('should show chevron icon when has questions and executions', () => {
    const { container } = renderWithRouter(<RegionRow {...defaultProps} />);

    const chevron = container.querySelector('svg');
    expect(chevron).toBeInTheDocument();
  });

  it('should rotate chevron when expanded', () => {
    const { container } = renderWithRouter(<RegionRow {...defaultProps} isExpanded={true} />);

    const chevron = container.querySelector('svg');
    expect(chevron).toHaveClass('rotate-180');
  });

  it('should not rotate chevron when collapsed', () => {
    const { container } = renderWithRouter(<RegionRow {...defaultProps} isExpanded={false} />);

    const chevron = container.querySelector('svg');
    expect(chevron).not.toHaveClass('rotate-180');
  });

  it('should display model count when multiple models', () => {
    renderWithRouter(<RegionRow {...defaultProps} />);

    expect(screen.getByText('1 models')).toBeInTheDocument();
  });

  it('should display View link when executions exist', () => {
    renderWithRouter(<RegionRow {...defaultProps} />);

    const viewLink = screen.getByText('View');
    expect(viewLink).toBeInTheDocument();
    expect(viewLink).toHaveAttribute('href', '/executions?job=test-job-123&region=us-east');
  });

  it('should display — when no executions', () => {
    renderWithRouter(<RegionRow {...defaultProps} regionExecs={[]} />);

    const dashes = screen.getAllByText('—');
    expect(dashes.length).toBeGreaterThan(0);
  });

  it('should show completed status when job is completed', () => {
    renderWithRouter(<RegionRow {...defaultProps} jobCompleted={true} />);

    expect(screen.getByText('completed')).toBeInTheDocument();
  });

  it('should show failed status when job failed', () => {
    renderWithRouter(<RegionRow {...defaultProps} jobFailed={true} />);

    expect(screen.getByText('failed')).toBeInTheDocument();
  });

  it('should display job failure message', () => {
    const failureInfo = {
      title: 'Job Failed',
      message: 'Job failed with status: failed'
    };

    renderWithRouter(<RegionRow {...defaultProps} jobFailed={true} failureInfo={failureInfo} />);

    expect(screen.getByText(/Job failed: processing/)).toBeInTheDocument();
  });

  it('should display timeout message when stuck', () => {
    renderWithRouter(<RegionRow {...defaultProps} jobStuckTimeout={true} jobAge={20} />);

    expect(screen.getByText('Job timeout')).toBeInTheDocument();
    expect(screen.getByText('20min stuck')).toBeInTheDocument();
  });

  it('should display execution-level failure message', () => {
    const execsWithFailure = [
      {
        id: 1,
        status: 'failed',
        output: {
          failure: {
            message: 'Connection timeout',
            code: 'TIMEOUT',
            stage: 'network'
          }
        }
      }
    ];

    renderWithRouter(<RegionRow {...defaultProps} regionExecs={execsWithFailure} />);

    expect(screen.getByText(/Connection timeout/)).toBeInTheDocument();
  });

  it('should truncate long failure messages', () => {
    const longMessage = 'A'.repeat(100);
    const execsWithFailure = [
      {
        id: 1,
        status: 'failed',
        error: longMessage
      }
    ];

    renderWithRouter(<RegionRow {...defaultProps} regionExecs={execsWithFailure} />);

    const truncatedText = screen.getByText(/A{60}…/);
    expect(truncatedText).toBeInTheDocument();
  });

  it('should show multi-model progress', () => {
    const multiModelExecs = [
      { id: 1, status: 'completed' },
      { id: 2, status: 'completed' },
      { id: 3, status: 'running' }
    ];

    renderWithRouter(<RegionRow {...defaultProps} regionExecs={multiModelExecs} />);

    expect(screen.getByText('2/3 models')).toBeInTheDocument();
  });

  it('should display time ago for started executions', () => {
    const recentTime = new Date(Date.now() - 30000).toISOString(); // 30 seconds ago
    const execsWithTime = [
      { id: 1, status: 'running', started_at: recentTime }
    ];

    renderWithRouter(<RegionRow {...defaultProps} regionExecs={execsWithTime} />);

    expect(screen.getByText(/ago/)).toBeInTheDocument();
  });

  it('should show refreshing status when loading', () => {
    renderWithRouter(<RegionRow {...defaultProps} loadingActive={true} />);

    expect(screen.getByText('refreshing')).toBeInTheDocument();
  });

  it('should handle pending status with no executions', () => {
    renderWithRouter(<RegionRow {...defaultProps} regionExecs={[]} />);

    expect(screen.getByText('pending')).toBeInTheDocument();
  });

  it('should prevent click propagation on View link', () => {
    const { container } = renderWithRouter(<RegionRow {...defaultProps} />);

    const viewLink = screen.getByText('View');
    const actionsCell = viewLink.closest('.px-3');
    
    fireEvent.click(actionsCell);

    // onToggle should not be called when clicking the actions cell
    expect(defaultProps.onToggle).not.toHaveBeenCalled();
  });

  it('should apply hover effect', () => {
    const { container } = renderWithRouter(<RegionRow {...defaultProps} />);

    const row = container.querySelector('.hover\\:bg-gray-700');
    expect(row).toBeInTheDocument();
  });

  it('should show cursor pointer', () => {
    const { container } = renderWithRouter(<RegionRow {...defaultProps} />);

    const row = container.querySelector('.cursor-pointer');
    expect(row).toBeInTheDocument();
  });
});
