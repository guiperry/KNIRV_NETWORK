module.exports = {
  displayName: 'KNIRVWALLET Integration Tests',
  testEnvironment: 'node',
  moduleFileExtensions: ['js', 'json', 'jsx', 'ts', 'tsx'],
  transform: {
    '^.+\\.ts?$': 'babel-jest',
  },
  rootDir: '.',
  moduleNameMapping: {
    '^@/(.*)$': '<rootDir>/../packages/knirvwallet-module/src/$1',
    '^@knirvwallet/(.*)$': '<rootDir>/../packages/$1/src',
  },
  testMatch: ['<rootDir>/**/*.integration.spec.(js|jsx|ts|tsx)'],
  transformIgnorePatterns: ['<rootDir>/node_modules/'],
  setupFilesAfterEnv: ['<rootDir>/setup.js'],
  testTimeout: 30000, // 30 seconds for integration tests
  collectCoverageFrom: [
    '../packages/*/src/**/*.{ts,tsx}',
    '!../packages/*/src/**/*.d.ts',
    '!../packages/*/src/**/*.spec.ts',
    '!../packages/*/src/**/*.test.ts',
  ],
  coverageDirectory: '<rootDir>/coverage',
  coverageReporters: ['text', 'lcov', 'html'],
};
