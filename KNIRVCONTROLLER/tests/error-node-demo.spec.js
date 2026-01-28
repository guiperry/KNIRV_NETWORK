import { test, expect } from '@playwright/test';

test.describe('Error Node Demo', () => {
  test('should display Error Node notifications when demo button is clicked', async ({ page }) => {
    // Navigate to the application
    await page.goto('http://localhost:3000');
    
    // Wait for page to load
    await page.waitForSelector('[data-testid="knirv-shell"]', { timeout: 10000 });
    
    // Initially, there should be no NRV notifications
    const initialNRVs = await page.locator('[data-testid="nrv-visualization"] > div').count();
    expect(initialNRVs).toBe(0);
    
    // Click the main action button to expand the menu
    await page.click('[data-testid="knirv-shell"] button[style*="cursor-pointer"]');
    
    // Wait for the radial buttons to appear
    await page.waitForSelector('button[title="Auto Demo"]', { timeout: 5000 });
    
    // Click the Auto Demo button (the one with Mic icon)
    await page.click('button[title="Auto Demo"]');
    
    // Wait for the demo NRVs to be created and displayed
    await page.waitForTimeout(2000);
    
    // Check that NRV notifications are now visible on the left side
    const nrvNotifications = await page.locator('[data-testid="nrv-visualization"] > div').count();
    expect(nrvNotifications).toBeGreaterThan(0);
    
    // Verify the content of the first NRV notification
    const firstNRV = page.locator('[data-testid="nrv-visualization"] > div').first();
    await expect(firstNRV).toBeVisible();
    await expect(firstNRV.locator('text=React component failed to render')).toBeVisible();
    await expect(firstNRV.locator('text=High')).toBeVisible();
    
    // Take a screenshot for verification
    await page.screenshot({ path: 'error-node-demo-test.png', fullPage: true });
    
    console.log('Error Node demo test passed! Notifications are visible.');
  });
});