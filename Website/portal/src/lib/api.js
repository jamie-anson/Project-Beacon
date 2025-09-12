// Use environment variable for API base, fallback to Fly.io runner app (Railway only has hybrid router)
// Normalize: ensure no trailing slash and strip a mistakenly included "/api/v1" suffix.
let __apiBase = (import.meta.env?.VITE_API_BASE || 'https://beacon-runner-change-me.fly.dev');
// Runtime override (useful during deploy previews/unique deploys)
try {
  const lsBase = localStorage.getItem('beacon:api_base');
  if (lsBase && lsBase.trim()) {
    __apiBase = lsBase.trim();
  }
} catch {}
// Prefer same-origin when on Netlify domains to avoid CORS in unique deploys
try {
  const host = window.location.host || '';
  if (/netlify\.app$/i.test(host)) {
    __apiBase = window.location.origin;
  }
} catch {}
try {
  __apiBase = String(__apiBase)
    .replace(/\s+/g, '')
    .replace(/\/?api\/v1\/?$/i, '') // strip trailing /api/v1 if present
    .replace(/\/$/, '');              // then strip trailing slash
} catch {}
// One-time debug in development builds to help diagnose misconfigurations
try { if (import.meta?.env?.DEV) console.info('[Beacon] API_BASE_V1 =', __apiBase); } catch {}
const API_BASE_V1 = __apiBase;

