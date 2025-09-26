import { runnerFetch } from '../http.js';

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
