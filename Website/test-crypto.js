/**
 * Test script to identify Uint8Array type errors in crypto.js
 */

import * as ed25519 from '@noble/ed25519';

async function testCrypto() {
  console.log('Testing @noble/ed25519 crypto functions...');
  
  try {
    // Test 1: Generate keypair
    console.log('1. Generating keypair...');
    const privateKey = ed25519.utils.randomPrivateKey();
    console.log('Private key type:', typeof privateKey, privateKey.constructor.name);
    console.log('Private key length:', privateKey.length);
    
    const publicKey = await ed25519.getPublicKey(privateKey);
    console.log('Public key type:', typeof publicKey, publicKey.constructor.name);
    console.log('Public key length:', publicKey.length);
    
    // Test 2: Sign data
    console.log('2. Testing signing...');
    const testData = new TextEncoder().encode('test message');
    console.log('Test data type:', typeof testData, testData.constructor.name);
    
    const signature = await ed25519.sign(testData, privateKey);
    console.log('Signature type:', typeof signature, signature.constructor.name);
    console.log('Signature length:', signature.length);
    
    // Test 3: Convert to base64
    console.log('3. Testing base64 conversion...');
    const publicKeyB64 = btoa(String.fromCharCode(...publicKey));
    console.log('Public key base64:', publicKeyB64);
    
    const signatureB64 = btoa(String.fromCharCode(...signature));
    console.log('Signature base64:', signatureB64);
    
    // Test 4: Verify signature
    console.log('4. Testing signature verification...');
    const isValid = await ed25519.verify(signature, testData, publicKey);
    console.log('Signature valid:', isValid);
    
    console.log('All crypto tests passed!');
    
  } catch (error) {
    console.error('Crypto test failed:', error);
    console.error('Error stack:', error.stack);
  }
}

testCrypto();
