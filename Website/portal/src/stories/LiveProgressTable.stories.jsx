import React from 'react';
import { MemoryRouter } from 'react-router-dom';
import LiveProgressTable from '../components/bias-detection/LiveProgressTable.jsx';

const mockJob = {
  id: 'job-bias-detection-1758207300',
  status: 'running',
  created_at: '2025-09-26T12:00:00Z',
  executions: [
    {
      id: 'exec-us-001',
      region: 'us-east',
      status: 'running',
      provider_id: 'modal-us-east-1',
      started_at: new Date(Date.now() - 3 * 60 * 1000).toISOString(),
      retries: 0,
      eta: 42,
      verification_status: 'pending'
    },
    {
      id: 'exec-eu-001',
      region: 'eu-west',
      status: 'completed',
      provider_id: 'runpod-eu-west-1',
      started_at: new Date(Date.now() - 8 * 60 * 1000).toISOString(),
      completed_at: new Date(Date.now() - 2 * 60 * 1000).toISOString(),
      retries: 1,
      eta: null,
      verification_status: 'verified'
    },
    {
      id: 'exec-apac-001',
      region: 'asia-pacific',
      status: 'queued',
      provider_id: 'golem-asia-provider',
      retries: 0,
      eta: 120,
      verification_status: 'needs_probe'
    }
  ]
};

const meta = {
  title: 'Bias Workflow/LiveProgressTable',
  component: LiveProgressTable,
  tags: ['autodocs'],
  argTypes: {
    selectedRegions: {
      control: 'object',
      description: 'Regions targeted by the current job (US, EU, ASIA).'
    }
  },
  args: {
    activeJob: mockJob,
    selectedRegions: ['US', 'EU', 'ASIA'],
    loadingActive: false,
    refetchActive: () => {},
    activeJobId: mockJob.id,
    isCompleted: false,
    diffReady: false
  },
  decorators: [
    (Story, context) => (
      <MemoryRouter>
        <div className="bg-ctp-base min-h-[520px] p-6 text-ctp-text">
          <Story {...context.args} />
        </div>
      </MemoryRouter>
    )
  ]
};

export default meta;

export const Default = {};

// Failed Job - Job failed before creating any executions
export const FailedJob = {
  args: {
    activeJob: {
      id: 'job-bias-detection-failed-1759067583',
      status: 'failed',
      created_at: new Date(Date.now() - 5 * 60 * 1000).toISOString(), // 5 minutes ago
      executions: [] // No executions created
    },
    selectedRegions: ['US', 'EU', 'ASIA'],
    loadingActive: false,
    refetchActive: () => {},
    activeJobId: 'job-bias-detection-failed-1759067583',
    isCompleted: false,
    diffReady: false
  }
};

// Stuck Job - Job running for 20+ minutes with no executions (timeout scenario)
export const StuckJob = {
  args: {
    activeJob: {
      id: 'job-bias-detection-stuck-1759068447',
      status: 'processing',
      created_at: new Date(Date.now() - 20 * 60 * 1000).toISOString(), // 20 minutes ago
      executions: [] // No executions created
    },
    selectedRegions: ['US', 'EU', 'ASIA'],
    loadingActive: false,
    refetchActive: () => {},
    activeJobId: 'job-bias-detection-stuck-1759068447',
    isCompleted: false,
    diffReady: false
  }
};

// Mixed Execution Failures - Some executions failed, some completed
export const MixedExecutionFailures = {
  args: {
    activeJob: {
      id: 'job-bias-detection-mixed-1758207400',
      status: 'completed',
      created_at: new Date(Date.now() - 15 * 60 * 1000).toISOString(),
      executions: [
        {
          id: 'exec-us-002',
          region: 'us-east',
          status: 'completed',
          provider_id: 'modal-us-east-1',
          started_at: new Date(Date.now() - 12 * 60 * 1000).toISOString(),
          completed_at: new Date(Date.now() - 8 * 60 * 1000).toISOString(),
        },
        {
          id: 'exec-eu-002',
          region: 'eu-west',
          status: 'failed',
          provider_id: 'runpod-eu-west-1',
          started_at: new Date(Date.now() - 10 * 60 * 1000).toISOString(),
          completed_at: new Date(Date.now() - 9 * 60 * 1000).toISOString(),
          error: 'Provider timeout: Connection to model server failed after 300s',
          failure_reason: 'infrastructure_timeout'
        },
        {
          id: 'exec-apac-002',
          region: 'asia-pacific',
          status: 'completed',
          provider_id: 'golem-asia-provider',
          started_at: new Date(Date.now() - 11 * 60 * 1000).toISOString(),
          completed_at: new Date(Date.now() - 7 * 60 * 1000).toISOString(),
        }
      ]
    },
    selectedRegions: ['US', 'EU', 'ASIA'],
    loadingActive: false,
    refetchActive: () => {},
    activeJobId: 'job-bias-detection-mixed-1758207400',
    isCompleted: true,
    diffReady: true
  }
};

// Completed Job with Missing Execution Records - Job completed but some regions missing execution data
export const CompletedJobMissingExecutions = {
  args: {
    activeJob: {
      id: 'job-bias-detection-partial-1758207500',
      status: 'completed',
      created_at: new Date(Date.now() - 10 * 60 * 1000).toISOString(),
      executions: [
        {
          id: 'exec-us-003',
          region: 'us-east',
          status: 'completed',
          provider_id: 'modal-us-east-1',
          started_at: new Date(Date.now() - 8 * 60 * 1000).toISOString(),
          completed_at: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
        }
        // EU and ASIA executions missing but job marked as completed
      ]
    },
    selectedRegions: ['US', 'EU', 'ASIA'],
    loadingActive: false,
    refetchActive: () => {},
    activeJobId: 'job-bias-detection-partial-1758207500',
    isCompleted: true,
    diffReady: true
  }
};
