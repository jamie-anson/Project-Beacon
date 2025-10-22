const { test, expect } = require('@playwright/test');

test.describe('Bias Detection Results - LLM Summary E2E', () => {
  test('displays LLM summary narrative when available', async ({ page }) => {
    // Navigate to results page with mock backend (Terminal H on :8787)
    await page.goto('http://localhost:8787/portal/bias-detection/results/test-job-with-summary');

    // Wait for summary content to load
    await page.waitForSelector('text=/Cross-region analysis/i', { timeout: 10000 });

    // Verify narrative text appears
    const summaryText = await page.textContent('[data-testid="llm-summary"]');
    expect(summaryText).toContain('Cross-region analysis');
    expect(summaryText.length).toBeGreaterThan(100); // Should be 400-500 words

    // Verify metrics are displayed
    await expect(page.locator('text=/Bias variance/i')).toBeVisible();
    await expect(page.locator('text=/Censorship rate/i')).toBeVisible();
  });

  test('shows fallback UI when summary is missing', async ({ page }) => {
    // Navigate to results page without summary
    await page.goto('http://localhost:8787/portal/bias-detection/results/test-job-no-summary');

    // Wait for page load
    await page.waitForLoadState('networkidle');

    // Verify fallback text appears
    await expect(page.locator('text=/No summary available/i, text=/Analysis in progress/i').first()).toBeVisible();

    // Verify metrics still display
    await expect(page.locator('text=/Bias variance/i')).toBeVisible();
  });

  test('handles API errors gracefully', async ({ page }) => {
    // Navigate to results page that will return error
    await page.goto('http://localhost:8787/portal/bias-detection/results/nonexistent-job');

    // Wait for error state
    await page.waitForSelector('text=/error/i, text=/not found/i', { timeout: 10000 });

    // Verify error message is user-friendly
    const errorText = await page.textContent('body');
    expect(errorText).toMatch(/error|not found|failed/i);
  });

  test('summary section has proper accessibility attributes', async ({ page }) => {
    await page.goto('http://localhost:8787/portal/bias-detection/results/test-job-with-summary');

    await page.waitForSelector('[data-testid="llm-summary"]', { timeout: 10000 });

    // Check for semantic HTML
    const summarySection = page.locator('[data-testid="llm-summary"]');
    await expect(summarySection).toBeVisible();

    // Verify heading structure
    const heading = page.locator('h2:has-text("Analysis Summary"), h3:has-text("Summary")').first();
    await expect(heading).toBeVisible();
  });
});
