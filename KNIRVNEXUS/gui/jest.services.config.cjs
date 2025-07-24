module.exports = {
  testEnvironment: 'node',
  setupFilesAfterEnv: ['<rootDir>/test/setup.js'],
  testMatch: ['<rootDir>/src/services/**/*.test.js'],
  transform: {
    '^.+\\.(js|jsx|ts|tsx)$': ['babel-jest', { presets: ['@babel/preset-env'] }]
  },
  testPathIgnorePatterns: ['/node_modules/', '/dist/'],
  coveragePathIgnorePatterns: ['/node_modules/', '/dist/'],
  collectCoverageFrom: [
    'src/services/systemUtils.js',
    'src/services/targetDiscovery.js'
  ],
  coverageThreshold: {
    global: {
      branches: 20,
      functions: 25,
      lines: 30,
      statements: 30
    }
  },
  // Clear mocks between tests
  clearMocks: true,
  resetMocks: true,
  restoreMocks: true,
  // Timeout for system detection tests
  testTimeout: 10000
};
