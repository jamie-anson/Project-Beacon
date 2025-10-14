/**
 * Job Status Utilities
 * Pure functions for job status detection, color mapping, and failure messaging
 */

/**
 * Get color classes for status badge
 * @param {string} status - The status string
 * @returns {string} Tailwind CSS classes for status badge
 */
export function getStatusColor(status) {
  switch (status) {
    case 'completed': 
      return 'bg-green-900/20 text-green-400 border-green-700';
    case 'running': 
    case 'processing': 
      return 'bg-yellow-900/20 text-yellow-400 border-yellow-700';
    case 'connecting': 
    case 'queued': 
      return 'bg-blue-900/20 text-blue-400 border-blue-700';
    case 'completing': 
      return 'bg-purple-900/20 text-purple-400 border-purple-700';
    case 'failed': 
      return 'bg-red-900/20 text-red-400 border-red-700';
    case 'stalled': 
      return 'bg-orange-900/20 text-orange-400 border-orange-700';
    case 'refreshing': 
      return 'bg-cyan-900/20 text-cyan-400 border-cyan-700';
    case 'pending': 
    default: 
      return 'bg-gray-900/20 text-gray-400 border-gray-700';
  }
}

/**
 * Determine the current job stage
 * @param {Object} job - The job object
 * @param {Array} executions - Array of execution objects
 * @param {boolean} isCompleted - Whether job is completed
 * @param {boolean} isFailed - Whether job failed
 * @param {boolean} isStuckTimeout - Whether job is stuck/timeout
 * @returns {string} Job stage identifier
 */
export function getJobStage(job, executions = [], isCompleted = false, isFailed = false, isStuckTimeout = false) {
  const statusStr = String(job?.status || job?.state || '').toLowerCase();
  const running = executions.filter((e) => (e?.status || e?.state) === 'running').length;
  
  // Check completion/failure states first
  if (isCompleted) return 'completed';
  if (isFailed || isStuckTimeout) return 'failed';
  
  if (statusStr === 'created') return 'creating';
  if (statusStr === 'queued' || statusStr === 'enqueued') return 'queued';
  if (statusStr === 'processing' && executions.length === 0) return 'spawning';
  if (statusStr === 'processing' && running > 0) return 'running';
  return 'unknown';
}

/**
 * Get enhanced status with granular state detection
 * @param {Object} execution - The execution object
 * @param {Object} job - The job object
 * @param {boolean} loadingActive - Whether actively loading
 * @param {boolean} jobCompleted - Whether job is completed
 * @param {boolean} jobFailed - Whether job failed
 * @param {boolean} jobStuckTimeout - Whether job is stuck
 * @returns {string} Enhanced status string
 */
export function getEnhancedStatus(execution, job, loadingActive, jobCompleted, jobFailed, jobStuckTimeout) {
  if (loadingActive) return 'refreshing';
  
  // Handle job-level failures first
  if (jobFailed || jobStuckTimeout) {
    return 'failed';
  }
  
  if (jobCompleted) {
    return 'completed';
  }
  
  if (!execution) return 'pending';
  
  const failure = execution?.output?.failure || execution?.failure || execution?.failure_reason || execution?.output?.failure_reason;
  const failureMessage = typeof failure === 'object' ? failure?.message : null;
  
  // Check for execution-level infrastructure errors
  if (execution?.error || execution?.failure_reason || failureMessage) {
    return 'failed';
  }
  
  const status = execution?.status || execution?.state || 'pending';
  const started = execution?.started_at || execution?.created_at;
  const now = new Date();
  const startTime = started ? new Date(started) : null;
  const runningTime = startTime ? (now - startTime) / 1000 / 60 : 0; // minutes
  
  // Detect granular states
  if (status === 'created' || status === 'enqueued') {
    return 'queued';
  }
  
  if (status === 'running') {
    // Check if it's been running for a while (might be stalled)
    if (runningTime > 30) {
      return 'stalled';
    }
    
    // Detect sub-states of running based on timing
    if (runningTime < 0.5) { // First 30 seconds
      return 'connecting';
    } else if (runningTime < 25) { // Most of execution time
      return 'processing';
    } else {
      return 'completing'; // Taking longer, probably finishing up
    }
  }
  
  return status;
}

/**
 * Generate failure message for failed jobs
 * @param {Object} job - The job object
 * @param {number} jobAge - Job age in minutes
 * @param {boolean} jobFailed - Whether job failed
 * @param {boolean} jobStuckTimeout - Whether job is stuck
 * @returns {Object|null} Failure info object or null
 */
export function getFailureMessage(job, jobAge, jobFailed, jobStuckTimeout) {
  const statusStr = String(job?.status || job?.state || '').toLowerCase();
  
  if (jobFailed) {
    return {
      title: "Job Failed",
      message: `Job failed with status: ${statusStr}. This may be due to system issues or invalid job configuration.`,
      action: "Try submitting a new job or contact support if the issue persists."
    };
  }
  
  if (jobStuckTimeout) {
    // Differentiate between stuck (no executions) and timeout (exceeded 60 min)
    if (jobAge > 60) {
      return {
        title: "Job Timeout",
        message: `Job exceeded maximum execution time (60 minutes). Current runtime: ${Math.round(jobAge)} minutes.`,
        action: "Job has been automatically terminated. Check partial results or submit a new job."
      };
    } else {
      return {
        title: "Job Timeout",
        message: `Job has been running for ${Math.round(jobAge)} minutes without creating any executions.`,
        action: "The job may be stuck. Try submitting a new job."
      };
    }
  }
  
  return null;
}

/**
 * Check if a question execution failed
 * @param {Object} exec - The execution object
 * @returns {boolean} True if execution failed
 */
export function isQuestionFailed(exec) {
  if (!exec) return false;
  const status = (exec.status || exec.state || '').toLowerCase();
  return status === 'failed' || status === 'timeout' || status === 'error' || !!exec.error || !!exec.failure_reason;
}

/**
 * Get classification badge info for execution
 * @param {Object} execution - The execution object
 * @returns {Object|null} Badge info object or null
 */
export function getClassificationBadge(execution) {
  if (!execution) return null;
  
  const classification = execution.response_classification;
  const isSubstantive = execution.is_substantive;
  const isRefusal = execution.is_content_refusal;
  const responseLength = execution.response_length || 0;
  
  // Return null if no classification info at all
  if (!classification && !isSubstantive && !isRefusal) return null;
  
  let badgeColor = 'bg-gray-900/20 text-gray-400 border-gray-700';
  let badgeText = classification;
  let badgeIcon = null;
  
  if (classification === 'substantive' || isSubstantive) {
    badgeColor = 'bg-green-900/20 text-green-400 border-green-700';
    badgeText = 'Substantive';
    badgeIcon = '✓';
  } else if (classification === 'content_refusal' || isRefusal) {
    badgeColor = 'bg-orange-900/20 text-orange-400 border-orange-700';
    badgeText = 'Refusal';
    badgeIcon = '⚠';
  } else if (classification === 'technical_failure') {
    badgeColor = 'bg-red-900/20 text-red-400 border-red-700';
    badgeText = 'Error';
    badgeIcon = '✗';
  }
  
  return {
    badgeColor,
    badgeText,
    badgeIcon,
    responseLength
  };
}
