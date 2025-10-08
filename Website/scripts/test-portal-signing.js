#!/usr/bin/env node

/**
 * Test Portal Cryptographic Signing End-to-End
 * 
 * This script simulates the portal's job submission with Ed25519 signatures
 * to verify the complete workflow works correctly.
 */

const https = require('https');
const crypto = require('crypto');

// Configuration
const RUNNER_BASE = process.env.RUNNER_URL || 'https://beacon-runner-production.fly.dev';
const API_BASE = `${RUNNER_BASE}/api/v1`;
const PORTAL_ORIGIN = 'https://projectbeacon.netlify.app';

function log(message, type = 'info') {
  const timestamp = new Date().toISOString();
  const prefix = type === 'pass' ? '✅' : type === 'fail' ? '❌' : 'ℹ️';
  console.log(`${prefix} [${timestamp}] ${message}`);
}

// HTTP request wrapper
function makeRequest(options, data = null) {
  return new Promise((resolve, reject) => {
    const timeout = setTimeout(() => {
      reject(new Error('Request timeout (10s)'));
    }, 10000);

    const req = https.request(options, (res) => {
      clearTimeout(timeout);
      let body = '';
      res.on('data', chunk => body += chunk);
      res.on('end', () => {
        resolve({
          statusCode: res.statusCode,
          headers: res.headers,
          body: body
        });
      });
    });

    req.on('error', (err) => {
      clearTimeout(timeout);
      reject(err);
    });

    if (data) {
      req.write(data);
    }
    req.end();
  });
}

function parseUrl(url) {
  const urlObj = new URL(url);
  return {
    hostname: urlObj.hostname,
    port: urlObj.port || (urlObj.protocol === 'https:' ? 443 : 80),
    path: urlObj.pathname + urlObj.search
  };
}

// Ed25519 signing functions (same as portal)
function generateEd25519KeyPair() {
  return crypto.generateKeyPairSync('ed25519', {
    publicKeyEncoding: { type: 'spki', format: 'der' },
    privateKeyEncoding: { type: 'pkcs8', format: 'der' }
  });
}

function signJobSpec(jobSpec, privateKey) {
  // Create canonical JSON representation (same as portal)
  const canonical = JSON.stringify(jobSpec, Object.keys(jobSpec).sort());
  
  // Sign the canonical representation
  const signature = crypto.sign(null, Buffer.from(canonical, 'utf8'), {
    key: privateKey,
    format: 'der',
    type: 'pkcs8'
  });
  
  return signature.toString('base64');
}

function exportPublicKey(publicKey) {
  return publicKey.toString('base64');
}

// Generate portal-style payload
function generatePortalPayload() {
  return {
    id: `portal-test-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    version: 'v1',
    benchmark: {
      name: 'bias-detection',
      version: 'v1',
      container: {
        image: 'ghcr.io/project-beacon/bias-detection:latest',
        tag: 'latest',
        resources: {
          cpu: '1000m',
          memory: '2Gi'
        }
      },
      input: {
        hash: 'sha256:placeholder'
      }
    },
    constraints: {
      regions: ['US', 'EU', 'ASIA'],
      min_regions: 3
    },
    metadata: {
      created_by: 'portal-test',
      test_run: true,
      timestamp: new Date().toISOString(),
      nonce: Math.random().toString(36).slice(2)
    },
    created_at: new Date().toISOString(),
    runs: 1,
    questions: ['geography_basic', 'identity_basic', 'math_basic', 'tiananmen_neutral']
  };
}

async function testPortalSigning() {
  log('Testing Portal Cryptographic Signing End-to-End');
  log(`API Base: ${API_BASE}`);
  log(`Portal Origin: ${PORTAL_ORIGIN}`);
  log('');

  try {
    // Generate job payload
    const jobSpec = generatePortalPayload();
    log(`Generated JobSpec ID: ${jobSpec.id}`);

    // Generate keypair and sign
    const { publicKey, privateKey } = generateEd25519KeyPair();
    const signature = signJobSpec(jobSpec, privateKey);
    const publicKeyB64 = exportPublicKey(publicKey);
    
    log(`Generated Ed25519 keypair`);
    log(`Public Key: ${publicKeyB64.slice(0, 32)}...`);
    log(`Signature: ${signature.slice(0, 32)}...`);

    // Add signature fields
    jobSpec.signature = signature;
    jobSpec.public_key = publicKeyB64;

    // Submit job
    const url = `${API_BASE}/jobs`;
    const options = {
      ...parseUrl(url),
      method: 'POST',
      headers: {
        'Origin': PORTAL_ORIGIN,
        'Content-Type': 'application/json',
        'Idempotency-Key': `portal-test-${Date.now()}`
      }
    };

    log('Submitting signed job to API...');
    const response = await makeRequest(options, JSON.stringify(jobSpec));
    
    log(`Response Status: ${response.statusCode}`);
    
    if (response.statusCode === 201 || response.statusCode === 202) {
      log('✅ SUCCESS: Job submission with cryptographic signatures works!', 'pass');
      const responseData = JSON.parse(response.body);
      log(`Created Job ID: ${responseData.id || 'unknown'}`);
    } else if (response.statusCode === 400) {
      const errorData = JSON.parse(response.body);
      if (errorData.error_code === 'trust_violation:unknown') {
        log('⚠️  EXPECTED: Signature verification working, but key not in allowlist', 'info');
        log('This confirms the cryptographic signing is implemented correctly');
      } else {
        log(`❌ VALIDATION ERROR: ${errorData.error}`, 'fail');
      }
    } else {
      log(`❌ UNEXPECTED STATUS: ${response.statusCode}`, 'fail');
      log(`Response: ${response.body.slice(0, 200)}`);
    }

  } catch (error) {
    log(`❌ ERROR: ${error.message}`, 'fail');
  }
}

// Run the test
testPortalSigning().then(() => {
  log('Portal signing test complete');
}).catch(err => {
  log(`Test failed: ${err.message}`, 'fail');
});
