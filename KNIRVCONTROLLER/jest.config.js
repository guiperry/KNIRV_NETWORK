export default {
  preset: 'ts-jest',
  testEnvironment: 'jsdom',
  setupFiles: ['<rootDir>/tests/polyfills.ts'],
  setupFilesAfterEnv: ['<rootDir>/src/setupTests.ts'],
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/src/$1',
    '\\.(css|less|scss|sass)$': 'identity-obj-proxy',
    '\\.(jpg|jpeg|png|gif|eot|otf|webp|svg|ttf|woff|woff2|mp4|webm|wav|mp3|m4a|aac|oga)$': 'jest-transform-stub'
  },
  transform: {
    '^.+\\.(ts|tsx)$': 'ts-jest',
    '^.+\\.(js|jsx)$': 'babel-jest'
  },
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
      branches: 30,
      functions: 30,
      lines: 30,
      statements: 30
    }
  },
  moduleFileExtensions: ['ts', 'tsx', 'js', 'jsx', 'json', 'node'],
  testTimeout: 30000,
  maxWorkers: '50%',
  projects: [
    {
      displayName: 'Unit Tests',
      testMatch: ['<rootDir>/tests/unit/**/*.test.(ts|tsx|js|jsx)'],
      testEnvironment: 'jsdom'
    },
    {
      displayName: 'Integration Tests', 
      testMatch: ['<rootDir>/tests/integration/**/*.test.(ts|tsx|js|jsx)'],
      testEnvironment: 'jsdom'
    },
    {
      displayName: 'Sensory Shell Tests',
      testMatch: ['<rootDir>/src/sensory-shell/**/__tests__/**/*.test.(ts|tsx|js|jsx)'],
      testEnvironment: 'jsdom'
    },
    {
      displayName: 'Phase 3 Tests',
      testMatch: ['<rootDir>/tests/phase3/**/*.test.(ts|tsx|js|jsx)'],
      testEnvironment: 'jsdom'
    },
    {
      displayName: 'E2E Tests',
      testMatch: ['<rootDir>/tests/e2e/**/*.test.(ts|tsx|js|jsx)'],
      testEnvironment: 'jsdom',
      testPathIgnorePatterns: [
        '<rootDir>/tests/e2e/.*\\.spec\\.(ts|tsx|js|jsx)$'
      ]
    },
    {
      displayName: 'Error Resolution Tests',
      testMatch: ['<rootDir>/tests/error-resolution/**/*.test.(ts|tsx|js|jsx)'],
      testEnvironment: 'jsdom'
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
