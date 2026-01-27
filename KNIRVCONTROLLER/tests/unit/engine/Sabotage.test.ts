import { SabotageEngine, SabotageType } from '../../../src/engine/Sabotage';
import { RFTAgent } from '../../../src/types/Agent';

describe('SabotageEngine', () => {
  let mockAgent: RFTAgent;
  
  beforeEach(() => {
    mockAgent = {
      id: 'test-agent',
      name: 'Test Agent',
      policy: 'greedy',
      resources: { compute: 100, parity: 100, generation: 1 },
      proposeSolution: jest.fn()
    };
  });
  
  describe('applyEffect', () => {
    it('should apply noise injection effect', () => {
      SabotageEngine.applyEffect(SabotageType.NOISE_INJECTION, mockAgent, 1);
    });
    
    it('should apply backprop pulse effect', () => {
      const initialParity = mockAgent.resources.parity;
      
      SabotageEngine.applyEffect(SabotageType.BACKPROP_PULSE, mockAgent, 1);
      
      expect(mockAgent.resources.parity).toBeLessThan(initialParity);
    });
    
    it('should apply gradient ghosting effect', () => {
      SabotageEngine.applyEffect(SabotageType.GRADIENT_GHOSTING, mockAgent, 1);
    });
    
    it('should apply effect with magnitude', () => {
      const initialParity = mockAgent.resources.parity;
      
      SabotageEngine.applyEffect(SabotageType.BACKPROP_PULSE, mockAgent, 2);
      
      expect(mockAgent.resources.parity).toBeLessThan(initialParity - 10);
    });
  });
});
