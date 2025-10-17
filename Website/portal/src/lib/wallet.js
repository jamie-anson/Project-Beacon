/**
 * Wallet integration utilities for Project Beacon portal
 * Implements MetaMask connection and wallet-based authentication
 * Compatible with existing Ed25519 signing flow
 */

import { ethers } from 'ethers';
import { exportPublicKey, generateNonce } from './crypto.js';

const HAS_WINDOW = typeof window !== 'undefined';
const TARGET_CHAIN_ID = typeof import.meta !== 'undefined' && import.meta.env && import.meta.env.VITE_WALLET_CHAIN_ID ? String(import.meta.env.VITE_WALLET_CHAIN_ID) : null;
const IS_DEV = typeof import.meta !== 'undefined' && import.meta.env && Boolean(import.meta.env.DEV);

/**
 * Tracks the last detected provider so downstream helpers can adjust behavior (e.g., Brave quirks).
 * @type {{ raw: any, isMetaMask: boolean, isBrave: boolean } | null}
 */
let lastDetectedProvider = null;

function setLastDetectedProvider(provider) {
  if (!provider) {
    lastDetectedProvider = null;
    return;
  }
  lastDetectedProvider = {
    raw: provider,
    isMetaMask: Boolean(provider.isMetaMask),
    isBrave: Boolean(provider.isBraveWallet)
  };
}

function getInjectedEthereumProvider() {
  if (!HAS_WINDOW) return null;
  const { ethereum } = window;
  if (!ethereum) return null;
  const providers = Array.isArray(ethereum.providers) && ethereum.providers.length > 0 ? ethereum.providers : [ethereum];
  const preferMetaMask = providers.find((provider) => provider && provider.isMetaMask);
  if (preferMetaMask) {
    if (IS_DEV) console.log('[Wallet] Using MetaMask provider');
    setLastDetectedProvider(preferMetaMask);
    return preferMetaMask;
  }
  const preferBrave = providers.find((provider) => provider && provider.isBraveWallet);
  if (preferBrave) {
    if (IS_DEV) console.log('[Wallet] Using Brave provider');
    setLastDetectedProvider(preferBrave);
    return preferBrave;
  }
  const fallback = providers[0] || null;
  if (fallback && IS_DEV) console.log('[Wallet] Using fallback provider');
  setLastDetectedProvider(fallback);
  return fallback;
}

async function ensureTargetChain(rawProvider) {
  if (!TARGET_CHAIN_ID || !rawProvider || typeof rawProvider.request !== 'function') return;
  try {
    const currentChainHex = await rawProvider.request({ method: 'eth_chainId' });
    if (typeof currentChainHex === 'string' && currentChainHex.toLowerCase() !== TARGET_CHAIN_ID.toLowerCase()) {
      await rawProvider.request({ method: 'wallet_switchEthereumChain', params: [{ chainId: TARGET_CHAIN_ID }] });
    }
  } catch (error) {
    if (IS_DEV) console.warn('[Wallet] Failed to switch chain', error);
  }
}

async function performPersonalSign(browserProvider, address, message) {
  const normalized = typeof message === 'string' ? message : String(message ?? '');
  const utf8Bytes = ethers.toUtf8Bytes(normalized);
  const hexMessage = ethers.hexlify(utf8Bytes);
  const providerMeta = lastDetectedProvider;
  const target = providerMeta?.raw && typeof providerMeta.raw.request === 'function' ? providerMeta.raw : null;

  const braveAttempts = [
    [hexMessage, address],
    [address, hexMessage],
    [normalized, address],
    [address, normalized]
  ];
  const defaultAttempts = [
    [normalized, address],
    [address, normalized],
    [hexMessage, address],
    [address, hexMessage]
  ];
  const attempts = providerMeta?.isBrave ? braveAttempts : defaultAttempts;

  let lastError = null;
  for (const params of attempts) {
    try {
      const signature = target
        ? await target.request({ method: 'personal_sign', params })
        : await browserProvider.send('personal_sign', params);
      if (typeof signature === 'string' && signature.length > 0) {
        return signature;
      }
    } catch (error) {
      if (error && error.code === 4001) throw error;
      lastError = error;
    }
  }
  if (lastError) throw lastError;
  throw new Error('Failed to sign message with personal_sign');
}

/**
 * Check if MetaMask is installed
 * @returns {boolean}
 */
export function isMetaMaskInstalled() {
  return Boolean(getInjectedEthereumProvider());
}

/**
 * Connect to MetaMask wallet
 * @returns {Promise<{address: string, provider: ethers.BrowserProvider}>}
 */
export async function connectWallet() {
  const ethereumProvider = getInjectedEthereumProvider();
  if (!ethereumProvider) {
    throw new Error('No compatible wallet detected. Please install MetaMask or Brave Wallet to continue.');
  }

  try {
    const accounts = await ethereumProvider.request({ method: 'eth_accounts' });
    if (accounts.length === 0) {
      const requestedAccounts = await ethereumProvider.request({
        method: 'eth_requestAccounts'
      });
      
      if (!requestedAccounts || requestedAccounts.length === 0) {
        throw new Error('No accounts found. Please unlock MetaMask and try again.');
      }
    }

    await ensureTargetChain(ethereumProvider);
    const provider = new ethers.BrowserProvider(ethereumProvider);
    const currentAccounts = await ethereumProvider.request({ method: 'eth_accounts' });
    const address = currentAccounts[0];

    try {
      await provider.getSigner();
    } catch (signerError) {
      throw new Error('Unable to access wallet signer. Please ensure your wallet is unlocked.');
    }

    return { address, provider };
  } catch (error) {
    if (error.code === 4001) {
      throw new Error('Connection request was rejected. Please approve the connection in your wallet to continue.');
    }
    if (error.code === -32002) {
      throw new Error('Connection request is already pending. Please check your wallet and approve the request.');
    }
    if (error.code === -32603) {
      throw new Error('Wallet internal error. Please try refreshing the page and connecting again.');
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
    const address = await signer.getAddress();
    const signature = await performPersonalSign(provider, address, message);
    return signature;
  } catch (error) {
    if (error.code === 4001) {
      throw new Error('Signing request was rejected. Please approve the signature in your wallet to authorize your Ed25519 key.');
    }
    if (error.code === -32603) {
      throw new Error('Wallet internal error during signing. Please try again.');
    }
    if (error.code === -32000) {
      throw new Error('Wallet is locked. Please unlock your wallet and try again.');
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
  let chainIdHex = '0x0';
  try {
    chainIdHex = await provider.send('eth_chainId', []);
  } catch (e) {
    try {
      const net = await provider.getNetwork();
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
  const provider = getInjectedEthereumProvider();
  if (!provider) {
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

  if (typeof provider.on === 'function') {
    provider.on('accountsChanged', handleAccountsChanged);
  }

  return () => {
    if (typeof provider.removeListener === 'function') {
      provider.removeListener('accountsChanged', handleAccountsChanged);
    }
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
