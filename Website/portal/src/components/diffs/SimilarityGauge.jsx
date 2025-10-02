import React from 'react';
import PropTypes from 'prop-types';

/**
 * Circular gauge showing response similarity across regions
 * @param {Object} props
 * @param {Array} props.regions - Array of region data
 * @param {Object} props.metrics - Analysis metrics
 */
export default function SimilarityGauge({ regions, metrics }) {
  if (!regions || regions.length === 0) return null;

  // Calculate average bias score (inverse of similarity) - memoized
  const similarity = React.useMemo(() => {
    const avgBiasScore = regions.reduce((sum, r) => sum + (r.bias_score || 0), 0) / regions.length;
    return Math.max(0, 100 - avgBiasScore);
  }, [regions]);

  // Determine color based on similarity (memoized)
  const { color, label } = React.useMemo(() => {
    if (similarity > 80) return { color: '#10b981', label: 'High Similarity' };
    if (similarity > 50) return { color: '#f59e0b', label: 'Moderate Similarity' };
    return { color: '#ef4444', label: 'Low Similarity' };
  }, [similarity]);
  
  // Calculate SVG circle properties for progress ring
  const radius = 70;
  const circumference = 2 * Math.PI * radius;
  const progress = (similarity / 100) * circumference;
  const dashOffset = circumference - progress;

  return (
    <div>
      <h3 className="text-sm font-medium text-gray-300 mb-4">Response Similarity</h3>
      <div className="flex items-center gap-8">
        {/* Circular gauge */}
        <div className="relative flex-shrink-0">
          <svg width="180" height="180" className="transform -rotate-90">
            {/* Background circle */}
            <circle
              cx="90"
              cy="90"
              r={radius}
              fill="none"
              stroke="currentColor"
              strokeWidth="12"
              className="text-gray-700"
            />
            {/* Progress circle */}
            <circle
              cx="90"
              cy="90"
              r={radius}
              fill="none"
              stroke="currentColor"
              strokeWidth="12"
              strokeDasharray={circumference}
              strokeDashoffset={dashOffset}
              strokeLinecap="round"
              className={`stroke-${color}`}
              style={{ transition: 'stroke-dashoffset 1s ease-in-out' }}
            />
          </svg>
          {/* Center text */}
          <div className="absolute inset-0 flex flex-col items-center justify-center">
            <div className={`text-4xl font-bold text-${color}`}>
              {Math.round(similarity)}%
            </div>
            <div className="text-xs text-gray-400 mt-1">{label}</div>
          </div>
        </div>

        {/* Description */}
        <div className="flex-1">
          <div className="space-y-3">
            <div>
              <div className="text-sm font-medium text-gray-200 mb-1">
                {label}
              </div>
              <div className="text-sm text-gray-400">
                {similarity >= 80 
                  ? 'Responses are highly consistent across regions with minimal variation.'
                  : similarity >= 50
                  ? 'Responses show moderate differences across regions.'
                  : 'Responses vary significantly across regions, indicating potential bias or censorship.'}
              </div>
            </div>

            {/* Regional breakdown */}
            <div className="space-y-2">
              {regions.map((region) => (
                <div key={region.region_code} className="flex items-center justify-between text-sm">
                  <span className="text-gray-300">
                    {region.flag} {region.region_name}
                  </span>
                  <span className={`font-medium ${
                    region.bias_score < 30 ? 'text-green-400' :
                    region.bias_score < 70 ? 'text-yellow-400' :
                    'text-red-400'
                  }`}>
                    {region.bias_score}% bias
                  </span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

SimilarityGauge.propTypes = {
  regions: PropTypes.arrayOf(PropTypes.shape({
    region_code: PropTypes.string.isRequired,
    region_name: PropTypes.string.isRequired,
    flag: PropTypes.string.isRequired,
    bias_score: PropTypes.number
  }))
};
