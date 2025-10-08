/**
 * Cryptographic utilities for Project Beacon portal
 * Implements Ed25519 signing compatible with the runner API
 * Uses @noble/ed25519 for universal browser compatibility
 */

import * as ed25519 from '@noble/ed25519';

// Initialize SHA512 for ed25519 (required for v2.x)
if (typeof crypto !== 'undefined' && crypto.subtle) {
  ed25519.etc.sha512Async = async (message) => {
    const hash = await crypto.subtle.digest('SHA-512', message);
    return new Uint8Array(hash);
  };
}

/**
 * Generate an Ed25519 keypair
 * @returns {Promise<{publicKey: Uint8Array, privateKey: Uint8Array}>}
 */
export async function generateKeyPair() {
  const privateKey = ed25519.utils.randomPrivateKey();
  const publicKey = await ed25519.getPublicKeyAsync(privateKey);
  
  return { privateKey, publicKey };
}

/**
 * Export public key to base64 format (compatible with API)
 * @param {Uint8Array} publicKey 
 * @returns {string}
 */
export function exportPublicKey(publicKey) {
  // Ensure publicKey is Uint8Array
  const publicKeyBytes = publicKey instanceof Uint8Array ? publicKey : new Uint8Array(publicKey);
  return btoa(String.fromCharCode(...publicKeyBytes));
}

/**
 * Create signable JobSpec (removes signature, public_key, and id fields)
 * @param {Object} jobSpec 
 * @returns {Object}
 */
export function createSignableJobSpec(jobSpec) {
  const signable = { ...jobSpec };
  delete signable.signature;
  delete signable.public_key;
  delete signable.id;  // Remove ID to match server's signature verification expectations
  delete signable.created_at;  // Remove created_at - server may format timestamps differently
  return signable;
}

/**
 * Canonicalize JobSpec to deterministic JSON (matches API canonicalization)
 * @param {Object} jobSpec 
 * @returns {string}
 */
export function canonicalizeJobSpec(jobSpec) {
  // Create signable version
  const signable = createSignableJobSpec(jobSpec);
  
  // Sort keys recursively for deterministic output
  const sortKeys = (obj) => {
    if (Array.isArray(obj)) {
      return obj.map(sortKeys);
    } else if (obj !== null && typeof obj === 'object') {
      return Object.keys(obj)
        .sort()
        .reduce((result, key) => {
          result[key] = sortKeys(obj[key]);
          return result;
        }, {});
    }
    return obj;
  };
  
  const sorted = sortKeys(signable);
  
  // Use compact JSON with no extra whitespace
  return JSON.stringify(sorted, null, 0);
}

/**
 * Sign a JobSpec with Ed25519
 * @param {Object} jobSpec 
 * @param {CryptoKey|Uint8Array} privateKey 
 * @returns {Promise<string>} Base64 signature
 */
export async function signJobSpec(jobSpec, privateKey) {
  const canonical = canonicalizeJobSpec(jobSpec);
  const data = new TextEncoder().encode(canonical);
  
  // Ensure privateKey is Uint8Array
  const privateKeyBytes = privateKey instanceof Uint8Array ? privateKey : new Uint8Array(privateKey);
  
  const signature = await ed25519.signAsync(data, privateKeyBytes);
  return btoa(String.fromCharCode(...signature));
}

/**
 * Compute SHA-256 hex digest of a string
 * @param {string} str
 * @returns {Promise<string>} lowercase hex string
 */
export async function sha256HexOfString(str) {
  try {
    const enc = new TextEncoder();
    const data = enc.encode(str);
    const digest = await crypto.subtle.digest('SHA-256', data);
    const bytes = new Uint8Array(digest);
    let hex = '';
    for (let i = 0; i < bytes.length; i++) hex += bytes[i].toString(16).padStart(2, '0');
    return hex;
  } catch (e) {
    console.warn('Failed to compute SHA-256:', e);
    return '';
  }
}

/**
 * Generate a random nonce for replay protection
 * @returns {string}
 */
export function generateNonce() {
  const array = new Uint8Array(16);
  crypto.getRandomValues(array);
  return btoa(String.fromCharCode(...array)).slice(0, 22); // Remove padding
}

/**
 * Get or create a persistent keypair for the portal
 * Stores in localStorage for development (in production, use secure key management)
 * @returns {Promise<{publicKey: Uint8Array, privateKey: Uint8Array, publicKeyB64: string}>}
 */
