/**
 * RegionRow Component
 * Displays summary row for a region with expand/collapse functionality
 */

import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import { getStatusColor, getEnhancedStatus } from '../../../lib/utils/jobStatusUtils';
import { mapRegionToDatabase } from '../../../lib/utils/regionUtils';
import { calculateRegionProgress } from '../../../lib/utils/progressUtils';
import { timeAgo, getFailureDetails } from '../../../lib/utils/executionUtils';

export default function RegionRow({ 
  region, 
  regionExecs,
  isExpanded,
  onToggle,
  jobCompleted,
  jobFailed,
  jobStuckTimeout,
  loadingActive,
  uniqueModels,
  hasQuestions,
  jobId,
  failureInfo,
  jobAge,
  statusStr
}) {
  const e = regionExecs[0]; // Primary execution for basic info
  const progress = calculateRegionProgress(regionExecs);
  
  // Calculate multi-model status
  let status;
  if (jobCompleted) {
    status = 'completed';
  } else if (jobFailed || jobStuckTimeout) {
    status = 'failed';
  } else if (progress.total === 0) {
    status = 'pending';
  } else if (progress.completed === progress.total) {
    status = 'completed';
  } else if (progress.failed === progress.total) {
    status = 'failed';
  } else if (progress.running > 0) {
    status = 'running';
  } else {
    status = e?.status || e?.state || 'pending';
  }
  
  const started = e?.started_at || e?.created_at;
  let provider = e?.provider_id || e?.provider;
  
  // For completed jobs without execution records, show a default provider
  if (jobCompleted && !e) {
    provider = 'completed';
  } else if (progress.total > 1) {
    // For multi-model, show model count
    provider = `${progress.total} models`;
  }

  const failureDetails = getFailureDetails(e);
  const enhancedStatus = getEnhancedStatus(e, { status: statusStr }, loadingActive, jobCompleted, jobFailed, jobStuckTimeout);

  return (
    <div className="grid grid-cols-7 text-sm border-t border-gray-600 hover:bg-gray-700 cursor-pointer" onClick={onToggle}>
      {/* Region */}
      <div className="px-3 py-2 font-medium flex items-center gap-2">
        <span>{region}</span>
        {hasQuestions && regionExecs.length > 0 && (
          <svg className={`w-3 h-3 text-gray-400 transition-transform ${isExpanded ? 'rotate-180' : ''}`} fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clipRule="evenodd" />
          </svg>
        )}
      </div>
      
      {/* Progress */}
      <div className="px-3 py-2">
        {regionExecs.length > 0 ? (
          <div className="flex items-center gap-2">
            <span className="text-xs">{progress.completed}/{regionExecs.length}</span>
            <div className="flex-1 h-2 bg-gray-700 rounded overflow-hidden min-w-[40px]">
              <div className="h-full bg-green-500" style={{ width: `${progress.percentage}%` }} />
            </div>
          </div>
        ) : (
          <span className="text-xs text-gray-500">—</span>
        )}
      </div>
      
      {/* Status */}
      <div className="px-3 py-2">
        <div className="flex flex-col gap-1">
          <span className={`text-xs px-2 py-0.5 rounded-full border ${getStatusColor(enhancedStatus)}`}>
            {String(enhancedStatus)}
          </span>
          {/* Show multi-model progress */}
          {progress.total > 1 && (
            <div className="text-xs text-gray-400">
              {progress.completed}/{progress.total} models
            </div>
          )}
          {/* Show job-level failure messages */}
          {(jobFailed || jobStuckTimeout) && (
            <div className="flex flex-col gap-0.5 text-red-500 text-xs">
              <span className="truncate" title={failureInfo?.message}>
                {jobFailed ? `Job failed: ${statusStr}` : 'Job timeout'}
              </span>
              <span className="text-red-400/80 uppercase tracking-wide">
                {jobStuckTimeout ? `${Math.round(jobAge)}min stuck` : 'system failure'}
              </span>
            </div>
          )}
          {/* Show execution-level failure messages */}
          {!jobFailed && !jobStuckTimeout && failureDetails?.message && (
            <div className="flex flex-col gap-0.5 text-red-500 text-xs">
              <span className="truncate" title={failureDetails.message}>
                {(failureDetails.message || '').slice(0, 60)}{(failureDetails.message || '').length > 60 ? '…' : ''}
              </span>
              {(failureDetails.code || failureDetails.stage) && (
                <span className="text-red-400/80 uppercase tracking-wide" title={`${failureDetails.code || ''} ${failureDetails.stage || ''}`.trim()}>
                  {failureDetails.code || ''}{failureDetails.code && failureDetails.stage ? ' · ' : ''}{failureDetails.stage || ''}
                </span>
              )}
            </div>
          )}
        </div>
      </div>
      
      {/* Models */}
      <div className="px-3 py-2 text-xs text-gray-400">
        {uniqueModels.length > 0 ? `${uniqueModels.length} models` : '—'}
      </div>
      
      {/* Questions */}
      <div className="px-3 py-2 text-xs text-gray-400">
        {hasQuestions ? `${regionExecs.length} questions` : '—'}
      </div>
      
      {/* Started */}
      <div className="px-3 py-2 text-xs" title={started ? new Date(started).toLocaleString() : ''}>
        {started ? timeAgo(started) : '—'}
      </div>
      
      {/* Actions */}
      <div className="px-3 py-2" onClick={(e) => e.stopPropagation()}>
        {regionExecs.length > 0 ? (
          <Link
            to={`/executions?job=${encodeURIComponent(jobId || '')}&region=${encodeURIComponent(mapRegionToDatabase(region))}`}
            className="text-xs text-beacon-600 underline decoration-dotted hover:text-beacon-500"
          >
            View
          </Link>
        ) : (
          <span className="text-xs text-gray-500">—</span>
        )}
      </div>
    </div>
  );
}

RegionRow.propTypes = {
  region: PropTypes.string.isRequired,
  regionExecs: PropTypes.array.isRequired,
  isExpanded: PropTypes.bool.isRequired,
  onToggle: PropTypes.func.isRequired,
  jobCompleted: PropTypes.bool.isRequired,
  jobFailed: PropTypes.bool.isRequired,
  jobStuckTimeout: PropTypes.bool.isRequired,
  loadingActive: PropTypes.bool.isRequired,
  uniqueModels: PropTypes.array.isRequired,
  hasQuestions: PropTypes.bool.isRequired,
  jobId: PropTypes.string,
  failureInfo: PropTypes.object,
  jobAge: PropTypes.number.isRequired,
  statusStr: PropTypes.string.isRequired
};
