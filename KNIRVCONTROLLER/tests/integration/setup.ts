/**
 * Integration Test Setup
 * Common setup for all integration tests with minimal mocking
 */

import '@testing-library/jest-dom';
import { TextEncoder, TextDecoder } from 'util';

// Polyfills for Node.js environment
global.TextEncoder = TextEncoder;
global.TextDecoder = TextDecoder as any;

// Mock only external network dependencies
global.fetch = jest.fn();

// Mock WebSocket for real-time features
global.WebSocket = jest.fn(() => ({
  addEventListener: jest.fn(),
  removeEventListener: jest.fn(),
  send: jest.fn(),
  close: jest.fn(),
  readyState: 1 // OPEN
})) as any;

// Mock WebAssembly for WASM tests
global.WebAssembly = {
  compile: jest.fn().mockResolvedValue({}),
  instantiate: jest.fn().mockResolvedValue({
    instance: {
      exports: {
        init: jest.fn(),
        memory: new ArrayBuffer(1024 * 1024) // 1MB
      }
    }
  }),
  Module: {
    exports: jest.fn().mockReturnValue([
      { name: 'init', kind: 'function' },
      { name: 'memory', kind: 'memory' }
    ])
  },
  Instance: jest.fn(),
  Memory: jest.fn(() => ({
    buffer: new ArrayBuffer(1024 * 1024)
  })),
  Table: jest.fn(),
  CompileError: Error,
  RuntimeError: Error,
  LinkError: Error
} as any;

// Mock crypto for secure operations
Object.defineProperty(global, 'crypto', {
  value: {
    getRandomValues: (arr: any) => {
      for (let i = 0; i < arr.length; i++) {
        arr[i] = Math.floor(Math.random() * 256);
      }
      return arr;
    },
    randomUUID: () => 'test-uuid-' + Math.random().toString(36).substr(2, 9)
  }
});

// Mock performance for timing measurements
global.performance = {
  now: jest.fn(() => Date.now()),
  mark: jest.fn(),
  measure: jest.fn(),
  getEntriesByName: jest.fn().mockReturnValue([]),
  getEntriesByType: jest.fn().mockReturnValue([]),
  clearMarks: jest.fn(),
  clearMeasures: jest.fn()
} as any;

// Mock localStorage
const localStorageMock = {
  getItem: jest.fn(),
  setItem: jest.fn(),
  removeItem: jest.fn(),
  clear: jest.fn(),
  length: 0,
  key: jest.fn()
};
Object.defineProperty(window, 'localStorage', { value: localStorageMock });

// Mock sessionStorage
const sessionStorageMock = {
  getItem: jest.fn(),
  setItem: jest.fn(),
  removeItem: jest.fn(),
  clear: jest.fn(),
  length: 0,
  key: jest.fn()
};
Object.defineProperty(window, 'sessionStorage', { value: sessionStorageMock });

// Mock URL for import.meta.url compatibility
global.URL = URL;

// Mock process.env for test environment
process.env.NODE_ENV = 'test';
process.env.INTEGRATION_TEST = 'true';

// Setup console methods for better test output
const originalConsoleError = console.error;
console.error = (...args: any[]) => {
  // Filter out React warnings in tests
  if (typeof args[0] === 'string' && args[0].includes('Warning:')) {
    return;
  }
  originalConsoleError(...args);
};

// Global test utilities
global.testUtils = {
  // Mock KNIRVGRAPH responses
  mockKNIRVGraphResponse: (skillNodeUri: string = 'knirv://skills/test/v1.0.0') => ({
    skillNodeUri,
    confidence: 0.85,
    similarErrors: ['error-001', 'error-002'],
    timestamp: Date.now()
  }),

  // Mock KNIRVROUTER responses
  mockKNIRVRouterResponse: (skillNodeUri: string = 'knirv://skills/test/v1.0.0') => ({
    status: 'SUCCESS',
    requestId: 'req-' + Math.random().toString(36).substr(2, 9),
    skillNodeUri,
    loraAdapter: new Uint8Array([1, 2, 3, 4]),
    executionTime: 150,
    networkLatency: 50,
    nrnCost: 25
  }),

  // Mock wallet responses
  mockWalletResponse: () => ({
    success: true,
    sessionId: 'session-' + Math.random().toString(36).substr(2, 9),
    walletAddress: 'knirv1wallet' + Math.random().toString(36).substr(2, 9),
    nrnBalance: 1000,
    connectionStatus: 'connected'
  }),

  // Mock agent responses
  mockAgentResponse: () => ({
    agents: [
      {
        id: 'agent-' + Math.random().toString(36).substr(2, 9),
        name: 'Test Agent',
        type: 'KNIRV-CORTEX',
        status: 'active',
        performance: 94,
        tasks: 12,
        specialization: ['test-capability'],
        nrnCost: 50,
        lastActive: '2 min ago'
      }
    ]
  }),

  // Setup fetch mock for specific endpoints
  setupFetchMock: (responses: Record<string, any>) => {
    (global.fetch as jest.Mock).mockImplementation((url: string, options?: any) => {
      const method = options?.method || 'GET';
      const key = `${method} ${url}`;
      
      if (responses[key]) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve(responses[key]),
          arrayBuffer: () => Promise.resolve(new Uint8Array([1, 2, 3, 4]).buffer)
        });
      }
      
      // Default response
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ success: true }),
        arrayBuffer: () => Promise.resolve(new Uint8Array([1, 2, 3, 4]).buffer)
      });
    });
  },

  // Wait for async operations
  waitForAsync: (ms: number = 100) => new Promise(resolve => setTimeout(resolve, ms)),

  // Create test ErrorContext
  createTestErrorContext: (overrides: any = {}) => ({
    agent_id: 'test-agent',
    agent_version: '1.0.0',
    base_model_id: 'hrm-v1',
    os: 'linux',
    architecture: 'x64',
    runtime_environment: 'node',
    error_type: 'Error',
    error_message: 'Test error message',
    stack_trace: 'Error: Test error\n    at test.js:1:1',
    source_code_snippet: 'const test = undefined.property;',
    task_description: 'Test task',
    input_data_hash: 'hash123',
    agent_state_hash: 'state456',
    timestamp: new Date(),
    additional_context: {},
    ...overrides
  })
};

// Cleanup function for after each test
global.afterEach(() => {
  jest.clearAllMocks();
  
  // Reset fetch mock
  (global.fetch as jest.Mock).mockReset();
  
  // Clear storage mocks
  localStorageMock.clear();
  sessionStorageMock.clear();
});

// Global error handler for unhandled promise rejections
process.on('unhandledRejection', (reason, promise) => {
  console.error('Unhandled Rejection at:', promise, 'reason:', reason);
});

// Increase timeout for integration tests
jest.setTimeout(30000);

export {};
