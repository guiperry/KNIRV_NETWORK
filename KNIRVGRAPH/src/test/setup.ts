import '@testing-library/jest-dom';
import * as React from 'react';

// Polyfills for Node.js environment
import { TextEncoder, TextDecoder } from 'util';

global.TextEncoder = TextEncoder;
global.TextDecoder = TextDecoder as typeof global.TextDecoder;

// Mock Web APIs
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

// Mock ResizeObserver
global.ResizeObserver = jest.fn().mockImplementation(() => ({
  observe: jest.fn(),
  unobserve: jest.fn(),
  disconnect: jest.fn(),
}));

// Mock IntersectionObserver
global.IntersectionObserver = jest.fn().mockImplementation(() => ({
  observe: jest.fn(),
  unobserve: jest.fn(),
  disconnect: jest.fn(),
}));

// Mock WebSocket
(global as typeof global & { WebSocket: unknown }).WebSocket = jest.fn().mockImplementation(() => ({
  addEventListener: jest.fn(),
  removeEventListener: jest.fn(),
  send: jest.fn(),
  close: jest.fn(),
  readyState: 1, // OPEN
}));

// Mock Canvas API
(HTMLCanvasElement.prototype.getContext as jest.Mock) = jest.fn((contextType) => {
  if (contextType === '2d') {
    return {
      fillRect: jest.fn(),
      clearRect: jest.fn(),
      getImageData: jest.fn(() => ({ data: new Array(4) })),
      putImageData: jest.fn(),
      createImageData: jest.fn(() => ({ data: new Array(4) })),
      setTransform: jest.fn(),
      drawImage: jest.fn(),
      save: jest.fn(),
      fillText: jest.fn(),
      restore: jest.fn(),
      beginPath: jest.fn(),
      moveTo: jest.fn(),
      lineTo: jest.fn(),
      closePath: jest.fn(),
      stroke: jest.fn(),
      translate: jest.fn(),
      scale: jest.fn(),
      rotate: jest.fn(),
      arc: jest.fn(),
      fill: jest.fn(),
      measureText: jest.fn(() => ({ width: 0 })),
      transform: jest.fn(),
      rect: jest.fn(),
      clip: jest.fn(),
    } as CanvasRenderingContext2D;
  }
  return null;
});

// Mock localStorage
const localStorageMock = {
  getItem: jest.fn(),
  setItem: jest.fn(),
  removeItem: jest.fn(),
  clear: jest.fn(),
  length: 0,
  key: jest.fn(),
};
Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
});

// Mock sessionStorage
Object.defineProperty(window, 'sessionStorage', {
  value: localStorageMock,
});

// Mock fetch
global.fetch = jest.fn(() =>
  Promise.resolve({
    ok: true,
    status: 200,
    json: () => Promise.resolve({}),
    text: () => Promise.resolve(''),
    blob: () => Promise.resolve(new Blob()),
    arrayBuffer: () => Promise.resolve(new ArrayBuffer(0)),
  })
) as jest.Mock;

// Note: D3.js and Three.js mocks removed as they are not used in this project

// Mock console methods to reduce noise in tests
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const originalConsole = { ...console };
beforeEach(() => {
  jest.spyOn(console, 'log').mockImplementation(() => {});
  jest.spyOn(console, 'warn').mockImplementation(() => {});
  jest.spyOn(console, 'error').mockImplementation(() => {});
});

afterEach(() => {
  jest.restoreAllMocks();
});

// Global test utilities
interface MockGraphData {
  nodes: Array<{ id: string; label: string; type: string }>;
  edges: Array<{ id: string; from: string; to: string }>;
}

interface MockBlockchainData {
  blocks: Array<{ id: string; height: number; hash: string }>;
  transactions: Array<{ id: string; from: string; to: string; amount: number }>;
}

interface MockNRVData {
  vectors: Array<{ id: string; coordinates: number[]; confidence: number }>;
}

declare global {
  const testUtils: {
    createMockGraphData: () => MockGraphData;
    createMockBlockchainData: () => MockBlockchainData;
    createMockNRVData: () => MockNRVData;
    waitFor: (condition: () => boolean, timeout?: number) => Promise<void>;
    mockComponent: (name: string) => React.ComponentType<{ children?: React.ReactNode }>;
  };
}

global.testUtils = {
  createMockGraphData: () => ({
    nodes: [
      { id: '1', label: 'Node 1', type: 'validator' },
      { id: '2', label: 'Node 2', type: 'miner' },
      { id: '3', label: 'Node 3', type: 'client' },
    ],
    edges: [
      { id: 'e1', source: '1', target: '2', weight: 0.8 },
      { id: 'e2', source: '2', target: '3', weight: 0.6 },
    ],
  }),
  
  createMockBlockchainData: () => ({
    height: 12345,
    status: 'running',
    blocks: [
      {
        height: 12345,
        hash: '0x1234567890abcdef',
        timestamp: Date.now(),
        transactions: 5,
      },
      {
        height: 12344,
        hash: '0xabcdef1234567890',
        timestamp: Date.now() - 60000,
        transactions: 3,
      },
    ],
  }),
  
  createMockNRVData: () => ({
    score: 0.75,
    stability: 0.85,
    reliability: 0.92,
    performance: 0.68,
    metrics: {
      uptime: 0.99,
      latency: 45,
      throughput: 1250,
    },
  }),
  
  waitFor: async (condition: () => boolean, timeout: number = 5000) => {
    const start = Date.now();
    while (!condition() && Date.now() - start < timeout) {
      await new Promise(resolve => setTimeout(resolve, 10));
    }
    if (!condition()) {
      throw new Error(`Condition not met within ${timeout}ms`);
    }
  },
  
  mockComponent: (name: string) => {
    return jest.fn(({ children, ...props }) => {
      return React.createElement('div', { 'data-testid': name, ...props }, children);
    });
  },
};
