/**
 * ProgressHeader Component
 * Displays job stage indicator, countdown timer, and progress bar
 */

import React from 'react';
import PropTypes from 'prop-types';
import { Loader2 } from 'lucide-react';

export default function ProgressHeader({ 
  stage, 
  timeRemaining, 
  completed, 
  running,
  failed,
  pending,
  total, 
  percentage,
  showShimmer,
  overallCompleted,
  overallFailed,
  hasQuestions,
  specQuestions,
  displayQuestions,
  specModels,
  uniqueModels,
  selectedRegions
}) {
  return (
    <div className="space-y-3">
      {/* Stage indicator */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          {stage === 'creating' && (
            <>
              <div className="animate-spin h-4 w-4 border-2 border-cyan-500 border-t-transparent rounded-full" />
              <span className="text-sm text-cyan-400">Creating job...</span>
            </>
          )}
          {stage === 'queued' && (
            <>
              <div className="animate-pulse h-4 w-4 bg-yellow-500 rounded-full" />
              <span className="text-sm text-yellow-400">Job queued, waiting for worker...</span>
            </>
          )}
          {stage === 'spawning' && (
            <>
              <div className="animate-spin h-4 w-4 border-2 border-blue-500 border-t-transparent rounded-full" />
              <span className="text-sm text-blue-400">Starting executions...</span>
            </>
          )}
          {stage === 'running' && (
            <>
              <div className="relative h-4 w-4">
                <div className="absolute inset-0 animate-ping h-4 w-4 bg-green-500 rounded-full opacity-20" />
                <div className="relative h-4 w-4 bg-green-500 rounded-full" />
              </div>
              <span className="text-sm text-green-400">Executing questions...</span>
            </>
          )}
          {stage === 'completed' && (
            <>
              <div className="h-4 w-4 bg-green-500 rounded-full flex items-center justify-center">
                <svg className="w-3 h-3 text-white" fill="currentColor" viewBox="0 0 20 20">
                  <path 
                    fillRule="evenodd" 
                    d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" 
                    clipRule="evenodd" 
                  />
                </svg>
              </div>
              <span className="text-sm text-green-400 font-medium">Job completed successfully!</span>
            </>
          )}
          {stage === 'failed' && (
            <>
              <div className="h-4 w-4 bg-red-500 rounded-full flex items-center justify-center">
                <svg className="w-3 h-3 text-white" fill="currentColor" viewBox="0 0 20 20">
                  <path 
                    fillRule="evenodd" 
                    d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" 
                    clipRule="evenodd" 
                  />
                </svg>
              </div>
              <span className="text-sm text-red-400 font-medium">Job failed</span>
            </>
          )}
        </div>
        <span className="text-xs text-gray-400">
          {timeRemaining ? `Time remaining: ~${timeRemaining}` : `${completed}/${total} executions`}
        </span>
      </div>
      
      {/* Progress bar */}
      <div>
        <div className={`w-full h-3 bg-gray-700 rounded overflow-hidden relative ${showShimmer ? 'animate-pulse' : ''}`}>
          <div className="h-full flex relative z-10">
            <div 
              className="h-full bg-green-500" 
              style={{ width: `${(completed / total) * 100}%` }}
            ></div>
            <div 
              className="h-full bg-yellow-500" 
              style={{ width: `${(running / total) * 100}%` }}
            ></div>
            <div 
              className="h-full bg-red-500" 
              style={{ width: `${(failed / total) * 100}%` }}
            ></div>
          </div>
          {showShimmer && (
            <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/10 to-transparent animate-shimmer"></div>
          )}
        </div>
        <div className="flex items-center justify-between text-xs mt-1">
          {!overallCompleted && !overallFailed ? (
            <div className="flex items-center gap-2 text-gray-400">
              <Loader2 className="h-3 w-3 animate-spin text-green-400" />
              <span>Processing...</span>
            </div>
          ) : (
            <span className="text-gray-400">
              {hasQuestions 
                ? `${specQuestions.length || displayQuestions.length} questions × ${specModels.length || uniqueModels.length} models × ${selectedRegions.length} regions` 
                : `${selectedRegions.length} regions`}
            </span>
          )}
          <span className="text-gray-400 font-medium">{completed}/{total} executions</span>
        </div>
      </div>
    </div>
  );
}

ProgressHeader.propTypes = {
  stage: PropTypes.string.isRequired,
  timeRemaining: PropTypes.string,
  completed: PropTypes.number.isRequired,
  running: PropTypes.number.isRequired,
  failed: PropTypes.number.isRequired,
  pending: PropTypes.number.isRequired,
  total: PropTypes.number.isRequired,
  percentage: PropTypes.number.isRequired,
  showShimmer: PropTypes.bool.isRequired,
  overallCompleted: PropTypes.bool.isRequired,
  overallFailed: PropTypes.bool.isRequired,
  hasQuestions: PropTypes.bool.isRequired,
  specQuestions: PropTypes.array.isRequired,
  displayQuestions: PropTypes.array.isRequired,
  specModels: PropTypes.array.isRequired,
  uniqueModels: PropTypes.array.isRequired,
  selectedRegions: PropTypes.array.isRequired
};
