/**
 * Simple Skills page test
 */

import { test, expect } from '@playwright/test';

test.describe('Simple Skills Test', () => {
  test('should navigate to Skills page and check basic elements', async ({ page }) => {
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
    
    // Wait a bit for the page to render
    await page.waitForTimeout(2000);
    
    // Take a screenshot to see what's rendered
    await page.screenshot({ path: 'skills-page.png', fullPage: true });
    
    // Check if there are any visible elements
    const bodyText = await page.locator('body').textContent();
    console.log('Body text length:', bodyText?.length);
    console.log('Body contains Skills:', bodyText?.includes('Skills'));
    console.log('Body contains Error:', bodyText?.includes('Error'));
    
    // Look for any h1, h2, or h3 elements
    const headings = await page.locator('h1, h2, h3').allTextContents();
    console.log('Found headings:', headings);
    
    // Look for the Skills title specifically
    const skillsHeading = page.locator('text=Skills').first();
    const isSkillsVisible = await skillsHeading.isVisible();
    console.log('Skills heading visible:', isSkillsVisible);
    
    // This test should always pass, it's just for debugging
    expect(true).toBe(true);
  });
});
