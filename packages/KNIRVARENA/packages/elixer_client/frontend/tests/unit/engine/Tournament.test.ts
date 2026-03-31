import { Tournament } from '../../../src/engine/Tournament';
import { Verifier } from '../../../src/engine/Verifier';
import { LoraxClient } from '../../../src/networking/LoraxClient';
import { RFTAgent } from '../../../src/types/Agent';

describe('Tournament', () => {
  let verifier: Verifier;
  let loraxClient: LoraxClient;
  let tournament: Tournament;
  
  beforeEach(() => {
    verifier = new Verifier();
    loraxClient = new LoraxClient('http://localhost:8080');
    tournament = new Tournament(verifier, loraxClient);
  });
  
  describe('initialization', () => {
    it('should initialize with correct default values', () => {
      expect(tournament.getSkillSlotOwner()).toBeNull();
      expect(tournament.getIncumbentScore()).toEqual(0.8);
    });
  });
  
  describe('runEpoch', () => {
    it('should run an epoch with agents', async () => {
      const mockAgents: RFTAgent[] = [
        {
          id: 'agent-1',
          name: 'Test Agent',
          policy: 'greedy',
          resources: { compute: 100, parity: 100, generation: 1 },
          proposeSolution: jest.fn().mockResolvedValue({
            chainOfThought: ['Test thought'],
            code: 'function test() { return 42; }'
          })
        }
      ];
      
      // Mock verifier
      const mockEvaluate = jest.spyOn(verifier, 'evaluate').mockResolvedValue(0.9);
      
      // Mock LoraxClient
      const mockFineTune = jest.spyOn(loraxClient, 'fineTune').mockResolvedValue({ success: true });
      
      await tournament.runEpoch(mockAgents, 'Test context');
      
      expect(mockEvaluate).toHaveBeenCalled();
      expect(mockFineTune).toHaveBeenCalled();
      
      expect(tournament.getSkillSlotOwner()).toEqual('agent-1');
      expect(tournament.getIncumbentScore()).toEqual(0.9);
    });
    
    it('should not update skill slot if no agent beats incumbent score', async () => {
      const mockAgents: RFTAgent[] = [
        {
          id: 'agent-1',
          name: 'Test Agent',
          policy: 'greedy',
          resources: { compute: 100, parity: 100, generation: 1 },
          proposeSolution: jest.fn().mockResolvedValue({
            chainOfThought: ['Test thought'],
            code: 'function test() { return 42; }'
          })
        }
      ];
      
      // Mock verifier to return score less than incumbent (0.8)
      const mockEvaluate = jest.spyOn(verifier, 'evaluate').mockResolvedValue(0.7);
      
      // Mock LoraxClient
      const mockFineTune = jest.spyOn(loraxClient, 'fineTune').mockResolvedValue({ success: true });
      
      await tournament.runEpoch(mockAgents, 'Test context');
      
      expect(mockEvaluate).toHaveBeenCalled();
      expect(mockFineTune).not.toHaveBeenCalled();
      
      expect(tournament.getSkillSlotOwner()).toBeNull();
      expect(tournament.getIncumbentScore()).toEqual(0.8);
    });
  });
});
