const API_BASE = '/api/v1';

async function http(path, opts = {}) {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { 'Content-Type': 'application/json', ...(opts.headers || {}) },
    ...opts,
  });
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  return res.status === 204 ? null : res.json();
}

export const getHealth = () => http('/health');

export const createJob = (jobspec) => http('/jobs', {
  method: 'POST',
  body: JSON.stringify({ jobspec }),
});

export const executeJob = (jobId) => http(`/jobs/${jobId}/execute`, { method: 'POST' });

export const getExecutions = ({ limit = 20 } = {}) => http(`/executions?limit=${limit}`);

export const getDiffs = ({ limit = 20 } = {}) => http(`/diffs?limit=${limit}`);
