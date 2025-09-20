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
const lines = txt.split(/\r?\n/).map((l) => l.trim()).filter(Boolean);

// Required rules in expected order
const requiredRules = [
  // Backend diffs proxies
  { pattern: /^\/backend-diffs\/health\s+.+\s+200!$/, name: 'backend-diffs health proxy' },
  { pattern: /^\/backend-diffs\/\*\s+.+\s+200!$/, name: 'backend-diffs splat proxy' },
  
  // API and health proxies
  { pattern: /^\/api\/v1\/\*\s+.+\s+200!$/, name: 'API v1 proxy' },
  { pattern: /^\/health\s+.+\s+200!$/, name: 'health proxy' },
  { pattern: /^\/hybrid\/\*\s+.+\s+200!$/, name: 'hybrid router proxy' },
  
  // WebSocket proxies
  { pattern: /^\/ws\s+.+\s+200!$/, name: 'WebSocket proxy' },
  { pattern: /^\/ws\/\*\s+.+\s+200!$/, name: 'WebSocket splat proxy' },
  
  // Docs SPA
  { pattern: /^\/docs\/\*\s+\/docs\/index\.html\s+200$/, name: 'docs SPA splat' },
  { pattern: /^\/docs\s+\/docs\/index\.html\s+200$/, name: 'docs SPA root' },
  
  // Portal assets (must come before portal SPA)
  { pattern: /^\/portal\/assets\/\*\s+\/portal\/assets\/:splat\s+200$/, name: 'portal assets passthrough' },
  
  // Portal SPA
  { pattern: /^\/portal\/\*\s+\/portal\/index\.html\s+200$/, name: 'portal SPA splat' },
  { pattern: /^\/portal\s+\/portal\/index\.html\s+200$/, name: 'portal SPA root' },
  
  // Demo results
  { pattern: /^\/demo-results\/\*\s+\/demo-results\/:splat\s+200$/, name: 'demo results passthrough' },
];

const errors = [];

// Check each required rule exists
for (const rule of requiredRules) {
  const found = lines.some(line => rule.pattern.test(line));
  if (!found) {
    errors.push(`Missing required rule: ${rule.name}`);
  }
}

// Check critical ordering: portal assets must come before portal SPA
const portalAssetsIndex = lines.findIndex(line => /^\/portal\/assets\/\*/.test(line));
const portalSpaIndex = lines.findIndex(line => /^\/portal\/\*/.test(line));

if (portalAssetsIndex === -1) {
  errors.push('Portal assets rule not found');
} else if (portalSpaIndex === -1) {
  errors.push('Portal SPA rule not found');
} else if (portalAssetsIndex > portalSpaIndex) {
  errors.push('Portal assets rule must come before portal SPA rule (order violation)');
}

if (errors.length > 0) {
  console.error('[verify-redirects] Validation failed:');
  errors.forEach(error => console.error(`  - ${error}`));
  console.error('--- dist/_redirects ---');
  console.error(txt);
  console.error('--- end _redirects ---');
  process.exit(3);
}

console.log('[verify-redirects] All required redirect rules present and correctly ordered.');
console.log(`[verify-redirects] Verified ${lines.length} redirect rules.`);
