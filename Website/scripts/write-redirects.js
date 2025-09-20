#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const dist = path.join(__dirname, '..', 'dist');
const redirectsPath = path.join(dist, '_redirects');

fs.mkdirSync(dist, { recursive: true });

// Minimal redirects to ensure proxy works, even if netlify.toml is ignored
const lines = [
  // Explicit health route (helps diagnose if splat rule is ignored)
  '/backend-diffs/health https://backend-diffs-production.up.railway.app/health 200!',
  // Diffs backend proxy (general)
  '/backend-diffs/* https://backend-diffs-production.up.railway.app/:splat 200!',
  // Sanity check: this path should 302 to Netlify home if redirects are active
  // '/redirect-test https://www.netlify.com 302',
];

try {
  fs.writeFileSync(redirectsPath, lines.join('\n'), 'utf8');
  console.log(`[write-redirects] Wrote ${redirectsPath}`);
} catch (e) {
  console.error('[write-redirects] Failed to write _redirects:', e);
  process.exit(1);
}
