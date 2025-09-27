#!/usr/bin/env node

/**
 * Import Validation Tests
 * Catches missing imports that build tools miss
 */

const fs = require('fs');
const path = require('path');

// Test configuration
const PORTAL_SRC = path.join(__dirname, '../portal/src');
const RESULTS_DIR = path.join(__dirname, '../test-results');
const TIMESTAMP = new Date().toISOString().replace(/[:.]/g, '-');

// Test results tracking
const testResults = {
  timestamp: new Date().toISOString(),
  testSuite: 'Import Validation',
  summary: { totalTests: 0, passed: 0, failed: 0, warnings: 0 },
  tests: []
};

function logTest(testName, status, details) {
  testResults.tests.push({
    name: testName,
    status,
    details,
    timestamp: new Date().toISOString()
  });
  
  testResults.summary.totalTests++;
  testResults.summary[status]++;
  
  console.log(`[${status.toUpperCase()}] ${testName}`);
  if (details) {
    console.log(JSON.stringify(details, null, 2));
  }
}

// Extract JSX component usage from file content
function extractJSXComponents(content) {
  const jsxComponentRegex = /<([A-Z][a-zA-Z0-9]*)/g;
  const components = new Set();
  let match;
  
  while ((match = jsxComponentRegex.exec(content)) !== null) {
    components.add(match[1]);
  }
  
  return Array.from(components);
}

// Extract imports from file content
function extractImports(content) {
  const importRegex = /import\s+(?:(?:\{[^}]*\}|\*\s+as\s+\w+|\w+)(?:\s*,\s*(?:\{[^}]*\}|\*\s+as\s+\w+|\w+))*\s+from\s+['"][^'"]+['"]|import\s+['"][^'"]+['"])/g;
  const defaultImportRegex = /import\s+(\w+)\s+from/g;
  const namedImportRegex = /import\s+\{([^}]+)\}\s+from/g;
  
  const imports = new Set();
  let match;
  
  // Extract default imports
  while ((match = defaultImportRegex.exec(content)) !== null) {
    imports.add(match[1]);
  }
  
  // Extract named imports
  while ((match = namedImportRegex.exec(content)) !== null) {
    const namedImports = match[1].split(',').map(imp => imp.trim().split(' as ')[0]);
    namedImports.forEach(imp => imports.add(imp));
  }
  
  return Array.from(imports);
}

// Test individual file for import/usage consistency
function validateFileImports(filePath) {
  try {
    const content = fs.readFileSync(filePath, 'utf8');
    const jsxComponents = extractJSXComponents(content);
    const imports = extractImports(content);
    
    const missingImports = jsxComponents.filter(component => 
      !imports.includes(component) && 
      component !== 'React' && // React is often globally available
      !['div', 'span', 'p', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'button', 'input', 'form', 'section', 'article', 'nav', 'header', 'footer', 'main', 'aside'].includes(component.toLowerCase()) // HTML elements
    );
    
    const unusedImports = imports.filter(imp => 
      !content.includes(imp) || 
      (jsxComponents.length > 0 && !jsxComponents.includes(imp) && !content.includes(`${imp}(`))
    );
    
    return {
      filePath: path.relative(PORTAL_SRC, filePath),
      jsxComponents,
      imports,
      missingImports,
      unusedImports,
      hasIssues: missingImports.length > 0 || unusedImports.length > 0
    };
    
  } catch (error) {
    return {
      filePath: path.relative(PORTAL_SRC, filePath),
      error: error.message,
      hasIssues: true
    };
  }
}