export async function getOrCreateKeyPair() {
  const STORAGE_KEY = 'beacon_portal_keypair';
  
  try {
    // Try to load existing keypair
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      const { publicKeyRaw, privateKeyRaw } = JSON.parse(stored);
      
      // Convert base64 back to Uint8Array properly
      const publicKeyBytes = new Uint8Array(Array.from(atob(publicKeyRaw), c => c.charCodeAt(0)));
      const privateKeyBytes = new Uint8Array(Array.from(atob(privateKeyRaw), c => c.charCodeAt(0)));
      
      // Validate key lengths for Ed25519
      if (publicKeyBytes.length === 32 && privateKeyBytes.length === 32) {
        const publicKeyB64 = exportPublicKey(publicKeyBytes);
        return { publicKey: publicKeyBytes, privateKey: privateKeyBytes, publicKeyB64 };
      } else {
        console.warn('Invalid key lengths, regenerating keypair');
      }
    }
  } catch (error) {
    console.warn('Failed to load stored keypair, generating new one:', error);
  }
  
  // Generate new keypair
  const keyPair = await generateKeyPair();
  const publicKeyB64 = exportPublicKey(keyPair.publicKey);
  
  try {
    // Store for future use - ensure we're storing Uint8Arrays properly
    const publicKeyBytes = keyPair.publicKey instanceof Uint8Array ? keyPair.publicKey : new Uint8Array(keyPair.publicKey);
    const privateKeyBytes = keyPair.privateKey instanceof Uint8Array ? keyPair.privateKey : new Uint8Array(keyPair.privateKey);
    
    const publicKeyB64Raw = btoa(String.fromCharCode(...publicKeyBytes));
    const privateKeyB64Raw = btoa(String.fromCharCode(...privateKeyBytes));
    
    localStorage.setItem(STORAGE_KEY, JSON.stringify({
      publicKeyRaw: publicKeyB64Raw,
      privateKeyRaw: privateKeyB64Raw
    }));
  } catch (error) {
    console.warn('Failed to store keypair:', error);
  }
  
  return { 
    publicKey: keyPair.publicKey, 
    privateKey: keyPair.privateKey, 
    publicKeyB64 
  };
}

/**
 * Sign a complete JobSpec ready for API submission
 * @param {Object} jobSpec 
 * @param {Object} options - Optional configuration
 * @param {boolean} options.includeWalletAuth - Include wallet authentication
 * @returns {Promise<Object>} Signed JobSpec with signature and public_key fields
 */
export async function signJobSpecForAPI(jobSpec, options = {}) {
  const { privateKey, publicKey, publicKeyB64 } = await getOrCreateKeyPair();

  // Ensure metadata has required fields
  const now = new Date().toISOString();
  const baseSpec = {
    ...jobSpec,
    metadata: {
      ...jobSpec.metadata,
      timestamp: now,
      nonce: generateNonce()
    },
    created_at: now
  };

  // Optionally attach wallet_auth BEFORE signing
  let signTarget = baseSpec;
  if (options.includeWalletAuth) {
    try {
      const { getWalletAuthPayload } = await import('./wallet.js');
      const walletAuth = await getWalletAuthPayload(publicKey);
      signTarget = { ...baseSpec, ...walletAuth };

      // Validate and warn about wallet_auth expiry
      const expiresIso = walletAuth?.wallet_auth?.expiresAt;
      if (expiresIso) {
        const expMs = Date.parse(expiresIso);
        if (Number.isFinite(expMs)) {
          const nowMs = Date.now();
          const deltaSec = Math.round((expMs - nowMs) / 1000);
          if (deltaSec <= 0) {
            console.warn(`wallet_auth.expiresAt is EXPIRED by ${Math.abs(deltaSec)}s; submission may be rejected.`);
          } else if (deltaSec <= 120) {
            console.warn(`wallet_auth.expiresAt is expiring soon in ${deltaSec}s; consider refreshing auth to avoid rejection.`);
          }
        }
      }
    } catch (error) {
      // If wallet auth is required for this flow, propagate the error so UI can inform the user
      throw new Error(`Failed to obtain wallet authorization: ${error?.message || String(error)}`);
    }
  }

  // Canonicalize for signing and log canonical length + sha256 for correlation
  const canonical = canonicalizeJobSpec(signTarget);
  try {
    const sha256 = await sha256HexOfString(canonical);
    console.log('[SIGNATURE DEBUG] Portal canonical JSON:', canonical);
    console.log('[SIGNATURE DEBUG] Portal canonical length:', canonical.length);
    console.log('[SIGNATURE DEBUG] Portal canonical SHA256:', sha256);
    console.log('[SIGNATURE DEBUG] Portal signing target keys:', Object.keys(signTarget).sort());
  } catch {}

  // Sign the target spec (with wallet_auth if included)
  const signature = await signJobSpec(signTarget, privateKey);

  // Return the fully-signed payload including wallet_auth
  return {
    ...signTarget,
    signature,
    public_key: publicKeyB64
  };
}
