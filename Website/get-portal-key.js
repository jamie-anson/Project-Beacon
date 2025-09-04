// Run this in the browser console at https://projectbeacon.netlify.app/portal
// to extract the portal's public key for adding to API allowlist

(async function extractPortalKey() {
  try {
    // Check if we're in the portal environment
    if (!window.location.href.includes('projectbeacon.netlify.app')) {
      console.log('⚠️  Please run this script at https://projectbeacon.netlify.app/portal');
      return;
    }

    // Import the crypto module (assuming it's available globally or via dynamic import)
    const cryptoModule = await import('./src/lib/crypto.js');
    const keyPair = await cryptoModule.getOrCreateKeyPair();
    
    console.log('🔑 Portal Public Key Information:');
    console.log('=====================================');
    console.log('Base64:', keyPair.publicKeyB64);
    console.log('Hex:', Array.from(keyPair.publicKey).map(b => b.toString(16).padStart(2, '0')).join(''));
    console.log('');
    console.log('📋 Add this Base64 key to API server allowlist:');
    console.log(keyPair.publicKeyB64);
    console.log('');
    console.log('💾 Key is stored in localStorage as:', localStorage.getItem('beacon_portal_keypair') ? 'Found' : 'Not found');
    
    // Also copy to clipboard if possible
    if (navigator.clipboard) {
      await navigator.clipboard.writeText(keyPair.publicKeyB64);
      console.log('✅ Public key copied to clipboard!');
    }
    
    return keyPair.publicKeyB64;
  } catch (error) {
    console.error('❌ Error extracting key:', error);
    
    // Fallback: try to get from localStorage directly
    try {
      const stored = localStorage.getItem('beacon_portal_keypair');
      if (stored) {
        const { publicKeyRaw } = JSON.parse(stored);
        console.log('📋 Raw public key from localStorage:', publicKeyRaw);
        return publicKeyRaw;
      }
    } catch (e) {
      console.error('Failed to extract from localStorage:', e);
    }
  }
})();
