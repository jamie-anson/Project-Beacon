#!/usr/bin/env node
/**
 * postbuild-pin.js
 *
 * Purpose: Produce a docs build CID for downstream embedding.
 * For now, compute a deterministic hash over dist/docs and write it to dist/docs-cid.txt.
 * This is NOT a real IPFS CID, but a stable placeholder derived from content.
 * CI/Netlify can later replace this with a real IPFS pin step.
 */

const fs = require('fs');
const path = require('path');
const crypto = require('crypto');

const DIST_DIR = path.join(__dirname, '..', 'dist');
const DOCS_DIR = path.join(DIST_DIR, 'docs');
const CID_FILE = path.join(DIST_DIR, 'docs-cid.txt');

function hashFile(fp) {
  const h = crypto.createHash('sha256');
  const buf = fs.readFileSync(fp);
  h.update(buf);
  return h.digest('hex');
}

function walk(dir) {
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  let files = [];
  for (const e of entries) {
    const p = path.join(dir, e.name);
    if (e.isDirectory()) files = files.concat(walk(p));
    else if (e.isFile()) files.push(p);
  }
  return files.sort();
}

function computePseudoCid(root) {
  if (!fs.existsSync(root)) return 'bafy...placeholder';
  const files = walk(root);
  const h = crypto.createHash('sha256');
  for (const f of files) {
    h.update(f.replace(root, '')); // stable path component
    h.update('\0');
    h.update(hashFile(f));
    h.update('\n');
  }
  const hex = h.digest('hex');
  // Produce a stable, recognizable placeholder that looks CID-like
  return 'bafy' + hex.slice(0, 56);
}

function main() {
  if (!fs.existsSync(DIST_DIR)) {
    fs.mkdirSync(DIST_DIR, { recursive: true });
  }
  const cid = computePseudoCid(DOCS_DIR);
  fs.writeFileSync(CID_FILE, cid + '\n');
  console.log(`[postbuild-pin] Wrote ${CID_FILE} with CID: ${cid}`);
}

try {
  main();
} catch (err) {
  console.error('[postbuild-pin] Error:', err && (err.stack || err.message || err));
  process.exit(1);
}
