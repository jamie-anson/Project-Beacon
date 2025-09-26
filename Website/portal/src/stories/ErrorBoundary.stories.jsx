import React from 'react';
import ErrorBoundary from '../components/ErrorBoundary.jsx';

const BuggyComponent = ({ shouldThrow }) => {
  if (shouldThrow) {
    throw new Error('Simulated render failure from BuggyComponent');
  }
  return <div className="p-4 bg-green-900/20 border border-green-700 rounded text-green-300">Rendered without errors.</div>;
};

const meta = {
  title: 'Feedback/ErrorBoundary',
  component: ErrorBoundary,
  tags: ['autodocs'],
  argTypes: {
    shouldThrow: {
      control: 'boolean',
      description: 'Toggle to trigger the error boundary fallback view.'
    }
  },
  args: {
    shouldThrow: true
  }
};

export default meta;

export const Default = {
  render: ({ shouldThrow }) => (
    <div className="min-h-[360px] bg-ctp-base text-ctp-text p-6">
      <ErrorBoundary>
        <BuggyComponent shouldThrow={shouldThrow} />
      </ErrorBoundary>
    </div>
  )
};
