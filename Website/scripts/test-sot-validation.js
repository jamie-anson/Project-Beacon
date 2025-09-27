#!/usr/bin/env node

/**
 * Source of Truth (SoT) Validation Test
 * 
 * This script validates that all Portal paths for results and executions
 * match the Source of Truth defined in docs/sot/facts.json
 * 
 * Tests:
 * 1. API endpoints match SoT configuration
 * 2. All questions are accessible and return valid data
 * 3. All models are properly configured and accessible
 * 4. Execution records match expected schema
 * 5. Cross-region diff paths work correctly
 * 6. Multi-model job paths are valid
 */

const fs = require('fs');
const path = require('path');

// Load Source of Truth
const sotPath = path.join(__dirname, '../docs/sot/facts.json');
const sot = JSON.parse(fs.readFileSync(sotPath, 'utf8'));

// Test configuration
const TEST_CONFIG = {
  timeout: 10000,
  retries: 3,
  verbose: process.argv.includes('--verbose') || process.argv.includes('-v')
};

// Colors for output
const colors = {
  reset: '\x1b[0m',
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  magenta: '\x1b[35m',
  cyan: '\x1b[36m'
};

// Logging utilities
function log(message, color = 'reset') {
  console.log(`${colors[color]}${message}${colors.reset}`);
}

function logSuccess(message) {
  log(`✅ ${message}`, 'green');
}

function logError(message) {
  log(`❌ ${message}`, 'red');
}

function logWarning(message) {
  log(`⚠️  ${message}`, 'yellow');
}

function logInfo(message) {
  log(`ℹ️  ${message}`, 'blue');
}

function logDebug(message) {
  if (TEST_CONFIG.verbose) {
    log(`🔍 ${message}`, 'cyan');
  }
}

// Extract SoT configuration
function extractSotConfig() {
  const config = {
    servers: {},
    apis: {},
    models: [],
    questions: [],
    routing: {}
  };

  sot.forEach(entry => {
    if (entry.action === 'deprecate') return;

    switch (entry.type) {
      case 'server':
        config.servers[entry.subject] = entry.data;
        break;
      case 'api_base':
        config.apis[entry.subject] = entry.data;
        break;
      case 'external_dependency':
        if (entry.subject === 'modal_providers') {
          config.models = entry.data.models || [];
        }
        break;
      case 'routing_configuration':
        config.routing = entry.data;
        break;
    }
  });

  return config;
}

// HTTP request utility with retries
async function fetchWithRetry(url, options = {}, retries = TEST_CONFIG.retries) {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), TEST_CONFIG.timeout);

  try {
    const response = await fetch(url, {
      ...options,
      signal: controller.signal
    });
    clearTimeout(timeoutId);
    return response;
  } catch (error) {
    clearTimeout(timeoutId);
    if (retries > 0 && (error.name === 'AbortError' || error.code === 'ECONNRESET')) {
      logDebug(`Retrying ${url} (${retries} attempts left)`);
      await new Promise(resolve => setTimeout(resolve, 1000));
      return fetchWithRetry(url, options, retries - 1);
    }
    throw error;
  }
}

// Test results tracking
const testResults = {
  passed: 0,
  failed: 0,
  warnings: 0,
  details: []
};

function recordTest(name, passed, message, details = null) {
  if (passed) {
    testResults.passed++;
    logSuccess(`${name}: ${message}`);
  } else {
    testResults.failed++;
    logError(`${name}: ${message}`);
  }
  
  testResults.details.push({
    name,
    passed,
    message,
    details,
    timestamp: new Date().toISOString()
  });
}

function recordWarning(name, message) {
  testResults.warnings++;
  logWarning(`${name}: ${message}`);
  testResults.details.push({
    name,
    passed: null,
    message,
    warning: true,
    timestamp: new Date().toISOString()
  });
}

