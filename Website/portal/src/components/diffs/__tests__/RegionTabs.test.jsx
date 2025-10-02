import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import RegionTabs from '../RegionTabs.jsx';

const regions = [
  { region_code: 'US', region_name: 'United States', flag: '🇺🇸' },
  { region_code: 'EU', region_name: 'Europe', flag: '🇪🇺' },
  { region_code: 'ASIA', region_name: 'Asia Pacific', flag: '🌏' }
];

function setup(props = {}) {
  const onSelectRegion = jest.fn();
  render(
    <RegionTabs
      regions={regions}
      activeRegion={props.activeRegion ?? 'US'}
      onSelectRegion={onSelectRegion}
      homeRegion={props.homeRegion ?? 'ASIA'}
    />
  );
  return { onSelectRegion };
}

describe('RegionTabs', () => {
  test('renders all regions as tabs', () => {
    setup();
    expect(screen.getByRole('tab', { name: /United States/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /Europe/i })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /Asia Pacific/i })).toBeInTheDocument();
  });

  test('highlights active region', () => {
    setup({ activeRegion: 'EU' });
    const euTab = screen.getByRole('tab', { name: /Europe/i });
    expect(euTab).toHaveAttribute('aria-selected', 'true');
    const usTab = screen.getByRole('tab', { name: /United States/i });
    expect(usTab).toHaveAttribute('aria-selected', 'false');
  });

  test('shows Home badge on home region', () => {
    setup({ homeRegion: 'EU' });
    const euTab = screen.getByRole('tab', { name: /europe/i });
    expect(euTab).toHaveTextContent(/home/i);
  });

  test('calls onSelectRegion when a tab is clicked', () => {
    const { onSelectRegion } = setup({ activeRegion: 'US' });
    const asiaTab = screen.getByRole('tab', { name: /asia pacific/i });
    fireEvent.click(asiaTab);
    expect(onSelectRegion).toHaveBeenCalledWith('ASIA');
  });

  test('has proper roles for accessibility', () => {
    setup();
    expect(screen.getByRole('tablist')).toBeInTheDocument();
    expect(screen.getAllByRole('tab')).toHaveLength(3);
  });
});
