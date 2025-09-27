#!/usr/bin/env node

/**
 * Master Test Runner
 * Orchestrates all test suites and provides comprehensive reporting
 */

const fs = require('fs');
const path = require('path');
const { spawn } = require('child_process');

// Test configuration
const RESULTS_DIR = path.join(__dirname, '../test-results');
const TIMESTAMP = new Date().toISOString().replace(/[:.]/g, '-');

// Test suites configuration
const TEST_SUITES = [
  {
    name: 'API Data Validation',
    script: 'test-model-mapping-validation.js',
    priority: 'high',
    description: 'Schema validation, model consistency checks'
  },
  {
    name: 'Portal State Management',
    script: 'test-portal-state-management.js',
    priority: 'high',
    description: 'React state transitions, model selection'
  },
  {
    name: 'End-to-End Workflow',
    script: 'test-end-to-end-workflow.js',
    priority: 'medium',
    description: 'Complete job pipeline validation'
  },
  {
    name: 'React Component Integration',
    script: 'test-react-component-integration.js',
    priority: 'medium',
    description: 'Component behavior, props flow, UI state'
  },
  {
    name: 'Mock Data Scenarios',
    script: 'test-mock-data-scenarios.js',
    priority: 'medium',
    description: 'Edge cases, malformed data, boundary conditions'
  }
];

// Master test results
const masterResults = {
  timestamp: new Date().toISOString(),
  testSuite: 'Master Test Runner',
  suites: [],
  summary: {
    totalSuites: TEST_SUITES.length,
    completedSuites: 0,
    passedSuites: 0,
    failedSuites: 0,
    totalTests: 0,
    totalPassed: 0,
    totalFailed: 0,
    totalWarnings: 0,
    overallSuccessRate: '0%',
    executionTime: '0ms'
  }
};

// Utility functions
function log(level, message, data = null) {
  const timestamp = new Date().toISOString();
  console.log(`[${timestamp}] ${level.toUpperCase()}: ${message}`);
  if (data) {
    console.log(JSON.stringify(data, null, 2));
  }
}

function runTestSuite(suite) {
  return new Promise((resolve, reject) => {
    const scriptPath = path.join(__dirname, suite.script);
    
    if (!fs.existsSync(scriptPath)) {
      resolve({
        name: suite.name,
        status: 'failed',
        error: `Script not found: ${suite.script}`,
        executionTime: 0,
        summary: { totalTests: 0, passed: 0, failed: 1, warnings: 0 }
      });
      return;
    }
    
    log('info', `Starting test suite: ${suite.name}`);
    
    const startTime = Date.now();
    const child = spawn('node', [scriptPath], {
      stdio: ['pipe', 'pipe', 'pipe'],
      cwd: path.dirname(scriptPath)
    });
    
    let stdout = '';
    let stderr = '';
    
    child.stdout.on('data', (data) => {
      stdout += data.toString();
    });
    
    child.stderr.on('data', (data) => {
      stderr += data.toString();
    });
    
    child.on('close', (code) => {
      const endTime = Date.now();
      const executionTime = endTime - startTime;
      
      // Parse test results from stdout
      let summary = { totalTests: 0, passed: 0, failed: 0, warnings: 0 };
      let reportPath = null;
      
      try {
        // Look for summary in stdout
        const summaryMatch = stdout.match(/Total Tests: (\d+)/);
        const passedMatch = stdout.match(/Passed: (\d+)/);
        const failedMatch = stdout.match(/Failed: (\d+)/);
        const warningsMatch = stdout.match(/Warnings: (\d+)/);
        const reportMatch = stdout.match(/Report Saved: (.+)/);
        
        if (summaryMatch) summary.totalTests = parseInt(summaryMatch[1]);
        if (passedMatch) summary.passed = parseInt(passedMatch[1]);
        if (failedMatch) summary.failed = parseInt(failedMatch[1]);
        if (warningsMatch) summary.warnings = parseInt(warningsMatch[1]);
        if (reportMatch) reportPath = reportMatch[1];
        
      } catch (error) {
        log('warn', `Failed to parse test results for ${suite.name}: ${error.message}`);
      }
      
      const result = {
        name: suite.name,
        script: suite.script,
        priority: suite.priority,
        description: suite.description,
        status: code === 0 ? 'passed' : 'failed',
        exitCode: code,
        executionTime,
        summary,
        reportPath,
        stdout: stdout.slice(-1000), // Last 1000 chars
        stderr: stderr.slice(-1000)  // Last 1000 chars
      };
      
      if (code === 0) {
        log('info', `✅ Test suite completed: ${suite.name}`, {
          tests: summary.totalTests,
          passed: summary.passed,
          failed: summary.failed,
          time: `${executionTime}ms`
        });
      } else {
        log('error', `❌ Test suite failed: ${suite.name}`, {
          exitCode: code,
          error: stderr || 'Unknown error',
          time: `${executionTime}ms`
        });
      }
      
      resolve(result);
    });
    
    child.on('error', (error) => {
      const endTime = Date.now();
      const executionTime = endTime - startTime;
      
      resolve({
        name: suite.name,
        status: 'failed',
        error: error.message,
        executionTime,
        summary: { totalTests: 0, passed: 0, failed: 1, warnings: 0 }
      });
    });
  });
}

