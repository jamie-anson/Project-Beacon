import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import ResponseViewer from '../ResponseViewer.jsx';

const regions = [
  { region_code: 'US', region_name: 'United States', flag: '🇺🇸', response: 'Hello from US', response_length: 12, censorship_detected: false, bias_score: 10, factual_accuracy: 90, political_sensitivity: 60, provider_id: 'modal-us-001', keywords: ['democracy'] },
  { region_code: 'EU', region_name: 'Europe', flag: '🇪🇺', response: 'Hello from EU', response_length: 12, censorship_detected: false, bias_score: 20, factual_accuracy: 88, political_sensitivity: 62, provider_id: 'modal-eu-001', keywords: [] },
  { region_code: 'ASIA', region_name: 'Asia Pacific', flag: '🌏', response: 'Hi', response_length: 2, censorship_detected: true, bias_score: 70, factual_accuracy: 50, political_sensitivity: 90, provider_id: 'modal-asia-001', keywords: ['censorship'] },
];

function setup(props = {}) {
  const onChangeCompareRegion = jest.fn();
  render(
    <ResponseViewer
      currentRegion={props.currentRegion ?? regions[0]}
      allRegions={regions}
      compareRegion={props.compareRegion ?? 'EU'}
      onChangeCompareRegion={onChangeCompareRegion}
      modelName="Test Model"
    />
  );
  return { onChangeCompareRegion };
}

describe('ResponseViewer', () => {
  test('renders response text', () => {
    setup();
    expect(screen.getByText(/Hello from US/i)).toBeInTheDocument();
  });

  test('shows diff toggle checkbox', () => {
    setup();
    expect(screen.getByRole('checkbox')).toBeInTheDocument();
  });

  test('when diff enabled, shows compare selector and legend labels render', () => {
    setup();
    const checkbox = screen.getByRole('checkbox');
    fireEvent.click(checkbox);
    expect(screen.getByText(/Compare with/i)).toBeInTheDocument();
    // After enabling diff, selector should include EU option
    expect(screen.getByRole('combobox')).toBeInTheDocument();
  });

  test('changing compare selector calls handler', () => {
    const { onChangeCompareRegion } = setup({ compareRegion: 'EU' });
    const checkbox = screen.getByRole('checkbox');
    fireEvent.click(checkbox);
    const select = screen.getByRole('combobox');
    fireEvent.change(select, { target: { value: 'ASIA' } });
    expect(onChangeCompareRegion).toHaveBeenCalledWith('ASIA');
  });

  test('handles missing currentRegion gracefully', () => {
    render(
      <ResponseViewer
        currentRegion={null}
        allRegions={regions}
        compareRegion={'EU'}
        onChangeCompareRegion={() => {}}
        modelName="Test"
      />
    );
    expect(screen.getByText(/No region selected/i)).toBeInTheDocument();
  });
});