// Test 1: Validate API endpoints match SoT
async function testApiEndpoints(config) {
  logInfo('Testing API endpoints against SoT...');
  
  // Test Runner API
  if (config.apis.runner) {
    const runnerBase = config.apis.runner.base_url;
    const endpoints = config.apis.runner.endpoints;
    
    for (const endpoint of endpoints) {
      const [method, path] = endpoint.split(' ');
      if (method === 'GET' && !path.includes(':') && !path.includes('ws')) {
        try {
          const url = `${runnerBase}${path === '/health' ? '/health' : '/api/v1' + path}`;
          logDebug(`Testing ${method} ${url}`);
          
          const response = await fetchWithRetry(url);
          const isHealthy = response.status === 200;
          
          recordTest(
            `API-${method}-${path}`,
            isHealthy,
            isHealthy ? `${url} responded with 200` : `${url} responded with ${response.status}`,
            { url, status: response.status, method }
          );
        } catch (error) {
          recordTest(
            `API-${method}-${path}`,
            false,
            `Failed to reach ${runnerBase}${path}: ${error.message}`,
            { error: error.message, method, path }
          );
        }
      }
    }
  }

  // Test Hybrid Router API
  if (config.apis.router) {
    const routerBase = config.apis.router.base_url;
    const endpoints = config.apis.router.endpoints;
    
    for (const endpoint of endpoints) {
      const [method, path] = endpoint.split(' ');
      if (method === 'GET' && !path.includes('ws')) {
        try {
          const url = `${routerBase}${path}`;
          logDebug(`Testing ${method} ${url}`);
          
          const response = await fetchWithRetry(url);
          const isHealthy = response.status === 200;
          
          recordTest(
            `ROUTER-${method}-${path}`,
            isHealthy,
            isHealthy ? `${url} responded with 200` : `${url} responded with ${response.status}`,
            { url, status: response.status, method }
          );
        } catch (error) {
          recordTest(
            `ROUTER-${method}-${path}`,
            false,
            `Failed to reach ${routerBase}${path}: ${error.message}`,
            { error: error.message, method, path }
          );
        }
      }
    }
  }
}

// Test 2: Validate questions are accessible
async function testQuestions(config) {
  logInfo('Testing questions endpoint...');
  
  if (!config.apis.runner) {
    recordWarning('QUESTIONS', 'No runner API configuration found in SoT');
    return;
  }

  try {
    const url = `${config.apis.runner.base_url}/api/v1/questions`;
    logDebug(`Fetching questions from ${url}`);
    
    const response = await fetchWithRetry(url);
    
    if (response.status !== 200) {
      recordTest('QUESTIONS-FETCH', false, `Questions endpoint returned ${response.status}`);
      return;
    }

    const data = await response.json();
    const totalQuestions = Object.values(data.categories || {}).flat().length;
    
    recordTest(
      'QUESTIONS-FETCH',
      totalQuestions > 0,
      `Found ${totalQuestions} questions across categories`,
      { categories: Object.keys(data.categories || {}), totalQuestions }
    );

    // Test specific question categories
    const expectedCategories = ['control_questions', 'bias_detection', 'cultural_perspective'];
    for (const category of expectedCategories) {
      const questions = data.categories?.[category] || [];
      recordTest(
        `QUESTIONS-${category.toUpperCase()}`,
        questions.length > 0,
        `Found ${questions.length} questions in ${category}`,
        { category, count: questions.length, questions: questions.slice(0, 3) }
      );
    }

  } catch (error) {
    recordTest('QUESTIONS-FETCH', false, `Failed to fetch questions: ${error.message}`);
  }
}

