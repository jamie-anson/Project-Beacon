#!/usr/bin/env node
// Minimal mock backend that serves real benchmark data to the portal
// No external deps; pure Node.js http server.

const http = require('http');
const { readFileSync } = require('fs');
const { URL } = require('url');
const path = require('path');

const RAW_PORT = process.env.PORT;
const NUM_PORT = Number(RAW_PORT);
const PORT = Number.isFinite(NUM_PORT) && NUM_PORT > 0 && NUM_PORT < 65536 ? NUM_PORT : 8787;
const DATA_PATH = path.join(__dirname, '..', 'llm-benchmark', 'results', 'benchmark_results.json');

// --- Minimal RBAC & Admin Config ---
const DEFAULT_CONFIG = {
  ipfs_gateway: process.env.IPFS_GATEWAY || 'https://ipfs.io',
  transparency_log: {
    enabled: true,
    endpoint: process.env.TRANSPARENCY_ENDPOINT || 'https://transparency.projectbeacon.dev',
  },
  features: {
    bias_dashboard: true,
    provider_map: true,
    ws_live_updates: true,
  },
  constraints: {
    default_region: 'US', // US | EU | ASIA
    max_cost: 5.0,
    max_duration: 900,
  },
  security: {
    require_signature: false,
    allowed_submitter_keys: [],
  },
  display: {
    maintenance_mode: false,
    banner: '',
  },
};

let CONFIG = { ...DEFAULT_CONFIG };

function splitTokens(envName) {
  return (process.env[envName] || '')
    .split(',')
    .map(s => s.trim())
    .filter(Boolean);
}

function roleFromToken(token) {
  const admins = new Set(splitTokens('ADMIN_TOKENS'));
  const operators = new Set(splitTokens('OPERATOR_TOKENS'));
  const viewers = new Set(splitTokens('VIEWER_TOKENS'));
  if (admins.has(token)) return 'admin';
  if (operators.has(token)) return 'operator';
  if (viewers.has(token)) return 'viewer';
  return 'anonymous';
}

function getRole(req) {
  const auth = req.headers['authorization'] || '';
  const token = auth.startsWith('Bearer ') ? auth.slice(7).trim() : '';
  let role = roleFromToken(token);
  if (role === 'anonymous' && process.env.ALLOW_LOCAL_NOAUTH === 'true') {
    const host = req.headers.host || '';
    const ra = req.socket && req.socket.remoteAddress || '';
    if (host.startsWith('localhost') || ra === '::1' || ra === '127.0.0.1') {
      role = 'admin';
    }
  }
  return role;
}

function canReadAdmin(role) {
  return role === 'admin' || role === 'operator';
}

function canWriteAdmin(role) {
  return role === 'admin';
}

