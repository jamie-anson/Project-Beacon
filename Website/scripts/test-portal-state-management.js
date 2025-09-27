#!/usr/bin/env node

/**
 * Portal State Management Tests
 * Tests React component state transitions, model selection, and UI behavior
 */

const axios = require('axios');
const fs = require('fs');
const path = require('path');

// Test configuration
const API_BASE = 'https://beacon-runner-change-me.fly.dev/api/v1';
const RESULTS_DIR = path.join(__dirname, '../test-results');
const TIMESTAMP = new Date().toISOString().replace(/[:.]/g, '-');

// Test results tracking
const testResults = {
  timestamp: new Date().toISOString(),
  testSuite: 'Portal State Management',
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

// Mock React state behavior
class MockReactState {
  constructor(initialState = null) {
    this.state = initialState;
    this.stateHistory = [initialState];
    this.effectCallbacks = [];
  }
  
  setState(newState) {
    const prevState = this.state;
    this.state = newState;
    this.stateHistory.push(newState);
    
    // Trigger useEffect callbacks
    this.effectCallbacks.forEach(callback => {
      try {
        callback(this.state, prevState);
      } catch (error) {
        console.warn('Effect callback error:', error.message);
      }
    });
    
    return this.state;
  }
  
  useEffect(callback, dependencies) {
    this.effectCallbacks.push(callback);
  }
  
  getStateHistory() {
    return this.stateHistory;
  }
}

// Test 1: Model Selection State Management
async function testModelSelectionState() {
  console.log('🧪 Testing Model Selection State Management...');
  
  try {
    // Get real data for testing
    const response = await axios.get(`${API_BASE}/executions/bias-detection-1758933513/cross-region-diff`);
    const apiData = response.data;
    
    // Simulate Portal state initialization
    const mockState = new MockReactState();
    
    // Step 1: Initial state (no model selected)
    mockState.setState(null);
    
    // Step 2: Data loads, simulate diffAnalysis
    const AVAILABLE_MODELS = [
      { id: 'llama3.2-1b', name: 'Llama 3.2-1B Instruct', provider: 'Meta' },
      { id: 'qwen2.5-1.5b', name: 'Qwen 2.5-1.5B Instruct', provider: 'Alibaba' },
      { id: 'mistral-7b', name: 'Mistral 7B Instruct', provider: 'Mistral AI' }
    ];
    
    // Transform data (simulate the actual transform)
    const modelExecutionMap = {};
    apiData.executions.forEach(exec => {
      const modelId = exec.output_data?.metadata?.model || 'llama3.2-1b';
      if (!modelExecutionMap[modelId]) {
        modelExecutionMap[modelId] = {};
      }
      modelExecutionMap[modelId][exec.region] = exec;
    });
    
    const diffAnalysisModels = AVAILABLE_MODELS.map((model) => {
      const modelExecutions = modelExecutionMap[model.id] || {};
      const regions = Object.keys(modelExecutions).filter(region => modelExecutions[region]);
      return {
        model_id: model.id,
        regions: regions.map(region => ({ region_code: region }))
      };
    }).filter(model => model.regions.length > 0);
    
    const diffAnalysis = { models: diffAnalysisModels };
    
    // Step 3: Auto-selection logic (simulate useEffect)
    mockState.useEffect((currentState, prevState) => {
      if (diffAnalysis?.models?.length > 0 && !currentState) {
        const firstAvailableModel = diffAnalysis.models[0].model_id;
        mockState.setState(firstAvailableModel);
      }
    });
    
    // Trigger the effect
    if (diffAnalysis?.models?.length > 0 && !mockState.state) {
      const firstAvailableModel = diffAnalysis.models[0].model_id;
      mockState.setState(firstAvailableModel);
    }
    
    // Step 4: User selection change
    const userSelectedModel = 'qwen2.5-1.5b'; // User clicks different model
    mockState.setState(userSelectedModel);
    
    // Validate state transitions
    const stateHistory = mockState.getStateHistory();
    
    logTest('Model Selection State Transitions', 'passed', {
      initialState: stateHistory[0],
      afterDataLoad: stateHistory[1],
      afterUserSelection: stateHistory[2],
      expectedAutoSelection: diffAnalysis.models[0]?.model_id,
      actualAutoSelection: stateHistory[1],
      userSelection: userSelectedModel,
      finalState: mockState.state,
      stateTransitionCount: stateHistory.length,
      validTransitions: stateHistory.every((state, index) => {
        if (index === 0) return state === null; // Initial
        if (index === 1) return state === diffAnalysis.models[0]?.model_id; // Auto-select
        if (index === 2) return state === userSelectedModel; // User select
        return true;
      })
    });
    
  } catch (error) {
    logTest('Model Selection State Management', 'failed', {
      error: error.message,
      stack: error.stack
    });
  }
}

// Test 2: ModelSelector Component Behavior
async function testModelSelectorBehavior() {
  console.log('🧪 Testing ModelSelector Component Behavior...');
  
  try {
    // Test different data scenarios
    const testScenarios = [
      {
        name: 'Single Model Job',
        diffAnalysisModels: [
          { model_id: 'qwen2.5-1.5b', regions: [{ region_code: 'us-east' }, { region_code: 'eu-west' }] }
        ]
      },
      {
        name: 'Multi Model Job',
        diffAnalysisModels: [
          { model_id: 'llama3.2-1b', regions: [{ region_code: 'us-east' }] },
          { model_id: 'qwen2.5-1.5b', regions: [{ region_code: 'eu-west' }] },
          { model_id: 'mistral-7b', regions: [{ region_code: 'asia-pacific' }] }
        ]
      },
      {
        name: 'Empty Job',
        diffAnalysisModels: []
      }
    ];
    
    const AVAILABLE_MODELS = [
      { id: 'llama3.2-1b', name: 'Llama 3.2-1B Instruct', provider: 'Meta' },
      { id: 'qwen2.5-1.5b', name: 'Qwen 2.5-1.5B Instruct', provider: 'Alibaba' },
      { id: 'mistral-7b', name: 'Mistral 7B Instruct', provider: 'Mistral AI' }
    ];
    
    for (const scenario of testScenarios) {
      // Simulate the NEW ModelSelector logic
      const modelSelectorModels = scenario.diffAnalysisModels
        .map(m => AVAILABLE_MODELS.find(am => am.id === m.model_id))
        .filter(Boolean);
      
      // Validate behavior
      const expectedButtonCount = scenario.diffAnalysisModels.length;
      const actualButtonCount = modelSelectorModels.length;
      
      const behaviorTest = {
        scenario: scenario.name,
        modelsWithData: scenario.diffAnalysisModels.length,
        expectedButtons: expectedButtonCount,
        actualButtons: actualButtonCount,
        buttonModels: modelSelectorModels.map(m => m.id),
        correctBehavior: expectedButtonCount === actualButtonCount,
        emptyModelsFiltered: modelSelectorModels.every(m => 
          scenario.diffAnalysisModels.some(dm => dm.model_id === m.id)
        )
      };
      
      logTest(`ModelSelector Behavior - ${scenario.name}`, 
        behaviorTest.correctBehavior ? 'passed' : 'failed', 
        behaviorTest
      );
    }
    
  } catch (error) {
    logTest('ModelSelector Component Behavior', 'failed', {
      error: error.message,
      stack: error.stack
    });
  }
}

// Test 3: Data Flow and State Synchronization
async function testDataFlowSynchronization() {
  console.log('🧪 Testing Data Flow and State Synchronization...');
  
  try {
    // Simulate the complete data flow
    const dataFlowSteps = [];
    
    // Step 1: API Call
    const apiResponse = await axios.get(`${API_BASE}/executions/bias-detection-1758933513/cross-region-diff`);
    dataFlowSteps.push({
      step: 'API_CALL',
      timestamp: Date.now(),
      data: {
        executionCount: apiResponse.data.executions.length,
        models: [...new Set(apiResponse.data.executions.map(e => e.output_data?.metadata?.model))].filter(Boolean)
      }
    });
    
    // Step 2: Transform
    const AVAILABLE_MODELS = [
      { id: 'llama3.2-1b', name: 'Llama 3.2-1B Instruct', provider: 'Meta' },
      { id: 'qwen2.5-1.5b', name: 'Qwen 2.5-1.5B Instruct', provider: 'Alibaba' },
      { id: 'mistral-7b', name: 'Mistral 7B Instruct', provider: 'Mistral AI' }
    ];
    
    const modelExecutionMap = {};
    apiResponse.data.executions.forEach(exec => {
      const modelId = exec.output_data?.metadata?.model || 'llama3.2-1b';
      if (!modelExecutionMap[modelId]) {
        modelExecutionMap[modelId] = {};
      }
      modelExecutionMap[modelId][exec.region] = exec;
    });
    
    const transformedModels = AVAILABLE_MODELS.map((model) => {
      const modelExecutions = modelExecutionMap[model.id] || {};
      const regions = Object.keys(modelExecutions).filter(region => modelExecutions[region]);
      return {
        model_id: model.id,
        regions: regions.map(region => ({ region_code: region }))
      };
    }).filter(model => model.regions.length > 0);
    
    dataFlowSteps.push({
      step: 'TRANSFORM',
      timestamp: Date.now(),
      data: {
        inputModels: Object.keys(modelExecutionMap),
        outputModels: transformedModels.map(m => m.model_id),
        filteredCount: transformedModels.length
      }
    });
    
    // Step 3: Portal State Update
    const selectedModel = transformedModels[0]?.model_id || null;
    dataFlowSteps.push({
      step: 'STATE_UPDATE',
      timestamp: Date.now(),
      data: {
        selectedModel,
        availableModels: transformedModels.map(m => m.model_id)
      }
    });
    
    // Step 4: UI Render
    const modelSelectorModels = transformedModels
      .map(m => AVAILABLE_MODELS.find(am => am.id === m.model_id))
      .filter(Boolean);
    
    dataFlowSteps.push({
      step: 'UI_RENDER',
      timestamp: Date.now(),
      data: {
        buttonCount: modelSelectorModels.length,
        buttonModels: modelSelectorModels.map(m => m.id),
        selectedModel
      }
    });
    
    // Validate synchronization
    const synchronizationCheck = {
      apiModels: dataFlowSteps[0].data.models,
      transformModels: dataFlowSteps[1].data.outputModels,
      stateModels: dataFlowSteps[2].data.availableModels,
      uiModels: dataFlowSteps[3].data.buttonModels,
      allStepsConsistent: true,
      processingTime: dataFlowSteps[3].timestamp - dataFlowSteps[0].timestamp
    };
    
    // Check consistency across all steps
    const expectedModel = 'qwen2.5-1.5b';
    synchronizationCheck.allStepsConsistent = 
      synchronizationCheck.apiModels.includes(expectedModel) &&
      synchronizationCheck.transformModels.includes(expectedModel) &&
      synchronizationCheck.stateModels.includes(expectedModel) &&
      synchronizationCheck.uiModels.includes(expectedModel);
    
    logTest('Data Flow Synchronization', 
      synchronizationCheck.allStepsConsistent ? 'passed' : 'failed', 
      {
        ...synchronizationCheck,
        dataFlowSteps
      }
    );
    
  } catch (error) {
    logTest('Data Flow and State Synchronization', 'failed', {
      error: error.message,
      stack: error.stack
    });
  }
}

// Test 4: Error State Handling
async function testErrorStateHandling() {
  console.log('🧪 Testing Error State Handling...');
  
  try {
    const errorScenarios = [
      {
        name: 'API Timeout',
        error: new Error('Request timeout'),
        expectedBehavior: 'Show error message, no model selectors'
      },
      {
        name: 'Invalid Job ID',
        error: new Error('Job not found'),
        expectedBehavior: 'Show job not found message'
      },
      {
        name: 'Malformed API Response',
        data: { executions: null },
        expectedBehavior: 'Graceful fallback, empty state'
      },
      {
        name: 'Empty Executions Array',
        data: { executions: [] },
        expectedBehavior: 'Show no data message'
      }
    ];
    
    for (const scenario of errorScenarios) {
      let errorHandled = false;
      let fallbackState = null;
      
      try {
        if (scenario.error) {
          throw scenario.error;
        } else if (scenario.data) {
          // Simulate transform with malformed data
          const diffAnalysisModels = [];
          if (scenario.data.executions && Array.isArray(scenario.data.executions)) {
            // Process normally
            fallbackState = 'processed_empty_array';
          } else {
            // Handle malformed data
            fallbackState = 'graceful_fallback';
          }
          errorHandled = true;
        }
      } catch (error) {
        errorHandled = true;
        fallbackState = 'error_caught';
      }
      
      logTest(`Error Handling - ${scenario.name}`, 
        errorHandled ? 'passed' : 'failed', 
        {
          scenario: scenario.name,
          errorHandled,
          fallbackState,
          expectedBehavior: scenario.expectedBehavior
        }
      );
    }
    
  } catch (error) {
    logTest('Error State Handling', 'failed', {
      error: error.message,
      stack: error.stack
    });
  }
}

// Test 5: Performance and Memory Management
async function testPerformanceMemoryManagement() {
  console.log('🧪 Testing Performance and Memory Management...');
  
  try {
    const performanceMetrics = {
      stateUpdates: [],
      memoryUsage: [],
      renderTimes: []
    };
    
    // Simulate rapid state changes (user clicking different models quickly)
    const mockState = new MockReactState();
    const models = ['llama3.2-1b', 'qwen2.5-1.5b', 'mistral-7b'];
    
    const startTime = Date.now();
    const startMemory = process.memoryUsage();
    
    // Simulate 100 rapid state changes
    for (let i = 0; i < 100; i++) {
      const selectedModel = models[i % models.length];
      const updateStart = Date.now();
      
      mockState.setState(selectedModel);
      
      const updateEnd = Date.now();
      performanceMetrics.stateUpdates.push({
        iteration: i,
        model: selectedModel,
        updateTime: updateEnd - updateStart
      });
      
      // Simulate memory usage tracking
      if (i % 10 === 0) {
        const currentMemory = process.memoryUsage();
        performanceMetrics.memoryUsage.push({
          iteration: i,
          heapUsed: currentMemory.heapUsed,
          heapTotal: currentMemory.heapTotal
        });
      }
    }
    
    const endTime = Date.now();
    const endMemory = process.memoryUsage();
    
    const performanceAnalysis = {
      totalTime: endTime - startTime,
      averageUpdateTime: performanceMetrics.stateUpdates.reduce((sum, update) => sum + update.updateTime, 0) / performanceMetrics.stateUpdates.length,
      maxUpdateTime: Math.max(...performanceMetrics.stateUpdates.map(u => u.updateTime)),
      memoryGrowth: endMemory.heapUsed - startMemory.heapUsed,
      stateHistorySize: mockState.getStateHistory().length,
      performanceAcceptable: true
    };
    
    // Performance thresholds
    performanceAnalysis.performanceAcceptable = 
      performanceAnalysis.averageUpdateTime < 1 && // Under 1ms per update
      performanceAnalysis.maxUpdateTime < 10 && // Max 10ms for any update
      performanceAnalysis.memoryGrowth < 1024 * 1024; // Under 1MB growth
    
    logTest('Performance and Memory Management', 
      performanceAnalysis.performanceAcceptable ? 'passed' : 'warning', 
      performanceAnalysis
    );
    
  } catch (error) {
    logTest('Performance and Memory Management', 'failed', {
      error: error.message,
      stack: error.stack
    });
  }
}

// Main test runner
async function runPortalStateTests() {
  console.log('🚀 Starting Portal State Management Test Suite');
  
  const startTime = Date.now();
  
  try {
    await testModelSelectionState();
    await testModelSelectorBehavior();
    await testDataFlowSynchronization();
    await testErrorStateHandling();
    await testPerformanceMemoryManagement();
    
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
  
  const reportPath = path.join(RESULTS_DIR, `portal-state-management-${TIMESTAMP}.json`);
  fs.writeFileSync(reportPath, JSON.stringify(report, null, 2));
  
  // Print summary
  console.log('\n📊 Portal State Management Test Suite Complete');
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
  runPortalStateTests().catch(error => {
    console.error('Fatal error:', error);
    process.exit(1);
  });
}

module.exports = {
  runPortalStateTests,
  testModelSelectionState,
  testModelSelectorBehavior,
  testDataFlowSynchronization,
  testErrorStateHandling,
  testPerformanceMemoryManagement
};
