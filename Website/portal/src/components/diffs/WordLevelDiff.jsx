import React from 'react';
import PropTypes from 'prop-types';
import * as Diff from 'diff';

/**
 * Word-level diff highlighting component
 * Shows additions in green, deletions in red with strikethrough
 * @param {Object} props
 * @param {string} props.baseText - Base text (current region)
 * @param {string} props.comparisonText - Text to compare against
 */
export default function WordLevelDiff({ baseText, comparisonText }) {
  if (!baseText || !comparisonText) {
    return (
      <div className="text-gray-400 text-sm">
        No comparison text available
      </div>
    );
  }

  // Calculate word-level differences
  const changes = Diff.diffWords(comparisonText, baseText);

  return (
    <div className="bg-gray-900 rounded-lg p-4 border-l-4 border-purple-500">
      <div className="text-sm text-gray-200 leading-relaxed space-y-1 max-w-prose">
        {changes.map((part, index) => {
          if (part.added) {
            // Text present in base but not in comparison (added)
            return (
              <span
                key={index}
                className="bg-green-900/40 text-green-200 px-1 rounded"
                title="Added in this region"
              >
                {part.value}
              </span>
            );
          }
          
          if (part.removed) {
            // Text present in comparison but not in base (removed/deleted)
            return (
              <span
                key={index}
                className="bg-red-900/40 text-red-200 line-through px-1 rounded opacity-75"
                title="Removed from this region"
              >
                {part.value}
              </span>
            );
          }
          
          // Unchanged text
          return (
            <span key={index} className="text-gray-200">
              {part.value}
            </span>
          );
        })}
      </div>

      {/* Legend */}
      <div className="mt-4 pt-3 border-t border-gray-700 flex items-center gap-4 text-xs">
        <div className="flex items-center gap-2">
          <span className="bg-green-900/40 text-green-200 px-2 py-1 rounded">
            Added
          </span>
          <span className="text-gray-400">
            Text present in this region
          </span>
        </div>
        <div className="flex items-center gap-2">
          <span className="bg-red-900/40 text-red-200 line-through px-2 py-1 rounded opacity-75">
            Removed
          </span>
          <span className="text-gray-400">
            Text from comparison region
          </span>
        </div>
      </div>
    </div>
  );
}

WordLevelDiff.propTypes = {
  baseText: PropTypes.string,
  comparisonText: PropTypes.string
};
