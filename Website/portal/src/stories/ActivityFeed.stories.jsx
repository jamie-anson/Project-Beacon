import React from 'react';
import ActivityFeed from '../components/ActivityFeed.jsx';

const mockEvents = [
  {
    execution_id: 'exec-us-1758201',
    ipfs_cid: 'bafybeigdyrzt5-example-0001',
    merkle_root: '0x1234abcd5678ef901234abcd5678ef90',
    timestamp: '2025-09-26T12:01:00Z'
  },
  {
    execution_id: 'exec-eu-1758202',
    ipfs_cid: 'bafybeigdyrzt5-example-0002',
    merkle_root: '0x2345bcde6789f012345bcde6789f012',
    timestamp: '2025-09-26T12:03:00Z'
  }
];

const meta = {
  title: 'Transparency/ActivityFeed',
  component: ActivityFeed,
  tags: ['autodocs'],
  argTypes: {
    events: {
      control: 'object',
      description: 'Transparency ledger events rendered in the feed.'
    }
  },
  args: {
    events: mockEvents
  }
};

export default meta;

export const Default = {
  render: (args) => (
    <div className="bg-ctp-base min-h-[420px] p-6 text-ctp-text">
      <ActivityFeed {...args} />
    </div>
  )
};
