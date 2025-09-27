#!/usr/bin/env node

/**
 * Test what data the Portal actually receives
 */

const axios = require('axios');

async function testPortalData() {
  console.log('🔍 Testing Portal Data Flow...');
  
  // Step 1: Get API data
  const response = await axios.get('https://beacon-runner-change-me.fly.dev/api/v1/executions/bias-detection-1758933513/cross-region-diff');
  const apiData = response.data;
  
  // Step 2: Simulate the exact transform logic from the Portal
  const AVAILABLE_MODELS = [
    { id: 'llama3.2-1b', name: 'Llama 3.2-1B Instruct', provider: 'Meta' },
    { id: 'qwen2.5-1.5b', name: 'Qwen 2.5-1.5B Instruct', provider: 'Alibaba' },
    { id: 'mistral-7b', name: 'Mistral 7B Instruct', provider: 'Mistral AI' }
  ];
  
  // Model detection
  const modelExecutionMap = {};
  apiData.executions.forEach(exec => {
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
    
    if (!modelExecutionMap[modelId]) {
      modelExecutionMap[modelId] = {};
    }
    
    modelExecutionMap[modelId][exec.region] = exec;
  });
  
  console.log('📊 Model Execution Map:', Object.keys(modelExecutionMap));
  
  // Transform to normalized models (exact Portal logic)
  const normalizedModels = AVAILABLE_MODELS.map((model) => {
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
  }).filter(model => model.regions.length > 0); // Only include models with actual execution data
  
  console.log('\n📊 Normalized Models (what Portal receives):');
  normalizedModels.forEach((model, index) => {
    console.log(`  [${index}] ${model.model_id}: ${model.regions.length} regions`);
    console.log(`      Regions: ${model.regions.map(r => r.region_code).join(', ')}`);
    console.log(`      Sample response: ${model.regions[0]?.response?.substring(0, 50)}...`);
  });
  
  // Step 3: Simulate Portal auto-selection
  if (normalizedModels.length > 0) {
    const firstModel = normalizedModels[0];
    console.log(`\n🎯 Portal would auto-select: ${firstModel.model_id}`);
    console.log(`   This should be: qwen2.5-1.5b`);
    console.log(`   Match: ${firstModel.model_id === 'qwen2.5-1.5b' ? '✅ CORRECT' : '❌ WRONG'}`);
  }
  
  // Step 4: Check if there's any data leakage
  console.log('\n🔍 Checking for data leakage...');
  const llamaModel = normalizedModels.find(m => m.model_id === 'llama3.2-1b');
  const qwenModel = normalizedModels.find(m => m.model_id === 'qwen2.5-1.5b');
  
  if (llamaModel) {
    console.log('❌ PROBLEM: Llama model found in results with', llamaModel.regions.length, 'regions');
    console.log('   Llama responses:', llamaModel.regions.map(r => r.response.substring(0, 50)));
  } else {
    console.log('✅ GOOD: No Llama model in results');
  }
  
  if (qwenModel) {
    console.log('✅ GOOD: Qwen model found with', qwenModel.regions.length, 'regions');
    console.log('   Qwen responses:', qwenModel.regions.map(r => r.response.substring(0, 50)));
  } else {
    console.log('❌ PROBLEM: No Qwen model in results');
  }
}

testPortalData().catch(console.error);
