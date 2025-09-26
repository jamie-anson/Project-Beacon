import React from 'react';
import WalletConnection from '../components/WalletConnection.jsx';
import { ToastProvider } from '../state/toast.jsx';

const meta = {
  title: 'Auth/WalletConnection',
  component: WalletConnection,
  tags: ['autodocs']
};

export default meta;

function WalletConnectionStory() {
  const [, rerender] = React.useState(0);

  const overrides = React.useMemo(() => {
    let status = { isAuthorized: false };
    const listeners = new Set();

    const notify = (account) => {
      listeners.forEach((cb) => {
        try {
          cb(account);
        } catch (err) {
          console.warn('Mock listener error:', err);
        }
      });
    };

    return {
      isMetaMaskInstalled: () => true,
      getWalletAuthStatus: () => status,
      getOrCreateKeyPair: async () => ({
        publicKey: new Uint8Array(32),
        privateKey: new Uint8Array(32),
        publicKeyB64: 'mocked-ed25519-public-key'
      }),
      getOrCreateWalletAuth: async () => {
        const now = Date.now();
        status = {
          isAuthorized: true,
          address: '0xBEEFCAFE1234ABCD5678',
          ed25519Key: 'mocked-ed25519-public-key',
          timestamp: now
        };
        rerender((n) => n + 1);
        notify(status.address);
        return {
          address: status.address,
          signature: '0xsignedmessage',
          message: 'Authorize Project Beacon key: mocked-ed25519-public-key',
          chainId: 137,
          nonce: 'mock-nonce',
          expiresAt: new Date(now + 60 * 60 * 1000).toISOString()
        };
      },
      clearWalletAuth: () => {
        status = { isAuthorized: false };
        rerender((n) => n + 1);
        notify(null);
      },
      onAccountsChanged: (cb) => {
        listeners.add(cb);
        return () => listeners.delete(cb);
      }
    };
  }, []);

  return (
    <ToastProvider>
      <div className="bg-ctp-base min-h-[420px] p-6 text-ctp-text">
        <WalletConnection overrides={overrides} />
      </div>
    </ToastProvider>
  );
}

export const Default = {
  render: () => <WalletConnectionStory />
};
