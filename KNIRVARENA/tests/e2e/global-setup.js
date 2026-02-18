// Global Setup for KNIRVWALLET E2E Tests
import { chromium } from 'playwright';
import fs from 'fs/promises';
import path from 'path';

async function globalSetup(config) {
  console.log('🚀 Starting KNIRVWALLET E2E Test Global Setup...');

  // Create test results directories
  await ensureDirectories();

  // Wait for services to be ready
  await waitForServices();

  // Setup test data
  await setupTestData();

  // Perform any necessary authentication
  await setupAuthentication();

  console.log('✅ Global setup completed successfully');
}

async function ensureDirectories() {
  const directories = [
    'test-results',
    'test-results/e2e-report',
    'test-results/e2e-artifacts',
    'test-results/screenshots',
    'test-results/videos',
    'test-results/traces'
  ];

  for (const dir of directories) {
    try {
      await fs.mkdir(dir, { recursive: true });
      console.log(`📁 Created directory: ${dir}`);
    } catch (error) {
      if (error.code !== 'EEXIST') {
        console.error(`❌ Failed to create directory ${dir}:`, error);
        throw error;
      }
    }
  }
}

async function waitForServices() {
  const services = [
    {
      name: 'KNIRVGATEWAY',
      url: 'http://localhost:8000/health',
      timeout: 120000
    },
    {
      name: 'KNIRVWALLET Backend',
      url: 'http://localhost:8083/health',
      timeout: 120000
    }
  ];

  console.log('⏳ Waiting for services to be ready...');

  for (const service of services) {
    await waitForService(service);
  }

  console.log('✅ All services are ready');
}

async function waitForService(service) {
  const startTime = Date.now();
  const { name, url, timeout } = service;

  while (Date.now() - startTime < timeout) {
    try {
      const browser = await chromium.launch();
      const context = await browser.newContext();
      const page = await context.newPage();

      const response = await page.goto(url, { 
        waitUntil: 'networkidle',
        timeout: 10000 
      });

      if (response && response.ok()) {
        console.log(`✅ ${name} is ready at ${url}`);
        await browser.close();
        return;
      }

      await browser.close();
    } catch (error) {
      // Service not ready yet, continue waiting
    }

    console.log(`⏳ Waiting for ${name} at ${url}...`);
    await new Promise(resolve => setTimeout(resolve, 2000));
  }

  throw new Error(`❌ ${name} failed to start within ${timeout}ms`);
}

async function setupTestData() {
  console.log('📝 Setting up test data...');

  const testData = {
    wallets: [
      {
        name: 'E2E Test Wallet 1',
        mnemonic: 'abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about',
        type: 'HD'
      },
      {
        name: 'E2E Test Wallet 2',
        mnemonic: 'abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon',
        type: 'HD'
      }
    ],
    accounts: [
      {
        name: 'Test Account 1',
        address: 'xion1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5',
        network: 'xion-testnet-1'
      },
      {
        name: 'Test Account 2',
        address: 'xion1234567890abcdef1234567890abcdef12345678',
        network: 'xion-testnet-1'
      }
    ],
    transactions: [
      {
        from: 'xion1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5',
        to: 'xion1234567890abcdef1234567890abcdef12345678',
        amount: '100000',
        memo: 'E2E test transaction'
      }
    ],
    skills: [
      {
        id: 'skill-e2e-test-001',
        name: 'E2E Test Skill',
        description: 'Test skill for E2E testing',
        cost: '50000'
      }
    ]
  };

  // Save test data to file for use in tests
  const testDataPath = path.join('test-results', 'test-data.json');
  await fs.writeFile(testDataPath, JSON.stringify(testData, null, 2));

  console.log('✅ Test data setup completed');
}

async function setupAuthentication() {
  console.log('🔐 Setting up authentication...');

  try {
    // Launch browser for authentication setup
    const browser = await chromium.launch();
    const context = await browser.newContext();
    const page = await context.newPage();

    // Navigate to wallet interface
    await page.goto('http://localhost:8000/wallet');

    // Check if authentication is required
    const loginRequired = await page.locator('[data-testid="login-required"]').isVisible().catch(() => false);

    if (loginRequired) {
      console.log('🔑 Authentication required, setting up...');
      
      // Perform any necessary authentication steps
      // This would depend on the specific authentication mechanism
      
      // For now, we'll just verify the interface loads
      await page.waitForSelector('[data-testid="wallet-interface"]', { timeout: 30000 });
    }

    // Save authentication state if needed
    const storageState = await context.storageState();
    const authStatePath = path.join('test-results', 'auth-state.json');
    await fs.writeFile(authStatePath, JSON.stringify(storageState, null, 2));

    await browser.close();
    console.log('✅ Authentication setup completed');
  } catch (error) {
    console.warn('⚠️ Authentication setup failed, tests may need to handle auth:', error.message);
  }
}

// Environment validation
async function validateEnvironment() {
  console.log('🔍 Validating test environment...');

  const requiredEnvVars = [
    'NODE_ENV'
  ];

  const optionalEnvVars = [
    'CI',
    'BASE_URL',
    'GATEWAY_URL',
    'WALLET_URL'
  ];

  // Check required environment variables
  for (const envVar of requiredEnvVars) {
    if (!process.env[envVar]) {
      console.warn(`⚠️ Required environment variable ${envVar} is not set`);
    }
  }

  // Log optional environment variables
  for (const envVar of optionalEnvVars) {
    if (process.env[envVar]) {
      console.log(`📋 ${envVar}: ${process.env[envVar]}`);
    }
  }

  // Set default values
  process.env.NODE_ENV = process.env.NODE_ENV || 'test';
  process.env.BASE_URL = process.env.BASE_URL || 'http://localhost:8000';
  process.env.GATEWAY_URL = process.env.GATEWAY_URL || 'http://localhost:8000';
  process.env.WALLET_URL = process.env.WALLET_URL || 'http://localhost:8083';

  console.log('✅ Environment validation completed');
}

// Network connectivity check
async function checkNetworkConnectivity() {
  console.log('🌐 Checking network connectivity...');

  const testUrls = [
    'https://www.google.com',
    'https://github.com',
    'https://rpc.xion-testnet-1.burnt.com'
  ];

  for (const url of testUrls) {
    try {
      const browser = await chromium.launch();
      const context = await browser.newContext();
      const page = await context.newPage();

      await page.goto(url, { timeout: 10000 });
      console.log(`✅ Network connectivity to ${url} - OK`);

      await browser.close();
    } catch (error) {
      console.warn(`⚠️ Network connectivity to ${url} - Failed: ${error.message}`);
    }
  }

  console.log('✅ Network connectivity check completed');
}

// Main setup function
async function main() {
  try {
    await validateEnvironment();
    await checkNetworkConnectivity();
    await globalSetup();
  } catch (error) {
    console.error('❌ Global setup failed:', error);
    process.exit(1);
  }
}

// Export for Playwright
export default globalSetup;

// Run directly if called from command line
if (import.meta.url === `file://${process.argv[1]}`) {
  main();
}
