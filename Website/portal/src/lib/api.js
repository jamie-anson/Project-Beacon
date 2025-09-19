// Use environment variable for API base, fallback to Fly.io runner app (Railway only has hybrid router)
// Precedence:
// 1) Explicit runtime override via localStorage 'beacon:api_base'
// 2) VITE_API_BASE (build-time)
// 3) Netlify same-origin fallback (only if no explicit base)
// 4) Default Fly runner
let __apiBase = 'https://beacon-runner-change-me.fly.dev';
let explicit = false;
// 1) Runtime override (highest precedence)
try {
  const lsBase = localStorage.getItem('beacon:api_base');
  if (lsBase && lsBase.trim()) {
    __apiBase = lsBase.trim();
    explicit = true;
  }
} catch {}
// 2) Build-time env if no runtime override
try {
  if (!explicit && import.meta.env?.VITE_API_BASE) {
    const envBase = String(import.meta.env.VITE_API_BASE).trim();
    if (envBase) {
      __apiBase = envBase;
      explicit = true;
    }
  }
} catch {}
// 3) Netlify should always use Fly.io runner (no same-origin API)
// Skip same-origin fallback for Netlify since it only hosts frontend
// Normalize: ensure no trailing slash and strip a mistakenly included "/api/v1" suffix.
try {
  __apiBase = String(__apiBase)
    .replace(/\s+/g, '')
    .replace(/\/?api\/v1\/?$/i, '')
    .replace(/\/$/, '');
} catch {}
// One-time debug in development builds to help diagnose misconfigurations
try { if (import.meta?.env?.DEV) console.info('[Beacon] API_BASE_V1 =', __apiBase, '(explicit =', explicit, ')'); } catch {}
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

// Hybrid router helpers: prefer direct Railway base to avoid Netlify 404s; allow runtime override
let HYBRID_BASE = 'https://project-beacon-production.up.railway.app';
try {
  const lsHybrid = localStorage.getItem('beacon:hybrid_base');
  if (lsHybrid && lsHybrid.trim()) HYBRID_BASE = lsHybrid.replace(/\/$/, '');
} catch {}
try {
  const envHybrid = import.meta?.env?.VITE_HYBRID_BASE;
  if (envHybrid && typeof envHybrid === 'string' && envHybrid.trim()) {
    HYBRID_BASE = envHybrid.replace(/\/$/, '');
  }
} catch {}

async function httpHybrid(path, opts = {}) {
  const direct = `${HYBRID_BASE}${path.startsWith('/') ? path : '/' + path}`;
  const proxy = `/hybrid${path.startsWith('/') ? path : '/' + path}`;
  const common = { ...opts, headers: { 'Accept': 'application/json', ...(opts.headers || {}) }, mode: 'cors', credentials: 'omit' };
  // Try direct first; on any failure (non-ok or thrown), fall back to proxy
  try {
    const r1 = await fetch(direct, common);
    if (r1.ok) return r1.status === 204 ? null : r1.json();
  } catch (err) {
    // Swallow and try proxy next (common for CORS in local/dev builds)
    try { console.warn(`[Hybrid] direct request failed, falling back to proxy: ${direct} -> ${proxy}:`, err?.message || String(err)); } catch {}
  }
  try {
    const r2 = await fetch(proxy, common);
    if (!r2.ok) throw new Error(`${r2.status} ${r2.statusText}`);
    return r2.status === 204 ? null : r2.json();
  } catch (err) {
    console.warn(`[Hybrid] proxy request failed (proxy=${proxy})`, err?.message || String(err));
    throw err;
  }
}

export const getHybridHealth = () => httpHybrid('/health');
export const getHybridProviders = () => httpHybrid('/providers').then(d => Array.isArray(d?.providers) ? d.providers : []);

// Compose infrastructure health from available hybrid endpoints and normalize
export const getInfrastructureHealth = async () => {
  const [healthRes, providersRes] = await Promise.allSettled([
    httpHybrid('/health'),
    httpHybrid('/providers'),
  ]);

  const health = healthRes.status === 'fulfilled' ? healthRes.value : null;
  const providersArr = providersRes.status === 'fulfilled' && Array.isArray(providersRes.value?.providers)
    ? providersRes.value.providers
    : [];

  const providersHealthy = providersArr.filter(p => p?.healthy).length;
  const providersTotal = providersArr.length;
  const derivedOverall = providersTotal === 0
    ? (health?.status || 'unknown')
    : (providersHealthy === providersTotal ? 'healthy' : (providersHealthy > 0 ? 'degraded' : 'down'));
  const overall_status = String(health?.status || derivedOverall || 'unknown').toLowerCase();

  const services = {
    router: {
      status: overall_status,
      response_time_ms: null,
      error: healthRes.status === 'rejected' ? (healthRes.reason?.message || String(healthRes.reason || '')) : null,
    },
  };

  for (const p of providersArr) {
    const key = `${String(p.type || 'provider')}_${String(p.region || 'unknown')}`;
    services[key] = {
      status: p.healthy ? 'healthy' : 'down',
      response_time_ms: Number.isFinite(p?.avg_latency) ? Math.round(Number(p.avg_latency) * 1000) : null,
      error: null,
    };
  }

  const vals = Object.values(services);
  const healthy_services = vals.filter(s => s.status === 'healthy').length;
  const degraded_services = vals.filter(s => s.status === 'degraded').length;
  const down_services = vals.filter(s => s.status === 'down').length;

  return {
    overall_status,
    services,
    healthy_services,
    degraded_services,
    down_services,
    total_services: Object.keys(services).length,
    last_checked: new Date().toISOString(),
  };
};

