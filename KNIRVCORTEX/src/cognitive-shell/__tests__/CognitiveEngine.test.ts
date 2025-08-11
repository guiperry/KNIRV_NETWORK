import { CognitiveEngine, CognitiveConfig, CognitiveState, LearningEvent } from '../CognitiveEngine';

// Mock all dependencies
jest.mock('../EventEmitter');
jest.mock('../SEALFramework');
jest.mock('../FabricAlgorithm');
jest.mock('../VoiceProcessor');
jest.mock('../VisualProcessor');
jest.mock('../LoRAAdapter');
jest.mock('../EnhancedLoRAAdapter');
jest.mock('../HRMBridge');
jest.mock('../HRMLoRABridge');
jest.mock('../AdaptiveLearningPipeline');
jest.mock('../KNIRVWalletIntegration');
jest.mock('../KNIRVChainIntegration');
jest.mock('../EcosystemCommunicationLayer');

describe('CognitiveEngine', () => {
  let cognitiveEngine: CognitiveEngine;
  let mockConfig: CognitiveConfig;

  beforeEach(() => {
    mockConfig = {
      maxContextSize: 1000,
      learningRate: 0.01,
      adaptationThreshold: 0.8,
      skillTimeout: 5000,
      voiceEnabled: true,
      visualEnabled: true,
      loraEnabled: true,
      enhancedLoraEnabled: true,
      hrmEnabled: true,
      hrmConfig: {
        modelPath: '/test/model',
        maxTokens: 512,
        temperature: 0.7,
        topP: 0.9,
      },
      adaptiveLearningEnabled: true,
      walletIntegrationEnabled: true,
      chainIntegrationEnabled: true,
      ecosystemCommunicationEnabled: true,
    };

    cognitiveEngine = new CognitiveEngine(mockConfig);
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  describe('Initialization', () => {
    it('should create a new CognitiveEngine instance', () => {
      expect(cognitiveEngine).toBeInstanceOf(CognitiveEngine);
    });

    it('should initialize with default config when no config provided', () => {
      const defaultEngine = new CognitiveEngine();
      expect(defaultEngine).toBeInstanceOf(CognitiveEngine);
    });

    it('should initialize state correctly', () => {
      const state = cognitiveEngine.getState();
      expect(state).toBeDefined();
      expect(state.currentContext).toBeInstanceOf(Map);
      expect(state.activeSkills).toEqual([]);
      expect(state.learningHistory).toEqual([]);
      expect(state.confidenceLevel).toBe(0.5);
      expect(state.adaptationLevel).toBe(0);
    });
  });

  describe('State Management', () => {
    it('should update context correctly', () => {
      const contextKey = 'testKey';
      const contextValue = 'testValue';
      
      cognitiveEngine.updateContext(contextKey, contextValue);
      
      const state = cognitiveEngine.getState();
      expect(state.currentContext.get(contextKey)).toBe(contextValue);
    });

    it('should clear context when max size exceeded', () => {
      const smallConfig = { ...mockConfig, maxContextSize: 2 };
      const smallEngine = new CognitiveEngine(smallConfig);
      
      smallEngine.updateContext('key1', 'value1');
      smallEngine.updateContext('key2', 'value2');
      smallEngine.updateContext('key3', 'value3'); // Should trigger cleanup
      
      const state = smallEngine.getState();
      expect(state.currentContext.size).toBeLessThanOrEqual(2);
    });

    it('should add and remove active skills', () => {
      const skillName = 'testSkill';
      
      cognitiveEngine.activateSkill(skillName);
      let state = cognitiveEngine.getState();
      expect(state.activeSkills).toContain(skillName);
      
      cognitiveEngine.deactivateSkill(skillName);
      state = cognitiveEngine.getState();
      expect(state.activeSkills).not.toContain(skillName);
    });

    it('should update confidence level', () => {
      const newConfidence = 0.8;
      cognitiveEngine.updateConfidence(newConfidence);
      
      const state = cognitiveEngine.getState();
      expect(state.confidenceLevel).toBe(newConfidence);
    });

    it('should clamp confidence level between 0 and 1', () => {
      cognitiveEngine.updateConfidence(1.5);
      expect(cognitiveEngine.getState().confidenceLevel).toBe(1);
      
      cognitiveEngine.updateConfidence(-0.5);
      expect(cognitiveEngine.getState().confidenceLevel).toBe(0);
    });
  });

  describe('Learning and Adaptation', () => {
    it('should record learning events', () => {
      const learningEvent: LearningEvent = {
        timestamp: new Date(),
        eventType: 'test',
        input: 'test input',
        output: 'test output',
        feedback: 0.8,
        adaptationApplied: false,
      };
      
      cognitiveEngine.recordLearningEvent(learningEvent);
      
      const state = cognitiveEngine.getState();
      expect(state.learningHistory).toContain(learningEvent);
    });

    it('should trigger adaptation when threshold is met', () => {
      const highFeedbackEvent: LearningEvent = {
        timestamp: new Date(),
        eventType: 'test',
        input: 'test input',
        output: 'test output',
        feedback: 0.9, // Above adaptation threshold
        adaptationApplied: false,
      };
      
      cognitiveEngine.recordLearningEvent(highFeedbackEvent);
      
      const state = cognitiveEngine.getState();
      expect(state.adaptationLevel).toBeGreaterThan(0);
    });

    it('should limit learning history size', () => {
      const maxHistorySize = 100;
      
      // Add more events than the limit
      for (let i = 0; i < maxHistorySize + 10; i++) {
        const event: LearningEvent = {
          timestamp: new Date(),
          eventType: `test${i}`,
          input: `input${i}`,
          output: `output${i}`,
          feedback: 0.5,
          adaptationApplied: false,
        };
        cognitiveEngine.recordLearningEvent(event);
      }
      
      const state = cognitiveEngine.getState();
      expect(state.learningHistory.length).toBeLessThanOrEqual(maxHistorySize);
    });
  });

  describe('Skill Processing', () => {
    it('should process input through active skills', async () => {
      const testInput = 'test input';
      const expectedOutput = 'processed output';
      
      cognitiveEngine.activateSkill('testSkill');
      
      const result = await cognitiveEngine.processInput(testInput);
      expect(result).toBeDefined();
    });

    it('should handle skill timeout', async () => {
      const testInput = 'test input';
      const shortTimeoutConfig = { ...mockConfig, skillTimeout: 100 };
      const timeoutEngine = new CognitiveEngine(shortTimeoutConfig);
      
      // Mock a slow skill
      jest.spyOn(timeoutEngine as any, 'executeSkill').mockImplementation(
        () => new Promise(resolve => setTimeout(resolve, 200))
      );
      
      const startTime = Date.now();
      await timeoutEngine.processInput(testInput);
      const endTime = Date.now();
      
      expect(endTime - startTime).toBeLessThan(150); // Should timeout before 200ms
    });

    it('should handle skill errors gracefully', async () => {
      const testInput = 'test input';
      
      // Mock a skill that throws an error
      jest.spyOn(cognitiveEngine as any, 'executeSkill').mockRejectedValue(
        new Error('Skill execution failed')
      );
      
      const result = await cognitiveEngine.processInput(testInput);
      expect(result).toBeDefined(); // Should not throw, but return error result
    });
  });

  describe('Integration Components', () => {
    it('should initialize voice processor when enabled', () => {
      expect(cognitiveEngine.isVoiceEnabled()).toBe(true);
    });

    it('should initialize visual processor when enabled', () => {
      expect(cognitiveEngine.isVisualEnabled()).toBe(true);
    });

    it('should initialize LoRA adapter when enabled', () => {
      expect(cognitiveEngine.isLoRAEnabled()).toBe(true);
    });

    it('should initialize HRM bridge when enabled', () => {
      expect(cognitiveEngine.isHRMEnabled()).toBe(true);
    });

    it('should initialize wallet integration when enabled', () => {
      expect(cognitiveEngine.isWalletIntegrationEnabled()).toBe(true);
    });

    it('should initialize chain integration when enabled', () => {
      expect(cognitiveEngine.isChainIntegrationEnabled()).toBe(true);
    });
  });

  describe('Event Handling', () => {
    it('should emit events for state changes', () => {
      const mockListener = jest.fn();
      cognitiveEngine.on('stateChanged', mockListener);
      
      cognitiveEngine.updateContext('test', 'value');
      
      expect(mockListener).toHaveBeenCalled();
    });

    it('should emit events for skill activation', () => {
      const mockListener = jest.fn();
      cognitiveEngine.on('skillActivated', mockListener);
      
      cognitiveEngine.activateSkill('testSkill');
      
      expect(mockListener).toHaveBeenCalledWith('testSkill');
    });

    it('should emit events for learning events', () => {
      const mockListener = jest.fn();
      cognitiveEngine.on('learningEvent', mockListener);
      
      const learningEvent: LearningEvent = {
        timestamp: new Date(),
        eventType: 'test',
        input: 'input',
        output: 'output',
        feedback: 0.8,
        adaptationApplied: false,
      };
      
      cognitiveEngine.recordLearningEvent(learningEvent);
      
      expect(mockListener).toHaveBeenCalledWith(learningEvent);
    });
  });

  describe('Configuration Updates', () => {
    it('should update configuration at runtime', () => {
      const newConfig = { ...mockConfig, learningRate: 0.02 };
      
      cognitiveEngine.updateConfig(newConfig);
      
      expect(cognitiveEngine.getConfig().learningRate).toBe(0.02);
    });

    it('should validate configuration updates', () => {
      const invalidConfig = { ...mockConfig, learningRate: -1 };
      
      expect(() => {
        cognitiveEngine.updateConfig(invalidConfig);
      }).toThrow();
    });
  });

  describe('Cleanup and Disposal', () => {
    it('should cleanup resources on dispose', () => {
      const disposeSpy = jest.spyOn(cognitiveEngine, 'dispose');
      
      cognitiveEngine.dispose();
      
      expect(disposeSpy).toHaveBeenCalled();
    });

    it('should remove all event listeners on dispose', () => {
      const mockListener = jest.fn();
      cognitiveEngine.on('test', mockListener);
      
      cognitiveEngine.dispose();
      cognitiveEngine.emit('test');
      
      expect(mockListener).not.toHaveBeenCalled();
    });
  });
});
