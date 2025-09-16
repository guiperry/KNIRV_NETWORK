// jest-dom adds custom jest matchers for asserting on DOM nodes.
// allows you to do things like:
// expect(element).toHaveTextContent(/react/i)
// learn more: https://github.com/testing-library/jest-dom
import '@testing-library/jest-dom';
// Import type definitions for tests
/// <reference path="./types/global-test.d.ts" />
import { setupCustomMatchers } from '../test-utils/jest-matchers';

// Setup custom matchers
setupCustomMatchers();

// Setup jest-axe for accessibility testing (after DOM is set up)
try {
  import('jest-axe').then(({ toHaveNoViolations }) => {
    expect.extend(toHaveNoViolations);
  }).catch((error) => {
    // jest-axe not available or DOM not ready
    console.warn('jest-axe setup skipped:', error.message);
  });
} catch (error) {
  // jest-axe not available or DOM not ready
  console.warn('jest-axe setup skipped:', error.message);
}

// Mock fetch globally for all tests
global.fetch = jest.fn();

// Mock WebAssembly for WASM-related tests
global.WebAssembly = {
  compile: jest.fn().mockResolvedValue({}),
  instantiate: jest.fn().mockResolvedValue({ instance: {}, module: {} }),
  Module: {
    exports: jest.fn().mockReturnValue([
      { name: 'init', kind: 'function' },
      { name: 'process', kind: 'function' },
      { name: 'cleanup', kind: 'function' },
      { name: 'memory', kind: 'memory' }
    ]),
    imports: jest.fn().mockReturnValue([]),
    customSections: jest.fn().mockReturnValue([])
  },
  Instance: jest.fn(),
  Memory: jest.fn(),
  Table: jest.fn(),
  CompileError: Error,
  RuntimeError: Error,
  LinkError: Error,
  compileStreaming: jest.fn().mockResolvedValue({}),
  instantiateStreaming: jest.fn().mockResolvedValue({ instance: {}, module: {} }),
  validate: jest.fn().mockReturnValue(true),
  Global: jest.fn()
} as typeof WebAssembly;

// Mock File constructor for file upload tests
global.File = class MockFile {
  name: string;
  size: number;
  type: string;
  lastModified: number;
  content: string;

  constructor(content: string[], name: string, options: { type?: string; lastModified?: number } = {}) {
    this.content = content.join('');
    this.name = name;
    this.size = this.content.length;
    this.type = options.type || '';
    this.lastModified = Date.now();
  }

  async arrayBuffer(): Promise<ArrayBuffer> {
    const buffer = new ArrayBuffer(this.content.length);
    const view = new Uint8Array(buffer);
    for (let i = 0; i < this.content.length; i++) {
      view[i] = this.content.charCodeAt(i);
    }
    return buffer;
  }

  async text(): Promise<string> {
    return this.content;
  }

  stream(): ReadableStream {
    const content = this.content;
    return new ReadableStream({
      start(controller) {
        controller.enqueue(new TextEncoder().encode(content));
        controller.close();
      }
    });
  }

  slice(): Blob {
    return this as Blob;
  }

  bytes(): Promise<Uint8Array> {
    return Promise.resolve(new TextEncoder().encode(this.content));
  }
} as typeof Blob;

// Mock crypto.subtle for hash generation tests
Object.defineProperty(global, 'crypto', {
  value: {
    subtle: {
      digest: jest.fn().mockResolvedValue(new ArrayBuffer(32))
    },
    getRandomValues: jest.fn().mockImplementation((arr: Uint8Array | Uint16Array | Uint32Array) => {
      for (let i = 0; i < arr.length; i++) {
        arr[i] = Math.floor(Math.random() * 256);
      }
      return arr;
    })
  }
});

// Mock window.open for wallet interface tests
Object.defineProperty(global, 'window', {
  value: {
    ...global.window,
    open: jest.fn(),
    location: {
      hostname: 'localhost',
      href: 'http://localhost:3000'
    }
  },
  writable: true
});

// Mock console methods to reduce noise in tests
const originalConsole = global.console;
global.console = {
  ...originalConsole,
  log: jest.fn(),
  warn: jest.fn(),
  error: jest.fn(),
  info: jest.fn(),
  debug: jest.fn()
};

// Reset all mocks before each test
beforeEach(() => {
  jest.clearAllMocks();
  (fetch as jest.Mock).mockClear();
});

// Clean up after each test
afterEach(() => {
  jest.restoreAllMocks();
});

// Mock IntersectionObserver for components that might use it
global.IntersectionObserver = class IntersectionObserver {
  constructor() {}
  observe() {}
  unobserve() {}
  disconnect() {}
} as typeof IntersectionObserver;

// Mock ResizeObserver for components that might use it
global.ResizeObserver = class ResizeObserver {
  constructor() {}
  observe() {}
  unobserve() {}
  disconnect() {}
} as typeof ResizeObserver;

// Mock matchMedia for responsive components
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: jest.fn().mockImplementation(query => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: jest.fn(), // deprecated
    removeListener: jest.fn(), // deprecated
    addEventListener: jest.fn(),
    removeEventListener: jest.fn(),
    dispatchEvent: jest.fn(),
  })),
});

// Mock requestAnimationFrame
global.requestAnimationFrame = jest.fn(cb => setTimeout(cb, 0));
global.cancelAnimationFrame = jest.fn(id => clearTimeout(id));

// Mock performance.now for timing tests
Object.defineProperty(global, 'performance', {
  value: {
    now: jest.fn(() => Date.now())
  }
});

// Global timer management to prevent conflicts
let timersInstalled = false;

// Override Jest timer methods to prevent conflicts
const originalUseFakeTimers = jest.useFakeTimers;
const originalUseRealTimers = jest.useRealTimers;

jest.useFakeTimers = (config?: Parameters<typeof originalUseFakeTimers>[0]) => {
  if (!timersInstalled) {
    timersInstalled = true;
    return originalUseFakeTimers.call(jest, config);
  }
  return jest;
};

jest.useRealTimers = () => {
  if (timersInstalled) {
    timersInstalled = false;
    return originalUseRealTimers.call(jest);
  }
  return jest;
};

// Suppress specific warnings in tests
const originalError = console.error;
beforeAll(() => {
  console.error = (...args: unknown[]) => {
    if (
      typeof args[0] === 'string' &&
      (args[0].includes('Warning: ReactDOM.render is deprecated') ||
       args[0].includes('fake timers twice'))
    ) {
      return;
    }
    originalError.call(console, ...args);
  };
});

afterAll(() => {
  console.error = originalError;
  // Clean up timers
  if (timersInstalled) {
    jest.useRealTimers();
    timersInstalled = false;
  }
});
