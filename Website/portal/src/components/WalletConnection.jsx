import React, { useState, useEffect } from 'react';
import {
  isMetaMaskInstalled,
  getWalletAuthStatus,
  getOrCreateWalletAuth,
  clearWalletAuth,
  onAccountsChanged
} from '../lib/wallet.js';
import { getOrCreateKeyPair } from '../lib/crypto.js';
import { useToast } from '../state/toast.jsx';
import { createErrorToast, createSuccessToast, createWarningToast } from '../lib/errorUtils.js';

export default function WalletConnection({ overrides = {} }) {
  const isMetaMaskInstalledImpl = overrides.isMetaMaskInstalled ?? isMetaMaskInstalled;
  const getWalletAuthStatusImpl = overrides.getWalletAuthStatus ?? getWalletAuthStatus;
  const getOrCreateWalletAuthImpl = overrides.getOrCreateWalletAuth ?? getOrCreateWalletAuth;
  const clearWalletAuthImpl = overrides.clearWalletAuth ?? clearWalletAuth;
  const onAccountsChangedImpl = overrides.onAccountsChanged ?? onAccountsChanged;
  const getOrCreateKeyPairImpl = overrides.getOrCreateKeyPair ?? getOrCreateKeyPair;
  const [walletStatus, setWalletStatus] = useState({ isAuthorized: false });
  const [isConnecting, setIsConnecting] = useState(false);
  const [showDetails, setShowDetails] = useState(false);
  const { add: addToast } = useToast();

  useEffect(() => {
    // Check initial wallet status
    updateWalletStatus();

    // Listen for account changes
    const cleanup = onAccountsChangedImpl((newAccount) => {
      updateWalletStatus();
      if (newAccount === null) {
        addToast(createWarningToast('Wallet disconnected. Please reconnect to submit jobs.'));
      } else {
        addToast(createWarningToast('Wallet account changed. Please re-authorize to submit jobs.'));
      }
    });

    return cleanup;
  }, [addToast, onAccountsChangedImpl]);

  const updateWalletStatus = () => {
    const status = getWalletAuthStatusImpl();
    setWalletStatus(status);
  };

  const handleConnectWallet = async () => {
    if (!isMetaMaskInstalledImpl()) {
      addToast(createErrorToast(new Error('MetaMask is not installed. Please install MetaMask from https://metamask.io')));
      return;
    }

    setIsConnecting(true);
    try {
      // Get the current Ed25519 keypair
      const { publicKey } = await getOrCreateKeyPairImpl();
      
      // Create wallet authorization
      const auth = await getOrCreateWalletAuthImpl(publicKey);
      
      updateWalletStatus();
      addToast(createSuccessToast(auth.address, 'wallet connected'));
    } catch (error) {
      console.error('Wallet connection failed:', error);
      addToast(createErrorToast(error));
    } finally {
      setIsConnecting(false);
    }
  };

  const handleDisconnect = () => {
    clearWalletAuthImpl();
    updateWalletStatus();
    addToast(createSuccessToast('Wallet disconnected'));
  };

  const truncateAddress = (address) => {
    if (!address) return '';
    return `${address.slice(0, 6)}...${address.slice(-4)}`;
  };

  const formatTimestamp = (timestamp) => {
    if (!timestamp) return '';
    const date = new Date(timestamp);
    return date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
  };

  if (!isMetaMaskInstalledImpl()) {
    const isSafari = (() => {
      if (typeof navigator === 'undefined' || !navigator.userAgent) return false;
      const ua = navigator.userAgent;
      const isSafariLike = /safari/i.test(ua) && !/chrome|chromium|crios|android/i.test(ua);
      return isSafariLike;
    })();
    return (
      <div className="bg-amber-900/20 border border-amber-700 rounded-lg p-4">
        <div className="flex items-start gap-3">
          <div className="flex-shrink-0">
            <svg className="h-5 w-5 text-amber-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z" />
            </svg>
          </div>
          <div className="flex-1">
            <h3 className="text-sm font-medium text-amber-400">Wallet required for authorization</h3>
            {isSafari ? (
              <p className="text-sm text-amber-300 mt-1">
                Safari doesn’t support the MetaMask extension. Please open this page in Chrome or Brave to continue.
              </p>
            ) : (
              <p className="text-sm text-amber-300 mt-1">
                It looks like a crypto wallet isn’t available in this browser. To continue, please use a browser with wallet support (Chrome or Brave).
              </p>
            )}
            <div className="mt-2 flex flex-wrap gap-3 text-sm">
              <a
                href="https://metamask.io"
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center px-3 py-1.5 border border-amber-300 rounded bg-white text-amber-900 hover:bg-amber-100"
              >
                Download MetaMask
              </a>
              <a
                href="/WALLET-INTEGRATION.md"
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center px-3 py-1.5 text-amber-300 underline decoration-dotted hover:text-amber-200"
              >
                Why do I need a wallet?
              </a>
              <button
                type="button"
                onClick={async () => {
                  try {
                    const url = window.location?.href || '';
                    if (navigator?.clipboard?.writeText) {
                      await navigator.clipboard.writeText(url);
                    }
                    addToast(createSuccessToast('Page link copied'));
                  } catch (e) {
                    console.warn('Copy failed', e);
                  }
                }}
                className="inline-flex items-center px-3 py-1.5 border border-amber-300 rounded bg-white text-amber-900 hover:bg-amber-100"
              >
                Copy link for Chrome/Brave
              </button>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="flex-shrink-0">
            <svg className="h-5 w-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z" />
            </svg>
          </div>
          <div>
            <h3 className="text-sm font-medium text-gray-100">Wallet Authentication</h3>
            {walletStatus.isAuthorized ? (
              <p className="text-sm text-green-400">
                Connected: {truncateAddress(walletStatus.address)}
              </p>
            ) : (
              <p className="text-sm text-gray-300">
                Connect your wallet to authorize job submissions
              </p>
            )}
          </div>
        </div>

        <div className="flex items-center gap-2">
          {walletStatus.isAuthorized && (
            <button
              onClick={() => setShowDetails(!showDetails)}
              className="text-xs text-gray-400 hover:text-gray-200 px-2 py-1 border border-gray-600 rounded"
            >
              {showDetails ? 'Hide' : 'Details'}
            </button>
          )}
          
          {walletStatus.isAuthorized ? (
            <button
              onClick={handleDisconnect}
              className="px-3 py-1.5 text-sm border border-red-600 text-red-400 rounded hover:bg-red-900/20"
            >
              Disconnect
            </button>
          ) : (
            <button
              onClick={handleConnectWallet}
              disabled={isConnecting}
              className="px-3 py-1.5 text-sm bg-orange-600 text-white rounded hover:bg-orange-700 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
            >
              {isConnecting && (
                <div className="animate-spin rounded-full h-3 w-3 border-b border-white"></div>
              )}
              {isConnecting ? 'Connecting...' : 'Connect Wallet'}
            </button>
          )}
        </div>
      </div>

      {showDetails && walletStatus.isAuthorized && (
        <div className="mt-4 pt-4 border-t border-gray-600">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
            <div>
              <span className="text-gray-300">Wallet Address:</span>
              <div className="font-mono text-xs mt-1 break-all">{walletStatus.address}</div>
            </div>
            <div>
              <span className="text-gray-300">Ed25519 Key:</span>
              <div className="font-mono text-xs mt-1 break-all">{walletStatus.ed25519Key}</div>
            </div>
            <div>
              <span className="text-gray-300">Authorized:</span>
              <div className="text-xs mt-1">{formatTimestamp(walletStatus.timestamp)}</div>
            </div>
            <div>
              <span className="text-gray-300">Status:</span>
              <div className="text-xs mt-1">
                <span className="px-2 py-0.5 bg-green-900/20 text-green-400 rounded-full">
                  Active
                </span>
              </div>
            </div>
          </div>
          
          <div className="mt-3 p-3 bg-gray-700 rounded text-xs text-gray-300">
            <strong>How it works:</strong> Your wallet signs a message authorizing your Ed25519 key for job submissions. 
            No funds are required and your private keys never leave your browser.
          </div>
        </div>
      )}
    </div>
  );
}
