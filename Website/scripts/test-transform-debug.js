#!/usr/bin/env node

/**
 * Debug the exact transform logic with real API data
 */

const axios = require('axios');

// Mock the transform function logic to debug step by step
async function debugTransform() {
  console.log('🔍 Fetching real API data...');
  
  const response = await axios.get('https://beacon-runner-production.fly.dev/api/v1/executions/bias-detection-1758933513/cross-region-diff');
  const apiData = response.data;
  
  console.log('📊 API Data:', {
    jobId: apiData.job_id,
    totalExecutions: apiData.executions.length,
    executionModels: apiData.executions.map(e => ({
      region: e.region,
      model: e.output_data?.metadata?.model,
      response: e.output_data?.response?.substring(0, 50) + '...'
    }))
  });
  
  // Step 1: Model detection (simulate the transform logic)
  console.log('\n🔍 Step 1: Model Detection');
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
    
    console.log(`  Execution ${exec.id}: ${exec.region} → ${modelId}`);
    
    if (!modelExecutionMap[modelId]) {
      modelExecutionMap[modelId] = {};
    }
    
    modelExecutionMap[modelId][exec.region] = exec;
  });
  
  console.log('\n📊 Model Execution Map:', {
    detectedModels: Object.keys(modelExecutionMap),
    executionsPerModel: Object.entries(modelExecutionMap).map(([modelId, regions]) => ({
      modelId,
      regionCount: Object.keys(regions).length,
      regions: Object.keys(regions)
    }))
  });
  
  // Step 2: Available models mapping
  console.log('\n🔍 Step 2: Available Models Mapping');
  const AVAILABLE_MODELS = [
    { id: 'llama3.2-1b', name: 'Llama 3.2-1B Instruct', provider: 'Meta' },
    { id: 'qwen2.5-1.5b', name: 'Qwen 2.5-1.5B Instruct', provider: 'Alibaba' },
    { id: 'mistral-7b', name: 'Mistral 7B Instruct', provider: 'Mistral AI' }
  ];
  
  const normalizedModels = AVAILABLE_MODELS.map((model) => {
    const modelExecutions = modelExecutionMap[model.id] || {};
    
    console.log(`  Model ${model.id}:`, {
      hasExecutions: Object.keys(modelExecutions).length > 0,
      regions: Object.keys(modelExecutions),
      executionCount: Object.keys(modelExecutions).length
    });
    
    const regions = Object.keys(modelExecutions).map((regionCode) => {
      const exec = modelExecutions[regionCode];
      const response = exec?.output_data?.response || 'No response available';
      
      return {
        region_code: regionCode,
        region_name: regionCode,
        response: response,
        hasData: !!exec
      };
    }).filter(region => region.hasData); // Only include regions with actual data
    
    return {
      model_id: model.id,
      model_name: model.name,
      provider: model.provider,
      regions: regions
    };
  });
  
  console.log('\n📊 Final Normalized Models:');
  normalizedModels.forEach(model => {
    console.log(`  ${model.model_id}:`, {
      regionCount: model.regions.length,
      regions: model.regions.map(r => r.region_code),
      hasData: model.regions.length > 0
    });
  });
  
  // Step 3: Check what Portal would see
  console.log('\n🎯 Portal View:');
  const modelsWithData = normalizedModels.filter(m => m.regions.length > 0);
  const modelsWithoutData = normalizedModels.filter(m => m.regions.length === 0);
  
  console.log('Models WITH data:', modelsWithData.map(m => m.model_id));
  console.log('Models WITHOUT data:', modelsWithoutData.map(m => m.model_id));
  
  if (modelsWithData.length === 1) {
    console.log('\n✅ This is a SINGLE-MODEL job');
    console.log('Expected behavior: Only show', modelsWithData[0].model_id, 'selector with data');
    console.log('Problem: Portal might be showing data under wrong selector');
  } else {
    console.log('\n✅ This is a MULTI-MODEL job');
  }
}

debugTransform().catch(console.error);
