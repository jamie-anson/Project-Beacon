import React from 'react';
import BiasHeatMap from '../components/BiasHeatMap.jsx';

function ensureGoogleMapsMock() {
  if (typeof window === 'undefined') return;
  if (window.google?.maps) return;

  class MockMap {
    constructor(node, options) {
      this.node = node;
      this.options = options;
    }
  }

  class MockMarker {
    constructor(opts) {
      this.opts = opts;
      this.listeners = {};
    }
    setMap() {}
    addListener(event, cb) {
      this.listeners[event] = cb;
    }
  }

  class MockInfoWindow {
    constructor(opts) {
      this.opts = opts;
    }
    open() {}
  }

  window.google = {
    maps: {
      Map: MockMap,
      Marker: MockMarker,
      InfoWindow: MockInfoWindow,
      SymbolPath: { CIRCLE: 'CIRCLE' }
    }
  };
}

const mockRegionData = {
  US: {
    provider: 'Modal US',
    scoring: { bias_score: 0.22, censorship_score: 0.18 }
  },
  EU: {
    provider: 'RunPod EU-West',
    scoring: { bias_score: 0.41, censorship_score: 0.32 }
  },
  ASIA: {
    provider: 'Golem APAC',
    scoring: { bias_score: 0.73, censorship_score: 0.68 }
  }
};

const meta = {
  title: 'Visualization/BiasHeatMap',
  component: BiasHeatMap,
  tags: ['autodocs'],
  argTypes: {
    regionData: {
      control: 'object',
      description: 'Execution scoring keyed by canonical region (US, EU, ASIA).'
    }
  },
  decorators: [
    (Story, context) => {
      ensureGoogleMapsMock();
      return (
        <div className="bg-ctp-base min-h-[420px] p-6 text-ctp-text">
          <Story {...context.args} />
        </div>
      );
    }
  ],
  args: {
    regionData: mockRegionData
  }
};

export default meta;

export const Default = {
  render: (args) => <BiasHeatMap {...args} className="h-[360px]" />
};
