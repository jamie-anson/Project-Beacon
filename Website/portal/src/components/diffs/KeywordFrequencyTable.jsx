import React from 'react';
import PropTypes from 'prop-types';

/**
 * Table showing keyword frequency across regions
 * @param {Object} props
 * @param {Array} props.regions - Array of region data
 */
export default function KeywordFrequencyTable({ regions }) {
  if (!regions || regions.length === 0) return null;

  // Collect all unique keywords across all regions (memoized)
  const keywords = React.useMemo(() => {
    const allKeywords = new Set();
    regions.forEach(region => {
      region.keywords?.forEach(keyword => allKeywords.add(keyword));
    });
    return Array.from(allKeywords).sort();
  }, [regions]);

  if (keywords.length === 0) {
    return (
      <div>
        <h3 className="text-sm font-medium text-gray-300 mb-4">Keyword Analysis</h3>
        <div className="text-sm text-gray-400">No keywords detected in responses</div>
      </div>
    );
  }

  // Count keyword occurrences in each region's response (memoized)
  const countKeywordOccurrences = React.useCallback((text, keyword) => {
    if (!text) return 0;
    const regex = new RegExp(keyword, 'gi');
    const matches = text.match(regex);
    return matches ? matches.length : 0;
  }, []);

  // Get color for keyword category (memoized)
  const getKeywordColor = React.useCallback((keyword) => {
    const colors = {
      censorship: 'text-red-400',
      violence: 'text-orange-400',
      democracy: 'text-blue-400',
      government: 'text-purple-400',
      neutral: 'text-gray-400'
    };
    return colors[keyword] || 'text-gray-400';
  }, []);

  return (
    <div>
      <h3 className="text-sm font-medium text-gray-300 mb-4">Keyword Frequency Analysis</h3>
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead className="bg-gray-900">
            <tr className="text-xs text-gray-400 uppercase tracking-wide">
              <th className="px-4 py-3 text-left font-medium">Keyword</th>
              {regions.map((region) => (
                <th key={region.region_code} className="px-4 py-3 text-center font-medium">
                  {region.flag}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-700">
            {keywords.map((keyword) => {
              const counts = regions.map(region => 
                countKeywordOccurrences(region.response, keyword)
              );
              const maxCount = Math.max(...counts);
              
              return (
                <tr key={keyword} className="hover:bg-gray-900/40">
                  <td className={`px-4 py-3 font-medium ${getKeywordColor(keyword)}`}>
                    {keyword}
                  </td>
                  {regions.map((region, index) => {
                    const count = counts[index];
                    const isMax = count === maxCount && count > 0;
                    
                    return (
                      <td 
                        key={region.region_code} 
                        className="px-4 py-3 text-center"
                      >
                        {count > 0 ? (
                          <span className={`inline-flex items-center justify-center min-w-[2rem] px-2 py-1 rounded ${
                            isMax 
                              ? 'bg-blue-900/30 text-blue-300 font-semibold' 
                              : 'text-gray-300'
                          }`}>
                            {count}×
                          </span>
                        ) : (
                          <span className="text-gray-600">—</span>
                        )}
                      </td>
                    );
                  })}
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
      <div className="mt-4 flex items-center gap-6 text-xs text-gray-400">
        <span className="font-medium">Legend:</span>
        <span className="text-red-400">● Censorship</span>
        <span className="text-orange-400">● Violence</span>
        <span className="text-blue-400">● Democracy</span>
        <span className="text-purple-400">● Government</span>
      </div>
    </div>
  );
}

KeywordFrequencyTable.propTypes = {
  regions: PropTypes.arrayOf(PropTypes.shape({
    region_code: PropTypes.string.isRequired,
    region_name: PropTypes.string.isRequired,
    flag: PropTypes.string.isRequired,
    response: PropTypes.string,
    keywords: PropTypes.arrayOf(PropTypes.string)
  }))
};
