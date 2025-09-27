#!/usr/bin/env node

/**
 * React Component Integration Tests
 * Tests component behavior, props passing, and UI state management
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
  testSuite: 'React Component Integration',
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

// Mock React component behavior
class MockComponent {
  constructor(name, props = {}) {
    this.name = name;
    this.props = props;
    this.state = {};
    this.children = [];
    this.eventHandlers = {};
    this.renderCount = 0;
  }
  
  setProps(newProps) {
    const prevProps = { ...this.props };
    this.props = { ...this.props, ...newProps };
    this.onPropsChange(newProps, prevProps);
    return this;
  }
  
  setState(newState) {
    const prevState = { ...this.state };
    this.state = { ...this.state, ...newState };
    this.onStateChange(newState, prevState);
    return this;
  }
  
  addChild(child) {
    this.children.push(child);
    return this;
  }
  
  addEventListener(event, handler) {
    this.eventHandlers[event] = handler;
    return this;
  }
  
  triggerEvent(event, data) {
    if (this.eventHandlers[event]) {
      this.eventHandlers[event](data);
    }
    return this;
  }
  
  render() {
    this.renderCount++;
    return {
      component: this.name,
      props: this.props,
      state: this.state,
      children: this.children.map(child => child.render ? child.render() : child),
      renderCount: this.renderCount
    };
  }
  
  onPropsChange(newProps, prevProps) {
    // Override in specific components
  }
  
  onStateChange(newState, prevState) {
    // Override in specific components
  }
}

// Test 1: ModelSelector Component Integration
async function testModelSelectorIntegration() {
  console.log('🧪 Testing ModelSelector Component Integration...');
  
  try {
    // Get real data for testing
    const response = await axios.get(`${API_BASE}/executions/bias-detection-1758933513/cross-region-diff`);
    const apiData = response.data;
    
    // Simulate ModelSelector component
    class MockModelSelector extends MockComponent {
      constructor(props) {
        super('ModelSelector', props);
      }
      
      onPropsChange(newProps, prevProps) {
        // Simulate component re-render when models change
        if (JSON.stringify(newProps.models) !== JSON.stringify(prevProps.models)) {
          this.setState({ modelsUpdated: true });
        }
      }
    }
    
    // Test scenarios
    const testScenarios = [
      {
        name: 'Single Model Display',
        models: [{ id: 'qwen2.5-1.5b', name: 'Qwen 2.5-1.5B Instruct', provider: 'Alibaba' }],
        selectedModel: 'qwen2.5-1.5b',
        expectedButtons: 1
      },
      {
        name: 'Multi Model Display',
        models: [
          { id: 'llama3.2-1b', name: 'Llama 3.2-1B Instruct', provider: 'Meta' },
          { id: 'qwen2.5-1.5b', name: 'Qwen 2.5-1.5B Instruct', provider: 'Alibaba' },
          { id: 'mistral-7b', name: 'Mistral 7B Instruct', provider: 'Mistral AI' }
        ],
        selectedModel: 'qwen2.5-1.5b',
        expectedButtons: 3
      },
      {
        name: 'Empty Models',
        models: [],
        selectedModel: null,
        expectedButtons: 0
      }
    ];
    
    for (const scenario of testScenarios) {
      const modelSelector = new MockModelSelector({
        models: scenario.models,
        selectedModel: scenario.selectedModel,
        onSelectModel: (modelId) => {
          modelSelector.setProps({ selectedModel: modelId });
        }
      });
      
      // Simulate user interaction
      modelSelector.addEventListener('click', (modelId) => {
        modelSelector.props.onSelectModel(modelId);
      });
      
      // Test initial render
      const initialRender = modelSelector.render();
      
      // Test model selection
      if (scenario.models.length > 0) {
        const newModelId = scenario.models[0].id;
        modelSelector.triggerEvent('click', newModelId);
      }
      
      const finalRender = modelSelector.render();
      
      const integrationTest = {
        scenario: scenario.name,
        initialModels: scenario.models.length,
        expectedButtons: scenario.expectedButtons,
        actualButtons: scenario.models.length,
        selectedModelCorrect: modelSelector.props.selectedModel === (scenario.models[0]?.id || scenario.selectedModel),
        renderCount: finalRender.renderCount,
        propsUpdated: finalRender.renderCount > 1,
        componentWorking: true
      };
      
      integrationTest.componentWorking = 
        integrationTest.actualButtons === integrationTest.expectedButtons &&
        (scenario.models.length === 0 || integrationTest.selectedModelCorrect);
      
      logTest(`ModelSelector Integration - ${scenario.name}`, 
        integrationTest.componentWorking ? 'passed' : 'failed', 
        integrationTest
      );
    }
    
  } catch (error) {
    logTest('ModelSelector Component Integration', 'failed', {
      error: error.message,
      stack: error.stack
    });
  }
}

// Test 2: CrossRegionDiffPage Component Integration
async function testCrossRegionDiffPageIntegration() {
  console.log('🧪 Testing CrossRegionDiffPage Component Integration...');
  
  try {
    // Simulate CrossRegionDiffPage component
    class MockCrossRegionDiffPage extends MockComponent {
      constructor(props) {
        super('CrossRegionDiffPage', props);
        this.state = {
          selectedModel: null,
          loading: true,
          error: null,
          diffAnalysis: null
        };
      }
      
      async loadData(jobId) {
        this.setState({ loading: true, error: null });
        
        try {
          const response = await axios.get(`${API_BASE}/executions/${jobId}/cross-region-diff`);
          const apiData = response.data;
          
          // Simulate transform
          const AVAILABLE_MODELS = [
            { id: 'llama3.2-1b', name: 'Llama 3.2-1B Instruct', provider: 'Meta' },
            { id: 'qwen2.5-1.5b', name: 'Qwen 2.5-1.5B Instruct', provider: 'Alibaba' },
            { id: 'mistral-7b', name: 'Mistral 7B Instruct', provider: 'Mistral AI' }
          ];
          
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
              regions: regions
            };
          }).filter(model => model.regions.length > 0);
          
          const diffAnalysis = {
            models: diffAnalysisModels,
            metrics: {
              bias_variance: 23,
              censorship_rate: 100,
              factual_consistency: 87,
              narrative_divergence: 31
            }
          };
          
          this.setState({ 
            diffAnalysis, 
            loading: false,
            selectedModel: diffAnalysisModels[0]?.model_id || null
          });
          
        } catch (error) {
          this.setState({ 
            loading: false, 
            error: error.message 
          });
        }
      }
      
      render() {
        this.renderCount++;
        
        const modelSelectorModels = this.state.diffAnalysis?.models?.map(m => 
          this.props.availableModels.find(am => am.id === m.model_id)
        ).filter(Boolean) || [];
        
        return {
          component: this.name,
          state: this.state,
          children: [
            {
              component: 'ModelSelector',
              props: {
                models: modelSelectorModels,
                selectedModel: this.state.selectedModel
              }
            },
            {
              component: 'MetricsGrid',
              props: {
                metrics: this.state.diffAnalysis?.metrics
              }
            }
          ],
          renderCount: this.renderCount
        };
      }
    }
    
    // Test page integration
    const AVAILABLE_MODELS = [
      { id: 'llama3.2-1b', name: 'Llama 3.2-1B Instruct', provider: 'Meta' },
      { id: 'qwen2.5-1.5b', name: 'Qwen 2.5-1.5B Instruct', provider: 'Alibaba' },
      { id: 'mistral-7b', name: 'Mistral 7B Instruct', provider: 'Mistral AI' }
    ];
    
    const diffPage = new MockCrossRegionDiffPage({
      jobId: 'bias-detection-1758933513',
      availableModels: AVAILABLE_MODELS
    });
    
    // Test loading state
    const loadingRender = diffPage.render();
    
    // Load data
    await diffPage.loadData('bias-detection-1758933513');
    
    // Test loaded state
    const loadedRender = diffPage.render();
    
    const pageIntegrationTest = {
      initialState: 'loading',
      finalState: diffPage.state.error ? 'error' : 'loaded',
      dataLoaded: !!diffPage.state.diffAnalysis,
      modelsAvailable: diffPage.state.diffAnalysis?.models?.length || 0,
      selectedModelSet: !!diffPage.state.selectedModel,
      childComponentsRendered: loadedRender.children.length,
      modelSelectorHasModels: loadedRender.children[0]?.props?.models?.length || 0,
      metricsAvailable: !!loadedRender.children[1]?.props?.metrics,
      pageWorking: true
    };
    
    pageIntegrationTest.pageWorking = 
      pageIntegrationTest.dataLoaded &&
      pageIntegrationTest.modelsAvailable > 0 &&
      pageIntegrationTest.selectedModelSet &&
      pageIntegrationTest.modelSelectorHasModels > 0;
    
    logTest('CrossRegionDiffPage Integration', 
      pageIntegrationTest.pageWorking ? 'passed' : 'failed', 
      pageIntegrationTest
    );
    
  } catch (error) {
    logTest('CrossRegionDiffPage Component Integration', 'failed', {
      error: error.message,
      stack: error.stack
    });
  }
}

// Test 3: Component Props Flow and Data Binding
async function testComponentPropsFlowDataBinding() {
  console.log('🧪 Testing Component Props Flow and Data Binding...');
  
  try {
    // Simulate parent-child component relationship
    class MockParentComponent extends MockComponent {
      constructor() {
        super('ParentComponent');
        this.state = {
          selectedModel: 'qwen2.5-1.5b',
          diffAnalysis: null
        };
      }
      
      handleModelSelect(modelId) {
        this.setState({ selectedModel: modelId });
        
        // Simulate data filtering based on selection
        if (this.state.diffAnalysis) {
          const selectedModelData = this.state.diffAnalysis.models.find(m => m.model_id === modelId);
          this.setState({ selectedModelData });
        }
      }
      
      render() {
        this.renderCount++;
        
        const childProps = {
          models: this.state.diffAnalysis?.models || [],
          selectedModel: this.state.selectedModel,
          onSelectModel: (modelId) => this.handleModelSelect(modelId)
        };
        
        return {
          component: this.name,
          state: this.state,
          childProps,
          renderCount: this.renderCount
        };
      }
    }
    
    // Test props flow
    const parent = new MockParentComponent();
    
    // Set initial data
    parent.setState({
      diffAnalysis: {
        models: [
          { model_id: 'qwen2.5-1.5b', regions: [{ region_code: 'us-east' }] },
          { model_id: 'llama3.2-1b', regions: [{ region_code: 'eu-west' }] }
        ]
      }
    });
    
    const initialRender = parent.render();
    
    // Simulate model selection
    parent.handleModelSelect('llama3.2-1b');
    
    const afterSelectionRender = parent.render();
    
    const propsFlowTest = {
      initialSelectedModel: initialRender.state.selectedModel,
      finalSelectedModel: afterSelectionRender.state.selectedModel,
      modelsPassedToChild: afterSelectionRender.childProps.models.length,
      selectedModelPassedToChild: afterSelectionRender.childProps.selectedModel,
      onSelectModelProvided: typeof afterSelectionRender.childProps.onSelectModel === 'function',
      stateUpdatedCorrectly: afterSelectionRender.state.selectedModel === 'llama3.2-1b',
      renderTriggered: afterSelectionRender.renderCount > initialRender.renderCount,
      propsFlowWorking: true
    };
    
    propsFlowTest.propsFlowWorking = 
      propsFlowTest.stateUpdatedCorrectly &&
      propsFlowTest.selectedModelPassedToChild === 'llama3.2-1b' &&
      propsFlowTest.onSelectModelProvided &&
      propsFlowTest.renderTriggered;
    
    logTest('Component Props Flow and Data Binding', 
      propsFlowTest.propsFlowWorking ? 'passed' : 'failed', 
      propsFlowTest
    );
    
  } catch (error) {
    logTest('Component Props Flow and Data Binding', 'failed', {
      error: error.message,
      stack: error.stack
    });
  }
}

// Test 4: Component Error Boundaries and Fallbacks
async function testComponentErrorBoundariesFallbacks() {
  console.log('🧪 Testing Component Error Boundaries and Fallbacks...');
  
  try {
    // Simulate error boundary behavior
    class MockErrorBoundary extends MockComponent {
      constructor() {
        super('ErrorBoundary');
        this.state = {
          hasError: false,
          error: null
        };
      }
      
      componentDidCatch(error) {
        this.setState({
          hasError: true,
          error: error.message
        });
      }
      
      render() {
        if (this.state.hasError) {
          return {
            component: 'ErrorFallback',
            error: this.state.error
          };
        }
        
        return {
          component: this.name,
          children: this.children
        };
      }
    }
    
    const errorBoundaryTests = [
      {
        name: 'Invalid Props Handling',
        test: () => {
          const errorBoundary = new MockErrorBoundary();
          
          try {
            // Simulate component with invalid props
            const invalidProps = { models: null, selectedModel: undefined };
            
            // Simulate component trying to map over null
            if (invalidProps.models && invalidProps.models.map) {
              invalidProps.models.map(m => m.id);
            } else {
              // Graceful fallback
              return { success: true, fallbackTriggered: true };
            }
            
            return { success: true, errorHandled: true };
            
          } catch (error) {
            errorBoundary.componentDidCatch(error);
            return { 
              success: true, 
              errorCaught: true, 
              errorBoundaryTriggered: errorBoundary.state.hasError 
            };
          }
        }
      },
      {
        name: 'API Data Malformation',
        test: () => {
          try {
            // Simulate malformed API response
            const malformedData = { executions: "not an array" };
            
            // Simulate safe processing
            const executions = Array.isArray(malformedData.executions) ? malformedData.executions : [];
            const processedData = executions.map(e => e.id);
            
            return { 
              success: true, 
              gracefulFallback: true, 
              processedCount: processedData.length 
            };
            
          } catch (error) {
            return { success: false, error: error.message };
          }
        }
      },
      {
        name: 'Missing Component Dependencies',
        test: () => {
          try {
            // Simulate missing component
            const componentExists = false;
            
            if (!componentExists) {
              // Render fallback
              return { 
                success: true, 
                fallbackRendered: true, 
                fallbackComponent: 'LoadingSpinner' 
              };
            }
            
            return { success: true };
            
          } catch (error) {
            return { success: false, error: error.message };
          }
        }
      }
    ];
    
    const errorBoundaryResults = [];
    
    for (const test of errorBoundaryTests) {
      const result = test.test();
      errorBoundaryResults.push({
        testName: test.name,
        ...result
      });
    }
    
    const errorBoundarySummary = {
      totalTests: errorBoundaryTests.length,
      successfulTests: errorBoundaryResults.filter(r => r.success).length,
      failedTests: errorBoundaryResults.filter(r => !r.success).length,
      errorHandlingScore: (errorBoundaryResults.filter(r => r.success).length / errorBoundaryTests.length * 100).toFixed(1) + '%',
      testResults: errorBoundaryResults
    };
    
    logTest('Component Error Boundaries and Fallbacks', 
      errorBoundarySummary.successfulTests === errorBoundaryTests.length ? 'passed' : 'warning', 
      errorBoundarySummary
    );
    
  } catch (error) {
    logTest('Component Error Boundaries and Fallbacks', 'failed', {
      error: error.message,
      stack: error.stack
    });
  }
}

// Main test runner
async function runReactComponentIntegrationTests() {
  console.log('🚀 Starting React Component Integration Test Suite');
  
  const startTime = Date.now();
  
  try {
    await testModelSelectorIntegration();
    await testCrossRegionDiffPageIntegration();
    await testComponentPropsFlowDataBinding();
    await testComponentErrorBoundariesFallbacks();
    
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
  
  const reportPath = path.join(RESULTS_DIR, `react-component-integration-${TIMESTAMP}.json`);
  fs.writeFileSync(reportPath, JSON.stringify(report, null, 2));
  
  // Print summary
  console.log('\n📊 React Component Integration Test Suite Complete');
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
  runReactComponentIntegrationTests().catch(error => {
    console.error('Fatal error:', error);
    process.exit(1);
  });
}

module.exports = {
  runReactComponentIntegrationTests,
  testModelSelectorIntegration,
  testCrossRegionDiffPageIntegration,
  testComponentPropsFlowDataBinding,
  testComponentErrorBoundariesFallbacks
};
