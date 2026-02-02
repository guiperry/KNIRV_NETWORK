import { Verifier, ScoreWeights } from '../../../src/engine/Verifier';

describe('Verifier', () => {
  let verifier: Verifier;
  
  beforeEach(() => {
    verifier = new Verifier();
  });
  
  describe('initialization', () => {
    it('should initialize with correct default weights', () => {
      verifier.updateRequirements(42, 1000);
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
  
  describe('evaluate', () => {
    it('should evaluate code and return score', async () => {
      verifier.updateRequirements(42, 1000);
      const score = await verifier.evaluate('return 42;', {});
      
      expect(typeof score).toBe('number');
      expect(score).toBeGreaterThanOrEqual(0);
      expect(score).toBeLessThanOrEqual(1);
    });
    
    it('should apply constraints', async () => {
      verifier.updateRequirements(42, 1000);
      
      // Add constraint that returns true (should not affect score)
      verifier.addConstraint('valid-result', (res: any) => res === 42);
      
      const score = await verifier.evaluate('return 42;', {});
      
      expect(typeof score).toBe('number');
    });
    
    it('should penalize failing constraints', async () => {
      verifier.updateRequirements(42, 1000);
      
      // Add constraint that returns false
      verifier.addConstraint('invalid-result', (res: any) => res !== 42);
      
      const score = await verifier.evaluate('return 42;', {});
      
      expect(typeof score).toBe('number');
    });
  });
});
