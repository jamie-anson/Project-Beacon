/**
 * ProgressBreakdown Component
 * Displays status breakdown and per-question progress
 */

import React from 'react';
import PropTypes from 'prop-types';
import { calculateQuestionProgress } from '../../../lib/utils/progressUtils';

export default function ProgressBreakdown({ 
  completed, 
  running, 
  failed, 
  pending,
  hasQuestions,
  displayQuestions,
  executions,
  specModels,
  selectedRegions,
  uniqueModels
}) {
  return (
    <div className="space-y-3">
      {/* Status breakdown */}
      <div className="flex items-center gap-4 text-xs">
        <div className="flex items-center gap-1">
          <div className="w-2 h-2 bg-green-500 rounded"></div>
          <span className="text-gray-300">Completed: {completed}</span>
        </div>
        <div className="flex items-center gap-1">
          <div className={`w-2 h-2 bg-yellow-500 rounded ${running > 0 ? 'animate-pulse' : ''}`}></div>
          <span className="text-gray-300">Running: {running}</span>
        </div>
        <div className="flex items-center gap-1">
          <div className="w-2 h-2 bg-red-500 rounded"></div>
          <span className="text-gray-300">Failed: {failed}</span>
        </div>
        <div className="flex items-center gap-1">
          <div className="w-2 h-2 bg-gray-500 rounded"></div>
          <span className="text-gray-300">Pending: {pending}</span>
        </div>
      </div>
      
      {/* Per-question breakdown (if applicable) */}
      {hasQuestions && displayQuestions.length > 0 && (
        <div className="bg-gray-800/50 border border-gray-600 rounded p-3 space-y-1">
          <div className="text-xs font-medium text-gray-300 mb-2">Question Progress</div>
          {displayQuestions.map(questionId => {
            const progress = calculateQuestionProgress(
              questionId, 
              executions, 
              specModels, 
              selectedRegions, 
              uniqueModels
            );
            
            return (
              <div key={questionId} className="flex items-center justify-between text-xs">
                <span className="text-gray-300 font-mono">{questionId}</span>
                <div className="flex items-center gap-2">
                  <span className="text-gray-400">{progress.completed}/{progress.expected}</span>
                  {progress.refused > 0 && (
                    <span className="px-2 py-0.5 bg-orange-900/20 text-orange-400 rounded-full text-xs">
                      {progress.refused} refusals
                    </span>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

ProgressBreakdown.propTypes = {
  completed: PropTypes.number.isRequired,
  running: PropTypes.number.isRequired,
  failed: PropTypes.number.isRequired,
  pending: PropTypes.number.isRequired,
  hasQuestions: PropTypes.bool.isRequired,
  displayQuestions: PropTypes.array.isRequired,
  executions: PropTypes.array.isRequired,
  specModels: PropTypes.array.isRequired,
  selectedRegions: PropTypes.array.isRequired,
  uniqueModels: PropTypes.array.isRequired
};
