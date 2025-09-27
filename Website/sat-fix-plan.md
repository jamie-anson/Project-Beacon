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

## 🧪 Test Cases to Implement

### Model Mapping Tests
```javascript
// Test 1: Single-model job (current scenario)
expect(transform(qwenOnlyData)).toHaveModels(['qwen2.5-1.5b']);
expect(transform(qwenOnlyData).models[0].regions).toHaveLength(3);

// Test 2: Multi-model job (target scenario)  
expect(transform(multiModelData)).toHaveModels(['llama3.2-1b', 'qwen2.5-1.5b', 'mistral-7b']);
expect(transform(multiModelData).models).toEach(model => 
  expect(model.regions).toHaveLength(3)
);

// Test 3: Model detection accuracy
expect(detectModelId(qwenExecution)).toBe('qwen2.5-1.5b');
expect(detectModelId(llamaExecution)).toBe('llama3.2-1b');
```

### Portal UI Tests
```javascript
// Test 1: Single-model UI state
render(<CrossRegionDiffPage jobId="bias-detection-1758933513" />);
expect(screen.getByText('Qwen 2.5-1.5B Instruct')).toBeVisible();
expect(screen.queryByText('Llama 3.2-1B Instruct')).toHaveClass('disabled');

// Test 2: Multi-model UI state  
render(<CrossRegionDiffPage jobId="true-multi-model-job" />);
expect(screen.getAllByText(/Instruct/)).toHaveLength(3);
```

### SoT Validation Enhancements
```javascript
// Test model mapping validation
await testModelMapping(jobId);
await testPortalUIState(jobId);
await testBiasMetricsAccuracy(jobId);
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
