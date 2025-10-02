import React from 'react';
import PropTypes from 'prop-types';

/**
 * Subtle metric card with right-aligned severity dot
 * Matches diff page aesthetics (no loud backgrounds)
 */
export default function MetricCard({ title, value, description, severity = 'low', inverted = false }) {
  // For inverted metrics (like accuracy), flip the severity colors
  const effectiveSeverity = inverted 
    ? (severity === 'high' ? 'low' : severity === 'low' ? 'high' : severity)
    : severity;

  const dotColors = {
    high: 'bg-red-500',
    medium: 'bg-yellow-500',
    low: 'bg-green-500'
  };

  return (
    <div className="border border-gray-700 rounded-lg p-4">
      <div className="text-2xl font-bold text-gray-100 mb-2">{value}</div>
      <div className="flex items-center justify-between gap-2 mb-1">
        <div className="text-sm font-medium text-gray-200">{title}</div>
        <span 
          className={`w-2 h-2 rounded-full ${dotColors[effectiveSeverity]} flex-shrink-0`}
          title={`${effectiveSeverity} severity`}
          aria-label={`${effectiveSeverity} severity`}
        />
      </div>
      <div className="text-xs text-gray-400">{description}</div>
    </div>
  );
}

MetricCard.propTypes = {
  title: PropTypes.string.isRequired,
  value: PropTypes.string.isRequired,
  description: PropTypes.string,
  severity: PropTypes.oneOf(['low', 'medium', 'high']),
  inverted: PropTypes.bool
};
