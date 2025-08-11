import { VoiceProcessor } from '../VoiceProcessor';

// Mock Web Audio API
const mockAudioContext = {
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
};

// Mock MediaStream
const mockMediaStream = {
  getTracks: jest.fn(() => []),
  getAudioTracks: jest.fn(() => [{
    stop: jest.fn(),
    enabled: true,
    kind: 'audio',
    label: 'Mock Audio Track',
  }]),
  getVideoTracks: jest.fn(() => []),
  addTrack: jest.fn(),
  removeTrack: jest.fn(),
  clone: jest.fn(),
};

describe('VoiceProcessor', () => {
  let voiceProcessor: VoiceProcessor;

  beforeEach(() => {
    // Reset mocks
    jest.clearAllMocks();
    
    // Mock AudioContext
    (global as any).AudioContext = jest.fn(() => mockAudioContext);
    (global as any).webkitAudioContext = jest.fn(() => mockAudioContext);
    
    // Mock getUserMedia
    Object.defineProperty(navigator, 'mediaDevices', {
      writable: true,
      value: {
        getUserMedia: jest.fn(() => Promise.resolve(mockMediaStream)),
        enumerateDevices: jest.fn(() => Promise.resolve([
          { deviceId: 'default', kind: 'audioinput', label: 'Default Microphone' }
        ])),
      },
    });

    voiceProcessor = new VoiceProcessor();
  });

  afterEach(() => {
    voiceProcessor.dispose();
  });

  describe('Initialization', () => {
    it('should create a new VoiceProcessor instance', () => {
      expect(voiceProcessor).toBeInstanceOf(VoiceProcessor);
    });

    it('should initialize with default configuration', () => {
      const config = voiceProcessor.getConfig();
      expect(config).toBeDefined();
      expect(config.sampleRate).toBe(44100);
      expect(config.bufferSize).toBe(4096);
    });

    it('should initialize audio context', async () => {
      await voiceProcessor.initialize();
      expect(voiceProcessor.isInitialized()).toBe(true);
    });
  });

  describe('Audio Input Management', () => {
    beforeEach(async () => {
      await voiceProcessor.initialize();
    });

    it('should start audio input successfully', async () => {
      await voiceProcessor.startListening();
      expect(voiceProcessor.isListening()).toBe(true);
    });

    it('should stop audio input successfully', async () => {
      await voiceProcessor.startListening();
      await voiceProcessor.stopListening();
      expect(voiceProcessor.isListening()).toBe(false);
    });

    it('should handle getUserMedia errors gracefully', async () => {
      // Mock getUserMedia to reject
      navigator.mediaDevices.getUserMedia = jest.fn(() => 
        Promise.reject(new Error('Permission denied'))
      );

      await expect(voiceProcessor.startListening()).rejects.toThrow('Permission denied');
    });

    it('should enumerate available audio devices', async () => {
      const devices = await voiceProcessor.getAvailableDevices();
      expect(devices).toBeDefined();
      expect(Array.isArray(devices)).toBe(true);
    });
  });

  describe('Audio Processing', () => {
    beforeEach(async () => {
      await voiceProcessor.initialize();
      await voiceProcessor.startListening();
    });

    it('should process audio data', () => {
      const mockAudioData = new Float32Array(1024);
      mockAudioData.fill(0.5); // Fill with test data
      
      const result = voiceProcessor.processAudioData(mockAudioData);
      expect(result).toBeDefined();
    });

    it('should detect voice activity', () => {
      const mockAudioData = new Float32Array(1024);
      mockAudioData.fill(0.8); // High amplitude = voice activity
      
      const hasVoice = voiceProcessor.detectVoiceActivity(mockAudioData);
      expect(typeof hasVoice).toBe('boolean');
    });

    it('should extract audio features', () => {
      const mockAudioData = new Float32Array(1024);
      mockAudioData.fill(0.5);
      
      const features = voiceProcessor.extractFeatures(mockAudioData);
      expect(features).toBeDefined();
      expect(features.energy).toBeDefined();
      expect(features.pitch).toBeDefined();
      expect(features.spectralCentroid).toBeDefined();
    });

    it('should apply noise reduction', () => {
      const mockAudioData = new Float32Array(1024);
      mockAudioData.fill(0.1); // Low amplitude = noise
      
      const cleanedData = voiceProcessor.applyNoiseReduction(mockAudioData);
      expect(cleanedData).toBeInstanceOf(Float32Array);
      expect(cleanedData.length).toBe(mockAudioData.length);
    });
  });

  describe('Speech Recognition', () => {
    beforeEach(async () => {
      await voiceProcessor.initialize();
    });

    it('should start speech recognition', async () => {
      const mockRecognition = {
        start: jest.fn(),
        stop: jest.fn(),
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
        continuous: false,
        interimResults: false,
        lang: 'en-US',
      };

      // Mock SpeechRecognition
      (global as any).SpeechRecognition = jest.fn(() => mockRecognition);
      (global as any).webkitSpeechRecognition = jest.fn(() => mockRecognition);

      await voiceProcessor.startSpeechRecognition();
      expect(mockRecognition.start).toHaveBeenCalled();
    });

    it('should handle speech recognition results', () => {
      const mockResult = {
        transcript: 'Hello world',
        confidence: 0.95,
        isFinal: true,
      };

      const callback = jest.fn();
      voiceProcessor.onSpeechResult(callback);
      
      // Simulate speech recognition result
      voiceProcessor.handleSpeechResult(mockResult);
      expect(callback).toHaveBeenCalledWith(mockResult);
    });

    it('should handle speech recognition errors', () => {
      const mockError = new Error('Speech recognition failed');
      const errorCallback = jest.fn();
      
      voiceProcessor.onSpeechError(errorCallback);
      voiceProcessor.handleSpeechError(mockError);
      
      expect(errorCallback).toHaveBeenCalledWith(mockError);
    });
  });

  describe('Audio Analysis', () => {
    beforeEach(async () => {
      await voiceProcessor.initialize();
    });

    it('should analyze frequency spectrum', () => {
      const mockFrequencyData = new Uint8Array(512);
      mockFrequencyData.fill(128); // Mid-range frequency data
      
      const analysis = voiceProcessor.analyzeFrequencySpectrum(mockFrequencyData);
      expect(analysis).toBeDefined();
      expect(analysis.dominantFrequency).toBeDefined();
      expect(analysis.spectralRolloff).toBeDefined();
      expect(analysis.spectralFlux).toBeDefined();
    });

    it('should detect silence', () => {
      const silentData = new Float32Array(1024);
      silentData.fill(0.001); // Very low amplitude
      
      const isSilent = voiceProcessor.detectSilence(silentData);
      expect(isSilent).toBe(true);
    });

    it('should detect loud audio', () => {
      const loudData = new Float32Array(1024);
      loudData.fill(0.9); // High amplitude
      
      const isLoud = voiceProcessor.detectLoudAudio(loudData);
      expect(isLoud).toBe(true);
    });

    it('should calculate audio statistics', () => {
      const mockAudioData = new Float32Array(1024);
      for (let i = 0; i < mockAudioData.length; i++) {
        mockAudioData[i] = Math.sin(i * 0.1); // Sine wave
      }
      
      const stats = voiceProcessor.calculateAudioStatistics(mockAudioData);
      expect(stats).toBeDefined();
      expect(stats.rms).toBeDefined();
      expect(stats.peak).toBeDefined();
      expect(stats.zeroCrossingRate).toBeDefined();
    });
  });

  describe('Configuration Management', () => {
    it('should update configuration', () => {
      const newConfig = {
        sampleRate: 48000,
        bufferSize: 8192,
        noiseReduction: true,
        voiceActivityDetection: true,
      };
      
      voiceProcessor.updateConfig(newConfig);
      const config = voiceProcessor.getConfig();
      
      expect(config.sampleRate).toBe(48000);
      expect(config.bufferSize).toBe(8192);
      expect(config.noiseReduction).toBe(true);
    });

    it('should validate configuration values', () => {
      const invalidConfig = {
        sampleRate: -1, // Invalid sample rate
        bufferSize: 0,  // Invalid buffer size
      };
      
      expect(() => {
        voiceProcessor.updateConfig(invalidConfig);
      }).toThrow();
    });
  });

  describe('Event Handling', () => {
    it('should emit audio data events', async () => {
      await voiceProcessor.initialize();
      
      const callback = jest.fn();
      voiceProcessor.onAudioData(callback);
      
      const mockAudioData = new Float32Array(1024);
      voiceProcessor.processAudioData(mockAudioData);
      
      expect(callback).toHaveBeenCalled();
    });

    it('should emit voice activity events', async () => {
      await voiceProcessor.initialize();
      
      const callback = jest.fn();
      voiceProcessor.onVoiceActivity(callback);
      
      const mockAudioData = new Float32Array(1024);
      mockAudioData.fill(0.8); // High amplitude
      
      voiceProcessor.detectVoiceActivity(mockAudioData);
      expect(callback).toHaveBeenCalled();
    });

    it('should emit error events', () => {
      const errorCallback = jest.fn();
      voiceProcessor.onError(errorCallback);
      
      const testError = new Error('Test error');
      voiceProcessor.handleError(testError);
      
      expect(errorCallback).toHaveBeenCalledWith(testError);
    });
  });

  describe('Resource Management', () => {
    it('should dispose of resources properly', async () => {
      await voiceProcessor.initialize();
      await voiceProcessor.startListening();
      
      voiceProcessor.dispose();
      
      expect(voiceProcessor.isListening()).toBe(false);
      expect(voiceProcessor.isInitialized()).toBe(false);
    });

    it('should stop all audio tracks on disposal', async () => {
      await voiceProcessor.initialize();
      await voiceProcessor.startListening();
      
      const stopSpy = jest.spyOn(mockMediaStream.getAudioTracks()[0], 'stop');
      
      voiceProcessor.dispose();
      expect(stopSpy).toHaveBeenCalled();
    });

    it('should close audio context on disposal', async () => {
      await voiceProcessor.initialize();
      
      voiceProcessor.dispose();
      expect(mockAudioContext.close).toHaveBeenCalled();
    });
  });
});