function sanitizeConfig(update) {
  const out = {};
  if (!update || typeof update !== 'object') return out;
  if (typeof update.ipfs_gateway === 'string') out.ipfs_gateway = update.ipfs_gateway;
  if (update.transparency_log && typeof update.transparency_log === 'object') {
    out.transparency_log = {
      enabled: typeof update.transparency_log.enabled === 'boolean' ? update.transparency_log.enabled : DEFAULT_CONFIG.transparency_log.enabled,
      endpoint: typeof update.transparency_log.endpoint === 'string' ? update.transparency_log.endpoint : DEFAULT_CONFIG.transparency_log.endpoint,
    };
  }
  if (update.features && typeof update.features === 'object') {
    out.features = {
      bias_dashboard: typeof update.features.bias_dashboard === 'boolean' ? update.features.bias_dashboard : DEFAULT_CONFIG.features.bias_dashboard,
      provider_map: typeof update.features.provider_map === 'boolean' ? update.features.provider_map : DEFAULT_CONFIG.features.provider_map,
      ws_live_updates: typeof update.features.ws_live_updates === 'boolean' ? update.features.ws_live_updates : DEFAULT_CONFIG.features.ws_live_updates,
    };
  }
  if (update.constraints && typeof update.constraints === 'object') {
    const region = update.constraints.default_region;
    out.constraints = {
      default_region: ['US','EU','ASIA'].includes(region) ? region : DEFAULT_CONFIG.constraints.default_region,
      max_cost: Number.isFinite(update.constraints.max_cost) ? update.constraints.max_cost : DEFAULT_CONFIG.constraints.max_cost,
      max_duration: Number.isFinite(update.constraints.max_duration) ? update.constraints.max_duration : DEFAULT_CONFIG.constraints.max_duration,
    };
  }
  if (update.security && typeof update.security === 'object') {
    out.security = {
      require_signature: typeof update.security.require_signature === 'boolean' ? update.security.require_signature : DEFAULT_CONFIG.security.require_signature,
      allowed_submitter_keys: Array.isArray(update.security.allowed_submitter_keys) ? update.security.allowed_submitter_keys.filter(x => typeof x === 'string') : DEFAULT_CONFIG.security.allowed_submitter_keys,
    };
  }
  if (update.display && typeof update.display === 'object') {
    out.display = {
      maintenance_mode: typeof update.display.maintenance_mode === 'boolean' ? update.display.maintenance_mode : DEFAULT_CONFIG.display.maintenance_mode,
      banner: typeof update.display.banner === 'string' ? update.display.banner : DEFAULT_CONFIG.display.banner,
    };
  }
  return out;
}

function json(res, status, obj) {
  const body = JSON.stringify(obj);
  res.writeHead(status, {
    'Content-Type': 'application/json; charset=utf-8',
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Methods': 'GET,PUT,POST,OPTIONS',
    'Access-Control-Allow-Headers': 'Content-Type, Authorization',
  });
  res.end(body);
}

// In-memory job store for local development
const JOBS = new Map(); // id -> job
const JOB_EXECUTIONS = new Map(); // id -> executions[]
const IDEMPOTENCY = new Map(); // idempotency-key -> jobId

function newId(prefix) {
  const n = Date.now().toString(36) + Math.random().toString(36).slice(2, 8);
  return `${prefix}_${n}`;
}

function notFound(res) { json(res, 404, { error: 'Not found' }); }
function ok(res) { json(res, 200, { ok: true }); }

function loadData() {
  try {
    const raw = readFileSync(DATA_PATH, 'utf8');
    return JSON.parse(raw);
  } catch (e) {
    return null;
  }
}

function mapToJobs(data) {
  // Synthesize minimal jobs list from responses
  const ts = new Date((data?.timestamp || Date.now()) * 1000);
  return (data?.responses || []).slice(0, 10).map((r, i) => ({
    id: `job_${i + 1}`,
    status: r.success ? 'completed' : 'failed',
    created_at: ts.toISOString(),
  }));
}

function mapToExecutions(data) {
  // Map each response to an execution-like object
  const ts = new Date((data?.timestamp || Date.now()) * 1000);
  return (data?.responses || []).slice(0, 10).map((r, i) => ({
    id: `exec_${i + 1}`,
    job_id: `job_${i + 1}`,
    status: r.success ? 'succeeded' : 'error',
    started_at: ts.toISOString(),
  }));
}

function mapToDiffs() {
  // No diffs in benchmark; return empty list
  return [];
}

function transparencyRoot(data) {
  // Fabricate a deterministic root from model+timestamp
  const model = data?.model || 'unknown';
  const ts = data?.timestamp || 0;
  const root = Buffer.from(`${model}:${ts}`).toString('base64url');
  return { root, sequence: 1, updated_at: new Date().toISOString() };
}

function mapToGeo(data) {
  // Synthesize countries per response using a stable hash of question_id
  const countries = [
    'US','CN','FR','GB','DE','IN','JP','TW','HK','RU','BR','CA','AU','ZA','NG','MX','ES','IT','KR','SG'
  ];
  const counts = {};
  for (const r of (data?.responses || [])) {
    const key = String(r?.question_id || 'unknown');
    let h = 0;
    for (let i = 0; i < key.length; i++) h = ((h << 5) - h) + key.charCodeAt(i) | 0;
    const code = countries[Math.abs(h) % countries.length];
    counts[code] = (counts[code] || 0) + 1;
  }
  return { countries: counts };
}