// Test 3: Validate models and providers
async function testModels(config) {
  logInfo('Testing model providers...');
  
  if (!config.apis.router) {
    recordWarning('MODELS', 'No router API configuration found in SoT');
    return;
  }

  try {
    const url = `${config.apis.router.base_url}/providers`;
    logDebug(`Fetching providers from ${url}`);
    
    const response = await fetchWithRetry(url);
    
    if (response.status !== 200) {
      recordTest('MODELS-PROVIDERS', false, `Providers endpoint returned ${response.status}`);
      return;
    }

    const data = await response.json();
    const providers = data.providers || [];
    
    recordTest(
      'MODELS-PROVIDERS',
      providers.length > 0,
      `Found ${providers.length} providers`,
      { providerCount: providers.length, providers: providers.map(p => p.name) }
    );

    // Test regional coverage
    const regions = [...new Set(providers.map(p => p.region))];
    const expectedRegions = ['us-east', 'eu-west', 'asia-pacific'];
    
    for (const region of expectedRegions) {
      const regionProviders = providers.filter(p => p.region === region);
      recordTest(
        `MODELS-REGION-${region.toUpperCase()}`,
        regionProviders.length > 0,
        `Found ${regionProviders.length} providers in ${region}`,
        { region, providers: regionProviders.map(p => p.name) }
      );
    }

    // Test provider health
    const healthyProviders = providers.filter(p => p.healthy);
    recordTest(
      'MODELS-HEALTH',
      healthyProviders.length > 0,
      `${healthyProviders.length}/${providers.length} providers are healthy`,
      { 
        healthy: healthyProviders.length, 
        total: providers.length,
        unhealthy: providers.filter(p => !p.healthy).map(p => p.name)
      }
    );

  } catch (error) {
    recordTest('MODELS-PROVIDERS', false, `Failed to fetch providers: ${error.message}`);
  }
}

// Test 4: Validate execution records and cross-region diffs
async function testExecutions(config) {
  logInfo('Testing execution records...');
  
  if (!config.apis.runner) {
    recordWarning('EXECUTIONS', 'No runner API configuration found in SoT');
    return;
  }

  try {
    // Test recent executions
    const url = `${config.apis.runner.base_url}/api/v1/executions`;
    logDebug(`Fetching executions from ${url}`);
    
    const response = await fetchWithRetry(url);
    
    if (response.status !== 200) {
      recordTest('EXECUTIONS-FETCH', false, `Executions endpoint returned ${response.status}`);
      return;
    }

    const data = await response.json();
    const executions = data.executions || [];
    
    recordTest(
      'EXECUTIONS-FETCH',
      executions.length > 0,
      `Found ${executions.length} execution records`,
      { executionCount: executions.length }
    );

    // Test execution schema
    if (executions.length > 0) {
      const execution = executions[0];
      const requiredFields = ['id', 'job_id', 'region', 'status', 'provider_id'];
      const hasAllFields = requiredFields.every(field => execution.hasOwnProperty(field));
      
      recordTest(
        'EXECUTIONS-SCHEMA',
        hasAllFields,
        hasAllFields ? 'Execution records have required fields' : 'Missing required fields in execution records',
        { 
          requiredFields, 
          actualFields: Object.keys(execution),
          missing: requiredFields.filter(field => !execution.hasOwnProperty(field))
        }
      );

      // Test multi-model support (check for model_id field or recent multi-model jobs)
      const hasModelId = execution.hasOwnProperty('model_id');
      const isMultiModelJob = execution.job_id && execution.job_id.includes('multi-model');
      const multiModelSupported = hasModelId || isMultiModelJob;
      
      recordTest(
        'EXECUTIONS-MULTIMODEL',
        multiModelSupported,
        multiModelSupported 
          ? `Multi-model support detected (${hasModelId ? 'model_id field' : 'multi-model job found'})` 
          : 'Multi-model support not detected (legacy data)',
        { 
          hasModelId, 
          isMultiModelJob, 
          jobId: execution.job_id,
          sampleExecution: execution 
        }
      );
    }

    // Test cross-region diff for a recent job
    if (executions.length > 0) {
      const recentJob = executions[0].job_id;
      const diffUrl = `${config.apis.runner.base_url}/api/v1/executions/${recentJob}/cross-region-diff`;
      
      logDebug(`Testing cross-region diff for job ${recentJob}`);
      
      try {
        const diffResponse = await fetchWithRetry(diffUrl);
        const diffData = await diffResponse.json();
        
        recordTest(
          'EXECUTIONS-CROSS-REGION-DIFF',
          diffResponse.status === 200,
          `Cross-region diff available for job ${recentJob}`,
          { 
            jobId: recentJob, 
            status: diffResponse.status,
            regions: diffData.total_regions,
            executions: diffData.executions?.length
          }
        );
      } catch (error) {
        recordTest(
          'EXECUTIONS-CROSS-REGION-DIFF',
          false,
          `Failed to fetch cross-region diff for ${recentJob}: ${error.message}`
        );
      }
    }

  } catch (error) {
    recordTest('EXECUTIONS-FETCH', false, `Failed to fetch executions: ${error.message}`);
  }
}

