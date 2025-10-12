/**
 * Unit tests for ProgressBreakdown component
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import ProgressBreakdown from '../ProgressBreakdown';

describe('ProgressBreakdown', () => {
  const defaultProps = {
    completed: 5,
    running: 3,
    failed: 1,
    pending: 1,
    hasQuestions: false,
    displayQuestions: [],
    executions: [],
    specModels: [],
    selectedRegions: ['US', 'EU'],
    uniqueModels: ['llama3.2-1b']
  };

  it('should render status breakdown with counts', () => {
    render(<ProgressBreakdown {...defaultProps} />);

    expect(screen.getByText('Completed: 5')).toBeInTheDocument();
    expect(screen.getByText('Running: 3')).toBeInTheDocument();
    expect(screen.getByText('Failed: 1')).toBeInTheDocument();
    expect(screen.getByText('Pending: 1')).toBeInTheDocument();
  });

  it('should render color-coded status indicators', () => {
    const { container } = render(<ProgressBreakdown {...defaultProps} />);

    const greenDot = container.querySelector('.bg-green-500');
    const yellowDot = container.querySelector('.bg-yellow-500');
    const redDot = container.querySelector('.bg-red-500');
    const grayDot = container.querySelector('.bg-gray-500');

    expect(greenDot).toBeInTheDocument();
    expect(yellowDot).toBeInTheDocument();
    expect(redDot).toBeInTheDocument();
    expect(grayDot).toBeInTheDocument();
  });

  it('should animate running indicator when running > 0', () => {
    const { container } = render(<ProgressBreakdown {...defaultProps} running={3} />);

    const runningDot = container.querySelectorAll('.bg-yellow-500')[0];
    expect(runningDot).toHaveClass('animate-pulse');
  });

  it('should not animate running indicator when running = 0', () => {
    const { container } = render(<ProgressBreakdown {...defaultProps} running={0} />);

    const runningDot = container.querySelectorAll('.bg-yellow-500')[0];
    expect(runningDot).not.toHaveClass('animate-pulse');
  });

  it('should not render question progress section (removed from UI)', () => {
    render(<ProgressBreakdown {...defaultProps} hasQuestions={false} />);

    expect(screen.queryByText('Question Progress')).not.toBeInTheDocument();
  });

  it('should not render question progress section even when hasQuestions is true (section removed)', () => {
    const propsWithQuestions = {
      ...defaultProps,
      hasQuestions: true,
      displayQuestions: ['q1', 'q2'],
      executions: [
        { question_id: 'q1', status: 'completed' },
        { question_id: 'q1', status: 'running' },
        { question_id: 'q2', status: 'completed' }
      ],
      specModels: [{ regions: ['US', 'EU'] }]
    };

    render(<ProgressBreakdown {...propsWithQuestions} />);

    expect(screen.queryByText('Question Progress')).not.toBeInTheDocument();
    expect(screen.queryByText('q1')).not.toBeInTheDocument();
    expect(screen.queryByText('q2')).not.toBeInTheDocument();
  });

  it('should not display question completion counts (section removed)', () => {
    const propsWithQuestions = {
      ...defaultProps,
      hasQuestions: true,
      displayQuestions: ['q1'],
      executions: [
        { question_id: 'q1', status: 'completed' },
        { question_id: 'q1', status: 'running' }
      ],
      specModels: [{ regions: ['US', 'EU'] }]
    };

    render(<ProgressBreakdown {...propsWithQuestions} />);

    expect(screen.queryByText('1/2')).not.toBeInTheDocument();
  });

  it('should not display refusal badges (question progress section removed)', () => {
    const propsWithRefusals = {
      ...defaultProps,
      hasQuestions: true,
      displayQuestions: ['q1'],
      executions: [
        { question_id: 'q1', status: 'completed', response_classification: 'content_refusal' },
        { question_id: 'q1', status: 'completed', is_content_refusal: true }
      ],
      specModels: [{ regions: ['US'] }]
    };

    render(<ProgressBreakdown {...propsWithRefusals} />);

    expect(screen.queryByText('2 refusals')).not.toBeInTheDocument();
  });

  it('should not display refusal badge when no refusals', () => {
    const propsWithoutRefusals = {
      ...defaultProps,
      hasQuestions: true,
      displayQuestions: ['q1'],
      executions: [
        { question_id: 'q1', status: 'completed' }
      ],
      specModels: [{ regions: ['US'] }]
    };

    render(<ProgressBreakdown {...propsWithoutRefusals} />);

    expect(screen.queryByText(/refusals/)).not.toBeInTheDocument();
  });

  it('should not display multiple questions (section removed)', () => {
    const propsMultipleQuestions = {
      ...defaultProps,
      hasQuestions: true,
      displayQuestions: ['q1', 'q2', 'q3'],
      executions: [
        { question_id: 'q1', status: 'completed' },
        { question_id: 'q1', status: 'completed' },
        { question_id: 'q2', status: 'completed' },
        { question_id: 'q3', status: 'running' }
      ],
      specModels: [{ regions: ['US', 'EU'] }]
    };

    render(<ProgressBreakdown {...propsMultipleQuestions} />);

    expect(screen.queryByText('q1')).not.toBeInTheDocument();
    expect(screen.queryByText('q2')).not.toBeInTheDocument();
    expect(screen.queryByText('q3')).not.toBeInTheDocument();
  });

  it('should not render question IDs (section removed)', () => {
    const propsWithQuestions = {
      ...defaultProps,
      hasQuestions: true,
      displayQuestions: ['q1'],
      executions: [{ question_id: 'q1', status: 'completed' }],
      specModels: [{ regions: ['US'] }]
    };

    render(<ProgressBreakdown {...propsWithQuestions} />);

    expect(screen.queryByText('q1')).not.toBeInTheDocument();
  });

  it('should handle zero counts gracefully', () => {
    const zeroProps = {
      ...defaultProps,
      completed: 0,
      running: 0,
      failed: 0,
      pending: 0
    };

    render(<ProgressBreakdown {...zeroProps} />);

    expect(screen.getByText('Completed: 0')).toBeInTheDocument();
    expect(screen.getByText('Running: 0')).toBeInTheDocument();
    expect(screen.getByText('Failed: 0')).toBeInTheDocument();
    expect(screen.getByText('Pending: 0')).toBeInTheDocument();
  });

  it('should not render question progress section styling (section removed)', () => {
    const propsWithQuestions = {
      ...defaultProps,
      hasQuestions: true,
      displayQuestions: ['q1'],
      executions: [{ question_id: 'q1', status: 'completed' }],
      specModels: [{ regions: ['US'] }]
    };

    const { container } = render(<ProgressBreakdown {...propsWithQuestions} />);

    const questionSection = container.querySelector('.bg-gray-800\\/50');
    expect(questionSection).not.toBeInTheDocument();
  });
});
