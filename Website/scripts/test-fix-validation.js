#!/usr/bin/env node

/**
 * Validate that the ModelSelector fix works correctly
 */

const axios = require('axios');

async function validateFix() {
  console.log('🧪 Validating ModelSelector Fix...');
  
  // Test the exact logic that Portal now uses
  const response = await axios.get('https://beacon-runner-production.fly.dev/api/v1/executions/bias-detection-1758933513/cross-region-diff');
  const apiData = response.data;
  
  // Simulate transform (same as before)
  const AVAILABLE_MODELS = [
    { id: 'llama3.2-1b', name: 'Llama 3.2-1B Instruct', provider: 'Meta' },
    { id: 'qwen2.5-1.5b', name: 'Qwen 2.5-1.5B Instruct', provider: 'Alibaba' },
    { id: 'mistral-7b', name: 'Mistral 7B Instruct', provider: 'Mistral AI' }
  ];
  
  const modelExecutionMap = {};
  apiData.executions.forEach(exec => {
    let modelId = exec.output_data?.metadata?.model || 'llama3.2-1b';
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
  
  console.log('📊 diffAnalysis.models (filtered):', diffAnalysisModels.map(m => ({
    model_id: m.model_id,
    regionCount: m.regions.length
  })));
  
  // NEW LOGIC: ModelSelector models (what Portal now shows)
  const modelSelectorModels = diffAnalysisModels.map(m => 
    AVAILABLE_MODELS.find(am => am.id === m.model_id)
  ).filter(Boolean);
  
  console.log('\n🎯 ModelSelector will show:', modelSelectorModels.map(m => ({
    id: m.id,
    name: m.name
  })));
  
  // Validation
  console.log('\n✅ VALIDATION RESULTS:');
  console.log(`   Models with data: ${diffAnalysisModels.length}`);
  console.log(`   ModelSelector buttons: ${modelSelectorModels.length}`);
  console.log(`   Match: ${diffAnalysisModels.length === modelSelectorModels.length ? '✅ CORRECT' : '❌ WRONG'}`);
  
  if (modelSelectorModels.length === 1 && modelSelectorModels[0].id === 'qwen2.5-1.5b') {
    console.log('🎉 SUCCESS: Only Qwen selector will be shown!');
    console.log('   Users can no longer click on empty Llama/Mistral selectors');
  } else {
    console.log('❌ PROBLEM: Fix did not work as expected');
  }
  
  // Test auto-selection
  const firstAvailableModel = diffAnalysisModels[0]?.model_id;
  console.log(`\n🎯 Auto-selected model: ${firstAvailableModel}`);
  console.log(`   Expected: qwen2.5-1.5b`);
  console.log(`   Match: ${firstAvailableModel === 'qwen2.5-1.5b' ? '✅ CORRECT' : '❌ WRONG'}`);
}

validateFix().catch(console.error);
