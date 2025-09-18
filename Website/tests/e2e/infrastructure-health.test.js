import { test, expect } from '@playwright/test';

/**
 * Infrastructure Health Endpoint E2E
 * Ensures the dashboard infrastructure widget does not call non-existent endpoints
 * and that no 404s or console errors are emitted for hybrid router checks.
 */

test.describe('Infrastructure Health Widget', () => {
  test('uses supported hybrid endpoints without 404s', async ({ page }) => {
    test.setTimeout(60000);

    const consoleErrors = [];
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });

    const responses = [];
    page.on('response', (response) => {
      responses.push({ url: response.url(), status: response.status() });
    });

    await page.goto('/portal/dashboard');
    await page.waitForLoadState('domcontentloaded');
    // Wait up to 7s for hybrid health to respond (first mount fetch)
    try {
      await page.waitForResponse((res) => {
        const u = res.url();
        return (u.includes('/hybrid/health') || (u.includes('/health') && u.includes('project-beacon-production.up.railway.app')));
      }, { timeout: 7000 });
    } catch {}

    // Assert no calls to deprecated or missing endpoint
    const badInfraCalls = responses.filter(r => r.url.includes('/infrastructure/health'));
    expect(badInfraCalls.length).toBe(0);

    // Validate expected hybrid endpoints were called and did not 404
    // Accept both direct Railway calls and proxied /hybrid/* calls
    let hybridHealthCalls = responses.filter(r => r.url.includes('/health') && (r.url.includes('project-beacon-production.up.railway.app') || r.url.includes('/hybrid/health')));
    let hybridProvidersCalls = responses.filter(r => r.url.includes('/providers') && (r.url.includes('project-beacon-production.up.railway.app') || r.url.includes('/hybrid/providers')));

    expect(hybridHealthCalls.length).toBeGreaterThan(0);
    // Providers request may not fire immediately on slow connections; allow 0 but prefer >0
    if (hybridProvidersCalls.length === 0) {
      // Try to wait briefly and re-check
      try {
        await page.waitForResponse((res) => {
          const u = res.url();
          return (u.includes('/hybrid/providers') || (u.includes('/providers') && u.includes('project-beacon-production.up.railway.app')));
        }, { timeout: 3000 });
      } catch {}
      // Recompute after waiting
      hybridProvidersCalls = responses.filter(r => r.url.includes('/providers') && (r.url.includes('project-beacon-production.up.railway.app') || r.url.includes('/hybrid/providers')));
    }
    expect(hybridProvidersCalls.length).toBeGreaterThan(0);

    hybridHealthCalls.forEach(r => expect(r.status).toBeLessThan(400));
    hybridProvidersCalls.forEach(r => expect(r.status).toBeLessThan(400));

    // No console error about failed infra fetch
    const infraErrors = consoleErrors.filter(e => e.includes('Failed to fetch infrastructure health') || e.includes('[Hybrid] request failed'));
    expect(infraErrors).toHaveLength(0);
  });
});
