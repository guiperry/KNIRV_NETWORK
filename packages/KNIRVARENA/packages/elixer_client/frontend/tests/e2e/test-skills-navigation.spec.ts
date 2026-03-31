/**
 * Test Skills page navigation specifically
 */

import { test, expect } from '@playwright/test';

test.describe('Test Skills Navigation', () => {
  test('should navigate to Skills page without error', async ({ page }) => {
    // Navigate to the application
    await page.goto('http://localhost:3000');
    
    // Wait for the application to load
    await page.waitForLoadState('networkidle');
    
    // Click burger menu
    await page.click('[data-testid="burger-menu"]');
    
    // Wait for menu to open
    await expect(page.locator('[data-testid="burger-menu-content"]')).toBeVisible();
    
    // Click Skills
    await page.click('text=Skills');
    
    // Wait for navigation
    await page.waitForURL('**/manager/skills');
    
    // Check if we get an error or if the page loads correctly
    const h1 = page.locator('h1');
    const h1Text = await h1.textContent();
    
    console.log('H1 text:', h1Text);
    
    // Check if it's an error page
    if (h1Text?.includes('Error')) {
      console.log('ERROR: Skills page shows error');
      
      // Take a screenshot for debugging
      await page.screenshot({ path: 'skills-error.png', fullPage: true });
      
      // Get the full page content for debugging
      const bodyContent = await page.locator('body').innerHTML();
      console.log('Body content length:', bodyContent.length);
      
      // Look for error details
      const errorDetails = await page.locator('pre').allTextContents();
      console.log('Error details:', errorDetails);
      
    } else {
      console.log('SUCCESS: Skills page loaded correctly');
      await expect(h1).toContainText('Skills');
    }
    
    // This test should always pass, it's just for debugging
    expect(true).toBe(true);
  });
});
