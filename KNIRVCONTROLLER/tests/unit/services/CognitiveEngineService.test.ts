/**
 * Tests for CognitiveEngineService
 */

import { cognitiveEngineService, CognitiveProcessingRequest, SkillExecutionRequest } from '../../../src/services/CognitiveEngineService';

// Mock fetch globally
global.fetch = jest.fn();

describe('CognitiveEngineService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    // Reset service state
    cognitiveEngineService['isRunning'] = false;
    cognitiveEngineService['context'].clear();
    cognitiveEngineService['activeSkills'].clear();
    cognitiveEngineService['learningEvents'] = [];
    cognitiveEngineService['metrics'] = cognitiveEngineService['initializeMetrics']();
  });

  describe('initialization', () => {
    it('should initialize with default metrics', () => {
      const metrics = cognitiveEngineService.getMetrics();
      
      expect(metrics.totalProcessingRequests).toBe(0);
      expect(metrics.averageProcessingTime).toBe(0);
      expect(metrics.skillInvocations).toBe(0);
      expect(metrics.learningEvents).toBe(0);
      expect(metrics.adaptationLevel).toBe(0.75);
      expect(metrics.confidenceLevel).toBe(0.95);
      expect(metrics.activeSkills).toBe(0);
      expect(metrics.contextSize).toBe(0);
    });

    it('should not be running initially', () => {
      expect(cognitiveEngineService.isEngineRunning()).toBe(false);
    });
  });

  describe('start/stop engine', () => {
    it('should start engine successfully', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ status: 'started' })
      });

      await cognitiveEngineService.start();

      expect(cognitiveEngineService.isEngineRunning()).toBe(true);
      expect(fetch).toHaveBeenCalledWith(
        'http://localhost:3001/api/cognitive/start',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' }
        })
      );
    });

    it('should stop engine successfully', async () => {
      // Start engine first
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ status: 'started' })
      });
      await cognitiveEngineService.start();

      // Stop engine
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ status: 'stopped' })
      });

      await cognitiveEngineService.stop();

      expect(cognitiveEngineService.isEngineRunning()).toBe(false);
    });

    it('should handle start failure', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: false,
        statusText: 'Internal Server Error'
      });

      await expect(cognitiveEngineService.start())
        .rejects.toThrow('Failed to start cognitive engine: Internal Server Error');
    });

    it('should not start if already running', async () => {
      // Start engine first
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ status: 'started' })
      });
      await cognitiveEngineService.start();

      // Try to start again - should return immediately
      await cognitiveEngineService.start();

      expect(fetch).toHaveBeenCalledTimes(1); // Only called once
    });
  });

  describe('processInput', () => {
    beforeEach(async () => {
      // Start engine
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ status: 'started' })
      });
      await cognitiveEngineService.start();
    });

    it('should process input successfully', async () => {
      const mockResponse = {
        output: 'Processed successfully',
        confidence: 0.95,
        skillsInvoked: ['analysis_skill'],
        contextUpdates: { lastInput: 'test input' },
        adaptationTriggered: false
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => {
          // Advance time to simulate processing time
          jest.advanceTimersByTime(50);
          return mockResponse;
        }
      });

      const request: CognitiveProcessingRequest = {
        input: 'test input',
        context: { user: 'test' },
        taskType: 'conversation',
        requiresSkillInvocation: true
      };

      const result = await cognitiveEngineService.processInput(request);

      expect(result.output).toBe('Processed successfully');
      expect(result.confidence).toBe(0.95);
      expect(result.skillsInvoked).toEqual(['analysis_skill']);
      expect(result.processingTime).toBeGreaterThan(0);

      // Check metrics were updated
      const metrics = cognitiveEngineService.getMetrics();
      expect(metrics.totalProcessingRequests).toBe(1);
      expect(metrics.skillInvocations).toBe(1);
    });

    it('should fail when engine is not running', async () => {
      await cognitiveEngineService.stop();

      const request: CognitiveProcessingRequest = {
        input: 'test input',
        taskType: 'conversation'
      };

      await expect(cognitiveEngineService.processInput(request))
        .rejects.toThrow('Cognitive engine is not running');
    });

    it('should handle processing failure gracefully', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: false,
        statusText: 'Processing Error'
      });

      const request: CognitiveProcessingRequest = {
        input: 'test input',
        taskType: 'conversation'
      };

      const result = await cognitiveEngineService.processInput(request);

      expect(result.output).toContain('Error processing input');
      expect(result.confidence).toBe(0.1);
      expect(result.skillsInvoked).toEqual([]);
    });

    it('should update context when provided', async () => {
      const mockResponse = {
        output: 'Processed',
        confidence: 0.9,
        skillsInvoked: [],
        contextUpdates: { newKey: 'newValue', existingKey: 'updatedValue' }
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse
      });

      const request: CognitiveProcessingRequest = {
        input: 'test input',
        taskType: 'conversation'
      };

      await cognitiveEngineService.processInput(request);

      const context = cognitiveEngineService.getContext();
      expect(context.get('newKey')).toBe('newValue');
      expect(context.get('existingKey')).toBe('updatedValue');

      const metrics = cognitiveEngineService.getMetrics();
      expect(metrics.contextSize).toBe(2);
    });
  });

  describe('executeSkill', () => {
    beforeEach(async () => {
      // Start engine
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ status: 'started' })
      });
      await cognitiveEngineService.start();
    });

    it('should execute skill successfully', async () => {
      const mockResponse = {
        output: { result: 'skill executed' },
        resourceUsage: { memory: 64, cpu: 0.5 }
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => {
          // Advance time to simulate execution time
          jest.advanceTimersByTime(100);
          return mockResponse;
        }
      });

      const request: SkillExecutionRequest = {
        skillId: 'test-skill',
        parameters: { param1: 'value1' },
        context: { user: 'test' },
        timeout: 30000
      };

      const result = await cognitiveEngineService.executeSkill(request);

      expect(result.success).toBe(true);
      expect(result.output).toEqual({ result: 'skill executed' });
      expect(result.resourceUsage).toEqual({ memory: 64, cpu: 0.5 });
      expect(result.executionTime).toBeGreaterThan(0);

      // Check metrics were updated
      const metrics = cognitiveEngineService.getMetrics();
      expect(metrics.skillInvocations).toBe(1);
    });

    it('should fail when engine is not running', async () => {
      await cognitiveEngineService.stop();

      const request: SkillExecutionRequest = {
        skillId: 'test-skill',
        parameters: {}
      };

      await expect(cognitiveEngineService.executeSkill(request))
        .rejects.toThrow('Cognitive engine is not running');
    });

    it('should handle skill execution failure', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: false,
        statusText: 'Skill Not Found'
      });

      const request: SkillExecutionRequest = {
        skillId: 'non-existent-skill',
        parameters: {}
      };

      const result = await cognitiveEngineService.executeSkill(request);

      expect(result.success).toBe(false);
      expect(result.error).toContain('Skill execution failed');
    });
  });

  describe('skill management', () => {
    it('should activate skill successfully', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ status: 'activated' })
      });

      await cognitiveEngineService.activateSkill('test-skill');

      expect(cognitiveEngineService.getActiveSkills()).toContain('test-skill');
      expect(cognitiveEngineService.getMetrics().activeSkills).toBe(1);
    });

    it('should deactivate skill successfully', async () => {
      // Activate first
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ status: 'activated' })
      });
      await cognitiveEngineService.activateSkill('test-skill');

      // Deactivate
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ status: 'deactivated' })
      });
      await cognitiveEngineService.deactivateSkill('test-skill');

      expect(cognitiveEngineService.getActiveSkills()).not.toContain('test-skill');
      expect(cognitiveEngineService.getMetrics().activeSkills).toBe(0);
    });

    it('should handle activation failure gracefully', async () => {
      (fetch as jest.Mock).mockRejectedValueOnce(new Error('Network error'));

      // Should not throw
      await cognitiveEngineService.activateSkill('test-skill');

      expect(cognitiveEngineService.getActiveSkills()).not.toContain('test-skill');
    });
  });

  describe('learning mode', () => {
    it('should start learning mode successfully', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ status: 'learning_started' })
      });

      await cognitiveEngineService.startLearningMode();

      expect(fetch).toHaveBeenCalledWith(
        'http://localhost:3001/api/cognitive/learning/start',
        expect.objectContaining({ method: 'POST' })
      );
    });

    it('should save adaptation successfully', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ status: 'adaptation_saved' })
      });

      await cognitiveEngineService.saveCurrentAdaptation();

      expect(fetch).toHaveBeenCalledWith(
        'http://localhost:3001/api/cognitive/adaptation/save',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: expect.stringContaining('context')
        })
      );
    });
  });

  describe('getters', () => {
    it('should return current metrics', () => {
      const metrics = cognitiveEngineService.getMetrics();
      
      expect(metrics).toHaveProperty('totalProcessingRequests');
      expect(metrics).toHaveProperty('averageProcessingTime');
      expect(metrics).toHaveProperty('skillInvocations');
      expect(metrics).toHaveProperty('learningEvents');
      expect(metrics).toHaveProperty('adaptationLevel');
      expect(metrics).toHaveProperty('confidenceLevel');
      expect(metrics).toHaveProperty('activeSkills');
      expect(metrics).toHaveProperty('contextSize');
    });

    it('should return current context', () => {
      // Add some context
      cognitiveEngineService['context'].set('testKey', 'testValue');
      
      const context = cognitiveEngineService.getContext();
      expect(context.get('testKey')).toBe('testValue');
      expect(context).toBeInstanceOf(Map);
    });

    it('should return active skills', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ status: 'activated' })
      });
      
      await cognitiveEngineService.activateSkill('skill1');
      
      const activeSkills = cognitiveEngineService.getActiveSkills();
      expect(activeSkills).toEqual(['skill1']);
    });
  });

  describe('metrics calculation', () => {
    it('should update processing metrics correctly', async () => {
      // Start engine
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ status: 'started' })
      });
      await cognitiveEngineService.start();

      // Mock multiple processing requests
      (fetch as jest.Mock)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => {
            // Advance time to simulate processing time
            jest.advanceTimersByTime(50);
            return { output: 'result1', confidence: 0.9, skillsInvoked: [] };
          }
        })
        .mockResolvedValueOnce({
          ok: true,
          json: async () => {
            // Advance time to simulate processing time
            jest.advanceTimersByTime(75);
            return { output: 'result2', confidence: 0.8, skillsInvoked: ['skill1'] };
          }
        });

      const request: CognitiveProcessingRequest = {
        input: 'test',
        taskType: 'conversation'
      };

      await cognitiveEngineService.processInput(request);
      await cognitiveEngineService.processInput(request);

      const metrics = cognitiveEngineService.getMetrics();
      expect(metrics.totalProcessingRequests).toBe(2);
      expect(metrics.skillInvocations).toBe(1);
      expect(metrics.averageProcessingTime).toBeGreaterThan(0);
    });
  });
});
