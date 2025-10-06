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
  onRefresh 
}) {
  const actionsDisabled = !isCompleted;
  const showDiffCta = !!jobId;

  return (
    <div className="flex items-center justify-end gap-2">
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
  onRefresh: PropTypes.func.isRequired
};
