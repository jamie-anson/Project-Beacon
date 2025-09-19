#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const dist = path.join(__dirname, '..', 'dist');
const redirectsPath = path.join(dist, '_redirects');

fs.mkdirSync(dist, { recursive: true });

// Minimal redirects to ensure proxy works, even if netlify.toml is ignored
const lines = [
  // Diffs backend proxy
  '/backend-diffs/* https://backend-diffs-production.up.railway.app/:splat 200!\n',
];

try {
  fs.writeFileSync(redirectsPath, lines.join(''), 'utf8');
  console.log(`[write-redirects] Wrote ${redirectsPath}`);
} catch (e) {
  console.error('[write-redirects] Failed to write _redirects:', e);
  process.exit(1);
}
