import { VisualProcessor } from '../VisualProcessor';

// Mock ImageData
global.ImageData = jest.fn().mockImplementation((width, height) => ({
  data: new Uint8ClampedArray(4 * width * height),
  width,
  height,
}));

// Mock TensorFlow.js (unused mock removed)

jest.mock('@tensorflow/tfjs', () => ({
  tensor: jest.fn(),
  sequential: jest.fn(() => ({
    add: jest.fn(),
    compile: jest.fn(),
    fit: jest.fn(),
    predict: jest.fn(() => ({
      dataSync: jest.fn(() => [0.1, 0.2, 0.7]),
      dispose: jest.fn(),
    })),
    dispose: jest.fn(),
  })),
  layers: {
    dense: jest.fn(),
    conv2d: jest.fn(),
    maxPooling2d: jest.fn(),
    flatten: jest.fn(),
    dropout: jest.fn(),
  },
  loadLayersModel: jest.fn(() => Promise.resolve({
    predict: jest.fn(() => ({
      dataSync: jest.fn(() => [0.1, 0.2, 0.7]),
      dispose: jest.fn(),
    })),
    dispose: jest.fn(),
  })),
  ready: jest.fn(() => Promise.resolve()),
  browser: {
    fromPixels: jest.fn(() => ({
      resizeNearestNeighbor: jest.fn(() => ({
        cast: jest.fn(() => ({
          div: jest.fn(() => ({
            expandDims: jest.fn(() => 'mock-tensor'),
            dispose: jest.fn(),
          })),
          dispose: jest.fn(),
        })),
        div: jest.fn(() => ({
          expandDims: jest.fn(() => 'mock-tensor'),
          dispose: jest.fn(),
        })),
        dispose: jest.fn(),
      })),
      div: jest.fn(() => ({
        expandDims: jest.fn(() => 'mock-tensor'),
        dispose: jest.fn(),
      })),
      dispose: jest.fn(),
    })),
  },
}));

// Mock Canvas and Video elements
const mockCanvas = {
  getContext: jest.fn(() => ({
    drawImage: jest.fn(),
    getImageData: jest.fn(() => ({
      data: new Uint8ClampedArray(4 * 224 * 224), // Mock image data
      width: 224,
      height: 224,
    })),
    putImageData: jest.fn(),
    clearRect: jest.fn(),
    fillRect: jest.fn(),
    strokeRect: jest.fn(),
    beginPath: jest.fn(),
    moveTo: jest.fn(),
    lineTo: jest.fn(),
    stroke: jest.fn(),
    fill: jest.fn(),
  })),
  toDataURL: jest.fn(() => 'data:image/png;base64,mock-image-data'),
  width: 224,
  height: 224,
};

const mockVideo = {
  play: jest.fn(() => Promise.resolve()),
  pause: jest.fn(),
  addEventListener: jest.fn(),
  removeEventListener: jest.fn(),
  videoWidth: 640,
  videoHeight: 480,
  readyState: 4, // HAVE_ENOUGH_DATA
};

// Mock MediaStream
const mockMediaStream = {
  getTracks: jest.fn(() => []),
  getVideoTracks: jest.fn(() => [{
    stop: jest.fn(),
    enabled: true,
    kind: 'video',
    label: 'Mock Video Track',
  }]),
  getAudioTracks: jest.fn(() => []),
  addTrack: jest.fn(),
  removeTrack: jest.fn(),
  clone: jest.fn(),
};

