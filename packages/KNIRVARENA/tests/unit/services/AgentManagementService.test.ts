/**
 * Tests for AgentManagementService using real WASM files
 */

// Mock the browser store (replaced the KNIRVBASE databaseService — §3.3)
const mockAgentData = new Map();

jest.mock('../../../src/storage/arenaStore', () => ({
  arenaStore: {
    createAgent: jest.fn().mockImplementation(async (agentData) => {
      const agent = {
        ...agentData,
        _id: Math.random().toString(36),
        agentId: agentData.agentId || Math.random().toString(36)
      };
      mockAgentData.set(agent.agentId, agent);
      return agent;
    }),
    updateAgent: jest.fn().mockImplementation(async (agentId, updateData) => {
      const existingAgent = mockAgentData.get(agentId);
      if (existingAgent) {
        const updatedAgent = { ...existingAgent, ...updateData };
        mockAgentData.set(agentId, updatedAgent);
        return updatedAgent;
      }
      return null;
    }),
    getAgent: jest.fn().mockImplementation(async (agentId) => {
      return mockAgentData.get(agentId) || null;
    }),
    getAllAgents: jest.fn().mockImplementation(async () => {
      return Array.from(mockAgentData.values());
    }),
    listAgents: jest.fn().mockImplementation(async () => {
      return Array.from(mockAgentData.values());
    }),
    deleteAgent: jest.fn().mockImplementation(async (agentId) => {
      const deleted = mockAgentData.has(agentId);
      mockAgentData.delete(agentId);
      return deleted;
    })
  }
}));

import { agentManagementService, AgentUploadRequest } from '../../../src/services/AgentManagementService';
import { loadWasmAsFile, createFileFromTestWasm, validateWasmFile } from '../../../test-utils/wasm-test-utils';

// Mock fetch globally
global.fetch = jest.fn();

describe('AgentManagementService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    // Clear mock agent data between tests
    mockAgentData.clear();
  });

  describe('uploadAgent', () => {
    it('should upload a real WASM agent successfully', async () => {
      const testWasm = await loadWasmAsFile('KNIRV_CONTROLLER_DEBUG', 'test-agent.wasm');
      const file = createFileFromTestWasm(testWasm);

      const uploadRequest: AgentUploadRequest = {
        file,
        metadata: {
          name: 'Test WASM Agent',
          description: 'A real WASM test agent',
          author: 'Test Author'
        },
        type: 'wasm'
      };

      const agent = await agentManagementService.uploadAgent(uploadRequest);

      expect(agent).toBeDefined();
      expect(agent.name).toBe('Test WASM Agent');
      expect(agent.type).toBe('wasm');
      expect(agent.status).toBe('Available');
      expect(agent.metadata.name).toBe('Test WASM Agent');
      expect(agent.nrnCost).toBeGreaterThan(0);
      expect(agent.wasmModule).toBeDefined();

      // Validate the WASM file was processed correctly
      expect(await validateWasmFile(testWasm)).toBe(true);
    });

    it('should upload a LoRA agent successfully', async () => {
      const mockFile = new File(['{"model": "test"}'], 'test-lora.json', { type: 'application/json' });
      
      const uploadRequest: AgentUploadRequest = {
        file: mockFile,
        metadata: {
          name: 'LoRA Agent',
          description: 'A LoRA test agent'
        },
        type: 'lora'
      };

      const agent = await agentManagementService.uploadAgent(uploadRequest);

      expect(agent.type).toBe('lora');
      expect(agent.loraAdapter).toBeDefined();
    });

    it('should calculate NRN cost based on requirements', async () => {
      const mockFile = new File(['test'], 'test.wasm');
      
      const uploadRequest: AgentUploadRequest = {
        file: mockFile,
        metadata: {
          requirements: {
            memory: 128,
            cpu: 2,
            storage: 20
          }
        },
        type: 'wasm'
      };

      const agent = await agentManagementService.uploadAgent(uploadRequest);
      
      // Base cost (10) + memory (128 * 0.1) + cpu (2 * 5) + storage (20 * 0.05) = 10 + 12.8 + 10 + 1 = 33.8 -> 34
      expect(agent.nrnCost).toBe(34);
    });
  });

  describe('removeAgent', () => {
    it('should remove an available agent', async () => {
      const mockFile = new File(['test'], 'test.wasm');
      const agent = await agentManagementService.uploadAgent({
        file: mockFile,
        metadata: { name: 'Test Agent' },
        type: 'wasm'
      });

      await agentManagementService.removeAgent(agent.agentId);

      expect(await agentManagementService.getAgent(agent.agentId)).toBeNull();
    });

    // Agents run on the paired CLI (§2); the arena cannot undeploy them, so a
    // deployed agent must be recalled there first.
    it('refuses to remove a deployed agent', async () => {
      const agent = await agentManagementService.uploadAgent({
        file: new File(['test'], 'test.wasm'),
        metadata: { name: 'Test Agent' },
        type: 'wasm'
      });
      mockAgentData.set(agent.agentId, { ...mockAgentData.get(agent.agentId), status: 'Deployed' });

      await expect(agentManagementService.removeAgent(agent.agentId)).rejects.toThrow(/recall it from its error node first/);
      expect(await agentManagementService.getAgent(agent.agentId)).not.toBeNull();
      expect(fetch).not.toHaveBeenCalled();
    });
  });

  describe('getters', () => {
    it('should return all agents', async () => {
      const mockFile1 = new File(['test1'], 'test1.wasm');
      const mockFile2 = new File(['test2'], 'test2.wasm');

      const agent1 = await agentManagementService.uploadAgent({
        file: mockFile1,
        metadata: { name: 'Agent 1' },
        type: 'wasm'
      });

      const agent2 = await agentManagementService.uploadAgent({
        file: mockFile2,
        metadata: { name: 'Agent 2' },
        type: 'wasm'
      });

      const agents = await agentManagementService.getAgents();
      expect(agents).toHaveLength(2);
      expect(agents.some(a => a.agentId === agent1.agentId)).toBe(true);
      expect(agents.some(a => a.agentId === agent2.agentId)).toBe(true);
    });

    it('should return deployed agents only', async () => {
      const mockFile = new File(['test'], 'test.wasm');
      const agent = await agentManagementService.uploadAgent({
        file: mockFile,
        metadata: { name: 'Test Agent' },
        type: 'wasm'
      });

      expect(await agentManagementService.getDeployedAgents()).toHaveLength(0);

      mockAgentData.set(agent.agentId, { ...mockAgentData.get(agent.agentId), status: 'Deployed' });

      const deployedAgents = await agentManagementService.getDeployedAgents();
      expect(deployedAgents).toHaveLength(1);
      expect(deployedAgents[0].agentId).toBe(agent.agentId);
    });
  });
});
