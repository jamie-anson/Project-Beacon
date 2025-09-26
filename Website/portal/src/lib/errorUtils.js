/**
 * Utility functions for parsing and formatting API errors into user-friendly messages
 */

import { ApiError } from './api/http.js';

function isApiError(error) {
  return error instanceof ApiError || error?.name === 'ApiError';
}

function getErrorStatus(error) {
  if (typeof error?.status === 'number') return error.status;
  if (typeof error?.response?.status === 'number') return error.response.status;
  return null;
}

function getErrorCode(error) {
  return (
    error?.code ??
    error?.errorCode ??
    error?.data?.error_code ??
    error?.data?.code ??
    error?.data?.failure?.code ??
    null
  );
}

function getErrorMessage(error) {
  return error?.data?.error || error?.data?.message || error?.message || error?.raw || null;
}

function formatStage(stage) {
  if (!stage || typeof stage !== 'string') return null;
  return stage
    .split(/[_\s]+/)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

/**
 * Parse an error object and return a user-friendly error message
 * @param {Error|Object} error - The error to parse
 * @returns {Object} - { title, message, details }
 */
export function parseApiError(error) {
  const status = getErrorStatus(error);
  const code = getErrorCode(error);
  const message = getErrorMessage(error);
  const isStructured = isApiError(error);
  const rawDetails = error?.raw || error?.data?.details;
  const failure = error?.data?.failure || error?.failure || error?.data?.error?.failure;

  if (failure && typeof failure === 'object') {
    const failureCode = failure.code || code || 'EXECUTION_FAILURE';
    const stageLabel = formatStage(failure.stage) || 'Execution Stage';
    const providerLabel = failure.provider || failure.provider_type;
    const regionLabel = failure.region;

    const detailParts = [];
    if (stageLabel) detailParts.push(`${stageLabel}`);
    if (regionLabel) detailParts.push(`Region: ${regionLabel}`);
    if (providerLabel) detailParts.push(`Provider: ${providerLabel}`);
    if (typeof failure.transient === 'boolean') {
      detailParts.push(failure.transient ? 'Transient: yes' : 'Transient: no');
    }
    if (failure.http_status) detailParts.push(`HTTP ${failure.http_status}`);
    if (failure.retry_after) detailParts.push(`Retry after ${failure.retry_after}s`);

    return {
      title: failureCode,
      message: failure.message || message || 'The execution failed. See details below.',
      details: detailParts.join(' · ') || rawDetails || '',
    };
  }

  // Handle network/CORS errors
  if (code === 'NETWORK_ERROR' || (!status && (message?.includes('Failed to fetch') || message?.includes('CORS')))) {
    return {
      title: 'Connection Error',
      message: 'Unable to connect to the API server. Please check your internet connection and try again.',
      details: 'This could be a CORS issue or the server might be temporarily unavailable.'
    };
  }

  if (code === 'ABORT_ERROR') {
    return {
      title: 'Request Cancelled',
      message: 'The request was cancelled before completion.',
      details: 'You may have navigated away or triggered a new request that aborted the previous one.'
    };
  }

  if (code === 'INVALID_JSON') {
    return {
      title: 'Invalid Response',
      message: 'Received malformed data from the API. Please try again shortly.',
      details: rawDetails
        ? `The server response could not be parsed as JSON. Raw response: ${rawDetails}`
        : 'The server response could not be parsed as JSON.'
    };
  }

  // Handle timeout errors
  if (message?.includes('timeout') || code === 'TIMEOUT') {
    return {
      title: 'Request Timeout',
      message: 'The request took too long to complete. Please try again.',
      details: 'The server may be experiencing high load or temporary issues.'
    };
  }

  // Handle cryptographic signing errors
  if (message?.includes('untrusted signing key') || message?.includes('trust violation')) {
    return {
      title: 'Authentication Error',
      message: 'Your signing key is not authorized for job submission.',
      details: 'Please contact an administrator to add your public key to the trusted keys allowlist.'
    };
  }

  if (message?.includes('signature') || message?.includes('signing')) {
    return {
      title: 'Signature Error',
      message: 'Failed to sign the job request. Please try refreshing the page.',
      details: 'There may be an issue with your cryptographic keys stored in the browser.'
    };
  }

  // Handle validation errors
  if (status === 400 || message?.includes('validation')) {
    return {
      title: 'Invalid Request',
      message: 'The job request contains invalid data. Please check your inputs and try again.',
      details: message || 'One or more fields in the job specification are invalid.'
    };
  }

  // Handle authorization errors
  if (status === 401 || message?.includes('unauthorized')) {
    return {
      title: 'Unauthorized',
      message: 'You are not authorized to perform this action.',
      details: 'Please check your authentication credentials and permissions.'
    };
  }

  if (status === 403 || message?.includes('forbidden')) {
    return {
      title: 'Access Denied',
      message: 'You do not have permission to access this resource.',
      details: 'Contact an administrator if you believe you should have access.'
    };
  }

  // Handle not found errors
  if (status === 404 || message?.includes('not found')) {
    return {
      title: 'Resource Not Found',
      message: 'The requested resource could not be found.',
      details: 'The job or resource may have been deleted or moved.'
    };
  }

  // Handle server errors
  if ((typeof status === 'number' && status >= 500) || message?.includes('server error')) {
    return {
      title: 'Server Error',
      message: 'The server encountered an error while processing your request.',
      details: 'Please try again in a few moments. If the problem persists, contact support.'
    };
  }

  // Handle rate limiting
  if (status === 429 || message?.includes('rate limit')) {
    return {
      title: 'Rate Limited',
      message: 'Too many requests. Please wait a moment before trying again.',
      details: 'You have exceeded the allowed number of requests per minute.'
    };
  }

  // Handle quota/resource errors
  if (message?.includes('quota') || message?.includes('insufficient')) {
    return {
      title: 'Resource Unavailable',
      message: 'Insufficient resources to process your request.',
      details: 'The system may be at capacity or you may have reached your usage limits.'
    };
  }

  // Handle job-specific errors
  if (message?.includes('job') && message?.includes('failed')) {
    return {
      title: 'Job Execution Failed',
      message: 'The job could not be completed successfully.',
      details: message || 'Check the job logs for more details about the failure.'
    };
  }

  // Generic fallback
  return {
    title: 'Unexpected Error',
    message: message || 'An unexpected error occurred. Please try again.',
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
