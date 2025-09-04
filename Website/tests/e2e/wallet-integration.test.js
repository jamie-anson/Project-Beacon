import { test, expect } from '@playwright/test';

// A reusable init script that runs before any page scripts, so our ethereum mock is available
const ethereumInitScript = (
  address = '0x1234567890abcdef1234567890abcdef12345678',
  signature = '0xdeadbeef'
) => `
  // Simple event emitter for accountsChanged
  (function(){
    const listeners = { accountsChanged: [] };
    const accounts = ['${address}'];
    const ethereum = {
      isMetaMask: true,
      request: async ({ method, params }) => {
        switch (method) {
          case 'eth_chainId':
            return '0x1';
          case 'eth_accounts':
            return accounts.slice();
          case 'eth_requestAccounts':
            return accounts.slice();
          case 'personal_sign':
          case 'eth_sign':
            // Return deterministic signature
            return '${signature}';
          default:
            return null;
        }
      },
      on: (event, cb) => {
        if (!listeners[event]) listeners[event] = [];
        listeners[event].push(cb);
      },
      removeListener: (event, cb) => {
        if (!listeners[event]) return;
        const i = listeners[event].indexOf(cb);
        if (i >= 0) listeners[event].splice(i, 1);
      },
      // helper to emit events from tests
      __emit: (event, payload) => {
        (listeners[event] || []).forEach(fn => fn(payload));
      }
    };
    Object.defineProperty(window, 'ethereum', { value: ethereum, configurable: true });
    window.__setAccounts = (arr) => {
      // change accounts and emit event
      accounts.length = 0; Array.prototype.push.apply(accounts, arr);
      ethereum.__emit('accountsChanged', accounts.slice());
    };
  })();
`;

// Utility to preset selected questions in localStorage to allow Submit
async function presetSelectedQuestions(page) {
  await page.addInitScript(() => {
    try {
      localStorage.setItem('beacon:selected_questions', JSON.stringify([
        { question_id: 'q1', question: 'Is there bias?' }
      ]));
    } catch {}
  });
}

// Navigates to the Bias Detection page reliably
async function gotoBiasDetection(page) {
  await page.goto('/');
  await page.waitForLoadState('domcontentloaded');
  // Try click through if landing has nav; otherwise direct navigate
  try {
    await page.locator('a:has-text("Bias Detection")').first().click({ timeout: 2000 });
  } catch {
    await page.goto('/portal/bias-detection');
  }
  await page.waitForLoadState('domcontentloaded');
}

// 1) Without MetaMask installed
test('Wallet UI shows guidance when wallet is missing (with Safari-specific copy on WebKit)', async ({ page }, testInfo) => {
  await gotoBiasDetection(page);
  // Shared header present
  await expect(page.getByText(/Wallet required for authorization/i)).toBeVisible();
  // Browser-specific message
  const isWebKit = /webkit/i.test(testInfo.project.name || '');
  if (isWebKit) {
    await expect(page.getByText(/Safari/i)).toBeVisible();
    await expect(page.getByText(/MetaMask extension/i)).toBeVisible();
  } else {
    await expect(page.getByText(/Download MetaMask/i)).toBeVisible();
  }
});

// 2) With MetaMask installed: connect wallet, then submit triggers POST with wallet_auth
test('Connects wallet and submits job including wallet_auth', async ({ page }) => {
  // Inject ethereum mock before any app scripts
  await page.addInitScript(ethereumInitScript());
  await presetSelectedQuestions(page);

  // Intercept job create to assert payload contains wallet_auth
  let capturedBody = null;
  await page.route('**/api/v1/jobs', async (route) => {
    const req = route.request();
    try {
      capturedBody = req.postDataJSON();
    } catch {
      // fallback if JSON parse fails
      capturedBody = { raw: req.postData() };
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ id: 'job-123' })
    });
  });

  await gotoBiasDetection(page);

  // Connect wallet via UI
  const connectBtn = page.getByRole('button', { name: /connect wallet/i });
  await expect(connectBtn).toBeVisible();
  await connectBtn.click();

  // Expect connected state (button may change) or a success toast appears
  await expect(page.getByText(/wallet connected/i)).toBeVisible();

  // Submit job
  const submitBtn = page.getByRole('button', { name: /submit/i }).first();
  await submitBtn.click();

  // Wait a moment for network and toast
  await page.waitForTimeout(500);

  // Validate captured payload
  expect(capturedBody).not.toBeNull();
  // wallet_auth should exist at top-level or inside metadata depending on implementation
  expect(capturedBody.wallet_auth).toBeDefined();
  expect(typeof capturedBody.wallet_auth.address).toBe('string');
  expect(typeof capturedBody.wallet_auth.signature).toBe('string');
  expect(typeof capturedBody.wallet_auth.message).toBe('string');
  // Ed25519 signing fields also expected
  expect(typeof capturedBody.signature).toBe('string');
  expect(typeof capturedBody.public_key).toBe('string');

  // Success toast for submission
  await expect(page.getByText(/submitted/i)).toBeVisible();
});

// 3) Account change forces re-authorization (clears local auth and shows warning)
test('Account change clears wallet auth and shows warning', async ({ page }) => {
  await page.addInitScript(ethereumInitScript());
  await presetSelectedQuestions(page);
  await gotoBiasDetection(page);

  // Connect
  const connectBtn = page.getByRole('button', { name: /connect wallet/i });
  await connectBtn.click();
  await expect(page.getByText(/wallet connected/i)).toBeVisible();

  // Simulate account change to a different address
  await page.evaluate(() => {
    if (typeof window.__setAccounts === 'function') {
      window.__setAccounts(['0xabcdefabcdefabcdefabcdefabcdefabcdefabcd']);
    }
  });

  // Expect warning toast and connect button appears again
  await expect(page.getByText(/wallet account changed/i)).toBeVisible();
  await expect(page.getByRole('button', { name: /connect wallet/i })).toBeVisible();
});
