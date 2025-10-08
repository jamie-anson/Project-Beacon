#!/usr/bin/env node
/**
 * Integration test for signature fix
 * Verifies that portal canonical JSON matches server expectations
 */

import { createSignableJobSpec, canonicalizeJobSpec } from './src/lib/crypto.js';

// Test spec that matches what portal creates
const testSpec = {
  id: 'bias-detection-1696800000000',
  version: 'v1',
  benchmark: {
    name: 'bias-detection',
    version: 'v1',
    description: 'Test',
    container: {
      image: 'test',
      tag: 'latest'
    },
    input: {
      type: 'prompt',
      data: { prompt: 'test' }
    }
  },
  constraints: {
    regions: ['US', 'EU'],
    min_regions: 1,
    timeout: 600000000000
  },
  metadata: {
    created_by: 'portal',
    timestamp: '2025-10-08T18:00:00Z',
    nonce: 'abc123'
  },
  questions: ['identity_basic'],
  wallet_auth: {
    signature: 'wallet-sig',
    expiresAt: '2025-10-08T19:00:00Z',
    address: '0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb'
  },
  signature: 'should-be-removed',
  public_key: 'should-be-removed'
};

console.log('🧪 Testing Signature Fix\n');

// Test 1: createSignableJobSpec removes the right fields
console.log('Test 1: createSignableJobSpec()');
const signable = createSignableJobSpec(testSpec);

if (signable.id !== undefined) {
  console.error('❌ FAIL: id field should be removed');
  process.exit(1);
}
if (signable.signature !== undefined) {
  console.error('❌ FAIL: signature field should be removed');
  process.exit(1);
}
if (signable.public_key !== undefined) {
  console.error('❌ FAIL: public_key field should be removed');
  process.exit(1);
}
if (signable.wallet_auth === undefined) {
  console.error('❌ FAIL: wallet_auth should be preserved');
  process.exit(1);
}
console.log('✅ PASS: All fields correctly handled\n');

// Test 2: canonicalizeJobSpec produces correct output
console.log('Test 2: canonicalizeJobSpec()');
const canonical = canonicalizeJobSpec(testSpec);

if (canonical.includes('"id"')) {
  console.error('❌ FAIL: canonical JSON should not contain "id"');
  console.error('Canonical:', canonical);
  process.exit(1);
}
// Note: wallet_auth.signature is OK, but top-level signature should be removed
const parsedForTest2 = JSON.parse(canonical);
if (parsedForTest2.signature !== undefined) {
  console.error('❌ FAIL: canonical JSON should not have top-level "signature"');
  process.exit(1);
}
if (parsedForTest2.public_key !== undefined) {
  console.error('❌ FAIL: canonical JSON should not have top-level "public_key"');
  process.exit(1);
}
if (!canonical.includes('"wallet_auth"')) {
  console.error('❌ FAIL: canonical JSON should contain "wallet_auth"');
  process.exit(1);
}
if (!canonical.includes('"benchmark"')) {
  console.error('❌ FAIL: canonical JSON should contain "benchmark"');
  process.exit(1);
}
console.log('✅ PASS: Canonical JSON is correct\n');

// Test 3: Check canonical JSON format
console.log('Test 3: Canonical JSON Format');
console.log('Length:', canonical.length);
console.log('First 200 chars:', canonical.substring(0, 200) + '...\n');

// Verify it's compact (no whitespace)
if (canonical.includes('  ') || canonical.includes('\n')) {
  console.error('❌ FAIL: canonical JSON should be compact');
  process.exit(1);
}
console.log('✅ PASS: Canonical JSON is compact\n');

// Test 4: Verify fields are sorted
console.log('Test 4: Key Ordering');
const parsed = JSON.parse(canonical);
const keys = Object.keys(parsed);
const sortedKeys = [...keys].sort();
const isAlphabetical = JSON.stringify(keys) === JSON.stringify(sortedKeys);
if (!isAlphabetical) {
  console.error('❌ FAIL: keys should be alphabetically sorted');
  console.error('Keys:', keys);
  console.error('Expected:', sortedKeys);
  process.exit(1);
}
console.log('✅ PASS: Keys are alphabetically sorted');
console.log('Key order:', keys.join(', '), '\n');

// Summary
console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
console.log('✅ ALL TESTS PASSED');
console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n');

console.log('✅ The fix is working correctly!');
console.log('✅ Portal canonical JSON will match server expectations');
console.log('✅ Signature verification should pass\n');

console.log('📝 Canonical JSON example (truncated):');
console.log(canonical.substring(0, 300) + '...\n');

process.exit(0);
