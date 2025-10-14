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
  isCancelled,
  onRefresh,
  onRetryJob,
  onCancelJob,
  isCancelling
}) {
  const actionsDisabled = !isCompleted;
  const showDiffCta = !!jobId;
  
  // Show cancel button only for active jobs (not completed, failed, or cancelled)
  const showCancelButton = jobId && !isCompleted && !isFailed && !isCancelled;

  return (
    <div className="flex items-center justify-end gap-2">
      {/* Cancel Button - Only button shown for active jobs */}
      {showCancelButton && onCancelJob && (
        <button 
          onClick={() => onCancelJob(jobId)} 
          disabled={isCancelling}
          className={`px-3 py-1.5 rounded text-sm flex items-center gap-2 transition-colors ${
            isCancelling 
              ? 'bg-gray-600 text-gray-400 cursor-not-allowed'
              : 'bg-red-600 text-white hover:bg-red-700'
          }`}
        >
          {isCancelling ? (
            <>
              <svg className="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"/>
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"/>
              </svg>
              Cancelling...
            </>
          ) : (
            <>
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
              Cancel Job
            </>
          )}
        </button>
      )}
    </div>
  );
}

ProgressActions.propTypes = {
  jobId: PropTypes.string,
  isCompleted: PropTypes.bool.isRequired,
  isFailed: PropTypes.bool,
  isCancelled: PropTypes.bool,
  onRefresh: PropTypes.func.isRequired,
  onRetryJob: PropTypes.func,
  onCancelJob: PropTypes.func,
  isCancelling: PropTypes.bool,
};
