/**
 * Unit tests for ProgressActions component
 */

import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import ProgressActions from '../ProgressActions';

const renderWithRouter = (component) => {
  return render(<BrowserRouter>{component}</BrowserRouter>);
};

describe('ProgressActions', () => {
  const mockOnRefresh = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should render refresh button', () => {
    renderWithRouter(<ProgressActions jobId="test-job" isCompleted={false} onRefresh={mockOnRefresh} />);

    expect(screen.getByText('Refresh')).toBeInTheDocument();
  });

  it('should call onRefresh when refresh button is clicked', () => {
    renderWithRouter(<ProgressActions jobId="test-job" isCompleted={false} onRefresh={mockOnRefresh} />);

    const refreshButton = screen.getByText('Refresh');
    fireEvent.click(refreshButton);

    expect(mockOnRefresh).toHaveBeenCalledTimes(1);
  });

  it('should render disabled View Diffs button when not completed', () => {
    renderWithRouter(<ProgressActions jobId="test-job" isCompleted={false} onRefresh={mockOnRefresh} />);

    const diffButton = screen.getByText('View Cross-Region Diffs');
    expect(diffButton).toBeDisabled();
    expect(diffButton).toHaveClass('opacity-50', 'cursor-not-allowed');
  });

  it('should render enabled View Diffs link when completed', () => {
    renderWithRouter(<ProgressActions jobId="test-job" isCompleted={true} onRefresh={mockOnRefresh} />);

    const diffLink = screen.getByText('View Cross-Region Diffs');
    expect(diffLink).not.toBeDisabled();
    expect(diffLink).toHaveAttribute('href', '/results/test-job/diffs');
  });

  it('should render View full results link when jobId exists', () => {
    renderWithRouter(<ProgressActions jobId="test-job-123" isCompleted={false} onRefresh={mockOnRefresh} />);

    const resultsLink = screen.getByText('View full results');
    expect(resultsLink).toBeInTheDocument();
    expect(resultsLink).toHaveAttribute('href', '/jobs/test-job-123');
  });

  it('should not render View full results link when jobId is null', () => {
    renderWithRouter(<ProgressActions jobId={null} isCompleted={false} onRefresh={mockOnRefresh} />);

    expect(screen.queryByText('View full results')).not.toBeInTheDocument();
  });

  it('should not render View Diffs button when jobId is null', () => {
    renderWithRouter(<ProgressActions jobId={null} isCompleted={false} onRefresh={mockOnRefresh} />);

    expect(screen.queryByText('View Cross-Region Diffs')).not.toBeInTheDocument();
  });

  it('should show tooltip on disabled Diffs button', () => {
    renderWithRouter(<ProgressActions jobId="test-job" isCompleted={false} onRefresh={mockOnRefresh} />);

    const diffButton = screen.getByText('View Cross-Region Diffs');
    expect(diffButton).toHaveAttribute('title', 'Available when job completes');
  });

  it('should style refresh button correctly', () => {
    renderWithRouter(<ProgressActions jobId="test-job" isCompleted={false} onRefresh={mockOnRefresh} />);

    const refreshButton = screen.getByText('Refresh');
    expect(refreshButton).toHaveClass('bg-green-600', 'text-white', 'hover:bg-green-700');
  });

  it('should style enabled Diffs link correctly', () => {
    renderWithRouter(<ProgressActions jobId="test-job" isCompleted={true} onRefresh={mockOnRefresh} />);

    const diffLink = screen.getByText('View Cross-Region Diffs');
    expect(diffLink).toHaveClass('bg-beacon-600', 'text-white', 'hover:bg-beacon-700');
  });

  it('should style results link correctly', () => {
    renderWithRouter(<ProgressActions jobId="test-job" isCompleted={false} onRefresh={mockOnRefresh} />);

    const resultsLink = screen.getByText('View full results');
    expect(resultsLink).toHaveClass('text-beacon-600', 'underline', 'decoration-dotted');
  });

  it('should render all buttons in correct order', () => {
    const { container } = renderWithRouter(
      <ProgressActions jobId="test-job" isCompleted={true} onRefresh={mockOnRefresh} />
    );

    const buttons = container.querySelectorAll('button, a');
    expect(buttons[0]).toHaveTextContent('Refresh');
    expect(buttons[1]).toHaveTextContent('View Cross-Region Diffs');
    expect(buttons[2]).toHaveTextContent('View full results');
  });

  it('should handle undefined jobId', () => {
    renderWithRouter(<ProgressActions jobId={undefined} isCompleted={false} onRefresh={mockOnRefresh} />);

    expect(screen.queryByText('View Cross-Region Diffs')).not.toBeInTheDocument();
    expect(screen.queryByText('View full results')).not.toBeInTheDocument();
  });

  it('should handle empty string jobId', () => {
    renderWithRouter(<ProgressActions jobId="" isCompleted={false} onRefresh={mockOnRefresh} />);

    expect(screen.queryByText('View Cross-Region Diffs')).not.toBeInTheDocument();
    expect(screen.queryByText('View full results')).not.toBeInTheDocument();
  });

  it('should use correct URL encoding for jobId', () => {
    renderWithRouter(<ProgressActions jobId="test job with spaces" isCompleted={true} onRefresh={mockOnRefresh} />);

    const diffLink = screen.getByText('View Cross-Region Diffs');
    expect(diffLink).toHaveAttribute('href', '/results/test job with spaces/diffs');
  });

  it('should maintain button functionality after multiple renders', () => {
    const { rerender } = renderWithRouter(
      <ProgressActions jobId="test-job" isCompleted={false} onRefresh={mockOnRefresh} />
    );

    const refreshButton = screen.getByText('Refresh');
    fireEvent.click(refreshButton);
    expect(mockOnRefresh).toHaveBeenCalledTimes(1);

    rerender(
      <BrowserRouter>
        <ProgressActions jobId="test-job" isCompleted={false} onRefresh={mockOnRefresh} />
      </BrowserRouter>
    );

    fireEvent.click(refreshButton);
    expect(mockOnRefresh).toHaveBeenCalledTimes(2);
  });
});
