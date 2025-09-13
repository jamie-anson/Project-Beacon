import React from 'react';

const ErrorMessage = ({ error, onRetry, retryAfter }) => {
  if (!error) return null;

  const getErrorInfo = (error) => {
    // Handle structured API errors
    if (error.code) {
      switch (error.code) {
        case 'DATABASE_CONNECTION_FAILED':
          return {
            title: 'Database Service Unavailable',
            message: error.user_message || 'The database service is temporarily unavailable. Please try again in a few moments.',
            icon: '🔌',
            color: 'red',
            canRetry: true
          };
        case 'INFRASTRUCTURE_UNAVAILABLE':
          return {
            title: 'Infrastructure Issues',
            message: error.user_message || 'Some infrastructure services are experiencing issues. Job execution may be affected.',
            icon: '⚠️',
            color: 'yellow',
            canRetry: true
          };
        case 'JOB_TRACKING_FAILED':
          return {
            title: 'Job Tracking Unavailable',
            message: error.user_message || 'Unable to track job progress. The job may still be running.',
            icon: '📊',
            color: 'yellow',
            canRetry: true
          };
        case 'CROSS_REGION_EXECUTION_FAILED':
          return {
            title: 'Cross-Region Analysis Failed',
            message: error.user_message || 'Cross-region analysis is temporarily unavailable. Please try again later.',
            icon: '🌍',
            color: 'red',
            canRetry: true
          };
        default:
          return {
            title: 'Service Error',
            message: error.user_message || error.message || 'An unexpected error occurred.',
            icon: '❌',
            color: 'red',
            canRetry: true
          };
      }
    }

    // Handle generic errors
    if (typeof error === 'string') {
      return {
        title: 'Error',
        message: error,
        icon: '❌',
        color: 'red',
        canRetry: true
      };
    }

    // Handle Error objects
    if (error.message) {
      return {
        title: 'Request Failed',
        message: error.message,
        icon: '🔌',
        color: 'red',
        canRetry: true
      };
    }

    return {
      title: 'Unknown Error',
      message: 'Something went wrong. Please try again.',
      icon: '❓',
      color: 'gray',
      canRetry: true
    };
  };

  const errorInfo = getErrorInfo(error);
  
  const colorClasses = {
    red: 'bg-red-50 border-red-200 text-red-800',
    yellow: 'bg-yellow-50 border-yellow-200 text-yellow-800',
    gray: 'bg-gray-50 border-gray-200 text-gray-800'
  };

  const buttonColorClasses = {
    red: 'bg-red-600 hover:bg-red-700 text-white',
    yellow: 'bg-yellow-600 hover:bg-yellow-700 text-white',
    gray: 'bg-gray-600 hover:bg-gray-700 text-white'
  };

  return (
    <div className={`border rounded-lg p-4 ${colorClasses[errorInfo.color]}`}>
      <div className="flex items-start">
        <div className="text-2xl mr-3 mt-0.5">{errorInfo.icon}</div>
        <div className="flex-1">
          <h3 className="font-semibold text-sm mb-1">{errorInfo.title}</h3>
          <p className="text-sm mb-3">{errorInfo.message}</p>
          
          {retryAfter && (
            <p className="text-xs opacity-75 mb-3">
              Recommended retry time: {retryAfter} seconds
            </p>
          )}
          
          <div className="flex items-center gap-2">
            {errorInfo.canRetry && onRetry && (
              <button
                onClick={onRetry}
                className={`px-3 py-1.5 rounded text-sm font-medium transition-colors ${buttonColorClasses[errorInfo.color]}`}
              >
                Try Again
              </button>
            )}
            
            <button
              onClick={() => window.location.reload()}
              className="px-3 py-1.5 rounded text-sm font-medium bg-gray-200 hover:bg-gray-300 text-gray-800 transition-colors"
            >
              Refresh Page
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ErrorMessage;
