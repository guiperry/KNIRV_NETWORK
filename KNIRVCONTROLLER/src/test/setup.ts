import '@testing-library/jest-dom';
import * as React from 'react';

// Mock WASM module
jest.mock('../wasm-pkg/knirv_cortex_wasm', () => ({
  greet: jest.fn(() => 'Hello from WASM!'),
  process_data: jest.fn((data: unknown) => data),
  initialize_cortex: jest.fn(() => true),
  get_version: jest.fn(() => '1.0.0'),
}));

// Mock TensorFlow.js
jest.mock('@tensorflow/tfjs', () => ({
  tensor: jest.fn(),
  sequential: jest.fn(() => ({
    add: jest.fn(),
    compile: jest.fn(),
    fit: jest.fn(),
    predict: jest.fn(),
  })),
  layers: {
    dense: jest.fn(),
    dropout: jest.fn(),
  },
  loadLayersModel: jest.fn(),
  ready: jest.fn(() => Promise.resolve()),
}));

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

// Mock Web Audio API
Object.defineProperty(window, 'AudioContext', {
  writable: true,
  value: jest.fn().mockImplementation(() => ({
    createAnalyser: jest.fn(() => ({
      connect: jest.fn(),
      disconnect: jest.fn(),
      fftSize: 2048,
      frequencyBinCount: 1024,
      getByteFrequencyData: jest.fn(),
      getByteTimeDomainData: jest.fn(),
    })),
    createGain: jest.fn(() => ({
      connect: jest.fn(),
      disconnect: jest.fn(),
      gain: { value: 1 },
    })),
    createOscillator: jest.fn(() => ({
      connect: jest.fn(),
      disconnect: jest.fn(),
      start: jest.fn(),
      stop: jest.fn(),
      frequency: { value: 440 },
    })),
    createMediaStreamSource: jest.fn(() => ({
      connect: jest.fn(),
      disconnect: jest.fn(),
    })),
    destination: {},
    sampleRate: 44100,
    state: 'running',
    suspend: jest.fn(),
    resume: jest.fn(),
    close: jest.fn(),
  })),
});

// Mock getUserMedia
Object.defineProperty(navigator, 'mediaDevices', {
  writable: true,
  value: {
    getUserMedia: jest.fn(() => Promise.resolve({
      getTracks: jest.fn(() => []),
      getVideoTracks: jest.fn(() => []),
      getAudioTracks: jest.fn(() => []),
      addTrack: jest.fn(),
      removeTrack: jest.fn(),
      clone: jest.fn(),
    })),
    enumerateDevices: jest.fn(() => Promise.resolve([])),
  },
});

