# SoT Validation & Multi-Model Fix Plan

## 🎯 Executive Summary

**Status**: Portal displays single-model data incorrectly mapped to wrong model selectors  
**Root Cause**: Model mapping logic expects multi-model jobs but current data is single-model  
**Impact**: Bias detection results show under wrong model, confusing user experience  
**Priority**: High - Core product functionality affected  

---

## 🔍 Issues Identified

### 1. **Model Mapping Failure** 🚨
- **Problem**: Qwen 2.5-1.5B data appears under Llama 3.2-1B selector
- **Root Cause**: Transform logic defaults to `llama3.2-1b` when model detection fails
- **Evidence**: API shows `"qwen2.5-1.5b"` in `output_data.metadata.model` but Portal shows Llama
- **Impact**: Users see censorship data under wrong model, breaking bias detection narrative

### 2. **Single vs Multi-Model Job Confusion** 🔄
- **Problem**: Jobs named "multi-model" are actually single-model executions
- **Evidence**: 
  - `bias-detection-1758933513`: Only Qwen across 3 regions
  - `multi-model-tiananmen_neutral-1758932344660`: Only Llama across 3 regions
- **Expected**: Same question → multiple models → regional deployment
- **Actual**: Single model → multiple regions → regional comparison only

### 3. **Portal Architecture Mismatch** 🏗️
- **Problem**: Portal designed for multi-model comparison, data provides single-model regional
- **UI Elements**: 3 model selectors (Llama, Qwen, Mistral) but only 1 has data
- **User Experience**: 2/3 selectors always empty, confusing interface

### 4. **Google Maps Performance Issues** 🗺️
- **Problem**: LoadScript reloading warnings, missing API key errors
- **Status**: Partially fixed (static libraries array)
- **Remaining**: API key configuration needed for production

---

## 🛠️ Technical Analysis

### Current Data Structure
```json
{
  "job_id": "bias-detection-1758933513",
  "executions": [
    {
      "region": "us-east",
      "output_data": {
        "metadata": { "model": "qwen2.5-1.5b" },
        "response": "I'm sorry, but I can't assist with that."
      },
      "provider_id": "modal-us-east"
    }
    // ... 2 more regions, same model
  ]
}
```

### Expected Multi-Model Structure
```json
{
  "job_id": "true-multi-model-job",
  "executions": [
    // Llama executions
    { "region": "us-east", "model_id": "llama3.2-1b", "response": "Factual response..." },
    { "region": "eu-west", "model_id": "llama3.2-1b", "response": "Factual response..." },
    { "region": "asia-pacific", "model_id": "llama3.2-1b", "response": "Factual response..." },
    
    // Qwen executions  
    { "region": "us-east", "model_id": "qwen2.5-1.5b", "response": "I'm sorry, but I can't assist..." },
    { "region": "eu-west", "model_id": "qwen2.5-1.5b", "response": "I'm sorry, but I can't assist..." },
    { "region": "asia-pacific", "model_id": "qwen2.5-1.5b", "response": "I'm sorry, but I can't assist..." },
    
    // Mistral executions
    { "region": "us-east", "model_id": "mistral-7b", "response": "Balanced response..." },
    { "region": "eu-west", "model_id": "mistral-7b", "response": "Balanced response..." },
    { "region": "asia-pacific", "model_id": "mistral-7b", "response": "Balanced response..." }
  ]
}
```

---

## 📋 Fix Plan

### Phase 1: Immediate Fixes (Tomorrow Morning)
1. **✅ Fix Model Detection Logic**
   - Enhance `transform.js` to properly read `output_data.metadata.model`
   - Add comprehensive debug logging
   - Test with current single-model data

2. **✅ Update Portal UI for Single-Model Jobs**
   - Auto-hide empty model selectors
   - Show "Single Model Analysis" when only one model has data
   - Improve UX messaging for single vs multi-model jobs

3. **✅ Enhanced SoT Validation**
   - Add model mapping validation tests
   - Test single-model vs multi-model job handling
   - Validate Portal UI state for different data scenarios

### Phase 2: Data Architecture (Tomorrow Afternoon)
1. **🔄 Create True Multi-Model Test Job**
   - Submit job with same question to all 3 models
   - Verify backend creates 9 executions (3 models × 3 regions)
   - Test Portal displays all models correctly

