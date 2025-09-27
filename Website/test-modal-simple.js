#!/usr/bin/env node

/**
 * Simple test for Modal HF fix with minimal prompt
 */

async function testModalSimple() {
  console.log('🧪 Testing Modal HF with simple prompt...\n');
  
  const testCases = [
    {
      name: 'Direct Question (No Chat Format)',
      prompt: 'What is 2+2?',
      expected: 'Should contain "4"'
    },
    {
      name: 'Simple Chat Format',
      prompt: 'user\nWhat is 2+2?\nassistant\n',
      expected: 'Should contain "4"'
    }
  ];
  
  for (const testCase of testCases) {
    console.log(`📝 Testing: ${testCase.name}`);
    console.log(`📤 Prompt: "${testCase.prompt}"`);
    
    try {
      const response = await fetch('https://jamie-anson--project-beacon-hf-inference-api.modal.run', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          prompt: testCase.prompt,
          model: 'qwen2.5-1.5b',
          region: 'us-east',
          max_tokens: 20,
          temperature: 0.1
        })
      });
      
      const result = await response.json();
      console.log(`📥 Status: ${result.status}`);
      console.log(`📥 Response: "${result.response}"`);
      console.log(`📥 Tokens: ${result.tokens_generated}`);
      console.log(`📥 Time: ${result.inference_time?.toFixed(2)}s`);
      console.log(`✅ Expected: ${testCase.expected}\n`);
      
    } catch (error) {
      console.log(`❌ ERROR: ${error.message}\n`);
    }
  }
}

testModalSimple().catch(console.error);