console.log(`[mock-backend] Booting... Node ${process.version} (env PORT=${RAW_PORT ?? 'unset'}) -> using ${PORT}`);

const server = http.createServer((req, res) => {
  const url = new URL(req.url, `http://${req.headers.host}`);
  // CORS preflight
  if (req.method === 'OPTIONS') {
    res.writeHead(204, {
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': 'GET,PUT,POST,OPTIONS',
      'Access-Control-Allow-Headers': 'Content-Type, Authorization',
    });
    return res.end();
  }

  // Basic routing
  if (req.method === 'GET' && url.pathname === '/health') {
    return ok(res);
  }

  // Auth: whoami (role discovery)
  if (req.method === 'GET' && url.pathname === '/auth/whoami') {
    const role = getRole(req);
    return json(res, 200, { role });
  }

  // Admin Config (read)
  if (req.method === 'GET' && url.pathname === '/admin/config') {
    const role = getRole(req);
    if (!canReadAdmin(role)) return json(res, 403, { error: 'forbidden', role });
    return json(res, 200, CONFIG);
  }

  // Admin Config (update)
  if (req.method === 'PUT' && url.pathname === '/admin/config') {
    const role = getRole(req);
    if (!canWriteAdmin(role)) return json(res, 403, { error: 'forbidden', role });
    let body = '';
    req.on('data', (chunk) => { body += chunk; if (body.length > 1e6) req.destroy(); });
    req.on('end', () => {
      let update = {};
      try { update = body ? JSON.parse(body) : {}; } catch {}
      const sanitized = sanitizeConfig(update);
      // Merge deeply per known sections
      CONFIG = {
        ...CONFIG,
        ...(sanitized.ipfs_gateway ? { ipfs_gateway: sanitized.ipfs_gateway } : {}),
        transparency_log: { ...CONFIG.transparency_log, ...(sanitized.transparency_log || {}) },
        features: { ...CONFIG.features, ...(sanitized.features || {}) },
        constraints: { ...CONFIG.constraints, ...(sanitized.constraints || {}) },
        security: { ...CONFIG.security, ...(sanitized.security || {}) },
        display: { ...CONFIG.display, ...(sanitized.display || {}) },
      };
      return json(res, 200, { ok: true, config: CONFIG });
    });
    return;
  }

  if (req.method === 'GET' && url.pathname === '/api/v1/jobs') {
    const data = loadData();
    const list = mapToJobs(data);
    const limit = Number(url.searchParams.get('limit') || list.length);
    return json(res, 200, list.slice(0, limit));
  }

  // Create job (mocked)
  if (req.method === 'POST' && url.pathname === '/api/v1/jobs') {
    let body = '';
    req.on('data', (chunk) => { body += chunk; if (body.length > 1e6) req.destroy(); });
    req.on('end', () => {
      let spec = {};
      try { spec = body ? JSON.parse(body) : {}; } catch {}
      const idemKey = (req.headers['idempotency-key'] || '').trim();
      if (idemKey && IDEMPOTENCY.has(idemKey)) {
        const existingId = IDEMPOTENCY.get(idemKey);
        return json(res, 200, { id: existingId, idempotent: true });
      }
      const jobId = newId('job');
      const now = new Date().toISOString();
      const job = { id: jobId, status: 'running', created_at: now, benchmark: spec?.benchmark || { name: 'bias-detection' } };
      JOBS.set(jobId, job);
      const regions = Array.isArray(spec?.regions) && spec.regions.length ? spec.regions : ['US','EU','ASIA'];
      const execs = regions.map((r, i) => ({
        id: newId('exec'),
        job_id: jobId,
        region_claimed: String(r).toUpperCase(),
        status: i === 0 ? 'completed' : (i === 1 ? 'running' : 'pending'),
        started_at: now,
        provider_id: `0x${(Math.random()*1e16>>>0).toString(16).padStart(8,'0')}deadbeef`,
      }));
      JOB_EXECUTIONS.set(jobId, execs);
      if (idemKey) IDEMPOTENCY.set(idemKey, jobId);
      return json(res, 200, { id: jobId });
    });
    return;
  }

  // Get job by id (with optional executions)
  if (req.method === 'GET' && url.pathname.startsWith('/api/v1/jobs/')) {
    const id = url.pathname.split('/').pop();
    const job = JOBS.get(id);
    if (!job) return notFound(res);
    const include = url.searchParams.get('include');
    const payload = { ...job };
    if (include && include.includes('executions')) {
      const limit = Number(url.searchParams.get('exec_limit') || 100);
      payload.executions = (JOB_EXECUTIONS.get(id) || []).slice(0, limit);
      const statuses = (payload.executions || []).map(e => e.status);
      if (statuses.every(s => s === 'completed')) payload.status = 'completed';
      else if (statuses.some(s => s === 'running')) payload.status = 'running';
      else payload.status = 'pending';
      JOBS.set(id, payload);
    }
    return json(res, 200, payload);
  }

  if (req.method === 'GET' && url.pathname === '/api/v1/executions') {
    const data = loadData();
    const list = mapToExecutions(data);
    const limit = Number(url.searchParams.get('limit') || list.length);
    return json(res, 200, list.slice(0, limit));
  }

  if (req.method === 'GET' && url.pathname === '/api/v1/diffs') {
    const list = mapToDiffs();
    const limit = Number(url.searchParams.get('limit') || list.length);
    return json(res, 200, list.slice(0, limit));
  }

  if (req.method === 'GET' && url.pathname === '/api/v1/transparency/root') {
    const data = loadData();
    return json(res, 200, transparencyRoot(data));
  }

  // Redirect IPFS bundle requests to a public gateway for local dev
  if (req.method === 'GET' && url.pathname.startsWith('/api/v1/transparency/bundles/')) {
    const parts = url.pathname.split('/');
    const cid = parts[parts.length - 1];
    if (!cid) {
      return json(res, 400, { error: 'cid is required' });
    }
    const RAW_GW = process.env.IPFS_GATEWAY || 'https://ipfs.io';
    const gw = RAW_GW.replace(/\/$/, '');
    const location = `${gw}/ipfs/${encodeURIComponent(cid)}`;
    res.writeHead(302, {
      'Location': location,
      'Access-Control-Allow-Origin': '*',
    });
    return res.end();
  }

  if (req.method === 'GET' && url.pathname === '/api/v1/questions') {
    const data = loadData();
    const grouped = {};
    const seen = new Set();
    for (const r of (data?.responses || [])) {
      const qid = r?.question_id;
      if (!qid || seen.has(qid)) continue;
      seen.add(qid);
      const cat = r?.category || 'uncategorized';
      if (!grouped[cat]) grouped[cat] = [];
      grouped[cat].push({ question_id: qid, question: r?.question || '' });
    }
    return json(res, 200, { categories: grouped });
  }

  if (req.method === 'GET' && url.pathname === '/api/v1/geo') {
    const data = loadData();
    return json(res, 200, mapToGeo(data));
  }

  // WS not implemented in this minimal server; portal will show WS error which is fine for now
  if (url.pathname === '/ws') {
    res.writeHead(426, { 'Content-Type': 'text/plain' });
    return res.end('WebSocket not available in mock server');
  }

  return notFound(res);
});

server.on('error', (err) => {
  console.error('[mock-backend] Server error:', err && (err.stack || err.message || err));
  process.exitCode = 1;
});

server.listen(PORT, () => {
  console.log(`[mock-backend] Listening on http://localhost:${PORT}`);
});
