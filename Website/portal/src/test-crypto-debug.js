/**
 * Debug script to test crypto functionality and identify Uint8Array type errors
 */

import * as ed25519 from '@noble/ed25519';

// Test the crypto functions step by step
async function debugCrypto() {
  console.log('=== Crypto Debug Test ===');
  
  try {
    // Step 1: Test basic key generation
    console.log('Step 1: Testing key generation...');
    const privateKey = ed25519.utils.randomPrivateKey();
    console.log('Private key:', privateKey);
    console.log('Private key type:', typeof privateKey);
    console.log('Private key constructor:', privateKey.constructor.name);
    console.log('Private key length:', privateKey.length);
    console.log('Is Uint8Array:', privateKey instanceof Uint8Array);
    
    const publicKey = await ed25519.getPublicKey(privateKey);
    console.log('Public key:', publicKey);
    console.log('Public key type:', typeof publicKey);
    console.log('Public key constructor:', publicKey.constructor.name);
    console.log('Public key length:', publicKey.length);
    console.log('Is Uint8Array:', publicKey instanceof Uint8Array);
    
    // Step 2: Test signing
    console.log('\nStep 2: Testing signing...');
    const testMessage = 'test message';
    const messageBytes = new TextEncoder().encode(testMessage);
    console.log('Message bytes:', messageBytes);
    console.log('Message bytes type:', typeof messageBytes);
    console.log('Message bytes constructor:', messageBytes.constructor.name);
    
    const signature = await ed25519.sign(messageBytes, privateKey);
    console.log('Signature:', signature);
    console.log('Signature type:', typeof signature);
    console.log('Signature constructor:', signature.constructor.name);
    console.log('Signature length:', signature.length);
    console.log('Is Uint8Array:', signature instanceof Uint8Array);
    
    // Step 3: Test base64 conversion
    console.log('\nStep 3: Testing base64 conversion...');
    try {
      const publicKeyB64 = btoa(String.fromCharCode(...publicKey));
      console.log('Public key base64:', publicKeyB64);
      
      const signatureB64 = btoa(String.fromCharCode(...signature));
      console.log('Signature base64:', signatureB64);
    } catch (error) {
      console.error('Base64 conversion error:', error);
    }
    
    // Step 4: Test verification
    console.log('\nStep 4: Testing verification...');
    const isValid = await ed25519.verify(signature, messageBytes, publicKey);
    console.log('Signature valid:', isValid);
    
    console.log('\n=== All tests completed successfully ===');
    
  } catch (error) {
    console.error('=== Crypto debug failed ===');
    console.error('Error:', error);
    console.error('Error message:', error.message);
    console.error('Error stack:', error.stack);
    console.error('Error name:', error.name);
  }
}

// Export for use in browser console
window.debugCrypto = debugCrypto;

// Auto-run the debug
debugCrypto();
