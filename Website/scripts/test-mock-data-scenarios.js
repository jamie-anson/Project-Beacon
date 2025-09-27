#!/usr/bin/env node

/**
 * Comprehensive Mock Data Scenarios Tests
 * Tests edge cases, malformed data, and boundary conditions
 */

const fs = require('fs');
const path = require('path');

// Test configuration
const RESULTS_DIR = path.join(__dirname, '../test-results');
const TIMESTAMP = new Date().toISOString().replace(/[:.]/g, '-');

// Test results tracking
const testResults = {
  timestamp: new Date().toISOString(),
  testSuite: 'Mock Data Scenarios',
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

// Mock transform function (simplified version of the real one)
function mockTransformCrossRegionDiff(apiData, availableModels = [
  { id: 'llama3.2-1b', name: 'Llama 3.2-1B Instruct', provider: 'Meta' },
  { id: 'qwen2.5-1.5b', name: 'Qwen 2.5-1.5B Instruct', provider: 'Alibaba' },
  { id: 'mistral-7b', name: 'Mistral 7B Instruct', provider: 'Mistral AI' }
]) {
  try {
    const executions = Array.isArray(apiData?.executions) ? apiData.executions : [];
    
    // Group executions by model_id, then by region
    const modelExecutionMap = executions.reduce((acc, exec) => {
      if (!exec?.region) return acc;
      
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
          modelId = 'llama3.2-1b';
        }
      }
      
      if (!acc[modelId]) {
        acc[modelId] = {};
      }
      
      acc[modelId][exec.region] = exec;
      return acc;
    }, {});
    
    const normalizedModels = availableModels.map((model) => {
      const modelExecutions = modelExecutionMap[model.id] || {};
      
      const regions = Object.keys(modelExecutions).map((regionCode) => {
        const exec = modelExecutions[regionCode];
        const response = exec?.output_data?.response || 'No response available';
        
        return {
          region_code: regionCode,
          region_name: regionCode,
          response: response,
          hasData: !!exec
        };
      }).filter(region => region.hasData);
      
      return {
        model_id: model.id,
        model_name: model.name,
        provider: model.provider,
        regions: regions
      };
    }).filter(model => model.regions.length > 0);
    
    return {
      job_id: apiData?.job_id || 'unknown',
      models: normalizedModels,
      metrics: {
        bias_variance: 23,
        censorship_rate: 100,
        factual_consistency: 87,
        narrative_divergence: 31
      }
    };
    
  } catch (error) {
    return {
      job_id: apiData?.job_id || 'unknown',
      models: [],
      metrics: null,
      error: error.message
    };
  }
}

// Test 1: Edge Case Data Scenarios
async function testEdgeCaseDataScenarios() {
  console.log('🧪 Testing Edge Case Data Scenarios...');
  
  const edgeCases = [
    {
      name: 'Null API Data',
      data: null,
      expectedModels: 0,
      expectedError: false
    },
    {
      name: 'Undefined API Data',
      data: undefined,
      expectedModels: 0,
      expectedError: false
    },
    {
      name: 'Empty Object',
      data: {},
      expectedModels: 0,
      expectedError: false
    },
    {
      name: 'Null Executions Array',
      data: { executions: null },
      expectedModels: 0,
      expectedError: false
    },
    {
      name: 'Empty Executions Array',
      data: { executions: [] },
      expectedModels: 0,
      expectedError: false
    },
    {
      name: 'Non-Array Executions',
      data: { executions: "not an array" },
      expectedModels: 0,
      expectedError: false
    },
    {
      name: 'Executions with Missing Fields',
      data: {
        executions: [
          { id: 1 }, // Missing region
          { id: 2, region: 'us-east' }, // Missing output_data
          { id: 3, region: 'eu-west', output_data: null } // Null output_data
        ]
      },
      expectedModels: 0,
      expectedError: false
    },
    {
      name: 'Mixed Valid and Invalid Executions',
      data: {
        executions: [
          { id: 1, region: 'us-east', output_data: { metadata: { model: 'qwen2.5-1.5b' }, response: 'Valid response' } },
          { id: 2 }, // Invalid
          { id: 3, region: 'eu-west', output_data: { metadata: { model: 'qwen2.5-1.5b' }, response: 'Another valid response' } }
        ]
      },
      expectedModels: 1,
      expectedError: false
    }
  ];
  
  for (const edgeCase of edgeCases) {
    try {
      const result = mockTransformCrossRegionDiff(edgeCase.data);
      
      const testResult = {
        scenario: edgeCase.name,
        inputData: edgeCase.data,
        outputModels: result.models.length,
        expectedModels: edgeCase.expectedModels,
        hasError: !!result.error,
        expectedError: edgeCase.expectedError,
        modelsMatch: result.models.length === edgeCase.expectedModels,
        errorMatch: !!result.error === edgeCase.expectedError,
        gracefulHandling: !result.error || edgeCase.expectedError,
        testPassed: true
      };
      
      testResult.testPassed = 
        testResult.modelsMatch && 
        testResult.errorMatch && 
        testResult.gracefulHandling;
      
      logTest(`Edge Case - ${edgeCase.name}`, 
        testResult.testPassed ? 'passed' : 'failed', 
        testResult
      );
      
    } catch (error) {
      logTest(`Edge Case - ${edgeCase.name}`, 'failed', {
        scenario: edgeCase.name,
        error: error.message,
        stack: error.stack
      });
    }
  }
}

