#!/usr/bin/env node
import fs from 'node:fs/promises';
import { spawnSync } from 'node:child_process';

function parseArgs(argv) {
  const args = { file: 'docs/sot/facts.json', compareRef: null };
  for (let i = 2; i < argv.length; i++) {
    const a = argv[i];
    if (a === '--file' && argv[i + 1]) {
      args.file = argv[++i];
    } else if (a === '--compare-ref' && argv[i + 1]) {
      args.compareRef = argv[++i];
    } else if (a === '--help' || a === '-h') {
      console.log('Usage: node scripts/sot-validate.mjs --file <path> [--compare-ref <git-ref>]');
      process.exit(0);
    }
  }
  return args;
}

function isObject(v) {
  return v !== null && typeof v === 'object' && !Array.isArray(v);
}

function deepEqual(a, b) {
  if (a === b) return true;
  if (Array.isArray(a) && Array.isArray(b)) {
    if (a.length !== b.length) return false;
    for (let i = 0; i < a.length; i++) {
      if (!deepEqual(a[i], b[i])) return false;
    }
    return true;
  }
  if (isObject(a) && isObject(b)) {
    const ak = Object.keys(a).sort();
    const bk = Object.keys(b).sort();
    if (ak.length !== bk.length) return false;
    for (let i = 0; i < ak.length; i++) {
      if (ak[i] !== bk[i]) return false;
      if (!deepEqual(a[ak[i]], b[bk[i]])) return false;
    }
    return true;
  }
  // handle NaN edge
  if (typeof a === 'number' && typeof b === 'number') {
    return Number.isNaN(a) && Number.isNaN(b);
  }
  return false;
}

function isValidISO8601(s) {
  if (typeof s !== 'string') return false;
  const d = new Date(s);
  return !Number.isNaN(d.getTime());
}

async function readJson(file) {
  const txt = await fs.readFile(file, 'utf8');
  try {
    return JSON.parse(txt);
  } catch (e) {
    throw new Error(`Invalid JSON in ${file}: ${e.message}`);
  }
}

function readJsonFromGit(ref, file) {
  const res = spawnSync('git', ['show', `${ref}:${file}`], { encoding: 'utf8' });
  if (res.status !== 0) {
    return null; // No previous version available
  }
  try {
    return JSON.parse(res.stdout);
  } catch (e) {
    throw new Error(`Previous version at ${ref}:${file} is not valid JSON: ${e.message}`);
  }
}

function validateSchema(arr, file) {
  const errors = [];
  if (!Array.isArray(arr)) {
    errors.push(`Root of ${file} must be a JSON array`);
    return errors;
  }
  const ids = new Set();
  const allowedActions = new Set(['add', 'deprecate']);

  arr.forEach((e, idx) => {
    const where = `${file}[${idx}]`;
    if (!isObject(e)) {
      errors.push(`${where}: entry must be an object`);
      return;
    }
    if (typeof e.id !== 'string' || !e.id.trim()) errors.push(`${where}: missing non-empty string 'id'`);
    if (typeof e.action !== 'string' || !allowedActions.has(e.action)) errors.push(`${where}: 'action' must be one of ${Array.from(allowedActions).join(', ')}`);
    if (typeof e.type !== 'string' || !e.type.trim()) errors.push(`${where}: missing non-empty string 'type'`);
    if (typeof e.subject !== 'string' || !e.subject.trim()) errors.push(`${where}: missing non-empty string 'subject'`);
    if (typeof e.env !== 'string' || !e.env.trim()) errors.push(`${where}: missing non-empty string 'env'`);
    if (!isObject(e.data)) errors.push(`${where}: 'data' must be an object`);
    if (!isValidISO8601(e.effective_at)) errors.push(`${where}: 'effective_at' must be ISO-8601 datetime string`);

    if (typeof e.id === 'string') {
      if (ids.has(e.id)) errors.push(`${where}: duplicate id '${e.id}'`);
      ids.add(e.id);
    }
  });

  return errors;
}

function validateAppendOnly(prevArr, currArr, file) {
  if (!prevArr) return [];
  const errors = [];
  if (!Array.isArray(prevArr)) {
    errors.push(`Previous version is not an array – cannot enforce append-only`);
    return errors;
  }
  if (prevArr.length > currArr.length) {
    errors.push(`Append-only violation: previous length ${prevArr.length} > current length ${currArr.length}`);
    return errors;
  }
  for (let i = 0; i < prevArr.length; i++) {
    if (!deepEqual(prevArr[i], currArr[i])) {
      errors.push(`Append-only violation at index ${i}: previous entry changed`);
      break;
    }
  }
  return errors;
}

async function main() {
  const { file, compareRef } = parseArgs(process.argv);
  try {
    const curr = await readJson(file);
    const schemaErrors = validateSchema(curr, file);
    if (schemaErrors.length) {
      console.error('Schema validation failed:');
      for (const e of schemaErrors) console.error(' -', e);
      process.exit(1);
    }

    if (compareRef) {
      const prev = readJsonFromGit(compareRef, file);
      if (prev) {
        const appendErrors = validateAppendOnly(prev, curr, file);
        if (appendErrors.length) {
          console.error(`Append-only validation failed against ${compareRef}:`);
          for (const e of appendErrors) console.error(' -', e);
          process.exit(1);
        }
      } else {
        console.log(`No previous ${file} at ${compareRef}; skipping append-only check.`);
      }
    }

    console.log('SoT validation OK');
  } catch (err) {
    console.error(err.message || err);
    process.exit(1);
  }
}

main();
