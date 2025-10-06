/**
 * FailureAlert Component
 * Displays prominent failure alerts with contextual messaging
 */

import React from 'react';
import PropTypes from 'prop-types';

export default function FailureAlert({ failureInfo }) {
  if (!failureInfo) return null;

  return (
    <div className="bg-red-900/20 border border-red-700 rounded-lg p-4">
      <div className="flex items-start gap-3">
        <div className="flex-shrink-0">
          <svg className="w-5 h-5 text-red-400 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
            <path 
              fillRule="evenodd" 
              d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" 
              clipRule="evenodd" 
            />
          </svg>
        </div>
        <div className="flex-1">
          <h4 className="text-red-400 font-medium text-sm">{failureInfo.title}</h4>
          <p className="text-red-300 text-sm mt-1">{failureInfo.message}</p>
          <p className="text-red-200 text-xs mt-2">{failureInfo.action}</p>
        </div>
      </div>
    </div>
  );
}

FailureAlert.propTypes = {
  failureInfo: PropTypes.shape({
    title: PropTypes.string.isRequired,
    message: PropTypes.string.isRequired,
    action: PropTypes.string.isRequired
  })
};
