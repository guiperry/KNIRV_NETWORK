/**
 * Jest Configuration for Integration Tests
 * Optimized for the new testing structure with minimal mocking
 */

const baseConfig = require('./jest.config.cjs');

module.exports = {
  ...baseConfig,
  
  // Test environment
  testEnvironment: 'jsdom',
  
  // Test file patterns for integration tests
  testMatch: [
    '<rootDir>/tests/integration/**/*.test.ts',
    '<rootDir>/tests/integration/**/*.test.tsx'
  ],
  
  // Setup files
  setupFilesAfterEnv: [
    '<rootDir>/test-utils/jest-setup-after-env.js',
    '<rootDir>/tests/integration/setup.ts'
  ],
  
  // Module name mapping for integration tests
  moduleNameMapping: {
    ...baseConfig.moduleNameMapping,
    '^@/(.*)$': '<rootDir>/src/$1',
    '^@tests/(.*)$': '<rootDir>/tests/$1',
    '^@test-utils/(.*)$': '<rootDir>/test-utils/$1'
  },
  
  // Transform configuration
  transform: {
    '^.+\\.(ts|tsx)$': ['ts-jest', {
      tsconfig: {
        jsx: 'react-jsx',
        esModuleInterop: true,
        allowSyntheticDefaultImports: true,
        moduleResolution: 'node',
        resolveJsonModule: true
      }
    }],
    '^.+\\.(js|jsx)$': 'babel-jest'
  },
  
  // Module file extensions
  moduleFileExtensions: ['ts', 'tsx', 'js', 'jsx', 'json'],
  
  // Coverage configuration for integration tests
  collectCoverageFrom: [
    'src/**/*.{ts,tsx}',
    '!src/**/*.d.ts',
    '!src/**/*.test.{ts,tsx}',
    '!src/test/**/*',
    '!src/**/__tests__/**/*',
    // Focus on integration points
    'src/components/**/*.{ts,tsx}',
    'src/pages/**/*.{ts,tsx}',
    'src/sensory-shell/**/*.ts',
    'src/core/**/*.ts'
  ],
  
  // Coverage thresholds for integration tests
  coverageThreshold: {
    global: {
      branches: 100,
      functions: 100,
      lines: 100,
      statements: 100
    },
    // Higher thresholds for critical integration components
    'src/sensory-shell/CognitiveEngine.ts': {
      branches: 100,
      functions: 100,
      lines: 100,
      statements: 100
    },
    'src/sensory-shell/WASMOrchestrator.ts': {
      branches: 100,
      functions: 100,
      lines: 100,
      statements: 100
    },
    'src/sensory-shell/KNIRVRouterIntegration.ts': {
      branches: 100,
      functions: 100,
      lines: 100,
      statements: 100
    }
  },
  
  // Test timeout for integration tests (longer than unit tests)
  testTimeout: 30000,
  
  // Global setup and teardown
  globalSetup: '<rootDir>/tests/integration/global-setup.ts',
  globalTeardown: '<rootDir>/tests/integration/global-teardown.ts',
  
  // Test environment options
  testEnvironmentOptions: {
    url: 'http://localhost:3000'
  },
  
  // Clear mocks between tests
  clearMocks: true,
  restoreMocks: true,
  
  // Verbose output for integration tests
  verbose: true,
  
  // Error handling
  errorOnDeprecated: true,
  
  // Module resolution
  resolver: undefined,
  
  // Watch mode configuration
  watchPathIgnorePatterns: [
    '<rootDir>/node_modules/',
    '<rootDir>/dist/',
    '<rootDir>/coverage/',
    '<rootDir>/temp/'
  ],
  
  // Ignore patterns
  testPathIgnorePatterns: [
    '<rootDir>/node_modules/',
    '<rootDir>/dist/',
    '<rootDir>/tests/unit/',
    '<rootDir>/tests/e2e/',
    '<rootDir>/tests/legacy/'
  ],
  
  // Mock configuration - minimal mocking for integration tests
  clearMocks: true,
  resetMocks: false, // Don't reset mocks to allow persistent external mocks
  restoreMocks: true,
  
  // Globals for integration tests
  globals: {
    'ts-jest': {
      useESM: true,
      tsconfig: {
        jsx: 'react-jsx'
      }
    },
    // Integration test specific globals
    INTEGRATION_TEST: true,
    MOCK_EXTERNAL_ONLY: true
  },
  
  // Reporter configuration
  reporters: [
    'default',
    ['jest-html-reporters', {
      publicPath: './test-results',
      filename: 'integration-test-report.html',
      expand: true,
      hideIcon: false,
      pageTitle: 'KNIRVCONTROLLER Integration Test Report'
    }],
    ['jest-junit', {
      outputDirectory: './test-results',
      outputName: 'integration-junit.xml',
      suiteName: 'KNIRVCONTROLLER Integration Tests'
    }]
  ],
  
  // Cache configuration
  cacheDirectory: '<rootDir>/node_modules/.cache/jest-integration',
  
  // Preset
  preset: undefined,
  
  // Projects configuration for different test phases
  projects: [
    {
      displayName: 'Phase 1: Infrastructure',
      testMatch: ['<rootDir>/tests/integration/phase1-*.test.ts'],
      setupFilesAfterEnv: ['<rootDir>/tests/integration/phase1-setup.ts']
    },
    {
      displayName: 'Phase 2: Revolutionary Architecture',
      testMatch: ['<rootDir>/tests/integration/phase2-*.test.ts'],
      setupFilesAfterEnv: ['<rootDir>/tests/integration/phase2-setup.ts']
    },
    {
      displayName: 'Phase 3: Frontend-Backend',
      testMatch: ['<rootDir>/tests/integration/phase3-*.test.ts'],
      setupFilesAfterEnv: ['<rootDir>/tests/integration/phase3-setup.ts']
    },
    {
      displayName: 'Phase 4: Mock Removal',
      testMatch: ['<rootDir>/tests/integration/phase4-*.test.ts'],
      setupFilesAfterEnv: ['<rootDir>/tests/integration/phase4-setup.ts']
    }
  ]
};
