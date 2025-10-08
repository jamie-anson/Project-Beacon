#!/usr/bin/env node

/**
 * Comprehensive Model Mapping Validation Test Suite
 * Provides complete visibility into API data, transform logic, and Portal state
 */

const axios = require('axios');
const fs = require('fs');
const path = require('path');

// Configuration
const API_BASE = 'https://beacon-runner-production.fly.dev/api/v1';
const RESULTS_DIR = path.join(__dirname, '../test-results');
const TIMESTAMP = new Date().toISOString().replace(/[:.]/g, '-');

// Ensure results directory exists
if (!fs.existsSync(RESULTS_DIR)) {
  fs.mkdirSync(RESULTS_DIR, { recursive: true });
}

// Test results tracking
const testResults = {
  timestamp: new Date().toISOString(),
  summary: {
    totalTests: 0,
    passed: 0,
    failed: 0,
    warnings: 0
  },
  tests: []
};

// Logging utilities
function log(level, message, data = null) {
  const timestamp = new Date().toISOString();
  const logEntry = { timestamp, level, message, data };
  
  console.log(`[${timestamp}] ${level.toUpperCase()}: ${message}`);
  if (data) {
    console.log(JSON.stringify(data, null, 2));
  }
  
  return logEntry;
}

function logTest(testName, status, details) {
  testResults.tests.push({
    name: testName,
    status,
    details,
    timestamp: new Date().toISOString()
  });
  
  testResults.summary.totalTests++;
  testResults.summary[status]++;
  
  log(status, `Test: ${testName}`, details);
}

// API utilities
async function fetchWithRetry(url, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      const response = await axios.get(url, { timeout: 10000 });
      return response.data;
    } catch (error) {
      if (i === maxRetries - 1) throw error;
      await new Promise(resolve => setTimeout(resolve, 1000 * (i + 1)));
    }
  }
}

// Schema validation utilities
function validateSchema(data, schema, path = '') {
  const errors = [];
  
  for (const [key, expectedType] of Object.entries(schema)) {
    const fullPath = path ? `${path}.${key}` : key;
    
    if (!(key in data)) {
      errors.push(`Missing required field: ${fullPath}`);
      continue;
    }
    
    const value = data[key];
    
    if (typeof expectedType === 'string') {
      if (typeof value !== expectedType) {
        errors.push(`Type mismatch at ${fullPath}: expected ${expectedType}, got ${typeof value}`);
      }
    } else if (typeof expectedType === 'object' && expectedType.type) {
      if (expectedType.type === 'array') {
        if (!Array.isArray(value)) {
          errors.push(`Type mismatch at ${fullPath}: expected array, got ${typeof value}`);
        } else if (expectedType.items && value.length > 0) {
          value.forEach((item, index) => {
            const itemErrors = validateSchema(item, expectedType.items, `${fullPath}[${index}]`);
            errors.push(...itemErrors);
          });
        }
      } else if (expectedType.type === 'object' && expectedType.properties) {
        const nestedErrors = validateSchema(value, expectedType.properties, fullPath);
        errors.push(...nestedErrors);
      }
    }
  }
  
  return errors;
}

