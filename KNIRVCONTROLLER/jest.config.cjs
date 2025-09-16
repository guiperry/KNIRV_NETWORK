module.exports = {
  preset: 'ts-jest',
  testEnvironment: 'jsdom',
  testEnvironmentOptions: {
    html: '<html><body><div id="root"></div></body></html>',
    url: 'http://localhost:3000'
  },
  setupFiles: ['<rootDir>/tests/polyfills.ts', '<rootDir>/jest.setup.js'],
  setupFilesAfterEnv: ['<rootDir>/src/setupTests.ts', '<rootDir>/tests/test-setup.ts'],
  clearMocks: true,
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/src/$1',
    '^@components/(.*)$': '<rootDir>/src/components/$1',
    '^@pages/(.*)$': '<rootDir>/src/pages/$1',
    '^@hooks/(.*)$': '<rootDir>/src/hooks/$1',
    '^@services/(.*)$': '<rootDir>/src/services/$1',
    '^@types/(.*)$': '<rootDir>/src/types/$1',
    '^@core/(.*)$': '<rootDir>/src/core/$1',
    '^@manager/(.*)$': '<rootDir>/src/manager/$1',
    '^@shared/(.*)$': '<rootDir>/src/shared/$1',
    '^@sensory-shell/(.*)$': '<rootDir>/src/sensory-shell/$1',
    '^@wasm/(.*)$': '<rootDir>/src/wasm-pkg/$1',
    // React Native module mappings for Jest
    '^react-native$': '<rootDir>/tests/mocks/react-native.js',
    '^react-native/(.*)$': '<rootDir>/tests/mocks/react-native.js',
    'react-native': '<rootDir>/tests/mocks/react-native.js',
    // KNIRVWALLET module mappings for missing dependencies
    '^../../../../KNIRVWALLET/browser-bridge/packages/knirvwallet-module/src/wallet/wallet$': '<rootDir>/tests/mocks/knirvwallet.js',
    '^../../../../KNIRVWALLET/browser-bridge/packages/knirvwallet-module/src/wallet/wallet-crypto-util$': '<rootDir>/tests/mocks/knirvwallet.js',
    '^../../../../KNIRVWALLET/browser-bridge/packages/knirvwallet-module/src/test-utils/mock-ledgerconnector$': '<rootDir>/tests/mocks/knirvwallet.js',
    '^../../../../KNIRVWALLET/(.*)$': '<rootDir>/tests/mocks/knirvwallet.js',
    '\\.(css|less|scss|sass)$': 'identity-obj-proxy',
    '\\.(jpg|jpeg|png|gif|eot|otf|webp|svg|ttf|woff|woff2|mp4|webm|wav|mp3|m4a|aac|oga)$': 'jest-transform-stub'
  },
  transform: {
    '^.+\\.(ts|tsx)$': 'ts-jest',
    '^.+\\.(js|jsx)$': 'babel-jest',
    'node_modules/react-native/.+\\.(js)$': 'babel-jest'
  },
  transformIgnorePatterns: [
    'node_modules/(?!(@react-native|@expo|expo|@burnt-labs|@paralleldrive|@noble|formidable|superagent|supertest)/)'
  ],
  testMatch: [
    '<rootDir>/tests/**/*.test.(ts|tsx|js|jsx)',
    '<rootDir>/tests/**/*.spec.(ts|tsx|js|jsx)',
    '<rootDir>/src/**/__tests__/**/*.(ts|tsx|js|jsx)'
  ],
  testPathIgnorePatterns: [
    '<rootDir>/node_modules/',
    '<rootDir>/tests/e2e/',
    '<rootDir>/tests/playwright/',
    '<rootDir>/dist/',
    '<rootDir>/build/'
  ],
  collectCoverageFrom: [
    'src/**/*.{ts,tsx}',
    '!src/**/*.d.ts',
    '!src/main.tsx',
    '!src/vite-env.d.ts',
    '!src/**/*.stories.{ts,tsx}',
    '!src/**/__tests__/**',
    '!src/**/test-utils/**',
    '!src/core/agent-core-compiler/build/**',
    '!src/core/agent-core-compiler/templates/**',
    '!src/**/*.template.ts',
    '!src/**/*.wasm',
    '!src/**/*.generated.ts'
  ],
  coverageDirectory: 'coverage',
  coverageReporters: ['text', 'lcov', 'html'],
  coverageThreshold: {
    global: {
      branches: 100,
      functions: 100,
      lines: 100,
      statements: 100
    }
  },
  moduleFileExtensions: ['ts', 'tsx', 'js', 'jsx', 'json', 'node'],
  testTimeout: 15000,
  maxWorkers: 1,
  workerIdleMemoryLimit: '512MB',
  detectOpenHandles: true,
  forceExit: true,
  projects: [
    {
      displayName: 'Unit Tests',
      testMatch: ['<rootDir>/tests/unit/**/*.test.(ts|tsx|js|jsx)'],
      testEnvironment: 'jsdom',
      testEnvironmentOptions: {
        html: '<html><body><div id="root"></div></body></html>',
        url: 'http://localhost:3000'
      },
      setupFiles: ['<rootDir>/tests/polyfills.ts', '<rootDir>/jest.setup.js'],
      setupFilesAfterEnv: ['<rootDir>/src/setupTests.ts', '<rootDir>/tests/test-setup.ts']
    },
    {
      displayName: 'Integration Tests',
      testMatch: ['<rootDir>/tests/integration/**/*.test.(ts|tsx|js|jsx)'],
      testEnvironment: 'jsdom',
      testEnvironmentOptions: {
        html: '<html><body><div id="root"></div></body></html>',
        url: 'http://localhost:3000'
      },
      setupFiles: ['<rootDir>/tests/polyfills.ts', '<rootDir>/jest.setup.js'],
      setupFilesAfterEnv: ['<rootDir>/src/setupTests.ts', '<rootDir>/tests/test-setup.ts']
    },
    {
      displayName: 'Sensory Shell Tests',
      testMatch: ['<rootDir>/src/sensory-shell/**/__tests__/**/*.test.(ts|tsx|js|jsx)'],
      testEnvironment: 'jsdom',
      testEnvironmentOptions: {
        html: '<html><body><div id="root"></div></body></html>',
        url: 'http://localhost:3000'
      },
      setupFiles: ['<rootDir>/tests/polyfills.ts', '<rootDir>/jest.setup.js'],
      setupFilesAfterEnv: ['<rootDir>/src/setupTests.ts', '<rootDir>/tests/test-setup.ts']
    },
    {
      displayName: 'Phase 3 Tests',
      testMatch: ['<rootDir>/tests/phase3/**/*.test.(ts|tsx|js|jsx)'],
      testEnvironment: 'jsdom',
      testEnvironmentOptions: {
        html: '<html><body><div id="root"></div></body></html>',
        url: 'http://localhost:3000'
      },
      setupFiles: ['<rootDir>/tests/polyfills.ts', '<rootDir>/jest.setup.js'],
      setupFilesAfterEnv: ['<rootDir>/src/setupTests.ts', '<rootDir>/tests/test-setup.ts']
    },

    {
      displayName: 'Error Resolution Tests',
      testMatch: ['<rootDir>/tests/error-resolution/**/*.test.(ts|tsx|js|jsx)'],
      testEnvironment: 'jsdom',
      testEnvironmentOptions: {
        html: '<html><body><div id="root"></div></body></html>',
        url: 'http://localhost:3000'
      },
      setupFiles: ['<rootDir>/tests/polyfills.ts', '<rootDir>/jest.setup.js'],
      setupFilesAfterEnv: ['<rootDir>/src/setupTests.ts', '<rootDir>/tests/test-setup.ts']
    }
  ],
  reporters: [
    'default',
    ['jest-html-reporters', {
      publicPath: './test-results',
      filename: 'test-report.html',
      expand: true
    }],
    ['jest-junit', {
      outputDirectory: './test-results',
      outputName: 'junit.xml'
    }]
  ]
};
