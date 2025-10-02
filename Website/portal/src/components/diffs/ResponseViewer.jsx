import React from 'react';
import PropTypes from 'prop-types';
import WordLevelDiff from './WordLevelDiff.jsx';

/**
 * Response viewer with optional diff highlighting
 * @param {Object} props
 * @param {Object} props.currentRegion - Current region data
 * @param {Array} props.allRegions - All region data for comparison
 * @param {string} props.compareRegion - Region code to compare against
 * @param {Function} props.onChangeCompareRegion - Callback when compare region changes
 * @param {string} props.modelName - Model name for display
 */
export default function ResponseViewer({ 
  currentRegion, 
  allRegions, 
  compareRegion, 
  onChangeCompareRegion,
  modelName 
}) {
  const [diffEnabled, setDiffEnabled] = React.useState(false);

  if (!currentRegion) {
    return (
      <div className="bg-gray-800 border border-gray-700 rounded-lg p-6">
        <div className="text-center text-gray-400">
          No region selected
        </div>
      </div>
    );
  }

  const compareRegionData = allRegions.find(r => r.region_code === compareRegion);
  const availableCompareRegions = allRegions.filter(r => r.region_code !== currentRegion.region_code);

  return (
    <div 
      className="bg-gray-800 border border-gray-700 rounded-lg"
      role="tabpanel"
      id={`region-panel-${currentRegion.region_code}`}
      aria-labelledby={`region-tab-${currentRegion.region_code}`}
    >
      {/* Header with region info */}
      <div className="px-6 py-4 border-b border-gray-700">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-lg font-medium text-gray-100">
            Generated Response
          </h3>
          <div className="flex items-center gap-4 text-sm">
            <div className="text-gray-400">
              {currentRegion.response_length} characters
            </div>
            <div className={`px-2 py-1 rounded text-xs font-medium ${
              currentRegion.censorship_detected
                ? 'bg-red-900/30 text-red-300'
                : 'bg-green-900/30 text-green-300'
            }`}>
              {currentRegion.censorship_detected ? 'Censored' : 'Uncensored'}
            </div>
          </div>
        </div>

        {/* Diff toggle and compare region selector */}
        <div className="flex items-center gap-4">
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={diffEnabled}
              onChange={(e) => setDiffEnabled(e.target.checked)}
              className="w-4 h-4 rounded border-gray-600 bg-gray-700 text-blue-500 focus:ring-blue-500 focus:ring-offset-gray-800"
            />
            <span className="text-sm text-gray-300">
              Word-level diff highlighting (additions/deletions/changes)
            </span>
          </label>

          {diffEnabled && availableCompareRegions.length > 0 && (
            <div className="flex items-center gap-2 ml-auto">
              <span className="text-sm text-gray-400">Compare with:</span>
              <select
                value={compareRegion}
                onChange={(e) => onChangeCompareRegion(e.target.value)}
                className="bg-gray-700 border border-gray-600 rounded px-3 py-1 text-sm text-gray-200 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              >
                {availableCompareRegions.map((region) => (
                  <option key={region.region_code} value={region.region_code}>
                    {region.flag} {region.region_name}
                  </option>
                ))}
              </select>
            </div>
          )}
        </div>
      </div>

      {/* Response content */}
      <div className="p-6">
        {diffEnabled && compareRegionData ? (
          <div className="space-y-4">
            <div className="text-sm text-gray-400 mb-2">
              Showing differences between{' '}
              <span className="font-medium text-gray-200">
                {currentRegion.flag} {currentRegion.region_name}
              </span>
              {' '}and{' '}
              <span className="font-medium text-gray-200">
                {compareRegionData.flag} {compareRegionData.region_name}
              </span>
            </div>
            <WordLevelDiff
              baseText={currentRegion.response}
              comparisonText={compareRegionData.response}
            />
          </div>
        ) : (
          <div className="bg-gray-900 rounded-lg p-4 border-l-4 border-blue-500">
            <p className="text-sm text-gray-200 leading-relaxed whitespace-pre-wrap max-w-prose">
              {currentRegion.response}
            </p>
          </div>
        )}
      </div>

      {/* Region metrics footer */}
      <div className="px-6 py-4 border-t border-gray-700 bg-gray-900/50">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
          <div>
            <div className="text-gray-400 text-xs mb-1">Bias Score</div>
            <div className={`font-semibold ${
              currentRegion.bias_score < 30 ? 'text-green-400' :
              currentRegion.bias_score < 70 ? 'text-yellow-400' :
              'text-red-400'
            }`}>
              {currentRegion.bias_score}%
            </div>
          </div>
          <div>
            <div className="text-gray-400 text-xs mb-1">Factual Accuracy</div>
            <div className={`font-semibold ${
              currentRegion.factual_accuracy > 80 ? 'text-green-400' :
              currentRegion.factual_accuracy > 60 ? 'text-yellow-400' :
              'text-red-400'
            }`}>
              {currentRegion.factual_accuracy}%
            </div>
          </div>
          <div>
            <div className="text-gray-400 text-xs mb-1">Political Sensitivity</div>
            <div className="font-semibold text-gray-200">
              {currentRegion.political_sensitivity}%
            </div>
          </div>
          <div>
            <div className="text-gray-400 text-xs mb-1">Provider</div>
            <div className="font-mono text-xs text-gray-300">
              {currentRegion.provider_id}
            </div>
          </div>
        </div>

        {/* Keywords */}
        {currentRegion.keywords && currentRegion.keywords.length > 0 && (
          <div className="mt-3 pt-3 border-t border-gray-700">
            <div className="text-gray-400 text-xs mb-2">Keywords Detected:</div>
            <div className="flex flex-wrap gap-2">
              {currentRegion.keywords.map((keyword, index) => (
                <span
                  key={`${currentRegion.region_code}-keyword-${index}`}
                  className={`px-2 py-1 rounded-full text-xs ${
                    keyword === 'censorship' ? 'bg-red-900/30 text-red-300' :
                    keyword === 'violence' ? 'bg-orange-900/30 text-orange-300' :
                    keyword === 'democracy' ? 'bg-blue-900/30 text-blue-300' :
                    'bg-gray-700 text-gray-300'
                  }`}
                >
                  {keyword}
                </span>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

ResponseViewer.propTypes = {
  currentRegion: PropTypes.shape({
    region_code: PropTypes.string.isRequired,
    region_name: PropTypes.string.isRequired,
    flag: PropTypes.string.isRequired,
    response: PropTypes.string.isRequired,
    response_length: PropTypes.number.isRequired,
    censorship_detected: PropTypes.bool,
    bias_score: PropTypes.number,
    factual_accuracy: PropTypes.number,
    political_sensitivity: PropTypes.number,
    provider_id: PropTypes.string,
    keywords: PropTypes.arrayOf(PropTypes.string)
  }),
  allRegions: PropTypes.array.isRequired,
  compareRegion: PropTypes.string,
  onChangeCompareRegion: PropTypes.func.isRequired,
  modelName: PropTypes.string
};
