import React from 'react';
import PropTypes from 'prop-types';
import SimilarityGauge from './SimilarityGauge.jsx';
import ResponseLengthChart from './ResponseLengthChart.jsx';
import KeywordFrequencyTable from './KeywordFrequencyTable.jsx';

/**
 * Container for all visualization components
 * @param {Object} props
 * @param {Array} props.regions - Array of region data
 * @param {Object} props.metrics - Analysis metrics
 */
export default function VisualizationsSection({ regions, metrics }) {
  if (!regions || regions.length === 0) return null;

  return (
    <div className="bg-gray-800 border border-gray-700 rounded-lg">
      <div className="px-6 py-4 border-b border-gray-700">
        <h2 className="text-lg font-medium text-gray-100">
          📈 Analysis Visualizations
        </h2>
        <p className="text-sm text-gray-300 mt-1">
          Visual analysis of response patterns and regional differences
        </p>
      </div>

      <div className="p-6 space-y-8">
        {/* Similarity Gauge */}
        <SimilarityGauge regions={regions} metrics={metrics} />

        {/* Response Length Comparison */}
        <ResponseLengthChart regions={regions} />

        {/* Keyword Frequency */}
        <KeywordFrequencyTable regions={regions} />
      </div>
    </div>
  );
}

VisualizationsSection.propTypes = {
  regions: PropTypes.arrayOf(PropTypes.shape({
    region_code: PropTypes.string.isRequired,
    region_name: PropTypes.string.isRequired,
    flag: PropTypes.string.isRequired,
    response: PropTypes.string,
    response_length: PropTypes.number,
    keywords: PropTypes.arrayOf(PropTypes.string)
  })),
  metrics: PropTypes.object
};