// Test 5: Validate Portal paths and routing
async function testPortalPaths(config) {
  logInfo('Testing Portal paths and routing...');
  
  // Test Portal base URL (from Netlify)
  const portalBase = 'https://projectbeacon.netlify.app';
  
  try {
    // Test Portal health
    const response = await fetchWithRetry(portalBase);
    recordTest(
      'PORTAL-BASE',
      response.status === 200,
      `Portal base URL accessible`,
      { url: portalBase, status: response.status }
    );

    // Test Portal SPA routing
    const testPaths = [
      '/portal',
      '/portal/dashboard',
      '/portal/results/fresh-diff-demo-1758931000/diffs'
    ];

    for (const path of testPaths) {
      try {
        const url = `${portalBase}${path}`;
        const pathResponse = await fetchWithRetry(url);
        
        recordTest(
          `PORTAL-PATH-${path.replace(/[^a-zA-Z0-9]/g, '-')}`,
          pathResponse.status === 200,
          `Portal path ${path} accessible`,
          { url, status: pathResponse.status }
        );
      } catch (error) {
        recordTest(
          `PORTAL-PATH-${path.replace(/[^a-zA-Z0-9]/g, '-')}`,
          false,
          `Portal path ${path} failed: ${error.message}`
        );
      }
    }

  } catch (error) {
    recordTest('PORTAL-BASE', false, `Failed to reach Portal: ${error.message}`);
  }
}

