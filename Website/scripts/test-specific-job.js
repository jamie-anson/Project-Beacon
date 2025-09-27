#!/usr/bin/env node

const axios = require('axios');

async function testSpecificJob(jobId) {
  console.log(`🔍 Testing job: ${jobId}`);
  
  try {
    // Test API call
    const response = await axios.get(`https://beacon-runner-change-me.fly.dev/api/v1/executions/${jobId}/cross-region-diff`);
    const apiData = response.data;
    
    console.log('📊 API Response:', {
      jobId: apiData.job_id,
      totalRegions: apiData.total_regions,
      executionCount: apiData.executions.length,
      models: [...new Set(apiData.executions.map(e => e.output_data?.metadata?.model))].filter(Boolean)
    });
    
    // Test transform
    const AVAILABLE_MODELS = [
      { id: 'llama3.2-1b', name: 'Llama 3.2-1B Instruct', provider: 'Meta' },
      { id: 'qwen2.5-1.5b', name: 'Qwen 2.5-1.5B Instruct', provider: 'Alibaba' },
      { id: 'mistral-7b', name: 'Mistral 7B Instruct', provider: 'Mistral AI' }
    ];
    
    // Simulate transform
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
    
    const transformedModels = AVAILABLE_MODELS.map((model) => {
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
    
    console.log('🎯 Transform Result:', {
      modelsWithData: transformedModels.length,
      modelIds: transformedModels.map(m => m.model_id),
      regionCounts: transformedModels.map(m => ({ model: m.model_id, regions: m.regions.length }))
    });
    
    if (transformedModels.length === 0) {
      console.log('❌ PROBLEM: No models with data after transform!');
      console.log('🔍 Debug Info:', {
        rawExecutions: apiData.executions.map(e => ({
          id: e.id,
          region: e.region,
          model_id: e.model_id,
          metadata_model: e.output_data?.metadata?.model,
          provider_id: e.provider_id
        })),
        modelExecutionMap
      });
    } else {
      console.log('✅ Transform successful');
    }
    
  } catch (error) {
    console.error('❌ Error:', error.message);
  }
}

const jobId = process.argv[2] || 'bias-detection-1758973117';
testSpecificJob(jobId);
