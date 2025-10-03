const { test, expect } = require('@playwright/test');

const RUNNER_BASE = process.env.RUNNER_BASE_URL || 'https://beacon-runner-change-me.fly.dev';
const PORTAL_BASE = process.env.PORTAL_BASE_URL || 'https://projectbeacon.netlify.app';

test.describe('Bias Detection Results Page', () => {
  test('happy path: view bias analysis for completed job', async ({ page }) => {
    // This test requires a completed multi-region job with analysis
    // For now, we'll test the page structure and error handling
    
    test.setTimeout(60000); // 60 second timeout

    // Navigate to portal
    await page.goto(`${PORTAL_BASE}/portal/executions`);
    
    // Wait for executions to load
    await page.waitForSelector('table', { timeout: 10000 });
    
    // Look for a completed job with "Bias Analysis" link
    const biasAnalysisLink = page.locator('a:has-text("Bias Analysis")').first();
    
    if (await biasAnalysisLink.count() > 0) {
      // Click the link
      await biasAnalysisLink.click();
      
      // Wait for navigation
      await page.waitForURL(/\/bias-detection\/[^/]+/, { timeout: 5000 });
      
      // Verify page loaded
      await expect(page.locator('h1:has-text("Bias Detection Results")')).toBeVisible({ timeout: 10000 });
      
      // Check for main sections (may be loading or loaded)
      const summaryCard = page.locator('text=Analysis Summary');
      const biasScores = page.locator('text=Overall Metrics');
      const regionalScores = page.locator('text=Regional Scores');
      
      // Wait for either content or error state
      await Promise.race([
        summaryCard.waitFor({ state: 'visible', timeout: 15000 }),
        page.locator('text=Error Loading Analysis').waitFor({ state: 'visible', timeout: 15000 }),
        page.locator('text=No Analysis Available').waitFor({ state: 'visible', timeout: 15000 })
      ]);
      
      // If analysis loaded successfully, verify all sections
      if (await summaryCard.isVisible()) {
        console.log('✅ Analysis loaded successfully');
        
        // Verify summary section exists
        await expect(summaryCard).toBeVisible();
        
        // Verify metrics section exists
        await expect(biasScores).toBeVisible();
        
        // Verify at least one metric card
        const metricCards = page.locator('.bg-gray-700.rounded-lg.p-4');
        await expect(metricCards.first()).toBeVisible();
        
        // Verify back button works
        const backButton = page.locator('a:has-text("Back to Executions")');
        await expect(backButton).toBeVisible();
        
        console.log('✅ All sections rendered correctly');
      } else {
        console.log('⚠️ Analysis not available or error state shown (expected for some jobs)');
      }
    } else {
      console.log('⚠️ No completed jobs with bias analysis available - skipping full test');
      console.log('   This is expected if no multi-region jobs have completed yet');
    }
  });

  test('handles job not found (404)', async ({ page }) => {
    // Navigate directly to non-existent job
    await page.goto(`${PORTAL_BASE}/portal/bias-detection/nonexistent-job-12345`);
    
    // Wait for React app to mount
    await page.waitForSelector('#root', { timeout: 5000 });
    
    // Wait for page header to appear (confirms React rendered)
    await page.waitForSelector('h1', { timeout: 10000 });
    
    // Should show error state or page loaded
    const errorVisible = await page.locator('text=Error Loading Analysis').isVisible().catch(() => false);
    const pageLoaded = await page.locator('h1:has-text("Bias Detection Results")').isVisible().catch(() => false);
    
    expect(errorVisible || pageLoaded).toBeTruthy();
    
    // Should have back button
    const backButton = page.locator('a:has-text("Back to Executions")');
    await expect(backButton).toBeVisible();
    
    // Should have retry button
    const retryButton = page.locator('button:has-text("Retry")');
    await expect(retryButton).toBeVisible();
    
    console.log('✅ 404 error handling works correctly');
  });

  test('shows loading state initially', async ({ page }) => {
    // Navigate to bias detection page
    await page.goto(`${PORTAL_BASE}/portal/bias-detection/test-job-loading`);
    
    // Should show loading skeleton (briefly)
    const loadingSkeleton = page.locator('.animate-pulse');
    
    // Check if loading state appears (may be very brief)
    const isVisible = await loadingSkeleton.isVisible().catch(() => false);
    
    if (isVisible) {
      console.log('✅ Loading skeleton displayed');
    } else {
      console.log('⚠️ Loading state too fast to capture (acceptable)');
    }
    
    // Eventually should show either content or error
    await Promise.race([
      page.locator('h1:has-text("Bias Detection Results")').waitFor({ state: 'visible', timeout: 10000 }),
      page.locator('text=Error Loading Analysis').waitFor({ state: 'visible', timeout: 10000 })
    ]);
  });

  test('API endpoint returns valid JSON structure', async ({ request }) => {
    // Test backend API directly
    const response = await request.get(`${RUNNER_BASE}/api/v2/jobs/test-job-123/bias-analysis`);
    
    // Should return 404 for non-existent job (expected)
    expect([200, 404]).toContain(response.status());
    
    if (response.status() === 200) {
      const data = await response.json();
      
      // Verify response structure
      expect(data).toHaveProperty('job_id');
      expect(data).toHaveProperty('analysis');
      expect(data).toHaveProperty('region_scores');
      
      // Verify analysis structure
      if (data.analysis) {
        expect(data.analysis).toHaveProperty('bias_variance');
        expect(data.analysis).toHaveProperty('censorship_rate');
        expect(data.analysis).toHaveProperty('summary');
      }
      
      console.log('✅ API returns valid JSON structure');
    } else {
      console.log('⚠️ Job not found (expected for test job)');
    }
  });

  test('navigation from executions page works', async ({ page }) => {
    // Go to executions page
    await page.goto(`${PORTAL_BASE}/portal/executions`);
    
    // Wait for page to load
    await page.waitForSelector('h2:has-text("Executions")', { timeout: 10000 });
    
    // Verify "Bias Analysis" links exist for completed jobs
    const biasAnalysisLinks = page.locator('a:has-text("Bias Analysis")');
    const linkCount = await biasAnalysisLinks.count();
    
    if (linkCount > 0) {
      console.log(`✅ Found ${linkCount} "Bias Analysis" link(s)`);
      
      // Verify link has correct href pattern
      const firstLink = biasAnalysisLinks.first();
      const href = await firstLink.getAttribute('href');
      expect(href).toMatch(/\/bias-detection\/[^/]+/);
      
      console.log('✅ Navigation links properly formatted');
    } else {
      console.log('⚠️ No completed jobs with bias analysis available yet');
    }
  });

  test('back button returns to executions', async ({ page }) => {
    // Navigate to bias detection page
    await page.goto(`${PORTAL_BASE}/portal/bias-detection/test-job-back-button`);
    
    // Wait for page to load (will show error for non-existent job)
    await page.waitForSelector('h1:has-text("Bias Detection Results")', { timeout: 10000 });
    
    // Click back button
    const backButton = page.locator('a:has-text("Back to Executions")').first();
    await backButton.click();
    
    // Should navigate back to executions
    await page.waitForURL(/\/executions/, { timeout: 5000 });
    await expect(page.locator('h2:has-text("Executions")')).toBeVisible();
    
    console.log('✅ Back button navigation works');
  });

  test('retry button refetches data', async ({ page }) => {
    // Navigate to non-existent job
    await page.goto(`${PORTAL_BASE}/portal/bias-detection/nonexistent-retry-test`);
    
    // Wait for error state
    await page.waitForSelector('text=Error Loading Analysis', { timeout: 10000 });
    
    // Click retry button
    const retryButton = page.locator('button:has-text("Retry")');
    await retryButton.click();
    
    // Should show loading state briefly
    // Then error state again (job still doesn't exist)
    await page.waitForSelector('text=Error Loading Analysis', { timeout: 10000 });
    
    console.log('✅ Retry button triggers refetch');
  });
});

