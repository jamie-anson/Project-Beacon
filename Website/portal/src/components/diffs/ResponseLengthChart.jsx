import React from 'react';
import PropTypes from 'prop-types';

/**
 * Horizontal bar chart showing response length comparison
 * @param {Object} props
 * @param {Array} props.regions - Array of region data
 */
export default function ResponseLengthChart({ regions }) {
  if (!regions || regions.length === 0) return null;

  // Find max length for scaling (memoized)
  const maxLength = React.useMemo(() => 
    Math.max(...regions.map(r => r.response_length)),
    [regions]
  );

  return (
    <div>
      <h3 className="text-sm font-medium text-gray-300 mb-4">Response Length Comparison</h3>
      <div className="space-y-4">
        {regions.map((region) => {
          const percentage = (region.response_length / maxLength) * 100;
          
          // Color based on whether this is significantly shorter (potential censorship indicator)
          const isShort = region.response_length < maxLength * 0.6;
          const barColor = isShort ? 'bg-red-500' : 'bg-blue-500';
          
          return (
            <div key={region.region_code}>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-gray-300">
                  {region.flag} {region.region_name}
                </span>
                <span className="text-sm font-medium text-gray-200">
                  {region.response_length.toLocaleString()} chars
                </span>
              </div>
              <div className="relative w-full h-8 bg-gray-700 rounded-lg overflow-hidden">
                <div
                  className={`h-full ${barColor} transition-all duration-1000 ease-out flex items-center justify-end px-3`}
                  style={{ width: `${percentage}%` }}
                >
                  {percentage > 15 && (
                    <span className="text-xs font-medium text-white">
                      {Math.round(percentage)}%
                    </span>
                  )}
                </div>
              </div>
              {isShort && (
                <div className="mt-1 text-xs text-red-400">
                  ⚠️ Significantly shorter than other regions
                </div>
              )}
            </div>
          );
        })}
      </div>
      <div className="mt-4 text-xs text-gray-400">
        Shorter responses may indicate censorship or information filtering
      </div>
    </div>
  );
}

ResponseLengthChart.propTypes = {
  regions: PropTypes.arrayOf(PropTypes.shape({
    region_code: PropTypes.string.isRequired,
    region_name: PropTypes.string.isRequired,
    flag: PropTypes.string.isRequired,
    response_length: PropTypes.number.isRequired
  }))
};
