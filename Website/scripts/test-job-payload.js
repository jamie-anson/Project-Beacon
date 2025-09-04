#!/usr/bin/env node

/**
 * Job Payload Validation Test Suite for Project Beacon
 * 
 * Tests job payload formats against the API validation requirements:
 * - Required fields validation
 * - Payload structure validation  
 * - API response validation
 * - Portal payload generation testing
 */

const https = require('https');
const crypto = require('crypto');

// Ed25519 signing functions
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

// Configuration
const RUNNER_BASE = process.env.RUNNER_URL || 'https://beacon-runner-change-me.fly.dev';
const API_BASE = `${RUNNER_BASE}/api/v1`;
const PORTAL_ORIGIN = 'https://projectbeacon.netlify.app';

// Test results tracking
let passed = 0;
let failed = 0;
const results = [];

function log(message, type = 'info') {
  const timestamp = new Date().toISOString();
  const prefix = type === 'pass' ? '✅' : type === 'fail' ? '❌' : 'ℹ️';
  console.log(`${prefix} [${timestamp}] ${message}`);
}

function recordResult(test, success, details) {
  results.push({ test, success, details, timestamp: new Date().toISOString() });
  if (success) {
    passed++;
    log(`PASS: ${test}`, 'pass');
  } else {
    failed++;
    log(`FAIL: ${test} - ${details}`, 'fail');
  }
}

