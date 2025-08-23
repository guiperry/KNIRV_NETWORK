// Comprehensive End-to-End Tests for KNIRVWALLET - Complete Wallet Workflow
const { chromium, firefox, webkit } = require('playwright');
const { expect } = require('@playwright/test');

describe('KNIRVWALLET End-to-End Wallet Workflow', () => {
  let browser;
  let context;
  let page;
  let mobileContext;
  let mobilePage;

  const TEST_CONFIG = {
    GATEWAY_URL: process.env.GATEWAY_URL || 'http://localhost:8000',
    WALLET_URL: process.env.WALLET_URL || 'http://localhost:8083',
    MOBILE_VIEWPORT: { width: 375, height: 812 },
    DESKTOP_VIEWPORT: { width: 1280, height: 720 },
    TEST_TIMEOUT: 60000,
    NETWORK_TIMEOUT: 30000
  };

  const TEST_DATA = {
    MNEMONIC: 'abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about',
    WALLET_NAME: 'E2E Test Wallet',
    RECIPIENT_ADDRESS: 'xion1234567890abcdef1234567890abcdef12345678',
    TRANSFER_AMOUNT: '100000',
    SKILL_ID: 'skill-e2e-test-001',
    SKILL_AMOUNT: '50000'
  };

  beforeAll(async () => {
    // Launch browser with appropriate configuration
    browser = await chromium.launch({
      headless: process.env.CI === 'true',
      slowMo: process.env.CI === 'true' ? 0 : 100
    });

    // Create desktop context
    context = await browser.newContext({
      viewport: TEST_CONFIG.DESKTOP_VIEWPORT,
      permissions: ['clipboard-read', 'clipboard-write']
    });

    page = await context.newPage();

    // Create mobile context for cross-platform testing
    mobileContext = await browser.newContext({
      viewport: TEST_CONFIG.MOBILE_VIEWPORT,
      userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.1.15',
      permissions: ['clipboard-read', 'clipboard-write']
    });

    mobilePage = await mobileContext.newPage();

    // Set timeouts
    page.setDefaultTimeout(TEST_CONFIG.TEST_TIMEOUT);
    mobilePage.setDefaultTimeout(TEST_CONFIG.TEST_TIMEOUT);

    console.log('E2E Test Environment initialized');
  });

  afterAll(async () => {
    if (browser) {
      await browser.close();
    }
    console.log('E2E Test Environment cleanup completed');
  });

  describe('Browser Wallet Extension Workflow', () => {
    test('should load KNIRVWALLET extension interface', async () => {
      await page.goto(`${TEST_CONFIG.GATEWAY_URL}/wallet`);
      
      // Wait for wallet interface to load
      await page.waitForSelector('[data-testid="wallet-interface"]', { timeout: TEST_CONFIG.NETWORK_TIMEOUT });
      
      // Verify main components are present
      await expect(page.locator('h1')).toContainText('KNIRV Wallet');
      await expect(page.locator('[data-testid="create-wallet-btn"]')).toBeVisible();
      await expect(page.locator('[data-testid="import-wallet-btn"]')).toBeVisible();
    });

    test('should create HD wallet from mnemonic', async () => {
      await page.goto(`${TEST_CONFIG.GATEWAY_URL}/wallet`);
      
      // Click import wallet button
      await page.click('[data-testid="import-wallet-btn"]');
      
      // Wait for import modal
      await page.waitForSelector('[data-testid="import-wallet-modal"]');
      
      // Fill wallet details
      await page.fill('[data-testid="wallet-name-input"]', TEST_DATA.WALLET_NAME);
      await page.fill('[data-testid="mnemonic-input"]', TEST_DATA.MNEMONIC);
      
      // Submit wallet creation
      await page.click('[data-testid="import-wallet-submit"]');
      
      // Wait for wallet to be created
      await page.waitForSelector('[data-testid="wallet-dashboard"]', { timeout: TEST_CONFIG.NETWORK_TIMEOUT });
      
      // Verify wallet is created
      await expect(page.locator('[data-testid="wallet-address"]')).toBeVisible();
      await expect(page.locator('[data-testid="wallet-balance"]')).toBeVisible();
      
      console.log('HD wallet created successfully');
    });

    test('should display wallet balance and account information', async () => {
      // Assuming wallet is already created from previous test
      await page.waitForSelector('[data-testid="wallet-dashboard"]');
      
      // Check balance display
      const balanceElement = page.locator('[data-testid="wallet-balance"]');
      await expect(balanceElement).toBeVisible();
      
      // Check address display
      const addressElement = page.locator('[data-testid="wallet-address"]');
      await expect(addressElement).toBeVisible();
      
      // Verify address format (should start with appropriate prefix)
      const address = await addressElement.textContent();
      expect(address).toMatch(/^(g1|xion1|0x)/);
      
      console.log(`Wallet address: ${address}`);
    });

    test('should refresh wallet balance', async () => {
      await page.waitForSelector('[data-testid="wallet-dashboard"]');
      
      // Get initial balance
      const initialBalance = await page.locator('[data-testid="wallet-balance"]').textContent();
      
      // Click refresh button
      await page.click('[data-testid="refresh-balance-btn"]');
      
      // Wait for refresh to complete
      await page.waitForSelector('[data-testid="balance-loading"]', { state: 'hidden', timeout: TEST_CONFIG.NETWORK_TIMEOUT });
      
      // Verify balance is still displayed (might be same value)
      await expect(page.locator('[data-testid="wallet-balance"]')).toBeVisible();
      
      console.log('Balance refresh completed');
    });

    test('should sign and send transaction', async () => {
      await page.waitForSelector('[data-testid="wallet-dashboard"]');
      
      // Click send button
      await page.click('[data-testid="send-btn"]');
      
      // Wait for send modal
      await page.waitForSelector('[data-testid="send-transaction-modal"]');
      
      // Fill transaction details
      await page.fill('[data-testid="recipient-input"]', TEST_DATA.RECIPIENT_ADDRESS);
      await page.fill('[data-testid="amount-input"]', TEST_DATA.TRANSFER_AMOUNT);
      await page.fill('[data-testid="memo-input"]', 'E2E test transaction');
      
      // Submit transaction
      await page.click('[data-testid="send-transaction-submit"]');
      
      // Wait for transaction confirmation
      await page.waitForSelector('[data-testid="transaction-success"]', { timeout: TEST_CONFIG.NETWORK_TIMEOUT });
      
      // Verify transaction hash is displayed
      await expect(page.locator('[data-testid="transaction-hash"]')).toBeVisible();
      
      const txHash = await page.locator('[data-testid="transaction-hash"]').textContent();
      expect(txHash).toMatch(/^0x[a-fA-F0-9]{64}$/);
      
      console.log(`Transaction sent: ${txHash}`);
    });
  });

  describe('React Native Mobile App Workflow', () => {
    test('should load mobile wallet interface', async () => {
      // Navigate to mobile wallet interface
      await mobilePage.goto(`${TEST_CONFIG.GATEWAY_URL}/mobile-wallet`);
      
      // Wait for mobile interface to load
      await mobilePage.waitForSelector('[data-testid="mobile-wallet-interface"]', { timeout: TEST_CONFIG.NETWORK_TIMEOUT });
      
      // Verify mobile-specific components
      await expect(mobilePage.locator('[data-testid="mobile-header"]')).toBeVisible();
      await expect(mobilePage.locator('[data-testid="mobile-navigation"]')).toBeVisible();
    });

    test('should create XION meta account', async () => {
      await mobilePage.goto(`${TEST_CONFIG.GATEWAY_URL}/mobile-wallet`);
      
      // Navigate to XION Meta tab
      await mobilePage.click('[data-testid="xion-meta-tab"]');
      
      // Wait for XION interface
      await mobilePage.waitForSelector('[data-testid="xion-meta-dashboard"]');
      
      // Click create meta account
      await mobilePage.click('[data-testid="create-meta-account-btn"]');
      
      // Fill account details
      await mobilePage.fill('[data-testid="meta-account-name-input"]', 'E2E Meta Account');
      
      // Submit creation
      await mobilePage.click('[data-testid="create-meta-account-submit"]');
      
      // Wait for account creation
      await mobilePage.waitForSelector('[data-testid="meta-account-created"]', { timeout: TEST_CONFIG.NETWORK_TIMEOUT });
      
      // Verify meta account is displayed
      await expect(mobilePage.locator('[data-testid="meta-account-address"]')).toBeVisible();
      
      console.log('XION meta account created successfully');
    });

    test('should request NRN from faucet', async () => {
      // Assuming meta account is created
      await mobilePage.waitForSelector('[data-testid="xion-meta-dashboard"]');
      
      // Click faucet button
      await mobilePage.click('[data-testid="faucet-request-btn"]');
      
      // Wait for faucet request to complete
      await mobilePage.waitForSelector('[data-testid="faucet-success"]', { timeout: TEST_CONFIG.NETWORK_TIMEOUT });
      
      // Verify success message
      await expect(mobilePage.locator('[data-testid="faucet-success"]')).toContainText('Faucet request successful');
      
      // Wait for balance to update
      await mobilePage.waitForTimeout(5000);
      
      // Verify NRN balance is updated
      const nrnBalance = await mobilePage.locator('[data-testid="nrn-balance"]').textContent();
      expect(parseFloat(nrnBalance)).toBeGreaterThan(0);
      
      console.log(`NRN balance after faucet: ${nrnBalance}`);
    });

    test('should invoke skill with NRN burn', async () => {
      await mobilePage.waitForSelector('[data-testid="xion-meta-dashboard"]');
      
      // Click invoke skill button
      await mobilePage.click('[data-testid="invoke-skill-btn"]');
      
      // Wait for skill modal
      await mobilePage.waitForSelector('[data-testid="skill-invocation-modal"]');
      
      // Fill skill details
      await mobilePage.fill('[data-testid="skill-id-input"]', TEST_DATA.SKILL_ID);
      await mobilePage.fill('[data-testid="skill-amount-input"]', TEST_DATA.SKILL_AMOUNT);
      
      // Submit skill invocation
      await mobilePage.click('[data-testid="invoke-skill-submit"]');
      
      // Wait for skill invocation to complete
      await mobilePage.waitForSelector('[data-testid="skill-invocation-success"]', { timeout: TEST_CONFIG.NETWORK_TIMEOUT });
      
      // Verify success message and transaction hash
      await expect(mobilePage.locator('[data-testid="skill-invocation-success"]')).toBeVisible();
      await expect(mobilePage.locator('[data-testid="skill-transaction-hash"]')).toBeVisible();
      
      console.log('Skill invocation completed successfully');
    });
  });

  describe('Cross-Platform Wallet Synchronization', () => {
    test('should generate QR code for wallet sync', async () => {
      // On mobile, initiate sync
      await mobilePage.goto(`${TEST_CONFIG.GATEWAY_URL}/mobile-wallet`);
      await mobilePage.click('[data-testid="sync-wallet-btn"]');
      
      // Wait for QR code generation
      await mobilePage.waitForSelector('[data-testid="sync-qr-code"]', { timeout: TEST_CONFIG.NETWORK_TIMEOUT });
      
      // Verify QR code is displayed
      await expect(mobilePage.locator('[data-testid="sync-qr-code"]')).toBeVisible();
      
      // Get sync session ID from QR code data
      const qrData = await mobilePage.getAttribute('[data-testid="sync-qr-code"]', 'data-session-id');
      expect(qrData).toBeTruthy();
      
      console.log(`Sync session created: ${qrData}`);
    });

    test('should scan QR code and establish sync connection', async () => {
      // Simulate QR code scanning on browser
      await page.goto(`${TEST_CONFIG.GATEWAY_URL}/wallet`);
      await page.click('[data-testid="scan-qr-btn"]');
      
      // Wait for QR scanner
      await page.waitForSelector('[data-testid="qr-scanner"]');
      
      // Simulate successful QR scan (in real test, this would use camera)
      // For E2E test, we'll trigger the sync directly
      await page.evaluate(() => {
        window.triggerSyncConnection('test-session-id', 'test-encryption-key');
      });
      
      // Wait for sync connection established
      await page.waitForSelector('[data-testid="sync-connected"]', { timeout: TEST_CONFIG.NETWORK_TIMEOUT });
      
      // Verify sync status
      await expect(page.locator('[data-testid="sync-status"]')).toContainText('Connected');
      
      console.log('Cross-platform sync connection established');
    });

    test('should synchronize wallet data between platforms', async () => {
      // Assuming sync connection is established
      await page.waitForSelector('[data-testid="sync-connected"]');
      
      // Trigger wallet data sync from mobile
      await mobilePage.click('[data-testid="sync-wallet-data-btn"]');
      
      // Wait for sync to complete
      await mobilePage.waitForSelector('[data-testid="sync-complete"]', { timeout: TEST_CONFIG.NETWORK_TIMEOUT });
      
      // Verify data is synchronized on browser
      await page.waitForSelector('[data-testid="synced-wallet-data"]', { timeout: TEST_CONFIG.NETWORK_TIMEOUT });
      
      // Check that wallet data is present on browser
      await expect(page.locator('[data-testid="synced-accounts"]')).toBeVisible();
      await expect(page.locator('[data-testid="synced-preferences"]')).toBeVisible();
      
      console.log('Wallet data synchronized successfully');
    });
  });

  describe('Error Handling and Edge Cases', () => {
    test('should handle network connectivity issues', async () => {
      // Simulate network offline
      await page.context().setOffline(true);
      
      // Try to perform an operation
      await page.click('[data-testid="refresh-balance-btn"]');
      
      // Wait for error message
      await page.waitForSelector('[data-testid="network-error"]', { timeout: 10000 });
      
      // Verify error message is displayed
      await expect(page.locator('[data-testid="network-error"]')).toContainText('Network error');
      
      // Restore network
      await page.context().setOffline(false);
      
      // Verify recovery
      await page.click('[data-testid="retry-btn"]');
      await page.waitForSelector('[data-testid="network-error"]', { state: 'hidden' });
      
      console.log('Network error handling verified');
    });

    test('should handle invalid transaction inputs', async () => {
      await page.goto(`${TEST_CONFIG.GATEWAY_URL}/wallet`);
      await page.click('[data-testid="send-btn"]');
      
      // Enter invalid recipient address
      await page.fill('[data-testid="recipient-input"]', 'invalid-address');
      await page.fill('[data-testid="amount-input"]', TEST_DATA.TRANSFER_AMOUNT);
      
      // Try to submit
      await page.click('[data-testid="send-transaction-submit"]');
      
      // Wait for validation error
      await page.waitForSelector('[data-testid="validation-error"]');
      
      // Verify error message
      await expect(page.locator('[data-testid="validation-error"]')).toContainText('Invalid address');
      
      console.log('Input validation working correctly');
    });

    test('should handle insufficient balance scenarios', async () => {
      await page.goto(`${TEST_CONFIG.GATEWAY_URL}/wallet`);
      await page.click('[data-testid="send-btn"]');
      
      // Enter amount larger than balance
      await page.fill('[data-testid="recipient-input"]', TEST_DATA.RECIPIENT_ADDRESS);
      await page.fill('[data-testid="amount-input"]', '999999999999');
      
      // Try to submit
      await page.click('[data-testid="send-transaction-submit"]');
      
      // Wait for insufficient balance error
      await page.waitForSelector('[data-testid="insufficient-balance-error"]');
      
      // Verify error message
      await expect(page.locator('[data-testid="insufficient-balance-error"]')).toContainText('Insufficient balance');
      
      console.log('Insufficient balance handling verified');
    });
  });

  describe('Performance and Responsiveness', () => {
    test('should load wallet interface within acceptable time', async () => {
      const startTime = Date.now();
      
      await page.goto(`${TEST_CONFIG.GATEWAY_URL}/wallet`);
      await page.waitForSelector('[data-testid="wallet-interface"]');
      
      const loadTime = Date.now() - startTime;
      
      // Wallet should load within 5 seconds
      expect(loadTime).toBeLessThan(5000);
      
      console.log(`Wallet interface loaded in ${loadTime}ms`);
    });

    test('should handle multiple concurrent operations', async () => {
      await page.goto(`${TEST_CONFIG.GATEWAY_URL}/wallet`);
      
      // Start multiple operations concurrently
      const operations = [
        page.click('[data-testid="refresh-balance-btn"]'),
        page.click('[data-testid="transaction-history-btn"]'),
        page.click('[data-testid="settings-btn"]')
      ];
      
      // Wait for all operations to complete
      await Promise.all(operations);
      
      // Verify all operations completed successfully
      await expect(page.locator('[data-testid="wallet-dashboard"]')).toBeVisible();
      
      console.log('Concurrent operations handled successfully');
    });
  });

  describe('Security and Privacy', () => {
    test('should not expose sensitive data in DOM', async () => {
      await page.goto(`${TEST_CONFIG.GATEWAY_URL}/wallet`);
      
      // Check that private keys and mnemonics are not in DOM
      const pageContent = await page.content();
      
      expect(pageContent).not.toContain(TEST_DATA.MNEMONIC);
      expect(pageContent).not.toContain('private');
      expect(pageContent).not.toContain('seed');
      
      console.log('Sensitive data protection verified');
    });

    test('should clear sensitive data on logout', async () => {
      await page.goto(`${TEST_CONFIG.GATEWAY_URL}/wallet`);
      
      // Perform logout
      await page.click('[data-testid="logout-btn"]');
      
      // Wait for logout to complete
      await page.waitForSelector('[data-testid="login-screen"]');
      
      // Verify wallet data is cleared
      const localStorage = await page.evaluate(() => JSON.stringify(localStorage));
      expect(localStorage).not.toContain('wallet');
      expect(localStorage).not.toContain('mnemonic');
      
      console.log('Logout data clearing verified');
    });
  });
});
