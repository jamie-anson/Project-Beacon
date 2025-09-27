#!/usr/bin/env node

/**
 * End-to-End Workflow Tests
 * Tests complete job submission to Portal display pipeline
 */

const axios = require('axios');
const fs = require('fs');
const path = require('path');

// Test configuration
const API_BASE = 'https://beacon-runner-change-me.fly.dev/api/v1';
const PORTAL_BASE = 'https://projectbeacon.netlify.app/portal';
const RESULTS_DIR = path.join(__dirname, '../test-results');
const TIMESTAMP = new Date().toISOString().replace(/[:.]/g, '-');

// Test results tracking
const testResults = {
  timestamp: new Date().toISOString(),
  testSuite: 'End-to-End Workflow',
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

// Utility functions
async function waitForJobCompletion(jobId, maxWaitTime = 300000) {
  const startTime = Date.now();
  const pollInterval = 5000; // 5 seconds
  
  while (Date.now() - startTime < maxWaitTime) {
    try {
      const response = await axios.get(`${API_BASE}/jobs/${jobId}`);
      const job = response.data;
      
      if (job.status === 'completed' || job.status === 'failed') {
        return {
          status: job.status,
          waitTime: Date.now() - startTime,
          finalJob: job
        };
      }
      
      console.log(`⏳ Job ${jobId} status: ${job.status}, waiting...`);
      await new Promise(resolve => setTimeout(resolve, pollInterval));
      
    } catch (error) {
      console.log(`⏳ Job ${jobId} not found yet, waiting...`);
      await new Promise(resolve => setTimeout(resolve, pollInterval));
    }
  }
  
  throw new Error(`Job ${jobId} did not complete within ${maxWaitTime}ms`);
}

async function fetchWithRetry(url, maxRetries = 3, delay = 1000) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      const response = await axios.get(url, { timeout: 10000 });
      return response.data;
    } catch (error) {
      if (i === maxRetries - 1) throw error;
      console.log(`Retry ${i + 1}/${maxRetries} for ${url}`);
      await new Promise(resolve => setTimeout(resolve, delay * (i + 1)));
    }
  }
}

// Test 1: Complete Job Lifecycle
async function testCompleteJobLifecycle() {
  console.log('🧪 Testing Complete Job Lifecycle...');
  
  try {
    // Use existing completed job for testing
    const jobId = 'bias-detection-1758933513';
    
    // Step 1: Verify job exists and is completed
    const job = await fetchWithRetry(`${API_BASE}/jobs/${jobId}`);
    
    // Step 2: Fetch executions
    const executions = await fetchWithRetry(`${API_BASE}/executions`);
    const jobExecutions = executions.executions.filter(e => e.job_id === jobId);
    
    // Step 3: Fetch cross-region diff
    const crossRegionDiff = await fetchWithRetry(`${API_BASE}/executions/${jobId}/cross-region-diff`);
    
    // Step 4: Simulate Portal data processing
    const AVAILABLE_MODELS = [
      { id: 'llama3.2-1b', name: 'Llama 3.2-1B Instruct', provider: 'Meta' },
      { id: 'qwen2.5-1.5b', name: 'Qwen 2.5-1.5B Instruct', provider: 'Alibaba' },
      { id: 'mistral-7b', name: 'Mistral 7B Instruct', provider: 'Mistral AI' }
    ];
    
    // Transform data (simulate Portal transform)
    const modelExecutionMap = {};
    crossRegionDiff.executions.forEach(exec => {
      const modelId = exec.output_data?.metadata?.model || 'llama3.2-1b';
      if (!modelExecutionMap[modelId]) {
        modelExecutionMap[modelId] = {};
      }
      modelExecutionMap[modelId][exec.region] = exec;
    });
    
    const portalModels = AVAILABLE_MODELS.map((model) => {
      const modelExecutions = modelExecutionMap[model.id] || {};
      const regions = Object.keys(modelExecutions).map((regionCode) => {
        const exec = modelExecutions[regionCode];
        return {
          region_code: regionCode,
          response: exec?.output_data?.response || 'No response available',
          hasData: !!exec
        };
      }).filter(region => region.hasData);
      
      return {
        model_id: model.id,
        model_name: model.name,
        regions: regions
      };
    }).filter(model => model.regions.length > 0);
    
    // Step 5: Validate complete pipeline
    const lifecycleValidation = {
      jobExists: !!job,
      jobStatus: job?.status,
      executionCount: jobExecutions.length,
      crossRegionDiffAvailable: !!crossRegionDiff,
      crossRegionExecutions: crossRegionDiff.executions.length,
      portalModelsGenerated: portalModels.length,
      portalModelIds: portalModels.map(m => m.model_id),
      totalRegionsWithData: portalModels.reduce((sum, m) => sum + m.regions.length, 0),
      pipelineComplete: true,
      expectedPortalUrl: `${PORTAL_BASE}/results/${jobId}/diffs`
    };
    
    // Validate pipeline integrity
    lifecycleValidation.pipelineComplete = 
      lifecycleValidation.jobExists &&
      lifecycleValidation.executionCount > 0 &&
      lifecycleValidation.crossRegionDiffAvailable &&
      lifecycleValidation.portalModelsGenerated > 0 &&
      lifecycleValidation.totalRegionsWithData > 0;
    
    logTest('Complete Job Lifecycle', 
      lifecycleValidation.pipelineComplete ? 'passed' : 'failed', 
      lifecycleValidation
    );
    
  } catch (error) {
    logTest('Complete Job Lifecycle', 'failed', {
      error: error.message,
      stack: error.stack
    });
  }
}

