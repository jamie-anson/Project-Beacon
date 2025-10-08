#!/usr/bin/env node

/**
 * Pre-deployment validation script
 * Runs all tests and checks before deployment to prevent broken releases
 */

const { execSync } = require('child_process');
const https = require('https');
const http = require('http');
const fs = require('fs');
const path = require('path');

console.log('🚀 Starting pre-deployment validation...\n');

let exitCode = 0;
const results = [];

function runCommand(command, description) {
  console.log(`📋 ${description}...`);
  try {
    const output = execSync(command, { 
      stdio: 'pipe', 
      encoding: 'utf8',
      cwd: process.cwd()
    });
    console.log(`✅ ${description} - PASSED\n`);
    results.push({ test: description, status: 'PASSED', output: output.trim() });
    return true;
  } catch (error) {
    console.error(`❌ ${description} - FAILED`);
    console.error(`Error: ${error.message}`);
    if (error.stdout) console.error(`Stdout: ${error.stdout}`);
    if (error.stderr) console.error(`Stderr: ${error.stderr}`);
    console.log('');
    results.push({ 
      test: description, 
      status: 'FAILED', 
      error: error.message,
      stdout: error.stdout,
      stderr: error.stderr
    });
    exitCode = 1;
    return false;
  }
}

function checkFileExists(filePath, description) {
  console.log(`📋 ${description}...`);
  if (fs.existsSync(filePath)) {
    console.log(`✅ ${description} - PASSED\n`);
    results.push({ test: description, status: 'PASSED' });
    return true;
  } else {
    console.error(`❌ ${description} - FAILED`);
    console.error(`File not found: ${filePath}\n`);
    results.push({ test: description, status: 'FAILED', error: `File not found: ${filePath}` });
    exitCode = 1;
    return false;
  }
}

function checkRailwayService(url, serviceName, expectedEndpoints = []) {
  return new Promise((resolve) => {
    console.log(`📋 Checking Railway service: ${serviceName}...`);

    // Check health endpoint first
    const healthUrl = new URL(url);
    const req = (healthUrl.protocol === 'https:' ? https : http).get({
      hostname: healthUrl.hostname,
      path: healthUrl.pathname + '/health',
      timeout: 10000,
      headers: { 'User-Agent': 'Project-Beacon-PreDeploy-Check' }
    }, (res) => {
      let data = '';
      res.on('data', chunk => data += chunk);
      res.on('end', () => {
        if (res.statusCode === 200) {
          try {
            const healthData = JSON.parse(data);
            console.log(`✅ ${serviceName} health check - PASSED (${healthData.status || 'ok'})`);
            results.push({ test: `${serviceName} health check`, status: 'PASSED' });

            // Check specific endpoints if service is healthy
            if (expectedEndpoints.length > 0) {
              checkServiceEndpoints(url, serviceName, expectedEndpoints)
                .then(endpointResults => {
                  const failedEndpoints = endpointResults.filter(r => !r.success);
                  if (failedEndpoints.length > 0) {
                    console.log(`⚠️  ${serviceName} endpoints check - WARNING (${failedEndpoints.length} failed)`);
                    failedEndpoints.forEach(ep => {
                      results.push({
                        test: `${serviceName} ${ep.endpoint} endpoint`,
                        status: 'WARNING',
                        error: ep.error
                      });
                    });
                  } else {
                    console.log(`✅ ${serviceName} endpoints check - PASSED`);
                  }
                  resolve(true);
                });
            } else {
              resolve(true);
            }
          } catch (e) {
            console.log(`✅ ${serviceName} health check - PASSED (not JSON but 200 OK)`);
            results.push({ test: `${serviceName} health check`, status: 'PASSED' });
            resolve(true);
          }
        } else {
          console.error(`❌ ${serviceName} health check - FAILED (${res.statusCode})`);
          console.error(`Response: ${data.substring(0, 200)}...`);
          results.push({
            test: `${serviceName} health check`,
            status: 'FAILED',
            error: `HTTP ${res.statusCode}: ${data.substring(0, 100)}`
          });
          resolve(false);
        }
      });
    });

    req.on('error', (err) => {
      console.error(`❌ ${serviceName} health check - FAILED (connection error)`);
      console.error(`Error: ${err.message}`);
      results.push({
        test: `${serviceName} health check`,
        status: 'FAILED',
        error: err.message
      });
      resolve(false);
    });

    req.on('timeout', () => {
      console.error(`❌ ${serviceName} health check - FAILED (timeout)`);
      req.destroy();
      results.push({
        test: `${serviceName} health check`,
        status: 'FAILED',
        error: 'Connection timeout (>10s)'
      });
      resolve(false);
    });

    req.setTimeout(10000);
  });
}

