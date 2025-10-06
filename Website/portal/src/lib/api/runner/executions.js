import { runnerFetch } from '../http.js';
import { resolveRunnerBase } from '../config.js';

export function getExecutions({ limit = 20 } = {}) {
  return runnerFetch(`/executions?limit=${limit}`).then((data) => {
    if (Array.isArray(data)) return data;
    if (data && Array.isArray(data.executions)) return data.executions;
    return [];
  });
}

export function getDiffs({ limit = 20 } = {}) {
  return runnerFetch(`/diffs?limit=${limit}`).then((data) => {
    if (Array.isArray(data)) return data;
    if (data && Array.isArray(data.diffs)) return data.diffs;
    return [];
  });
}

export function getExecution(id) {
  return runnerFetch(`/executions/${encodeURIComponent(id)}/details`);
}

export function getExecutionReceipt(id) {
  return runnerFetch(`/executions/${encodeURIComponent(id)}/receipt`);
}

export function createCrossRegionDiff(jobId) {
  return runnerFetch(`/executions/${encodeURIComponent(jobId)}/cross-region-diff`, { method: 'POST' });
}

export function getRegionResults(jobId) {
  return runnerFetch(`/executions/${encodeURIComponent(jobId)}/regions`);
}

export function getBiasAnalysis(jobId) {
  // Note: This endpoint is v2, but runnerFetch adds /api/v1 prefix
  // So we need to construct the full path without the prefix
  const base = resolveRunnerBase();
  const url = base 
    ? `${base}/api/v2/jobs/${encodeURIComponent(jobId)}/bias-analysis`
    : `/api/v2/jobs/${encodeURIComponent(jobId)}/bias-analysis`;
  
  return fetch(url, {
    method: 'GET',
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json'
    },
    mode: 'cors',
    credentials: 'omit'
  }).then(async (res) => {
    if (!res.ok) {
      throw new Error(`HTTP ${res.status}: ${res.statusText}`);
    }
    return res.json();
  });
}

/**
 * Retry a single failed question for a specific execution and region
 * @param {string} executionId - The execution ID
 * @param {string} region - The region name (e.g., "United States", "Europe")
 * @param {number} questionIndex - The zero-based question index
 * @returns {Promise<Object>} Updated execution data with retry result
 */
export function retryQuestion(executionId, region, questionIndex) {
  return runnerFetch(`/executions/${encodeURIComponent(executionId)}/retry-question`, {
    method: 'POST',
    body: JSON.stringify({
      region,
      question_index: questionIndex
    })
  });
}

/**
 * Retry all failed questions for a given execution
 * @param {string} executionId - The execution ID
 * @returns {Promise<Object>} Batch retry results
 */
export function retryAllFailed(executionId) {
  return runnerFetch(`/executions/${encodeURIComponent(executionId)}/retry-all-failed`, {
    method: 'POST'
  });
}
