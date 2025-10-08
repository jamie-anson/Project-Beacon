import { runnerFetch } from '../http.js';
import { computeIdempotencyKey, shouldSendIdempotency } from '../idempotency.js';

export const getIdempotencyKeyForJob = computeIdempotencyKey;

export async function createJob(initialJobspec, opts = {}) {
  let jobspec = initialJobspec;
  const key = opts.idempotencyKey || computeIdempotencyKey(jobspec);

  console.log('Creating job with payload:', jobspec);

  try {
    const isV1 = String(jobspec?.version || '').toLowerCase() === 'v1';
    const benchName = String(jobspec?.benchmark?.name || '').toLowerCase();
    const hasQuestions = Array.isArray(jobspec?.questions) && jobspec.questions.length > 0;
    if (isV1 && benchName.includes('bias') && !hasQuestions) {
      let selected = [];
      try {
        const raw = localStorage.getItem('beacon:selected_questions');
        if (raw) {
          const arr = JSON.parse(raw);
          if (Array.isArray(arr)) selected = arr;
        }
      } catch {}
      if (!selected || selected.length === 0) {
        selected = ['identity_basic'];
        console.warn('[Beacon] No questions found; injecting minimal default questions:', selected);
      } else {
        console.info('[Beacon] Injecting selected questions into JobSpec:', selected.length);
      }
      jobspec = { ...jobspec, questions: selected };
    }
    if (isV1 && benchName.includes('bias')) {
      const ok = Array.isArray(jobspec?.questions) && jobspec.questions.length > 0;
      if (!ok) {
        throw new Error('questions are required for bias-detection; please select at least one on the Questions page');
      }
    }
  } catch (e) {
    console.warn('[Beacon] questions injection skipped:', e?.message || String(e));
  }

  // Check if this is a bias detection job with multiple regions
  const isBiasDetection = String(jobspec?.benchmark?.name || '').toLowerCase().includes('bias');
  const regions = jobspec?.constraints?.regions || [];
  const isMultiRegion = regions.length > 0;

  // Use cross-region endpoint for bias detection jobs
  if (isBiasDetection && isMultiRegion) {
    console.log('[Beacon] Using cross-region endpoint for bias detection job');
    
    // Transform payload for cross-region endpoint
    const { jobspec_id, constraints, ...rest } = jobspec;
    const crossRegionPayload = {
      jobspec: {  // CRITICAL: use "jobspec" not "job_spec"
        id: jobspec_id,  // Rename jobspec_id -> id
        ...rest,
        constraints
      },
      target_regions: constraints.regions,
      min_regions: constraints.min_regions || 1,
      min_success_rate: constraints.min_success_rate || 0.67,
      enable_analysis: true
    };

    console.log('Cross-region payload:', crossRegionPayload);

    const headers = {};
    const enableIdem = opts.forceIdempotency === true || (opts.forceIdempotency !== false && shouldSendIdempotency());
    if (enableIdem) {
      headers['Idempotency-Key'] = key;
    }

    return runnerFetch('/jobs/cross-region', {
      method: 'POST',
      headers,
      body: JSON.stringify(crossRegionPayload),
    });
  }

  // Standard endpoint for non-bias-detection jobs
  let bodyString;
  try {
    bodyString = JSON.stringify(jobspec);
    console.log('Serialized JSON:', bodyString);
  } catch (error) {
    console.error('JSON serialization failed:', error);
    throw new Error(`Failed to serialize job payload: ${error.message}`);
  }

  const headers = {};
  const enableIdem = opts.forceIdempotency === true || (opts.forceIdempotency !== false && shouldSendIdempotency());
  if (enableIdem) {
    headers['Idempotency-Key'] = key;
  }

  return runnerFetch('/jobs', {
    method: 'POST',
    headers,
    body: bodyString,
  });
}

export function getJob({ id, include, exec_limit, exec_offset }) {
  const params = new URLSearchParams();
  if (include) params.set('include', include);
  if (exec_limit != null) params.set('exec_limit', String(exec_limit));
  if (exec_offset != null) params.set('exec_offset', String(exec_offset));
  const qs = params.toString();
  return runnerFetch(`/jobs/${encodeURIComponent(id)}${qs ? `?${qs}` : ''}`);
}

export function listJobs({ limit = 50 } = {}) {
  const params = new URLSearchParams();
  params.set('limit', String(limit));
  return runnerFetch(`/jobs?${params.toString()}`).then((data) => {
    if (Array.isArray(data)) return { jobs: data };
    return data;
  });
}

export function executeJob(jobId) {
  return runnerFetch(`/jobs/${jobId}/execute`, { method: 'POST' });
}

export function createCrossRegionJob(payload) {
  return runnerFetch('/jobs/cross-region', { method: 'POST', body: JSON.stringify(payload) });
}
