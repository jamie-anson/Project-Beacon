#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const portalRoot = path.resolve(__dirname, '..');
const srcDir = path.join(portalRoot, 'src');

const importPattern = /from\s+['"]([^'"]*lib\/api\.js)['"];?/g;
const requirePattern = /require\(['"]([^'"]*lib\/api\.js)['"]\)/g;

const violations = [];

function scanDirectory(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (entry.name === 'node_modules' || entry.name.startsWith('.')) continue;
    const fullPath = path.join(dir, entry.name);

    if (entry.isDirectory()) {
      scanDirectory(fullPath);
      continue;
    }

    if (!/\.(c|m)?(j|t)sx?$/.test(entry.name)) continue;
    if (fullPath.endsWith(path.join('src', 'lib', 'api.js'))) continue;

    const content = fs.readFileSync(fullPath, 'utf8');
    if (importPattern.test(content) || requirePattern.test(content)) {
      const relPath = path.relative(portalRoot, fullPath);
      violations.push(relPath);
    }
  }
}

try {
  if (!fs.existsSync(srcDir)) {
    console.error('Unable to locate portal/src for lint check.');
    process.exit(1);
  }
  scanDirectory(srcDir);

  if (violations.length > 0) {
    console.error('Legacy shim import detected. Please import from modular API files instead of `lib/api.js`');
    for (const file of violations) {
      console.error(` - ${file}`);
    }
    process.exit(1);
  }

  console.log('No legacy shim imports detected.');
} catch (err) {
  console.error('Failed to complete legacy shim lint check:', err);
  process.exit(1);
}
