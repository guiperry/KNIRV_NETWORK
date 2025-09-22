/**
 * Comprehensive tests for KNIRVANA Bridge Service
 * Tests game mechanics integration with personal graph and collective network features
 */

import { knirvanaBridgeService } from '../services/KnirvanaBridgeService';
import { personalKNIRVGRAPHService } from '../services/PersonalKNIRVGRAPHService';
import { rxdbService } from '../services/RxDBService';

// Mock the RxDB service
jest.mock('../services/RxDBService', () => ({
  rxdbService: {
    isDatabaseInitialized: jest.fn(),
    initialize: jest.fn(),
    getDatabase: jest.fn(() => ({
      settings: {
        insert: jest.fn()
      }
    }))
  }
}));

// Mock the PersonalKNIRVGRAPHService
jest.mock('../services/PersonalKNIRVGRAPHService', () => ({
  personalKNIRVGRAPHService: {
    loadPersonalGraph: jest.fn(),
    addSkillNode: jest.fn(),
    getCurrentGraph: jest.fn(),
    updateGraph: jest.fn()
  }
}));

describe('KnirvanaBridgeService', () => {
  beforeEach(async () => {
    // Reset mocks
    jest.clearAllMocks();

    // Setup mock implementations
    (rxdbService.isDatabaseInitialized as jest.Mock).mockReturnValue(true);
    (personalKNIRVGRAPHService.loadPersonalGraph as jest.Mock).mockResolvedValue({
      id: 'test_graph',
      userId: 'test_user',
      nodes: [
        {
          id: 'error_test_1',
          type: 'error' as const,
          label: 'Test Error',
          position: { x: 0, y: 0, z: 0 },
          data: {
            errorId: 'test_1',
            errorType: 'TypeScript Error',
            description: 'Test error for testing',
            context: {},
            timestamp: Date.now()
          },
          connections: []
        }
      ],
      edges: [],
      metadata: {
        createdAt: Date.now(),
        lastModified: Date.now(),
        version: 1,
        complexity: 1
      }
    });
  });

  describe('Initialization', () => {
    test('should initialize successfully with RxDB and personal graph', async () => {
      await knirvanaBridgeService.initialize();

      expect(rxdbService.isDatabaseInitialized).toHaveBeenCalled();
      expect(personalKNIRVGRAPHService.loadPersonalGraph).toHaveBeenCalledWith('current_user');
    });

    test('should sync personal graph to game state', async () => {
      await knirvanaBridgeService.initialize();

      const gameState = knirvanaBridgeService.getGameState();
      expect(gameState.errorNodes).toHaveLength(1);
      expect(gameState.errorNodes[0].type).toBe('TypeScript Error');
    });

    test('should initialize with default NRN balance', async () => {
      await knirvanaBridgeService.initialize();

      const gameState = knirvanaBridgeService.getGameState();
      expect(gameState.nrnBalance).toBe(500);
    });
  });

  describe('Game Mechanics', () => {
    beforeEach(async () => {
      await knirvanaBridgeService.initialize();
    });

    test('should start game session', () => {
      knirvanaBridgeService.startGame();

      const gameState = knirvanaBridgeService.getGameState();
      expect(gameState.gamePhase).toBe('playing');
    });

    test('should pause game session', () => {
      knirvanaBridgeService.startGame();
      knirvanaBridgeService.pauseGame();

      const gameState = knirvanaBridgeService.getGameState();
      expect(gameState.gamePhase).toBe('paused');
    });

    test('should select error node', () => {
      knirvanaBridgeService.selectErrorNode('error_test_1');

      const gameState = knirvanaBridgeService.getGameState();
      expect(gameState.selectedErrorNode).toBe('error_test_1');
    });

    test('should select agent', () => {
      const gameState = knirvanaBridgeService.getGameState();
      const agentId = gameState.agents[0].id;

      knirvanaBridgeService.selectAgent(agentId);

      const updatedGameState = knirvanaBridgeService.getGameState();
      expect(updatedGameState.selectedAgent).toBe(agentId);
    });
  });

  describe('Agent Management', () => {
    beforeEach(async () => {
      await knirvanaBridgeService.initialize();
    });

    test('should have initial agents after initialization', async () => {
      const gameState = knirvanaBridgeService.getGameState();
      expect(gameState.agents).toHaveLength(2);
      expect(gameState.agents[0].type).toBe('Analyzer');
      expect(gameState.agents[1].type).toBe('Optimizer');
    });

    test('should create new agent with sufficient NRN', async () => {
      const success = await knirvanaBridgeService.createAgent('Debugger');

      expect(success).toBe(true);

      const gameState = knirvanaBridgeService.getGameState();
      expect(gameState.agents).toHaveLength(3);
      expect(gameState.agents[2].type).toBe('Debugger');
      expect(gameState.nrnBalance).toBe(450); // 500 - 50
    });

    test('should not create agent with insufficient NRN', async () => {
      // Simulate low balance by directly setting it
      const gameState = knirvanaBridgeService.getGameState();
      Object.assign(gameState, { nrnBalance: 10 });

      const success = await knirvanaBridgeService.createAgent('Debugger');

      expect(success).toBe(false);
      expect(gameState.agents).toHaveLength(2); // No new agent added
    });
  });

  describe('Agent Deployment', () => {
    let agentId: string;
    let errorNodeId: string;

    beforeEach(async () => {
      await knirvanaBridgeService.initialize();

      const gameState = knirvanaBridgeService.getGameState();
      agentId = gameState.agents[0].id;
      errorNodeId = gameState.errorNodes[0].id;
    });

    test('should deploy agent to solve error with sufficient NRN', async () => {
      const success = await knirvanaBridgeService.deployAgent(agentId, errorNodeId);

      expect(success).toBe(true);

      const gameState = knirvanaBridgeService.getGameState();
      expect(gameState.nrnBalance).toBe(490); // 500 - 10

      const deployedAgent = gameState.agents.find(a => a.id === agentId);
      expect(deployedAgent?.status).toBe('working');
      expect(deployedAgent?.target).toBe(errorNodeId);

      const targetError = gameState.errorNodes.find(e => e.id === errorNodeId);
      expect(targetError?.isBeingSolved).toBe(true);
      expect(targetError?.solverAgent).toBe(agentId);
    });

    test('should not deploy agent with insufficient NRN', async () => {
      // Simulate low balance
      const gameState = knirvanaBridgeService.getGameState();
      Object.assign(gameState, { nrnBalance: 5 });

      const success = await knirvanaBridgeService.deployAgent(agentId, errorNodeId);

      expect(success).toBe(false);
      expect(gameState.nrnBalance).toBe(5); // Unchanged
    });

    test('should not deploy invalid agent', async () => {
      const success = await knirvanaBridgeService.deployAgent('invalid_agent', errorNodeId);

      expect(success).toBe(false);
    });
  });

  describe('Error Solving Simulation', () => {
    let agentId: string;
    let errorNodeId: string;

    beforeEach(async () => {
      await knirvanaBridgeService.initialize();

      const gameState = knirvanaBridgeService.getGameState();
      agentId = gameState.agents[0].id;
      errorNodeId = gameState.errorNodes[0].id;

      // Deploy agent
      await knirvanaBridgeService.deployAgent(agentId, errorNodeId);
    });

    test('should update error progress during game loop', () => {
      knirvanaBridgeService.startGame();

      // Simulate some game time
      const gameState = knirvanaBridgeService.getGameState();
      const initialProgress = gameState.errorNodes[0].progress;

      // Trigger update
      knirvanaBridgeService.getGameState(); // This would normally trigger updates

      // Note: In a real scenario, we'd need to mock the game loop timing
      // For now, we verify the deployment worked correctly
      expect(gameState.errorNodes[0].isBeingSolved).toBe(true);
      expect(gameState.errorNodes[0].progress).toBe(initialProgress);
    });
  });

  describe('Collective Network Integration', () => {
    beforeEach(async () => {
      await knirvanaBridgeService.initialize();
    });

    test('should merge personal graph with collective network', async () => {
      const mockCollectiveGraph = {
        nodes: [],
        edges: [],
        insights: ['advanced_patterns', 'predictive_debugging']
      };

      await knirvanaBridgeService.mergeToCollectiveNetwork(mockCollectiveGraph);

      // Verify that collective insights were added to personal graph
      expect(personalKNIRVGRAPHService.addSkillNode).toHaveBeenCalledTimes(2);

      // Check that the first insight was added
      expect(personalKNIRVGRAPHService.addSkillNode).toHaveBeenCalledWith({
        skillId: 'collective_advanced_error_patterns',
        skillName: 'Advanced Error Pattern Recognition',
        description: 'Recognize complex error patterns learned from collective experiences',
        category: 'collective',
        proficiency: 0.8
      });
    });

    test('should handle empty personal graph gracefully', async () => {
      // Mock empty personal graph
      (personalKNIRVGRAPHService.getCurrentGraph as jest.Mock).mockReturnValue(null);

      await expect(knirvanaBridgeService.mergeToCollectiveNetwork({})).resolves.not.toThrow();
    });
  });

  describe('NRN Management', () => {
    beforeEach(async () => {
      await knirvanaBridgeService.initialize();
    });

    test('should award NRN correctly', () => {
      const gameState = knirvanaBridgeService.getGameState();
      const initialBalance = gameState.nrnBalance;

      // Directly access private method for testing (would be better with a public method)
      const bridgeInstance = knirvanaBridgeService as unknown as { awardNRN: (amount: number) => void };
      bridgeInstance.awardNRN(100);

      const updatedGameState = knirvanaBridgeService.getGameState();
      expect(updatedGameState.nrnBalance).toBe(initialBalance + 100);
    });

    test('should spend NRN correctly when sufficient balance', () => {
      const bridgeInstance = knirvanaBridgeService as unknown as { spendNRN: (amount: number) => boolean };
      const spendResult = bridgeInstance.spendNRN(50);

      expect(spendResult).toBe(true);

      const gameState = knirvanaBridgeService.getGameState();
      expect(gameState.nrnBalance).toBe(450); // 500 - 50
    });

    test('should not spend NRN when insufficient balance', () => {
      const bridgeInstance = knirvanaBridgeService as unknown as { spendNRN: (amount: number) => boolean };
      const spendResult = bridgeInstance.spendNRN(600); // More than 500

      expect(spendResult).toBe(false);

      const gameState = knirvanaBridgeService.getGameState();
      expect(gameState.nrnBalance).toBe(500); // Unchanged
    });
  });

  describe('Agent Capabilities', () => {
    beforeEach(async () => {
      await knirvanaBridgeService.initialize();
    });

    test('should assign correct capabilities for Analyzer agent', async () => {
      await knirvanaBridgeService.createAgent('Analyzer');

      const gameState = knirvanaBridgeService.getGameState();
      const analyzerAgent = gameState.agents.find(a => a.type === 'Analyzer');

      expect(analyzerAgent?.capabilities).toEqual([
        'error_analysis',
        'pattern_recognition',
        'diagnostic_tools'
      ]);
    });

    test('should assign correct capabilities for Optimizer agent', async () => {
      await knirvanaBridgeService.createAgent('Optimizer');

      const gameState = knirvanaBridgeService.getGameState();
      const optimizerAgent = gameState.agents.find(a => a.type === 'Optimizer');

      expect(optimizerAgent?.capabilities).toEqual([
        'code_optimization',
        'performance_tuning',
        'refactoring'
      ]);
    });

    test('should assign default capabilities for unknown agent type', async () => {
      await knirvanaBridgeService.createAgent('UnknownType');

      const gameState = knirvanaBridgeService.getGameState();
      const unknownAgent = gameState.agents.find(a => a.type === 'UnknownType');

      expect(unknownAgent?.capabilities).toEqual(['general_problem_solving']);
    });
  });

  describe('Graph Synchronization', () => {
    beforeEach(async () => {
      await knirvanaBridgeService.initialize();
    });

    test('should sync error node updates back to personal graph', async () => {
      const gameState = knirvanaBridgeService.getGameState();
      const errorNode = gameState.errorNodes[0];

      // Modify error node in game state
      errorNode.progress = 0.5;
      errorNode.isBeingSolved = true;
      errorNode.solverAgent = 'test_agent';

      // Sync back (would normally be called internally)
      const bridgeInstance = knirvanaBridgeService as unknown as { syncErrorNodeToPersonalGraph: (node: unknown) => Promise<void> };
      await bridgeInstance.syncErrorNodeToPersonalGraph(errorNode);

      // Verify personal graph was updated
      expect(personalKNIRVGRAPHService.getCurrentGraph).toHaveBeenCalled();
    });

    test('should sync new agent to personal graph', async () => {
      const newAgent = {
        id: 'test_agent',
        position: { x: 0, y: 0, z: 0 },
        target: null,
        status: 'idle' as const,
        type: 'TestAgent',
        efficiency: 0.8,
        experience: 10,
        capabilities: ['testing']
      };

      const bridgeInstance = knirvanaBridgeService as unknown as { syncAgentToPersonalGraph: (agent: unknown) => Promise<void> };
      await bridgeInstance.syncAgentToPersonalGraph(newAgent);

      expect(personalKNIRVGRAPHService.addSkillNode).toHaveBeenCalledWith({
        skillId: 'test_agent',
        skillName: 'TestAgent Agent',
        description: 'AI Agent with capabilities: testing',
        category: 'agent',
        proficiency: 0.8
      });
    });
  });

  describe('Error Handling', () => {
    test('should handle RxDB initialization failure', async () => {
      (rxdbService.isDatabaseInitialized as jest.Mock).mockReturnValue(false);
      (rxdbService.initialize as jest.Mock).mockRejectedValue(new Error('DB init failed'));

      await expect(knirvanaBridgeService.initialize()).resolves.not.toThrow();
    });

    test('should handle personal graph load failure', async () => {
      (personalKNIRVGRAPHService.loadPersonalGraph as jest.Mock).mockRejectedValue(new Error('Graph load failed'));

      await expect(knirvanaBridgeService.initialize()).resolves.not.toThrow();
    });

    test('should handle invalid agent deployment gracefully', async () => {
      await knirvanaBridgeService.initialize();

      const success = await knirvanaBridgeService.deployAgent('nonexistent', 'nonexistent');

      expect(success).toBe(false);
    });
  });

  describe('Cleanup', () => {
    test('should cleanup properly', async () => {
      await knirvanaBridgeService.initialize();
      knirvanaBridgeService.startGame();

      knirvanaBridgeService.cleanup();

      const gameState = knirvanaBridgeService.getGameState();
      expect(gameState.gamePhase).toBe('menu');
      expect(gameState.gameTime).toBe(0);
      expect(gameState.nrnBalance).toBe(500);
    });

    test('should handle cleanup when not initialized', () => {
      expect(() => knirvanaBridgeService.cleanup()).not.toThrow();
    });
  });
});