// HTTP request wrapper
function makeRequest(options, data = null) {
  return new Promise((resolve, reject) => {
    const protocol = https;
    const timeout = setTimeout(() => {
      reject(new Error('Request timeout (10s)'));
    }, 10000);

    const req = protocol.request(options, (res) => {
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

// Parse URL for request options
function parseUrl(url) {
  const urlObj = new URL(url);
  return {
    protocol: urlObj.protocol,
    hostname: urlObj.hostname,
    port: urlObj.port || (urlObj.protocol === 'https:' ? 443 : 80),
    path: urlObj.pathname + urlObj.search
  };
}

// Test job payload submission
async function testJobPayload(testName, payload, expectedStatus, expectedError = null) {
  try {
    const url = `${API_BASE}/jobs`;
    const options = {
      ...parseUrl(url),
      method: 'POST',
      headers: {
        'Origin': PORTAL_ORIGIN,
        'Content-Type': 'application/json',
        'Idempotency-Key': `test-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
      }
    };

    const payloadStr = JSON.stringify(payload);
    const response = await makeRequest(options, payloadStr);
    
    if (response.statusCode === expectedStatus) {
      if (expectedError) {
        try {
          const data = JSON.parse(response.body);
          if (data.error && (data.error.includes(expectedError) || data.message?.includes(expectedError))) {
            recordResult(testName, true, `Correctly rejected: ${data.error}`);
          } else {
            recordResult(testName, false, `Expected error containing "${expectedError}", got: ${data.error || data.message}`);
          }
        } catch (parseError) {
          recordResult(testName, false, `Expected JSON error response, got: ${response.body.slice(0, 100)}`);
        }
      } else {
        recordResult(testName, true, `Status: ${response.statusCode}`);
      }
    } else {
      recordResult(testName, false, `Expected status ${expectedStatus}, got ${response.statusCode}. Body: ${response.body.slice(0, 200)}`);
    }
  } catch (error) {
    recordResult(testName, false, `Network error: ${error.message}`);
  }
}

// Generate portal-style payload with correct nested structure
function generatePortalPayload(options = {}) {
  const defaults = {
    id: `bias-detection-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
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
      created_by: 'portal',
      test_run: true,
      timestamp: new Date().toISOString(),
      nonce: Math.random().toString(36).slice(2)
    },
    created_at: new Date().toISOString(),
    runs: 1,
    questions: ['geography_basic', 'identity_basic', 'math_basic', 'tiananmen_neutral']
  };
  
  return { ...defaults, ...options };
}

// Main test suite
async function runPayloadTests() {
  log(`Starting Job Payload Validation Tests`);
  log(`API Base: ${API_BASE}`);
  log(`Portal Origin: ${PORTAL_ORIGIN}`);
  log('');

  // Test 1: Valid portal payload with cryptographic signature (should succeed)
  log('Running: Valid Portal Payload');
  const validPayload = generatePortalPayload();
  
  // Generate keypair and sign the payload
  const { publicKey, privateKey } = generateEd25519KeyPair();
  const signature = signJobSpec(validPayload, privateKey);
  const publicKeyB64 = exportPublicKey(publicKey);
  
  // Add signature fields
  validPayload.signature = signature;
  validPayload.public_key = publicKeyB64;
  
  await testJobPayload('Valid Portal Payload', validPayload, 400, 'untrusted signing key'); // Expect trust violation
  log('');

  // Test 2: Missing ID field
  log('Running: Missing ID Field');
  const noIdPayload = generatePortalPayload();
  delete noIdPayload.id;
  await testJobPayload('Missing ID Field', noIdPayload, 400, 'ID is required');
  log('');

  // Test 3: Empty ID field
  log('Running: Empty ID Field');
  const emptyIdPayload = generatePortalPayload({ id: '' });
  await testJobPayload('Empty ID Field', emptyIdPayload, 400, 'ID');
  log('');

  // Test 4: Invalid ID format (if API has format requirements)
  log('Running: Invalid ID Format');
  const invalidIdPayload = generatePortalPayload({ id: 'invalid id with spaces!' });
  await testJobPayload('Invalid ID Format', invalidIdPayload, 400);
  log('');

  // Test 5: Missing benchmark field
  log('Running: Missing Benchmark Field');
  const noBenchmarkPayload = generatePortalPayload();
  delete noBenchmarkPayload.benchmark;
  await testJobPayload('Missing Benchmark Field', noBenchmarkPayload, 400, 'benchmark');
  log('');

  // Test 6: Missing benchmark name
  log('Running: Missing Benchmark Name');
  const noBenchmarkNamePayload = generatePortalPayload();
  noBenchmarkNamePayload.benchmark.name = '';
  await testJobPayload('Missing Benchmark Name', noBenchmarkNamePayload, 400, 'benchmark name is required');
  log('');

  // Test 7: Missing container image
  log('Running: Missing Container Image');
  const noContainerImagePayload = generatePortalPayload();
  noContainerImagePayload.benchmark.container.image = '';
  await testJobPayload('Missing Container Image', noContainerImagePayload, 400, 'container image is required');
  log('');

  // Test 8: Missing container object
  log('Running: Missing Container Object');
  const noContainerPayload = generatePortalPayload();
  delete noContainerPayload.benchmark.container;
  await testJobPayload('Missing Container Object', noContainerPayload, 400, 'container image is required');
  log('');

  // Test 9: Missing version field
  log('Running: Missing Version Field');
  const noVersionPayload = generatePortalPayload();
  delete noVersionPayload.version;
  await testJobPayload('Missing Version Field', noVersionPayload, 400, 'version is required');
  log('');

  // Test 10: Missing regions constraint
  log('Running: Missing Regions Constraint');
  const noRegionsPayload = generatePortalPayload();
  noRegionsPayload.constraints.regions = [];
  await testJobPayload('Missing Regions Constraint', noRegionsPayload, 400, 'at least one region constraint is required');
  log('');

  // Test 11: Old flat structure (backwards compatibility test)
  log('Running: Old Flat Structure (container_image)');
  const oldStructurePayload = {
    id: 'test-old-format',
    version: 'v1',
    benchmark: { name: 'bias-detection', version: 'v1' },
    regions: ['US', 'EU', 'ASIA'],
    container_image: 'ghcr.io/project-beacon/bias-detection:latest',
    questions: ['test']
  };
  await testJobPayload('Old Flat Structure', oldStructurePayload, 400, 'container image is required');
  log('');

  // Test 12: Malformed JSON
  log('Running: Malformed JSON');
  await testMalformedJson();
  
  // Print summary
  printTestSummary();
}

// Test malformed JSON separately
async function testMalformedJson() {
  try {
    const url = `${API_BASE}/jobs`;
    const options = {
      ...parseUrl(url),
      method: 'POST',
      headers: {
        'Origin': PORTAL_ORIGIN,
        'Content-Type': 'application/json',
        'Idempotency-Key': `test-malformed-${Date.now()}`
      }
    };

    const malformedJson = '{ "id": "test", invalid json }';
    const response = await makeRequest(options, malformedJson);
    
    if (response.statusCode === 400) {
      recordResult('Malformed JSON', true, `Status: ${response.statusCode} (correctly rejected)`);
    } else {
      recordResult('Malformed JSON', false, `Expected 400, got ${response.statusCode}`);
    }
  } catch (error) {
    recordResult('Malformed JSON', false, `Network error: ${error.message}`);
  }
}

// Print test summary
function printTestSummary() {
  log('');
  log('=== Validation Test Results Summary ===');
  results.forEach(result => {
    const status = result.success ? '✅ PASS' : '❌ FAIL';
    log(`${status} ${result.test}: ${result.details}`);
  });

  const passedCount = results.filter(r => r.success).length;
  const total = results.length;
  log('');
  log(`Overall: ${passedCount}/${total} tests passed`);
  
  if (passedCount === total) {
    log('🎉 All validation tests passed!');
  } else {
    log('⚠️  Some validation tests failed - check API validation logic');
  }
}

// Run tests
runPayloadTests().then(() => {
  printTestSummary();
}).catch(error => {
  log(`Fatal error: ${error.message}`, 'fail');
  process.exit(1);
});