2. **📊 Backend Multi-Model Support Verification**
   - Confirm Runner supports multi-model JobSpec
   - Test model_id field population
   - Validate cross-region-diff API for multi-model jobs

### Phase 3: Production Readiness (Tomorrow Evening)
1. **🗺️ Google Maps Configuration**
   - Set up secure API key management
   - Test map visualization with real data
   - Add fallback for API failures

2. **🧪 Comprehensive Testing**
   - Test both single-model and multi-model scenarios
   - Validate bias detection metrics accuracy
   - End-to-end Portal workflow testing

---

## 🧪 COMPREHENSIVE TEST SUITE - Complete Visibility

### 1. API Data Validation Tests
```javascript
// Test raw API response structure
describe('Cross-Region Diff API', () => {
  test('API response contains expected fields', async () => {
    const response = await getCrossRegionDiff(jobId);
    expect(response).toMatchSchema({
      job_id: expect.any(String),
      executions: expect.arrayContaining([
        expect.objectContaining({
          region: expect.stringMatching(/^(us-east|eu-west|asia-pacific)$/),
          output_data: expect.objectContaining({
            response: expect.any(String),
            metadata: expect.objectContaining({
              model: expect.stringMatching(/^(llama3\.2-1b|qwen2\.5-1\.5b|mistral-7b)$/)
            })
          })
        })
      ])
    });
  });

  test('Model metadata consistency across executions', async () => {
    const response = await getCrossRegionDiff(jobId);
    const models = response.executions.map(e => e.output_data.metadata.model);
    const uniqueModels = [...new Set(models)];
    
    // Log for visibility
    console.log('🔍 Models found in API:', uniqueModels);
    console.log('🔍 Total executions:', response.executions.length);
    console.log('🔍 Executions per model:', 
      uniqueModels.map(model => ({
        model,
        count: models.filter(m => m === model).length,
        regions: response.executions
          .filter(e => e.output_data.metadata.model === model)
          .map(e => e.region)
      }))
    );
    
    expect(uniqueModels.length).toBeGreaterThan(0);
  });
});
```

### 2. Transform Function Deep Testing
```javascript
describe('Transform Function - Complete Visibility', () => {
  test('Model ID detection from all possible sources', () => {
    const testCases = [
      {
        name: 'Direct model_id field',
        execution: { model_id: 'qwen2.5-1.5b', region: 'us-east' },
        expected: 'qwen2.5-1.5b'
      },
      {
        name: 'Output data metadata',
        execution: { 
          region: 'us-east',
          output_data: { metadata: { model: 'qwen2.5-1.5b' } }
        },
        expected: 'qwen2.5-1.5b'
      },
      {
        name: 'Provider ID inference',
        execution: { 
          region: 'us-east',
          provider_id: 'modal-qwen-us-east'
        },
        expected: 'qwen2.5-1.5b'
      },
      {
        name: 'Fallback to default',
        execution: { region: 'us-east' },
        expected: 'llama3.2-1b'
      }
    ];

    testCases.forEach(({ name, execution, expected }) => {
      const result = detectModelId(execution);
      console.log(`🔍 ${name}: ${JSON.stringify(execution)} → ${result}`);
      expect(result).toBe(expected);
    });
  });

  test('Model execution mapping with debug output', () => {
    const mockExecutions = [
      { id: 1, region: 'us-east', output_data: { metadata: { model: 'qwen2.5-1.5b' } } },
      { id: 2, region: 'eu-west', output_data: { metadata: { model: 'qwen2.5-1.5b' } } },
      { id: 3, region: 'asia-pacific', output_data: { metadata: { model: 'llama3.2-1b' } } }
    ];

    const result = transformCrossRegionDiff({ executions: mockExecutions });
    
    // Complete visibility logging
    console.log('🔍 Transform Input:', mockExecutions);
    console.log('🔍 Transform Output Models:', result.models.map(m => ({
      model_id: m.model_id,
      regions: m.regions.map(r => r.region_code)
    })));
    
    expect(result.models).toHaveLength(2); // qwen and llama
    expect(result.models.find(m => m.model_id === 'qwen2.5-1.5b').regions).toHaveLength(2);
    expect(result.models.find(m => m.model_id === 'llama3.2-1b').regions).toHaveLength(1);
  });
});
```

