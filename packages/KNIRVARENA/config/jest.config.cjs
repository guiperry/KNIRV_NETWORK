const path = require('path');
const projectRoot = path.resolve(__dirname, '..');

// Projects don't inherit the root `transform`; without it the TypeScript
// setup file (tests/polyfills.ts) can't be parsed and every suite fails.
const tsJest = ['ts-jest', { astTransformers: { before: ['<rootDir>/config/jest-import-meta-transformer.cjs'] } }];
// The Babel config lives in config/, where babel-jest wouldn't find it.
const babelJest = ['babel-jest', { configFile: path.join(projectRoot, 'config', 'babel.config.cjs') }];
const projectTransform = {
  '^.+\\.(ts|tsx)$': tsJest,
  '^.+\\.(js|jsx)$': babelJest
};

const config = {
  rootDir: '../',
  preset: 'ts-jest',
  testEnvironment: 'jsdom',
  testEnvironmentOptions: {
    html: '<html><body><div id="root"></div></body></html>',
    url: 'http://localhost:3000'
  },
  setupFiles: ['<rootDir>/tests/polyfills.ts', '<rootDir>/config/jest.setup.js'],
  setupFilesAfterEnv: ['<rootDir>/src/setupTests.ts', '<rootDir>/tests/test-setup.ts', '<rootDir>/tests/setup-safety-checks.ts'],
  clearMocks: true,
  moduleNameMapper: {
    // TypeScript ESM sources import with .js extensions; map them
    // back to the .ts/.tsx sources Jest actually resolves.
    '^(\\.{1,2}/.*)\\.js$': '$1',
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
    '^@engine/(.*)$': '<rootDir>/src/engine/$1',
    '^@networking/(.*)$': '<rootDir>/src/networking/$1',
    // React Native module mappings for Jest
    '^react-native$': '<rootDir>/tests/mocks/react-native.js',
    '^react-native/(.*)$': '<rootDir>/tests/mocks/react-native.js',
    'react-native': '<rootDir>/tests/mocks/react-native.js',
    // KNIRVWALLET module mappings for missing dependencies
    '^../../../../KNIRVWALLET/browser-bridge/packages/knirvwallet-module/src/wallet/wallet$': '<rootDir>/tests/mocks/knirvwallet.js',
    '^../../../../KNIRVWALLET/browser-bridge/packages/knirvwallet-module/src/wallet/wallet-crypto-util$': '<rootDir>/tests/mocks/knirvwallet.js',
    '^../../../../KNIRVWALLET/browser-bridge/packages/knirvwallet-module/src/test-utils/mock-ledgerconnector$': '<rootDir>/tests/mocks/knirvwallet.js',
    '^../../../../KNIRVWALLET/(.*)$': '<rootDir>/tests/mocks/knirvwallet.js',
    // Babel's transform-runtime injects these into the symlinked KNIRVBASE
    // dist, whose own node_modules doesn't carry @babel/runtime.
    '^@babel/runtime/(.*)$': '<rootDir>/node_modules/@babel/runtime/$1',
    '^react-markdown$': '<rootDir>/tests/mocks/react-markdown.tsx',
    '^\\./libs/MeshoptDecoder\\.cjs$': '<rootDir>/tests/mocks/meshoptDecoder.js',
    '^react-syntax-highlighter/dist/esm/(.*)$': '<rootDir>/node_modules/react-syntax-highlighter/dist/cjs/$1',
    '\\.(css|less|scss|sass)$': 'identity-obj-proxy',
    '\\.(jpg|jpeg|png|gif|eot|otf|webp|svg|ttf|woff|woff2|mp4|webm|wav|mp3|m4a|aac|oga)$': 'jest-transform-stub'
  },
  transform: {
    '^.+\\.(ts|tsx)$': 'ts-jest',
    '^.+\\.(js|jsx)$': 'babel-jest',
    'node_modules/react-native/.+\\.(js)$': 'babel-jest'
  },
  transformIgnorePatterns: [
    'node_modules/(?!(@react-native|@expo|expo|@paralleldrive|@noble|formidable|superagent|three/examples/jsm)/)'
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
    '<rootDir>/build/',
    '<rootDir>/tests/legacy/legacy/' // Prevent nested legacy directories from being tested
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
      rootDir: projectRoot,
      transform: projectTransform,
      testMatch: ['<rootDir>/tests/unit/**/*.test.(ts|tsx|js|jsx)'],
      testEnvironment: 'jsdom',
      testEnvironmentOptions: {
        html: '<html><body><div id="root"></div></body></html>',
        url: 'http://localhost:3000'
      },
      setupFiles: ['<rootDir>/tests/polyfills.ts', '<rootDir>/config/jest.setup.js'],
      setupFilesAfterEnv: ['<rootDir>/src/setupTests.ts', '<rootDir>/tests/test-setup.ts', '<rootDir>/tests/setup-safety-checks.ts']
    },
    {
      displayName: 'Integration Tests',
      rootDir: projectRoot,
      transform: projectTransform,
      testMatch: ['<rootDir>/tests/integration/**/*.test.(ts|tsx|js|jsx)'],
      testEnvironment: 'jsdom',
      testEnvironmentOptions: {
        html: '<html><body><div id="root"></div></body></html>',
        url: 'http://localhost:3000'
      },
      setupFiles: ['<rootDir>/tests/polyfills.ts', '<rootDir>/config/jest.setup.js'],
      setupFilesAfterEnv: ['<rootDir>/src/setupTests.ts', '<rootDir>/tests/test-setup.ts', '<rootDir>/tests/setup-safety-checks.ts']
    },
    {
      displayName: 'Sensory Shell Tests',
      rootDir: projectRoot,
      transform: projectTransform,
      testMatch: ['<rootDir>/src/sensory-shell/**/__tests__/**/*.test.(ts|tsx|js|jsx)'],
      testEnvironment: 'jsdom',
      testEnvironmentOptions: {
        html: '<html><body><div id="root"></div></body></html>',
        url: 'http://localhost:3000'
      },
      setupFiles: ['<rootDir>/tests/polyfills.ts', '<rootDir>/config/jest.setup.js'],
      setupFilesAfterEnv: ['<rootDir>/src/setupTests.ts', '<rootDir>/tests/test-setup.ts', '<rootDir>/tests/setup-safety-checks.ts']
    },
    {
      displayName: 'Phase 3 Tests',
      rootDir: projectRoot,
      transform: projectTransform,
      testMatch: ['<rootDir>/tests/phase3/**/*.test.(ts|tsx|js|jsx)'],
      testEnvironment: 'jsdom',
      testEnvironmentOptions: {
        html: '<html><body><div id="root"></div></body></html>',
        url: 'http://localhost:3000'
      },
      setupFiles: ['<rootDir>/tests/polyfills.ts', '<rootDir>/config/jest.setup.js'],
      setupFilesAfterEnv: ['<rootDir>/src/setupTests.ts', '<rootDir>/tests/test-setup.ts', '<rootDir>/tests/setup-safety-checks.ts']
    },

    {
      displayName: 'Error Resolution Tests',
      rootDir: projectRoot,
      transform: projectTransform,
      testMatch: ['<rootDir>/tests/error-resolution/**/*.test.(ts|tsx|js|jsx)'],
      testEnvironment: 'jsdom',
      testEnvironmentOptions: {
        html: '<html><body><div id="root"></div></body></html>',
        url: 'http://localhost:3000'
      },
      setupFiles: ['<rootDir>/tests/polyfills.ts', '<rootDir>/config/jest.setup.js'],
      setupFilesAfterEnv: ['<rootDir>/src/setupTests.ts', '<rootDir>/tests/test-setup.ts', '<rootDir>/tests/setup-safety-checks.ts']
    },
    {
      // Service-layer unit tests colocated with src/services. The Actuarial
      // test needs its own setup and runs in its own project below.
      displayName: 'Service Tests',
      rootDir: projectRoot,
      testMatch: ['<rootDir>/src/services/**/__tests__/**/*.test.(ts|tsx|js|jsx)'],
      testPathIgnorePatterns: ['<rootDir>/node_modules/', '<rootDir>/src/services/__tests__/ActuarialSyndicateService.test.ts'],
      transform: { '^.+\\.(ts|tsx)$': tsJest, '^.+\\.(js|jsx)$': babelJest },
      testEnvironment: 'jsdom',
      testEnvironmentOptions: {
        html: '<html><body><div id="root"></div></body></html>',
        url: 'http://localhost:3000'
      },
      setupFiles: ['<rootDir>/tests/polyfills.ts', '<rootDir>/config/jest.setup.js'],
      setupFilesAfterEnv: ['<rootDir>/src/setupTests.ts', '<rootDir>/tests/test-setup.ts', '<rootDir>/tests/setup-safety-checks.ts']
    },
    {
      displayName: 'Actuarial Arena Tests',
      rootDir: projectRoot,
      testMatch: ['<rootDir>/src/services/__tests__/ActuarialSyndicateService.test.ts'],
      testEnvironment: 'jsdom',
      transform: { '^.+\\.(ts|tsx)$': tsJest, '^.+\\.(js|jsx)$': babelJest },
      setupFiles: ['<rootDir>/config/jest.actuarial.setup.cjs'],
      setupFilesAfterEnv: []
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

// Projects inherit none of the root resolution settings; share them so path
// aliases (@services/…) and module stubs resolve the same in every project.
// The unanchored 'react-native' key would also capture
// @testing-library/react-native; the anchored entries cover react-native.
const { 'react-native': _unanchoredReactNative, ...sharedModuleNameMapper } = config.moduleNameMapper;
const shared = {
  moduleNameMapper: sharedModuleNameMapper,
  transformIgnorePatterns: config.transformIgnorePatterns,
  moduleFileExtensions: config.moduleFileExtensions,
};
for (const project of config.projects) {
  for (const [key, value] of Object.entries(shared)) {
    if (!(key in project)) project[key] = value;
  }
}

module.exports = config;
