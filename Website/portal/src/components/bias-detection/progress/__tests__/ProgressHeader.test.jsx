/**
 * Unit tests for ProgressHeader component
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import ProgressHeader from '../ProgressHeader';

describe('ProgressHeader', () => {
  const defaultProps = {
    stage: 'running',
    timeRemaining: '8:30',
    completed: 5,
    running: 3,
    failed: 1,
    pending: 1,
    total: 10,
    percentage: 50,
    showShimmer: true,
    overallCompleted: false,
    overallFailed: false,
    hasQuestions: true,
    specQuestions: ['q1', 'q2'],
    displayQuestions: ['q1', 'q2'],
    specModels: [{ id: 'llama3.2-1b' }],
    uniqueModels: ['llama3.2-1b'],
    selectedRegions: ['US', 'EU']
  };

  it('should render running stage indicator', () => {
    render(<ProgressHeader {...defaultProps} />);

    expect(screen.getByText('Executing questions...')).toBeInTheDocument();
  });

  it('should render creating stage', () => {
    render(<ProgressHeader {...defaultProps} stage="creating" />);

    expect(screen.getByText('Creating job...')).toBeInTheDocument();
  });

  it('should render queued stage', () => {
    render(<ProgressHeader {...defaultProps} stage="queued" />);

    expect(screen.getByText('Job queued, waiting for worker...')).toBeInTheDocument();
  });

  it('should render spawning stage', () => {
    render(<ProgressHeader {...defaultProps} stage="spawning" />);

    expect(screen.getByText('Starting executions...')).toBeInTheDocument();
  });

  it('should render completed stage', () => {
    render(<ProgressHeader {...defaultProps} stage="completed" overallCompleted={true} />);

    expect(screen.getByText('Job completed successfully!')).toBeInTheDocument();
  });

  it('should render failed stage', () => {
    render(<ProgressHeader {...defaultProps} stage="failed" overallFailed={true} />);

    expect(screen.getByText('Job failed')).toBeInTheDocument();
  });

  it('should display time remaining when provided', () => {
    render(<ProgressHeader {...defaultProps} />);

    expect(screen.getByText('Time remaining: ~8:30')).toBeInTheDocument();
  });

  it('should display execution count when no time remaining', () => {
    render(<ProgressHeader {...defaultProps} timeRemaining={null} />);

    expect(screen.getByText('5/10 executions')).toBeInTheDocument();
  });

  it('should render progress bar with correct segments', () => {
    const { container } = render(<ProgressHeader {...defaultProps} />);

    const greenBar = container.querySelector('.bg-green-500');
    const yellowBar = container.querySelector('.bg-yellow-500');
    const redBar = container.querySelector('.bg-red-500');

    expect(greenBar).toBeInTheDocument();
    expect(yellowBar).toBeInTheDocument();
    expect(redBar).toBeInTheDocument();
  });

  it('should show shimmer animation when active', () => {
    const { container } = render(<ProgressHeader {...defaultProps} showShimmer={true} />);

    const shimmer = container.querySelector('.animate-shimmer');
    expect(shimmer).toBeInTheDocument();
  });

  it('should not show shimmer when inactive', () => {
    const { container } = render(<ProgressHeader {...defaultProps} showShimmer={false} />);

    const shimmer = container.querySelector('.animate-shimmer');
    expect(shimmer).not.toBeInTheDocument();
  });

  it('should display question and model breakdown', () => {
    render(<ProgressHeader {...defaultProps} />);

    expect(screen.getByText(/2 questions × 1 models × 2 regions/)).toBeInTheDocument();
  });

  it('should display only regions when no questions', () => {
    render(<ProgressHeader {...defaultProps} hasQuestions={false} />);

    expect(screen.getByText('2 regions')).toBeInTheDocument();
  });

  it('should show processing indicator when not completed', () => {
    render(<ProgressHeader {...defaultProps} />);

    expect(screen.getByText('Processing...')).toBeInTheDocument();
  });

  it('should calculate progress bar widths correctly', () => {
    const { container } = render(<ProgressHeader {...defaultProps} />);

    const greenBar = container.querySelector('.bg-green-500');
    expect(greenBar).toHaveStyle({ width: '50%' }); // 5/10 = 50%
  });

  it('should show pulse animation on progress bar when shimmer is active', () => {
    const { container } = render(<ProgressHeader {...defaultProps} showShimmer={true} />);

    const progressBar = container.querySelector('.animate-pulse');
    expect(progressBar).toBeInTheDocument();
  });

  it('should render animated icons for different stages', () => {
    const { container, rerender } = render(<ProgressHeader {...defaultProps} stage="creating" />);

    let spinner = container.querySelector('.animate-spin');
    expect(spinner).toBeInTheDocument();

    rerender(<ProgressHeader {...defaultProps} stage="running" />);
    
    const ping = container.querySelector('.animate-ping');
    expect(ping).toBeInTheDocument();
  });

  it('should display execution count in bottom right', () => {
    render(<ProgressHeader {...defaultProps} />);

    expect(screen.getByText('5/10 executions')).toBeInTheDocument();
  });

  it('should handle zero executions', () => {
    render(<ProgressHeader {...defaultProps} completed={0} running={0} failed={0} total={10} />);

    expect(screen.getByText('0/10 executions')).toBeInTheDocument();
  });

  it('should handle all executions completed', () => {
    render(<ProgressHeader {...defaultProps} completed={10} running={0} failed={0} total={10} percentage={100} />);

    expect(screen.getByText('10/10 executions')).toBeInTheDocument();
  });
});