// Individual execution and receipt APIs
export const getExecution = (id) => httpV1(`/executions/${encodeURIComponent(id)}`);
export const getExecutionReceipt = (id) => httpV1(`/executions/${encodeURIComponent(id)}/receipt`);

// Cross-region diff APIs
// Cross-region diff APIs (robust):
// Some backends expose GET (fetch existing) and/or POST (generate) for the same route.
export const getCrossRegionDiff = async (jobId) => {
  const id = encodeURIComponent(jobId);
  const candidates = [
    // Plan-aligned endpoints (prefer these)
    { path: `/executions/${id}/cross-region`, method: 'GET' },
    { path: `/executions/${id}/cross-region`, method: 'POST' },
    { path: `/executions/${id}/diff-analysis`, method: 'GET' },
    { path: `/executions/${id}/diff-analysis`, method: 'POST' },
    // Legacy/older experimental endpoint
    { path: `/executions/${id}/cross-region-diff`, method: 'GET' },
    { path: `/executions/${id}/cross-region-diff`, method: 'POST' },
  ];
  let lastErr;
  for (const c of candidates) {
    try {
      const res = await httpV1(c.path, c.method === 'POST' ? { method: 'POST' } : {});
      return res;
    } catch (err) {
      lastErr = err;
      // Continue to next candidate
    }
  }
  throw lastErr || new Error('cross-region diff endpoints unavailable');
};

export const createCrossRegionDiff = (jobId) => httpV1(`/executions/${encodeURIComponent(jobId)}/cross-region-diff`, { method: 'POST' });
export const createCrossRegionJob = (payload) => httpV1('/jobs/cross-region', { method: 'POST', body: JSON.stringify(payload) });
export const findDiffsByJob = async (jobId, { limit = 1 } = {}) => {
  const qs = new URLSearchParams();
  if (jobId) qs.set('job_id', jobId);
  qs.set('limit', String(limit));
  const data = await httpV1(`/diffs?${qs.toString()}`);
  if (Array.isArray(data)) return data;
  if (data && Array.isArray(data.diffs)) return data.diffs;
  return [];
};
export const getRegionResults = (jobId) => httpV1(`/executions/${encodeURIComponent(jobId)}/regions`);

// ------------------------------
// Diffs Backend (FastAPI on Railway)
// ------------------------------
let DIFFS_BASE = '';
try {
  const ls = localStorage.getItem('beacon:diffs_base');
  if (ls && ls.trim()) DIFFS_BASE = ls.replace(/\/$/, '');
} catch {}
try {
  const env = import.meta?.env?.VITE_DIFFS_BASE;
  if (!DIFFS_BASE && env && typeof env === 'string' && env.trim()) DIFFS_BASE = env.replace(/\/$/, '');
} catch {}

// If no explicit base, use same-origin proxy configured in netlify.toml
if (!DIFFS_BASE) DIFFS_BASE = '/backend-diffs';

async function httpDiffs(path, opts = {}) {
  const url = `${DIFFS_BASE}${path.startsWith('/') ? path : '/' + path}`;
  const fetchOptions = {
    ...opts,
    headers: { 'Accept': 'application/json', 'Content-Type': 'application/json; charset=utf-8', ...(opts.headers || {}) },
  };
  fetchOptions.mode = 'cors';
  fetchOptions.credentials = 'omit';
  const res = await fetch(url, fetchOptions);
  if (!res.ok) {
    let msg = `${res.status} ${res.statusText}`;
    try { const b = await res.json(); msg = b?.error || b?.message || msg; } catch {}
    throw new Error(msg);
  }
  return res.status === 204 ? null : res.json();
}

// Compare two region outputs
export const compareDiffs = ({ a, b, algorithm = 'simple' }) =>
  httpDiffs('/api/v1/diffs/compare', { method: 'POST', body: JSON.stringify({ a, b, algorithm }) });

// List recent diff results from backend
export const listRecentDiffs = ({ limit = 10 } = {}) =>
  httpDiffs(`/api/v1/diffs/recent?limit=${encodeURIComponent(String(limit))}`);
