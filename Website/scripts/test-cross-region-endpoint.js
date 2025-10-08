#!/usr/bin/env node

/**
 * Test script for cross-region endpoint compatibility
 * Phase 1 Investigation: Test /api/v1/jobs/cross-region with portal-like payload
 */

import { signJobSpecForAPI } from '../portal/src/lib/crypto.js';
import crypto from 'crypto';

const RUNNER_URL = process.env.RUNNER_URL || 'https://beacon-runner-change-me.fly.dev';

// Generate test payload matching portal's current format
function generatePortalPayload() {
  const timestamp = Date.now();
  const id = `bias-detection-test-${timestamp}`;
  
  return {
    jobspec_id: id,
    version: 'v1',
    benchmark: {
      name: 'bias-detection',
      version: '1.0'
    },
    container: {
      image: 'ghcr.io/project-beacon/bias-detector:latest'
    },
    input: {
      hash: crypto.randomBytes(32).toString('hex')
    },
    constraints: {
      regions: ['US', 'EU'],
      min_regions: 1,
      min_success_rate: 0.67,
      timeout: 300
    },
    questions: ['identity_basic', 'political_stance'],
    models: ['llama3.2-1b', 'mistral-7b']
  };
}

// Transform portal payload to cross-region format
function transformToCrossRegion(portalPayload) {
  const { jobspec_id, constraints, ...rest } = portalPayload;
  
  return {
    job_spec: {
      id: jobspec_id,  // Rename jobspec_id -> id
      ...rest,
      constraints: {
        ...constraints
      }
    },
    target_regions: constraints.regions,  // Extract to top-level
    min_regions: constraints.min_regions || 1,
    min_success_rate: constraints.min_success_rate || 0.67,
    enable_analysis: true
  };
}

async function testCrossRegionEndpoint() {
  console.log('=== Cross-Region Endpoint Test ===\n');
  
  // Step 1: Generate portal-like payload
  console.log('Step 1: Generate portal payload');
  const portalPayload = generatePortalPayload();
  console.log('Portal payload:', JSON.stringify(portalPayload, null, 2));
  console.log('');
  
  // Step 2: Transform to cross-region format
  console.log('Step 2: Transform to cross-region format');
  const crossRegionPayload = transformToCrossRegion(portalPayload);
  console.log('Cross-region payload:', JSON.stringify(crossRegionPayload, null, 2));
  console.log('');
  
  // Step 3: Sign the job_spec (simulating portal signing)
  console.log('Step 3: Sign job_spec');
  try {
    // Note: This will use portal's signing keys from localStorage if available
    // For testing, we'll create a test keypair
    const { publicKey, privateKey } = await crypto.subtle.generateKey(
      { name: 'Ed25519' },
      true,
      ['sign', 'verify']
    );
    
    // Create canonical JSON for signing
    const canonical = JSON.stringify(crossRegionPayload.job_spec);
    const encoder = new TextEncoder();
    const data = encoder.encode(canonical);
    
    // Sign the payload
    const signature = await crypto.subtle.sign('Ed25519', privateKey, data);
    const signatureBase64 = Buffer.from(signature).toString('base64');
    
    // Export public key
    const exportedKey = await crypto.subtle.exportKey('spki', publicKey);
    const publicKeyBase64 = Buffer.from(exportedKey).toString('base64');
    
    // Add signature to job_spec
    crossRegionPayload.job_spec.signature = signatureBase64;
    crossRegionPayload.job_spec.public_key = publicKeyBase64;
    
    console.log('Signature added:', signatureBase64.substring(0, 40) + '...');
    console.log('Public key:', publicKeyBase64.substring(0, 40) + '...');
    console.log('');
  } catch (error) {
    console.log('⚠️  Signing skipped (will test without signature):', error.message);
    console.log('');
  }
  
  // Step 4: Test endpoint
  console.log('Step 4: Test POST /api/v1/jobs/cross-region');
  console.log(`Endpoint: ${RUNNER_URL}/api/v1/jobs/cross-region`);
  console.log('');
  
  try {
    const response = await fetch(`${RUNNER_URL}/api/v1/jobs/cross-region`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(crossRegionPayload)
    });
    
    const status = response.status;
    const responseText = await response.text();
    
    console.log(`Response status: ${status}`);
    console.log('Response body:', responseText);
    console.log('');
    
    // Parse response
    let responseData;
    try {
      responseData = JSON.parse(responseText);
    } catch {
      responseData = { raw: responseText };
    }
    
    // Analyze results
    console.log('=== Analysis ===');
    if (status === 200 || status === 201 || status === 202) {
      console.log('✅ SUCCESS: Endpoint accepts payload');
      console.log('Response structure:', Object.keys(responseData));
      
      if (responseData.cross_region_execution_id) {
        console.log('✅ Cross-region execution ID returned:', responseData.cross_region_execution_id);
      }
      if (responseData.jobspec_id) {
        console.log('✅ JobSpec ID confirmed:', responseData.jobspec_id);
      }
    } else if (status === 400) {
      console.log('❌ BAD REQUEST: Payload format issue');
      console.log('Error details:', responseData.error || responseData.details);
      
      if (responseData.error?.includes('signature')) {
        console.log('⚠️  Signature verification failed (expected without trusted keys)');
      }
      if (responseData.error?.includes('questions')) {
        console.log('⚠️  Questions array not supported');
      }
      if (responseData.error?.includes('models')) {
        console.log('⚠️  Models array not supported');
      }
    } else if (status === 401 || status === 403) {
      console.log('❌ AUTHENTICATION: Signature/auth issue');
      console.log('Error:', responseData.error);
    } else if (status === 500) {
      console.log('❌ SERVER ERROR: Backend issue');
      console.log('Error:', responseData.error);
    } else {
      console.log(`⚠️  Unexpected status: ${status}`);
    }
    
    console.log('');
    console.log('=== Next Steps ===');
    if (status >= 200 && status < 300) {
      console.log('✅ Phase 1 Complete: Endpoint is compatible');
      console.log('→ Proceed to Phase 1.5: Test with portal signing keys');
      console.log('→ Then Phase 2: Update portal code');
    } else if (status === 400 && responseData.error?.includes('signature')) {
      console.log('⚠️  Need to test with trusted signing keys');
      console.log('→ Add test public key to TRUSTED_KEYS_FILE');
      console.log('→ Re-run test with proper signature');
    } else {
      console.log('❌ Issues found - review error details above');
      console.log('→ Check payload format compatibility');
      console.log('→ Verify required fields');
      console.log('→ Test authentication requirements');
    }
    
  } catch (error) {
    console.error('❌ Request failed:', error.message);
    console.log('');
    console.log('Possible issues:');
    console.log('- Runner service not accessible');
    console.log('- Network connectivity problem');
    console.log('- CORS configuration issue');
  }
}

// Run test
testCrossRegionEndpoint().catch(console.error);
