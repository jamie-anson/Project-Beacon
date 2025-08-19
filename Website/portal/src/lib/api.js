const API_BASE_V1 = '/api/v1';

async function httpV1(path, opts = {}) {
  const res = await fetch(`${API_BASE_V1}${path}`, {
    headers: { 'Content-Type': 'application/json', ...(opts.headers || {}) },
    ...opts,
  });
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  return res.status === 204 ? null : res.json();
}

async function httpRoot(path, opts = {}) {
  const res = await fetch(`${path}`, {
    headers: { 'Content-Type': 'application/json', ...(opts.headers || {}) },
    ...opts,
  });
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  return res.status === 204 ? null : res.json();
}

// Health endpoints are mounted at root (/health)
export const getHealth = () => httpRoot('/health');

// Jobs API
export const createJob = (jobspec) => httpV1('/jobs', {
  method: 'POST',
  body: JSON.stringify(jobspec),
});

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
  return httpV1(`/jobs?${params.toString()}`);
};

// Transparency API
export const getTransparencyRoot = () => httpV1('/transparency/root');

export const getTransparencyProof = ({ execution_id, ipfs_cid }) => {
  const params = new URLSearchParams();
  if (execution_id) params.set('execution_id', execution_id);
  if (ipfs_cid) params.set('ipfs_cid', ipfs_cid);
  return httpV1(`/transparency/proof?${params.toString()}`);
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
  // Fallback to local API gateway
  return `/api/v1/transparency/bundles/${encodeURIComponent(cid)}`;
};

// Legacy/placeholder exports (may be removed once pages migrate)
export const executeJob = (jobId) => httpV1(`/jobs/${jobId}/execute`, { method: 'POST' });

export const getExecutions = ({ limit = 20 } = {}) => httpV1(`/executions?limit=${limit}`);

export const getDiffs = ({ limit = 20 } = {}) => httpV1(`/diffs?limit=${limit}`);
