// Jest setup file

// Setup canvas for xterm.js
const { createCanvas } = require('canvas');

// Mock HTMLCanvasElement.getContext
HTMLCanvasElement.prototype.getContext = function(contextType) {
  if (contextType === '2d') {
    const canvas = createCanvas(200, 200);
    return canvas.getContext('2d');
  }
  return null;
};

// Mock for window.matchMedia
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: jest.fn().mockImplementation(query => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: jest.fn(),
    removeListener: jest.fn(),
    addEventListener: jest.fn(),
    removeEventListener: jest.fn(),
    dispatchEvent: jest.fn(),
  })),
});

// Mock for IntersectionObserver
class IntersectionObserver {
  constructor(callback) {
    this.callback = callback;
  }
  observe() { return null; }
  unobserve() { return null; }
  disconnect() { return null; }
}

window.IntersectionObserver = IntersectionObserver;

// Mock for ResizeObserver
class ResizeObserver {
  constructor(callback) {
    this.callback = callback;
  }
  observe() { return null; }
  unobserve() { return null; }
  disconnect() { return null; }
}

window.ResizeObserver = ResizeObserver;