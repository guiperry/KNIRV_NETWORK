// KNIRVCONTROLLER Comprehensive Jest Configuration
module.exports = {
  displayName: 'KNIRVCONTROLLER Tests',

  // Test environment
  testEnvironment: 'jsdom',

  // Root directories for tests
  roots: [
    '<rootDir>/tests',
    '<rootDir>/src'
  ],

  // Setup files
  setupFiles: ['<rootDir>/test-utils/jest-setup.js'],

  // Module file extensions
  moduleFileExtensions: ['ts', 'tsx', 'js', 'jsx', 'json'],

  // Transform configuration - use babel for all files
  transform: {
    '^.+\\.(ts|tsx|js|jsx)$': 'babel-jest'
  },

  // Module name mapping for path resolution and problematic modules
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/src/$1',
    '^@core/(.*)$': '<rootDir>/src/core/$1',
    '^@sensory-shell/(.*)$': '<rootDir>/src/sensory-shell/$1',
    '^@components/(.*)$': '<rootDir>/src/components/$1',
    '^@test-utils/(.*)$': '<rootDir>/test-utils/$1',
    // Mock problematic native modules
    '^react-native$': '<rootDir>/test-utils/mocks/react-native.js',
    '^@cosmjs/proto-signing$': '<rootDir>/test-utils/mocks/cosmjs-proto-signing.js',
    '^pino$': '<rootDir>/test-utils/mocks/pino.js',
    // Mock missing browser-wallet modules
    '^.*browser-wallet/packages/knirvwallet-module/src/wallet/wallet$': '<rootDir>/test-utils/mocks/browser-wallet/KnirvWallet.js',
    '^.*browser-wallet/packages/knirvwallet-module/src/test-utils/mock-ledgerconnector$': '<rootDir>/test-utils/mocks/browser-wallet/MockLedgerConnector.js',
    // Mock React JSX runtime for React Native
    '^react/jsx-runtime$': '<rootDir>/test-utils/mocks/jsx-runtime.js',
    '^react/jsx-dev-runtime$': '<rootDir>/test-utils/mocks/jsx-runtime.js',
    '\\.(css|less|scss|sass)$': 'identity-obj-proxy',
    '\\.(jpg|jpeg|png|gif|eot|otf|webp|svg|ttf|woff|woff2|mp4|webm|wav|mp3|m4a|aac|oga)$': 'jest-transform-stub'
  },
  
  // Test match patterns
  testMatch: [
    '<rootDir>/tests/**/*.(test|spec).(js|jsx|ts|tsx)',
    '<rootDir>/**/__tests__/**/*.(js|jsx|ts|tsx)',
    '<rootDir>/**/*.(test|spec).(js|jsx|ts|tsx)'
  ],
  
  // Ignore patterns
  testPathIgnorePatterns: [
    '/node_modules/',
    '/dist/',
    '/build/',
    '/coverage/'
  ],
  
  // Transform ignore patterns for native modules
  transformIgnorePatterns: [
    'node_modules/(?!(.*\\.mjs$|@gnolang|@cosmjs|libsodium-wrappers-sumo|libsodium-sumo|bech32|react-native|@react-native|react-native-.*|@testing-library)/)'
  ],
  
  // Setup files after environment
  setupFilesAfterEnv: [
    '<rootDir>/test-utils/jest-setup-after-env.js'
  ],
  
  // Global timeout - increased for complex tests
  testTimeout: 120000,
  
  // Coverage configuration
  collectCoverage: true,
  collectCoverageFrom: [
    'src/**/*.{ts,tsx,js,jsx}',
    '!**/*.d.ts',
    '!**/*.spec.ts',
    '!**/*.test.ts',
    '!**/node_modules/**',
    '!**/dist/**',
    '!**/build/**',
    '!**/coverage/**',
    '!**/temp/**'
  ],
  
  // Coverage directory
  coverageDirectory: '<rootDir>/coverage',
  
  // Coverage reporters
  coverageReporters: [
    'text',
    'text-summary',
    'lcov',
    'html',
    'json'
  ],
  
  // Coverage thresholds - Lowered for development phase
  coverageThreshold: {
    global: {
      branches: 30,
      functions: 30,
      lines: 30,
      statements: 30
    }
    // Removed browser-wallet path as it doesn't exist in this structure
  },
  
  // Test projects for different environments
  projects: [
    {
      displayName: 'Unit Tests',
      testMatch: ['<rootDir>/tests/unit/**/*.(test|spec).(js|jsx|ts|tsx)'],
      testEnvironment: 'jsdom',
      setupFilesAfterEnv: ['<rootDir>/test-utils/jest-setup-after-env.js']
    },
    {
      displayName: 'Integration Tests',
      testMatch: ['<rootDir>/tests/integration/**/*.(test|spec).(js|jsx|ts|tsx)'],
      testEnvironment: 'node'
    },
    {
      displayName: 'Sensory Shell Tests',
      testMatch: ['<rootDir>/src/sensory-shell/**/*.(test|spec).(js|jsx|ts|tsx)'],
      testEnvironment: 'jsdom',
      setupFilesAfterEnv: ['<rootDir>/test-utils/jest-setup-after-env.js']
    },
    {
      displayName: 'E2E Tests',
      testMatch: ['<rootDir>/tests/e2e/**/*.(test|spec).(js|jsx|ts|tsx)'],
      testEnvironment: 'node'
    },
    {
      displayName: 'Phase 3 Tests',
      testMatch: ['<rootDir>/tests/phase3/**/*.(test|spec).(js|jsx|ts|tsx)'],
      testEnvironment: 'node',
      setupFilesAfterEnv: ['<rootDir>/test-utils/jest-setup-after-env.js']
    }
  ],
  

  
  // Verbose output
  verbose: true,
  
  // Bail on first test failure in CI
  bail: process.env.CI ? 1 : 0,
  
  // Clear mocks between tests
  clearMocks: true,
  
  // Restore mocks after each test
  restoreMocks: true,
  
  // Error on deprecated features
  errorOnDeprecated: true,
  
  // Notify mode for watch
  notify: false,
  
  // Watch plugins
  watchPlugins: [
    'jest-watch-typeahead/filename',
    'jest-watch-typeahead/testname'
  ],
  
  // Reporters
  reporters: [
    'default',
    [
      'jest-junit',
      {
        outputDirectory: '<rootDir>/test-results',
        outputName: 'junit.xml',
        suiteName: 'KNIRVWALLET Tests'
      }
    ],
    [
      'jest-html-reporters',
      {
        publicPath: '<rootDir>/test-results',
        filename: 'test-report.html',
        expand: true
      }
    ]
  ],
  
  // Max workers for parallel execution
  maxWorkers: process.env.CI ? 2 : '50%',
  
  // Cache directory
  cacheDirectory: '<rootDir>/.jest-cache'
};
