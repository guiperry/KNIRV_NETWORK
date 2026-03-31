import { test, expect } from '@playwright/test';

test.describe('Error Node Demo', () => {
  test('should display Error Node notifications when demo button is clicked', async ({ page }) => {
    // Navigate to application
    await page.goto('http://localhost:3000');
    
    // Wait for page to load
    await page.waitForLoadState('networkidle');
    
    // Wait for knirv shell to load
    await page.waitForSelector('[data-testid="knirv-shell"]', { timeout: 15000 });
    
    // Check initial state - should have no NRV notifications
    await page.waitForTimeout(2000);
    
    // Click the main action button to expand the radial menu
    await page.click('button[style*="cursor-pointer"]');
    
    // Wait for radial buttons to appear
    await page.waitForSelector('button[title="Auto Demo"]', { timeout: 10000 });
    
    // Click the Auto Demo button
    await page.click('button[title="Auto Demo"]');
    
    // Wait for demo to create NRVs
    await page.waitForTimeout(3000);
    
    // Check if NRV visualization container exists
    const nrvContainer = page.locator('[data-testid="nrv-visualization"]');
    await expect(nrvContainer).toBeVisible();
    
    // Check for NRV notifications
    const nrvNotifications = nrvContainer.locator('> div');
    const count = await nrvNotifications.count();
    
    console.log(`Found ${count} NRV notifications`);
    
    // Take screenshot for debugging
    await page.screenshot({ path: 'error-node-debug.png', fullPage: true });
    
    // Verify we have at least some notifications
    expect(count).toBeGreaterThan(0);
  });
});