// Playwright Configuration for KNIRVWALLET E2E Tests
const { defineConfig, devices } = require('@playwright/test');

module.exports = defineConfig({
  // Test directory
  testDir: './tests/e2e',
  
  // Global test timeout
  timeout: 60000,
  
  // Expect timeout for assertions
  expect: {
    timeout: 10000
  },
  
  // Fail the build on CI if you accidentally left test.only in the source code
  forbidOnly: !!process.env.CI,
  
  // Retry on CI only
  retries: process.env.CI ? 2 : 0,
  
  // Opt out of parallel tests on CI
  workers: process.env.CI ? 1 : undefined,
  
  // Reporter configuration
  reporter: [
    ['html', { outputFolder: 'test-results/e2e-report' }],
    ['json', { outputFile: 'test-results/e2e-results.json' }],
    ['junit', { outputFile: 'test-results/e2e-junit.xml' }],
    process.env.CI ? ['github'] : ['list']
  ],
  
  // Shared settings for all projects
  use: {
    // Base URL for tests
    baseURL: process.env.BASE_URL || 'http://localhost:8000',
    
    // Collect trace when retrying the failed test
    trace: 'on-first-retry',
    
    // Record video on failure
    video: 'retain-on-failure',
    
    // Take screenshot on failure
    screenshot: 'only-on-failure',
    
    // Global timeout for actions
    actionTimeout: 15000,
    
    // Global timeout for navigation
    navigationTimeout: 30000,
    
    // Ignore HTTPS errors
    ignoreHTTPSErrors: true,
    
    // Accept downloads
    acceptDownloads: true,
    
    // Locale
    locale: 'en-US',
    
    // Timezone
    timezoneId: 'America/New_York'
  },

  // Configure projects for major browsers
  projects: [
    {
      name: 'chromium',
      use: { 
        ...devices['Desktop Chrome'],
        // Additional Chrome-specific settings
        launchOptions: {
          args: [
            '--disable-web-security',
            '--disable-features=VizDisplayCompositor',
            '--no-sandbox',
            '--disable-setuid-sandbox'
          ]
        }
      },
    },

    {
      name: 'firefox',
      use: { 
        ...devices['Desktop Firefox'],
        // Firefox-specific settings
        launchOptions: {
          firefoxUserPrefs: {
            'dom.webnotifications.enabled': false,
            'dom.push.enabled': false
          }
        }
      },
    },

    {
      name: 'webkit',
      use: { 
        ...devices['Desktop Safari'],
        // Safari-specific settings
      },
    },

    // Mobile testing
    {
      name: 'Mobile Chrome',
      use: { 
        ...devices['Pixel 5'],
        // Mobile Chrome settings
      },
    },
    
    {
      name: 'Mobile Safari',
      use: { 
        ...devices['iPhone 12'],
        // Mobile Safari settings
      },
    },

    // Tablet testing
    {
      name: 'iPad',
      use: { 
        ...devices['iPad Pro'],
        // iPad settings
      },
    },

    // Cross-platform testing setup
    {
      name: 'cross-platform-desktop',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1280, height: 720 },
        // Desktop-specific settings for cross-platform tests
      },
      testMatch: '**/cross-platform-*.test.js'
    },

    {
      name: 'cross-platform-mobile',
      use: {
        ...devices['iPhone 12'],
        // Mobile-specific settings for cross-platform tests
      },
      testMatch: '**/cross-platform-*.test.js'
    }
  ],

  // Global setup and teardown
  globalSetup: require.resolve('./tests/e2e/global-setup.js'),
  globalTeardown: require.resolve('./tests/e2e/global-teardown.js'),

  // Web server configuration for local development
  webServer: process.env.CI ? undefined : [
    {
      command: 'cd ../KNIRVGATEWAY && python3 -m http.server 8000',
      port: 8000,
      timeout: 120000,
      reuseExistingServer: !process.env.CI,
    },
    {
      command: 'cd ../KNIRVWALLET/agentic-wallet/go-backend && go run main.go',
      port: 8083,
      timeout: 120000,
      reuseExistingServer: !process.env.CI,
    }
  ],

  // Test output directory
  outputDir: 'test-results/e2e-artifacts',

  // Maximum number of test failures before stopping
  maxFailures: process.env.CI ? 10 : undefined,

  // Test metadata
  metadata: {
    'test-suite': 'KNIRVWALLET E2E Tests',
    'version': '1.0.0',
    'environment': process.env.NODE_ENV || 'test',
    'browser-versions': 'Latest stable versions',
    'test-scope': 'Full wallet workflow, cross-platform sync, XION integration'
  }
});