// Run test suites in sequence
async function runAllTestSuites() {
  log('info', '🚀 Starting Master Test Runner');
  log('info', `Running ${TEST_SUITES.length} test suites...`);
  
  const overallStartTime = Date.now();
  
  // Ensure results directory exists
  if (!fs.existsSync(RESULTS_DIR)) {
    fs.mkdirSync(RESULTS_DIR, { recursive: true });
  }
  
  // Run each test suite
  for (const suite of TEST_SUITES) {
    try {
      const result = await runTestSuite(suite);
      masterResults.suites.push(result);
      
      // Update summary
      masterResults.summary.completedSuites++;
      if (result.status === 'passed') {
        masterResults.summary.passedSuites++;
      } else {
        masterResults.summary.failedSuites++;
      }
      
      masterResults.summary.totalTests += result.summary.totalTests;
      masterResults.summary.totalPassed += result.summary.passed;
      masterResults.summary.totalFailed += result.summary.failed;
      masterResults.summary.totalWarnings += result.summary.warnings;
      
    } catch (error) {
      log('error', `Failed to run test suite ${suite.name}: ${error.message}`);
      
      masterResults.suites.push({
        name: suite.name,
        status: 'failed',
        error: error.message,
        executionTime: 0,
        summary: { totalTests: 0, passed: 0, failed: 1, warnings: 0 }
      });
      
      masterResults.summary.completedSuites++;
      masterResults.summary.failedSuites++;
      masterResults.summary.totalFailed++;
    }
  }
  
  const overallEndTime = Date.now();
  const totalExecutionTime = overallEndTime - overallStartTime;
  
  // Calculate final metrics
  masterResults.summary.executionTime = `${totalExecutionTime}ms`;
  masterResults.summary.overallSuccessRate = masterResults.summary.totalTests > 0 
    ? `${(masterResults.summary.totalPassed / masterResults.summary.totalTests * 100).toFixed(1)}%`
    : '0%';
  
  // Generate comprehensive report
  const report = {
    ...masterResults,
    analysis: {
      suiteSuccessRate: `${(masterResults.summary.passedSuites / masterResults.summary.totalSuites * 100).toFixed(1)}%`,
      averageExecutionTime: `${Math.round(totalExecutionTime / masterResults.summary.totalSuites)}ms`,
      highPrioritySuites: masterResults.suites.filter(s => s.priority === 'high'),
      failedSuites: masterResults.suites.filter(s => s.status === 'failed'),
      criticalIssues: masterResults.suites.filter(s => s.status === 'failed' && s.priority === 'high').length,
      recommendations: []
    }
  };
  
  // Generate recommendations
  if (report.analysis.criticalIssues > 0) {
    report.analysis.recommendations.push('🚨 Critical high-priority test suites failed - immediate attention required');
  }
  
  if (masterResults.summary.totalFailed > 0) {
    report.analysis.recommendations.push(`⚠️ ${masterResults.summary.totalFailed} individual tests failed - review detailed reports`);
  }
  
  if (masterResults.summary.totalWarnings > 0) {
    report.analysis.recommendations.push(`⚡ ${masterResults.summary.totalWarnings} warnings detected - consider addressing for robustness`);
  }
  
  if (report.analysis.failedSuites.length === 0) {
    report.analysis.recommendations.push('✅ All test suites passed - system is stable and ready for production');
  }
  
  // Save master report
  const reportPath = path.join(RESULTS_DIR, `master-test-report-${TIMESTAMP}.json`);
  fs.writeFileSync(reportPath, JSON.stringify(report, null, 2));
  
  // Print comprehensive summary
  console.log('\n' + '='.repeat(80));
  console.log('📊 MASTER TEST RUNNER COMPLETE');
  console.log('='.repeat(80));
  
  console.log('\n🎯 OVERALL SUMMARY:');
  console.log(`   Total Test Suites: ${masterResults.summary.totalSuites}`);
  console.log(`   Passed Suites: ${masterResults.summary.passedSuites}`);
  console.log(`   Failed Suites: ${masterResults.summary.failedSuites}`);
  console.log(`   Suite Success Rate: ${report.analysis.suiteSuccessRate}`);
  
  console.log('\n📈 TEST METRICS:');
  console.log(`   Total Individual Tests: ${masterResults.summary.totalTests}`);
  console.log(`   Passed Tests: ${masterResults.summary.totalPassed}`);
  console.log(`   Failed Tests: ${masterResults.summary.totalFailed}`);
  console.log(`   Warnings: ${masterResults.summary.totalWarnings}`);
  console.log(`   Overall Success Rate: ${masterResults.summary.overallSuccessRate}`);
  
  console.log('\n⏱️ PERFORMANCE:');
  console.log(`   Total Execution Time: ${masterResults.summary.executionTime}`);
  console.log(`   Average Suite Time: ${report.analysis.averageExecutionTime}`);
  
  console.log('\n📋 SUITE DETAILS:');
  masterResults.suites.forEach(suite => {
    const status = suite.status === 'passed' ? '✅' : '❌';
    const priority = suite.priority === 'high' ? '🔥' : '📋';
    console.log(`   ${status} ${priority} ${suite.name}: ${suite.summary.passed}/${suite.summary.totalTests} tests (${suite.executionTime}ms)`);
  });
  
  if (report.analysis.recommendations.length > 0) {
    console.log('\n💡 RECOMMENDATIONS:');
    report.analysis.recommendations.forEach(rec => {
      console.log(`   ${rec}`);
    });
  }
  
  console.log(`\n📄 Master Report Saved: ${reportPath}`);
  console.log('='.repeat(80));
  
  // Exit with appropriate code
  const exitCode = masterResults.summary.failedSuites > 0 ? 1 : 0;
  process.exit(exitCode);
}

