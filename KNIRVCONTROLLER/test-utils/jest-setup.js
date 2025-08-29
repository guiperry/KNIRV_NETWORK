// Jest Global Setup for KNIRVWALLET Tests
const { TextEncoder, TextDecoder } = require('util');

// Polyfills for Node.js environment
global.TextEncoder = TextEncoder;
global.TextDecoder = TextDecoder;

// Mock crypto for Node.js environment
const crypto = require('crypto');

// Mock Web Crypto API
global.crypto = {
  getRandomValues: (arr) => {
    const bytes = crypto.randomBytes(arr.length);
    arr.set(bytes);
    return arr;
  },
  subtle: {
    digest: async (algorithm, data) => {
      const hash = crypto.createHash(algorithm.replace('-', '').toLowerCase());
      hash.update(data);
      return hash.digest();
    },
    encrypt: async (algorithm, key, data) => {
      // Mock encryption for testing
      return new Uint8Array(data.length);
    },
    decrypt: async (algorithm, key, data) => {
      // Mock decryption for testing
      return new Uint8Array(data.length);
    },
    generateKey: async (algorithm, extractable, keyUsages) => {
      // Mock key generation
      return { type: 'secret', algorithm, extractable, usages: keyUsages };
    },
    importKey: async (format, keyData, algorithm, extractable, keyUsages) => {
      // Mock key import
      return { type: 'secret', algorithm, extractable, usages: keyUsages };
    },
    exportKey: async (format, key) => {
      // Mock key export
      return new Uint8Array(32);
    },
    sign: async (algorithm, key, data) => {
      // Mock signing
      return new Uint8Array(64);
    },
    verify: async (algorithm, key, signature, data) => {
      // Mock verification
      return true;
    }
  }
};

// Mock MediaRecorder for VoiceProcessor tests
global.MediaRecorder = class MockMediaRecorder {
  constructor(stream) {
    this.stream = stream;
    this.state = 'inactive';
    this.ondataavailable = null;
    this.onstop = null;
    this.onstart = null;
    this.onerror = null;
  }

  start() {
    this.state = 'recording';
    if (this.onstart) this.onstart();
  }

  stop() {
    this.state = 'inactive';
    if (this.onstop) this.onstop();
  }

  pause() {
    this.state = 'paused';
  }

  resume() {
    this.state = 'recording';
  }

  addEventListener(event, handler) {
    this[`on${event}`] = handler;
  }

  removeEventListener(event, handler) {
    this[`on${event}`] = null;
  }
};

global.MediaRecorder.isTypeSupported = jest.fn().mockReturnValue(true);

// Mock localStorage
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
  get length() {
    return Object.keys(this.store).length;
  },
  key: function(index) {
    const keys = Object.keys(this.store);
    return keys[index] || null;
  }
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
  get length() {
    return Object.keys(this.store).length;
  },
  key: function(index) {
    const keys = Object.keys(this.store);
    return keys[index] || null;
  }
};

// Mock IndexedDB
global.indexedDB = {
  open: jest.fn(() => ({
    onsuccess: null,
    onerror: null,
    result: {
      createObjectStore: jest.fn(),
      transaction: jest.fn(() => ({
        objectStore: jest.fn(() => ({
          add: jest.fn(),
          get: jest.fn(),
          put: jest.fn(),
          delete: jest.fn(),
          clear: jest.fn()
        }))
      }))
    }
  })),
  deleteDatabase: jest.fn()
};

// Mock WebSocket
global.WebSocket = class MockWebSocket {
  constructor(url) {
    this.url = url;
    this.readyState = 0; // CONNECTING
    this.onopen = null;
    this.onclose = null;
    this.onmessage = null;
    this.onerror = null;
    
    // Simulate connection
    setTimeout(() => {
      this.readyState = 1; // OPEN
      if (this.onopen) this.onopen({});
    }, 100);
  }
  
  send(data) {
    if (this.readyState !== 1) {
      throw new Error('WebSocket is not open');
    }
    // Mock sending data
  }
  
  close() {
    this.readyState = 3; // CLOSED
    if (this.onclose) this.onclose({});
  }
};

// Mock fetch
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

// Mock performance
global.performance = {
  now: jest.fn(() => Date.now()),
  mark: jest.fn(),
  measure: jest.fn(),
  getEntriesByName: jest.fn(() => []),
  getEntriesByType: jest.fn(() => [])
};

// Mock URL
global.URL = class MockURL {
  constructor(url, base) {
    this.href = url;
    this.origin = 'http://localhost';
    this.protocol = 'http:';
    this.host = 'localhost';
    this.hostname = 'localhost';
    this.port = '';
    this.pathname = '/';
    this.search = '';
    this.hash = '';
  }
  
  static createObjectURL(blob) {
    return 'blob:mock-url';
  }
  
  static revokeObjectURL(url) {
    // Mock implementation
  }
};

// Mock Blob
global.Blob = class MockBlob {
  constructor(parts = [], options = {}) {
    this.size = 0;
    this.type = options.type || '';
    this.parts = parts;
  }
  
  slice(start, end, contentType) {
    return new MockBlob([], { type: contentType });
  }
  
  stream() {
    return new ReadableStream();
  }
  
  text() {
    return Promise.resolve('');
  }
  
  arrayBuffer() {
    return Promise.resolve(new ArrayBuffer(0));
  }
};

// Mock File
global.File = class MockFile extends global.Blob {
  constructor(parts, name, options = {}) {
    super(parts, options);
    this.name = name;
    this.lastModified = options.lastModified || Date.now();
  }
};

// Mock FileReader
global.FileReader = class MockFileReader {
  constructor() {
    this.readyState = 0;
    this.result = null;
    this.error = null;
    this.onload = null;
    this.onerror = null;
    this.onloadend = null;
  }
  
  readAsText(file) {
    setTimeout(() => {
      this.readyState = 2;
      this.result = 'mock file content';
      if (this.onload) this.onload({ target: this });
      if (this.onloadend) this.onloadend({ target: this });
    }, 10);
  }
  
  readAsArrayBuffer(file) {
    setTimeout(() => {
      this.readyState = 2;
      this.result = new ArrayBuffer(0);
      if (this.onload) this.onload({ target: this });
      if (this.onloadend) this.onloadend({ target: this });
    }, 10);
  }
  
  readAsDataURL(file) {
    setTimeout(() => {
      this.readyState = 2;
      this.result = 'data:text/plain;base64,';
      if (this.onload) this.onload({ target: this });
      if (this.onloadend) this.onloadend({ target: this });
    }, 10);
  }
};

// Set test environment variables
process.env.NODE_ENV = 'test';
process.env.JEST_WORKER_ID = process.env.JEST_WORKER_ID || '1';

// Increase test timeout for integration tests
jest.setTimeout(30000);

// Suppress console output during tests unless explicitly needed
const originalConsole = global.console;
global.console = {
  ...originalConsole,
  log: jest.fn(),
  debug: jest.fn(),
  info: jest.fn(),
  warn: jest.fn(),
  error: jest.fn()
};

// Restore console for specific tests that need it
global.restoreConsole = () => {
  global.console = originalConsole;
};

// Mock timers
jest.useFakeTimers('legacy');
