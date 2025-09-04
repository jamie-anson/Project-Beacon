/**
 * Utility functions for parsing and formatting API errors into user-friendly messages
 */

/**
 * Parse an error object and return a user-friendly error message
 * @param {Error|Object} error - The error to parse
 * @returns {Object} - { title, message, details }
 */
export function parseApiError(error) {
  // Handle network/CORS errors
  if (error?.message?.includes('Failed to fetch') || error?.message?.includes('CORS')) {
    return {
      title: 'Connection Error',
      message: 'Unable to connect to the API server. Please check your internet connection and try again.',
      details: 'This could be a CORS issue or the server might be temporarily unavailable.'
    };
  }

  // Handle timeout errors
  if (error?.message?.includes('timeout') || error?.code === 'TIMEOUT') {
    return {
      title: 'Request Timeout',
      message: 'The request took too long to complete. Please try again.',
      details: 'The server may be experiencing high load or temporary issues.'
    };
  }

  // Handle cryptographic signing errors
  if (error?.message?.includes('untrusted signing key') || error?.message?.includes('trust violation')) {
    return {
      title: 'Authentication Error',
      message: 'Your signing key is not authorized for job submission.',
      details: 'Please contact an administrator to add your public key to the trusted keys allowlist.'
    };
  }

  if (error?.message?.includes('signature') || error?.message?.includes('signing')) {
    return {
      title: 'Signature Error',
      message: 'Failed to sign the job request. Please try refreshing the page.',
      details: 'There may be an issue with your cryptographic keys stored in the browser.'
    };
  }

  // Handle validation errors
  if (error?.message?.includes('validation') || error?.status === 400) {
    return {
      title: 'Invalid Request',
      message: 'The job request contains invalid data. Please check your inputs and try again.',
      details: error?.message || 'One or more fields in the job specification are invalid.'
    };
  }

  // Handle authorization errors
  if (error?.status === 401 || error?.message?.includes('unauthorized')) {
    return {
      title: 'Unauthorized',
      message: 'You are not authorized to perform this action.',
      details: 'Please check your authentication credentials and permissions.'
    };
  }

  if (error?.status === 403 || error?.message?.includes('forbidden')) {
    return {
      title: 'Access Denied',
      message: 'You do not have permission to access this resource.',
      details: 'Contact an administrator if you believe you should have access.'
    };
  }

  // Handle not found errors
  if (error?.status === 404 || error?.message?.includes('not found')) {
    return {
      title: 'Resource Not Found',
      message: 'The requested resource could not be found.',
      details: 'The job or resource may have been deleted or moved.'
    };
  }

  // Handle server errors
  if (error?.status >= 500 || error?.message?.includes('server error')) {
    return {
      title: 'Server Error',
      message: 'The server encountered an error while processing your request.',
      details: 'Please try again in a few moments. If the problem persists, contact support.'
    };
  }

  // Handle rate limiting
  if (error?.status === 429 || error?.message?.includes('rate limit')) {
    return {
      title: 'Rate Limited',
      message: 'Too many requests. Please wait a moment before trying again.',
      details: 'You have exceeded the allowed number of requests per minute.'
    };
  }

  // Handle quota/resource errors
  if (error?.message?.includes('quota') || error?.message?.includes('insufficient')) {
    return {
      title: 'Resource Unavailable',
      message: 'Insufficient resources to process your request.',
      details: 'The system may be at capacity or you may have reached your usage limits.'
    };
  }

  // Handle job-specific errors
  if (error?.message?.includes('job') && error?.message?.includes('failed')) {
    return {
      title: 'Job Execution Failed',
      message: 'The job could not be completed successfully.',
      details: error?.message || 'Check the job logs for more details about the failure.'
    };
  }

  // Generic fallback
  return {
    title: 'Unexpected Error',
    message: error?.message || 'An unexpected error occurred. Please try again.',
    details: 'If this problem continues, please contact support with the error details.'
  };
}

/**
 * Create a toast-friendly error object from an API error
 * @param {Error|Object} error - The error to parse
 * @returns {Object} - Toast configuration object
 */
export function createErrorToast(error) {
  const parsed = parseApiError(error);
  
  return {
    title: parsed.title,
    message: parsed.message,
    timeout: 8000, // Longer timeout for error messages
    type: 'error'
  };
}

/**
 * Create a success toast for job operations
 * @param {string} jobId - The job ID
 * @param {string} action - The action performed (e.g., 'created', 'submitted')
 * @returns {Object} - Toast configuration object
 */
export function createSuccessToast(jobId, action = 'created') {
  return {
    title: 'Success',
    message: `Job ${action} successfully: ${jobId}`,
    timeout: 5000,
    type: 'success'
  };
}

/**
 * Create a warning toast for partial failures
 * @param {string} message - The warning message
 * @returns {Object} - Toast configuration object
 */
export function createWarningToast(message) {
  return {
    title: 'Warning',
    message,
    timeout: 6000,
    type: 'warning'
  };
}
