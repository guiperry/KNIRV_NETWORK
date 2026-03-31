import { LoraxClient } from '../../../src/networking/LoraxClient';

describe('LoraxClient', () => {
  let loraxClient: LoraxClient;
  
  beforeEach(() => {
    loraxClient = new LoraxClient('http://localhost:8080');
  });
  
  describe('fineTune', () => {
    it('should call LoRAX API to fine-tune', async () => {
      const mockAgent = {
        id: 'test-agent',
        resources: { generation: 1 }
      };
      
      const mockProposal = {
        chainOfThought: ['Test thought'],
        code: 'function test() { return 42; }'
      };
      
      const result = await loraxClient.fineTune(mockAgent, mockProposal, 0.9);
      
      expect(result).toEqual(expect.objectContaining({ success: true }));
    });
  });
  
  describe('commitToMainSlot', () => {
    it('should commit adapter to main slot', async () => {
      const result = await loraxClient.commitToMainSlot('test-adapter');
      
      expect(result).toEqual(expect.objectContaining({ success: true }));
    });
  });
});
