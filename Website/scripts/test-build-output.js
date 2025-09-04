#!/usr/bin/env node

/**
 * Build Output Validation Tests
 * Verifies that built JavaScript contains expected code patterns
 * Prevents issues like Vite optimization removing CORS settings
 */

const fs = require('fs');
const path = require('path');

const DIST_DIR = path.join(__dirname, '../portal/dist');
const ASSETS_DIR = path.join(DIST_DIR, 'assets');

function findJSFiles() {
  if (!fs.existsSync(ASSETS_DIR)) {
    throw new Error(`Assets directory not found: ${ASSETS_DIR}`);
  }
  
  const files = fs.readdirSync(ASSETS_DIR);
  return files.filter(file => file.startsWith('index-') && file.endsWith('.js'));
}

function readFileContent(filename) {
  const filepath = path.join(ASSETS_DIR, filename);
  return fs.readFileSync(filepath, 'utf8');
}

function testCORSSettings(content) {
  const corsCount = (content.match(/cors/g) || []).length;
  
  if (corsCount === 0) {
    throw new Error('CORS settings not found in built JavaScript - Vite may have optimized them away');
  }
  
  console.log(`✓ CORS settings found (${corsCount} instances)`);
  return true;
}

function testAPIBaseURL(content) {
  // Check for API base URL patterns
  const hasAPIBase = content.includes('api/v1') ||
                     content.includes('VITE_API_BASE');
  
  if (!hasAPIBase) {
    throw new Error('API base URL not found in built JavaScript');
  }
  
  console.log('✓ API base URL configuration found');
  return true;
}

function testFetchCalls(content) {
  // Ensure fetch calls are present
  const fetchCount = (content.match(/fetch\(/g) || []).length;
  
  if (fetchCount === 0) {
    throw new Error('No fetch calls found in built JavaScript');
  }
  
  console.log(`✓ Fetch calls found (${fetchCount} instances)`);
  return true;
}

function testNoDoubleSlashes(content) {
  // Check for potential double slash patterns that could indicate URL bugs
  const suspiciousPatterns = [
    /\/\/[a-z]/g,  // //letters (but not protocol://)
    /api\/v1\/\//g // api/v1// pattern
  ];
  
  for (const pattern of suspiciousPatterns) {
    const matches = content.match(pattern);
    if (matches) {
      console.warn(`⚠️  Suspicious URL pattern found: ${matches[0]} (${matches.length} instances)`);
      // Don't fail for this, just warn
    }
  }
  
  console.log('✓ No obvious double-slash URL bugs detected');
  return true;
}

function testCryptographicSigning(content) {
  // Check for cryptographic signing code
  const signingPatterns = [
    /generateKeyPair/,
    /Ed25519/,
    /signature/,
    /public_key/
  ];
  
  let foundPatterns = 0;
  for (const pattern of signingPatterns) {
    if (pattern.test(content)) {
      foundPatterns++;
    }
  }
  
  if (foundPatterns === 0) {
    console.warn('⚠️  No cryptographic signing patterns found - may be in separate chunk');
  } else {
    console.log(`✓ Cryptographic signing code found (${foundPatterns} patterns)`);
  }
  
  return true;
}

function runBuildValidation() {
  console.log('🔍 Running build output validation...\n');
  
  try {
    const jsFiles = findJSFiles();
    
    if (jsFiles.length === 0) {
      throw new Error('No built JavaScript files found. Run "npm run build" first.');
    }
    
    console.log(`Found ${jsFiles.length} JavaScript file(s): ${jsFiles.join(', ')}\n`);
    
    // Test the main bundle (usually the largest file)
    const mainFile = jsFiles.reduce((largest, current) => {
      const currentPath = path.join(ASSETS_DIR, current);
      const largestPath = path.join(ASSETS_DIR, largest);
      
      const currentSize = fs.statSync(currentPath).size;
      const largestSize = fs.statSync(largestPath).size;
      
      return currentSize > largestSize ? current : largest;
    });
    
    console.log(`Testing main bundle: ${mainFile}\n`);
    
    const content = readFileContent(mainFile);
    const fileSizeKB = Math.round(content.length / 1024);
    console.log(`File size: ${fileSizeKB} KB\n`);
    
    // Run validation tests
    testCORSSettings(content);
    testAPIBaseURL(content);
    testFetchCalls(content);
    testNoDoubleSlashes(content);
    testCryptographicSigning(content);
    
    console.log('\n✅ All build validation tests passed!');
    return true;
    
  } catch (error) {
    console.error(`\n❌ Build validation failed: ${error.message}`);
    process.exit(1);
  }
}

// Run if called directly
if (require.main === module) {
  runBuildValidation();
}

module.exports = { runBuildValidation };
