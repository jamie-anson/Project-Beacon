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
