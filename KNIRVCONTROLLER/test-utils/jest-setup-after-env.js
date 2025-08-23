// Jest Setup After Environment for KNIRVWALLET Tests
// Note: TypeScript files will be transformed by ts-jest before requiring

// Add browser globals for Node.js environment
const { TextEncoder, TextDecoder } = require('util');
global.TextEncoder = TextEncoder;
global.TextDecoder = TextDecoder;

// Add crypto polyfill for Node.js
const crypto = require('crypto');
Object.defineProperty(global, 'crypto', {
  value: {
    getRandomValues: (arr) => crypto.randomBytes(arr.length),
    subtle: crypto.webcrypto?.subtle
  }
});

// Global test utilities
global.testUtils = {
  mockServices: null,
  testData: null
};

// Setup before each test
beforeEach(() => {
  // Clear all storage
  if (global.localStorage) {
    global.localStorage.clear();
  }
  if (global.sessionStorage) {
    global.sessionStorage.clear();
  }

  // Reset fetch mock
  if (global.fetch && jest.isMockFunction && jest.isMockFunction(global.fetch)) {
    global.fetch.mockClear();
  }

  // Reset timers
  if (jest.clearAllTimers) {
    jest.clearAllTimers();
  }
});

// Cleanup after each test
afterEach(() => {
  // Clear all mocks
  if (jest.clearAllMocks) {
    jest.clearAllMocks();
  }

  // Reset modules
  if (jest.resetModules) {
    jest.resetModules();
  }

  // Clear storage
  if (global.localStorage) {
    global.localStorage.clear();
  }
  if (global.sessionStorage) {
    global.sessionStorage.clear();
  }

  // Reset global test utilities
  if (global.testUtils) {
    global.testUtils.mockServices = null;
    global.testUtils.testData = null;
  }
});

// Global test helpers
global.waitFor = async (condition, timeout = 5000) => {
  const startTime = Date.now();
  
  while (!condition() && (Date.now() - startTime) < timeout) {
    await new Promise(resolve => setTimeout(resolve, 100));
  }
  
  if (!condition()) {
    throw new Error(`Condition not met within ${timeout}ms`);
  }
};

global.waitForAsync = async (asyncCondition, predicate, timeout = 5000) => {
  const startTime = Date.now();
  
  while ((Date.now() - startTime) < timeout) {
    try {
      const result = await asyncCondition();
      if (predicate(result)) {
        return result;
      }
    } catch (error) {
      // Continue trying
    }
    await new Promise(resolve => setTimeout(resolve, 100));
  }
  
  throw new Error(`Async condition not met within ${timeout}ms`);
};

global.simulateDelay = (ms = 100) => {
  return new Promise(resolve => setTimeout(resolve, ms));
};

// Custom matchers
expect.extend({
  toBeValidAddress(received, prefix) {
    const pass = typeof received === 'string' && 
                 received.length > 10 && 
                 (!prefix || received.startsWith(prefix));
    
    if (pass) {
      return {
        message: () => `expected ${received} not to be a valid address`,
        pass: true,
      };
    } else {
      return {
        message: () => `expected ${received} to be a valid address${prefix ? ` with prefix ${prefix}` : ''}`,
        pass: false,
      };
    }
  },
  
  toBeValidMnemonic(received) {
    const words = received ? received.trim().split(/\s+/) : [];
    const pass = [12, 15, 18, 21, 24].includes(words.length);
    
    if (pass) {
      return {
        message: () => `expected ${received} not to be a valid mnemonic`,
        pass: true,
      };
    } else {
      return {
        message: () => `expected ${received} to be a valid mnemonic (12, 15, 18, 21, or 24 words)`,
        pass: false,
      };
    }
  },
  
  toBeValidPrivateKey(received) {
    const cleanKey = received ? received.replace(/^0x/, '') : '';
    const pass = /^[0-9a-fA-F]{64}$/.test(cleanKey);
    
    if (pass) {
      return {
        message: () => `expected ${received} not to be a valid private key`,
        pass: true,
      };
    } else {
      return {
        message: () => `expected ${received} to be a valid private key (64 hex characters)`,
        pass: false,
      };
    }
  },
  
  toBeValidTransactionHash(received) {
    const cleanHash = received ? received.replace(/^0x/, '') : '';
    const pass = /^[0-9a-fA-F]{64}$/.test(cleanHash);
    
    if (pass) {
      return {
        message: () => `expected ${received} not to be a valid transaction hash`,
        pass: true,
      };
    } else {
      return {
        message: () => `expected ${received} to be a valid transaction hash (64 hex characters)`,
        pass: false,
      };
    }
  },
  
  toHaveWalletStructure(received) {
    const hasAccounts = received && Array.isArray(received.accounts);
    const hasKeyrings = received && Array.isArray(received.keyrings);
    const hasCurrentAccountId = received && typeof received.currentAccountId === 'string';
    
    const pass = hasAccounts && hasKeyrings && hasCurrentAccountId;
    
    if (pass) {
      return {
        message: () => `expected object not to have wallet structure`,
        pass: true,
      };
    } else {
      return {
        message: () => `expected object to have wallet structure (accounts, keyrings, currentAccountId)`,
        pass: false,
      };
    }
  },
  
  toHaveTransactionStructure(received) {
    const hasFrom = received && typeof received.from === 'string';
    const hasTo = received && typeof received.to === 'string';
    const hasAmount = received && typeof received.amount === 'string';
    
    const pass = hasFrom && hasTo && hasAmount;
    
    if (pass) {
      return {
        message: () => `expected object not to have transaction structure`,
        pass: true,
      };
    } else {
      return {
        message: () => `expected object to have transaction structure (from, to, amount)`,
        pass: false,
      };
    }
  },
  
  toHaveApiResponseStructure(received) {
    const hasSuccess = received && typeof received.success === 'boolean';
    const hasDataOrError = received && (received.data !== undefined || received.error !== undefined);
    
    const pass = hasSuccess && hasDataOrError;
    
    if (pass) {
      return {
        message: () => `expected object not to have API response structure`,
        pass: true,
      };
    } else {
      return {
        message: () => `expected object to have API response structure (success, data/error)`,
        pass: false,
      };
    }
  }
});

// Global error handler for unhandled promise rejections
process.on('unhandledRejection', (reason, promise) => {
  console.error('Unhandled Rejection at:', promise, 'reason:', reason);
  // Don't exit the process in tests, just log the error
});

// Global error handler for uncaught exceptions
process.on('uncaughtException', (error) => {
  console.error('Uncaught Exception:', error);
  // Don't exit the process in tests, just log the error
});

// Mock console methods but allow them to be restored
const originalConsole = {
  log: console.log,
  warn: console.warn,
  error: console.error,
  debug: console.debug,
  info: console.info
};

global.mockConsole = () => {
  console.log = jest.fn();
  console.warn = jest.fn();
  console.error = jest.fn();
  console.debug = jest.fn();
  console.info = jest.fn();
};

global.restoreConsole = () => {
  console.log = originalConsole.log;
  console.warn = originalConsole.warn;
  console.error = originalConsole.error;
  console.debug = originalConsole.debug;
  console.info = originalConsole.info;
};

// Start with mocked console
global.mockConsole();