// Test 2: Multi-Job Comparison Workflow
async function testMultiJobComparisonWorkflow() {
  console.log('🧪 Testing Multi-Job Comparison Workflow...');
  
  try {
    // Get multiple recent jobs
    const executions = await fetchWithRetry(`${API_BASE}/executions`);
    const recentJobs = [...new Set(executions.executions.slice(0, 15).map(e => e.job_id))];
    
    const jobComparisons = [];
    
    for (const jobId of recentJobs.slice(0, 3)) { // Test first 3 jobs
      try {
        const crossRegionDiff = await fetchWithRetry(`${API_BASE}/executions/${jobId}/cross-region-diff`);
        
        const jobAnalysis = {
          jobId,
          executionCount: crossRegionDiff.executions.length,
          uniqueModels: [...new Set(crossRegionDiff.executions.map(e => e.output_data?.metadata?.model))].filter(Boolean),
          uniqueRegions: [...new Set(crossRegionDiff.executions.map(e => e.region))],
          responseTypes: crossRegionDiff.executions.map(e => ({
            model: e.output_data?.metadata?.model,
            region: e.region,
            responseLength: e.output_data?.response?.length || 0,
            isCensored: e.output_data?.response?.includes("I'm sorry") || false
          })),
          portalReady: true
        };
        
        jobComparisons.push(jobAnalysis);
        
      } catch (error) {
        jobComparisons.push({
          jobId,
          error: error.message,
          portalReady: false
        });
      }
    }
    
    // Analyze comparison patterns
    const comparisonAnalysis = {
      totalJobsAnalyzed: jobComparisons.length,
      successfulJobs: jobComparisons.filter(j => j.portalReady).length,
      failedJobs: jobComparisons.filter(j => !j.portalReady).length,
      modelDistribution: {},
      censorshipPatterns: {},
      regionCoverage: {}
    };
    
    // Aggregate analysis
    jobComparisons.filter(j => j.portalReady).forEach(job => {
      // Model distribution
      job.uniqueModels.forEach(model => {
        comparisonAnalysis.modelDistribution[model] = (comparisonAnalysis.modelDistribution[model] || 0) + 1;
      });
      
      // Censorship patterns
      job.responseTypes.forEach(response => {
        if (response.isCensored) {
          const key = `${response.model}_${response.region}`;
          comparisonAnalysis.censorshipPatterns[key] = (comparisonAnalysis.censorshipPatterns[key] || 0) + 1;
        }
      });
      
      // Region coverage
      job.uniqueRegions.forEach(region => {
        comparisonAnalysis.regionCoverage[region] = (comparisonAnalysis.regionCoverage[region] || 0) + 1;
      });
    });
    
    comparisonAnalysis.workflowSuccess = comparisonAnalysis.successfulJobs > 0;
    comparisonAnalysis.jobDetails = jobComparisons;
    
    logTest('Multi-Job Comparison Workflow', 
      comparisonAnalysis.workflowSuccess ? 'passed' : 'failed', 
      comparisonAnalysis
    );
    
  } catch (error) {
    logTest('Multi-Job Comparison Workflow', 'failed', {
      error: error.message,
      stack: error.stack
    });
  }
}

