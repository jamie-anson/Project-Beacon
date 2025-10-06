/**
 * Unit tests for FailureAlert component
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import FailureAlert from '../FailureAlert';

describe('FailureAlert', () => {
  const mockFailureInfo = {
    title: 'Job Failed',
    message: 'Job failed with status: failed. This may be due to system issues.',
    action: 'Try submitting a new job or contact support.'
  };

  it('should render failure alert with all information', () => {
    render(<FailureAlert failureInfo={mockFailureInfo} />);

    expect(screen.getByText('Job Failed')).toBeInTheDocument();
    expect(screen.getByText(/Job failed with status: failed/)).toBeInTheDocument();
    expect(screen.getByText(/Try submitting a new job/)).toBeInTheDocument();
  });

  it('should not render when failureInfo is null', () => {
    const { container } = render(<FailureAlert failureInfo={null} />);

    expect(container.firstChild).toBeNull();
  });

  it('should not render when failureInfo is undefined', () => {
    const { container } = render(<FailureAlert failureInfo={undefined} />);

    expect(container.firstChild).toBeNull();
  });

  it('should render with correct styling classes', () => {
    const { container } = render(<FailureAlert failureInfo={mockFailureInfo} />);

    const alertBox = container.querySelector('.bg-red-900\\/20');
    expect(alertBox).toBeInTheDocument();
    expect(alertBox).toHaveClass('border', 'border-red-700', 'rounded-lg');
  });

  it('should render error icon', () => {
    const { container } = render(<FailureAlert failureInfo={mockFailureInfo} />);

    const icon = container.querySelector('svg');
    expect(icon).toBeInTheDocument();
    expect(icon).toHaveClass('text-red-400');
  });

  it('should render title with correct styling', () => {
    render(<FailureAlert failureInfo={mockFailureInfo} />);

    const title = screen.getByText('Job Failed');
    expect(title).toHaveClass('text-red-400', 'font-medium', 'text-sm');
  });

  it('should render message with correct styling', () => {
    render(<FailureAlert failureInfo={mockFailureInfo} />);

    const message = screen.getByText(/Job failed with status: failed/);
    expect(message).toHaveClass('text-red-300', 'text-sm');
  });

  it('should render action with correct styling', () => {
    render(<FailureAlert failureInfo={mockFailureInfo} />);

    const action = screen.getByText(/Try submitting a new job/);
    expect(action).toHaveClass('text-red-200', 'text-xs');
  });

  it('should render timeout failure info', () => {
    const timeoutInfo = {
      title: 'Job Timeout',
      message: 'Job has been running for 20 minutes without creating any executions.',
      action: 'The job may be stuck. Try submitting a new job.'
    };

    render(<FailureAlert failureInfo={timeoutInfo} />);

    expect(screen.getByText('Job Timeout')).toBeInTheDocument();
    expect(screen.getByText(/20 minutes/)).toBeInTheDocument();
  });

  it('should handle long messages', () => {
    const longMessage = {
      title: 'Error',
      message: 'A'.repeat(500),
      action: 'Please try again'
    };

    render(<FailureAlert failureInfo={longMessage} />);

    expect(screen.getByText('Error')).toBeInTheDocument();
    expect(screen.getByText('A'.repeat(500))).toBeInTheDocument();
  });

  it('should handle special characters in messages', () => {
    const specialCharsInfo = {
      title: 'Error: <script>alert("xss")</script>',
      message: 'Failed with error: {"code": 500}',
      action: 'Contact support@example.com'
    };

    render(<FailureAlert failureInfo={specialCharsInfo} />);

    expect(screen.getByText(/Error: <script>alert/)).toBeInTheDocument();
  });
});
