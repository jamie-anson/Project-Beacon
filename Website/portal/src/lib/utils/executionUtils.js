/**
 * Execution Utilities
 * Pure functions for execution data extraction and formatting
 */

import { regionCodeFromExec, normalizeRegion } from './regionUtils';

/**
 * Extract text output from execution
 * @param {Object} exec - The execution object
 * @returns {string} Extracted text or empty string
 */
export function extractExecText(exec) {
  const out = exec?.output || exec?.result || {};
  try {
    if (typeof out?.response === 'string' && out.response) {
      // For multi-question responses, show a preview
      const response = out.response;
      if (response.length > 200) {
        return response.substring(0, 200) + '... (click to view full response)'; // 200 + 33 = 233
      }
      return response;
    }
    if (out.responses && Array.isArray(out.responses) && out.responses.length > 0) {
      const r = out.responses[0];
      return r.response || r.answer || r.output || '';
    }
    if (out.text_output) return out.text_output;
    if (out.output) return out.output;
  } catch {}
  return '';
}

/**
 * Prefill diff comparison from executions
 * @param {Object} activeJob - The active job object
 * @param {Object} setters - Object containing setter functions
 * @returns {void}
 */
export function prefillFromExecutions(activeJob, setters) {
  const { setARegion, setBRegion, setAText, setBText, setError } = setters || {};
  try {
    const execs = Array.isArray(activeJob?.executions) ? activeJob.executions : [];
    if (execs.length === 0) throw new Error('No executions available to prefill');
    
    const completed = execs.filter(e => String(e?.status || e?.state || '').toLowerCase() === 'completed');
    const pick = (regionCode) => completed.find(e => regionCodeFromExec(e) === regionCode);
    
    // Prefer US/EU, fallback to any two different regions
    let eA = pick('US') || completed[0] || execs[0];
    let eB = pick('EU') || completed.find(e => e !== eA) || execs.find(e => e !== eA) || eA;
    
    const rA = normalizeRegion(eA?.region || eA?.region_claimed);
    const rB = normalizeRegion(eB?.region || eB?.region_claimed);
    const tA = extractExecText(eA);
    const tB = extractExecText(eB);
    
    if (setARegion) setARegion(rA);
    if (setBRegion) setBRegion(rB);
    if (setAText) setAText(tA || '');
    if (setBText) setBText(tB || '');
  } catch (err) {
    if (setError) setError(err?.message || String(err));
  }
}

/**
 * Truncate string in the middle
 * @param {string} str - String to truncate
 * @param {number} head - Number of characters to keep at start
 * @param {number} tail - Number of characters to keep at end
 * @returns {string} Truncated string
 */
export function truncateMiddle(str, head = 6, tail = 4) {
  if (!str || typeof str !== 'string') return '—';
  if (str.length <= head + tail + 1) return str;
  return `${str.slice(0, head)}…${str.slice(-tail)}`;
}

/**
 * Format timestamp as relative time
 * @param {string|number|Date} ts - Timestamp
 * @returns {string} Formatted relative time string
 */
export function timeAgo(ts) {
  if (!ts) return '';
  try {
    const d = new Date(ts).getTime();
    if (isNaN(d)) return String(ts);
    const diff = Date.now() - d;
    const sec = Math.floor(diff / 1000);
    if (sec < 60) return `${sec}s ago`;
    const min = Math.floor(sec / 60);
    const hr = Math.floor(min / 60);
    if (hr < 24) return `${hr}h ago`;
    const day = Math.floor(hr / 24);
    return `${day}d ago`;
  } catch {
    return String(ts);
  }
}

/**
 * Get failure details from execution
 * @param {Object} execution - The execution object
 * @returns {Object} Failure details object
 */
export function getFailureDetails(execution) {
  if (!execution) return null;
  
  const failure = execution?.output?.failure || execution?.failure || execution?.failure_reason || execution?.output?.failure_reason;
  const failureMessage = typeof failure === 'object' ? failure?.message : failure;
  const failureCode = typeof failure === 'object' ? failure?.code : null;
  const failureStage = typeof failure === 'object' ? failure?.stage : null;
  
  const message = failureMessage || execution?.error || execution?.failure_reason;
  
  // Return null if no failure information at all
  if (!message && !failureCode && !failureStage) return null;
  
  return {
    message,
    code: failureCode,
    stage: failureStage
  };
}