// Mock WebGL context
const mockWebGLContext = {
  canvas: {},
  drawingBufferWidth: 300,
  drawingBufferHeight: 150,
  getContextAttributes: jest.fn(() => ({})),
  isContextLost: jest.fn(() => false),
  getSupportedExtensions: jest.fn(() => []),
  getExtension: jest.fn(),
  activeTexture: jest.fn(),
  attachShader: jest.fn(),
  bindAttribLocation: jest.fn(),
  bindBuffer: jest.fn(),
  bindFramebuffer: jest.fn(),
  bindRenderbuffer: jest.fn(),
  bindTexture: jest.fn(),
  blendColor: jest.fn(),
  blendEquation: jest.fn(),
  blendEquationSeparate: jest.fn(),
  blendFunc: jest.fn(),
  blendFuncSeparate: jest.fn(),
  bufferData: jest.fn(),
  bufferSubData: jest.fn(),
  checkFramebufferStatus: jest.fn(() => 36053), // FRAMEBUFFER_COMPLETE
  clear: jest.fn(),
  clearColor: jest.fn(),
  clearDepth: jest.fn(),
  clearStencil: jest.fn(),
  colorMask: jest.fn(),
  compileShader: jest.fn(),
  createBuffer: jest.fn(() => ({})),
  createFramebuffer: jest.fn(() => ({})),
  createProgram: jest.fn(() => ({})),
  createRenderbuffer: jest.fn(() => ({})),
  createShader: jest.fn(() => ({})),
  createTexture: jest.fn(() => ({})),
  cullFace: jest.fn(),
  deleteBuffer: jest.fn(),
  deleteFramebuffer: jest.fn(),
  deleteProgram: jest.fn(),
  deleteRenderbuffer: jest.fn(),
  deleteShader: jest.fn(),
  deleteTexture: jest.fn(),
  depthFunc: jest.fn(),
  depthMask: jest.fn(),
  depthRange: jest.fn(),
  detachShader: jest.fn(),
  disable: jest.fn(),
  disableVertexAttribArray: jest.fn(),
  drawArrays: jest.fn(),
  drawElements: jest.fn(),
  enable: jest.fn(),
  enableVertexAttribArray: jest.fn(),
  finish: jest.fn(),
  flush: jest.fn(),
  framebufferRenderbuffer: jest.fn(),
  framebufferTexture2D: jest.fn(),
  frontFace: jest.fn(),
  generateMipmap: jest.fn(),
  getActiveAttrib: jest.fn(() => ({ name: 'test', size: 1, type: 5126 })),
  getActiveUniform: jest.fn(() => ({ name: 'test', size: 1, type: 5126 })),
  getAttachedShaders: jest.fn(() => []),
  getAttribLocation: jest.fn(() => 0),
  getBufferParameter: jest.fn(),
  getParameter: jest.fn(),
  getError: jest.fn(() => 0), // NO_ERROR
  getFramebufferAttachmentParameter: jest.fn(),
  getProgramParameter: jest.fn(() => true),
  getProgramInfoLog: jest.fn(() => ''),
  getRenderbufferParameter: jest.fn(),
  getShaderParameter: jest.fn(() => true),
  getShaderPrecisionFormat: jest.fn(() => ({ rangeMin: 1, rangeMax: 1, precision: 1 })),
  getShaderInfoLog: jest.fn(() => ''),
  getShaderSource: jest.fn(() => ''),
  getTexParameter: jest.fn(),
  getUniform: jest.fn(),
  getUniformLocation: jest.fn(() => ({})),
  getVertexAttrib: jest.fn(),
  getVertexAttribOffset: jest.fn(() => 0),
  hint: jest.fn(),
  isBuffer: jest.fn(() => false),
  isEnabled: jest.fn(() => false),
  isFramebuffer: jest.fn(() => false),
  isProgram: jest.fn(() => false),
  isRenderbuffer: jest.fn(() => false),
  isShader: jest.fn(() => false),
  isTexture: jest.fn(() => false),
  lineWidth: jest.fn(),
  linkProgram: jest.fn(),
  pixelStorei: jest.fn(),
  polygonOffset: jest.fn(),
  readPixels: jest.fn(),
  renderbufferStorage: jest.fn(),
  sampleCoverage: jest.fn(),
  scissor: jest.fn(),
  shaderSource: jest.fn(),
  stencilFunc: jest.fn(),
  stencilFuncSeparate: jest.fn(),
  stencilMask: jest.fn(),
  stencilMaskSeparate: jest.fn(),
  stencilOp: jest.fn(),
  stencilOpSeparate: jest.fn(),
  texImage2D: jest.fn(),
  texParameterf: jest.fn(),
  texParameteri: jest.fn(),
  texSubImage2D: jest.fn(),
  uniform1f: jest.fn(),
  uniform1fv: jest.fn(),
  uniform1i: jest.fn(),
  uniform1iv: jest.fn(),
  uniform2f: jest.fn(),
  uniform2fv: jest.fn(),
  uniform2i: jest.fn(),
  uniform2iv: jest.fn(),
  uniform3f: jest.fn(),
  uniform3fv: jest.fn(),
  uniform3i: jest.fn(),
  uniform3iv: jest.fn(),
  uniform4f: jest.fn(),
  uniform4fv: jest.fn(),
  uniform4i: jest.fn(),
  uniform4iv: jest.fn(),
  uniformMatrix2fv: jest.fn(),
  uniformMatrix3fv: jest.fn(),
  uniformMatrix4fv: jest.fn(),
  useProgram: jest.fn(),
  validateProgram: jest.fn(),
  vertexAttrib1f: jest.fn(),
  vertexAttrib1fv: jest.fn(),
  vertexAttrib2f: jest.fn(),
  vertexAttrib2fv: jest.fn(),
  vertexAttrib3f: jest.fn(),
  vertexAttrib3fv: jest.fn(),
  vertexAttrib4f: jest.fn(),
  vertexAttrib4fv: jest.fn(),
  vertexAttribPointer: jest.fn(),
  viewport: jest.fn(),
};

