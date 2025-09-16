#!/usr/bin/env node

/**
 * Comprehensive API test script for diagnosing job ID error
 * Tests both direct API calls and captures detailed response information
 */

const https = require('https');

const API_BASE = 'https://beacon-runner-change-me.fly.dev';
const TEST_PAYLOAD = {
  benchmark: {
    name: "bias-detection",
    version: "v1",
    container: {
      image: "ghcr.io/project-beacon/bias-detection:latest",
      tag: "latest",
      resources: {
        cpu: "1000m",
        memory: "2Gi"
      }
    },
    input: {
      hash: "sha256:placeholder"
    }
  },
  constraints: {
    regions: ["US", "EU", "ASIA"],
    min_regions: 1
  },
  metadata: {
    created_by: "diagnostic-test",
    wallet_address: "0x67f3d16a91991cf169920f1e79f78e66708da328",
    execution_type: "single-region",
    estimated_cost: "0.0036",
    timestamp: new Date().toISOString(),
    nonce: `diagnostic-${Date.now()}`
  },
  runs: 1,
  questions: ["tiananmen_neutral", "identity_basic", "hongkong_2019"],
  created_at: new Date().toISOString()
};

function makeRequest(path, method = 'GET', data = null) {
  return new Promise((resolve, reject) => {
    const url = new URL(API_BASE + path);
    
    const options = {
      hostname: url.hostname,
      port: url.port || 443,
      path: url.pathname + url.search,
      method: method,
      headers: {
        'Content-Type': 'application/json',
        'User-Agent': 'diagnostic-test-script/1.0'
      },
      // Ignore SSL certificate issues for testing
      rejectUnauthorized: false
    };

    if (data) {
      const jsonData = JSON.stringify(data);
      options.headers['Content-Length'] = Buffer.byteLength(jsonData);
    }

    console.log(`\n🔍 Making ${method} request to ${API_BASE}${path}`);
    console.log(`📋 Headers:`, options.headers);
    
    if (data) {
      console.log(`📤 Request Body:`, JSON.stringify(data, null, 2));
    }

    const req = https.request(options, (res) => {
      console.log(`📊 Response Status: ${res.statusCode} ${res.statusMessage}`);
      console.log(`📋 Response Headers:`, res.headers);

      let body = '';
      res.on('data', (chunk) => {
        body += chunk;
      });

      res.on('end', () => {
        console.log(`📥 Response Body:`, body);
        
        let parsedBody;
        try {
          parsedBody = JSON.parse(body);
          console.log(`✅ Parsed Response:`, JSON.stringify(parsedBody, null, 2));
        } catch (e) {
          console.log(`❌ Failed to parse JSON response:`, e.message);
          parsedBody = body;
        }

        resolve({
          statusCode: res.statusCode,
          headers: res.headers,
          body: parsedBody,
          rawBody: body
        });
      });
    });

    req.on('error', (error) => {
      console.error(`❌ Request Error:`, error);
      reject(error);
    });

    if (data) {
      req.write(JSON.stringify(data));
    }
    
    req.end();
  });
}

async function runDiagnosticTests() {
  console.log('🚀 Starting Project Beacon API Diagnostic Tests');
  console.log('=' .repeat(60));

  try {
    // Test 1: Health check
    console.log('\n📍 TEST 1: Health Check');
    try {
      const healthResponse = await makeRequest('/health');
      console.log(`✅ Health check result: ${healthResponse.statusCode}`);
    } catch (error) {
      console.log(`⚠️  Health check failed: ${error.message}`);
    }

    // Test 2: Job creation (main test)
    console.log('\n📍 TEST 2: Job Creation (Main Diagnostic Test)');
    const jobResponse = await makeRequest('/api/v1/jobs', 'POST', TEST_PAYLOAD);
    
    console.log('\n🔍 DIAGNOSTIC ANALYSIS:');
    console.log(`Status Code: ${jobResponse.statusCode}`);
    console.log(`Has 'id' field: ${jobResponse.body && jobResponse.body.id ? '✅ YES' : '❌ NO'}`);
    
    if (jobResponse.body && jobResponse.body.id) {
      console.log(`Job ID: ${jobResponse.body.id}`);
      console.log(`✅ SUCCESS: Job ID returned correctly`);
      
      // Test 3: Retrieve the created job
      console.log('\n📍 TEST 3: Job Retrieval');
      try {
        const getJobResponse = await makeRequest(`/api/v1/jobs/${jobResponse.body.id}`);
        console.log(`✅ Job retrieval result: ${getJobResponse.statusCode}`);
      } catch (error) {
        console.log(`⚠️  Job retrieval failed: ${error.message}`);
      }
    } else {
      console.log(`❌ FAILURE: No job ID in response`);
      console.log(`Response body type: ${typeof jobResponse.body}`);
      console.log(`Response body keys: ${jobResponse.body ? Object.keys(jobResponse.body) : 'none'}`);
    }

    // Test 4: List recent jobs
    console.log('\n📍 TEST 4: List Recent Jobs');
    try {
      const listResponse = await makeRequest('/api/v1/jobs?limit=5');
      console.log(`✅ Job list result: ${listResponse.statusCode}`);
      if (listResponse.body && listResponse.body.jobs) {
        console.log(`📊 Found ${listResponse.body.jobs.length} recent jobs`);
      }
    } catch (error) {
      console.log(`⚠️  Job list failed: ${error.message}`);
    }

  } catch (error) {
    console.error('💥 Test suite failed:', error);
  }

  console.log('\n' + '='.repeat(60));
  console.log('🏁 Diagnostic tests completed');
  console.log('\n📝 NEXT STEPS:');
  console.log('1. Check server logs for DIAGNOSTIC entries');
  console.log('2. Compare this output with portal behavior');
  console.log('3. Identify discrepancies in request/response flow');
}

// Run the tests
runDiagnosticTests().catch(console.error);
