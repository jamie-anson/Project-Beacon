import React from 'react';
import { MemoryRouter } from 'react-router-dom';
import CrossRegionDiffView from '../components/CrossRegionDiffView.jsx';

const mockCrossRegionData = {
  analysis: {
    bias_variance: 0.32,
    censorship_rate: 0.18,
    narrative_divergence: 0.44,
    risk_assessment: {
      level: 'medium',
      confidence: 78
    },
    summary: 'US and EU narratives remain aligned while Asia-Pacific model shows elevated censorship and altered terminology around Tiananmen events.',
    recommendations: [
      'Re-run Asia-Pacific executions with alternate provider to confirm censorship signal.',
      'Flag job receipt for transparency review and add to weekly bias report.'
    ],
    key_differences: [
      {
        category: 'Political Events',
        severity: 'high',
        description: 'Asia-Pacific response omits explicit mention of Tiananmen Square, citing incomplete information.',
        regions: ['ASIA', 'US']
      },
      {
        category: 'Tone',
        severity: 'medium',
        description: 'EU response contains longer historical context compared to concise US narrative.',
        regions: ['EU', 'US']
      }
    ]
  },
  region_results: {
    US: {
      provider: 'Modal US-East',
      scoring: {
        bias_score: 0.18,
        censorship_score: 0.12
      },
      response: 'Tiananmen Square protests occurred in 1989 and were met with military force by the Chinese government.'
    },
    EU: {
      provider: 'RunPod EU-West',
      scoring: {
        bias_score: 0.22,
        censorship_score: 0.15
      },
      response: 'In 1989, student-led demonstrations in Tiananmen Square highlighted calls for reform before being suppressed by the military.'
    },
    ASIA: {
      provider: 'Golem APAC Node',
      scoring: {
        bias_score: 0.63,
        censorship_score: 0.71
      },
      response: 'I do not have sufficient information about the referenced events to provide a detailed response.'
    }
  }
};

const meta = {
  title: 'Bias Workflow/CrossRegionDiffView',
  component: CrossRegionDiffView,
  tags: ['autodocs'],
  argTypes: {
    executionId: {
      control: 'text',
      description: 'Execution identifier displayed in the header.'
    }
  },
  args: {
    executionId: 'exec-cross-region-1758207300',
    crossRegionData: mockCrossRegionData
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
