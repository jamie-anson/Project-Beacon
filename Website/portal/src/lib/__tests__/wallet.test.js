import {
  isMetaMaskInstalled,
  connectWallet,
  createAuthMessage,
  getOrCreateWalletAuth,
  getWalletAuthStatus,
  clearWalletAuth,
  getWalletAuthPayload,
} from '../wallet.js';

// Mock exportPublicKey to a stable value so stored auth matches and avoids connectWallet
jest.mock('../crypto.js', () => ({
  exportPublicKey: () => 'K',
  generateNonce: () => 'TEST_NONCE',
}));

// Mock ethers BrowserProvider and signer
jest.mock('ethers', () => {
  const mockSignMessage = jest.fn(async (msg) => `0xSIG_FOR_${msg.length}`);
  const mockGetSigner = jest.fn(async () => ({ signMessage: mockSignMessage }));
  const MockBrowserProvider = jest.fn().mockImplementation(() => ({
    getSigner: mockGetSigner,
  }));
  return { ethers: { BrowserProvider: MockBrowserProvider } };
});

function setupMetaMask({ accounts = ['0xabc'], failRequest = {}, isInstalled = true } = {}) {
  const req = jest.fn(async ({ method }) => {
    if (failRequest[method]) {
      const err = new Error(failRequest[method].message || 'err');
      Object.assign(err, { code: failRequest[method].code });
      throw err;
    }
    if (method === 'eth_accounts' || method === 'eth_requestAccounts') return accounts;
    return null;
  });
  Object.defineProperty(window, 'ethereum', {
    value: {
      isMetaMask: !!isInstalled,
      request: req,
      on: jest.fn(),
      removeListener: jest.fn(),
    },
    configurable: true,
  });
  return { request: req };
}

beforeEach(() => {
  // jsdom localStorage is available; clear between tests
  localStorage.clear();
  // Reset window.ethereum
  delete window.ethereum;
  jest.resetModules();
  jest.clearAllMocks();
});

describe('wallet lib', () => {
  test('isMetaMaskInstalled false when no provider', () => {
    expect(isMetaMaskInstalled()).toBe(false);
  });

  test('isMetaMaskInstalled true when MetaMask present', () => {
    setupMetaMask();
    expect(isMetaMaskInstalled()).toBe(true);
  });

  test('connectWallet success returns address and provider', async () => {
    setupMetaMask({ accounts: ['0xDEADBEEF'] });
    const { address, provider } = await connectWallet();
    expect(address).toBe('0xDEADBEEF');
    expect(provider).toBeDefined();
  });

  test('connectWallet maps user rejection (4001)', async () => {
    setupMetaMask({ failRequest: { eth_requestAccounts: { code: 4001, message: 'User rejected' } }, accounts: [] });
    await expect(connectWallet()).rejects.toThrow('rejected');
  });

  test('createAuthMessage format', () => {
    const msg = createAuthMessage('BASE64KEY');
    expect(msg).toBe('Authorize Project Beacon key: BASE64KEY');
  });

  test('getOrCreateWalletAuth uses stored valid entry', async () => {
    const now = Date.now();
    const stored = {
      address: '0xSTORED',
      signature: '0xSIG',
      message: 'Authorize Project Beacon key: K',
      ed25519Key: 'K',
      timestamp: now,
    };
    localStorage.setItem('beacon:wallet_auth', JSON.stringify(stored));

    // exportPublicKey will be called inside; simulate with a Uint8Array and expect same base64 back in stored
    // We pass any Uint8Array; the function will produce new base64 but we prepared stored to match 'K'
    // To avoid coupling, just ensure when stored exists and valid, it returns that object
    const auth = await getOrCreateWalletAuth(new Uint8Array([1, 2, 3]));
    expect(auth.address).toBe('0xSTORED');
    expect(auth.signature).toBe('0xSIG');
  });

  test('getOrCreateWalletAuth creates new when none stored', async () => {
    setupMetaMask({ accounts: ['0xA1'] });
    const edKey = new Uint8Array([11, 22, 33]);
    const auth = await getOrCreateWalletAuth(edKey);
    expect(auth.address).toBe('0xA1');
    expect(auth.signature.startsWith('0xSIG_FOR_')).toBe(true);
    expect(JSON.parse(localStorage.getItem('beacon:wallet_auth')).address).toBe('0xA1');
  });

  test('getWalletAuthStatus valid and expired', async () => {
    const now = Date.now();
    const ok = { address: '0xOK', signature: '0xS', message: 'M', ed25519Key: 'K', timestamp: now };
    localStorage.setItem('beacon:wallet_auth', JSON.stringify(ok));
    expect(getWalletAuthStatus()).toEqual({ isAuthorized: true, address: '0xOK', ed25519Key: 'K', timestamp: ok.timestamp });

    const old = { ...ok, timestamp: now - 8 * 24 * 60 * 60 * 1000 };
    localStorage.setItem('beacon:wallet_auth', JSON.stringify(old));
    expect(getWalletAuthStatus()).toEqual({ isAuthorized: false });
  });

  test('clearWalletAuth removes storage key', () => {
    localStorage.setItem('beacon:wallet_auth', JSON.stringify({ a: 1 }));
    clearWalletAuth();
    expect(localStorage.getItem('beacon:wallet_auth')).toBeNull();
  });

  test('getWalletAuthPayload wraps fields correctly', async () => {
    setupMetaMask({ accounts: ['0xBEEF'] });
    const payload = await getWalletAuthPayload(new Uint8Array([9, 9, 9]));
    expect(payload).toHaveProperty('wallet_auth.address', '0xBEEF');
    expect(payload.wallet_auth).toHaveProperty('signature');
    expect(payload.wallet_auth).toHaveProperty('message');
  });
});
