/**
 * Cross-Browser Compatibility Tests for KNIRVCONTROLLER
 * Tests core functionality across different browser environments
 */

import { test, expect, Browser, BrowserContext, Page } from '@playwright/test';

interface BrowserTestConfig {
  name: string;
  browserType: 'chromium' | 'firefox' | 'webkit';
  userAgent?: string;
  viewport?: { width: number; height: number };
}

const browsers: BrowserTestConfig[] = [
  {
    name: 'Chrome Desktop',
    browserType: 'chromium',
    viewport: { width: 1920, height: 1080 }
  },
  {
    name: 'Firefox Desktop',
    browserType: 'firefox',
    viewport: { width: 1920, height: 1080 }
  },
  {
    name: 'Safari Desktop',
    browserType: 'webkit',
    viewport: { width: 1920, height: 1080 }
  },
  {
    name: 'Chrome Mobile',
    browserType: 'chromium',
    userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.1.15',
    viewport: { width: 375, height: 667 }
  },
  {
    name: 'Safari Mobile',
    browserType: 'webkit',
    userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.1.15',
    viewport: { width: 375, height: 667 }
  }
];

// Core functionality tests across browsers
browsers.forEach(browserConfig => {
  test.describe(`Cross-Browser: ${browserConfig.name}`, () => {
    let browser: Browser;
    let context: BrowserContext;
    let page: Page;

    test.beforeAll(async ({ playwright }) => {
      browser = await playwright[browserConfig.browserType].launch();
      context = await browser.newContext({
        userAgent: browserConfig.userAgent,
        viewport: browserConfig.viewport
      });
      page = await context.newPage();
    });

    test.afterAll(async () => {
      await context.close();
      await browser.close();
    });

    test('should load application successfully', async () => {
      await page.goto('http://localhost:5173');
      
      // Wait for app to load
      await page.waitForSelector('[data-testid="app-container"]', { timeout: 10000 });
      
      // Check for critical elements
      await expect(page.locator('h1')).toBeVisible();
      await expect(page.locator('[data-testid="navigation"]')).toBeVisible();
    });

    test('should support PWA features', async () => {
      await page.goto('http://localhost:5173');
      
      // Check for PWA manifest
      const manifestLink = page.locator('link[rel="manifest"]');
      await expect(manifestLink).toHaveAttribute('href', '/manifest.json');
      
      // Check for service worker registration
      const swRegistration = await page.evaluate(() => {
        return 'serviceWorker' in navigator;
      });
      expect(swRegistration).toBe(true);
    });

    test('should handle touch gestures on mobile', async () => {
      if (browserConfig.viewport && browserConfig.viewport.width < 768) {
        await page.goto('http://localhost:5173');
        
        // Test touch interactions
        const touchElement = page.locator('[data-testid="touch-area"]').first();
        if (await touchElement.isVisible()) {
          await touchElement.tap();
          await expect(touchElement).toHaveClass(/active|touched/);
        }
      }
    });

    test('should support keyboard navigation', async () => {
      await page.goto('http://localhost:5173');
      
      // Test tab navigation
      await page.keyboard.press('Tab');
      const focusedElement = await page.evaluate(() => document.activeElement?.tagName);
      expect(['BUTTON', 'A', 'INPUT']).toContain(focusedElement);
      
      // Test keyboard shortcuts
      await page.keyboard.press('Control+k');
      // Should open keyboard shortcuts help or command palette
    });

    test('should handle WebAssembly support', async () => {
      await page.goto('http://localhost:5173');
      
      const wasmSupport = await page.evaluate(() => {
        return typeof WebAssembly === 'object' && typeof WebAssembly.instantiate === 'function';
      });
      expect(wasmSupport).toBe(true);
    });

    test('should support modern JavaScript features', async () => {
      await page.goto('http://localhost:5173');
      
      const modernFeatures = await page.evaluate(() => {
        return {
          asyncAwait: typeof (async () => {})().then === 'function',
          modules: typeof (window as any).import === 'function',
          fetch: typeof fetch === 'function',
          promises: typeof Promise === 'function',
          arrow: (() => true)(),
          destructuring: (() => { const [a] = [1]; return a === 1; })(),
          templateLiterals: `test` === 'test'
        };
      });
      
      Object.values(modernFeatures).forEach(supported => {
        expect(supported).toBe(true);
      });
    });

    test('should handle local storage and indexedDB', async () => {
      await page.goto('http://localhost:5173');
      
      const storageSupport = await page.evaluate(() => {
        return {
          localStorage: typeof localStorage === 'object',
          sessionStorage: typeof sessionStorage === 'object',
          indexedDB: typeof indexedDB === 'object'
        };
      });
      
      expect(storageSupport.localStorage).toBe(true);
      expect(storageSupport.sessionStorage).toBe(true);
      expect(storageSupport.indexedDB).toBe(true);
    });

    test('should support CSS Grid and Flexbox', async () => {
      await page.goto('http://localhost:5173');
      
      const cssSupport = await page.evaluate(() => {
        const testElement = document.createElement('div');
        testElement.style.display = 'grid';
        const gridSupport = testElement.style.display === 'grid';
        
        testElement.style.display = 'flex';
        const flexSupport = testElement.style.display === 'flex';
        
        return { grid: gridSupport, flex: flexSupport };
      });
      
      expect(cssSupport.grid).toBe(true);
      expect(cssSupport.flex).toBe(true);
    });

    test('should handle responsive design', async () => {
      await page.goto('http://localhost:5173');
      
      // Test different viewport sizes
      const viewports = [
        { width: 320, height: 568 }, // Mobile
        { width: 768, height: 1024 }, // Tablet
        { width: 1920, height: 1080 } // Desktop
      ];
      
      for (const viewport of viewports) {
        await page.setViewportSize(viewport);
        await page.waitForTimeout(500); // Allow layout to adjust
        
        // Check that layout adapts
        const bodyWidth = await page.evaluate(() => document.body.offsetWidth);
        expect(bodyWidth).toBeLessThanOrEqual(viewport.width);
      }
    });

    test('should handle error boundaries gracefully', async () => {
      await page.goto('http://localhost:5173');
      
      // Trigger an error and check error boundary
      await page.evaluate(() => {
        // Simulate an error
        window.dispatchEvent(new ErrorEvent('error', {
          error: new Error('Test error for error boundary'),
          message: 'Test error for error boundary'
        }));
      });
      
      // Error boundary should catch and display error
      await page.waitForTimeout(1000);
      // App should still be functional
      await expect(page.locator('body')).toBeVisible();
    });
  });
});