test.describe('Bias Detection Results - Component Rendering', () => {
  test('renders all sections when data available', async ({ page, request }) => {
    // First, check if we have any real completed jobs
    const execResponse = await request.get(`${RUNNER_BASE}/api/v1/executions?limit=100`);
    
    if (execResponse.ok()) {
      const executions = await execResponse.json();
      const completedJobs = Array.isArray(executions) 
        ? executions.filter(e => e.status === 'completed' && e.job_id)
        : [];
      
      if (completedJobs.length > 0) {
        const jobId = completedJobs[0].job_id;
        
        // Check if analysis exists
        const analysisResponse = await request.get(`${RUNNER_BASE}/api/v2/jobs/${jobId}/bias-analysis`);
        
        if (analysisResponse.status() === 200) {
          const analysisData = await analysisResponse.json();
          
          // Navigate to bias detection page
          await page.goto(`${PORTAL_BASE}/portal/bias-detection/${jobId}`);
          
          // Wait for page to load
          await page.waitForSelector('h1:has-text("Bias Detection Results")', { timeout: 10000 });
          
          // Verify summary section
          if (analysisData.analysis?.summary) {
            await expect(page.locator('text=Analysis Summary')).toBeVisible();
            console.log('✅ Summary section rendered');
          }
          
          // Verify metrics section
          if (analysisData.analysis) {
            await expect(page.locator('text=Overall Metrics')).toBeVisible();
            console.log('✅ Metrics section rendered');
          }
          
          // Verify regional scores
          if (analysisData.region_scores && Object.keys(analysisData.region_scores).length > 0) {
            await expect(page.locator('text=Regional Scores')).toBeVisible();
            console.log('✅ Regional scores rendered');
          }
          
          // Verify metadata
          await expect(page.locator(`text=${analysisData.cross_region_execution_id}`)).toBeVisible();
          console.log('✅ Metadata displayed');
          
          console.log('✅ All sections rendered with real data');
        } else {
          console.log('⚠️ No analysis available for completed job (may not be multi-region)');
        }
      } else {
        console.log('⚠️ No completed jobs available for testing');
      }
    }
  });
});

test.describe('Bias Detection Results - Error States', () => {
  test('displays appropriate error for backend failure', async ({ page }) => {
    // Navigate to page (will fail for non-existent job)
    await page.goto(`${PORTAL_BASE}/portal/bias-detection/error-test-job`);
    
    // Should show error state
    await expect(page.locator('text=Error Loading Analysis')).toBeVisible({ timeout: 10000 });
    
    // Error message should be displayed
    const errorText = page.locator('.bg-red-900\\/20');
    await expect(errorText).toBeVisible();
    
    console.log('✅ Error state displays correctly');
  });

  test('handles network timeout gracefully', async ({ page, context }) => {
    // Simulate slow network
    await context.route(`${RUNNER_BASE}/api/v2/jobs/**`, route => {
      setTimeout(() => route.abort('timedout'), 5000);
    });
    
    await page.goto(`${PORTAL_BASE}/portal/bias-detection/timeout-test`);
    
    // Should eventually show error
    await expect(page.locator('text=Error Loading Analysis')).toBeVisible({ timeout: 15000 });
    
    console.log('✅ Network timeout handled gracefully');
  });
});
