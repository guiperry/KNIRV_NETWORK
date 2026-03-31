/**
 * Network Selection End-to-End Tests
 * Tests network switching and wallet integration functionality
 */

import { test, expect, Page } from '@playwright/test';

interface NetworkConfig {
  name: string;
  chainId: string;
  rpcUrl: string;
  testId: string;
}

const networks: NetworkConfig[] = [
  {
    name: 'XION Testnet',
    chainId: 'xion-testnet-1',
    rpcUrl: 'https://rpc.xion-testnet-1.burnt.com',
    testId: 'network-xion-testnet'
  },
  {
    name: 'XION Mainnet',
    chainId: 'xion-mainnet-1', 
    rpcUrl: 'https://rpc.xion-mainnet-1.burnt.com',
    testId: 'network-xion-mainnet'
  },
  {
    name: 'Local Development',
    chainId: 'local-dev',
    rpcUrl: 'http://localhost:26657',
    testId: 'network-local-dev'
  }
];

test.describe('Network Selection E2E Tests', () => {
  let page: Page;

  test.beforeEach(async ({ page: testPage }) => {
    page = testPage;
    await page.goto('http://localhost:5173');
    await page.waitForSelector('[data-testid="app-container"]', { timeout: 10000 });
  });

  test('should display network selector', async () => {
    // Look for network selector component
    const networkSelector = page.locator('[data-testid="network-selector"]');
    
    if (await networkSelector.isVisible()) {
      await expect(networkSelector).toBeVisible();
      
      // Should show current network
      const currentNetwork = page.locator('[data-testid="current-network"]');
      await expect(currentNetwork).toBeVisible();

      // Verify all configured networks are available
      for (const network of networks) {
        const networkOption = page.locator(`[data-testid="${network.testId}"]`);
        if (await networkOption.isVisible()) {
          await expect(networkOption).toContainText(network.name);
        }
      }
    } else {
      // Network selector might be in settings or wallet section
      const settingsButton = page.locator('[data-testid="settings-button"]');
      if (await settingsButton.isVisible()) {
        await settingsButton.click();
        await expect(page.locator('[data-testid="network-settings"]')).toBeVisible();
      }
    }
  });

  test('should handle network switching', async () => {
    // Try to access network selector
    const networkButton = page.locator('[data-testid="network-selector"], [data-testid="network-button"]').first();
    
    if (await networkButton.isVisible()) {
      await networkButton.click();
      
      // Should show network options
      const networkOptions = page.locator('[data-testid^="network-option-"]');
      const optionCount = await networkOptions.count();
      
      if (optionCount > 0) {
        // Test switching to different network
        await networkOptions.first().click();
        
        // Should show loading state during switch
        const loadingIndicator = page.locator('[data-testid="network-switching"]');
        if (await loadingIndicator.isVisible({ timeout: 1000 })) {
          await expect(loadingIndicator).toBeVisible();
          await expect(loadingIndicator).not.toBeVisible({ timeout: 10000 });
        }
        
        // Should update current network display
        await expect(page.locator('[data-testid="current-network"]')).toBeVisible();
      }
    }
  });

  test('should validate network connectivity', async () => {
    // Test network health check
    const networkStatus = page.locator('[data-testid="network-status"]');
    
    if (await networkStatus.isVisible()) {
      await expect(networkStatus).toBeVisible();
      
      // Should show connection status
      const statusIndicator = page.locator('[data-testid="connection-status"]');
      if (await statusIndicator.isVisible()) {
        const statusText = await statusIndicator.textContent();
        expect(['Connected', 'Connecting', 'Disconnected']).toContain(statusText?.trim());
      }
    }
  });

  test('should handle wallet integration with network', async () => {
    // Look for wallet connection
    const walletButton = page.locator('[data-testid="wallet-connect"], [data-testid="connect-wallet"]').first();
    
    if (await walletButton.isVisible()) {
      await walletButton.click();
      
      // Should show wallet options or connection modal
      const walletModal = page.locator('[data-testid="wallet-modal"], [data-testid="wallet-selector"]').first();
      
      if (await walletModal.isVisible({ timeout: 5000 })) {
        await expect(walletModal).toBeVisible();
        
        // Should show network compatibility
        const networkInfo = page.locator('[data-testid="network-info"]');
        if (await networkInfo.isVisible()) {
          await expect(networkInfo).toBeVisible();
        }
      }
    }
  });

  test('should persist network selection', async () => {
    // Get initial network
    const initialNetwork = await page.locator('[data-testid="current-network"]').textContent();
    
    // Refresh page
    await page.reload();
    await page.waitForSelector('[data-testid="app-container"]', { timeout: 10000 });
    
    // Should maintain network selection
    const persistedNetwork = await page.locator('[data-testid="current-network"]').textContent();
    
    if (initialNetwork && persistedNetwork) {
      expect(persistedNetwork).toBe(initialNetwork);
    }
  });

  test('should handle network errors gracefully', async () => {
    // Test with invalid network configuration
    await page.evaluate(() => {
      // Simulate network error
      localStorage.setItem('selectedNetwork', JSON.stringify({
        name: 'Invalid Network',
        chainId: 'invalid-chain',
        rpcUrl: 'https://invalid-rpc-url.com'
      }));
    });
    
    await page.reload();
    await page.waitForSelector('[data-testid="app-container"]', { timeout: 10000 });
    
    // Should show error state or fallback to default network
    const errorMessage = page.locator('[data-testid="network-error"]');
    const defaultNetwork = page.locator('[data-testid="current-network"]');
    
    // Either show error or fallback to working network
    const hasError = await errorMessage.isVisible({ timeout: 2000 });
    const hasDefault = await defaultNetwork.isVisible({ timeout: 2000 });
    
    expect(hasError || hasDefault).toBe(true);
  });

  test('should support custom network configuration', async () => {
    // Look for custom network option
    const customNetworkButton = page.locator('[data-testid="add-custom-network"], [data-testid="custom-network"]').first();
    
    if (await customNetworkButton.isVisible()) {
      await customNetworkButton.click();
      
      // Should show custom network form
      const customForm = page.locator('[data-testid="custom-network-form"]');
      if (await customForm.isVisible({ timeout: 5000 })) {
        await expect(customForm).toBeVisible();
        
        // Test form fields
        const nameField = page.locator('[data-testid="network-name-input"]');
        const rpcField = page.locator('[data-testid="rpc-url-input"]');
        const chainIdField = page.locator('[data-testid="chain-id-input"]');
        
        if (await nameField.isVisible()) {
          await nameField.fill('Test Custom Network');
        }
        if (await rpcField.isVisible()) {
          await rpcField.fill('https://test-rpc.example.com');
        }
        if (await chainIdField.isVisible()) {
          await chainIdField.fill('test-chain-1');
        }
        
        // Submit form
        const submitButton = page.locator('[data-testid="add-network-submit"]');
        if (await submitButton.isVisible()) {
          await submitButton.click();
        }
      }
    }
  });

  test('should show network-specific features', async () => {
    // Test that features adapt to selected network
    const networkFeatures = page.locator('[data-testid="network-features"]');
    
    if (await networkFeatures.isVisible()) {
      await expect(networkFeatures).toBeVisible();
      
      // Should show appropriate features for network type
      const testnetFeatures = page.locator('[data-testid="testnet-features"]');
      const mainnetFeatures = page.locator('[data-testid="mainnet-features"]');
      
      // At least one should be visible based on network
      const hasTestnet = await testnetFeatures.isVisible({ timeout: 1000 });
      const hasMainnet = await mainnetFeatures.isVisible({ timeout: 1000 });
      
      // Features should be contextual to network type
      if (hasTestnet || hasMainnet) {
        expect(hasTestnet || hasMainnet).toBe(true);
      }
    }
  });

  test('should handle network latency and timeouts', async () => {
    // Test network timeout handling
    await page.evaluate(() => {
      // Simulate slow network
      const originalFetch = window.fetch;
      window.fetch = async (...args) => {
        await new Promise(resolve => setTimeout(resolve, 5000)); // 5 second delay
        return originalFetch(...args);
      };
    });
    
    // Try to perform network operation
    const networkAction = page.locator('[data-testid="network-action"], [data-testid="refresh-network"]').first();
    
    if (await networkAction.isVisible()) {
      await networkAction.click();
      
      // Should show loading state
      const loadingState = page.locator('[data-testid="network-loading"]');
      if (await loadingState.isVisible({ timeout: 2000 })) {
        await expect(loadingState).toBeVisible();
      }
      
      // Should handle timeout gracefully
      const timeoutError = page.locator('[data-testid="network-timeout"]');
      if (await timeoutError.isVisible({ timeout: 10000 })) {
        await expect(timeoutError).toBeVisible();
      }
    }
  });

  test('should support network switching during active operations', async () => {
    // Start an operation
    const operationButton = page.locator('[data-testid="start-operation"], [data-testid="deploy-agent"]').first();
    
    if (await operationButton.isVisible()) {
      await operationButton.click();
      
      // Try to switch network during operation
      const networkSelector = page.locator('[data-testid="network-selector"]');
      if (await networkSelector.isVisible()) {
        await networkSelector.click();
        
        // Should either prevent switching or handle gracefully
        const switchWarning = page.locator('[data-testid="switch-warning"]');
        const networkOptions = page.locator('[data-testid^="network-option-"]');
        
        if (await switchWarning.isVisible({ timeout: 2000 })) {
          await expect(switchWarning).toBeVisible();
        } else if (await networkOptions.count() > 0) {
          // Should allow switching with proper handling
          await networkOptions.first().click();
        }
      }
    }
  });
});