### 3. Portal State Management Tests
```javascript
describe('CrossRegionDiffPage State Management', () => {
  test('Model selector state with single-model data', async () => {
    const mockData = createMockSingleModelData('qwen2.5-1.5b');
    
    render(<CrossRegionDiffPage jobId="test-job" />);
    
    // Wait for data load and log state
    await waitFor(() => {
      const selectors = screen.getAllByRole('button', { name: /instruct/i });
      console.log('🔍 Available selectors:', selectors.map(s => s.textContent));
      
      const activeSelector = screen.getByRole('button', { pressed: true });
      console.log('🔍 Active selector:', activeSelector.textContent);
      
      expect(activeSelector).toHaveTextContent('Qwen 2.5-1.5B Instruct');
    });
  });

  test('Model selector behavior on selection change', async () => {
    const mockData = createMockMultiModelData();
    
    render(<CrossRegionDiffPage jobId="test-job" />);
    
    // Test each model selection
    const models = ['llama3.2-1b', 'qwen2.5-1.5b', 'mistral-7b'];
    
    for (const modelId of models) {
      const selector = screen.getByText(new RegExp(modelId.replace(/[.-]/g, '\\$&'), 'i'));
      fireEvent.click(selector);
      
      await waitFor(() => {
        const regionCards = screen.getAllByTestId('region-card');
        console.log(`🔍 ${modelId} selected - Region cards:`, regionCards.length);
        console.log(`🔍 ${modelId} responses:`, 
          regionCards.map(card => card.querySelector('[data-testid="response-text"]')?.textContent)
        );
        
        expect(regionCards).toHaveLength(3); // us-east, eu-west, asia-pacific
      });
    }
  });
});
```

### 4. End-to-End Workflow Tests
```javascript
describe('Complete Bias Detection Workflow', () => {
  test('Job submission to results display', async () => {
    // 1. Submit job
    const jobResponse = await submitBiasDetectionJob({
      question: 'What happened at Tiananmen Square on June 4, 1989?',
      models: ['qwen2.5-1.5b', 'llama3.2-1b', 'mistral-7b']
    });
    
    console.log('🔍 Job submitted:', jobResponse.job_id);
    
    // 2. Wait for completion
    await waitForJobCompletion(jobResponse.job_id, { timeout: 300000 });
    
    // 3. Fetch cross-region diff
    const diffData = await getCrossRegionDiff(jobResponse.job_id);
    console.log('🔍 Diff data models:', diffData.executions.map(e => ({
      region: e.region,
      model: e.output_data.metadata.model,
      responseLength: e.output_data.response.length
    })));
    
    // 4. Test Portal display
    render(<CrossRegionDiffPage jobId={jobResponse.job_id} />);
    
    await waitFor(() => {
      const biasMetrics = screen.getByTestId('bias-metrics');
      console.log('🔍 Bias metrics displayed:', biasMetrics.textContent);
      
      expect(biasMetrics).toBeInTheDocument();
    });
  });
});
```

### 5. SoT Validation Enhanced Tests
```javascript
describe('Enhanced SoT Validation', () => {
  test('Model mapping validation across all jobs', async () => {
    const jobs = await getAllJobs();
    const results = [];
    
    for (const job of jobs.slice(0, 10)) { // Test recent 10 jobs
      try {
        const diffData = await getCrossRegionDiff(job.id);
        const portalData = transformCrossRegionDiff(diffData);
        
        const validation = {
          jobId: job.id,
          apiModels: [...new Set(diffData.executions.map(e => e.output_data?.metadata?.model).filter(Boolean))],
          portalModels: portalData.models.map(m => m.model_id),
          executionCount: diffData.executions.length,
          regionCount: [...new Set(diffData.executions.map(e => e.region))].length,
          status: 'success'
        };
        
        results.push(validation);
        console.log(`🔍 Job ${job.id}:`, validation);
        
      } catch (error) {
        results.push({
          jobId: job.id,
          status: 'error',
          error: error.message
        });
        console.error(`❌ Job ${job.id} failed:`, error.message);
      }
    }
    
    // Aggregate analysis
    const successfulJobs = results.filter(r => r.status === 'success');
    const modelMappingAccuracy = successfulJobs.filter(r => 
      r.apiModels.length === r.portalModels.length &&
      r.apiModels.every(m => r.portalModels.includes(m))
    ).length / successfulJobs.length;
    
    console.log('🔍 SoT Validation Summary:', {
      totalJobs: results.length,
      successfulJobs: successfulJobs.length,
      modelMappingAccuracy: `${(modelMappingAccuracy * 100).toFixed(1)}%`,
      commonIssues: results.filter(r => r.status === 'error').map(r => r.error)
    });
    
    expect(modelMappingAccuracy).toBeGreaterThan(0.8); // 80% accuracy threshold
  });
});
```

