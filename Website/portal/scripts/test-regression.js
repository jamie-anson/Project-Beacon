#!/usr/bin/env node

/**
 * Regression Test Suite for Project Beacon Portal
 * 
 * This script runs specific tests to prevent the following issues from recurring:
 * 1. Prompt Structure Bug: "I'm sorry, but I can't assist with that" responses
 * 2. Region Filtering Bug: "No executions match current filters" 
 * 3. Multi-Model Display Issues: Incorrect progress indicators
 */

import { execSync } from 'child_process';
import chalk from 'chalk';

const REGRESSION_TESTS = [
  // Prompt Structure Tests
  {
    name: 'Prompt Structure - useBiasDetection Hook',
    pattern: 'src/hooks/__tests__/useBiasDetection.test.js',
    description: 'Ensures job specifications include proper prompt data in benchmark.input.data'
  },
  
  // Region Filtering Tests  
  {
    name: 'Region Filtering - Executions Page',
    pattern: 'src/pages/__tests__/Executions.test.jsx',
    description: 'Verifies executions page filters by correct database region names (us-east, eu-west, asia-pacific)'
  },
  
  // Multi-Model Display Tests
  {
    name: 'Multi-Model Display - LiveProgressTable',
    pattern: 'src/components/bias-detection/__tests__/LiveProgressTable.test.jsx',
    description: 'Validates multi-model execution progress display and Answer link region mapping'
  },
  
  // Integration Tests
  {
    name: 'End-to-End Integration Flow',
    pattern: 'src/__tests__/integration/BiasDetectionFlow.test.jsx',
    description: 'Tests complete flow from job submission to execution viewing'
  }
];

function runTest(test) {
  console.log(chalk.blue(`\n🧪 Running: ${test.name}`));
  console.log(chalk.gray(`   ${test.description}`));
  
  try {
    const result = execSync(`npm test -- --testPathPattern="${test.pattern}" --verbose`, {
      encoding: 'utf8',
      stdio: 'pipe'
    });
    
    console.log(chalk.green(`✅ PASSED: ${test.name}`));
    return true;
  } catch (error) {
    console.log(chalk.red(`❌ FAILED: ${test.name}`));
    console.log(chalk.red(error.stdout || error.message));
    return false;
  }
}

function main() {
  console.log(chalk.bold.cyan('\n🔍 Project Beacon - Regression Test Suite\n'));
  console.log(chalk.yellow('Testing for known issues:'));
  console.log(chalk.yellow('• Prompt structure bugs (malformed job specs)'));
  console.log(chalk.yellow('• Region filtering bugs (UI/DB mapping mismatch)'));
  console.log(chalk.yellow('• Multi-model display issues (incorrect progress)'));
  
  let passed = 0;
  let failed = 0;
  
  for (const test of REGRESSION_TESTS) {
    if (runTest(test)) {
      passed++;
    } else {
      failed++;
    }
  }
  
  console.log(chalk.bold('\n📊 Test Results:'));
  console.log(chalk.green(`✅ Passed: ${passed}`));
  console.log(chalk.red(`❌ Failed: ${failed}`));
  
  if (failed === 0) {
    console.log(chalk.bold.green('\n🎉 All regression tests passed! The fixes are working correctly.'));
    process.exit(0);
  } else {
    console.log(chalk.bold.red('\n💥 Some regression tests failed! Please review the issues above.'));
    process.exit(1);
  }
}

// Run if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
  main();
}

export { REGRESSION_TESTS, runTest };