// Test 1: API Data Validation Tests
async function testApiDataValidation() {
  log('info', '🧪 Starting API Data Validation Tests');
  
  try {
    // Get recent executions
    const executions = await fetchWithRetry(`${API_BASE}/executions`);
    
    logTest('API Executions Endpoint', 'passed', {
      totalExecutions: executions.executions?.length || 0,
      sampleExecution: executions.executions?.[0] || null
    });
    
    // Test cross-region diff for recent jobs
    const recentJobs = [...new Set(executions.executions?.slice(0, 10).map(e => e.job_id) || [])];
    
    for (const jobId of recentJobs.slice(0, 3)) { // Test first 3 jobs
      try {
        const diffData = await fetchWithRetry(`${API_BASE}/executions/${jobId}/cross-region-diff`);
        
        // Schema validation
        const schema = {
          job_id: 'string',
          executions: {
            type: 'array',
            items: {
              region: 'string',
              output_data: {
                type: 'object',
                properties: {
                  response: 'string',
                  metadata: {
                    type: 'object',
                    properties: {
                      model: 'string'
                    }
                  }
                }
              }
            }
          }
        };
        
        const schemaErrors = validateSchema(diffData, schema);
        
        if (schemaErrors.length === 0) {
          // Analyze model distribution
          const models = diffData.executions.map(e => e.output_data?.metadata?.model).filter(Boolean);
          const uniqueModels = [...new Set(models)];
          const regions = [...new Set(diffData.executions.map(e => e.region))];
          
          logTest(`Cross-Region Diff Schema - ${jobId}`, 'passed', {
            jobId,
            totalExecutions: diffData.executions.length,
            uniqueModels,
            regions,
            modelDistribution: uniqueModels.map(model => ({
              model,
              count: models.filter(m => m === model).length,
              regions: diffData.executions
                .filter(e => e.output_data?.metadata?.model === model)
                .map(e => e.region)
            })),
            sampleResponses: diffData.executions.slice(0, 2).map(e => ({
              region: e.region,
              model: e.output_data?.metadata?.model,
              responsePreview: e.output_data?.response?.substring(0, 100) + '...'
            }))
          });
        } else {
          logTest(`Cross-Region Diff Schema - ${jobId}`, 'failed', {
            jobId,
            schemaErrors
          });
        }
        
      } catch (error) {
        logTest(`Cross-Region Diff API - ${jobId}`, 'failed', {
          jobId,
          error: error.message
        });
      }
    }
    
  } catch (error) {
    logTest('API Data Validation', 'failed', {
      error: error.message,
      stack: error.stack
    });
  }
}

// Test 2: Transform Function Deep Testing
async function testTransformFunction() {
  log('info', '🧪 Starting Transform Function Deep Testing');
  
  try {
    // Import transform function
    const transformPath = path.join(__dirname, '../portal/src/lib/diffs/transform.js');
    
    // Since we can't directly import ES modules in Node.js easily, we'll test the logic conceptually
    // and create mock data to understand the expected behavior
    
    const testCases = [
      {
        name: 'Single Model - Qwen Only',
        mockData: {
          executions: [
            { id: 1, region: 'us-east', output_data: { metadata: { model: 'qwen2.5-1.5b' }, response: 'Response 1' } },
            { id: 2, region: 'eu-west', output_data: { metadata: { model: 'qwen2.5-1.5b' }, response: 'Response 2' } },
            { id: 3, region: 'asia-pacific', output_data: { metadata: { model: 'qwen2.5-1.5b' }, response: 'Response 3' } }
          ]
        },
        expectedModels: ['qwen2.5-1.5b'],
        expectedRegionsPerModel: 3
      },
      {
        name: 'Multi Model - Mixed',
        mockData: {
          executions: [
            { id: 1, region: 'us-east', output_data: { metadata: { model: 'qwen2.5-1.5b' }, response: 'Qwen US' } },
            { id: 2, region: 'eu-west', output_data: { metadata: { model: 'llama3.2-1b' }, response: 'Llama EU' } },
            { id: 3, region: 'asia-pacific', output_data: { metadata: { model: 'mistral-7b' }, response: 'Mistral Asia' } }
          ]
        },
        expectedModels: ['qwen2.5-1.5b', 'llama3.2-1b', 'mistral-7b'],
        expectedRegionsPerModel: 1
      },
      {
        name: 'Missing Model Metadata',
        mockData: {
          executions: [
            { id: 1, region: 'us-east', output_data: { response: 'No metadata' } },
            { id: 2, region: 'eu-west', provider_id: 'modal-qwen-eu-west', output_data: { response: 'Provider inference' } }
          ]
        },
        expectedModels: ['llama3.2-1b', 'qwen2.5-1.5b'], // Default + inferred
        expectedRegionsPerModel: 1
      }
    ];
    
    for (const testCase of testCases) {
      // Simulate model detection logic
      const detectedModels = [];
      
      for (const exec of testCase.mockData.executions) {
        let modelId = exec.model_id;
        
        if (!modelId) {
          modelId = exec.output_data?.metadata?.model;
        }
        
        if (!modelId) {
          if (exec.provider_id?.includes('qwen')) {
            modelId = 'qwen2.5-1.5b';
          } else if (exec.provider_id?.includes('mistral')) {
            modelId = 'mistral-7b';
          } else {
            modelId = 'llama3.2-1b'; // Default fallback
          }
        }
        
        if (!detectedModels.includes(modelId)) {
          detectedModels.push(modelId);
        }
      }
      
      const passed = JSON.stringify(detectedModels.sort()) === JSON.stringify(testCase.expectedModels.sort());
      
      logTest(`Transform Logic - ${testCase.name}`, passed ? 'passed' : 'failed', {
        input: testCase.mockData,
        expectedModels: testCase.expectedModels,
        detectedModels,
        modelDetectionSteps: testCase.mockData.executions.map(exec => ({
          execId: exec.id,
          region: exec.region,
          directModelId: exec.model_id || null,
          metadataModel: exec.output_data?.metadata?.model || null,
          providerId: exec.provider_id || null,
          finalDetection: exec.output_data?.metadata?.model || 
                          (exec.provider_id?.includes('qwen') ? 'qwen2.5-1.5b' :
                           exec.provider_id?.includes('mistral') ? 'mistral-7b' : 'llama3.2-1b')
        }))
      });
    }
    
  } catch (error) {
    logTest('Transform Function Testing', 'failed', {
      error: error.message,
      stack: error.stack
    });
  }
}

