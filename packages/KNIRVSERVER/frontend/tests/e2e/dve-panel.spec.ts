import { test, expect } from '@playwright/test';

const FRONTEND_URL = 'http://localhost:8090';

test.describe('DVE Workspace UI', () => {
  test('should open Metadata Panel when "INITIALIZE SECURE SHELL" button is clicked', async ({ page }) => {
    await page.goto(FRONTEND_URL, { waitUntil: 'domcontentloaded' });

    // Wait for the DVE Workspace to be visible
    // Assuming DVE Workspace is immediately visible or can be triggered
    // If not, we might need to add steps to open it first.
    // For now, let's assume it's visible.

    // Find the "INITIALIZE SECURE SHELL" button
    const initializeButton = page.getByRole('button', { name: 'INITIALIZE SECURE SHELL' });
    await expect(initializeButton).toBeVisible();

    // Click the button
    await initializeButton.click();

    // Assert that the Metadata Panel is visible
    const metadataPanel = page.locator('section', { hasText: 'Metadata Panel' }); // Adjust selector as needed
    await expect(metadataPanel).toBeVisible();
    
    // Also check for the title of the Metadata Panel
    await expect(page.getByText('Metadata Panel')).toBeVisible();

    // Verify hover state (optional, for debugging)
    // await initializeButton.hover();
    // await page.screenshot({ path: 'screenshot-hover.png' });
  });
});
