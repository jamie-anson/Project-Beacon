#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const dist = path.join(__dirname, '..', 'dist');
const redirectsPath = path.join(dist, '_redirects');

if (!fs.existsSync(redirectsPath)) {
  console.error('[verify-redirects] dist/_redirects not found');
  process.exit(2);
}

const txt = fs.readFileSync(redirectsPath, 'utf8');
const hasHealth = /\n?\s*\/backend-diffs\/health\b/.test(txt);
const hasSplat = /\n?\s*\/backend-diffs\/\*\b/.test(txt);

if (!hasHealth || !hasSplat) {
  console.error('[verify-redirects] Missing backend-diffs rules in dist/_redirects');
  console.error('--- dist/_redirects ---');
  console.error(txt);
  process.exit(3);
}

console.log('[verify-redirects] Redirects contain backend-diffs rules.');