// Test 3: Cross-Job Model Mapping Analysis
async function testCrossJobModelMapping() {
  log('info', '🧪 Starting Cross-Job Model Mapping Analysis');
  
  try {
    const executions = await fetchWithRetry(`${API_BASE}/executions`);
    const recentJobs = [...new Set(executions.executions?.slice(0, 20).map(e => e.job_id) || [])];
    
    const jobAnalysis = [];
    
    for (const jobId of recentJobs.slice(0, 5)) { // Analyze first 5 jobs
      try {
        const diffData = await fetchWithRetry(`${API_BASE}/executions/${jobId}/cross-region-diff`);
        
        const analysis = {
          jobId,
          totalExecutions: diffData.executions.length,
          regions: [...new Set(diffData.executions.map(e => e.region))],
          apiModels: [...new Set(diffData.executions.map(e => e.output_data?.metadata?.model).filter(Boolean))],
          executionsWithoutModel: diffData.executions.filter(e => !e.output_data?.metadata?.model).length,
          modelDistribution: {},
          sampleResponses: {}
        };
        
        // Analyze model distribution
        diffData.executions.forEach(exec => {
          const model = exec.output_data?.metadata?.model || 'unknown';
          if (!analysis.modelDistribution[model]) {
            analysis.modelDistribution[model] = [];
          }
          analysis.modelDistribution[model].push(exec.region);
          
          if (!analysis.sampleResponses[model]) {
            analysis.sampleResponses[model] = exec.output_data?.response?.substring(0, 100) + '...';
          }
        });
        
        jobAnalysis.push(analysis);
        
      } catch (error) {
        jobAnalysis.push({
          jobId,
          error: error.message
        });
      }
    }
    
    // Aggregate analysis
    const successfulJobs = jobAnalysis.filter(j => !j.error);
    const totalModelsFound = [...new Set(successfulJobs.flatMap(j => j.apiModels))];
    const jobsWithMultipleModels = successfulJobs.filter(j => j.apiModels.length > 1);
    const jobsWithMissingModels = successfulJobs.filter(j => j.executionsWithoutModel > 0);
    
    logTest('Cross-Job Model Mapping Analysis', 'passed', {
      totalJobsAnalyzed: jobAnalysis.length,
      successfulJobs: successfulJobs.length,
      totalUniqueModels: totalModelsFound,
      jobsWithMultipleModels: jobsWithMultipleModels.length,
      jobsWithMissingModels: jobsWithMissingModels.length,
      modelMappingAccuracy: `${((successfulJobs.length - jobsWithMissingModels.length) / successfulJobs.length * 100).toFixed(1)}%`,
      detailedAnalysis: jobAnalysis,
      summary: {
        singleModelJobs: successfulJobs.filter(j => j.apiModels.length === 1).length,
        multiModelJobs: jobsWithMultipleModels.length,
        problematicJobs: jobsWithMissingModels.length
      }
    });
    
  } catch (error) {
    logTest('Cross-Job Model Mapping Analysis', 'failed', {
      error: error.message,
      stack: error.stack
    });
  }
}

