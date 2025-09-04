/**
 * Wallet integration utilities for Project Beacon portal
 * Implements MetaMask connection and wallet-based authentication
 * Compatible with existing Ed25519 signing flow
 */

import { ethers } from 'ethers';
import { exportPublicKey, generateNonce } from './crypto.js';

/**
 * Check if MetaMask is installed
 * @returns {boolean}
 */
export function isMetaMaskInstalled() {
  return typeof window !== 'undefined' && 
         typeof window.ethereum !== 'undefined' && 
         window.ethereum.isMetaMask;
}

/**
 * Connect to MetaMask wallet
 * @returns {Promise<{address: string, provider: ethers.BrowserProvider}>}
 */
export async function connectWallet() {
  if (!isMetaMaskInstalled()) {
    throw new Error('MetaMask is not installed. Please install MetaMask from https://metamask.io to continue.');
  }

  try {
    // Check if MetaMask is locked
    const accounts = await window.ethereum.request({ method: 'eth_accounts' });
    if (accounts.length === 0) {
      // Request account access if no accounts are available
      const requestedAccounts = await window.ethereum.request({
        method: 'eth_requestAccounts'
      });
      
      if (!requestedAccounts || requestedAccounts.length === 0) {
        throw new Error('No accounts found. Please unlock MetaMask and try again.');
      }
    }

    const provider = new ethers.BrowserProvider(window.ethereum);
    const currentAccounts = await window.ethereum.request({ method: 'eth_accounts' });
    const address = currentAccounts[0];

    // Verify we can create a signer
    try {
      await provider.getSigner();
    } catch (signerError) {
      throw new Error('Unable to access wallet signer. Please ensure MetaMask is unlocked.');
    }

    return { address, provider };
  } catch (error) {
    if (error.code === 4001) {
      throw new Error('Connection request was rejected. Please approve the connection in MetaMask to continue.');
    }
    if (error.code === -32002) {
      throw new Error('Connection request is already pending. Please check MetaMask and approve the request.');
    }
    if (error.code === -32603) {
      throw new Error('MetaMask internal error. Please try refreshing the page and connecting again.');
    }
    throw new Error(`Failed to connect wallet: ${error.message}`);
  }
}

/**
 * Sign a message with the connected wallet
 * @param {ethers.BrowserProvider} provider 
 * @param {string} message 
 * @returns {Promise<string>} Signature
 */
export async function signMessage(provider, message) {
  try {
    const signer = await provider.getSigner();
    const signature = await signer.signMessage(message);
    return signature;
  } catch (error) {
    if (error.code === 4001) {
      throw new Error('Signing request was rejected. Please approve the signature in MetaMask to authorize your Ed25519 key.');
    }
    if (error.code === -32603) {
      throw new Error('MetaMask internal error during signing. Please try again.');
    }
    if (error.code === -32000) {
      throw new Error('MetaMask is locked. Please unlock your wallet and try again.');
    }
    throw new Error(`Failed to sign message: ${error.message}`);
  }
}

/**
 * Create authorization message for Ed25519 key
 * @param {string} ed25519PublicKey Base64 encoded Ed25519 public key
 * @returns {string}
 */
export function createAuthMessage(ed25519PublicKey) {
  return `Authorize Project Beacon key: ${ed25519PublicKey}`;
}

/**
 * Get or create wallet authorization
 * @param {Uint8Array} ed25519PublicKey 
 * @returns {Promise<{address: string, signature: string, message: string, ed25519Key: string, timestamp: number}>}
 */
