#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const dist = path.join(__dirname, '..', 'dist');
const redirectsPath = path.join(dist, '_redirects');

fs.mkdirSync(dist, { recursive: true });

// Helper: try to read a JSON file, returning null on failure
function readJson(filePath) {
  try {
    if (!fs.existsSync(filePath)) return null;
    const txt = fs.readFileSync(filePath, 'utf8');
    return JSON.parse(txt);
  } catch (e) {
    console.warn(`[write-redirects] Failed to read JSON from ${filePath}:`, e?.message || String(e));
    return null;
  }
}

// Derive upstream targets
let REDIRECT_DIFFS_BASE = process.env.REDIRECT_DIFFS_BASE || 'https://backend-diffs-production.up.railway.app';
let REDIRECT_RUNNER_BASE = process.env.REDIRECT_RUNNER_BASE || '';
let REDIRECT_HYBRID_BASE = process.env.REDIRECT_HYBRID_BASE || '';

// Attempt to source from facts.json when env vars are missing or placeholders
const factsPath = path.join(__dirname, '..', 'docs', 'sot', 'facts.json');
const facts = readJson(factsPath);

function urlLooksPlaceholder(url) {
  try { return /change-me/i.test(String(url || '')); } catch { return false; }
}

function deriveFromFacts(subject) {
  if (!Array.isArray(facts)) return '';
  // Prefer explicit server entries for prod
  const srv = facts.find(f => f.type === 'server' && f.subject === subject && f.env === 'prod' && f.data && typeof f.data.url === 'string');
  if (srv && srv.data.url) return srv.data.url;
  // Fallback to api_base entries
  const api = facts.find(f => f.type === 'api_base' && f.subject === subject && f.env === 'prod' && f.data && typeof f.data.base_url === 'string');
  if (api && api.data.base_url) return api.data.base_url;
  // Fallback to routing_configuration for known paths
  if (subject === 'runner') {
    const rc = facts.find(f => f.type === 'routing_configuration' && f.subject === 'netlify_redirects' && f.data && f.data.route_mapping && f.data.route_mapping['/api/v1/*']);
    return rc?.data?.route_mapping['/api/v1/*']?.url || '';
  }
  if (subject === 'router') {
    const rc = facts.find(f => f.type === 'routing_configuration' && f.subject === 'netlify_redirects' && f.data && f.data.route_mapping && f.data.route_mapping['/hybrid/*']);
    return rc?.data?.route_mapping['/hybrid/*']?.url || '';
  }
  return '';
}

if (!REDIRECT_RUNNER_BASE) {
  const candidate = deriveFromFacts('runner');
  if (candidate) REDIRECT_RUNNER_BASE = candidate;
}
if (!REDIRECT_HYBRID_BASE) {
  const candidate = deriveFromFacts('router');
  if (candidate) REDIRECT_HYBRID_BASE = candidate;
}

// Final fallbacks if still empty
if (!REDIRECT_RUNNER_BASE) REDIRECT_RUNNER_BASE = 'https://beacon-runner-production.fly.dev';
if (!REDIRECT_HYBRID_BASE) REDIRECT_HYBRID_BASE = 'https://project-beacon-production.up.railway.app';

// Validation warnings
if (urlLooksPlaceholder(REDIRECT_RUNNER_BASE)) {
  console.warn('[write-redirects] WARNING: REDIRECT_RUNNER_BASE appears to be a placeholder:', REDIRECT_RUNNER_BASE);
  console.warn('  Update Netlify env REDIRECT_RUNNER_BASE or docs/sot/facts.json (server:runner env=prod).');
}

// Complete redirects file - ORDER MATTERS (first match wins)
const lines = [
  // 1. Backend diffs proxies (force)
  `/backend-diffs/health ${REDIRECT_DIFFS_BASE}/health 200!`,
  `/backend-diffs/* ${REDIRECT_DIFFS_BASE}/:splat 200!`,
  
  // 2. API and health proxies (force)
  `/api/v2/* ${REDIRECT_RUNNER_BASE}/api/v2/:splat 200!`,
  `/api/v1/* ${REDIRECT_RUNNER_BASE}/api/v1/:splat 200!`,
  `/health ${REDIRECT_RUNNER_BASE}/health 200!`,
  `/hybrid/* ${REDIRECT_HYBRID_BASE}/:splat 200!`,
  
  // 3. Google Maps API proxy (force) - hybrid router injects API key
  `/maps/* ${REDIRECT_HYBRID_BASE}/maps/:splat 200!`,
  
  // 4. WebSocket proxy (force)
  `/ws ${REDIRECT_HYBRID_BASE}/ws 200!`,
  `/ws/* ${REDIRECT_HYBRID_BASE}/ws/:splat 200!`,
  
  // 5. Docs SPA fallback
  '/docs/* /docs/index.html 200',
  '/docs /docs/index.html 200',
  
  // 6. Portal asset passthrough (before SPA catch-all)
  '/portal/assets/* /portal/assets/:splat 200',
  
  // 7. Portal SPA fallback
  '/portal/* /portal/index.html 200',
  '/portal /portal/index.html 200',
  
  // 8. Demo results passthrough
  '/demo-results/* /demo-results/:splat 200',
];

try {
  const content = lines.join('\n');
  fs.writeFileSync(redirectsPath, content, 'utf8');
  console.log(`[write-redirects] Wrote ${redirectsPath}`);
  console.log(`[write-redirects] Using upstream targets:`);
  console.log(`  REDIRECT_DIFFS_BASE: ${REDIRECT_DIFFS_BASE}`);
  console.log(`  REDIRECT_RUNNER_BASE: ${REDIRECT_RUNNER_BASE}`);
  console.log(`  REDIRECT_HYBRID_BASE: ${REDIRECT_HYBRID_BASE}`);
  if (facts) {
    console.log(`[write-redirects] Derived from facts.json: ${factsPath}`);
  }
  console.log(`[write-redirects] Generated _redirects content:`);
  console.log('--- dist/_redirects ---');
  console.log(content);
  console.log('--- end _redirects ---');
} catch (e) {
  console.error('[write-redirects] Failed to write _redirects:', e);
  process.exit(1);
}