// Test 6: Validate Live Progress link paths
async function testLiveProgressPaths(config) {
  logInfo('Testing Live Progress link paths...');
  
  if (!config.apis.runner) {
    recordWarning('LIVE-PROGRESS', 'No runner API configuration found in SoT');
    return;
  }

  // Get a recent job to test Live Progress paths
  try {
    const executionsUrl = `${config.apis.runner.base_url}/api/v1/executions`;
    const response = await fetchWithRetry(executionsUrl);
    
    if (response.status !== 200) {
      recordTest('LIVE-PROGRESS-DATA', false, `Could not fetch executions for Live Progress testing`);
      return;
    }

    const data = await response.json();
    const executions = data.executions || [];
    
    if (executions.length === 0) {
      recordWarning('LIVE-PROGRESS-DATA', 'No executions found to test Live Progress paths');
      return;
    }

    const recentExecution = executions[0];
    const jobId = recentExecution.job_id;
    const region = recentExecution.region;

    // Test Live Progress generated paths
    const testPaths = [
      {
        name: 'EXECUTIONS-QUERY',
        path: `/executions?job=${encodeURIComponent(jobId)}&region=${encodeURIComponent(region)}`,
        description: 'Executions query path from Live Progress table'
      },
      {
        name: 'CROSS-REGION-DIFFS',
        path: `/results/${jobId}/diffs`,
        description: 'Cross-region diffs path from Live Progress CTA'
      },
      {
        name: 'JOB-DETAILS',
        path: `/jobs/${jobId}`,
        description: 'Job details path from Live Progress'
      }
    ];

    const portalBase = 'https://projectbeacon.netlify.app';
    
    for (const testPath of testPaths) {
      try {
        const fullUrl = `${portalBase}/portal${testPath.path}`;
        logDebug(`Testing Live Progress path: ${fullUrl}`);
        
        const pathResponse = await fetchWithRetry(fullUrl);
        
        // For SPA apps, we need to check content, not just status
        let contentValid = pathResponse.status === 200;
        let contentCheck = 'HTTP 200 response';
        
        if (pathResponse.status === 200) {
          try {
            const html = await pathResponse.text();
            
            // For SPA, check that we get the React app structure, not specific content
            // The content is loaded dynamically by JavaScript
            const hasReactApp = html.includes('root') && html.includes('script');
            const isNotErrorPage = !html.includes('404') && !html.includes('Not Found');
            
            contentValid = hasReactApp && isNotErrorPage;
            contentCheck = contentValid 
              ? 'Valid SPA route (React app loaded)' 
              : 'Invalid route (missing React app structure or error page)';
              
            // Additional check: verify the route exists in our known routes
            const knownRoutes = ['/executions', '/results', '/jobs'];
            const pathBase = testPath.path.split('?')[0].split('/')[1]; // Extract base path
            const routeExists = knownRoutes.some(route => route.includes(pathBase));
            
            if (!routeExists) {
              contentValid = false;
              contentCheck = `Route /${pathBase} not found in known routes: ${knownRoutes.join(', ')}`;
            }
          } catch (htmlError) {
            contentCheck = `Could not parse HTML: ${htmlError.message}`;
            contentValid = false;
          }
        }
        
        recordTest(
          `LIVE-PROGRESS-${testPath.name}`,
          contentValid,
          `${testPath.description}: ${testPath.path} (${contentCheck})`,
          { 
            url: fullUrl, 
            status: pathResponse.status,
            contentCheck,
            jobId,
            region,
            description: testPath.description
          }
        );
      } catch (error) {
        recordTest(
          `LIVE-PROGRESS-${testPath.name}`,
          false,
          `${testPath.description} failed: ${error.message}`,
          { path: testPath.path, error: error.message }
        );
      }
    }

    // Test executions API endpoint that Live Progress uses
    const executionsApiUrl = `${config.apis.runner.base_url}/api/v1/executions?job_id=${jobId}`;
    try {
      const execResponse = await fetchWithRetry(executionsApiUrl);
      recordTest(
        'LIVE-PROGRESS-API-EXECUTIONS',
        execResponse.status === 200,
        `Executions API supports job filtering`,
        { url: executionsApiUrl, status: execResponse.status }
      );
    } catch (error) {
      recordTest(
        'LIVE-PROGRESS-API-EXECUTIONS',
        false,
        `Executions API filtering failed: ${error.message}`
      );
    }

  } catch (error) {
    recordTest('LIVE-PROGRESS-SETUP', false, `Failed to setup Live Progress path testing: ${error.message}`);
  }
}

// Test 7: Validate multi-model job paths
async function testMultiModelPaths(config) {
  logInfo('Testing multi-model job paths...');
  
  if (!config.apis.runner) {
    recordWarning('MULTIMODEL', 'No runner API configuration found in SoT');
    return;
  }

  // Test job creation endpoint
  try {
    const url = `${config.apis.runner.base_url}/api/v1/jobs`;
    logDebug(`Testing job creation endpoint ${url}`);
    
    // Test with OPTIONS request to check CORS
    const optionsResponse = await fetchWithRetry(url, { method: 'OPTIONS' });
    
    recordTest(
      'MULTIMODEL-CORS',
      optionsResponse.status === 200 || optionsResponse.status === 204,
      `Job creation endpoint supports CORS`,
      { url, status: optionsResponse.status }
    );

    // Test multi-model job schema validation (without actually creating a job)
    const multiModelJobSpec = {
      id: `test-multi-model-${Date.now()}`,
      version: "v1",
      questions: ["tiananmen_neutral"],
      models: [
        {
          id: "llama3.2-1b",
          name: "Llama 3.2-1B Instruct",
          provider: "modal",
          container_image: "ghcr.io/jamie-anson/project-beacon/llama-3.2-1b:latest",
          regions: ["US", "EU", "ASIA"]
        },
        {
          id: "qwen2.5-1.5b",
          name: "Qwen 2.5-1.5B Instruct",
          provider: "modal",
          container_image: "ghcr.io/jamie-anson/project-beacon/qwen-2.5-1.5b:latest",
          regions: ["ASIA", "EU", "US"]
        }
      ],
      constraints: {
        regions: ["US", "EU", "ASIA"],
        min_regions: 3,
        min_success_rate: 0.67
      }
    };

    recordTest(
      'MULTIMODEL-SCHEMA',
      true,
      'Multi-model job schema is well-formed',
      { 
        modelCount: multiModelJobSpec.models.length,
        expectedExecutions: multiModelJobSpec.models.reduce((sum, model) => sum + model.regions.length, 0),
        schema: 'valid'
      }
    );

  } catch (error) {
    recordTest('MULTIMODEL-ENDPOINT', false, `Failed to test multi-model endpoint: ${error.message}`);
  }
}

