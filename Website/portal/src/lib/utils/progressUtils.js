/**
 * Progress Utilities
 * Pure functions for progress calculation and tracking
 */

/**
 * Calculate expected total executions
 * @param {Object} job - The job object
 * @param {Array} selectedRegions - Array of selected region codes
 * @returns {number} Expected total executions
 */
export function calculateExpectedTotal(job, selectedRegions = []) {
  const jobSpec = job?.job || job;
  const specQuestions = jobSpec?.questions || [];
  // Models may be defined either at jobSpec.models or jobSpec.metadata.models
  const rawModels = jobSpec?.models || jobSpec?.metadata?.models || [];
  // Normalize model entries to unique identifiers (string id or object.id/name)
  const normalizedModelIds = Array.isArray(rawModels)
    ? rawModels.map((m) => {
        if (typeof m === 'string') return m;
        if (m && typeof m === 'object') {
          return (
            m.id || m.model_id || m.model || m.name || m.value || JSON.stringify(m)
          );
        }
        return String(m ?? '');
      })
      .filter(Boolean)
    : [];
  const specModelsCount = new Set(normalizedModelIds).size;
  
  let expectedTotal = 0;
  
  if (specQuestions.length > 0 && specModelsCount > 0) {
    // Questions × Models × Selected Regions
    expectedTotal = specQuestions.length * specModelsCount * selectedRegions.length;
  } else if (specModelsCount > 0) {
    // No questions, just Models × Selected Regions
    expectedTotal = specModelsCount * selectedRegions.length;
  } else {
    // Fallback to selected regions
    expectedTotal = selectedRegions.length;
  }
  
  return expectedTotal;
}

/**
 * Calculate progress metrics from executions
 * @param {Array} executions - Array of execution objects
 * @param {number} total - Total expected executions
 * @returns {Object} Progress metrics
 */
export function calculateProgress(executions = [], total = 0) {
  const completed = executions.filter((e) => (e?.status || e?.state) === 'completed').length;
  const running = executions.filter((e) => (e?.status || e?.state) === 'running').length;
  const failed = executions.filter((e) => (e?.status || e?.state) === 'failed').length;
  const pending = Math.max(0, total - completed - running - failed);
  const percentage = Math.round((completed / Math.max(total, 1)) * 100);
  
  return {
    completed,
    running,
    failed,
    pending,
    percentage,
    total
  };
}

/**
 * Calculate time remaining for job (10 minute countdown)
 * @param {Object} jobStartTime - Object with jobId and startTime
 * @param {number} tick - Current tick for re-calculation
 * @param {boolean} jobCompleted - Whether job is completed
 * @param {boolean} jobFailed - Whether job failed
 * @returns {string|null} Formatted time remaining or null
 */
export function calculateTimeRemaining(jobStartTime, tick, jobCompleted, jobFailed) {
  // Early exit if job is not active
  if (jobCompleted || jobFailed) return null;
  
  // Need start time to calculate countdown
  if (!jobStartTime) return null;
  
  const estimatedDuration = 10 * 60; // 10 minutes in seconds
  const now = Date.now();
  const elapsedMs = now - jobStartTime.startTime;
  const elapsedSeconds = Math.floor(elapsedMs / 1000);
  
  // Calculate remaining time (ensure non-negative)
  const remainingSeconds = Math.max(0, estimatedDuration - elapsedSeconds);
  
  // Stop showing countdown when time expires
  if (remainingSeconds <= 0) return null;
  
  const remainingMinutes = Math.floor(remainingSeconds / 60);
  const remainingSecsDisplay = remainingSeconds % 60;
  
  return `${remainingMinutes}:${remainingSecsDisplay.toString().padStart(2, '0')}`;
}

/**
 * Calculate job age in minutes
 * @param {Object} jobStartTime - Object with jobId and startTime (deprecated, use job.created_at instead)
 * @param {Object} job - Job object with created_at timestamp
 * @returns {number} Job age in minutes
 */
export function calculateJobAge(jobStartTime, job = null) {
  // Prefer actual job creation time over component mount time
  if (job && job.created_at) {
    const createdAt = new Date(job.created_at).getTime();
    return (Date.now() - createdAt) / 1000 / 60;
  }
  
  // Fallback to component mount time (legacy behavior)
  if (!jobStartTime) return 0;
  return (Date.now() - jobStartTime.startTime) / 1000 / 60;
}

/**
 * Check if job is stuck (timeout)
 * @param {number} jobAge - Job age in minutes
 * @param {Array} executions - Array of execution objects
 * @param {boolean} jobCompleted - Whether job is completed
 * @param {boolean} jobFailed - Whether job failed
 * @returns {boolean} True if job is stuck
 */
export function isJobStuck(jobAge, executions = [], jobCompleted, jobFailed) {
  // Job stuck: no executions after 15 minutes
  if (jobAge > 15 && executions.length === 0 && !jobCompleted && !jobFailed) {
    return true;
  }
  
  // Job timeout: exceeded 60 minutes and still running
  if (jobAge > 60 && !jobCompleted && !jobFailed) {
    return true;
  }
  
  return false;
}

/**
 * Get unique models from executions
 * @param {Array} executions - Array of execution objects
 * @returns {Array} Array of unique model IDs
 */
export function getUniqueModels(executions = []) {
  return [...new Set(executions.map(e => e.model_id).filter(Boolean))];
}

/**
 * Get unique questions from executions
 * @param {Array} executions - Array of execution objects
 * @returns {Array} Array of unique question IDs
 */
export function getUniqueQuestions(executions = []) {
  return [...new Set(executions.map(e => e.question_id).filter(Boolean))];
}

/**
 * Calculate per-question progress
 * @param {string} questionId - The question ID
 * @param {Array} executions - Array of execution objects
 * @param {Array} specModels - Array of model specs
 * @param {Array} selectedRegions - Array of selected regions
 * @param {Array} uniqueModels - Array of unique model IDs
 * @returns {Object} Question progress metrics
 */
export function calculateQuestionProgress(questionId, executions, specModels, selectedRegions, uniqueModels) {
  const questionExecs = executions.filter(e => e.question_id === questionId);
  const qCompleted = questionExecs.filter(e => e.status === 'completed').length;
  const qTotal = questionExecs.length;
  
  // Calculate expected per question from spec
  let qExpected = 0;
  if (specModels.length > 0) {
    for (const model of specModels) {
      qExpected += (model.regions || []).length;
    }
  } else {
    qExpected = selectedRegions.length * (uniqueModels.length || 1);
  }
  
  const qRefused = questionExecs.filter(e => 
    e.response_classification === 'content_refusal' || e.is_content_refusal
  ).length;
  
  return {
    completed: qCompleted,
    total: qTotal,
    expected: qExpected,
    refused: qRefused
  };
}

/**
 * Calculate region progress for multi-model jobs
 * @param {Array} regionExecs - Executions for this region
 * @returns {Object} Region progress metrics
 */
export function calculateRegionProgress(regionExecs = []) {
  const completedCount = regionExecs.filter(ex => ex?.status === 'completed').length;
  const failedCount = regionExecs.filter(ex => ex?.status === 'failed').length;
  const runningCount = regionExecs.filter(ex => ex?.status === 'running').length;
  const totalModels = regionExecs.length;
  
  return {
    completed: completedCount,
    failed: failedCount,
    running: runningCount,
    total: totalModels,
    percentage: totalModels > 0 ? Math.round((completedCount / totalModels) * 100) : 0
  };
}