// Handle command line arguments
function parseArguments() {
  const args = process.argv.slice(2);
  const options = {
    suites: [],
    priority: null,
    verbose: false,
    help: false
  };
  
  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    
    if (arg === '--help' || arg === '-h') {
      options.help = true;
    } else if (arg === '--verbose' || arg === '-v') {
      options.verbose = true;
    } else if (arg === '--priority' || arg === '-p') {
      options.priority = args[++i];
    } else if (arg === '--suites' || arg === '-s') {
      options.suites = args[++i].split(',');
    }
  }
  
  return options;
}

function printHelp() {
  console.log(`
🧪 Master Test Runner - Project Beacon Test Suite

USAGE:
  node test-master-runner.js [options]

OPTIONS:
  -h, --help              Show this help message
  -v, --verbose           Enable verbose output
  -p, --priority <level>  Run only tests with specified priority (high|medium|low)
  -s, --suites <names>    Run only specified test suites (comma-separated)

EXAMPLES:
  node test-master-runner.js                           # Run all test suites
  node test-master-runner.js --priority high           # Run only high priority suites
  node test-master-runner.js --suites "API Data Validation,Portal State Management"
  node test-master-runner.js --verbose                 # Run with detailed output

AVAILABLE TEST SUITES:
${TEST_SUITES.map(suite => `  • ${suite.name} (${suite.priority}) - ${suite.description}`).join('\n')}

REPORTS:
  Detailed reports are saved to: ${RESULTS_DIR}
  Master report includes comprehensive analysis and recommendations.
`);
}

// Main execution
async function main() {
  const options = parseArguments();
  
  if (options.help) {
    printHelp();
    process.exit(0);
  }
  
  // Filter test suites based on options
  let suitesToRun = [...TEST_SUITES];
  
  if (options.priority) {
    suitesToRun = suitesToRun.filter(suite => suite.priority === options.priority);
    log('info', `Filtering by priority: ${options.priority}`);
  }
  
  if (options.suites.length > 0) {
    suitesToRun = suitesToRun.filter(suite => options.suites.includes(suite.name));
    log('info', `Running specific suites: ${options.suites.join(', ')}`);
  }
  
  if (suitesToRun.length === 0) {
    log('error', 'No test suites match the specified criteria');
    process.exit(1);
  }
  
  // Update TEST_SUITES for execution
  TEST_SUITES.length = 0;
  TEST_SUITES.push(...suitesToRun);
  
  // Run the test suites
  await runAllTestSuites();
}

// Run if called directly
if (require.main === module) {
  main().catch(error => {
    console.error('Fatal error in master test runner:', error);
    process.exit(1);
  });
}

module.exports = {
  runAllTestSuites,
  runTestSuite,
  TEST_SUITES
};