// Performance tests across browsers
test.describe('Cross-Browser Performance', () => {
  browsers.forEach(browserConfig => {
    test(`Performance: ${browserConfig.name}`, async ({ playwright }) => {
      const browser = await playwright[browserConfig.browserType].launch();
      const context = await browser.newContext({
        userAgent: browserConfig.userAgent,
        viewport: browserConfig.viewport
      });
      const page = await context.newPage();
      
      // Measure page load performance
      const startTime = Date.now();
      await page.goto('http://localhost:5173');
      await page.waitForSelector('[data-testid="app-container"]', { timeout: 10000 });
      const loadTime = Date.now() - startTime;
      
      // Performance should be reasonable across browsers
      expect(loadTime).toBeLessThan(10000); // 10 seconds max
      
      // Check for memory leaks
      const initialMemory = await page.evaluate(() => {
        return (performance as any).memory?.usedJSHeapSize || 0;
      });
      
      // Perform some operations
      for (let i = 0; i < 10; i++) {
        await page.click('body');
        await page.waitForTimeout(100);
      }
      
      const finalMemory = await page.evaluate(() => {
        return (performance as any).memory?.usedJSHeapSize || 0;
      });
      
      // Memory shouldn't grow excessively
      if (initialMemory > 0 && finalMemory > 0) {
        const memoryGrowth = finalMemory - initialMemory;
        expect(memoryGrowth).toBeLessThan(50 * 1024 * 1024); // 50MB max growth
      }
      
      await context.close();
      await browser.close();
    });
  });
});
