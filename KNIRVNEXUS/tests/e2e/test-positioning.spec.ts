import { test, expect } from '@playwright/test';

test.describe('UI Positioning Tests', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to the application
    await page.goto('http://localhost:8090');
    // Wait for the page to load
    await page.waitForLoadState('networkidle');
  });

  test('Console panel should slide out from left edge of CDE modal', async ({ page }) => {
    // Navigate to Network & Resources tab
    await page.click('button[value="system"]');

    // Click on a DVE node to open CDE modal
    await page.click('text=DVE-Node-1');

    // Click Console tool button
    await page.click('text=Console >> .. >> button:has-text("Show")');

    // Check that console modal is visible and positioned on the left of CDE modal
    const consoleModal = page.locator('text=Real-Time Console').locator('..').locator('..').locator('..');
    await expect(consoleModal).toBeVisible();

    // Verify it's positioned with right: 896px, so left = viewport.width - 1216px (896px + 320px panel width)
    const modalBox = await consoleModal.boundingBox();
    const viewport = page.viewportSize();
    expect(viewport).toBeTruthy();

    const expectedLeft = viewport!.width - 1216;
    expect(Math.abs(modalBox!.x - expectedLeft)).toBeLessThan(10);

    // Should be positioned at top: 100px
    expect(Math.abs(modalBox!.y - 100)).toBeLessThan(10);

    // Should have width of 320px (w-80)
    expect(Math.abs(modalBox!.width - 320)).toBeLessThan(10);
  });

  test('Policy editor should slide out from right edge of CDE modal', async ({ page }) => {
    // Navigate to Network & Resources tab
    await page.click('button[value="system"]');

    // Click on a DVE node to open CDE modal
    await page.click('text=DVE-Node-1');

    // Click Policy tool button
    await page.click('text=Policy >> .. >> button:has-text("Show")');

    // Check that policy modal is visible and positioned on the right
    const policyModal = page.locator('text=Security Policy').locator('..').locator('..').locator('..');
    await expect(policyModal).toBeVisible();

    // Verify it's positioned on the right side with correct left position
    const modalBox = await policyModal.boundingBox();
    const viewport = page.viewportSize();
    expect(viewport).toBeTruthy();

    // Modal should be positioned with right: 896px, so left = viewport.width - 1216px (896px + 320px panel width)
    const expectedLeft = viewport!.width - 1216;
    expect(Math.abs(modalBox!.x - expectedLeft)).toBeLessThan(10);

    // Should be positioned at top: 420px
    expect(Math.abs(modalBox!.y - 420)).toBeLessThan(10);

    // Should have width of 320px (w-80)
    expect(Math.abs(modalBox!.width - 320)).toBeLessThan(10);
  });

  test('DVE Solver should replace Connected NRVs panel on left side', async ({ page }) => {
   // Navigate to Network & Resources tab
   await page.click('button[value="system"]');

   // Click on a DVE node to open CDE modal
   await page.click('text=DVE-Node-1');

   // Click DVE Solver tool button
   await page.click('text=DVE Solver >> .. >> button:has-text("Open")');

   // Check that DVE Solver modal is visible and positioned on the left side
   const dveSolverModal = page.locator('text=DVE Solver').locator('..').locator('..').locator('..');
   await expect(dveSolverModal).toBeVisible();

   // Verify it's positioned on the left side replacing Connections panel
   const modalBox = await dveSolverModal.boundingBox();
   const viewport = page.viewportSize();
   expect(viewport).toBeTruthy();

   // Should be positioned at left: 0
   expect(Math.abs(modalBox!.x - 0)).toBeLessThan(10);

   // Should be positioned at top: 0
   expect(Math.abs(modalBox!.y - 0)).toBeLessThan(10);

   // Should have width of 320px (w-80)
   expect(Math.abs(modalBox!.width - 320)).toBeLessThan(10);
 });

  test('Rent button modal should be visible from Network & Resources view', async ({ page }) => {
    // Navigate to Network & Resources tab
    await page.click('button[value="system"]');
    
    // Click Rent button in DVE nodes panel
    await page.click('button:has-text("Rent")');
    
    // Check if modal is visible
    const modal = page.locator('[data-testid="dve-rental-modal"]');
    await expect(modal).toBeVisible();
    
    // Check if modal content is accessible
    const modalTitle = page.locator('h2:has-text("DVE Rental Management")');
    await expect(modalTitle).toBeVisible();
  });
});