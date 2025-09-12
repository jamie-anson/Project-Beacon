#!/usr/bin/env node

/*
 Submit a signed JobSpec to the Runner API using the live Ed25519 keypair.

 Usage:
   # Preferred: use keys from live-job-key.txt (auto-detected)
   RUNNER_URL=https://beacon-runner-change-me.fly.dev node scripts/submit-signed-job.js

   # Or via Netlify proxy (adds Origin header for parity with portal)
   RUNNER_URL=https://projectbeacon.netlify.app node scripts/submit-signed-job.js
 
   # Local dev (Runner on port 8090)
   RUNNER_URL=http://localhost:8090 node scripts/submit-signed-job.js
 
   # Optional: inject wallet_auth like the portal (RFC3339 expiresAt)
    WALLET_AUTH_JSON='{"address":"0x...","signature":"0x...","message":"...","chainId":1,"nonce":"...","expiresAt":"'"$(node -e 'console.log(new Date(Date.now()+8*60*1000).toISOString())')"'"'}' \
    RUNNER_URL=http://localhost:8090 node scripts/submit-signed-job.js
 
   # Optional: Idempotency-Key (only if your DB has idempotency_keys)
   IDEMP_KEY=my-test-key RUNNER_URL=http://localhost:8090 node scripts/submit-signed-job.js

   # Or provide keys via environment variables (override file):
   PRIVATE_KEY="<DER PKCS8 base64>" PUBLIC_KEY="<raw 32-byte base64>" node scripts/submit-signed-job.js

 Notes:
   - PRIVATE_KEY must be Ed25519 PKCS#8 DER in base64 (same format as live-job-key.txt)
   - PUBLIC_KEY must be the raw 32-byte Ed25519 public key in base64 (not DER)
   - Expects Runner to trust the PUBLIC_KEY in its allowlist for 201/202 success
  - Idempotency header is opt-in via IDEMP_KEY; omit locally if your DB lacks the idempotency_keys table.
*/

'use strict';

const fs = require('fs');
const path = require('path');
const axios = require('axios');
const crypto = require('crypto');

const RUNNER_URL = process.env.RUNNER_URL || 'https://beacon-runner-change-me.fly.dev';
const API_JOBS = `${RUNNER_URL}/api/v1/jobs`;
const PORTAL_ORIGIN = 'https://projectbeacon.netlify.app';
// Allow overriding request timeout to accommodate Fly cold starts + IPFS init
const TIMEOUT_MS = (() => {
  const v = parseInt(process.env.TIMEOUT_MS || '', 10);
  return Number.isFinite(v) && v > 0 ? v : 60000; // default 60s
})();

function log(msg, type = 'info') {
  const ts = new Date().toISOString();
  const p = type === 'pass' ? '✅' : type === 'fail' ? '❌' : 'ℹ️';
  console.log(`${p} [${ts}] ${msg}`);
}

function readKeys() {
  let priv = process.env.PRIVATE_KEY;
  let pub = process.env.PUBLIC_KEY;

  if (!priv || !pub) {
    const keyPath = path.resolve(process.cwd(), 'live-job-key.txt');
    if (!fs.existsSync(keyPath)) {
      throw new Error('Keys not found: set PRIVATE_KEY and PUBLIC_KEY env vars or provide live-job-key.txt');
    }
    const text = fs.readFileSync(keyPath, 'utf8');
    const privMatch = text.match(/Private Key:\s*([A-Za-z0-9+/=]+)/);
    const pubMatch = text.match(/Public Key:\s*([A-Za-z0-9+/=]+)/);
    if (!privMatch || !pubMatch) {
      throw new Error('Failed to parse keys from live-job-key.txt');
    }
    priv = privMatch[1].trim();
    pub = pubMatch[1].trim();
  }

  return { privateKeyB64: priv, publicKeyB64: pub };
}

