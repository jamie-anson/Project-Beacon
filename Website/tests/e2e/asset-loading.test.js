import { test, expect } from '@playwright/test';

test.describe('Asset Loading Tests', () => {
  test.beforeEach(async ({ page }) => {
    test.setTimeout(60000);
  });

  test('should load portal assets with correct MIME types', async ({ page }) => {
    const responses = [];
    const consoleErrors = [];
    
    // Monitor all responses for assets
    page.on('response', response => {
      if (response.url().includes('/portal/assets/')) {
        responses.push({
          url: response.url(),
          status: response.status(),
          contentType: response.headers()['content-type'],
          statusText: response.statusText()
        });
      }
    });

    // Monitor console for MIME type errors
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });

    // Navigate to portal and wait for all assets to load
    await page.goto('/portal');
    await page.waitForLoadState('networkidle');

    // Verify we have asset responses
    expect(responses.length).toBeGreaterThan(0);

    // Verify CSS files load with correct MIME type
    const cssFiles = responses.filter(r => r.url.endsWith('.css'));
    cssFiles.forEach(file => {
      expect(file.status).toBe(200);
      expect(file.contentType).toContain('text/css');
      expect(file.contentType).not.toContain('text/html');
    });

    // Verify JS files load with correct MIME type
    const jsFiles = responses.filter(r => r.url.endsWith('.js'));
    jsFiles.forEach(file => {
      expect(file.status).toBe(200);
      expect(file.contentType).toMatch(/application\/javascript|text\/javascript/);
      expect(file.contentType).not.toContain('text/html');
    });

    // Check for MIME type errors in console
    const mimeErrors = consoleErrors.filter(error => 
      error.includes('MIME type') || 
      error.includes('stylesheet') ||
      error.includes('Refused to apply style')
    );
    
    expect(mimeErrors).toHaveLength(0);
  });

  test('should not redirect asset requests to HTML pages', async ({ page }) => {
    const assetRedirects = [];
    
    page.on('response', response => {
      const url = response.url();
      if (url.includes('/portal/assets/')) {
        const contentType = response.headers()['content-type'] || '';
        if (contentType.includes('text/html')) {
          assetRedirects.push({
            url,
            status: response.status(),
            contentType
          });
        }
      }
    });

    await page.goto('/portal');
    await page.waitForLoadState('networkidle');

    // Asset requests should never return HTML
    expect(assetRedirects).toHaveLength(0);
  });

  test('should load portal with all required assets', async ({ page }) => {
    const failedRequests = [];
    
    page.on('requestfailed', request => {
      if (request.url().includes('/portal/')) {
        failedRequests.push({
          url: request.url(),
          failure: request.failure()
        });
      }
    });

    await page.goto('/portal');
    await page.waitForLoadState('networkidle');

    // Portal should load without any failed asset requests
    expect(failedRequests).toHaveLength(0);

    // Verify the portal actually rendered
    await expect(page.locator('#root')).toBeVisible();
    
    // Check that React app loaded (should have navigation)
    await expect(page.locator('nav')).toBeVisible({ timeout: 10000 });
  });

  test('should handle asset requests consistently across different routes', async ({ page }) => {
    const routes = ['/portal', '/portal/dashboard', '/portal/bias-detection'];
    
    for (const route of routes) {
      const responses = [];
      
      page.on('response', response => {
        if (response.url().includes('/portal/assets/')) {
          responses.push({
            route,
            url: response.url(),
            status: response.status(),
            contentType: response.headers()['content-type']
          });
        }
      });

      await page.goto(route);
      await page.waitForLoadState('networkidle');
      
      // Each route should load assets successfully
      const cssFiles = responses.filter(r => r.url.endsWith('.css'));
      const jsFiles = responses.filter(r => r.url.endsWith('.js'));
      
      cssFiles.forEach(file => {
        expect(file.status).toBe(200);
        expect(file.contentType).toContain('text/css');
      });
      
      jsFiles.forEach(file => {
        expect(file.status).toBe(200);
        expect(file.contentType).toMatch(/application\/javascript|text\/javascript/);
      });
    }
  });
});
