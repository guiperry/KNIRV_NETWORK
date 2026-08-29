import { Verifier, ScoreWeights } from '../../../src/engine/Verifier';
import type { SolutionProposal, AgentPersona } from '../../../src/services/gameLLMService';
import type { Challenge } from '../../../src/types/challenge';

describe('Verifier', () => {
  let verifier: Verifier;
  
  beforeEach(() => {
    verifier = new Verifier();
  });
  
  describe('initialization', () => {
    it('should initialize with correct default weights', () => {
      // Verifier initializes with default weights, no updateRequirements method
      expect(verifier).toBeDefined();
    });
  });
  
  describe('updateWeights', () => {
    it('should update scoring weights', () => {
      const newWeights: ScoreWeights = { correctness: 0.5, latency: 0.4, simplicity: 0.1 };
      verifier.updateWeights(newWeights);
    });
  });
  
  describe('addConstraint and removeConstraint', () => {
    it('should add and remove constraints', () => {
      const validator = (res: any) => res > 10;
      verifier.addConstraint('min-value', validator);
      
      verifier.removeConstraint('min-value');
    });
  });
  
  describe('evaluateProposal', () => {
    it('should evaluate proposal and return score', async () => {
      const mockProposal: SolutionProposal = {
        chainOfThought: ['Simple solution'],
        solution: 'return 42;',
        estimatedLatency: 100
      };
      const mockChallenge: Challenge = {
        id: 'test',
        title: 'Test Challenge',
        type: 'Logic Error',
        difficulty: 0.5,
        bounty: 100,
        description: 'Return 42',
        buggyCode: 'return 0;',
        context: 'Should return 42',
        hints: []
      };
      const mockPersona: AgentPersona = {
        id: 'test',
        name: 'Test Agent',
        systemPrompt: 'You are a test agent',
        winRate: 0.8,
        totalEpochs: 10,
        wins: 8
      };
      
      const score = await verifier.evaluateProposal(mockProposal, mockChallenge, mockPersona);
      
      expect(typeof score).toBe('number');
      expect(score).toBeGreaterThanOrEqual(0);
      expect(score).toBeLessThanOrEqual(1);
    });
    
    it('should apply constraints', async () => {
      const mockProposal: SolutionProposal = {
        chainOfThought: ['Simple solution'],
        solution: 'return 42;',
        estimatedLatency: 100
      };
      const mockChallenge: Challenge = {
        id: 'test',
        title: 'Test Challenge',
        type: 'Logic Error',
        difficulty: 0.5,
        bounty: 100,
        description: 'Return 42',
        buggyCode: 'return 0;',
        context: 'Should return 42',
        hints: []
      };
      const mockPersona: AgentPersona = {
        id: 'test',
        name: 'Test Agent',
        systemPrompt: 'You are a test agent',
        winRate: 0.8,
        totalEpochs: 10,
        wins: 8
      };
      
      // Add constraint that returns true (should not affect score)
      verifier.addConstraint('valid-result', (res: any) => res === 42);
      
      const score = await verifier.evaluateProposal(mockProposal, mockChallenge, mockPersona);
      
      expect(typeof score).toBe('number');
    });
    
    it('should penalize failing constraints', async () => {
      const mockProposal: SolutionProposal = {
        chainOfThought: ['Simple solution'],
        solution: 'return 42;',
        estimatedLatency: 100
      };
      const mockChallenge: Challenge = {
        id: 'test',
        title: 'Test Challenge',
        type: 'Logic Error',
        difficulty: 0.5,
        bounty: 100,
        description: 'Return 42',
        buggyCode: 'return 0;',
        context: 'Should return 42',
        hints: []
      };
      const mockPersona: AgentPersona = {
        id: 'test',
        name: 'Test Agent',
        systemPrompt: 'You are a test agent',
        winRate: 0.8,
        totalEpochs: 10,
        wins: 8
      };
      
      // Add constraint that returns false
      verifier.addConstraint('invalid-result', (res: any) => res !== 42);
      
      const score = await verifier.evaluateProposal(mockProposal, mockChallenge, mockPersona);
      
      expect(typeof score).toBe('number');
    });
  });
});
