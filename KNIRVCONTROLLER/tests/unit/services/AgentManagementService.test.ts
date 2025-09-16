/**
 * Tests for AgentManagementService using real WASM files
 */

// Mock the database service to avoid RxDB issues in tests
const mockAgentData = new Map();

jest.mock('../../../src/core/services/databaseService', () => ({
  databaseService: {
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

import { agentManagementService, Agent, AgentUploadRequest, AgentDeploymentRequest } from '../../../src/services/AgentManagementService';
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

  describe('deployAgent', () => {
    let testAgent: Agent;

    beforeEach(async () => {
      const mockFile = new File(['test'], 'test.wasm');
      const uploadRequest: AgentUploadRequest = {
        file: mockFile,
        metadata: { name: 'Test Agent' },
        type: 'wasm'
      };
      testAgent = await agentManagementService.uploadAgent(uploadRequest);
    });

    it('should deploy an available agent successfully', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ deploymentId: 'test-deployment-123' })
      });

      const deploymentRequest: AgentDeploymentRequest = {
        agentId: testAgent.agentId,
        targetNRV: 'test-nrv',
        configuration: { param1: 'value1' },
        resources: { memory: 64, cpu: 1, timeout: 30000 }
      };

      const deploymentId = await agentManagementService.deployAgent(deploymentRequest);

      expect(deploymentId).toBe('test-deployment-123');

      // Get the updated agent from the service
      const deployedAgents = await agentManagementService.getDeployedAgents();
      const deployedAgent = deployedAgents.find(a => a.agentId === testAgent.agentId);

      expect(deployedAgent).toBeDefined();
      expect(deployedAgent!.status).toBe('Deployed');
    });

    it('should fail to deploy non-existent agent', async () => {
      const deploymentRequest: AgentDeploymentRequest = {
        agentId: 'non-existent-id'
      };

      await expect(agentManagementService.deployAgent(deploymentRequest))
        .rejects.toThrow('Agent non-existent-id not found');
    });

    it('should fail to deploy already deployed agent', async () => {
      // First deployment
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ deploymentId: 'test-deployment-123' })
      });

      await agentManagementService.deployAgent({ agentId: testAgent.agentId });

      // Second deployment attempt
      await expect(agentManagementService.deployAgent({ agentId: testAgent.agentId }))
        .rejects.toThrow(`Agent ${testAgent.agentId} is not available for deployment`);
    });

    it('should handle deployment API failure', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: false,
        statusText: 'Internal Server Error'
      });

      const deploymentRequest: AgentDeploymentRequest = {
        agentId: testAgent.agentId
      };

      await expect(agentManagementService.deployAgent(deploymentRequest))
        .rejects.toThrow('Deployment failed: Internal Server Error');

      // Agent should remain available
      expect(testAgent.status).toBe('Available');
    });
  });

  describe('executeSkill', () => {
    let deployedAgent: Agent;

    beforeEach(async () => {
      const mockFile = new File(['test'], 'test.wasm');
      const uploadRequest: AgentUploadRequest = {
        file: mockFile,
        metadata: { name: 'Test Agent' },
        type: 'wasm'
      };
      deployedAgent = await agentManagementService.uploadAgent(uploadRequest);

      // Deploy the agent
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ deploymentId: 'test-deployment' })
      });
      await agentManagementService.deployAgent({ agentId: deployedAgent.agentId });
    });

    it('should execute skill on deployed agent successfully', async () => {
      (fetch as jest.Mock).mockImplementationOnce(() =>
        new Promise(resolve =>
          setTimeout(() => resolve({
            ok: true,
            json: async () => ({
              output: { result: 'skill executed successfully' },
              resourceUsage: { memory: 32, cpu: 0.5 }
            })
          }), 10) // 10ms delay to ensure execution time > 0
        )
      );

      const result = await agentManagementService.executeSkill(
        deployedAgent.agentId,
        'test-skill',
        { param1: 'value1' }
      );

      expect(result.success).toBe(true);
      expect(result.output).toEqual({ result: 'skill executed successfully' });
      expect(result.resourceUsage).toEqual({ memory: 32, cpu: 0.5 });
      expect(result.executionTime).toBeGreaterThan(0);
    });

    it('should fail to execute skill on non-deployed agent', async () => {
      const nonDeployedAgent = await agentManagementService.uploadAgent({
        file: new File(['test'], 'test2.wasm'),
        metadata: { name: 'Non-deployed Agent' },
        type: 'wasm'
      });

      await expect(agentManagementService.executeSkill(nonDeployedAgent.agentId, 'test-skill', {}))
        .rejects.toThrow(`Agent ${nonDeployedAgent.agentId} is not deployed`);
    });

    it('should handle skill execution API failure', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: false,
        statusText: 'Skill Not Found'
      });

      const result = await agentManagementService.executeSkill(
        deployedAgent.agentId,
        'non-existent-skill',
        {}
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain('Skill execution failed');
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

    it('should undeploy and remove a deployed agent', async () => {
      const mockFile = new File(['test'], 'test.wasm');
      const agent = await agentManagementService.uploadAgent({
        file: mockFile,
        metadata: { name: 'Test Agent' },
        type: 'wasm'
      });

      // Deploy agent
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ deploymentId: 'test' })
      });
      await agentManagementService.deployAgent({ agentId: agent.agentId });

      // Mock undeploy
      (fetch as jest.Mock).mockResolvedValueOnce({ ok: true });

      await agentManagementService.removeAgent(agent.agentId);

      expect(await agentManagementService.getAgent(agent.agentId)).toBeNull();
      expect(agentManagementService.getDeployedAgents()).not.toContain(agent);
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

      // Deploy agent
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ deploymentId: 'test' })
      });
      await agentManagementService.deployAgent({ agentId: agent.agentId });

      const deployedAgents = await agentManagementService.getDeployedAgents();
      expect(deployedAgents).toHaveLength(1);
      expect(deployedAgents[0].agentId).toBe(agent.agentId);
    });
  });
});
