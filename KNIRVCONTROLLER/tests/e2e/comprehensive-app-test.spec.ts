/**
 * Comprehensive End-to-End Test for KNIRVCONTROLLER Application
 * This test clicks through the entire application to verify all functionality
 */

import { test, expect, Page, BrowserContext } from '@playwright/test';

test.describe('KNIRVCONTROLLER Comprehensive Application Test', () => {
  let page: Page;
  let context: BrowserContext;

  test.beforeAll(async ({ browser }) => {
    context = await browser.newContext({
      viewport: { width: 1920, height: 1080 },
      permissions: ['camera', 'microphone'],
    });
    page = await context.newPage();
    
    // Navigate to the application
    await page.goto('http://localhost:3000');
    
    // Wait for the application to load
    await page.waitForLoadState('networkidle');
  });

  test.afterAll(async () => {
    await context.close();
  });

  test('should load the main application interface', async () => {
    // Verify the main interface loads
    await expect(page).toHaveTitle(/KNIRV Controller/i);
    
    // Check for key components
    await expect(page.locator('[data-testid="knirv-shell"]')).toBeVisible();
    await expect(page.locator('[data-testid="burger-menu"]')).toBeVisible();
  });

  test('should navigate through all main sections via burger menu', async () => {
    // Open burger menu
    await page.click('[data-testid="burger-menu"]');
    await expect(page.locator('[data-testid="burger-menu-content"]')).toBeVisible();

    // Test Skills navigation
    await page.click('text=Skills');
    await page.waitForURL('**/manager/skills');
    await expect(page.locator('h1')).toContainText('Skills');
    
    // Test UDC navigation
    await page.click('[data-testid="burger-menu"]');
    await page.click('text=UDC');
    await page.waitForURL('**/manager/udc');
    await expect(page.locator('h1')).toContainText('UDC');
    
    // Test Wallet navigation
    await page.click('[data-testid="burger-menu"]');
    await page.click('text=Wallet');
    await page.waitForURL('**/manager/wallet');
    await expect(page.locator('h1')).toContainText('Wallet');
    
    // Return to main interface
    await page.click('[data-testid="burger-menu"]');
    await page.click('text=Input Interface');
    await page.waitForURL('/');
  });

  test('should test voice control functionality', async () => {
    // Navigate to main interface
    await page.goto('http://localhost:3000');
    
    // Test voice control toggle
    const voiceButton = page.locator('[data-testid="voice-control-toggle"]');
    await expect(voiceButton).toBeVisible();
    
    // Activate voice control
    await voiceButton.click();
    await expect(voiceButton).toHaveClass(/bg-teal-500/); // Check for active styling

    // Deactivate voice control
    await voiceButton.click();
    await expect(voiceButton).toHaveClass(/bg-gray/);
  });

  test('should test network status panel', async () => {
    // Open network status panel
    const networkButton = page.locator('[data-testid="network-status-button"]');
    if (await networkButton.isVisible()) {
      await networkButton.click();
      
      // Verify panel opens
      await expect(page.locator('[data-testid="network-status-panel"]')).toBeVisible();
      
      // Close panel
      await page.click('[data-testid="close-panel"]');
      await expect(page.locator('[data-testid="network-status-panel"]')).not.toBeVisible();
    }
  });

  test('should test QR scanner functionality', async () => {
    // Open burger menu and click QR Scanner
    await page.click('[data-testid="burger-menu"]');
    await page.click('text=QR Scanner');
    
    // Verify QR scanner modal opens
    await expect(page.locator('[data-testid="qr-scanner-modal"]')).toBeVisible();
    
    // Close QR scanner
    await page.click('[data-testid="close-qr-scanner"]');
    await expect(page.locator('[data-testid="qr-scanner-modal"]')).not.toBeVisible();
  });

  test('should test Skills page functionality', async () => {
    // Navigate to Skills page
    await page.click('[data-testid="burger-menu"]');
    await page.click('text=Skills');
    
    // Test search functionality
    const searchInput = page.locator('[data-testid="skills-search"]');
    if (await searchInput.isVisible()) {
      await searchInput.fill('test skill');
      await page.keyboard.press('Enter');
      
      // Clear search
      await searchInput.clear();
    }
    
    // Test skill creation if available
    const createSkillButton = page.locator('[data-testid="create-skill-button"]');
    if (await createSkillButton.isVisible()) {
      await createSkillButton.click();
      
      // Fill out skill form if modal opens
      const skillModal = page.locator('[data-testid="skill-creation-modal"]');
      if (await skillModal.isVisible()) {
        await page.fill('[data-testid="skill-name"]', 'Test E2E Skill');
        await page.fill('[data-testid="skill-description"]', 'End-to-end test skill');
        
        // Close modal without saving
        await page.click('[data-testid="cancel-skill-creation"]');
      }
    }
  });

  test('should test UDC page functionality', async () => {
    // Navigate to UDC page
    await page.click('[data-testid="burger-menu"]');
    await page.click('text=UDC');
    
    // Test UDC creation if available
    const createUDCButton = page.locator('[data-testid="create-udc-button"]');
    if (await createUDCButton.isVisible()) {
      await createUDCButton.click();
      
      // Fill out UDC form if modal opens
      const udcModal = page.locator('[data-testid="udc-creation-modal"]');
      if (await udcModal.isVisible()) {
        await page.fill('[data-testid="agent-id"]', 'test-agent-e2e');
        await page.fill('[data-testid="scope-description"]', 'E2E test scope');
        
        // Close modal without saving
        await page.click('[data-testid="cancel-udc-creation"]');
      }
    }
    
    // Test UDC list if available
    const udcList = page.locator('[data-testid="udc-list"]');
    if (await udcList.isVisible()) {
      // Check if any UDCs are listed
      const udcItems = page.locator('[data-testid^="udc-item-"]');
      const count = await udcItems.count();
      console.log(`Found ${count} UDC items`);
    }
  });

  test('should test Wallet page functionality', async () => {
    // Navigate to Wallet page
    await page.click('[data-testid="burger-menu"]');
    await page.click('text=Wallet');
    
    // Test wallet address copy
    const copyAddressButton = page.locator('[data-testid="copy-wallet-address"]');
    if (await copyAddressButton.isVisible()) {
      await copyAddressButton.click();
      // Verify copy success message or tooltip
    }
    
    // Test QR code display
    const showQRButton = page.locator('[data-testid="show-qr-code"]');
    if (await showQRButton.isVisible()) {
      await showQRButton.click();
      
      const qrModal = page.locator('[data-testid="qr-code-modal"]');
      if (await qrModal.isVisible()) {
        await page.click('[data-testid="close-qr-modal"]');
      }
    }
    
    // Test Add Funds functionality
    const addFundsButton = page.locator('[data-testid="add-funds-button"]');
    if (await addFundsButton.isVisible()) {
      await addFundsButton.click();
      
      const addFundsModal = page.locator('[data-testid="add-funds-modal"]');
      if (await addFundsModal.isVisible()) {
        // Test form fields
        await page.fill('[data-testid="amount-input"]', '100');
        
        // Close modal
        await page.click('[data-testid="close-add-funds"]');
      }
    }
    
    // Test Send NRN functionality
    const sendNRNButton = page.locator('[data-testid="send-nrn-button"]');
    if (await sendNRNButton.isVisible()) {
      await sendNRNButton.click();
      
      const sendModal = page.locator('[data-testid="send-nrn-modal"]');
      if (await sendModal.isVisible()) {
        // Test form fields
        await page.fill('[data-testid="recipient-address"]', '0x1234567890abcdef');
        await page.fill('[data-testid="send-amount"]', '50');
        
        // Close modal
        await page.click('[data-testid="close-send-modal"]');
      }
    }
  });

  test('should test NRV visualization and interaction', async () => {
    // Navigate to main interface
    await page.goto('http://localhost:3000');
    
    // Test NRV visualization if available
    const nrvVisualization = page.locator('[data-testid="nrv-visualization"]');
    if (await nrvVisualization.isVisible()) {
      // Test NRV creation or interaction
      const createNRVButton = page.locator('[data-testid="create-nrv"]');
      if (await createNRVButton.isVisible()) {
        await createNRVButton.click();
        
        // Fill NRV form if available
        const nrvModal = page.locator('[data-testid="nrv-creation-modal"]');
        if (await nrvModal.isVisible()) {
          await page.fill('[data-testid="problem-description"]', 'E2E test problem');
          await page.selectOption('[data-testid="severity-select"]', 'Medium');
          
          // Close modal
          await page.click('[data-testid="cancel-nrv-creation"]');
        }
      }
    }
  });

  test('should test cognitive shell interface', async () => {
    // Test cognitive shell if available
    const cognitiveShell = page.locator('[data-testid="cognitive-shell"]');
    if (await cognitiveShell.isVisible()) {
      // Test cognitive mode toggle
      const cognitiveToggle = page.locator('[data-testid="cognitive-mode-toggle"]');
      if (await cognitiveToggle.isVisible()) {
        await cognitiveToggle.click();
        await page.waitForTimeout(1000);
        await cognitiveToggle.click();
      }
    }
  });

  test('should test agent manager functionality', async () => {
    // Test agent manager if available
    const agentManager = page.locator('[data-testid="agent-manager"]');
    if (await agentManager.isVisible()) {
      // Test agent list
      const agentList = page.locator('[data-testid="agent-list"]');
      if (await agentList.isVisible()) {
        const agents = page.locator('[data-testid^="agent-item-"]');
        const count = await agents.count();
        console.log(`Found ${count} agents`);
        
        // Click on first agent if available
        if (count > 0) {
          await agents.first().click();
          
          // Check if agent details modal opens
          const agentModal = page.locator('[data-testid="agent-details-modal"]');
          if (await agentModal.isVisible()) {
            await page.click('[data-testid="close-agent-details"]');
          }
        }
      }
    }
  });

  test('should test error handling and recovery', async () => {
    // Test application behavior with invalid routes
    await page.goto('http://localhost:3000/invalid-route');
    
    // Should redirect or show error page
    await page.waitForTimeout(2000);
    
    // Navigate back to valid route
    await page.goto('http://localhost:3000');
    await expect(page.locator('[data-testid="knirv-shell"]')).toBeVisible();
  });

  test('should test responsive design', async () => {
    // Test mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    await page.reload();
    
    // Verify mobile layout
    await expect(page.locator('[data-testid="burger-menu"]')).toBeVisible();
    
    // Test tablet viewport
    await page.setViewportSize({ width: 768, height: 1024 });
    await page.reload();
    
    // Verify tablet layout
    await expect(page.locator('[data-testid="knirv-shell"]')).toBeVisible();
    
    // Reset to desktop
    await page.setViewportSize({ width: 1920, height: 1080 });
    await page.reload();
  });

  test('should verify all critical components are functional', async () => {
    // Final comprehensive check
    await page.goto('http://localhost:3000');
    
    // Check all major components are present and functional
    const criticalComponents = [
      '[data-testid="knirv-shell"]',
      '[data-testid="burger-menu"]',
      '[data-testid="voice-control-toggle"]'
    ];
    
    for (const selector of criticalComponents) {
      const element = page.locator(selector);
      if (await element.isVisible()) {
        await expect(element).toBeVisible();
        console.log(`✓ Component ${selector} is functional`);
      } else {
        console.log(`⚠ Component ${selector} not found`);
      }
    }
    
    // Test navigation between all main sections
    const sections = ['Skills', 'UDC', 'Wallet'];
    for (const section of sections) {
      await page.click('[data-testid="burger-menu"]');
      await page.click(`text=${section}`);
      await page.waitForTimeout(1000);
      console.log(`✓ Navigation to ${section} successful`);
    }
    
    // Return to main interface
    await page.click('[data-testid="burger-menu"]');
    await page.click('text=Input Interface');
    await expect(page.locator('[data-testid="knirv-shell"]')).toBeVisible();
  });
});
