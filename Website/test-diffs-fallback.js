#!/usr/bin/env node

/**
 * Test the cross-region diff fallback construction
 */

async function testDiffsFallback() {
  console.log('🧪 Testing cross-region diff fallback construction...\n');
  
  try {
    // Fetch the actual execution data
    const response = await fetch('https://beacon-runner-production.fly.dev/api/v1/jobs/bias-detection-1758981108801/executions/all');
    const data = await response.json();
    
    console.log('📊 Execution Data Summary:');
    console.log(`- Total executions: ${data.executions?.length || 0}`);
    
    if (data.executions?.length > 0) {
      data.executions.forEach((exec, i) => {
        console.log(`- Execution ${i + 1}: ${exec.region} (${exec.status}) - Model: ${exec.output?.metadata?.model || 'unknown'}`);
      });
    }
    
    // Test the fallback construction logic manually
    const executions = data.executions || [];
    
    if (executions.length === 0) {
      console.log('❌ No executions found');
      return;
    }
    
    // Group by region
    const regionGroups = {};
    for (const execution of executions) {
      const region = execution.region || 'unknown';
      if (!regionGroups[region]) regionGroups[region] = [];
      regionGroups[region].push(execution);
    }
    
    console.log('\n📍 Regional Grouping:');
    Object.entries(regionGroups).forEach(([region, execs]) => {
      console.log(`- ${region}: ${execs.length} executions`);
      execs.forEach(exec => {
        const response = exec.output?.response || 'No response';
        const model = exec.output?.metadata?.model || 'unknown';
        console.log(`  - ${exec.id}: ${model} → "${response.substring(0, 50)}${response.length > 50 ? '...' : ''}"`);
      });
    });
    
    // Check if responses are meaningful
    const hasValidResponses = executions.some(exec => {
      const response = exec.output?.response || '';
      return response.length > 10 && !response.includes("I'm sorry, but I can't assist");
    });
    
    console.log(`\n🤖 Response Quality: ${hasValidResponses ? '✅ Valid responses found' : '❌ Only refusal responses'}`);
    
    if (!hasValidResponses) {
      console.log('🚨 ROOT CAUSE: All AI responses are refusals - Modal HF fix needed');
    }
    
  } catch (error) {
    console.error('❌ Error:', error.message);
  }
}

testDiffsFallback().catch(console.error);