// Test all React files for import validation
async function testImportValidation() {
  console.log('🧪 Testing Import Validation...');
  
  try {
    const reactFiles = [];
    
    // Find all .jsx and .js files in portal/src
    function findReactFiles(dir) {
      const entries = fs.readdirSync(dir, { withFileTypes: true });
      
      for (const entry of entries) {
        const fullPath = path.join(dir, entry.name);
        
        if (entry.isDirectory()) {
          findReactFiles(fullPath);
        } else if (entry.isFile() && (entry.name.endsWith('.jsx') || entry.name.endsWith('.js'))) {
          reactFiles.push(fullPath);
        }
      }
    }
    
    findReactFiles(PORTAL_SRC);
    
    const validationResults = [];
    let totalIssues = 0;
    
    for (const filePath of reactFiles) {
      const result = validateFileImports(filePath);
      validationResults.push(result);
      
      if (result.hasIssues) {
        totalIssues++;
      }
    }
    
    // Analyze results
    const filesWithMissingImports = validationResults.filter(r => r.missingImports && r.missingImports.length > 0);
    const filesWithUnusedImports = validationResults.filter(r => r.unusedImports && r.unusedImports.length > 0);
    
    const testResult = {
      totalFiles: reactFiles.length,
      filesWithIssues: totalIssues,
      filesWithMissingImports: filesWithMissingImports.length,
      filesWithUnusedImports: filesWithUnusedImports.length,
      missingImportDetails: filesWithMissingImports.map(f => ({
        file: f.filePath,
        missing: f.missingImports
      })),
      unusedImportDetails: filesWithUnusedImports.map(f => ({
        file: f.filePath,
        unused: f.unusedImports
      })),
      validationPassed: filesWithMissingImports.length === 0
    };
    
    logTest('Import Validation', 
      testResult.validationPassed ? 'passed' : 'failed', 
      testResult
    );
    
  } catch (error) {
    logTest('Import Validation', 'failed', {
      error: error.message,
      stack: error.stack
    });
  }
}

// Test specific known problematic files
async function testBiasDetectionImports() {
  console.log('🧪 Testing BiasDetection Import Issues...');
  
  try {
    const biasDetectionPath = path.join(PORTAL_SRC, 'pages/BiasDetection.jsx');
    
    if (!fs.existsSync(biasDetectionPath)) {
      logTest('BiasDetection Import Issues', 'failed', {
        error: 'BiasDetection.jsx not found'
      });
      return;
    }
    
    const result = validateFileImports(biasDetectionPath);
    
    // Known components that should be imported
    const expectedComponents = [
      'ModelSelector',
      'RegionSelector', 
      'JobSummaryCards',
      'WalletConnection',
      'ErrorMessage',
      'InfrastructureStatus'
    ];
    
    const actuallyMissing = expectedComponents.filter(comp => 
      result.jsxComponents.includes(comp) && !result.imports.includes(comp)
    );
    
    const testResult = {
      ...result,
      expectedComponents,
      actuallyMissing,
      criticalIssues: actuallyMissing.length,
      testPassed: actuallyMissing.length === 0
    };
    
    logTest('BiasDetection Import Issues', 
      testResult.testPassed ? 'passed' : 'failed', 
      testResult
    );
    
  } catch (error) {
    logTest('BiasDetection Import Issues', 'failed', {
      error: error.message,
      stack: error.stack
    });
  }
}

// Main test runner
async function runImportValidationTests() {
  console.log('🚀 Starting Import Validation Test Suite');
  
  const startTime = Date.now();
  
  try {
    await testImportValidation();
    await testBiasDetectionImports();
    
  } catch (error) {
    console.error('Test suite execution failed:', error);
  }
  
  const endTime = Date.now();
  const totalTime = endTime - startTime;
  
  // Generate final report
  const report = {
    ...testResults,
    executionTime: `${totalTime}ms`,
    summary: {
      ...testResults.summary,
      successRate: `${(testResults.summary.passed / testResults.summary.totalTests * 100).toFixed(1)}%`
    }
  };
  
  // Save detailed results
  if (!fs.existsSync(RESULTS_DIR)) {
    fs.mkdirSync(RESULTS_DIR, { recursive: true });
  }
  
  const reportPath = path.join(RESULTS_DIR, `import-validation-${TIMESTAMP}.json`);
  fs.writeFileSync(reportPath, JSON.stringify(report, null, 2));
  
  // Print summary
  console.log('\n📊 Import Validation Test Suite Complete');
  console.log(`Total Tests: ${report.summary.totalTests}`);
  console.log(`Passed: ${report.summary.passed}`);
  console.log(`Failed: ${report.summary.failed}`);
  console.log(`Warnings: ${report.summary.warnings}`);
  console.log(`Success Rate: ${report.summary.successRate}`);
  console.log(`Execution Time: ${report.executionTime}`);
  console.log(`Report Saved: ${reportPath}`);
  
  // Exit with appropriate code
  process.exit(report.summary.failed > 0 ? 1 : 0);
}

// Run tests if called directly
if (require.main === module) {
  runImportValidationTests().catch(error => {
    console.error('Fatal error:', error);
    process.exit(1);
  });
}

module.exports = {
  runImportValidationTests,
  testImportValidation,
  testBiasDetectionImports,
  validateFileImports
};
