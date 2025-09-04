#!/usr/bin/env node

/**
 * Pre-deployment validation script
 * Runs all tests and checks before deployment to prevent broken releases
 */

const { execSync } = require('child_process');
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

// 9. Validate built assets
console.log('📋 Built assets validation...');
const distDir = path.join('portal', 'dist');
if (fs.existsSync(distDir)) {
  const assets = fs.readdirSync(path.join(distDir, 'assets')).filter(f => f.endsWith('.js'));
  if (assets.length > 0) {
    console.log(`✅ Found ${assets.length} JavaScript assets`);
    results.push({ test: 'Built JavaScript assets exist', status: 'PASSED' });
  } else {
    console.error('❌ No JavaScript assets found in build');
    results.push({ test: 'Built JavaScript assets exist', status: 'FAILED' });
    exitCode = 1;
  }
} else {
  console.error('❌ Portal dist directory not found');
  results.push({ test: 'Portal dist directory exists', status: 'FAILED' });
  exitCode = 1;
}

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
}

console.log('\n' + '='.repeat(60));

process.exit(exitCode);