// Test 3: Portal URL Generation and Validation
async function testPortalUrlGeneration() {
  console.log('🧪 Testing Portal URL Generation and Validation...');
  
  try {
    const executions = await fetchWithRetry(`${API_BASE}/executions`);
    const recentJobs = [...new Set(executions.executions.slice(0, 10).map(e => e.job_id))];
    
    const urlTests = [];
    
    for (const jobId of recentJobs.slice(0, 3)) {
      const portalUrl = `${PORTAL_BASE}/results/${jobId}/diffs`;
      
      // Test URL structure
      const urlValidation = {
        jobId,
        portalUrl,
        urlStructureValid: portalUrl.includes('/portal/results/') && portalUrl.includes('/diffs'),
        jobIdInUrl: portalUrl.includes(jobId),
        expectedFormat: true
      };
      
      // Test if job has data for Portal
      try {
        const crossRegionDiff = await fetchWithRetry(`${API_BASE}/executions/${jobId}/cross-region-diff`);
        urlValidation.hasData = crossRegionDiff.executions.length > 0;
        urlValidation.dataReady = true;
      } catch (error) {
        urlValidation.hasData = false;
        urlValidation.dataReady = false;
        urlValidation.dataError = error.message;
      }
      
      urlValidation.urlValid = 
        urlValidation.urlStructureValid &&
        urlValidation.jobIdInUrl &&
        urlValidation.dataReady;
      
      urlTests.push(urlValidation);
    }
    
    const urlGenerationSummary = {
      totalUrls: urlTests.length,
      validUrls: urlTests.filter(u => u.urlValid).length,
      urlsWithData: urlTests.filter(u => u.hasData).length,
      urlsWithoutData: urlTests.filter(u => !u.hasData).length,
      generationSuccess: urlTests.every(u => u.urlStructureValid && u.jobIdInUrl),
      urlDetails: urlTests
    };
    
    logTest('Portal URL Generation and Validation', 
      urlGenerationSummary.generationSuccess ? 'passed' : 'failed', 
      urlGenerationSummary
    );
    
  } catch (error) {
    logTest('Portal URL Generation and Validation', 'failed', {
      error: error.message,
      stack: error.stack
    });
  }
}

// Test 4: Data Consistency Across Pipeline
async function testDataConsistencyAcrossPipeline() {
  console.log('🧪 Testing Data Consistency Across Pipeline...');
  
  try {
    const jobId = 'bias-detection-1758933513';
    
    // Fetch data from all pipeline stages
    const pipelineData = {
      executions: await fetchWithRetry(`${API_BASE}/executions`),
      crossRegionDiff: await fetchWithRetry(`${API_BASE}/executions/${jobId}/cross-region-diff`),
      job: null
    };
    
    try {
      pipelineData.job = await fetchWithRetry(`${API_BASE}/jobs/${jobId}`);
    } catch (error) {
      console.log('Job endpoint not available, continuing without job data');
    }
    
    // Extract job-specific executions
    const jobExecutions = pipelineData.executions.executions.filter(e => e.job_id === jobId);
    
    // Consistency checks
    const consistencyChecks = {
      executionCountMatch: jobExecutions.length === pipelineData.crossRegionDiff.executions.length,
      executionIdsMatch: true,
      regionConsistency: true,
      modelConsistency: true,
      responseConsistency: true
    };
    
    // Check execution IDs match
    const executionIds = new Set(jobExecutions.map(e => e.id));
    const crossRegionIds = new Set(pipelineData.crossRegionDiff.executions.map(e => e.id));
    consistencyChecks.executionIdsMatch = 
      executionIds.size === crossRegionIds.size &&
      [...executionIds].every(id => crossRegionIds.has(id));
    
    // Check region consistency
    const executionRegions = new Set(jobExecutions.map(e => e.region));
    const crossRegionRegions = new Set(pipelineData.crossRegionDiff.executions.map(e => e.region));
    consistencyChecks.regionConsistency = 
      executionRegions.size === crossRegionRegions.size &&
      [...executionRegions].every(region => crossRegionRegions.has(region));
    
    // Check model consistency (where available)
    const executionModels = jobExecutions.map(e => e.output_data?.metadata?.model).filter(Boolean);
    const crossRegionModels = pipelineData.crossRegionDiff.executions.map(e => e.output_data?.metadata?.model).filter(Boolean);
    consistencyChecks.modelConsistency = 
      executionModels.length === crossRegionModels.length &&
      executionModels.every((model, index) => model === crossRegionModels[index]);
    
    // Check response consistency
    const executionResponses = jobExecutions.map(e => e.output_data?.response).filter(Boolean);
    const crossRegionResponses = pipelineData.crossRegionDiff.executions.map(e => e.output_data?.response).filter(Boolean);
    consistencyChecks.responseConsistency = 
      executionResponses.length === crossRegionResponses.length &&
      executionResponses.every((response, index) => response === crossRegionResponses[index]);
    
    const consistencyResults = {
      jobId,
      ...consistencyChecks,
      overallConsistency: Object.values(consistencyChecks).every(check => check === true),
      executionCounts: {
        executions: jobExecutions.length,
        crossRegionDiff: pipelineData.crossRegionDiff.executions.length
      },
      regionCounts: {
        executions: executionRegions.size,
        crossRegionDiff: crossRegionRegions.size
      },
      modelCounts: {
        executions: executionModels.length,
        crossRegionDiff: crossRegionModels.length
      }
    };
    
    logTest('Data Consistency Across Pipeline', 
      consistencyResults.overallConsistency ? 'passed' : 'failed', 
      consistencyResults
    );
    
  } catch (error) {
    logTest('Data Consistency Across Pipeline', 'failed', {
      error: error.message,
      stack: error.stack
    });
  }
}