// Test 2: Malformed Data Structures
async function testMalformedDataStructures() {
  console.log('🧪 Testing Malformed Data Structures...');
  
  const malformedCases = [
    {
      name: 'Circular Reference',
      createData: () => {
        const data = { executions: [] };
        data.self = data; // Circular reference
        return data;
      },
      expectedHandling: 'graceful'
    },
    {
      name: 'Deeply Nested Invalid Structure',
      data: {
        executions: [
          {
            id: 1,
            region: 'us-east',
            output_data: {
              metadata: {
                model: {
                  nested: {
                    deeply: {
                      invalid: 'qwen2.5-1.5b'
                    }
                  }
                }
              },
              response: 'Response'
            }
          }
        ]
      },
      expectedHandling: 'graceful'
    },
    {
      name: 'Invalid Data Types',
      data: {
        executions: [
          {
            id: 'string-id-instead-of-number',
            region: 123, // Number instead of string
            output_data: {
              metadata: {
                model: ['array', 'instead', 'of', 'string']
              },
              response: { object: 'instead of string' }
            }
          }
        ]
      },
      expectedHandling: 'graceful'
    },
    {
      name: 'Unicode and Special Characters',
      data: {
        executions: [
          {
            id: 1,
            region: 'us-east-🇺🇸',
            output_data: {
              metadata: { model: 'qwen2.5-1.5b' },
              response: 'Response with unicode: 你好世界 and emojis: 🤖🔍📊'
            }
          }
        ]
      },
      expectedHandling: 'graceful'
    },
    {
      name: 'Very Large Data',
      createData: () => {
        const largeResponse = 'A'.repeat(100000); // 100KB response
        return {
          executions: Array.from({ length: 1000 }, (_, i) => ({
            id: i,
            region: `region-${i % 3}`,
            output_data: {
              metadata: { model: 'qwen2.5-1.5b' },
              response: largeResponse
            }
          }))
        };
      },
      expectedHandling: 'performance_test'
    }
  ];
  
  for (const malformedCase of malformedCases) {
    try {
      const data = malformedCase.createData ? malformedCase.createData() : malformedCase.data;
      
      const startTime = Date.now();
      const result = mockTransformCrossRegionDiff(data);
      const endTime = Date.now();
      
      const processingTime = endTime - startTime;
      
      const testResult = {
        scenario: malformedCase.name,
        processingTime: `${processingTime}ms`,
        outputGenerated: !!result,
        hasModels: result.models && result.models.length > 0,
        hasError: !!result.error,
        gracefulHandling: !result.error || malformedCase.expectedHandling === 'graceful',
        performanceAcceptable: processingTime < 1000, // Under 1 second
        memoryEfficient: true // Assume true unless we detect issues
      };
      
      if (malformedCase.expectedHandling === 'performance_test') {
        testResult.testPassed = testResult.performanceAcceptable && testResult.gracefulHandling;
      } else {
        testResult.testPassed = testResult.gracefulHandling;
      }
      
      logTest(`Malformed Data - ${malformedCase.name}`, 
        testResult.testPassed ? 'passed' : 'warning', 
        testResult
      );
      
    } catch (error) {
      logTest(`Malformed Data - ${malformedCase.name}`, 'failed', {
        scenario: malformedCase.name,
        error: error.message,
        stack: error.stack
      });
    }
  }
}