function checkServiceEndpoints(baseUrl, serviceName, endpoints) {
  const promises = endpoints.map(endpoint => {
    return new Promise((resolve) => {
      const url = new URL(baseUrl);
      const req = (url.protocol === 'https:' ? https : http).get({
        hostname: url.hostname,
        path: url.pathname + endpoint,
        timeout: 5000,
        headers: { 'User-Agent': 'Project-Beacon-PreDeploy-Check' }
      }, (res) => {
        let data = '';
        res.on('data', chunk => data += chunk);
        res.on('end', () => {
          if (res.statusCode === 200) {
            resolve({ endpoint, success: true });
          } else {
            resolve({
              endpoint,
              success: false,
              error: `HTTP ${res.statusCode}: ${data.substring(0, 50)}`
            });
          }
        });
      });

      req.on('error', (err) => {
        resolve({ endpoint, success: false, error: err.message });
      });

      req.on('timeout', () => {
        req.destroy();
        resolve({ endpoint, success: false, error: 'timeout' });
      });

      req.setTimeout(5000);
    });
  });

  return Promise.all(promises);
}

// 1. Check critical files exist
checkFileExists('portal/src/lib/api.js', 'Portal API client exists');
checkFileExists('portal/package.json', 'Portal package.json exists');
checkFileExists('scripts/test-build-output.js', 'Build validation script exists');

// 2. Install portal dependencies
runCommand('cd portal && npm ci', 'Install portal dependencies');

// 3. Run portal unit tests
runCommand('cd portal && npm test -- --watchAll=false', 'Portal unit tests');

// 4. Build portal
runCommand('cd portal && npm run build', 'Build portal');

// 4.5. Copy portal build to dist/portal for Playwright
runCommand('mkdir -p dist/portal && cp -R portal/dist/* dist/portal/', 'Copy portal build to dist/portal');

// 5. Run build output validation
runCommand('node scripts/test-build-output.js', 'Build output validation');

// 6. Run API payload tests
runCommand('node scripts/test-job-payload.js', 'API payload validation');

// 7. Run browser-based CORS tests (skip wallet tests for now)
runCommand('npx playwright test --project=chromium tests/e2e/cors-integration.test.js', 'Browser CORS integration tests');

// 8. Check environment configuration
console.log('📋 Environment configuration check...');
const portalEnvExample = path.join('portal', '.env.example');
const portalEnv = path.join('portal', '.env');
if (fs.existsSync(portalEnvExample)) {
  console.log('✅ Portal .env.example found');
  results.push({ test: 'Portal .env.example exists', status: 'PASSED' });
} else {
  console.log('⚠️  Portal .env.example not found (optional)');
  results.push({ test: 'Portal .env.example exists', status: 'WARNING' });
}

// 10. Check Railway services are actually running
console.log('📋 Railway services health check...');
const services = [
  { url: 'https://project-beacon-production.up.railway.app', name: 'Hybrid Router', endpoints: ['/health', '/providers'] },
  { url: 'https://backend-diffs-production.up.railway.app', name: 'Backend Diffs', endpoints: ['/health'] },
  { url: 'https://beacon-runner-production.fly.dev', name: 'Runner API', endpoints: ['/health'] }
];

Promise.all(services.map(s => checkRailwayService(s.url, s.name, s.endpoints)))
  .then(serviceResults => {
    const failedServices = serviceResults.filter(r => !r);
    if (failedServices.length > 0) {
      console.error('❌ Railway services check - FAILED');
      console.error('Some Railway services are not responding. This will cause 404 errors in production.');
      results.push({
        test: 'Railway services health check',
        status: 'FAILED',
        error: `${failedServices.length} services failed health checks`
      });
      exitCode = 1;
    } else {
      console.log('✅ Railway services check - PASSED (all services responding)');
      results.push({ test: 'Railway services health check', status: 'PASSED' });
    }
  })
  .catch(err => {
    console.error('❌ Railway services check - FAILED (error during checks)');
    console.error(`Error: ${err.message}`);
    results.push({
      test: 'Railway services health check',
      status: 'FAILED',
      error: err.message
    });
    exitCode = 1;
  });

// Summary
console.log('\n' + '='.repeat(60));
console.log('📊 PRE-DEPLOYMENT VALIDATION SUMMARY');
console.log('='.repeat(60));

const passed = results.filter(r => r.status === 'PASSED').length;
const failed = results.filter(r => r.status === 'FAILED').length;
const warnings = results.filter(r => r.status === 'WARNING').length;

console.log(`✅ Passed: ${passed}`);
console.log(`❌ Failed: ${failed}`);
console.log(`⚠️  Warnings: ${warnings}`);
console.log(`📋 Total: ${results.length}`);

if (failed > 0) {
  console.log('\n❌ VALIDATION FAILED - Deployment blocked');
  console.log('Fix the following issues before deploying:');
  results.filter(r => r.status === 'FAILED').forEach(r => {
    console.log(`  • ${r.test}`);
    if (r.error) console.log(`    Error: ${r.error}`);
  });
} else {
  console.log('\n✅ VALIDATION PASSED - Ready for deployment');
  console.log('All critical tests passed. Portal is ready to deploy.');

  if (warnings > 0) {
    console.log('\n⚠️  Warnings detected:');
    results.filter(r => r.status === 'WARNING').forEach(r => {
      console.log(`  • ${r.test}`);
      if (r.error) console.log(`    Warning: ${r.error}`);
    });
    console.log('Warnings are non-blocking but should be addressed.');
  }
}

console.log('\n' + '='.repeat(60));

process.exit(exitCode);
