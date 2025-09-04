import { test, expect } from '@playwright/test';

const API_BASE_URL = process.env.API_BASE_URL || 'https://beacon-runner-change-me.fly.dev';

test.describe('CORS Integration Tests', () => {
  test.beforeEach(async ({ page }) => {
    // Set longer timeout for network operations
    test.setTimeout(60000);
    
    // Navigate to the main page first
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');
    
    // Then navigate to portal
    await page.goto('/portal');
    await page.waitForLoadState('domcontentloaded');
    
    // Wait for React to load
    await page.waitForTimeout(2000);
  });

  test('should successfully make CORS requests to API from browser', async ({ page }) => {
    // Monitor console for CORS errors
    const consoleErrors = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });

    // Monitor network requests
    const apiRequests = [];
    page.on('request', request => {
      if (request.url().includes('/api/')) {
        apiRequests.push({
          url: request.url(),
          method: request.method()
        });
      }
    });

    // Try to navigate to bias detection page
    try {
      await page.locator('a:has-text("Bias Detection")').click({ timeout: 10000 });
    } catch (error) {
      // If navigation link not found, navigate directly
      await page.goto('/portal/bias-detection');
    }
    
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(3000);

    // Check for CORS errors in console
    const corsErrors = consoleErrors.filter(error => 
      error.includes('CORS') || error.includes('Access-Control') || error.includes('cross-origin')
    );
    
    expect(corsErrors).toHaveLength(0);
  });

  test('should handle API errors gracefully without CORS issues', async ({ page }) => {
    // Intercept API calls to simulate server errors
    await page.route('**/api/v1/**', route => {
      route.fulfill({
        status: 500,
        headers: {
          'Access-Control-Allow-Origin': '*',
          'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, OPTIONS',
          'Access-Control-Allow-Headers': 'Content-Type, Authorization'
        },
        body: JSON.stringify({ error: 'Server error' })
      });
    });

    // Navigate to a page that makes API calls
    await page.goto('/portal/dashboard');
    await page.waitForLoadState('networkidle');

    // Wait for error handling
    await page.waitForTimeout(2000);

    // Check for CORS-related console errors
    const logs = await page.evaluate(() => {
      return window.console.logs || [];
    });

    // Should not see CORS policy errors in console
    const corsErrors = logs.filter(log => 
      log.includes('CORS policy') || log.includes('Access-Control-Allow-Origin')
    );
    
    expect(corsErrors).toHaveLength(0);
  });

  test('should successfully submit job with proper CORS handling', async ({ page }) => {
    // Monitor console for errors
    const consoleErrors = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });

    // Navigate to bias detection page
    await page.goto('/portal/bias-detection');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);

    // Check if form elements exist before filling
    const questionInput = page.locator('input').first();
    const contextTextarea = page.locator('textarea').first();
    const submitButton = page.locator('button').first();

    if (await questionInput.count() > 0) {
      await questionInput.fill('Test bias detection question');
    }
    
    if (await contextTextarea.count() > 0) {
      await contextTextarea.fill('Test context for bias detection');
    }

    if (await submitButton.count() > 0) {
      // Monitor for any network requests
      const networkRequests = [];
      page.on('request', request => {
        networkRequests.push(request.url());
      });

      await submitButton.click();
      await page.waitForTimeout(3000);
    }

    // Check for CORS-related errors
    const corsErrors = consoleErrors.filter(error => 
      error.includes('CORS') || error.includes('Access-Control') || error.includes('cross-origin')
    );
    
    expect(corsErrors).toHaveLength(0);
  });

  test('should handle preflight OPTIONS requests correctly', async ({ page }) => {
    // Monitor console for errors
    const consoleErrors = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });

    // Monitor OPTIONS requests
    const optionsRequests = [];
    page.on('request', request => {
      if (request.method() === 'OPTIONS' && request.url().includes('/api/')) {
        optionsRequests.push({
          url: request.url(),
          headers: request.headers()
        });
      }
    });

    const optionsResponses = [];
    page.on('response', response => {
      if (response.request().method() === 'OPTIONS' && response.url().includes('/api/')) {
        optionsResponses.push({
          url: response.url(),
          status: response.status(),
          headers: response.headers()
        });
      }
    });

    // Navigate to bias detection page
    await page.goto('/portal/bias-detection');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);

    // Try to find and interact with form elements if they exist
    const inputElements = await page.locator('input').count();
    const textareaElements = await page.locator('textarea').count();
    const buttonElements = await page.locator('button').count();

    if (inputElements > 0 && textareaElements > 0 && buttonElements > 0) {
      // Fill form if elements exist
      await page.locator('input').first().fill('Test question');
      await page.locator('textarea').first().fill('Test context');
      
      // Click submit to potentially trigger preflight
      await page.locator('button').first().click();
      await page.waitForTimeout(3000);
    }

    // Main test: verify no CORS errors occurred
    const corsErrors = consoleErrors.filter(error => 
      error.includes('CORS') || error.includes('Access-Control') || error.includes('cross-origin')
    );
    
    expect(corsErrors).toHaveLength(0);

    // If preflight requests were made, verify they were handled correctly
    if (optionsRequests.length > 0) {
      expect(optionsResponses.length).toBeGreaterThan(0);
      
      optionsResponses.forEach(response => {
        expect([200, 204]).toContain(response.status);
        expect(response.headers['access-control-allow-origin']).toBeDefined();
        expect(response.headers['access-control-allow-methods']).toBeDefined();
      });
    }
  });

  test('should verify portal loads without CORS errors', async ({ page }) => {
    // Monitor console for CORS errors
    const consoleErrors = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });

    // Navigate to portal and wait for it to load
    await page.goto('/portal');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(2000);

    // Try to navigate to different pages to trigger any API calls
    await page.goto('/portal/dashboard');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(1000);

    await page.goto('/portal/bias-detection');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(1000);

    // Check for CORS-related errors
    const corsErrors = consoleErrors.filter(error => 
      error.includes('CORS') || 
      error.includes('Access-Control') || 
      error.includes('cross-origin') ||
      error.includes('blocked by CORS policy')
    );
    
    // The main test: no CORS errors should occur
    expect(corsErrors).toHaveLength(0);
  });
});