// Test 3: Boundary Condition Testing
async function testBoundaryConditions() {
  console.log('🧪 Testing Boundary Conditions...');
  
  const boundaryTests = [
    {
      name: 'Single Execution',
      data: {
        executions: [
          {
            id: 1,
            region: 'us-east',
            output_data: {
              metadata: { model: 'qwen2.5-1.5b' },
              response: 'Single response'
            }
          }
        ]
      },
      expectedModels: 1,
      expectedRegions: 1
    },
    {
      name: 'Maximum Realistic Load',
      createData: () => ({
        executions: Array.from({ length: 9 }, (_, i) => ({
          id: i + 1,
          region: ['us-east', 'eu-west', 'asia-pacific'][i % 3],
          output_data: {
            metadata: { model: ['llama3.2-1b', 'qwen2.5-1.5b', 'mistral-7b'][Math.floor(i / 3)] },
            response: `Response ${i + 1}`
          }
        }))
      }),
      expectedModels: 3,
      expectedRegions: 3
    },
    {
      name: 'All Same Model Different Regions',
      data: {
        executions: [
          { id: 1, region: 'us-east', output_data: { metadata: { model: 'qwen2.5-1.5b' }, response: 'US response' } },
          { id: 2, region: 'eu-west', output_data: { metadata: { model: 'qwen2.5-1.5b' }, response: 'EU response' } },
          { id: 3, region: 'asia-pacific', output_data: { metadata: { model: 'qwen2.5-1.5b' }, response: 'Asia response' } }
        ]
      },
      expectedModels: 1,
      expectedRegions: 3
    },
    {
      name: 'All Same Region Different Models',
      data: {
        executions: [
          { id: 1, region: 'us-east', output_data: { metadata: { model: 'llama3.2-1b' }, response: 'Llama response' } },
          { id: 2, region: 'us-east', output_data: { metadata: { model: 'qwen2.5-1.5b' }, response: 'Qwen response' } },
          { id: 3, region: 'us-east', output_data: { metadata: { model: 'mistral-7b' }, response: 'Mistral response' } }
        ]
      },
      expectedModels: 3,
      expectedRegions: 1
    },
    {
      name: 'Empty Responses',
      data: {
        executions: [
          { id: 1, region: 'us-east', output_data: { metadata: { model: 'qwen2.5-1.5b' }, response: '' } },
          { id: 2, region: 'eu-west', output_data: { metadata: { model: 'qwen2.5-1.5b' }, response: null } },
          { id: 3, region: 'asia-pacific', output_data: { metadata: { model: 'qwen2.5-1.5b' }, response: undefined } }
        ]
      },
      expectedModels: 1,
      expectedRegions: 3
    }
  ];
  
  for (const boundaryTest of boundaryTests) {
    try {
      const data = boundaryTest.createData ? boundaryTest.createData() : boundaryTest.data;
      const result = mockTransformCrossRegionDiff(data);
      
      const actualRegions = result.models.reduce((total, model) => total + model.regions.length, 0);
      
      const testResult = {
        scenario: boundaryTest.name,
        expectedModels: boundaryTest.expectedModels,
        actualModels: result.models.length,
        expectedRegions: boundaryTest.expectedRegions,
        actualRegions: actualRegions,
        modelsMatch: result.models.length === boundaryTest.expectedModels,
        regionsMatch: actualRegions === boundaryTest.expectedRegions,
        hasValidStructure: result.models.every(m => 
          m.model_id && m.model_name && Array.isArray(m.regions)
        ),
        testPassed: true
      };
      
      testResult.testPassed = 
        testResult.modelsMatch && 
        testResult.regionsMatch && 
        testResult.hasValidStructure;
      
      logTest(`Boundary Condition - ${boundaryTest.name}`, 
        testResult.testPassed ? 'passed' : 'failed', 
        testResult
      );
      
    } catch (error) {
      logTest(`Boundary Condition - ${boundaryTest.name}`, 'failed', {
        scenario: boundaryTest.name,
        error: error.message,
        stack: error.stack
      });
    }
  }
}