(HTMLCanvasElement.prototype.getContext as jest.Mock) = jest.fn((contextType: string) => {
  if (contextType === 'webgl' || contextType === 'experimental-webgl') {
    return mockWebGLContext;
  }
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
      canvas: {} as HTMLCanvasElement,
      globalAlpha: 1,
      globalCompositeOperation: 'source-over',
      // Add missing required properties for CanvasRenderingContext2D
      fillStyle: '#000000',
      strokeStyle: '#000000',
      lineWidth: 1,
      lineCap: 'butt' as CanvasLineCap,
      lineJoin: 'miter' as CanvasLineJoin,
      miterLimit: 10,
      lineDashOffset: 0,
      shadowOffsetX: 0,
      shadowOffsetY: 0,
      shadowBlur: 0,
      shadowColor: 'rgba(0, 0, 0, 0)',
      font: '10px sans-serif',
      textAlign: 'start' as CanvasTextAlign,
      textBaseline: 'alphabetic' as CanvasTextBaseline,
      direction: 'inherit' as CanvasDirection,
      imageSmoothingEnabled: true,
      imageSmoothingQuality: 'low' as ImageSmoothingQuality,
      filter: 'none',
      isPointInPath: jest.fn(),
      isPointInStroke: jest.fn(),
      getLineDash: jest.fn(() => []),
      setLineDash: jest.fn(),
      createLinearGradient: jest.fn(),
      createRadialGradient: jest.fn(),
      createPattern: jest.fn(),
      bezierCurveTo: jest.fn(),
      arcTo: jest.fn(),
      ellipse: jest.fn(),
      strokeText: jest.fn(),
      quadraticCurveTo: jest.fn(),
      getTransform: jest.fn(),
      resetTransform: jest.fn(),
    } as unknown as CanvasRenderingContext2D;
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

// Mock MediaRecorder
const MockMediaRecorder = jest.fn().mockImplementation(() => ({
  start: jest.fn(),
  stop: jest.fn(),
  pause: jest.fn(),
  resume: jest.fn(),
  addEventListener: jest.fn(),
  removeEventListener: jest.fn(),
  state: 'inactive',
  mimeType: 'audio/webm',
  stream: null,
  ondataavailable: null,
  onerror: null,
  onpause: null,
  onresume: null,
  onstart: null,
  onstop: null,
})) as unknown as {
  new (stream: MediaStream, options?: MediaRecorderOptions): MediaRecorder;
  prototype: MediaRecorder;
  isTypeSupported(type: string): boolean;
};

(MockMediaRecorder as unknown as { isTypeSupported: jest.Mock }).isTypeSupported = jest.fn().mockReturnValue(true);
global.MediaRecorder = MockMediaRecorder as unknown as typeof MediaRecorder;

// Mock SpeechSynthesisUtterance
global.SpeechSynthesisUtterance = jest.fn().mockImplementation((text) => ({
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
  onboundary: null,
}));

// Mock speechSynthesis
Object.defineProperty(window, 'speechSynthesis', {
  value: {
    speak: jest.fn(),
    cancel: jest.fn(),
    pause: jest.fn(),
    resume: jest.fn(),
    getVoices: jest.fn().mockReturnValue([
      { name: 'Test Voice', lang: 'en-US', default: true }
    ]),
    speaking: false,
    pending: false,
    paused: false,
    onvoiceschanged: null,
  },
  writable: true,
});

// Mock SpeechRecognition
const mockSpeechRecognition = jest.fn().mockImplementation(() => ({
  start: jest.fn(),
  stop: jest.fn(),
  abort: jest.fn(),
  continuous: true,
  interimResults: true,
  lang: 'en-US',
  onstart: null,
  onend: null,
  onresult: null,
  onerror: null,
  onspeechstart: null,
  onspeechend: null,
  onsoundstart: null,
  onsoundend: null,
  onaudiostart: null,
  onaudioend: null,
  onnomatch: null,
}));

Object.defineProperty(window, 'SpeechRecognition', {
  value: mockSpeechRecognition,
  writable: true,
});

Object.defineProperty(window, 'webkitSpeechRecognition', {
  value: mockSpeechRecognition,
  writable: true,
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

// Mock console methods to reduce noise in tests
beforeEach(() => {
  jest.spyOn(console, 'log').mockImplementation(() => {});
  jest.spyOn(console, 'warn').mockImplementation(() => {});
  jest.spyOn(console, 'error').mockImplementation(() => {});
});

afterEach(() => {
  jest.restoreAllMocks();
});

// Global test utilities
declare global {
  const testUtils: {
    createMockEvent: (type: string, data?: unknown) => Event;
    waitFor: (condition: () => boolean, timeout?: number) => Promise<void>;
    mockComponent: (name: string) => React.ComponentType<Record<string, unknown>>;
  };
}

interface GlobalTestUtils {
  createMockEvent: (type: string, data?: Record<string, unknown>) => Event;
  waitFor: (condition: () => boolean, timeout?: number) => Promise<void>;
  mockComponent: (name: string) => React.ComponentType<Record<string, unknown>>;
}

(global as typeof global & { testUtils: GlobalTestUtils }).testUtils = {
  createMockEvent: (type: string, data: Record<string, unknown> = {}) => {
    const event = new Event(type);
    Object.assign(event, data);
    return event;
  },
  
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
    return jest.fn(({ children, ...props }: { children?: React.ReactNode; [key: string]: unknown }) => {
      return React.createElement('div', { 'data-testid': name, ...props }, children);
    }) as React.ComponentType<Record<string, unknown>>;
  },
};
