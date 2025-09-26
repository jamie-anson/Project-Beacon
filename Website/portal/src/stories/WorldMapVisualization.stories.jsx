import React from 'react';
import WorldMapVisualization from '../components/WorldMapVisualization.jsx';

function ensureGoogleMapsMock() {
  if (typeof window === 'undefined') return;

  if (!window.google) {
    const noop = () => {};
    window.google = {
      maps: {
        Map: function MockMap() {},
        Polygon: function MockPolygon() {},
        event: {
          addListener: noop,
          clearListeners: noop
        }
      }
    };
  }
}

const mockData = [
  { code: 'US', value: 20, category: 'low' },
  { code: 'CN', value: 85, category: 'high' },
  { code: 'SG', value: 35, category: 'medium' },
  { code: 'TH', value: 55, category: 'medium' }
];

const meta = {
  title: 'Visualization/WorldMapVisualization',
  component: WorldMapVisualization,
  tags: ['autodocs'],
  argTypes: {
    biasData: {
      control: 'object',
      description: 'Array of country annotations. Each entry should include `code`, `value`, and optional `coords`.'
    }
  },
  decorators: [
    (Story, context) => {
      ensureGoogleMapsMock();
      return (
        <div className="bg-ctp-base min-h-[520px] p-6 text-ctp-text">
          <Story {...context.args} />
        </div>
      );
    }
  ],
  args: {
    biasData: mockData
  }
};

export default meta;

export const Default = {
  render: (args) => <WorldMapVisualization {...args} />
};
