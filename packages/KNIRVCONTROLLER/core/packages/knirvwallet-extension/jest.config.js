const { pathsToModuleNameMapper } = require('ts-jest');
const { compilerOptions } = require('./tsconfig');

const jestConfig = {
  roots: ['<rootDir>'],
  modulePaths: [compilerOptions.baseUrl],
  transform: {
    '\\.[jt]sx?$': 'babel-jest',
    '\\.(svg)$': '<rootDir>/svgTransformer.js',
  },
  moduleNameMapper: {
    ...pathsToModuleNameMapper(compilerOptions.paths, {
      prefix: '<rootDir>/',
    }),
    '^knirvwallet-module$': '<rootDir>/../knirvwallet-module/src/index.ts',
  },
  setupFilesAfterEnv: ['<rootDir>/jest.setup.js'],
  testEnvironment: 'jest-environment-jsdom',
};

module.exports = jestConfig;