// Simple tab identifier for semi-stable idempotency keys
function getTabId() {
  try {
    let id = sessionStorage.getItem('beacon:tab_id');
    if (!id) {
      id = `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
      sessionStorage.setItem('beacon:tab_id', id);
    }
    return id;
  } catch {
    return 'tab-unknown';
  }
}

function shortHash(str) {
  let h = 0;
  for (let i = 0; i < str.length; i++) h = ((h << 5) - h) + str.charCodeAt(i) | 0;
  // convert to unsigned and base36
  return (h >>> 0).toString(36);
}

// Compute a deterministic idempotency key for a jobspec within a short time window (e.g., 60s)
function computeIdempotencyKey(jobspec, windowSeconds = 60) {
  const tab = getTabId();
  const bucket = Math.floor(Date.now() / 1000 / windowSeconds); // time bucket
  let specStr = '';
  try { specStr = JSON.stringify(jobspec); } catch { specStr = String(jobspec || ''); }
  const base = `${tab}:${bucket}:${specStr}`;
  return `beacon-${shortHash(base)}`;
}

// Determine whether to send the Idempotency-Key header.
// Default: disabled unless explicitly enabled via env or localStorage.
function isTruthy(v) {
  try { return /^(1|true|yes|on)$/i.test(String(v || '')); } catch { return false; }
}

function shouldSendIdempotency() {
  // Prefer build-time flag
  try {
    const envVal = import.meta?.env?.VITE_ENABLE_IDEMPOTENCY;
    if (envVal != null) return isTruthy(envVal);
  } catch {}
  // Allow runtime toggle via localStorage for debugging
  try {
    const lsVal = localStorage.getItem('beacon:enable_idempotency');
    if (lsVal != null) return isTruthy(lsVal);
  } catch {}
  return false;
}

async function httpV1(path, opts = {}) {
  const url = `${API_BASE_V1}/api/v1${path.startsWith('/') ? path : '/' + path}`;
  try {
    const fetchOptions = {
      // Spread opts first so we can reliably override/merge headers below
      ...opts,
      headers: { 'Accept': 'application/json', 'Content-Type': 'application/json; charset=utf-8', ...(opts.headers || {}) },
    };
    
    // Explicitly set CORS mode to prevent browser blocking
    fetchOptions.mode = 'cors';
    fetchOptions.credentials = 'omit';
    
    // Enforce JSON content-type at the wire even if some layer mutates headers.
    // If the body is a string, wrap it in a Blob with application/json so the browser sets the header correctly.
    try {
      const h = fetchOptions.headers || {};
      const ct = (h['Content-Type'] || h['content-type'] || '').toString();
      if (fetchOptions.body && typeof fetchOptions.body === 'string') {
        if (!/application\/json/i.test(ct)) {
          fetchOptions.headers = { ...h, 'Content-Type': 'application/json; charset=utf-8' };
        }
        // Do NOT wrap body in a Blob; send raw JSON string to preserve exact bytes for signature verification.
      }
    } catch {}
    
    const res = await fetch(url, fetchOptions);
    if (!res.ok) {
      // Try to extract API error message from response body
      let errorMessage = `${res.status} ${res.statusText}`;
      try {
        const errorBody = await res.json();
        if (errorBody.error || errorBody.message) {
          errorMessage = errorBody.error || errorBody.message;
          if (errorBody.error_code) {
            errorMessage += ` (${errorBody.error_code})`;
          }
        }
      } catch {
        // If we can't parse the error body, use the status text
      }
      throw new Error(errorMessage);
    }
    return res.status === 204 ? null : res.json();
  } catch (err) {
    // Check if this is a CORS/network error vs API error
    if (err.message.includes('Failed to fetch') || err.message.includes('Load failed')) {
      console.warn(`Network/CORS error for ${url}:`, err.message);
      console.warn('This may be a CORS issue or the API server may be unreachable');
    } else {
      console.warn(`API call failed: ${url}`, err.message);
    }
    throw err;
  }
}

async function httpRoot(path, opts = {}) {
  try {
    const fetchOptions = {
      headers: { 'Content-Type': 'application/json', 'Accept': 'application/json', ...(opts.headers || {}) },
      ...opts,
    };
    
    // Explicitly set CORS mode to prevent browser blocking
    fetchOptions.mode = 'cors';
    fetchOptions.credentials = 'omit';
    
    const res = await fetch(`${path}`, fetchOptions);
    if (!res.ok) {
      // Try to extract API error message from response body
      let errorMessage = `${res.status} ${res.statusText}`;
      try {
        const errorBody = await res.json();
        if (errorBody.error || errorBody.message) {
          errorMessage = errorBody.error || errorBody.message;
          if (errorBody.error_code) {
            errorMessage += ` (${errorBody.error_code})`;
          }
        }
      } catch {
        // If we can't parse the error body, use the status text
      }
      throw new Error(errorMessage);
    }
    return res.status === 204 ? null : res.json();
  } catch (err) {
    // Check if this is a CORS/network error vs API error
    if (err.message.includes('Failed to fetch') || err.message.includes('Load failed')) {
      console.warn(`Network/CORS error for ${path}:`, err.message);
      console.warn('This may be a CORS issue or the API server may be unreachable');
    } else {
      console.warn(`API call failed: ${path}`, err.message);
    }
    throw err;
  }
}

// Health endpoint: use API base URL to call runner health
export const getHealth = () => httpV1('/health').then(data => {
  // Transform the runner health response to match dashboard expectations
  if (data && data.services) {
    const healthData = {};
    data.services.forEach(service => {
      healthData[service.name] = service.status;
    });
    healthData.overall = data.status;
    return healthData;
  }
  return data;
});

// Jobs API
export const createJob = (jobspec, opts = {}) => {
  const key = opts.idempotencyKey || computeIdempotencyKey(jobspec);
  
  // Debug logging to identify JSON serialization issues
  console.log('Creating job with payload:', jobspec);
  
  // Safeguard: ensure questions are present for bias-detection v1 jobspecs.
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
      // Minimal defaults if nothing selected
      if (!selected || selected.length === 0) {
        selected = ['identity_basic'];
        console.warn('[Beacon] No questions found; injecting minimal default questions:', selected);
      } else {
        console.info('[Beacon] Injecting selected questions into JobSpec:', selected.length);
      }
      jobspec = { ...jobspec, questions: selected };
    }
    // Final guard: if still missing, fail fast with clear message
    if (isV1 && benchName.includes('bias')) {
      const ok = Array.isArray(jobspec?.questions) && jobspec.questions.length > 0;
      if (!ok) {
        throw new Error('questions are required for bias-detection; please select at least one on the Questions page');
      }
    }
  } catch (e) {
    console.warn('[Beacon] questions injection skipped:', e?.message || String(e));
  }
  
  let bodyString;
  try {
    bodyString = JSON.stringify(jobspec);
    console.log('Serialized JSON:', bodyString);
  } catch (error) {
    console.error('JSON serialization failed:', error);
    throw new Error(`Failed to serialize job payload: ${error.message}`);
  }
  // Conditionally include Idempotency-Key to accommodate runners without idempotency support
  const headers = {};
  const enableIdem = opts.forceIdempotency === true || (opts.forceIdempotency !== false && shouldSendIdempotency());
  if (enableIdem) {
    headers['Idempotency-Key'] = key;
  }

  return httpV1('/jobs', {
    method: 'POST',
    headers,
    body: bodyString,
  });
};

// Export helper if callers want to pre-generate their own stable keys
export const getIdempotencyKeyForJob = computeIdempotencyKey;

export const getJob = ({ id, include, exec_limit, exec_offset }) => {
  const params = new URLSearchParams();
  if (include) params.set('include', include);
  if (exec_limit != null) params.set('exec_limit', String(exec_limit));
  if (exec_offset != null) params.set('exec_offset', String(exec_offset));
  const qs = params.toString();
  return httpV1(`/jobs/${encodeURIComponent(id)}${qs ? `?${qs}` : ''}`);
};

export const listJobs = ({ limit = 50 } = {}) => {
  const params = new URLSearchParams();
  params.set('limit', String(limit));
  return httpV1(`/jobs?${params.toString()}`).then((data) => {
    // Normalize to { jobs: [...] }
    if (Array.isArray(data)) return { jobs: data };
    return data;
  });
};

// Transparency API
export const getTransparencyRoot = () => httpV1('/transparency/root');

export const getTransparencyProof = ({ execution_id, ipfs_cid }) => {
  const params = new URLSearchParams();
  if (execution_id) params.set('execution_id', execution_id);
  if (ipfs_cid) params.set('ipfs_cid', ipfs_cid);
  return httpV1(`/transparency/proof?${params.toString()}`);
};

// Questions API (grouped by category)
export const getQuestions = async () => {
  const data = await httpV1('/questions');
  // Normalize to { categories: { [cat]: [{question_id, question}] } }
  if (data && data.categories) return data;
  // If backend returns flat array, group here
  if (Array.isArray(data)) {
    const grouped = {};
    for (const q of data) {
      const cat = q.category || 'uncategorized';
      if (!grouped[cat]) grouped[cat] = [];
      grouped[cat].push({ question_id: q.question_id, question: q.question });
    }
    return { categories: grouped };
  }
  return { categories: {} };
};

export function getIpfsGateway() {
  try {
    const override = localStorage.getItem('beacon:ipfs_gateway');
    if (override && override.trim()) return override.replace(/\/$/, '');
  } catch {}
  const envGw = import.meta?.env?.VITE_IPFS_GATEWAY;
  if (envGw && typeof envGw === 'string' && envGw.trim()) return envGw.replace(/\/$/, '');
  return null;
}

export const bundleUrl = (cid) => {
  const gw = getIpfsGateway();
  if (gw) return `${gw}/ipfs/${encodeURIComponent(cid)}`;
  // Fallback to configured API gateway
  return `${API_BASE_V1}/transparency/bundles/${encodeURIComponent(cid)}`;
};

// Legacy/placeholder exports (may be removed once pages migrate)
export const executeJob = (jobId) => httpV1(`/jobs/${jobId}/execute`, { method: 'POST' });

export const getExecutions = ({ limit = 20 } = {}) =>
  httpV1(`/executions?limit=${limit}`).then((data) => {
    // Normalize to { executions: [...] } or [] depending on consumers
    if (Array.isArray(data)) return data; // existing pages expect an array
    if (data && Array.isArray(data.executions)) return data.executions;
    return [];
  });

export const getDiffs = ({ limit = 20 } = {}) =>
  httpV1(`/diffs?limit=${limit}`).then((data) => {
    // Normalize to array
    if (Array.isArray(data)) return data;
    if (data && Array.isArray(data.diffs)) return data.diffs;
    return [];
  });

// Geo API: country counts
export const getGeo = async () => {
  const data = await httpV1('/geo');
  if (data && data.countries) return data;
  return { countries: {} };
};

// Hybrid router helpers via Netlify proxy (/hybrid/*) with fallback to direct Railway URL
let HYBRID_BASE = 'https://project-beacon-production.up.railway.app';
try {
  const envHybrid = import.meta?.env?.VITE_HYBRID_BASE;
  if (envHybrid && typeof envHybrid === 'string' && envHybrid.trim()) {
    HYBRID_BASE = envHybrid.replace(/\/$/, '');
  }
} catch {}

async function httpHybrid(path, opts = {}) {
  const url = `/hybrid${path.startsWith('/') ? path : '/' + path}`;
  try {
    const res = await fetch(url, { ...opts, headers: { 'Accept': 'application/json', ...(opts.headers || {}) } });
    if (res.ok) return res.status === 204 ? null : res.json();
    // Fallback to direct Railway if proxy not configured yet (404/502/etc.)
    const res2 = await fetch(`${HYBRID_BASE}${path.startsWith('/') ? path : '/' + path}`, { ...opts, headers: { 'Accept': 'application/json', ...(opts.headers || {}) } });
    if (!res2.ok) throw new Error(`${res2.status} ${res2.statusText}`);
    return res2.status === 204 ? null : res2.json();
  } catch (err) {
    console.warn(`[Hybrid] request failed: ${url}`, err.message);
    throw err;
  }
}

export const getHybridHealth = () => httpHybrid('/health');
export const getHybridProviders = () => httpHybrid('/providers').then(d => Array.isArray(d?.providers) ? d.providers : []);
