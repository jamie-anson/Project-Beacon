#!/usr/bin/env node

/**
 * Debug Portal API Configuration
 * 
 * This script simulates the portal's API base resolution logic
 * and tests actual requests to help diagnose why the portal
 * might still be experiencing network errors.
 */

const https = require('https');

// Simulate portal's API base resolution
const VITE_API_BASE = process.env.VITE_API_BASE;
const API_BASE_V1 = VITE_API_BASE || 'https://beacon-runner-production.fly.dev/api/v1';

console.log('🔍 Portal API Configuration Debug');
console.log('================================');
console.log(`VITE_API_BASE env var: ${VITE_API_BASE || 'NOT SET'}`);
console.log(`Resolved API_BASE_V1: ${API_BASE_V1}`);
console.log('');

// Test the exact same request the portal would make
function makePortalRequest(endpoint) {
  return new Promise((resolve, reject) => {
    const url = `${API_BASE_V1}${endpoint}`;
    const urlObj = new URL(url);
    
    console.log(`📡 Testing: ${url}`);
    
    const options = {
      hostname: urlObj.hostname,
      port: urlObj.port || 443,
      path: urlObj.pathname + urlObj.search,
      method: 'GET',
      headers: {
        'Origin': 'https://projectbeacon.netlify.app',
        'Content-Type': 'application/json',
        'User-Agent': 'Portal-Debug/1.0'
      }
    };

    const req = https.request(options, (res) => {
      let body = '';
      res.on('data', chunk => body += chunk);
      res.on('end', () => {
        resolve({
          statusCode: res.statusCode,
          headers: res.headers,
          body: body,
          url: url
        });
      });
    });

    req.on('error', (err) => {
      reject({ error: err.message, url: url });
    });

    req.setTimeout(10000, () => {
      req.destroy();
      reject({ error: 'Timeout (10s)', url: url });
    });

    req.end();
  });
}

async function debugPortalAPI() {
  const endpoints = ['/questions', '/executions', '/diffs'];
  
  for (const endpoint of endpoints) {
    try {
      const result = await makePortalRequest(endpoint);
      
      if (result.statusCode === 200) {
        console.log(`✅ ${endpoint}: Status ${result.statusCode}`);
        try {
          const data = JSON.parse(result.body);
          console.log(`   Data: ${Array.isArray(data) ? `${data.length} items` : 'Object'}`);
        } catch (e) {
          console.log(`   Body: ${result.body.slice(0, 100)}...`);
        }
      } else {
        console.log(`❌ ${endpoint}: Status ${result.statusCode}`);
        console.log(`   Body: ${result.body.slice(0, 200)}`);
      }
      
      // Check CORS headers
      const corsOrigin = result.headers['access-control-allow-origin'];
      if (corsOrigin) {
        console.log(`   CORS Origin: ${corsOrigin}`);
      } else {
        console.log(`   ⚠️  No CORS Origin header`);
      }
      
    } catch (error) {
      console.log(`💥 ${endpoint}: ${error.error}`);
      if (error.error.includes('ENOTFOUND') || error.error.includes('ECONNREFUSED')) {
        console.log(`   🔍 DNS/Connection issue with ${error.url}`);
      }
    }
    console.log('');
  }

  // Test job submission (like portal would do)
  console.log('📡 Testing job submission POST...');
  try {
    const url = `${API_BASE_V1}/jobs`;
    const urlObj = new URL(url);
    
    const postData = JSON.stringify({
      invalid: 'payload',
      test: true
    });
    
    const options = {
      hostname: urlObj.hostname,
      port: urlObj.port || 443,
      path: urlObj.pathname,
      method: 'POST',
      headers: {
        'Origin': 'https://projectbeacon.netlify.app',
        'Content-Type': 'application/json',
        'Idempotency-Key': `debug-${Date.now()}`,
        'Content-Length': Buffer.byteLength(postData)
      }
    };

    const req = https.request(options, (res) => {
      let body = '';
      res.on('data', chunk => body += chunk);
      res.on('end', () => {
        if (res.statusCode === 400) {
          console.log(`✅ POST /jobs: Status ${res.statusCode} (expected validation error)`);
        } else if (res.statusCode === 0) {
          console.log(`❌ POST /jobs: Status 0 (network error - CORS issue)`);
        } else {
          console.log(`❓ POST /jobs: Status ${res.statusCode}`);
        }
        console.log(`   Body: ${body.slice(0, 200)}`);
      });
    });

    req.on('error', (err) => {
      console.log(`💥 POST /jobs: ${err.message}`);
    });

    req.write(postData);
    req.end();
    
  } catch (error) {
    console.log(`💥 POST /jobs: ${error.message}`);
  }
}

debugPortalAPI().catch(console.error);
