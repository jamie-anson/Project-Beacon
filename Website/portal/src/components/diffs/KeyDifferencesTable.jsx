import React from 'react';
import PropTypes from 'prop-types';

/**
 * Displays key narrative differences across regions for a single model
 * Uses backend analysis data (key_differences)
 * @param {Object} props
 * @param {Array} props.keyDifferences - Array of difference objects from backend
 * @param {Array} props.regions - Array of region objects
 */
export default function KeyDifferencesTable({ keyDifferences, regions }) {
  if (!keyDifferences?.length || !regions?.length) return null;

  return (
    <div className="bg-gray-800 border border-gray-700 rounded-lg">
      <div className="px-6 py-4 border-b border-gray-700">
        <h2 className="text-lg font-medium text-gray-100">
          📊 Key Narrative Differences
        </h2>
        <p className="text-sm text-gray-300 mt-1">
          Analysis of how responses vary across regions, revealing systematic differences in content and framing.
        </p>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead className="bg-gray-900">
            <tr className="text-xs text-gray-400 uppercase tracking-wide">
              <th className="px-6 py-3 text-left font-medium w-48">Dimension</th>
              {regions.map((region) => (
                <th key={region.region_code} className="px-6 py-3 text-left font-medium">
                  {region.flag} {region.region_name}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-700">
            {keyDifferences.map((diff, index) => (
              <tr key={index} className="hover:bg-gray-900/40 transition-colors">
                <td className="px-6 py-4 align-top">
                  <div className="flex items-start gap-2">
                    <SeverityIndicator severity={diff.severity} />
                    <div>
                      <div className="font-medium text-gray-100">
                        {diff.dimension_label}
                      </div>
                      {diff.description && (
                        <div className="text-xs text-gray-400 mt-1">
                          {diff.description}
                        </div>
                      )}
                    </div>
                  </div>
                </td>
                {regions.map((region) => {
                  const variation = diff.variations?.[region.region_code];
                  return (
                    <td 
                      key={`${diff.dimension}-${region.region_code}`} 
                      className="px-6 py-4 text-sm text-gray-300 align-top"
                    >
                      {variation || (
                        <span className="text-gray-500 italic">No data</span>
                      )}
                    </td>
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      
      {/* Legend */}
      <div className="px-6 py-3 border-t border-gray-700 bg-gray-900/50">
        <div className="flex items-center gap-6 text-xs text-gray-400">
          <span className="font-medium">Severity:</span>
          <div className="flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-red-500"></span>
            <span>High</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-yellow-500"></span>
            <span>Medium</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-green-500"></span>
            <span>Low</span>
          </div>
        </div>
      </div>
    </div>
  );
}

KeyDifferencesTable.propTypes = {
  keyDifferences: PropTypes.arrayOf(PropTypes.shape({
    dimension: PropTypes.string.isRequired,
    dimension_label: PropTypes.string.isRequired,
    variations: PropTypes.object.isRequired,
    severity: PropTypes.oneOf(['low', 'medium', 'high']),
    description: PropTypes.string
  })),
  regions: PropTypes.arrayOf(PropTypes.shape({
    region_code: PropTypes.string.isRequired,
    region_name: PropTypes.string.isRequired,
    flag: PropTypes.string.isRequired
  }))
};

function SeverityIndicator({ severity }) {
  const colors = {
    high: 'bg-red-500',
    medium: 'bg-yellow-500',
    low: 'bg-green-500'
  };

  const labels = {
    high: 'High severity',
    medium: 'Medium severity',
    low: 'Low severity'
  };

  return (
    <span 
      className={`w-2 h-2 rounded-full ${colors[severity] || colors.medium} mt-1.5 flex-shrink-0`}
      title={labels[severity] || 'Unknown severity'}
      aria-label={labels[severity] || 'Unknown severity'}
    />
  );
}

SeverityIndicator.propTypes = {
  severity: PropTypes.oneOf(['low', 'medium', 'high'])
};