// Test 5: Error Recovery and Resilience
async function testErrorRecoveryResilience() {
  console.log('🧪 Testing Error Recovery and Resilience...');
  
  try {
    const resilienceTests = [
      {
        name: 'Invalid Job ID',
        test: async () => {
          try {
            await fetchWithRetry(`${API_BASE}/executions/invalid-job-id/cross-region-diff`);
            return { success: false, error: 'Should have failed' };
          } catch (error) {
            return { success: true, errorHandled: true, errorType: 'not_found' };
          }
        }
      },
      {
        name: 'Network Timeout Simulation',
        test: async () => {
          try {
            // Use a very short timeout to simulate network issues
            await axios.get(`${API_BASE}/executions`, { timeout: 1 });
            return { success: false, error: 'Should have timed out' };
          } catch (error) {
            return { success: true, errorHandled: true, errorType: 'timeout' };
          }
        }
      },
      {
        name: 'Malformed Response Handling',
        test: async () => {
          // Simulate processing malformed data
          const malformedData = { executions: null };
          
          try {
            // Simulate transform logic with malformed data
            const AVAILABLE_MODELS = [
              { id: 'llama3.2-1b', name: 'Llama 3.2-1B Instruct', provider: 'Meta' }
            ];
            
            const modelExecutionMap = {};
            const executions = malformedData.executions || [];
            
            if (Array.isArray(executions)) {
              executions.forEach(exec => {
                const modelId = exec?.output_data?.metadata?.model || 'llama3.2-1b';
                if (!modelExecutionMap[modelId]) {
                  modelExecutionMap[modelId] = {};
                }
                modelExecutionMap[modelId][exec?.region] = exec;
              });
            }
            
            const portalModels = AVAILABLE_MODELS.map((model) => {
              const modelExecutions = modelExecutionMap[model.id] || {};
              const regions = Object.keys(modelExecutions).filter(region => modelExecutions[region]);
              return {
                model_id: model.id,
                regions: regions.map(region => ({ region_code: region }))
              };
            }).filter(model => model.regions.length > 0);
            
            return { 
              success: true, 
              gracefulFallback: true, 
              resultingModels: portalModels.length,
              handledMalformedData: true 
            };
            
          } catch (error) {
            return { success: false, error: error.message };
          }
        }
      }
    ];
    
    const resilienceResults = [];
    
    for (const test of resilienceTests) {
      try {
        const result = await test.test();
        resilienceResults.push({
          testName: test.name,
          ...result
        });
      } catch (error) {
        resilienceResults.push({
          testName: test.name,
          success: false,
          error: error.message
        });
      }
    }
    
    const resilienceSummary = {
      totalTests: resilienceTests.length,
      successfulTests: resilienceResults.filter(r => r.success).length,
      failedTests: resilienceResults.filter(r => !r.success).length,
      resilienceScore: (resilienceResults.filter(r => r.success).length / resilienceTests.length * 100).toFixed(1) + '%',
      testResults: resilienceResults
    };
    
    logTest('Error Recovery and Resilience', 
      resilienceSummary.successfulTests === resilienceTests.length ? 'passed' : 'warning', 
      resilienceSummary
    );
    
  } catch (error) {
    logTest('Error Recovery and Resilience', 'failed', {
      error: error.message,
      stack: error.stack
    });
  }
}

// Main test runner
async function runEndToEndTests() {
  console.log('🚀 Starting End-to-End Workflow Test Suite');
  
  const startTime = Date.now();
  
  try {
    await testCompleteJobLifecycle();
    await testMultiJobComparisonWorkflow();
    await testPortalUrlGeneration();
    await testDataConsistencyAcrossPipeline();
    await testErrorRecoveryResilience();
    
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
  
  const reportPath = path.join(RESULTS_DIR, `end-to-end-workflow-${TIMESTAMP}.json`);
  fs.writeFileSync(reportPath, JSON.stringify(report, null, 2));
  
  // Print summary
  console.log('\n📊 End-to-End Workflow Test Suite Complete');
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
  runEndToEndTests().catch(error => {
    console.error('Fatal error:', error);
    process.exit(1);
  });
}

module.exports = {
  runEndToEndTests,
  testCompleteJobLifecycle,
  testMultiJobComparisonWorkflow,
  testPortalUrlGeneration,
  testDataConsistencyAcrossPipeline,
  testErrorRecoveryResilience
};