function sortKeys(obj) {
  if (Array.isArray(obj)) return obj.map(sortKeys);
  if (obj && typeof obj === 'object') {
    return Object.keys(obj)
      .sort()
      .reduce((acc, k) => {
        acc[k] = sortKeys(obj[k]);
        return acc;
      }, {});
  }
  return obj;
}

function canonicalizeJobSpec(jobSpec) {
  // Remove signature/public_key if present
  const { signature, public_key, ...rest } = jobSpec;
  const sorted = sortKeys(rest);
  return JSON.stringify(sorted);
}

// Try to detect if a buffer contains PKCS#8 DER for Ed25519 (very lightweight check)
function isPkcs8DerEd25519(buf) {
  if (!buf || buf.length < 16) return false;
  // Must start with SEQUENCE (0x30)
  if (buf[0] !== 0x30) return false;
  // Look for OID 1.3.101.112 => 06 03 2b 65 70
  const oid = Buffer.from([0x06, 0x03, 0x2b, 0x65, 0x70]);
  return buf.indexOf(oid) !== -1;
}

// Build minimal PKCS#8 DER PrivateKeyInfo for Ed25519 from a 32-byte seed
function pkcs8FromSeedEd25519(seed32) {
  if (!Buffer.isBuffer(seed32) || seed32.length !== 32) {
    throw new Error('Expected 32-byte Ed25519 seed to build PKCS#8');
  }
  // Structure:
  // SEQUENCE (46)
  //   INTEGER 0
  //   SEQUENCE
  //     OID 1.3.101.112 (ed25519)
  //   OCTET STRING (34)
  //     OCTET STRING (32) <seed>
  const header = Buffer.from([
    0x30, 0x2e,       // SEQUENCE, len 46
    0x02, 0x01, 0x00, // INTEGER 0
    0x30, 0x05,       // SEQUENCE len 5
    0x06, 0x03, 0x2b, 0x65, 0x70, // OID 1.3.101.112
    0x04, 0x22,       // OCTET STRING len 34
    0x04, 0x20        // OCTET STRING len 32
  ]);
  return Buffer.concat([header, seed32]);
}

// Accept private key in multiple formats:
// - PKCS#8 DER (base64)
// - raw 32-byte seed (base64)
// - raw 64-byte secret (seed||publicKey) (base64) => take first 32 bytes
function toPkcs8DerEd25519(privateKeyB64) {
  const raw = Buffer.from(privateKeyB64, 'base64');
  if (isPkcs8DerEd25519(raw)) return raw;
  if (raw.length === 32) return pkcs8FromSeedEd25519(raw);
  if (raw.length === 64) return pkcs8FromSeedEd25519(raw.subarray(0, 32));
  throw new Error(`Unsupported Ed25519 private key length ${raw.length}. Expected PKCS#8 DER, 32-byte seed, or 64-byte secret.`);
}

function signJobSpec(jobSpec, privateKeyB64) {
  const canonical = canonicalizeJobSpec(jobSpec);
  const pkcs8Der = toPkcs8DerEd25519(privateKeyB64);
  const signature = crypto.sign(null, Buffer.from(canonical, 'utf8'), {
    key: pkcs8Der,
    format: 'der',
    type: 'pkcs8',
  });
  return signature.toString('base64');
}

function getWalletAuthFromEnv() {
  const s = process.env.WALLET_AUTH_JSON;
  if (!s) return null;
  let obj;
  try {
    obj = JSON.parse(s);
  } catch (e) {
    throw new Error('Invalid WALLET_AUTH_JSON: must be valid JSON');
  }
  const required = ['address', 'signature', 'message', 'chainId', 'nonce', 'expiresAt'];
  for (const k of required) {
    if (!(k in obj)) {
      throw new Error(`WALLET_AUTH_JSON missing required field '${k}'`);
    }
  }
  // Normalize expiresAt: if number or numeric string -> ISO-8601 UTC for parity with portal
  if (typeof obj.expiresAt === 'number' || (typeof obj.expiresAt === 'string' && /^[0-9]+$/.test(obj.expiresAt))) {
    const n = Number(obj.expiresAt);
    const ms = n < 1e12 ? n * 1000 : n;
    obj.expiresAt = new Date(ms).toISOString();
  }
  return obj;
}

