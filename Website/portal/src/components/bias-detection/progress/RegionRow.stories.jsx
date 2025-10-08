/**
 * Storybook stories for RegionRow component
 * Visual testing for processing animations and status states
 */

import React from 'react';
import { BrowserRouter } from 'react-router-dom';
import RegionRow from './RegionRow';

export default {
  title: 'BiasDetection/Progress/RegionRow',
  component: RegionRow,
  decorators: [
    (Story) => (
      <BrowserRouter>
        <div className="bg-gray-900 p-4">
          <div className="border border-gray-600 rounded">
            <div className="grid grid-cols-7 text-xs bg-gray-700 text-gray-300">
              <div className="px-3 py-2">Region</div>
              <div className="px-3 py-2">Progress</div>
              <div className="px-3 py-2">Status</div>
              <div className="px-3 py-2">Models</div>
              <div className="px-3 py-2">Questions</div>
              <div className="px-3 py-2">Started</div>
              <div className="px-3 py-2">Actions</div>
            </div>
            <Story />
          </div>
        </div>
      </BrowserRouter>
    ),
  ],
};

const baseProps = {
  region: 'US',
  isExpanded: false,
  onToggle: () => console.log('Toggle clicked'),
  jobCompleted: false,
  jobFailed: false,
  jobStuckTimeout: false,
  loadingActive: false,
  uniqueModels: ['llama3.2-1b', 'mistral-7b'],
  hasQuestions: true,
  jobId: 'test-job-123',
  failureInfo: null,
  jobAge: 5,
  statusStr: 'processing'
};

// Running job with processing animation
export const ProcessingWithAnimation = {
  args: {
    ...baseProps,
    regionExecs: [
      { id: 1, status: 'running', started_at: new Date().toISOString(), model_id: 'llama3.2-1b' },
      { id: 2, status: 'completed', started_at: new Date(Date.now() - 60000).toISOString(), model_id: 'mistral-7b' }
    ]
  }
};

// Processing status with shimmer
export const ProcessingStatus = {
  args: {
    ...baseProps,
    regionExecs: [
      { id: 1, status: 'processing', started_at: new Date().toISOString(), model_id: 'llama3.2-1b' },
      { id: 2, status: 'processing', started_at: new Date().toISOString(), model_id: 'mistral-7b' }
    ]
  }
};

// Completed job (no animation)
export const Completed = {
  args: {
    ...baseProps,
    jobCompleted: true,
    regionExecs: [
      { id: 1, status: 'completed', started_at: new Date(Date.now() - 120000).toISOString(), model_id: 'llama3.2-1b' },
      { id: 2, status: 'completed', started_at: new Date(Date.now() - 60000).toISOString(), model_id: 'mistral-7b' }
    ]
  }
};

// Failed job (no animation)
export const Failed = {
  args: {
    ...baseProps,
    jobFailed: true,
    statusStr: 'failed',
    failureInfo: {
      title: 'Job Failed',
      message: 'Job failed with status: failed'
    },
    regionExecs: [
      { id: 1, status: 'failed', started_at: new Date(Date.now() - 60000).toISOString(), model_id: 'llama3.2-1b' },
      { id: 2, status: 'failed', started_at: new Date(Date.now() - 60000).toISOString(), model_id: 'mistral-7b' }
    ]
  }
};

// Timeout (no animation)
export const Timeout = {
  args: {
    ...baseProps,
    jobStuckTimeout: true,
    jobAge: 20,
    regionExecs: [
      { id: 1, status: 'running', started_at: new Date(Date.now() - 1200000).toISOString(), model_id: 'llama3.2-1b' }
    ]
  }
};

// Mixed progress with running animation
export const MixedProgress = {
  args: {
    ...baseProps,
    regionExecs: [
      { id: 1, status: 'completed', started_at: new Date(Date.now() - 120000).toISOString(), model_id: 'llama3.2-1b' },
      { id: 2, status: 'running', started_at: new Date(Date.now() - 30000).toISOString(), model_id: 'mistral-7b' },
      { id: 3, status: 'pending', model_id: 'qwen2.5-1.5b' }
    ],
    uniqueModels: ['llama3.2-1b', 'mistral-7b', 'qwen2.5-1.5b']
  }
};

// Pending (no executions, no animation)
export const Pending = {
  args: {
    ...baseProps,
    regionExecs: []
  }
};

// Loading/Refreshing state
export const Refreshing = {
  args: {
    ...baseProps,
    loadingActive: true,
    regionExecs: [
      { id: 1, status: 'running', started_at: new Date().toISOString(), model_id: 'llama3.2-1b' }
    ]
  }
};

// Expanded view
export const Expanded = {
  args: {
    ...baseProps,
    isExpanded: true,
    regionExecs: [
      { id: 1, status: 'running', started_at: new Date().toISOString(), model_id: 'llama3.2-1b' },
      { id: 2, status: 'completed', started_at: new Date(Date.now() - 60000).toISOString(), model_id: 'mistral-7b' }
    ]
  }
};

// Single model running
export const SingleModelRunning = {
  args: {
    ...baseProps,
    uniqueModels: ['llama3.2-1b'],
    regionExecs: [
      { id: 1, status: 'running', started_at: new Date().toISOString(), model_id: 'llama3.2-1b' }
    ]
  }
};

// All models running (maximum animation)
export const AllModelsRunning = {
  args: {
    ...baseProps,
    uniqueModels: ['llama3.2-1b', 'mistral-7b', 'qwen2.5-1.5b'],
    regionExecs: [
      { id: 1, status: 'running', started_at: new Date().toISOString(), model_id: 'llama3.2-1b' },
      { id: 2, status: 'running', started_at: new Date().toISOString(), model_id: 'mistral-7b' },
      { id: 3, status: 'running', started_at: new Date().toISOString(), model_id: 'qwen2.5-1.5b' }
    ]
  }
};
