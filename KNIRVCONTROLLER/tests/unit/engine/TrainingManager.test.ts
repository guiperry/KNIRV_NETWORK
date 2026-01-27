import { TrainingManager } from '../../../src/engine/TrainingManager';
import { RFTAgent } from '../../../src/types/Agent';
import { TrajectoryStep } from '../../../src/types/Trajectory';

describe('TrainingManager', () => {
  let trainingManager: TrainingManager;
  let mockAgent: RFTAgent;
  
  beforeEach(() => {
    trainingManager = new TrainingManager();
    mockAgent = {
      id: 'test-agent',
      name: 'Test Agent',
      policy: 'greedy',
      resources: { compute: 100, parity: 100, generation: 1 },
      proposeSolution: jest.fn()
    };
  });
  
  describe('distill', () => {
    it('should distill trajectory', () => {
      const trajectory: TrajectoryStep[] = [
        { step: 1, thought: 'Long thought process', action: 'code' },
        { step: 2, thought: 'Short', action: 'code' },
        { step: 3, thought: 'Another long thought', action: 'code' }
      ];
      
      const distilled = trainingManager.distill(trajectory);
      
      expect(distilled.length).toBeLessThan(trajectory.length);
    });
  });
  
  describe('harden', () => {
    it('should harden agent', () => {
      const initialParity = mockAgent.resources.parity;
      const initialCompute = mockAgent.resources.compute;
      
      trainingManager.harden(mockAgent);
      
      expect(mockAgent.resources.parity).toBeGreaterThan(initialParity);
      expect(mockAgent.resources.compute).toBeGreaterThan(initialCompute);
    });
  });
  
  describe('prune', () => {
    it('should prune trajectory', () => {
      const trajectory: TrajectoryStep[] = [
        { step: 1, thought: 'Step 1', action: 'code' },
        { step: 2, thought: 'Step 1', action: 'code' },
        { step: 3, thought: 'Step 3', action: 'code' }
      ];
      
      const pruned = trainingManager.prune(trajectory);
      
      expect(pruned.length).toBeLessThan(trajectory.length);
    });
  });
  
  describe('denoise', () => {
    it('should denoise trajectory', () => {
      const trajectory: TrajectoryStep[] = [
        { step: 1, thought: 'Th0ught w1th n0ise', action: 'c0de w1th n0ise' },
        { step: 2, thought: 'Clean thought', action: 'clean code' }
      ];
      
      const denoised = trainingManager.denoise(trajectory);
      
      denoised.forEach(step => {
        expect(step.thought).not.toContain('0');
        expect(step.action).not.toContain('0');
      });
    });
  });
  
  describe('quantize', () => {
    it('should quantize trajectory', () => {
      const trajectory: TrajectoryStep[] = [
        { step: 1, thought: 'Long thought process with many details', action: 'long code with many lines' },
        { step: 2, thought: 'Short', action: 'short' }
      ];
      
      const quantized = trainingManager.quantize(trajectory, 5);
      
      quantized.forEach(step => {
        expect(step.thought.length).toBeLessThan(100);
        expect(step.action.length).toBeLessThan(100);
      });
    });
  });
  
  describe('stressTest', () => {
    it('should run stress test', async () => {
      const variations = ['var1', 'var2', 'var3'];
      
      mockAgent.proposeSolution = jest.fn().mockResolvedValue({
        chainOfThought: ['Test'],
        code: 'function test() { return 42; }'
      });
      
      const successRate = await trainingManager.stressTest(mockAgent, variations);
      
      expect(typeof successRate).toBe('number');
      expect(successRate).toBeGreaterThanOrEqual(0);
      expect(successRate).toBeLessThanOrEqual(1);
    });
  });
});