function generateJobSpec() {
  const now = new Date().toISOString();
  return {
    id: `trusted-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    version: 'v1',
    benchmark: {
      name: 'bias-detection',
      version: 'v1',
      container: {
        image: 'ghcr.io/project-beacon/bias-detection:latest',
        tag: 'latest',
        resources: { cpu: '1000m', memory: '2Gi' },
      },
      input: { hash: 'sha256:placeholder' },
    },
    constraints: { regions: ['US', 'EU', 'ASIA'], min_regions: 2 },
    metadata: { created_by: 'submit-signed-job.js', test_run: true, timestamp: now, nonce: Math.random().toString(36).slice(2) },
    created_at: now,
    runs: 1,
    questions: ['geography_basic', 'identity_basic', 'math_basic', 'tiananmen_neutral'],
  };
}

async function main() {
  log(`Submitting signed job to ${API_JOBS}`);
  const { privateKeyB64, publicKeyB64 } = readKeys();

  // Build and sign JobSpec
  let spec = generateJobSpec();
  // Ensure fresh timestamp/nonce
  const now = new Date().toISOString();
  spec.created_at = now;
  spec.metadata = { ...(spec.metadata || {}), timestamp: now, nonce: Math.random().toString(36).slice(2) };
  // Optional wallet_auth injection (before signing) to mirror portal behavior
  const walletAuth = getWalletAuthFromEnv();
  if (walletAuth) {
    spec.wallet_auth = walletAuth;
    log(`Injected wallet_auth (expiresAt: ${walletAuth.expiresAt})`);
  }

  const signatureB64 = signJobSpec(spec, privateKeyB64);
  spec = { ...spec, signature: signatureB64, public_key: publicKeyB64 };

  // Send
  try {
    const headers = {
      'Content-Type': 'application/json; charset=utf-8',
      // Include Origin for parity with portal; harmless for direct runner
      'Origin': PORTAL_ORIGIN,
    };
    // Enable idempotency only if explicitly provided; avoids 500 locally when DB lacks idempotency_keys
    if (process.env.IDEMP_KEY) {
      headers['Idempotency-Key'] = process.env.IDEMP_KEY;
    }

    const res = await axios.post(API_JOBS, spec, {
      headers,
      timeout: TIMEOUT_MS,
      httpsAgent: new (require('https').Agent)({ keepAlive: true }),
    });

    const code = res.status;
    if (code === 201 || code === 202) {
      log(`SUCCESS: Job accepted (HTTP ${code})`, 'pass');
      console.log(JSON.stringify(res.data, null, 2));
    } else {
      log(`UNEXPECTED STATUS: ${code}`, 'fail');
      console.log(JSON.stringify(res.data, null, 2));
      process.exit(2);
    }
  } catch (err) {
    if (err.response) {
      log(`Error status: ${err.response.status}`, 'fail');
      try { console.error('Body:', JSON.stringify(err.response.data, null, 2)); } catch {}
      if (String(err.response.data?.error || '').includes('untrusted') || String(err.response.data?.message || '').includes('untrusted')) {
        log('Trust violation: ensure the public key is present and active in the Runner allowlist and Runner has reloaded.', 'info');
      } else if (String(err.response.data?.error || '').toLowerCase().includes('signature')) {
        log('Signature error: verify key formats (PKCS#8 DER private key, raw 32-byte public key base64).', 'info');
      }
    } else {
      log(`Request error: ${err.message}`, 'fail');
    }
    process.exit(1);
  }
}

main().catch((e) => {
  log(`Fatal error: ${e.message}`, 'fail');
  process.exit(1);
});
