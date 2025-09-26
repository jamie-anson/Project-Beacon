import React from 'react';
import InfrastructureStatus from '../components/InfrastructureStatus.jsx';

const mockHealth = {
  overall_status: 'degraded',
  healthy_services: 4,
  degraded_services: 2,
  down_services: 1,
  total_services: 7,
  last_checked: new Date().toISOString(),
  services: {
    router: { status: 'healthy', response_time_ms: 120, error: null },
    'modal_us': { status: 'healthy', response_time_ms: 310, error: null },
    'modal_eu': { status: 'degraded', response_time_ms: 680, error: 'modal CLI cold start detected' },
    'modal_apac': { status: 'down', response_time_ms: null, error: 'Timeout contacting Modal APAC function' },
    'runpod_us': { status: 'healthy', response_time_ms: 220, error: null },
    'runpod_eu': { status: 'healthy', response_time_ms: 205, error: null },
    'golem_network': { status: 'degraded', response_time_ms: 950, error: 'market subscription slow to respond' }
  }
};

const meta = {
  title: 'Status/InfrastructureStatus',
  component: InfrastructureStatus,
  tags: ['autodocs'],
  argTypes: {
    compact: {
      control: 'boolean',
      description: 'Renders condensed inline status instead of full card layout.'
    }
  },
  args: {
    compact: false
  }
};

export default meta;

export const Standard = {
  render: (args) => (
    <div className="bg-ctp-base min-h-[420px] p-6 text-ctp-text">
      <InfrastructureStatus
        {...args}
        initialHealth={mockHealth}
        fetchHealth={async () => mockHealth}
        refreshIntervalMs={0}
      />
    </div>
  )
};

export const Compact = {
  args: {
    compact: true
  },
  render: (args) => (
    <div className="bg-ctp-base p-6 text-ctp-text">
      <InfrastructureStatus
        {...args}
        initialHealth={mockHealth}
        fetchHealth={null}
        refreshIntervalMs={0}
      />
    </div>
  )
};
