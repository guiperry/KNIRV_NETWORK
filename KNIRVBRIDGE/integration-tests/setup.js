// Global test setup for KNIRVWALLET integration tests
global.console = {
  ...console,
  // Suppress console.log during tests unless explicitly needed
  log: jest.fn(),
  debug: jest.fn(),
  info: jest.fn(),
  warn: jest.fn(),
  error: jest.fn(),
};

// Set up test environment variables
process.env.NODE_ENV = 'test';

// Global test timeout
jest.setTimeout(30000);

// Mock browser APIs that might be used in wallet operations
global.crypto = require('crypto').webcrypto || {
  getRandomValues: (arr) => {
    const crypto = require('crypto');
    const bytes = crypto.randomBytes(arr.length);
    arr.set(bytes);
    return arr;
  },
  subtle: {
    digest: async (algorithm, data) => {
      const crypto = require('crypto');
      const hash = crypto.createHash(algorithm.replace('-', '').toLowerCase());
      hash.update(data);
      return hash.digest();
    },
  },
};

// Mock localStorage for browser wallet tests
global.localStorage = {
  store: {},
  getItem: function(key) {
    return this.store[key] || null;
  },
  setItem: function(key, value) {
    this.store[key] = value.toString();
  },
  removeItem: function(key) {
    delete this.store[key];
  },
  clear: function() {
    this.store = {};
  },
};

// Mock sessionStorage
global.sessionStorage = {
  store: {},
  getItem: function(key) {
    return this.store[key] || null;
  },
  setItem: function(key, value) {
    this.store[key] = value.toString();
  },
  removeItem: function(key) {
    delete this.store[key];
  },
  clear: function() {
    this.store = {};
  },
};

// Clean up after each test
afterEach(() => {
  global.localStorage.clear();
  global.sessionStorage.clear();
  jest.clearAllMocks();
});