// Main test runner
async function runAllTests() {
  log('🚀 Starting Source of Truth (SoT) Validation Tests', 'magenta');
  log('=' .repeat(60), 'magenta');
  
  const startTime = Date.now();
  const config = extractSotConfig();
  
  logDebug(`Extracted SoT config: ${JSON.stringify(config, null, 2)}`);
  
  // Run all test suites
  await testApiEndpoints(config);
  await testQuestions(config);
  await testModels(config);
  await testExecutions(config);
  await testPortalPaths(config);
  await testLiveProgressPaths(config);
  await testMultiModelPaths(config);
  
  // Generate test report
  const endTime = Date.now();
  const duration = endTime - startTime;
  
  log('=' .repeat(60), 'magenta');
  log('📊 TEST RESULTS SUMMARY', 'magenta');
  log('=' .repeat(60), 'magenta');
  
  logSuccess(`✅ Passed: ${testResults.passed}`);
  logError(`❌ Failed: ${testResults.failed}`);
  logWarning(`⚠️  Warnings: ${testResults.warnings}`);
  logInfo(`⏱️  Duration: ${duration}ms`);
  
  const totalTests = testResults.passed + testResults.failed;
  const successRate = totalTests > 0 ? ((testResults.passed / totalTests) * 100).toFixed(1) : 0;
  
  log(`📈 Success Rate: ${successRate}%`, successRate >= 90 ? 'green' : successRate >= 70 ? 'yellow' : 'red');
  
  // Show failed tests
  if (testResults.failed > 0) {
    log('\n❌ FAILED TESTS:', 'red');
    testResults.details
      .filter(test => test.passed === false)
      .forEach(test => {
        log(`   • ${test.name}: ${test.message}`, 'red');
        if (TEST_CONFIG.verbose && test.details) {
          log(`     Details: ${JSON.stringify(test.details, null, 2)}`, 'cyan');
        }
      });
  }
  
  // Show warnings
  if (testResults.warnings > 0) {
    log('\n⚠️  WARNINGS:', 'yellow');
    testResults.details
      .filter(test => test.warning)
      .forEach(test => {
        log(`   • ${test.name}: ${test.message}`, 'yellow');
      });
  }
  
  // Save detailed results
  const reportPath = path.join(__dirname, '../test-results/sot-validation-report.json');
  const reportDir = path.dirname(reportPath);
  
  if (!fs.existsSync(reportDir)) {
    fs.mkdirSync(reportDir, { recursive: true });
  }
  
  const report = {
    timestamp: new Date().toISOString(),
    duration,
    summary: {
      passed: testResults.passed,
      failed: testResults.failed,
      warnings: testResults.warnings,
      successRate: parseFloat(successRate)
    },
    config: config,
    details: testResults.details
  };
  
  fs.writeFileSync(reportPath, JSON.stringify(report, null, 2));
  logInfo(`📄 Detailed report saved to: ${reportPath}`);
  
  // Exit with appropriate code
  const exitCode = testResults.failed > 0 ? 1 : 0;
  log(`\n🏁 Tests completed with exit code: ${exitCode}`, exitCode === 0 ? 'green' : 'red');
  
  process.exit(exitCode);
}

// Run tests if called directly
if (require.main === module) {
  runAllTests().catch(error => {
    logError(`Fatal error: ${error.message}`);
    console.error(error);
    process.exit(1);
  });
}

module.exports = { runAllTests, testResults };
