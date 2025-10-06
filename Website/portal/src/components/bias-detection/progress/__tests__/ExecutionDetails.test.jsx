/**
 * Unit tests for ExecutionDetails component
 */

import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import ExecutionDetails from '../ExecutionDetails';

const renderWithRouter = (component) => {
  return render(<BrowserRouter>{component}</BrowserRouter>);
};

describe('ExecutionDetails', () => {
  const mockOnRetry = jest.fn();
  const mockIsRetrying = jest.fn(() => false);

  const defaultProps = {
    region: 'us-east',
    regionExecs: [
      {
        id: 'exec-1',
        model_id: 'llama3.2-1b',
        question_id: 'q1',
        status: 'completed',
        response_classification: 'substantive'
      },
      {
        id: 'exec-2',
        model_id: 'llama3.2-1b',
        question_id: 'q2',
        status: 'failed',
        error: 'Timeout'
      },
      {
        id: 'exec-3',
        model_id: 'mistral-7b',
        question_id: 'q1',
        status: 'completed',
        is_content_refusal: true
      }
    ],
    uniqueModels: ['llama3.2-1b', 'mistral-7b'],
    uniqueQuestions: ['q1', 'q2'],
    onRetry: mockOnRetry,
    isRetrying: mockIsRetrying
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should render execution details header', () => {
    renderWithRouter(<ExecutionDetails {...defaultProps} />);

    expect(screen.getByText('Execution Details for us-east')).toBeInTheDocument();
  });

  it('should render model sections', () => {
    renderWithRouter(<ExecutionDetails {...defaultProps} />);

    expect(screen.getByText('llama3.2-1b')).toBeInTheDocument();
    expect(screen.getByText('mistral-7b')).toBeInTheDocument();
  });

  it('should render question IDs', () => {
    renderWithRouter(<ExecutionDetails {...defaultProps} />);

    const questionIds = screen.getAllByText(/^q[12]$/);
    expect(questionIds.length).toBeGreaterThan(0);
  });

  it('should display status badges', () => {
    renderWithRouter(<ExecutionDetails {...defaultProps} />);

    expect(screen.getByText('completed')).toBeInTheDocument();
    expect(screen.getByText('failed')).toBeInTheDocument();
  });

  it('should display substantive classification badge', () => {
    renderWithRouter(<ExecutionDetails {...defaultProps} />);

    expect(screen.getByText('✓ Substantive')).toBeInTheDocument();
  });

  it('should display refusal classification badge', () => {
    renderWithRouter(<ExecutionDetails {...defaultProps} />);

    expect(screen.getByText('⚠ Refusal')).toBeInTheDocument();
  });

  it('should display — for executions without classification', () => {
    const propsNoClassification = {
      ...defaultProps,
      regionExecs: [
        {
          id: 'exec-1',
          model_id: 'llama3.2-1b',
          question_id: 'q1',
          status: 'completed'
        }
      ]
    };

    renderWithRouter(<ExecutionDetails {...propsNoClassification} />);

    expect(screen.getByText('—')).toBeInTheDocument();
  });

  it('should render Answer link for successful executions', () => {
    renderWithRouter(<ExecutionDetails {...defaultProps} />);

    const answerLinks = screen.getAllByText('Answer');
    expect(answerLinks.length).toBeGreaterThan(0);
    expect(answerLinks[0]).toHaveAttribute('href', '/portal/executions/exec-1');
  });

  it('should render Retry button for failed executions', () => {
    renderWithRouter(<ExecutionDetails {...defaultProps} />);

    expect(screen.getByText('Retry')).toBeInTheDocument();
  });

  it('should call onRetry when Retry button is clicked', () => {
    renderWithRouter(<ExecutionDetails {...defaultProps} />);

    const retryButton = screen.getByText('Retry');
    fireEvent.click(retryButton);

    expect(mockOnRetry).toHaveBeenCalledWith('exec-2', 'us-east', 1); // q2 is at index 1
  });

  it('should disable Retry button when retrying', () => {
    mockIsRetrying.mockReturnValue(true);

    renderWithRouter(<ExecutionDetails {...defaultProps} />);

    const retryButton = screen.getByText('Retrying...');
    expect(retryButton).toBeDisabled();
  });

  it('should show Retrying... text when retry is in progress', () => {
    mockIsRetrying.mockReturnValue(true);

    renderWithRouter(<ExecutionDetails {...defaultProps} />);

    expect(screen.getByText('Retrying...')).toBeInTheDocument();
  });

  it('should render executions in model × question grid', () => {
    const { container } = renderWithRouter(<ExecutionDetails {...defaultProps} />);

    const grid = container.querySelector('.grid-cols-4');
    expect(grid).toBeInTheDocument();
  });

  it('should apply hover effect to execution rows', () => {
    const { container } = renderWithRouter(<ExecutionDetails {...defaultProps} />);

    const hoverRow = container.querySelector('.hover\\:bg-gray-700\\/50');
    expect(hoverRow).toBeInTheDocument();
  });

  it('should render question IDs in monospace font', () => {
    renderWithRouter(<ExecutionDetails {...defaultProps} />);

    const questionId = screen.getAllByText(/^q[12]$/)[0];
    expect(questionId).toHaveClass('font-mono');
  });

  it('should handle empty executions', () => {
    const emptyProps = {
      ...defaultProps,
      regionExecs: [],
      uniqueModels: [],
      uniqueQuestions: []
    };

    const { container } = renderWithRouter(<ExecutionDetails {...emptyProps} />);

    expect(screen.getByText('Execution Details for us-east')).toBeInTheDocument();
    expect(container.querySelectorAll('.grid-cols-4').length).toBe(0);
  });

  it('should not render execution if model/question combination not found', () => {
    const propsPartialExecs = {
      ...defaultProps,
      uniqueModels: ['llama3.2-1b', 'mistral-7b'],
      uniqueQuestions: ['q1', 'q2', 'q3'], // q3 doesn't exist in executions
      regionExecs: defaultProps.regionExecs
    };

    renderWithRouter(<ExecutionDetails {...propsPartialExecs} />);

    // Should only render rows for existing combinations
    const grids = screen.getAllByText(/^q[123]$/).length;
    expect(grids).toBeLessThan(6); // Less than 2 models × 3 questions
  });

  it('should handle multiple models and questions', () => {
    const multiProps = {
      ...defaultProps,
      uniqueModels: ['model1', 'model2', 'model3'],
      uniqueQuestions: ['q1', 'q2', 'q3', 'q4'],
      regionExecs: [
        { id: '1', model_id: 'model1', question_id: 'q1', status: 'completed' },
        { id: '2', model_id: 'model2', question_id: 'q2', status: 'running' },
        { id: '3', model_id: 'model3', question_id: 'q3', status: 'failed' }
      ]
    };

    renderWithRouter(<ExecutionDetails {...multiProps} />);

    expect(screen.getByText('model1')).toBeInTheDocument();
    expect(screen.getByText('model2')).toBeInTheDocument();
    expect(screen.getByText('model3')).toBeInTheDocument();
  });

  it('should style classification badges correctly', () => {
    const { container } = renderWithRouter(<ExecutionDetails {...defaultProps} />);

    const substantiveBadge = screen.getByText('✓ Substantive').closest('span');
    expect(substantiveBadge).toHaveClass('bg-green-900/20', 'text-green-400');

    const refusalBadge = screen.getByText('⚠ Refusal').closest('span');
    expect(refusalBadge).toHaveClass('bg-orange-900/20', 'text-orange-400');
  });

  it('should link to correct execution detail page', () => {
    renderWithRouter(<ExecutionDetails {...defaultProps} />);

    const answerLink = screen.getAllByText('Answer')[0];
    expect(answerLink.getAttribute('href')).toContain('/portal/executions/');
  });

  it('should style retry button correctly', () => {
    renderWithRouter(<ExecutionDetails {...defaultProps} />);

    const retryButton = screen.getByText('Retry');
    expect(retryButton).toHaveClass('text-yellow-400', 'hover:text-yellow-300');
  });

  it('should style answer link correctly', () => {
    renderWithRouter(<ExecutionDetails {...defaultProps} />);

    const answerLink = screen.getAllByText('Answer')[0];
    expect(answerLink).toHaveClass('text-pink-400', 'hover:text-pink-300');
  });
});
