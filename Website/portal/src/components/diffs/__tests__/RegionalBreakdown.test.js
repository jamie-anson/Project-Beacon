import React from 'react';
import { render, screen } from '@testing-library/react';
import RegionalBreakdown from '../RegionalBreakdown.jsx';

describe('RegionalBreakdown', () => {
  const mockRegions = [
    {
      region_code: 'US',
      region_name: 'United States',
      flag: '🇺🇸',
      status: 'completed',
      provider_id: 'provider-us',
      bias_score: 45,
      censorship_level: 'low',
      response: 'Detailed response for the US region.',
      factual_accuracy: 82,
      political_sensitivity: 37,
      keywords: ['democracy', 'freedom']
    },
    {
      region_code: 'EU',
      region_name: 'Europe',
      flag: '🇪🇺',
      status: 'completed',
      provider_id: 'provider-eu',
      bias_score: 72,
      censorship_level: 'high',
      response: 'Detailed response for the EU region.',
      factual_accuracy: 68,
      political_sensitivity: 54,
      keywords: ['sanctions', 'intervention']
    }
  ];

  it('renders cards for each region with key metrics', () => {
    render(<RegionalBreakdown modelName="Test Model" regions={mockRegions} />);

    expect(screen.getByText('🇺🇸 United States')).toBeInTheDocument();
    expect(screen.getByText('🇪🇺 Europe')).toBeInTheDocument();
    expect(screen.getAllByText(/Bias:/i)).toHaveLength(2);
    expect(screen.getByText(/Detailed response for the US region./i)).toBeInTheDocument();
    expect(screen.getByText(/Factual Accuracy/i)).toBeInTheDocument();
  });

  it('returns null when no regions provided', () => {
    const { container } = render(<RegionalBreakdown modelName="Test Model" regions={[]} />);
    expect(container.firstChild).toBeNull();
  });
});
