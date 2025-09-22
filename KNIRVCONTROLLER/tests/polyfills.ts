/**
 * Jest polyfills for Node.js environment
 * Provides browser APIs that are missing in Node.js test environment
 */

// Simple mock fetch API for Jest environment
// @ts-expect-error - Global polyfill
global.fetch = jest.fn(() =>
  Promise.resolve({
    ok: true,
    status: 200,
    json: () => Promise.resolve({}),
    text: () => Promise.resolve(''),
    blob: () => Promise.resolve(new Blob()),
    arrayBuffer: () => Promise.resolve(new ArrayBuffer(0))
  })
);

global.Headers = jest.fn().mockImplementation((init) => {
  const headers = new Map();
  if (init) {
    Object.entries(init).forEach(([key, value]) => headers.set(key, value));
  }
  return {
    get: (name: string) => headers.get(name),
    set: (name: string, value: string) => headers.set(name, value),
    has: (name: string) => headers.has(name),
    delete: (name: string) => headers.delete(name),
    entries: () => headers.entries(),
    keys: () => headers.keys(),
    values: () => headers.values()
  };
});

global.Request = jest.fn().mockImplementation((url, options = {}) => ({
  url,
  method: options.method || 'GET',
  headers: new global.Headers(options.headers),
  body: options.body
}));

// @ts-expect-error - Global polyfill
global.Response = jest.fn().mockImplementation((body, options = {}) => ({
  ok: options.status ? options.status >= 200 && options.status < 300 : true,
  status: options.status || 200,
  statusText: options.statusText || 'OK',
  headers: new global.Headers(options.headers),
  json: () => Promise.resolve(typeof body === 'string' ? JSON.parse(body) : body),
  text: () => Promise.resolve(typeof body === 'string' ? body : JSON.stringify(body)),
  blob: () => Promise.resolve(new Blob([body])),
  arrayBuffer: () => Promise.resolve(new ArrayBuffer(0))
}));

// Polyfill for TextEncoder/TextDecoder using Node.js built-in APIs
import { TextEncoder, TextDecoder } from 'node:util';

// Global polyfill - ensure they're available globally
if (!global.TextEncoder) {
  global.TextEncoder = TextEncoder;
}
if (!global.TextDecoder) {
  // @ts-expect-error - Global polyfill
  global.TextDecoder = TextDecoder;
}

// Also ensure they're available on globalThis
if (typeof globalThis !== 'undefined') {
  if (!globalThis.TextEncoder) {
    globalThis.TextEncoder = TextEncoder;
  }
  if (!globalThis.TextDecoder) {
    // @ts-expect-error - Global polyfill
    globalThis.TextDecoder = TextDecoder;
  }
}

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

// Enhanced timer polyfills for @testing-library/react-native
const timers = {
  setTimeout: global.setTimeout.bind(global),
  clearTimeout: global.clearTimeout.bind(global),
  setInterval: global.setInterval.bind(global),
  clearInterval: global.clearInterval.bind(global),
  setImmediate: global.setImmediate?.bind(global) || ((fn: () => void) => setTimeout(fn, 0)),
  clearImmediate: global.clearImmediate?.bind(global) || clearTimeout
};

// Ensure timer functions are available on globalThis for @testing-library/react-native
Object.assign(globalThis, timers);

// Polyfill for window object in Node.js environment with event handling
const eventListeners = new Map<string, EventListener[]>();

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
    },
    // Add event listener support for ErrorHandler and other services
    addEventListener: jest.fn((event: string, listener: EventListener) => {
      if (!eventListeners.has(event)) {
        eventListeners.set(event, []);
      }
      eventListeners.get(event)!.push(listener);
    }),
    removeEventListener: jest.fn((event: string, listener: EventListener) => {
      if (eventListeners.has(event)) {
        const listeners = eventListeners.get(event)!;
        const index = listeners.indexOf(listener);
        if (index > -1) {
          listeners.splice(index, 1);
        }
      }
    }),
    dispatchEvent: jest.fn((event: Event) => {
      const listeners = eventListeners.get(event.type) || [];
      listeners.forEach(listener => listener(event));
      return true;
    }),
    // Add timer functions to window object
    ...timers
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

