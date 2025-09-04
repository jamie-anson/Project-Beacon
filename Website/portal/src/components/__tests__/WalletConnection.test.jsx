import React from 'react';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import WalletConnection from '../WalletConnection.jsx';

// Mocks
jest.mock('../../lib/wallet.js', () => ({
  isMetaMaskInstalled: jest.fn(() => true),
  getWalletAuthStatus: jest.fn(() => ({ isAuthorized: false })),
  getOrCreateWalletAuth: jest.fn(async () => ({ address: '0xABCDEF', signature: '0xS', message: 'M' })),
  clearWalletAuth: jest.fn(),
  onAccountsChanged: jest.fn((cb) => {
    // Return cleanup
    return () => {};
  }),
}));

jest.mock('../../lib/crypto.js', () => ({
  getOrCreateKeyPair: jest.fn(async () => ({ publicKey: new Uint8Array([1,2,3]) })),
}));

const mockAddToast = jest.fn();
jest.mock('../../state/toast.jsx', () => ({
  useToast: () => ({ add: mockAddToast }),
}));

// Helpers to access mocks
const walletLib = require('../../lib/wallet.js');

describe('WalletConnection component', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('shows wallet required notice when not installed', () => {
    walletLib.isMetaMaskInstalled.mockReturnValue(false);
    render(<WalletConnection />);
    expect(screen.getByText(/Wallet required for authorization/i)).toBeInTheDocument();
    expect(screen.getByText(/Download MetaMask/)).toBeInTheDocument();
  });

  test('connect flow: creates auth and shows connected state', async () => {
    walletLib.isMetaMaskInstalled.mockReturnValue(true);
    // initial unauthorized
    walletLib.getWalletAuthStatus.mockReturnValueOnce({ isAuthorized: false });
    // after connect, authorized
    walletLib.getWalletAuthStatus.mockReturnValueOnce({ isAuthorized: true, address: '0xABCDEF', ed25519Key: 'K', timestamp: Date.now() });

    render(<WalletConnection />);

    const btn = screen.getByRole('button', { name: /connect wallet/i });
    fireEvent.click(btn);

    await waitFor(() => expect(walletLib.getOrCreateWalletAuth).toHaveBeenCalled());
    // Connected text visible
    expect(await screen.findByText(/Connected:/)).toBeInTheDocument();
    expect(mockAddToast).toHaveBeenCalled();
  });

  test('disconnect clears auth and shows connect button', async () => {
    walletLib.isMetaMaskInstalled.mockReturnValue(true);
    // start authorized
    walletLib.getWalletAuthStatus.mockReturnValueOnce({ isAuthorized: true, address: '0xAAAA', ed25519Key: 'K', timestamp: Date.now() });
    // after disconnect unauthorized
    walletLib.getWalletAuthStatus.mockReturnValueOnce({ isAuthorized: false });

    render(<WalletConnection />);

    const btn = await screen.findByRole('button', { name: /disconnect/i });
    fireEvent.click(btn);

    expect(walletLib.clearWalletAuth).toHaveBeenCalled();
    // Connect Wallet button appears again
    expect(await screen.findByRole('button', { name: /connect wallet/i })).toBeInTheDocument();
  });

  test('accountsChanged listener triggers warning toasts', async () => {
    const callbacks = [];
    walletLib.onAccountsChanged.mockImplementation((cb) => {
      callbacks.push(cb);
      return () => {};
    });

    walletLib.isMetaMaskInstalled.mockReturnValue(true);
    walletLib.getWalletAuthStatus.mockReturnValue({ isAuthorized: true, address: '0xAAAA', ed25519Key: 'K', timestamp: Date.now() });

    render(<WalletConnection />);

    // simulate disconnect
    await act(async () => {
      callbacks[0](null);
    });
    expect(mockAddToast).toHaveBeenCalled();
    // simulate account change
    await act(async () => {
      callbacks[0]('0xBBBB');
    });
    expect(mockAddToast).toHaveBeenCalledTimes(2);
  });
});
