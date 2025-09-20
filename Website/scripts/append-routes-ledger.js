#!/usr/bin/env node
const fs = require('fs');
const path = require('path');
const crypto = require('crypto');

const dist = path.join(__dirname, '..', 'dist');
const redirectsPath = path.join(dist, '_redirects');
const ledgerPath = path.join(__dirname, '..', 'attestations', 'routes-ledger.jsonl');

// Ensure attestations directory exists
fs.mkdirSync(path.dirname(ledgerPath), { recursive: true });

if (!fs.existsSync(redirectsPath)) {
  console.error('[append-routes-ledger] dist/_redirects not found');
  process.exit(1);
}

const content = fs.readFileSync(redirectsPath, 'utf8');
const lines = content.split(/\r?\n/).filter(Boolean);
const sha256 = crypto.createHash('sha256').update(content).digest('hex');

// Get commit SHA if available
let commit = 'unknown';
try {
  const { execSync } = require('child_process');
  commit = execSync('git rev-parse --short HEAD', { encoding: 'utf8' }).trim();
} catch (e) {
  // Git not available or not in a repo
}

const entry = {
  timestamp: new Date().toISOString(),
  commit,
  sha256,
  line_count: lines.length,
  first_lines: lines.slice(0, 3),
  last_lines: lines.slice(-3),
  note: `Routes ledger entry for ${lines.length} redirect rules`
};

// Append to ledger
fs.appendFileSync(ledgerPath, JSON.stringify(entry) + '\n');

console.log(`[append-routes-ledger] Appended entry to ${ledgerPath}`);
console.log(`[append-routes-ledger] SHA-256: ${sha256}`);
console.log(`[append-routes-ledger] Lines: ${lines.length}`);
console.log(`[append-routes-ledger] Commit: ${commit}`);
