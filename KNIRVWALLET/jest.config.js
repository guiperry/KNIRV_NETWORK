// KNIRVWALLET Comprehensive Jest Configuration
module.exports = {
  displayName: 'KNIRVWALLET Tests',

  // Test environment
  testEnvironment: 'jsdom',

  // Root directories for tests
  roots: [
    '<rootDir>/tests'
  ],

  // Setup files
  setupFiles: ['<rootDir>/test-utils/jest-setup.js'],

  // Module file extensions
  moduleFileExtensions: ['ts', 'tsx', 'js', 'jsx', 'json'],

  // Transform configuration - explicitly override any babel configs
  transform: {
    '^.+\\.tsx?$': 'ts-jest',
    '^.+\\.jsx?$': 'babel-jest'
  },

  // Transform ignore patterns for native modules
  transformIgnorePatterns: [
    'node_modules/(?!(libsodium-wrappers-sumo|libsodium-sumo|@cosmjs|bech32)/)'
  ],

  // Module name mapping for problematic modules
  moduleNameMapper: {
    '^libsodium-sumo$': '<rootDir>/browser-wallet/node_modules/libsodium-sumo',
    '^libsodium-wrappers-sumo$': '<rootDir>/browser-wallet/node_modules/libsodium-wrappers-sumo'
  },

  // Globals for ts-jest
  globals: {
    'ts-jest': {
      useESM: false,
      tsconfig: {
        compilerOptions: {
          module: 'commonjs',
          target: 'es2020',
          lib: ['es2020', 'dom'],
          esModuleInterop: true,
          allowSyntheticDefaultImports: true,
          strict: false,
          skipLibCheck: true,
          isolatedModules: false
        }
      }
    }
  },
  
  // Module name mapping for path resolution
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/test-utils/$1',
    '^@knirvwallet/(.*)$': '<rootDir>/browser-wallet/packages/$1/src',
    '^@agentic/(.*)$': '<rootDir>/agentic-wallet/src/$1',
    '^@test-utils/(.*)$': '<rootDir>/test-utils/$1',
    '^@integration/(.*)$': '<rootDir>/../integration-tests/$1'
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
  
  // Transform ignore patterns
  transformIgnorePatterns: [
    'node_modules/(?!(.*\\.mjs$|@gnolang|@cosmjs))'
  ],
  
  // Setup files
  setupFiles: [
    '<rootDir>/test-utils/jest-setup.js'
  ],
  
  // Setup files after environment
  setupFilesAfterEnv: [
    '<rootDir>/test-utils/jest-setup-after-env.js'
  ],
  
  // Global timeout
  testTimeout: 30000,
  
  // Coverage configuration
  collectCoverage: true,
  collectCoverageFrom: [
    'browser-wallet/packages/*/src/**/*.{ts,tsx}',
    'agentic-wallet/src/**/*.{ts,tsx,js,jsx}',
    'agentic-wallet/go-backend/**/*.go',
    '!**/*.d.ts',
    '!**/*.spec.ts',
    '!**/*.test.ts',
    '!**/node_modules/**',
    '!**/dist/**',
    '!**/build/**',
    '!**/coverage/**'
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
  
  // Coverage thresholds
  coverageThreshold: {
    global: {
      branches: 70,
      functions: 70,
      lines: 70,
      statements: 70
    },
    './browser-wallet/packages/knirvwallet-module/src/': {
      branches: 80,
      functions: 80,
      lines: 80,
      statements: 80
    }
  },
  
  // Test projects for different environments
  projects: [
    {
      displayName: 'Browser Wallet Unit Tests',
      testMatch: ['<rootDir>/browser-wallet/**/*.(test|spec).(js|jsx|ts|tsx)'],
      testEnvironment: 'jsdom',
      setupFilesAfterEnv: ['<rootDir>/browser-wallet/packages/knirvwallet-extension/jest.setup.js']
    },
    {
      displayName: 'Agentic Wallet Unit Tests',
      testMatch: ['<rootDir>/agentic-wallet/**/*.(test|spec).(js|jsx|ts|tsx)'],
      testEnvironment: 'node'
    },
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
      displayName: 'E2E Tests',
      testMatch: ['<rootDir>/tests/e2e/**/*.(test|spec).(js|jsx|ts|tsx)'],
      testEnvironment: 'node'
    }
  ],
  
  // Global variables
  globals: {
    'ts-jest': {
      useESM: true
    }
  },
  
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
