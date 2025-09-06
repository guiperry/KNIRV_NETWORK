/**
 * Jest polyfills for Node.js environment
 * Provides browser APIs that are missing in Node.js test environment
 */

// Polyfill for fetch API
import fetch, { Headers, Request, Response } from 'node-fetch';

// @ts-expect-error - Global polyfill
global.fetch = fetch;
// @ts-expect-error - Global polyfill
global.Headers = Headers;
// @ts-expect-error - Global polyfill
global.Request = Request;
// @ts-expect-error - Global polyfill
global.Response = Response;

// Polyfill for TextEncoder/TextDecoder using Node.js built-in APIs
import { TextEncoder, TextDecoder } from 'node:util';

// Global polyfill
global.TextEncoder = TextEncoder;
// @ts-expect-error - Global polyfill
global.TextDecoder = TextDecoder;

// Polyfill for crypto.subtle and crypto.getRandomValues
import { webcrypto } from 'node:crypto';
if (!global.crypto) {
  // @ts-expect-error - Global polyfill
  global.crypto = webcrypto;
}

// Polyfill for URL constructor
import { URL, URLSearchParams } from 'node:url';

// @ts-expect-error - Global polyfill
global.URL = URL;
// @ts-expect-error - Global polyfill
global.URLSearchParams = URLSearchParams;

Object.defineProperty(global, 'crypto', {
  value: {
    getRandomValues: (arr: Uint8Array) => webcrypto.getRandomValues(arr),
    subtle: webcrypto.subtle,
    randomUUID: webcrypto.randomUUID
  }
});

// Polyfill for performance API with memory
Object.defineProperty(global, 'performance', {
  value: {
    now: () => Date.now(),
    mark: () => {},
    measure: () => {},
    getEntriesByType: () => [],
    getEntriesByName: () => [],
    clearMarks: () => {},
    clearMeasures: () => {},
    memory: {
      usedJSHeapSize: 50 * 1024 * 1024, // 50MB
      totalJSHeapSize: 100 * 1024 * 1024, // 100MB
      jsHeapSizeLimit: 2 * 1024 * 1024 * 1024 // 2GB
    }
  },
  writable: true,
  configurable: true
});

// Polyfill for gc function
Object.defineProperty(global, 'gc', {
  value: jest.fn(() => {
    // Mock garbage collection
    console.log('Mock garbage collection triggered');
  }),
  writable: true,
  configurable: true
});

// Polyfill for window object in Node.js environment
Object.defineProperty(global, 'window', {
  value: {
    performance: global.performance,
    gc: global.gc,
    location: {
      href: 'http://localhost',
      origin: 'http://localhost',
      protocol: 'http:',
      host: 'localhost',
      hostname: 'localhost',
      port: '',
      pathname: '/',
      search: '',
      hash: ''
    },
    navigator: {
      userAgent: 'Node.js Test Environment'
    }
  },
  writable: true,
  configurable: true
});

// Polyfill for localStorage
const localStorageMock = {
  getItem: jest.fn((_key: string) => null),
  setItem: jest.fn((_key: string, _value: string) => {}),
  removeItem: jest.fn((_key: string) => {}),
  clear: jest.fn(() => {}),
  length: 0,
  key: jest.fn((_index: number) => null)
};

Object.defineProperty(global, 'localStorage', {
  value: localStorageMock,
  writable: true,
  configurable: true
});

// Polyfill for sessionStorage
Object.defineProperty(global, 'sessionStorage', {
  value: localStorageMock,
  writable: true,
  configurable: true
});

// Polyfill for WebAssembly
Object.defineProperty(global, 'WebAssembly', {
  value: {
    instantiate: jest.fn(() => Promise.resolve({
      instance: {
        exports: {}
      }
    })),
    compile: jest.fn(() => Promise.resolve({})),
    validate: jest.fn(() => true),
    Module: jest.fn().mockImplementation((bytes: BufferSource) => {
      // Mock WebAssembly.Module constructor
      if (!bytes || (bytes as any).byteLength < 8) {
        throw new Error('WebAssembly.Module(): expected 4 bytes, fell off end @+4');
      }
      return {};
    }),
    Instance: jest.fn().mockImplementation(() => ({
      exports: {}
    }))
  },
  writable: true,
  configurable: true
});

// Polyfill for console methods that might be missing
if (!console.debug) {
  console.debug = console.log;
}

if (!console.info) {
  console.info = console.log;
}

// Polyfill for requestAnimationFrame
Object.defineProperty(global, 'requestAnimationFrame', {
  value: (callback: FrameRequestCallback) => {
    return setTimeout(callback, 16); // ~60fps
  },
  writable: true,
  configurable: true
});

Object.defineProperty(global, 'cancelAnimationFrame', {
  value: (id: number) => {
    clearTimeout(id);
  },
  writable: true,
  configurable: true
});

// Polyfill for ResizeObserver
Object.defineProperty(global, 'ResizeObserver', {
  value: class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  },
  writable: true,
  configurable: true
});

// Polyfill for IntersectionObserver
Object.defineProperty(global, 'IntersectionObserver', {
  value: class IntersectionObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  },
  writable: true,
  configurable: true
});

// Polyfill for MutationObserver
Object.defineProperty(global, 'MutationObserver', {
  value: class MutationObserver {
    observe() {}
    disconnect() {}
    takeRecords() { return []; }
  },
  writable: true,
  configurable: true
});

console.log('✅ Jest polyfills loaded successfully');
