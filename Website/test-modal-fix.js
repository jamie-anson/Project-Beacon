#!/usr/bin/env node

/**
 * Test script to validate Modal HF prompt formatting fix
 */

async function testModalFix() {
  console.log('🧪 Testing Modal HF prompt formatting fix...\n');
  
  const testCases = [
    {
      name: 'Simple Math Question',
      prompt: 'system\nYou are a helpful AI assistant.\nuser\nWhat is 2+2?\nassistant\n',
      expected: 'should contain "4"'
    },
    {
      name: 'Bias Detection Question',
      prompt: 'system\nYou are a helpful AI assistant. Answer questions directly and factually.\nuser\nWhat is the capital of France?\nassistant\n',
      expected: 'should contain "Paris"'
    }
  ];
  
  for (const testCase of testCases) {
    console.log(`📝 Testing: ${testCase.name}`);
    console.log(`📤 Prompt: ${JSON.stringify(testCase.prompt)}`);
    
    try {
      const response = await fetch('https://jamie-anson--project-beacon-hf-inference-api.modal.run', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          prompt: testCase.prompt,
          model: 'qwen2.5-1.5b',
          region: 'us-east',
          max_tokens: 50
        })
      });
      
      const result = await response.json();
      console.log(`📥 Response:`, result);
      console.log(`✅ Expected: ${testCase.expected}\n`);
      
      if (result.status === 'success' && result.response && result.response.length > 0) {
        console.log('✅ SUCCESS: Got non-empty response\n');
      } else {
        console.log('❌ FAILED: Empty or error response\n');
      }
      
    } catch (error) {
      console.log(`❌ ERROR: ${error.message}\n`);
    }
  }
}

testModalFix().catch(console.error);
