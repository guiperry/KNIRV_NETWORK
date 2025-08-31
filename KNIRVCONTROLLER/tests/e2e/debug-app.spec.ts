/**
 * Debug test to check what's actually being rendered
 */

import { test, expect } from '@playwright/test';

test.describe('Debug KNIRVCONTROLLER Application', () => {
  test('should debug what elements are present', async ({ page }) => {
    // Navigate to the application
    await page.goto('http://localhost:3000');
    
    // Wait for the application to load
    await page.waitForLoadState('networkidle');
    
    // Take a screenshot
    await page.screenshot({ path: 'debug-screenshot.png', fullPage: true });
    
    // Check what's actually in the DOM
    const bodyContent = await page.locator('body').innerHTML();
    console.log('Body content length:', bodyContent.length);
    
    // Check for any elements with data-testid
    const testIdElements = await page.locator('[data-testid]').all();
    console.log('Found elements with data-testid:', testIdElements.length);
    
    for (const element of testIdElements) {
      const testId = await element.getAttribute('data-testid');
      const isVisible = await element.isVisible();
      console.log(`- ${testId}: visible=${isVisible}`);
    }
    
    // Check for burger menu specifically
    const burgerMenus = await page.locator('button').all();
    console.log('Found buttons:', burgerMenus.length);
    
    for (let i = 0; i < Math.min(burgerMenus.length, 10); i++) {
      const button = burgerMenus[i];
      const classes = await button.getAttribute('class');
      const testId = await button.getAttribute('data-testid');
      const isVisible = await button.isVisible();
      console.log(`Button ${i}: testId=${testId}, visible=${isVisible}, classes=${classes?.substring(0, 100)}`);
    }
    
    // Check if the app is actually loading
    const title = await page.title();
    console.log('Page title:', title);
    
    // Check for any error messages
    const errors = await page.locator('.error, [class*="error"]').all();
    console.log('Found error elements:', errors.length);
    
    // Check for React components
    const reactElements = await page.locator('[data-reactroot], #root, #app').all();
    console.log('Found React root elements:', reactElements.length);
    
    // Wait a bit more and check again
    await page.waitForTimeout(3000);
    
    const testIdElementsAfterWait = await page.locator('[data-testid]').all();
    console.log('Found elements with data-testid after wait:', testIdElementsAfterWait.length);
    
    // Check console logs
    const logs = await page.evaluate(() => {
      return window.console;
    });

    // Log console information for debugging
    console.log('Browser console object available:', !!logs);

    // This test should always pass, it's just for debugging
    expect(true).toBe(true);
  });
});
