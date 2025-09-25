#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const dist = path.join(__dirname, '..', 'dist');
const redirectsPath = path.join(dist, '_redirects');

fs.mkdirSync(dist, { recursive: true });

// Environment-parameterized upstream targets with safe defaults
const REDIRECT_DIFFS_BASE = process.env.REDIRECT_DIFFS_BASE || 'https://backend-diffs-production.up.railway.app';
const REDIRECT_RUNNER_BASE = process.env.REDIRECT_RUNNER_BASE || 'https://beacon-runner-change-me.fly.dev';
const REDIRECT_HYBRID_BASE = process.env.REDIRECT_HYBRID_BASE || 'https://project-beacon-production.up.railway.app';

// Complete redirects file - ORDER MATTERS (first match wins)
const lines = [
  // 1. Backend diffs proxies (force)
  `/backend-diffs/health ${REDIRECT_DIFFS_BASE}/health 200!`,
  `/backend-diffs/* ${REDIRECT_DIFFS_BASE}/:splat 200!`,
  
  // 2. API and health proxies (force)
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
  console.log(`[write-redirects] Generated _redirects content:`);
  console.log('--- dist/_redirects ---');
  console.log(content);
  console.log('--- end _redirects ---');
} catch (e) {
  console.error('[write-redirects] Failed to write _redirects:', e);
  process.exit(1);
}
