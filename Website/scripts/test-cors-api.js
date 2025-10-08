#!/usr/bin/env node

/**
 * CORS & API Endpoint Test Suite for Project Beacon
 * 
 * Tests both success and error scenarios:
 * - CORS preflight requests from portal origin
 * - API endpoint availability and responses
 * - Error handling for missing/invalid endpoints
 * - Network connectivity validation
 */

const https = require('https');
const http = require('http');

// Configuration
const PORTAL_ORIGIN = 'https://projectbeacon.netlify.app';
const RUNNER_BASE = process.env.RUNNER_URL || 'https://beacon-runner-production.fly.dev';
const API_BASE = `${RUNNER_BASE}/api/v1`;

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

// HTTP request wrapper with timeout
function makeRequest(options, data = null) {
  return new Promise((resolve, reject) => {
    const protocol = options.protocol === 'https:' ? https : http;
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

// Test 1: CORS Preflight for /questions
async function testCorsPreflightQuestions() {
  try {
    const url = `${API_BASE}/questions`;
    const options = {
      ...parseUrl(url),
      method: 'OPTIONS',
      headers: {
        'Origin': PORTAL_ORIGIN,
        'Access-Control-Request-Method': 'GET',
        'Access-Control-Request-Headers': 'content-type,authorization,Idempotency-Key'
      }
    };

    const response = await makeRequest(options);
    
    const allowOrigin = response.headers['access-control-allow-origin'];
    const allowHeaders = response.headers['access-control-allow-headers'];
    
    if (response.statusCode === 204 && 
        allowOrigin === PORTAL_ORIGIN &&
        allowHeaders && allowHeaders.toLowerCase().includes('idempotency-key')) {
      recordResult('CORS Preflight /questions', true, `Status: ${response.statusCode}, Origin: ${allowOrigin}`);
    } else {
      recordResult('CORS Preflight /questions', false, 
        `Status: ${response.statusCode}, Origin: ${allowOrigin}, Headers: ${allowHeaders}`);
    }
  } catch (error) {
    recordResult('CORS Preflight /questions', false, `Network error: ${error.message}`);
  }
}

// Test 2: CORS Preflight for /executions
async function testCorsPreflightExecutions() {
  try {
    const url = `${API_BASE}/executions`;
    const options = {
      ...parseUrl(url),
      method: 'OPTIONS',
      headers: {
        'Origin': PORTAL_ORIGIN,
        'Access-Control-Request-Method': 'GET',
        'Access-Control-Request-Headers': 'content-type,authorization,Idempotency-Key'
      }
    };

    const response = await makeRequest(options);
    
    const allowOrigin = response.headers['access-control-allow-origin'];
    const allowHeaders = response.headers['access-control-allow-headers'];
    
    if (response.statusCode === 204 && 
        allowOrigin === PORTAL_ORIGIN &&
        allowHeaders && allowHeaders.toLowerCase().includes('idempotency-key')) {
      recordResult('CORS Preflight /executions', true, `Status: ${response.statusCode}, Origin: ${allowOrigin}`);
    } else {
      recordResult('CORS Preflight /executions', false, 
        `Status: ${response.statusCode}, Origin: ${allowOrigin}, Headers: ${allowHeaders}`);
    }
  } catch (error) {
    recordResult('CORS Preflight /executions', false, `Network error: ${error.message}`);
  }
}

// Test 3: GET /questions endpoint
async function testQuestionsEndpoint() {
  try {
    const url = `${API_BASE}/questions`;
    const options = {
      ...parseUrl(url),
      method: 'GET',
      headers: {
        'Origin': PORTAL_ORIGIN,
        'Content-Type': 'application/json'
      }
    };

    const response = await makeRequest(options);
    
    if (response.statusCode === 200) {
      try {
        const data = JSON.parse(response.body);
        recordResult('GET /questions', true, `Status: 200, Questions count: ${data.length || 'N/A'}`);
      } catch (parseError) {
        recordResult('GET /questions', false, `Status: 200 but invalid JSON: ${parseError.message}`);
      }
    } else {
      recordResult('GET /questions', false, `Status: ${response.statusCode}, Body: ${response.body.slice(0, 100)}`);
    }
  } catch (error) {
    recordResult('GET /questions', false, `Network error: ${error.message}`);
  }
}

// Test 4: GET /executions endpoint
async function testExecutionsEndpoint() {
  try {
    const url = `${API_BASE}/executions`;
    const options = {
      ...parseUrl(url),
      method: 'GET',
      headers: {
        'Origin': PORTAL_ORIGIN,
        'Content-Type': 'application/json'
      }
    };

    const response = await makeRequest(options);
    
    if (response.statusCode === 200) {
      try {
        const data = JSON.parse(response.body);
        recordResult('GET /executions', true, `Status: 200, Executions count: ${data.length || 'N/A'}`);
      } catch (parseError) {
        recordResult('GET /executions', false, `Status: 200 but invalid JSON: ${parseError.message}`);
      }
    } else if (response.statusCode === 404) {
      recordResult('GET /executions', false, `Route not found (404) - runner missing new endpoints`);
    } else {
      recordResult('GET /executions', false, `Status: ${response.statusCode}, Body: ${response.body.slice(0, 100)}`);
    }
  } catch (error) {
    recordResult('GET /executions', false, `Network error: ${error.message}`);
  }
}

// Test 5: GET /diffs endpoint
async function testDiffsEndpoint() {
  try {
    const url = `${API_BASE}/diffs`;
    const options = {
      ...parseUrl(url),
      method: 'GET',
      headers: {
        'Origin': PORTAL_ORIGIN,
        'Content-Type': 'application/json'
      }
    };

    const response = await makeRequest(options);
    
    if (response.statusCode === 200) {
      try {
        const data = JSON.parse(response.body);
        recordResult('GET /diffs', true, `Status: 200, Diffs count: ${data.length || 'N/A'}`);
      } catch (parseError) {
        recordResult('GET /diffs', false, `Status: 200 but invalid JSON: ${parseError.message}`);
      }
    } else if (response.statusCode === 404) {
      recordResult('GET /diffs', false, `Route not found (404) - runner missing new endpoints`);
    } else {
      recordResult('GET /diffs', false, `Status: ${response.statusCode}, Body: ${response.body.slice(0, 100)}`);
    }
  } catch (error) {
    recordResult('GET /diffs', false, `Network error: ${error.message}`);
  }
}

// Test 6: Health check (root path)
async function testHealthEndpoint() {
  try {
    const url = `${RUNNER_BASE}/health`;
    const options = {
      ...parseUrl(url),
      method: 'GET',
      headers: {
        'Origin': PORTAL_ORIGIN
      }
    };

    const response = await makeRequest(options);
    
    if (response.statusCode === 200) {
      try {
        const data = JSON.parse(response.body);
        const status = data.status || 'unknown';
        recordResult('GET /health', true, `Status: 200, Health: ${status}`);
      } catch (parseError) {
        recordResult('GET /health', true, `Status: 200 (non-JSON response)`);
      }
    } else {
      recordResult('GET /health', false, `Status: ${response.statusCode}`);
    }
  } catch (error) {
    recordResult('GET /health', false, `Network error: ${error.message}`);
  }
}

// Test 7: Invalid endpoint (should 404)
async function testInvalidEndpoint() {
  try {
    const url = `${API_BASE}/nonexistent`;
    const options = {
      ...parseUrl(url),
      method: 'GET',
      headers: {
        'Origin': PORTAL_ORIGIN
      }
    };

    const response = await makeRequest(options);
    
    if (response.statusCode === 404) {
      recordResult('GET /nonexistent (error test)', true, `Correctly returned 404`);
    } else {
      recordResult('GET /nonexistent (error test)', false, `Expected 404, got ${response.statusCode}`);
    }
  } catch (error) {
    recordResult('GET /nonexistent (error test)', false, `Network error: ${error.message}`);
  }
}

// Test 8: Job submission POST (without valid payload - should get validation error)
async function testJobSubmissionError() {
  try {
    const url = `${API_BASE}/jobs`;
    const options = {
      ...parseUrl(url),
      method: 'POST',
      headers: {
        'Origin': PORTAL_ORIGIN,
        'Content-Type': 'application/json',
        'Idempotency-Key': `test-${Date.now()}`
      }
    };

    const invalidPayload = JSON.stringify({ invalid: 'payload' });
    const response = await makeRequest(options, invalidPayload);
    
    if (response.statusCode === 400) {
      recordResult('POST /jobs (error test)', true, `Correctly rejected invalid payload with 400`);
    } else if (response.statusCode === 0) {
      recordResult('POST /jobs (error test)', false, `Network error - status code 0 (CORS/connectivity issue)`);
    } else {
      recordResult('POST /jobs (error test)', false, `Expected 400, got ${response.statusCode}`);
    }
  } catch (error) {
    recordResult('POST /jobs (error test)', false, `Network error: ${error.message}`);
  }
}

// Main test runner
async function runAllTests() {
  log(`Starting CORS & API tests for Project Beacon`);
  log(`Portal Origin: ${PORTAL_ORIGIN}`);
  log(`Runner Base: ${RUNNER_BASE}`);
  log(`API Base: ${API_BASE}`);
  log('');

  const tests = [
    { name: 'CORS Preflight /questions', fn: testCorsPreflightQuestions },
    { name: 'CORS Preflight /executions', fn: testCorsPreflightExecutions },
    { name: 'GET /questions', fn: testQuestionsEndpoint },
    { name: 'GET /executions', fn: testExecutionsEndpoint },
    { name: 'GET /diffs', fn: testDiffsEndpoint },
    { name: 'GET /health', fn: testHealthEndpoint },
    { name: 'GET /nonexistent (error)', fn: testInvalidEndpoint },
    { name: 'POST /jobs (error)', fn: testJobSubmissionError }
  ];

  for (const test of tests) {
    log(`Running: ${test.name}`);
    await test.fn();
    log('');
  }

  // Summary
  log('='.repeat(60));
  log(`Test Results: ${passed} passed, ${failed} failed`);
  log('='.repeat(60));

  if (failed > 0) {
    log('FAILED TESTS:');
    results.filter(r => !r.success).forEach(r => {
      log(`  - ${r.test}: ${r.details}`);
    });
    log('');
  }

  // Specific diagnostics
  const corsFailures = results.filter(r => r.test.includes('CORS') && !r.success);
  const endpointFailures = results.filter(r => r.test.includes('GET /') && !r.success);
  const networkErrors = results.filter(r => r.details.includes('Network error'));

  if (corsFailures.length > 0) {
    log('🔍 CORS Issues Detected:');
    log('  - Check runner CORS middleware allows portal origin');
    log('  - Verify Idempotency-Key in allowed headers');
    log('  - Confirm preflight OPTIONS handler is working');
  }

  if (endpointFailures.length > 0) {
    log('🔍 Missing Endpoints Detected:');
    log('  - Runner may be old build missing /executions, /diffs');
    log('  - Redeploy runner from latest source with all routes');
    log('  - Check cmd/runner/main.go is the entrypoint (not cmd/server)');
  }

  if (networkErrors.length > 0) {
    log('🔍 Network Connectivity Issues:');
    log('  - Runner may not be listening on 0.0.0.0:8090');
    log('  - Fly.io port mapping may be incorrect');
    log('  - Check runner deployment logs for bind address');
  }

  process.exit(failed > 0 ? 1 : 0);
}

// Run tests
runAllTests().catch(error => {
  log(`Fatal error: ${error.message}`, 'fail');
  process.exit(1);
});
