// Mock WebAssembly for Jest environment
global.WebAssembly = {
  instantiate: jest.fn().mockResolvedValue({
    instance: {
      exports: {
        agentCoreExecute: jest.fn().mockResolvedValue('{"success": true, "result": "mock response"}'),
        agentCoreInitialize: jest.fn().mockResolvedValue(true),
        agentCoreDispose: jest.fn().mockResolvedValue(true),
        memory: { buffer: new ArrayBuffer(64 * 1024) }
      }
    },
    module: {}
  }),
  compile: jest.fn().mockResolvedValue({}),
  Module: jest.fn(),
  Instance: jest.fn(),
  Memory: jest.fn().mockImplementation(function(options) {
    this.buffer = new ArrayBuffer((options?.initial || 1) * 64 * 1024);
    return this;
  }),
  Table: jest.fn(),
  CompileError: Error,
  RuntimeError: Error
};

// Mock MediaRecorder for VoiceProcessor tests
global.MediaRecorder = jest.fn().mockImplementation(() => ({
  start: jest.fn(),
  stop: jest.fn(),
  pause: jest.fn(),
  resume: jest.fn(),
  addEventListener: jest.fn(),
  removeEventListener: jest.fn(),
  state: 'inactive',
  mimeType: 'audio/webm',
  ondataavailable: null,
  onerror: null,
  onpause: null,
  onresume: null,
  onstart: null,
  onstop: null
}));

// Mock navigator.mediaDevices for VoiceProcessor tests
Object.defineProperty(global, 'navigator', {
  value: {
    ...global.navigator,
    mediaDevices: {
      getUserMedia: jest.fn().mockResolvedValue({
        getTracks: jest.fn().mockReturnValue([
          {
            stop: jest.fn(),
            kind: 'audio',
            enabled: true
          }
        ])
      })
    }
  },
  writable: true
});

// Mock WebSocket for integration tests
global.WebSocket = jest.fn().mockImplementation(() => ({
  addEventListener: jest.fn(),
  removeEventListener: jest.fn(),
  send: jest.fn(),
  close: jest.fn(),
  readyState: 1, // OPEN
  CONNECTING: 0,
  OPEN: 1,
  CLOSING: 2,
  CLOSED: 3
}));

// Mock fetch for network requests
global.fetch = jest.fn().mockResolvedValue({
  ok: true,
  status: 200,
  json: jest.fn().mockResolvedValue({}),
  text: jest.fn().mockResolvedValue(''),
  blob: jest.fn().mockResolvedValue(new Blob())
});

// Mock crypto for random number generation
Object.defineProperty(global, 'crypto', {
  value: {
    getRandomValues: jest.fn().mockImplementation((array) => {
      for (let i = 0; i < array.length; i++) {
        array[i] = Math.floor(Math.random() * 256);
      }
      return array;
    }),
    randomUUID: jest.fn().mockReturnValue('mock-uuid-1234-5678-9abc-def0'),
    randomBytes: jest.fn().mockImplementation((size) => {
      // Return deterministic bytes for testing
      const buffer = Buffer.alloc(size);
      for (let i = 0; i < size; i++) {
        buffer[i] = (i * 17 + 42) % 256; // Deterministic pattern
      }
      return buffer;
    }),
    pbkdf2Sync: jest.fn().mockImplementation((password, salt, iterations, keylen, digest) => {
      // Return deterministic hash for testing
      if (password === 'test123' && salt === 'testsalt' && keylen === 32) {
        return Buffer.from('a6a22ebe2861e3c544e18232f0a909cb8b3def839e3ca751b885f220636b0a90', 'hex');
      }
      // Default deterministic response
      const buffer = Buffer.alloc(keylen);
      const hash = password + salt.toString();
      for (let i = 0; i < keylen; i++) {
        buffer[i] = hash.charCodeAt(i % hash.length) % 256;
      }
      return buffer;
    }),
    pbkdf2: jest.fn().mockImplementation((password, salt, iterations, keylen, digest, callback) => {
      // Simple mock that returns a deterministic buffer based on password
      const buffer = Buffer.alloc(keylen);
      const hash = password + salt.toString();
      for (let i = 0; i < keylen; i++) {
        buffer[i] = hash.charCodeAt(i % hash.length) % 256;
      }
      setTimeout(() => callback(null, buffer), 0);
    }),
    createCipherGCM: jest.fn().mockImplementation(() => ({
      update: jest.fn().mockReturnValue('encrypted'),
      final: jest.fn().mockReturnValue(''),
      getAuthTag: jest.fn().mockReturnValue(Buffer.from('authtag'))
    })),
    createDecipherGCM: jest.fn().mockImplementation(() => ({
      update: jest.fn().mockReturnValue('decrypted'),
      final: jest.fn().mockReturnValue(''),
      setAuthTag: jest.fn()
    }))
  },
  writable: true
});

// Mock URL constructor for import.meta.url issues
global.URL = jest.fn().mockImplementation((url, base) => ({
  href: `${base || 'file://'}/${url}`,
  toString: () => `${base || 'file://'}/${url}`
}));

// Mock TextEncoder/TextDecoder
global.TextEncoder = jest.fn().mockImplementation(() => ({
  encode: jest.fn().mockImplementation((str) => new Uint8Array(Buffer.from(str, 'utf8')))
}));

global.TextDecoder = jest.fn().mockImplementation(() => ({
  decode: jest.fn().mockImplementation((buffer) => Buffer.from(buffer).toString('utf8'))
}));

// Mock performance for timing
global.performance = {
  now: jest.fn().mockReturnValue(Date.now()),
  mark: jest.fn(),
  measure: jest.fn(),
  getEntriesByName: jest.fn().mockReturnValue([])
};

// Mock console methods to reduce noise in tests
const originalConsole = global.console;
global.console = {
  ...originalConsole,
  warn: jest.fn(),
  error: jest.fn(),
  log: jest.fn(),
  info: jest.fn(),
  debug: jest.fn()
};

// Mock React Native dependencies
jest.mock('react-native-safe-area-context', () => ({
  SafeAreaView: ({ children }) => children,
  useSafeAreaInsets: () => ({ top: 0, bottom: 0, left: 0, right: 0 })
}));

jest.mock('@cosmjs/cosmwasm-stargate', () => ({
  SigningCosmWasmClient: {
    connectWithSigner: jest.fn().mockResolvedValue({
      getBalance: jest.fn().mockResolvedValue({ amount: '1000000', denom: 'uxion' }),
      execute: jest.fn().mockResolvedValue({ transactionHash: 'mock-hash' }),
      queryContractSmart: jest.fn().mockResolvedValue({ result: 'mock-result' })
    })
  }
}));

jest.mock('react-native', () => ({
  StyleSheet: {
    create: jest.fn().mockImplementation((styles) => styles)
  },
  View: 'View',
  Text: 'Text',
  TouchableOpacity: 'TouchableOpacity',
  ScrollView: 'ScrollView',
  TextInput: 'TextInput',
  Alert: {
    alert: jest.fn()
  },
  Dimensions: {
    get: jest.fn().mockReturnValue({ width: 375, height: 812 })
  }
}));

// Mock document.elementsFromPoint for jest-axe (only if not available)
if (!document.elementsFromPoint) {
  Object.defineProperty(document, 'elementsFromPoint', {
    value: jest.fn().mockReturnValue([]),
    writable: true
  });
}