export async function getOrCreateWalletAuth(ed25519PublicKey) {
  const STORAGE_KEY = 'beacon:wallet_auth';
  const ed25519KeyB64 = exportPublicKey(ed25519PublicKey);
  
  try {
    // Check for existing valid authorization
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      const auth = JSON.parse(stored);
      
      // Validate stored auth (check if it's for the same Ed25519 key and not too old)
      const maxAge = 7 * 24 * 60 * 60 * 1000; // 7 days
      const isValid = auth.ed25519Key === ed25519KeyB64 && 
                     auth.timestamp && 
                     (Date.now() - auth.timestamp) < maxAge;
      
      if (isValid && auth.address && auth.signature && auth.message) {
        return auth;
      }
    }
  } catch (error) {
    console.warn('Failed to load stored wallet auth:', error);
  }

  // Create new authorization
  const { address, provider } = await connectWallet();
  // Resolve chainId from provider/ethereum (prefer direct RPC for reliability)
  let chainIdHex = '0x0';
  try {
    chainIdHex = await window.ethereum.request({ method: 'eth_chainId' });
  } catch (e) {
    try {
      const net = await provider.getNetwork();
      // ethers v6 returns bigint
      chainIdHex = '0x' + (Number(net?.chainId || 0)).toString(16);
    } catch {}
  }
  const chainId = (() => {
    try {
      return parseInt(String(chainIdHex), 16) || 0;
    } catch { return 0; }
  })();
  const message = createAuthMessage(ed25519KeyB64);
  const signature = await signMessage(provider, message);
  const nonce = generateNonce();
  const nowMs = Date.now();
  const maxAgeMs = 7 * 24 * 60 * 60 * 1000; // 7 days
  const expiresAt = nowMs + maxAgeMs;
  
  const auth = {
    address,
    signature,
    message,
    ed25519Key: ed25519KeyB64,
    timestamp: nowMs,
    chainId,
    nonce,
    expiresAt
  };

  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(auth));
  } catch (error) {
    console.warn('Failed to store wallet auth:', error);
  }

  return auth;
}

/**
 * Get current wallet authorization status
 * @returns {{isAuthorized: boolean, address?: string, ed25519Key?: string, timestamp?: number}}
 */
export function getWalletAuthStatus() {
  const STORAGE_KEY = 'beacon:wallet_auth';
  
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (!stored) {
      return { isAuthorized: false };
    }

    const auth = JSON.parse(stored);
    const maxAge = 7 * 24 * 60 * 60 * 1000; // 7 days
    const isValid = auth.timestamp && (Date.now() - auth.timestamp) < maxAge;

    if (!isValid) {
      localStorage.removeItem(STORAGE_KEY);
      return { isAuthorized: false };
    }

    return {
      isAuthorized: true,
      address: auth.address,
      ed25519Key: auth.ed25519Key,
      timestamp: auth.timestamp
    };
  } catch (error) {
    console.warn('Failed to check wallet auth status:', error);
    return { isAuthorized: false };
  }
}

/**
 * Clear wallet authorization
 */
export function clearWalletAuth() {
  const STORAGE_KEY = 'beacon:wallet_auth';
  try {
    localStorage.removeItem(STORAGE_KEY);
  } catch (error) {
    console.warn('Failed to clear wallet auth:', error);
  }
}

/**
 * Listen for MetaMask account changes
 * @param {function} callback Called when accounts change
 * @returns {function} Cleanup function
 */
export function onAccountsChanged(callback) {
  if (!isMetaMaskInstalled()) {
    return () => {};
  }

  const handleAccountsChanged = (accounts) => {
    if (accounts.length === 0) {
      // User disconnected
      clearWalletAuth();
      callback(null);
    } else {
      // Account changed - clear auth to force re-authorization
      clearWalletAuth();
      callback(accounts[0]);
    }
  };

  window.ethereum.on('accountsChanged', handleAccountsChanged);

  return () => {
    window.ethereum.removeListener('accountsChanged', handleAccountsChanged);
  };
}

/**
 * Get wallet auth payload for API submission
 * @param {Uint8Array} ed25519PublicKey 
 * @returns {Promise<{wallet_auth: {address: string, signature: string, message: string}}>}
 */
export async function getWalletAuthPayload(ed25519PublicKey) {
  const auth = await getOrCreateWalletAuth(ed25519PublicKey);

  // Normalize expiresAt to RFC3339 string as required by runner API
  let expiresAtIso;
  if (auth.expiresAt != null) {
    if (typeof auth.expiresAt === 'number') {
      // Stored as epoch millis
      expiresAtIso = new Date(auth.expiresAt).toISOString();
    } else if (typeof auth.expiresAt === 'string') {
      // Assume already RFC3339 or ISO-like
      expiresAtIso = auth.expiresAt;
    }
  }

  const payload = {
    wallet_auth: {
      address: auth.address,
      signature: auth.signature,
      message: auth.message,
      chainId: auth.chainId,
      nonce: auth.nonce
    }
  };
  if (expiresAtIso) payload.wallet_auth.expiresAt = expiresAtIso;

  return payload;
}
