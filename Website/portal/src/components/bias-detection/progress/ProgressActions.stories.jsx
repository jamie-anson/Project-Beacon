/**
 * Storybook stories for ProgressActions component
 * Visual testing for retry button and action states
 */

import React from 'react';
import { BrowserRouter } from 'react-router-dom';
import ProgressActions from './ProgressActions';

export default {
  title: 'BiasDetection/Progress/ProgressActions',
  component: ProgressActions,
  decorators: [
    (Story) => (
      <BrowserRouter>
        <div className="bg-gray-900 p-4">
          <Story />
        </div>
      </BrowserRouter>
    ),
  ],
};

const mockOnRefresh = () => console.log('Refresh clicked');
const mockOnRetryJob = () => console.log('Retry Job clicked');

// Job in progress (no retry button)
export const JobInProgress = {
  args: {
    jobId: 'test-job-123',
    isCompleted: false,
    isFailed: false,
    onRefresh: mockOnRefresh,
    onRetryJob: mockOnRetryJob
  }
};

// Job failed (shows retry button)
export const JobFailed = {
  args: {
    jobId: 'test-job-123',
    isCompleted: false,
    isFailed: true,
    onRefresh: mockOnRefresh,
    onRetryJob: mockOnRetryJob
  }
};

// Job completed successfully (no retry button)
export const JobCompleted = {
  args: {
    jobId: 'test-job-123',
    isCompleted: true,
    isFailed: false,
    onRefresh: mockOnRefresh,
    onRetryJob: mockOnRetryJob
  }
};

// Job timeout (shows retry button)
export const JobTimeout = {
  args: {
    jobId: 'test-job-timeout-456',
    isCompleted: false,
    isFailed: true,
    onRefresh: mockOnRefresh,
    onRetryJob: mockOnRetryJob
  }
};

// No job ID (minimal buttons)
export const NoJobId = {
  args: {
    jobId: null,
    isCompleted: false,
    isFailed: false,
    onRefresh: mockOnRefresh,
    onRetryJob: mockOnRetryJob
  }
};

// Failed job without retry handler (no retry button)
export const FailedWithoutRetryHandler = {
  args: {
    jobId: 'test-job-123',
    isCompleted: false,
    isFailed: true,
    onRefresh: mockOnRefresh,
    onRetryJob: undefined
  }
};

// All buttons visible (completed + failed state for testing)
export const AllButtonsVisible = {
  args: {
    jobId: 'test-job-123',
    isCompleted: true,
    isFailed: true,
    onRefresh: mockOnRefresh,
    onRetryJob: mockOnRetryJob
  }
};

// Long job ID (test overflow)
export const LongJobId = {
  args: {
    jobId: 'bias-detection-multi-region-comprehensive-test-1234567890',
    isCompleted: true,
    isFailed: false,
    onRefresh: mockOnRefresh,
    onRetryJob: mockOnRetryJob
  }
};

// Mobile view simulation
export const MobileView = {
  args: {
    jobId: 'test-job-123',
    isCompleted: false,
    isFailed: true,
    onRefresh: mockOnRefresh,
    onRetryJob: mockOnRetryJob
  },
  parameters: {
    viewport: {
      defaultViewport: 'mobile1'
    }
  }
};

// Interactive demo
export const InteractiveDemo = {
  render: () => {
    const [isCompleted, setIsCompleted] = React.useState(false);
    const [isFailed, setIsFailed] = React.useState(false);

    return (
      <div className="space-y-4">
        <div className="flex gap-2 mb-4">
          <button
            onClick={() => {
              setIsCompleted(false);
              setIsFailed(false);
            }}
            className="px-3 py-1 bg-blue-600 text-white rounded text-sm"
          >
            Set In Progress
          </button>
          <button
            onClick={() => {
              setIsCompleted(false);
              setIsFailed(true);
            }}
            className="px-3 py-1 bg-red-600 text-white rounded text-sm"
          >
            Set Failed
          </button>
          <button
            onClick={() => {
              setIsCompleted(true);
              setIsFailed(false);
            }}
            className="px-3 py-1 bg-green-600 text-white rounded text-sm"
          >
            Set Completed
          </button>
        </div>

        <div className="text-sm text-gray-400 mb-2">
          Current State: {isFailed ? 'Failed' : isCompleted ? 'Completed' : 'In Progress'}
        </div>

        <ProgressActions
          jobId="interactive-demo-job"
          isCompleted={isCompleted}
          isFailed={isFailed}
          onRefresh={() => console.log('Refresh clicked')}
          onRetryJob={() => {
            console.log('Retry clicked');
            setIsFailed(false);
            setIsCompleted(false);
          }}
        />
      </div>
    );
  }
};
