#!/usr/bin/env node
/**
 * Netlify build script with detailed error logging
 * This helps identify exactly where the build is failing
 */

const { execSync } = require('child_process');
const path = require('path');
const fs = require('fs');

function log(message) {
  console.log(`[BUILD] ${new Date().toISOString()} - ${message}`);
}

function runCommand(command, description) {
  log(`Starting: ${description}`);
  log(`Command: ${command}`);
  
  try {
    const output = execSync(command, { 
      cwd: process.cwd(),
      stdio: 'inherit',
      encoding: 'utf8'
    });
    log(`✅ Completed: ${description}`);
    return output;
  } catch (error) {
    log(`❌ Failed: ${description}`);
    log(`Error code: ${error.status}`);
    log(`Error message: ${error.message}`);
    if (error.stdout) log(`Stdout: ${error.stdout}`);
    if (error.stderr) log(`Stderr: ${error.stderr}`);
    throw error;
  }
}

async function main() {
  try {
    log('=== Project Beacon Netlify Build Started ===');
    
    // Check environment
    log(`Node version: ${process.version}`);
    log(`Working directory: ${process.cwd()}`);
    log(`Platform: ${process.platform}`);
    
    // Check if required directories exist
    const requiredDirs = ['docs', 'portal', 'scripts'];
    for (const dir of requiredDirs) {
      if (!fs.existsSync(dir)) {
        throw new Error(`Required directory missing: ${dir}`);
      }
      log(`✅ Directory exists: ${dir}`);
    }
    
    // Check package.json files
    const packageFiles = ['package.json', 'portal/package.json'];
    for (const file of packageFiles) {
      if (!fs.existsSync(file)) {
        throw new Error(`Required package.json missing: ${file}`);
      }
      log(`✅ Package file exists: ${file}`);
    }
    
    // Step 1: Install root dependencies (Netlify already ran npm ci before build)
    const npmFlags = (process.env.NPM_FLAGS || '').trim();
    const rootInstalled = fs.existsSync('node_modules') && fs.existsSync('package-lock.json');
    if (rootInstalled && process.env.NETLIFY) {
      log('Skipping root npm ci (detected NETLIFY and node_modules present)');
    } else {
      runCommand(`npm ci${npmFlags ? ' ' + npmFlags : ''}`.trim(), 'Installing root dependencies');
    }
    
    // Step 2: Install portal dependencies  
    const portalCmd = `npm ci${npmFlags ? ' ' + npmFlags : ''}`.trim();
    runCommand(`bash -lc "cd portal && ${portalCmd}"`, 'Installing portal dependencies');
    
    // Step 3: Build static site
    runCommand('npm run build:static', 'Building static site');
    
    // Step 4: Build docs (first pass)
    runCommand('npm run build:docs', 'Building documentation');
    
    // Step 5: Generate CID
    runCommand('npm run postbuild:cid', 'Generating IPFS CID');
    
    // Step 6: Build docs with CID
    runCommand('npm run build:docs:with-cid', 'Rebuilding docs with CID');
    
    // Step 7: Build portal
    runCommand('npm run build:portal', 'Building portal application');
    
    // Step 8: Finalize build
    runCommand('npm run build:finalize', 'Finalizing build output');
    
    // Verify build output
    const distDir = 'dist';
    if (!fs.existsSync(distDir)) {
      throw new Error('Build output directory (dist) not created');
    }
    
    const requiredOutputs = ['index.html', 'docs', 'portal'];
    for (const output of requiredOutputs) {
      const outputPath = path.join(distDir, output);
      if (!fs.existsSync(outputPath)) {
        log(`⚠️  Warning: Expected output missing: ${output}`);
      } else {
        log(`✅ Output verified: ${output}`);
      }
    }
    
    log('=== Project Beacon Netlify Build Completed Successfully ===');
    
  } catch (error) {
    log('=== Project Beacon Netlify Build Failed ===');
    log(`Final error: ${error.message}`);
    process.exit(1);
  }
}

main();