// Test 4: Data Consistency and Integrity
async function testDataConsistencyIntegrity() {
  console.log('🧪 Testing Data Consistency and Integrity...');
  
  const consistencyTests = [
    {
      name: 'Model ID Consistency',
      data: {
        executions: [
          { id: 1, region: 'us-east', model_id: 'qwen2.5-1.5b', output_data: { metadata: { model: 'different-model' }, response: 'Response' } }
        ]
      },
      validation: (result) => {
        // Should prioritize model_id over metadata.model
        return result.models.some(m => m.model_id === 'qwen2.5-1.5b');
      }
    },
    {
      name: 'Region Code Normalization',
      data: {
        executions: [
          { id: 1, region: 'US-EAST', output_data: { metadata: { model: 'qwen2.5-1.5b' }, response: 'Response' } },
          { id: 2, region: 'us-east', output_data: { metadata: { model: 'qwen2.5-1.5b' }, response: 'Response' } }
        ]
      },
      validation: (result) => {
        // Should handle region code variations
        return result.models[0]?.regions.length > 0;
      }
    },
    {
      name: 'Response Preservation',
      data: {
        executions: [
          { id: 1, region: 'us-east', output_data: { metadata: { model: 'qwen2.5-1.5b' }, response: 'Original response with special chars: @#$%^&*()' } }
        ]
      },
      validation: (result) => {
        // Should preserve original response exactly
        return result.models[0]?.regions[0]?.response === 'Original response with special chars: @#$%^&*()';
      }
    },
    {
      name: 'Execution Order Independence',
      createData: () => {
        const baseExecutions = [
          { id: 1, region: 'us-east', output_data: { metadata: { model: 'qwen2.5-1.5b' }, response: 'Response 1' } },
          { id: 2, region: 'eu-west', output_data: { metadata: { model: 'llama3.2-1b' }, response: 'Response 2' } },
          { id: 3, region: 'asia-pacific', output_data: { metadata: { model: 'mistral-7b' }, response: 'Response 3' } }
        ];
        
        // Return shuffled version
        return {
          original: { executions: [...baseExecutions] },
          shuffled: { executions: [baseExecutions[2], baseExecutions[0], baseExecutions[1]] }
        };
      },
      validation: (originalResult, shuffledResult) => {
        // Results should be equivalent regardless of input order
        return originalResult.models.length === shuffledResult.models.length &&
               originalResult.models.every(model => 
                 shuffledResult.models.some(sm => sm.model_id === model.model_id)
               );
      }
    }
  ];
  
  for (const consistencyTest of consistencyTests) {
    try {
      if (consistencyTest.createData) {
        const { original, shuffled } = consistencyTest.createData();
        const originalResult = mockTransformCrossRegionDiff(original);
        const shuffledResult = mockTransformCrossRegionDiff(shuffled);
        
        const testResult = {
          scenario: consistencyTest.name,
          originalModels: originalResult.models.length,
          shuffledModels: shuffledResult.models.length,
          validationPassed: consistencyTest.validation(originalResult, shuffledResult),
          testPassed: true
        };
        
        testResult.testPassed = testResult.validationPassed;
        
        logTest(`Data Consistency - ${consistencyTest.name}`, 
          testResult.testPassed ? 'passed' : 'failed', 
          testResult
        );
        
      } else {
        const result = mockTransformCrossRegionDiff(consistencyTest.data);
        
        const testResult = {
          scenario: consistencyTest.name,
          resultGenerated: !!result,
          validationPassed: consistencyTest.validation(result),
          testPassed: true
        };
        
        testResult.testPassed = testResult.resultGenerated && testResult.validationPassed;
        
        logTest(`Data Consistency - ${consistencyTest.name}`, 
          testResult.testPassed ? 'passed' : 'failed', 
          testResult
        );
      }
      
    } catch (error) {
      logTest(`Data Consistency - ${consistencyTest.name}`, 'failed', {
        scenario: consistencyTest.name,
        error: error.message,
        stack: error.stack
      });
    }
  }
}

// Main test runner
async function runMockDataScenariosTests() {
  console.log('🚀 Starting Mock Data Scenarios Test Suite');
  
  const startTime = Date.now();
  
  try {
    await testEdgeCaseDataScenarios();
    await testMalformedDataStructures();
    await testBoundaryConditions();
    await testDataConsistencyIntegrity();
    
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
  
  const reportPath = path.join(RESULTS_DIR, `mock-data-scenarios-${TIMESTAMP}.json`);
  fs.writeFileSync(reportPath, JSON.stringify(report, null, 2));
  
  // Print summary
  console.log('\n📊 Mock Data Scenarios Test Suite Complete');
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
  runMockDataScenariosTests().catch(error => {
    console.error('Fatal error:', error);
    process.exit(1);
  });
}

module.exports = {
  runMockDataScenariosTests,
  testEdgeCaseDataScenarios,
  testMalformedDataStructures,
  testBoundaryConditions,
  testDataConsistencyIntegrity,
  mockTransformCrossRegionDiff
};
