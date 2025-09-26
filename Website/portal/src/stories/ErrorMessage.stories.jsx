import React, { useState } from 'react';
import ErrorMessage from '../components/ErrorMessage.jsx';

const meta = {
  title: 'Feedback/ErrorMessage',
  component: ErrorMessage,
  tags: ['autodocs'],
  argTypes: {
    error: {
      control: 'object',
      description: 'Structured or string error payload that drives copy and styling.'
    },
    retryAfter: {
      control: 'number',
      description: 'Optional recommended retry window in seconds.'
    },
    onRetry: {
      action: 'retryRequested',
      description: 'Callback invoked when the retry button is pressed.'
    }
  },
  args: {
    error: {
      code: 'DATABASE_CONNECTION_FAILED',
      user_message: 'Neon Postgres is unreachable. Investigate Fly machine health or credentials.'
    },
    retryAfter: 60
  }
};

export default meta;

export const StructuredError = {
  render: (args) => (
    <div className="bg-ctp-base text-ctp-text p-6 min-h-[260px]">
      <div className="max-w-xl">
        <ErrorMessage {...args} />
      </div>
    </div>
  )
};

export const GenericError = {
  args: {
    error: 'Unexpected failure while fetching job executions.',
    retryAfter: undefined
  },
  render: StructuredError.render
};

export const RetryDisabled = {
  args: {
    error: { user_message: 'This error is informational only.' },
    onRetry: undefined
  },
  render: StructuredError.render
};