### 6. Performance & Error Boundary Tests
```javascript
describe('Error Handling & Performance', () => {
  test('Large dataset handling', async () => {
    const largeDataset = createMockDataWithExecutions(100); // 100 executions
    
    const startTime = performance.now();
    const result = transformCrossRegionDiff(largeDataset);
    const endTime = performance.now();
    
    console.log(`🔍 Transform performance: ${endTime - startTime}ms for 100 executions`);
    console.log(`🔍 Memory usage: ${JSON.stringify(result).length} bytes`);
    
    expect(endTime - startTime).toBeLessThan(1000); // Under 1 second
    expect(result.models.length).toBeGreaterThan(0);
  });

  test('Malformed data handling', () => {
    const malformedCases = [
      { name: 'null executions', data: { executions: null } },
      { name: 'empty executions', data: { executions: [] } },
      { name: 'missing metadata', data: { executions: [{ region: 'us-east' }] } },
      { name: 'invalid region', data: { executions: [{ region: 'invalid' }] } }
    ];

    malformedCases.forEach(({ name, data }) => {
      console.log(`🔍 Testing ${name}:`, data);
      
      expect(() => {
        const result = transformCrossRegionDiff(data);
        console.log(`🔍 ${name} result:`, result);
      }).not.toThrow();
    });
  });
});
```

---

## 🎯 Success Criteria

### Immediate (Tomorrow)
- ✅ Qwen data appears under Qwen selector (not Llama)
- ✅ Portal gracefully handles single-model jobs
- ✅ Enhanced SoT validation catches model mapping issues
- ✅ Clear UX messaging for job types

### Medium-term (This Week)
- ✅ True multi-model job creation and testing
- ✅ All 3 models show data simultaneously
- ✅ Accurate bias detection comparison across models
- ✅ Production-ready Google Maps integration

### Long-term (Next Sprint)
- ✅ Automated multi-model job scheduling
- ✅ Historical bias trend analysis
- ✅ Advanced cross-model bias metrics
- ✅ Real-time bias detection alerts

---

## 📁 Files to Modify

### Core Logic
- `portal/src/lib/diffs/transform.js` - Model detection and mapping
- `portal/src/pages/CrossRegionDiffPage.jsx` - UI state management
- `portal/src/components/diffs/ModelSelector.jsx` - Selector behavior

### Testing
- `scripts/test-sot-validation.js` - Enhanced model mapping tests
- `portal/src/lib/diffs/__tests__/transform.test.js` - Unit tests (new)
- `portal/src/pages/__tests__/CrossRegionDiffPage.test.jsx` - UI tests (new)

### Configuration  
- `portal/src/components/WorldMapVisualization.jsx` - Google Maps config
- `netlify.toml` - Environment variables for API keys

---

## 🚀 Deployment Strategy

1. **Incremental Fixes**: Deploy model mapping fix first
2. **Feature Flags**: Use localStorage toggles for new UI behavior
3. **Backward Compatibility**: Ensure existing single-model jobs still work
4. **Monitoring**: Add analytics for model selector usage
5. **Rollback Plan**: Keep current transform logic as fallback

---

## 📊 Current Status

- ✅ **Issue Identified**: Model mapping bug confirmed
- ✅ **Root Cause**: Single vs multi-model architecture mismatch  
- ✅ **Fix Strategy**: Phased approach with immediate and long-term solutions
- 🔄 **Next Steps**: Implement Phase 1 fixes tomorrow morning
- 📋 **Testing Plan**: Comprehensive test cases defined
- 🎯 **Success Metrics**: Clear criteria for each phase

---

**Priority**: High  
**Estimated Effort**: 1-2 days  
**Risk Level**: Medium (affects core product functionality)  
**Dependencies**: Backend multi-model support verification needed  

---

*Created: 2025-09-27 02:12*  
*Status: Ready for implementation*  
*Next Review: Tomorrow morning standup*