// Polyfill for PerformanceObserver
Object.defineProperty(global, 'PerformanceObserver', {
  value: jest.fn().mockImplementation((_callback: (list: PerformanceObserverEntryList, observer: PerformanceObserver) => void) => ({
    observe: jest.fn(),
    disconnect: jest.fn(),
    takeRecords: jest.fn(() => [])
  })),
  writable: true,
  configurable: true
});

// Polyfill for Speech APIs (SpeechSynthesis, SpeechSynthesisUtterance)
Object.defineProperty(global, 'SpeechSynthesisUtterance', {
  value: jest.fn().mockImplementation((text?: string) => ({
    text: text || '',
    lang: 'en-US',
    voice: null,
    volume: 1,
    rate: 1,
    pitch: 1,
    onstart: null,
    onend: null,
    onerror: null,
    onpause: null,
    onresume: null,
    onmark: null,
    onboundary: null
  })),
  writable: true,
  configurable: true
});

Object.defineProperty(global, 'speechSynthesis', {
  value: {
    speak: jest.fn(),
    cancel: jest.fn(),
    pause: jest.fn(),
    resume: jest.fn(),
    getVoices: jest.fn(() => []),
    speaking: false,
    pending: false,
    paused: false,
    onvoiceschanged: null
  },
  writable: true,
  configurable: true
});

// Add Speech APIs to window object as well
if (global.window) {
  (global.window as any).SpeechSynthesisUtterance = global.SpeechSynthesisUtterance;
  (global.window as any).speechSynthesis = global.speechSynthesis;
}

// Polyfill for IE-specific event methods that React DOM might try to use
// This fixes the "activeElement.attachEvent is not a function" error in JSDOM
if (typeof document !== 'undefined') {
  const originalCreateElement = document.createElement;
  document.createElement = function(tagName: string, options?: ElementCreationOptions) {
    const element = originalCreateElement.call(this, tagName, options);

    // Add IE-specific event methods to all elements
    if (!(element as any).attachEvent) {
      (element as any).attachEvent = function(event: string, handler: EventListener) {
        return element.addEventListener(event.replace('on', ''), handler);
      };
    }

    if (!(element as any).detachEvent) {
      (element as any).detachEvent = function(event: string, handler: EventListener) {
        return element.removeEventListener(event.replace('on', ''), handler);
      };
    }

    return element;
  };

  // Store the original activeElement getter to avoid recursion
  const originalActiveElementDescriptor = Object.getOwnPropertyDescriptor(Document.prototype, 'activeElement') ||
    Object.getOwnPropertyDescriptor(document, 'activeElement');

  // Add these methods to document.body and other common elements
  const addEventMethodsToElement = (element: Element) => {
    if (element && !(element as any).attachEvent) {
      (element as any).attachEvent = function(event: string, handler: EventListener) {
        return element.addEventListener(event.replace('on', ''), handler);
      };
      (element as any).detachEvent = function(event: string, handler: EventListener) {
        return element.removeEventListener(event.replace('on', ''), handler);
      };
    }
  };

  // Add methods to body and html elements
  if (document.body) addEventMethodsToElement(document.body);
  if (document.documentElement) addEventMethodsToElement(document.documentElement);

  // Override activeElement getter to add methods when accessed
  Object.defineProperty(document, 'activeElement', {
    get() {
      let activeEl;
      if (originalActiveElementDescriptor && originalActiveElementDescriptor.get) {
        activeEl = originalActiveElementDescriptor.get.call(this);
      } else {
        activeEl = document.body; // Fallback
      }

      if (activeEl) {
        addEventMethodsToElement(activeEl);
      }

      return activeEl;
    },
    configurable: true
  });
}

console.log('✅ Jest polyfills loaded successfully');
