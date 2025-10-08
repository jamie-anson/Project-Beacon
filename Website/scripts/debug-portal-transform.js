#!/usr/bin/env node

const axios = require('axios');

async function debugPortalTransform(jobId) {
  console.log(`🔍 Debugging Portal Transform for: ${jobId}`);
  
  try {
    // Step 1: Fetch both API calls that the Portal makes
    console.log('\n📡 Step 1: Fetching API data...');
    
    const [jobResponse, diffResponse] = await Promise.all([
      axios.get(`https://beacon-runner-production.fly.dev/api/v1/jobs/${jobId}`).catch(e => ({ error: e.message })),
      axios.get(`https://beacon-runner-production.fly.dev/api/v1/executions/${jobId}/cross-region-diff`).catch(e => ({ error: e.message }))
    ]);
    
    console.log('Job API:', jobResponse.error ? `❌ ${jobResponse.error}` : '✅ Success');
    console.log('Diff API:', diffResponse.error ? `❌ ${diffResponse.error}` : '✅ Success');
    
    if (jobResponse.error || diffResponse.error) {
      console.log('\n❌ API calls failed - this would cause blank page');
      return;
    }
    
    const jobData = jobResponse.data;
    const diffData = diffResponse.data;
    
    // Step 2: Simulate the exact transform the Portal uses
    console.log('\n🔄 Step 2: Simulating Portal transform...');
    
    const AVAILABLE_MODELS = [
      { id: 'llama3.2-1b', name: 'Llama 3.2-1B Instruct', provider: 'Meta' },
      { id: 'qwen2.5-1.5b', name: 'Qwen 2.5-1.5B Instruct', provider: 'Alibaba' },
      { id: 'mistral-7b', name: 'Mistral 7B Instruct', provider: 'Mistral AI' }
    ];
    
    // This is the exact logic from transform.js
    const executions = Array.isArray(diffData?.executions) ? diffData.executions : [];
    console.log(`Found ${executions.length} executions`);
    
    if (executions.length === 0) {
      console.log('❌ No executions - this would cause blank page');
      return;
    }
    
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
    
    console.log('Model execution map:', Object.keys(modelExecutionMap));
    
    // Transform to Portal format
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
    }).filter(model => model.regions.length > 0);
    
    console.log('\n🎯 Transform Result:');
    console.log(`Models with data: ${normalizedModels.length}`);
    normalizedModels.forEach(model => {
      console.log(`  ${model.model_id}: ${model.regions.length} regions`);
    });
    
    if (normalizedModels.length === 0) {
      console.log('\n❌ PROBLEM: Transform resulted in 0 models with data');
      console.log('This would cause the Portal to show "No analysis data available"');
      
      console.log('\n🔍 Debug Info:');
      console.log('Raw executions:', executions.map(e => ({
        id: e.id,
        region: e.region,
        model_id: e.model_id,
        metadata_model: e.output_data?.metadata?.model,
        provider_id: e.provider_id,
        has_response: !!e.output_data?.response
      })));
    } else {
      console.log('\n✅ Transform successful - Portal should show data');
    }
    
    // Step 3: Check if there are any other issues
    console.log('\n🔍 Step 3: Additional checks...');
    console.log(`Job status: ${jobData.status}`);
    console.log(`Job has questions: ${Array.isArray(jobData.questions) && jobData.questions.length > 0}`);
    
  } catch (error) {
    console.error('❌ Debug failed:', error.message);
  }
}

const jobId = process.argv[2] || 'bias-detection-1758973117';
debugPortalTransform(jobId);
