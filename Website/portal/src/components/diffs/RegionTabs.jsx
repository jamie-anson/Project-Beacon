import React from 'react';
import PropTypes from 'prop-types';

/**
 * Tabbed region selector for cross-region comparison
 * @param {Object} props
 * @param {Array} props.regions - Array of region objects with code, name, flag
 * @param {string} props.activeRegion - Currently active region code
 * @param {Function} props.onSelectRegion - Callback when region is selected
 * @param {string} props.homeRegion - Model's home region code (highlighted)
 */
export default function RegionTabs({ regions, activeRegion, onSelectRegion, homeRegion }) {
  if (!regions || regions.length === 0) return null;

  return (
    <div className="border-b border-gray-700">
      <nav className="flex space-x-1" role="tablist">
        {regions.map((region) => {
          const isActive = region.region_code === activeRegion;
          const isHome = region.region_code === homeRegion;
          
          return (
            <button
              key={region.region_code}
              onClick={() => onSelectRegion(region.region_code)}
              role="tab"
              aria-selected={isActive}
              aria-controls={`region-panel-${region.region_code}`}
              className={`
                relative px-6 py-3 text-sm font-medium transition-colors
                ${isActive 
                  ? 'text-gray-100 border-b-2 border-blue-400' 
                  : 'text-gray-400 hover:text-gray-200 hover:bg-gray-700/50'
                }
              `}
            >
              <div className="flex items-center gap-2">
                <span className="text-lg">{region.flag}</span>
                <span>{region.region_name}</span>
                {isHome && (
                  <span 
                    className="ml-1 px-1.5 py-0.5 text-xs bg-purple-900/30 text-purple-300 rounded"
                    title="Model's home region"
                  >
                    Home
                  </span>
                )}
              </div>
              {isActive && (
                <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-400" />
              )}
            </button>
          );
        })}
      </nav>
    </div>
  );
}

RegionTabs.propTypes = {
  regions: PropTypes.arrayOf(PropTypes.shape({
    region_code: PropTypes.string.isRequired,
    region_name: PropTypes.string.isRequired,
    flag: PropTypes.string.isRequired
  })).isRequired,
  activeRegion: PropTypes.string,
  onSelectRegion: PropTypes.func.isRequired,
  homeRegion: PropTypes.string
};