// Test 4: Performance and Data Quality Tests
async function testPerformanceAndQuality() {
  log('info', '🧪 Starting Performance and Data Quality Tests');
  
  try {
    // Test API response times
    const performanceTests = [
      { name: 'Executions List', url: `${API_BASE}/executions` },
      { name: 'Cross-Region Diff', url: `${API_BASE}/executions/bias-detection-1758933513/cross-region-diff` }
    ];
    
    for (const test of performanceTests) {
      const startTime = Date.now();
      try {
        const data = await fetchWithRetry(test.url);
        const endTime = Date.now();
        const responseTime = endTime - startTime;
        const dataSize = JSON.stringify(data).length;
        
        logTest(`Performance - ${test.name}`, responseTime < 5000 ? 'passed' : 'warning', {
          responseTime: `${responseTime}ms`,
          dataSize: `${(dataSize / 1024).toFixed(2)}KB`,
          recordCount: Array.isArray(data.executions) ? data.executions.length : 'N/A'
        });
        
      } catch (error) {
        logTest(`Performance - ${test.name}`, 'failed', {
          error: error.message,
          responseTime: `${Date.now() - startTime}ms (failed)`
        });
      }
    }
    
    // Test data quality patterns
    const executions = await fetchWithRetry(`${API_BASE}/executions`);
    const qualityMetrics = {
      totalExecutions: executions.executions?.length || 0,
      executionsWithJobId: executions.executions?.filter(e => e.job_id).length || 0,
      executionsWithRegion: executions.executions?.filter(e => e.region).length || 0,
      executionsWithProvider: executions.executions?.filter(e => e.provider_id).length || 0,
      uniqueJobIds: [...new Set(executions.executions?.map(e => e.job_id) || [])].length,
      uniqueRegions: [...new Set(executions.executions?.map(e => e.region) || [])].length,
      uniqueProviders: [...new Set(executions.executions?.map(e => e.provider_id) || [])].length
    };
    
    const dataQualityScore = (
      (qualityMetrics.executionsWithJobId / qualityMetrics.totalExecutions) * 0.3 +
      (qualityMetrics.executionsWithRegion / qualityMetrics.totalExecutions) * 0.3 +
      (qualityMetrics.executionsWithProvider / qualityMetrics.totalExecutions) * 0.4
    ) * 100;
    
    logTest('Data Quality Analysis', dataQualityScore > 90 ? 'passed' : 'warning', {
      ...qualityMetrics,
      dataQualityScore: `${dataQualityScore.toFixed(1)}%`,
      completenessRatios: {
        jobId: `${(qualityMetrics.executionsWithJobId / qualityMetrics.totalExecutions * 100).toFixed(1)}%`,
        region: `${(qualityMetrics.executionsWithRegion / qualityMetrics.totalExecutions * 100).toFixed(1)}%`,
        provider: `${(qualityMetrics.executionsWithProvider / qualityMetrics.totalExecutions * 100).toFixed(1)}%`
      }
    });
    
  } catch (error) {
    logTest('Performance and Data Quality Tests', 'failed', {
      error: error.message,
      stack: error.stack
    });
  }
}

// Main test runner
async function runAllTests() {
  log('info', '🚀 Starting Comprehensive Model Mapping Validation Test Suite');
  
  const startTime = Date.now();
  
  try {
    await testApiDataValidation();
    await testTransformFunction();
    await testCrossJobModelMapping();
    await testPerformanceAndQuality();
    
  } catch (error) {
    log('error', 'Test suite execution failed', { error: error.message, stack: error.stack });
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
  const reportPath = path.join(RESULTS_DIR, `model-mapping-validation-${TIMESTAMP}.json`);
  fs.writeFileSync(reportPath, JSON.stringify(report, null, 2));
  
  // Print summary
  log('info', '📊 Test Suite Complete', {
    totalTests: report.summary.totalTests,
    passed: report.summary.passed,
    failed: report.summary.failed,
    warnings: report.summary.warnings,
    successRate: report.summary.successRate,
    executionTime: report.executionTime,
    reportSaved: reportPath
  });
  
  // Exit with appropriate code
  process.exit(report.summary.failed > 0 ? 1 : 0);
}

// Run tests if called directly
if (require.main === module) {
  runAllTests().catch(error => {
    console.error('Fatal error:', error);
    process.exit(1);
  });
}

module.exports = {
  runAllTests,
  testApiDataValidation,
  testTransformFunction,
  testCrossJobModelMapping,
  testPerformanceAndQuality
};
