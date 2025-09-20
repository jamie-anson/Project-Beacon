#!/usr/bin/env node
const { spawn } = require('child_process');
const http = require('http');
const path = require('path');

const PORT = 8787;
const BASE_URL = `http://localhost:${PORT}`;

// Routes to test - focus on what we can reliably test locally
const testRoutes = [
  { path: '/', description: 'root page' },
  { path: '/docs/', description: 'docs SPA root' },
  { path: '/docs/intro', description: 'docs deep link' },
  { path: '/portal/', description: 'portal SPA root' },
  { path: '/portal/index.html', description: 'portal index.html direct' },
  { path: '/demo-results/', description: 'demo results' },
];

async function testRoute(path, description, expectSpaFallback = false) {
  return new Promise((resolve) => {
    const req = http.get(`${BASE_URL}${path}`, (res) => {
      let body = '';
      res.on('data', chunk => body += chunk);
      res.on('end', () => {
        const success = res.statusCode === 200 || res.statusCode === 301 || res.statusCode === 302;
        console.log(`${success ? '✅' : '❌'} ${path} (${description}) - ${res.statusCode}`);
        resolve(success);
      });
    });
    
    req.on('error', (err) => {
      console.log(`❌ ${path} (${description}) - ERROR: ${err.message}`);
      resolve(false);
    });
    
    req.setTimeout(5000, () => {
      console.log(`❌ ${path} (${description}) - TIMEOUT`);
      req.destroy();
      resolve(false);
    });
  });
}

async function waitForServer(maxAttempts = 10) {
  for (let i = 0; i < maxAttempts; i++) {
    try {
      await new Promise((resolve, reject) => {
        const req = http.get(`${BASE_URL}/`, (res) => resolve());
        req.on('error', reject);
        req.setTimeout(1000, () => {
          req.destroy();
          reject(new Error('timeout'));
        });
      });
      return true;
    } catch (e) {
      await new Promise(resolve => setTimeout(resolve, 500));
    }
  }
  return false;
}

async function main() {
  const distPath = path.join(__dirname, '..', 'dist');
  
  console.log(`[test-routes] Starting local server on port ${PORT}...`);
  console.log(`[test-routes] Serving from: ${distPath}`);
  
  // Start npx serve
  const server = spawn('npx', ['serve', distPath, '-l', PORT.toString()], {
    stdio: ['ignore', 'pipe', 'pipe']
  });
  
  let serverOutput = '';
  server.stdout.on('data', (data) => {
    serverOutput += data.toString();
  });
  
  server.stderr.on('data', (data) => {
    serverOutput += data.toString();
  });
  
  // Wait for server to be ready
  console.log('[test-routes] Waiting for server to start...');
  const serverReady = await waitForServer();
  
  if (!serverReady) {
    console.error('[test-routes] Server failed to start within timeout');
    console.error('Server output:', serverOutput);
    server.kill();
    process.exit(1);
  }
  
  console.log('[test-routes] Server ready, testing routes...\n');
  
  // Test all routes
  const results = [];
  for (const route of testRoutes) {
    const success = await testRoute(route.path, route.description);
    results.push(success);
  }
  
  // Clean up
  server.kill();
  
  // Summary
  const passed = results.filter(Boolean).length;
  const total = results.length;
  
  console.log(`\n[test-routes] Results: ${passed}/${total} routes passed`);
  
  if (passed === total) {
    console.log('[test-routes] ✅ All route tests passed!');
    process.exit(0);
  } else {
    console.log('[test-routes] ❌ Some route tests failed');
    process.exit(1);
  }
}

// Handle cleanup on exit
process.on('SIGINT', () => {
  console.log('\n[test-routes] Interrupted, cleaning up...');
  process.exit(1);
});

main().catch((err) => {
  console.error('[test-routes] Error:', err);
  process.exit(1);
});
