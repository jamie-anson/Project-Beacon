import React from 'react';
import { render, screen } from '@testing-library/react';
import DiffNarrativeTable from '../DiffNarrativeTable.jsx';

describe('DiffNarrativeTable', () => {
  const regions = [
    {
      region_code: 'US',
      region_name: 'United States',
      flag: '🇺🇸',
      bias_score: 30,
      censorship_level: 'low'
    },
    {
      region_code: 'EU',
      region_name: 'Europe',
      flag: '🇪🇺',
      bias_score: 75,
      censorship_level: 'high'
    }
  ];

  it('renders narrative rows for each region', () => {
    render(<DiffNarrativeTable modelName="Test Model" regions={regions} />);

    expect(screen.getByText('📊 Cross-Region Analysis: Test Model Narrative Differences')).toBeInTheDocument();
    expect(screen.getByText('🇺🇸 United States')).toBeInTheDocument();
    expect(screen.getByText('🇪🇺 Europe')).toBeInTheDocument();
    expect(screen.getByText(/Direct, factual/i)).toBeInTheDocument();
    expect(screen.getByText(/Heavy censorship/i)).toBeInTheDocument();
    expect(screen.getByText(/75% bias detected/i)).toBeInTheDocument();
  });

  it('returns null when no regions provided', () => {
    const { container } = render(<DiffNarrativeTable modelName="Test Model" regions={[]} />);
    expect(container.firstChild).toBeNull();
  });
});
