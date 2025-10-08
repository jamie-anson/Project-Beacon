/**
 * ProgressActions Component
 * Action buttons for refreshing and viewing results
 */

import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';

export default function ProgressActions({ 
  jobId, 
  isCompleted,
  isFailed,
  onRefresh,
  onRetryJob
}) {
  const actionsDisabled = !isCompleted;
  const showDiffCta = !!jobId;

  return (
    <div className="flex items-center justify-end gap-2">
      {isFailed && onRetryJob && (
        <button 
          onClick={onRetryJob} 
          className="px-3 py-1.5 bg-yellow-600 text-white rounded text-sm hover:bg-yellow-700 flex items-center gap-2"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          Retry Job
        </button>
      )}
      
      <button 
        onClick={onRefresh} 
        className="px-3 py-1.5 bg-green-600 text-white rounded text-sm hover:bg-green-700"
      >
        Refresh
      </button>
      
      {showDiffCta && (
        actionsDisabled ? (
          <button
            disabled
            className="px-3 py-1.5 bg-beacon-600 text-white rounded text-sm opacity-50 cursor-not-allowed"
            title="Available when job completes"
          >
            View Cross-Region Diffs
          </button>
        ) : (
          <Link 
            to={`/results/${jobId}/diffs`}
            className="px-3 py-1.5 bg-beacon-600 text-white rounded text-sm hover:bg-beacon-700"
          >
            View Cross-Region Diffs
          </Link>
        )
      )}
      
      {jobId && (
        <Link 
          to={`/jobs/${jobId}`} 
          className="text-sm text-beacon-600 underline decoration-dotted"
        >
          View full results
        </Link>
      )}
    </div>
  );
}

ProgressActions.propTypes = {
  jobId: PropTypes.string,
  isCompleted: PropTypes.bool.isRequired,
  isFailed: PropTypes.bool,
  onRefresh: PropTypes.func.isRequired,
  onRetryJob: PropTypes.func
};
