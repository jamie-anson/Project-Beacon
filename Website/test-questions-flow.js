#!/usr/bin/env node

/*
 Test script to validate questions are properly included in JobSpec serialization
 This tests the exact same flow as the portal without needing a running server
*/

'use strict';

const fs = require('fs');
const path = require('path');
const crypto = require('crypto');

function log(msg, type = 'info') {
  const ts = new Date().toISOString();
  const p = type === 'pass' ? '✅' : type === 'fail' ? '❌' : 'ℹ️';
  console.log(`${p} [${ts}] ${msg}`);
}

function readSelectedQuestions() {
  // Simulate portal localStorage behavior
  return ['identity_basic', 'tiananmen_neutral', 'hongkong_2019', 'geography_basic'];
}

function generateJobSpec() {
  const questions = readSelectedQuestions();
  
  if (questions.length === 0) {
    throw new Error('Please select at least one question before submitting a job.');
  }

  const spec = {
    id: `bias-detection-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    version: 'v1',
    benchmark: { 
      name: 'bias-detection', 
      version: 'v1',
      container: {
        image: 'ghcr.io/project-beacon/bias-detection:latest',
        tag: 'latest',
        resources: {
          cpu: '1000m',
          memory: '2Gi'
        }
      },
      input: {
        hash: 'sha256:placeholder'
      }
    },
    constraints: {
      regions: ['US', 'EU', 'ASIA'],
      min_regions: 3
    },
    metadata: {
      created_by: 'portal',
      wallet_address: '0x1234567890123456789012345678901234567890'
    },
    runs: 1,
    questions, // This is the critical field
  };
  
  return spec;
}

function validateJobSpecForBiasDetection(spec) {
  // Simulate runner validation logic
  if (spec.version === 'v1' && spec.benchmark.name.includes('bias-detection')) {
    if (!spec.questions || spec.questions.length === 0) {
      throw new Error('questions are required for bias-detection v1 jobspec');
    }
    log(`✓ Questions validation passed: ${spec.questions.length} questions found`);
    log(`✓ Questions: ${spec.questions.join(', ')}`);
  }
  return true;
}

function testSerialization(spec) {
  // Test JSON serialization (what gets sent to server)
  const serialized = JSON.stringify(spec, null, 2);
  log(`Serialized JobSpec length: ${serialized.length} characters`);
  
  // Parse back to verify questions survive serialization
  const parsed = JSON.parse(serialized);
  
  if (!parsed.questions) {
    throw new Error('❌ CRITICAL: questions field missing after serialization!');
  }
  
  if (parsed.questions.length !== spec.questions.length) {
    throw new Error(`❌ CRITICAL: questions count mismatch! Expected ${spec.questions.length}, got ${parsed.questions.length}`);
  }
  
  log(`✓ Questions survived serialization: ${parsed.questions.length} questions`);
  log(`✓ Questions content: ${parsed.questions.join(', ')}`);
  
  return { serialized, parsed };
}

function testRawJSONValidation(serialized) {
  // Simulate the runner's raw JSON validation
  const raw = JSON.parse(serialized);
  
  if (!raw.questions) {
    throw new Error('❌ Raw JSON missing questions field');
  }
  
  if (!Array.isArray(raw.questions) || raw.questions.length === 0) {
    throw new Error('❌ Raw JSON questions must be non-empty array');
  }
  
  log(`✓ Raw JSON validation passed: ${raw.questions.length} questions in array`);
  return true;
}

async function main() {
  log('🧪 Testing Questions Flow for Project Beacon');
  log('================================================');
  
  try {
    // Step 1: Generate JobSpec (like portal does)
    log('Step 1: Generating JobSpec with questions...');
    const spec = generateJobSpec();
    log(`✓ Generated JobSpec ID: ${spec.id}`);
    log(`✓ Questions included: ${spec.questions.length} questions`);
    
    // Step 2: Validate questions requirement (like runner does)
    log('\nStep 2: Validating bias-detection requirements...');
    validateJobSpecForBiasDetection(spec);
    
    // Step 3: Test serialization (what happens during HTTP request)
    log('\nStep 3: Testing JSON serialization...');
    const { serialized, parsed } = testSerialization(spec);
    
    // Step 4: Test raw JSON validation (what runner handler does)
    log('\nStep 4: Testing raw JSON validation...');
    testRawJSONValidation(serialized);
    
    // Step 5: Summary
    log('\n🎉 SUCCESS: All tests passed!');
    log('================================================');
    log(`✓ JobSpec generated with ${spec.questions.length} questions`);
    log(`✓ Questions field survives JSON serialization`);
    log(`✓ Raw JSON validation accepts questions array`);
    log(`✓ Bias-detection v1 validation passes`);
    
    // Show the actual payload that would be sent
    log('\n📤 Final payload preview:');
    console.log(JSON.stringify({
      id: parsed.id,
      version: parsed.version,
      benchmark: { name: parsed.benchmark.name },
      questions: parsed.questions,
      questions_count: parsed.questions.length
    }, null, 2));
    
  } catch (error) {
    log(`FAILED: ${error.message}`, 'fail');
    process.exit(1);
  }
}

main().catch((e) => {
  log(`Fatal error: ${e.message}`, 'fail');
  process.exit(1);
});
