import { test, expect } from '@playwright/test';

test.describe('Deployment Verification Tests', () => {
  test.beforeEach(async ({ page }) => {
    test.setTimeout(60000);
  });

  test('should verify Netlify redirect rules work correctly', async ({ page }) => {
    const responses = [];
    
    page.on('response', response => {
      responses.push({
        url: response.url(),
        status: response.status(),
        contentType: response.headers()['content-type'],
        finalUrl: response.url()
      });
    });

    // Test portal SPA routing
    await page.goto('/portal/some-non-existent-route');
    await page.waitForLoadState('networkidle');
    
    // Should serve portal index.html for non-existent routes
    const htmlResponse = responses.find(r => r.url.includes('/portal/') && r.contentType?.includes('text/html'));
    expect(htmlResponse).toBeDefined();
    expect(htmlResponse.status).toBe(200);

    // But assets should still be served directly
    await page.goto('/portal');
    await page.waitForLoadState('networkidle');
    
    const assetResponses = responses.filter(r => r.url.includes('/portal/assets/'));
    assetResponses.forEach(response => {
      expect(response.status).toBe(200);
      expect(response.contentType).not.toContain('text/html');
    });
  });

  test('should verify build output structure matches deployment expectations', async ({ page }) => {
    // Test that the expected build structure is accessible
    const testPaths = [
      '/portal/index.html',
      '/docs/index.html',
      '/index.html'
    ];

    for (const path of testPaths) {
      const response = await page.goto(path);
      expect(response.status()).toBe(200);
      expect(response.headers()['content-type']).toContain('text/html');
    }
  });

  test('should verify security headers are present', async ({ page }) => {
    const response = await page.goto('/portal');
    const headers = response.headers();

    // Check for important security headers
    expect(headers['x-frame-options']).toBeDefined();
    expect(headers['x-content-type-options']).toBeDefined();
    expect(headers['referrer-policy']).toBeDefined();
  });
});
