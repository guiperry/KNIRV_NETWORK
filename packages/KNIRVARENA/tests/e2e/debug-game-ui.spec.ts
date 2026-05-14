import { test, expect } from '@playwright/test';

test('debug game ui and menu visibility', async ({ page }) => {
  page.on('console', msg => console.log('BROWSER LOG:', msg.text()));
  test.setTimeout(60000);
  // Navigate to the app
  await page.goto('http://localhost:3000');

  // Wait for the app to load
  await page.waitForSelector('#root');
  
  // Manually hide loading screen if it's stuck
  await page.evaluate(() => {
    const ls = document.getElementById('loading-screen');
    if (ls) ls.style.display = 'none';
  });

  // Wait for the game to initialize
  await page.waitForTimeout(5000);

  // Take a screenshot
  await page.screenshot({ path: 'test-results/game-ui-debug.png' });

  // Check if GameMenu elements are present
  const bootTerminal = page.locator('text=ERGO_BOOT');
  const ergoLogo = page.locator('text=ERGO');
  
  console.log('Boot Terminal visible:', await bootTerminal.isVisible());
  console.log('ERGO Logo visible:', await ergoLogo.isVisible());

  // Check for the 3D canvas
  const canvas = page.locator('canvas');
  console.log('Canvas present:', await canvas.count() > 0);

  // Check z-index and positioning of the menu if it exists
  const menu = page.locator('.absolute.inset-0.flex.items-center.justify-center.z-50');
  if (await menu.isVisible()) {
    const box = await menu.boundingBox();
    console.log('Menu Box:', box);
    const zIndex = await menu.evaluate(el => window.getComputedStyle(el).zIndex);
    console.log('Menu z-index:', zIndex);
  } else {
    console.log('Menu container NOT visible');
  }
});