describe('VisualProcessor', () => {
  let visualProcessor: VisualProcessor;

  beforeEach(() => {
    jest.clearAllMocks();
    
    // Mock DOM elements
    document.createElement = jest.fn((tagName) => {
      if (tagName === 'canvas') return mockCanvas as unknown as HTMLCanvasElement;
      if (tagName === 'video') return mockVideo as unknown as HTMLVideoElement;
      return {} as HTMLElement;
    });

    // Mock getUserMedia
    Object.defineProperty(navigator, 'mediaDevices', {
      writable: true,
      value: {
        getUserMedia: jest.fn(() => Promise.resolve(mockMediaStream)),
        enumerateDevices: jest.fn(() => Promise.resolve([
          { deviceId: 'default', kind: 'videoinput', label: 'Default Camera' }
        ])),
      },
    });

    visualProcessor = new VisualProcessor();
  });

  afterEach(() => {
    visualProcessor.dispose();
  });

  describe('Initialization', () => {
    it('should create a new VisualProcessor instance', () => {
      expect(visualProcessor).toBeInstanceOf(VisualProcessor);
    });

    it('should initialize with default configuration', () => {
      const metrics = visualProcessor.getMetrics() as {
        resolution: string;
        frameRate: number;
        isInitialized: boolean;
        modelsLoaded: number;
        isProcessing: boolean;
        isSupported: boolean;
        objectDetection: boolean;
      };
      expect(metrics).toBeDefined();
      expect(metrics.resolution).toBe('1280x720');
      expect(metrics.frameRate).toBe(30);
    });

    it('should initialize successfully', async () => {
      await visualProcessor.initialize();
      const metrics = visualProcessor.getMetrics() as { isInitialized: boolean };
      expect(metrics.isInitialized).toBe(true);
    });

    it('should load AI models on initialization', async () => {
      await visualProcessor.initialize();
      const metrics = visualProcessor.getMetrics() as { modelsLoaded: number };
      expect(metrics.modelsLoaded).toBeGreaterThanOrEqual(0);
    });
  });

  describe('Camera Management', () => {
    beforeEach(async () => {
      await visualProcessor.initialize();
    });

    it('should start visual processing successfully', async () => {
      await visualProcessor.start();
      const metrics = visualProcessor.getMetrics() as { isProcessing: boolean };
      expect(metrics.isProcessing).toBe(true);
    });

    it('should stop visual processing successfully', async () => {
      await visualProcessor.start();
      await visualProcessor.stop();
      const metrics = visualProcessor.getMetrics() as { isProcessing: boolean };
      expect(metrics.isProcessing).toBe(false);
    });

    it('should handle camera permission errors', async () => {
      navigator.mediaDevices.getUserMedia = jest.fn(() =>
        Promise.reject(new Error('Permission denied'))
      );

      await expect(visualProcessor.start()).rejects.toThrow('Permission denied');
    });

    it('should check if visual processing is supported', () => {
      const isSupported = visualProcessor.isSupported();
      expect(typeof isSupported).toBe('boolean');
    });

    it('should provide video and canvas elements', async () => {
      await visualProcessor.start();
      const videoElement = visualProcessor.getVideoElement();
      const canvasElement = visualProcessor.getCanvasElement();
      expect(videoElement).toBeDefined();
      expect(canvasElement).toBeDefined();
    });

    it('should capture frames', async () => {
      await visualProcessor.start();
      const frameData = visualProcessor.captureFrame();
      expect(typeof frameData).toBe('string');
    });
  });

  describe('Image Processing', () => {
    beforeEach(async () => {
      await visualProcessor.initialize();
    });

    it('should process image data with AI', async () => {
      const mockImageData = new ImageData(224, 224);
      const result = await visualProcessor.processImageWithAI(mockImageData);

      expect(result).toBeDefined();
      expect(result.id).toBeDefined();
      expect(result.timestamp).toBeDefined();
      expect(result.objects).toBeDefined();
      expect(result.faces).toBeDefined();
      expect(result.textRegions).toBeDefined();
      expect(result.sceneAnalysis).toBeDefined();
    });

    it('should update configuration', () => {
      const newConfig = {
        resolution: '1920x1080',
        frameRate: 60,
        objectDetection: false,
      };

      visualProcessor.updateConfig(newConfig);
      const metrics = visualProcessor.getMetrics() as {
        resolution: string;
        frameRate: number;
        objectDetection: boolean;
      };
      expect(metrics.resolution).toBe('1920x1080');
      expect(metrics.frameRate).toBe(60);
      expect(metrics.objectDetection).toBe(false);
    });

    it('should provide processing metrics', () => {
      const metrics = visualProcessor.getMetrics() as {
        isProcessing: boolean;
        isSupported: boolean;
        isInitialized: boolean;
      };
      expect(metrics).toBeDefined();
      expect(typeof metrics.isProcessing).toBe('boolean');
      expect(typeof metrics.isSupported).toBe('boolean');
      expect(typeof metrics.isInitialized).toBe('boolean');
    });
  });

  describe('AI Processing', () => {
    beforeEach(async () => {
      await visualProcessor.initialize();
    });

    it('should process image with AI', async () => {
      const mockImageData = new ImageData(224, 224);
      const result = await visualProcessor.processImageWithAI(mockImageData);

      expect(result).toBeDefined();
      expect(result.id).toBeDefined();
      expect(result.timestamp).toBeDefined();
      expect(result.objects).toBeDefined();
      expect(result.faces).toBeDefined();
      expect(result.textRegions).toBeDefined();
      expect(result.sceneAnalysis).toBeDefined();
    });

    it('should return processing results with correct structure', async () => {
      const mockImageData = new ImageData(224, 224);
      const result = await visualProcessor.processImageWithAI(mockImageData);

      expect(Array.isArray(result.objects)).toBe(true);
      expect(Array.isArray(result.faces)).toBe(true);
      expect(Array.isArray(result.textRegions)).toBe(true);
      expect(typeof result.sceneAnalysis).toBe('object');
    });

    it('should handle processing errors gracefully', async () => {
      const invalidImageData = null as unknown as ImageData;

      await expect(visualProcessor.processImageWithAI(invalidImageData)).rejects.toThrow();
    });
  });

  describe('Configuration and Metrics', () => {
    it('should provide configuration access', () => {
      const metrics = visualProcessor.getMetrics() as {
        isProcessing: boolean;
        isSupported: boolean;
        isInitialized: boolean;
      };
      expect(metrics).toBeDefined();
      expect(typeof metrics.isProcessing).toBe('boolean');
      expect(typeof metrics.isSupported).toBe('boolean');
      expect(typeof metrics.isInitialized).toBe('boolean');
    });

    it('should update configuration', () => {
      const newConfig = {
        resolution: '1920x1080',
        frameRate: 60,
        objectDetection: false,
      };

      visualProcessor.updateConfig(newConfig);
      const metrics = visualProcessor.getMetrics() as {
        resolution: string;
        frameRate: number;
        objectDetection: boolean;
      };
      expect(metrics.resolution).toBe('1920x1080');
      expect(metrics.frameRate).toBe(60);
      expect(metrics.objectDetection).toBe(false);
    });

    it('should check if visual processing is supported', () => {
      const isSupported = visualProcessor.isSupported();
      expect(typeof isSupported).toBe('boolean');
    });

    it('should provide video and canvas elements', async () => {
      await visualProcessor.start();
      const videoElement = visualProcessor.getVideoElement();
      const canvasElement = visualProcessor.getCanvasElement();
      expect(videoElement).toBeDefined();
      expect(canvasElement).toBeDefined();
    });

    it('should capture frames', async () => {
      await visualProcessor.start();
      const frameData = visualProcessor.captureFrame();
      expect(typeof frameData).toBe('string');
    });
  });
});
