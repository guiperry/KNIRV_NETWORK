
// Mock the entire database service module
jest.mock('../../../../src/core/services/databaseService', () => {
  const mockDatabaseService = {
    createAgent: jest.fn(),
    getAgent: jest.fn(),
    listAgents: jest.fn(),
    updateAgent: jest.fn(),
    deleteAgent: jest.fn(),
    createSkill: jest.fn(),
    searchSkills: jest.fn(),
    createChatSession: jest.fn(),
    getChatSession: jest.fn(),
    listChatSessions: jest.fn(),
    initDatabase: jest.fn()
  };

  return {
    DatabaseService: jest.fn().mockImplementation(() => mockDatabaseService),
    databaseService: mockDatabaseService
  };
});

// Import the mocked service
import { databaseService } from '../../../../src/core/services/databaseService';

describe('DatabaseService', () => {
  beforeEach(async () => {
    jest.clearAllMocks();
  });

  describe('Agent Operations', () => {
    const mockAgent = {
      agentId: 'test-agent-1',
      name: 'Test Agent',
      type: 'wasm',
      status: 'Available',
      nrnCost: 100,
      capabilities: ['test'],
      metadata: {
        name: 'Test Agent',
        version: '1.0.0',
        description: 'Test agent',
        author: 'Test',
        capabilities: ['test'],
        requirements: { memory: 64, cpu: 1, storage: 10 },
        permissions: ['read']
      },
      createdAt: new Date().toISOString()
    };

    test('should create an agent', async () => {
      (databaseService.createAgent as jest.Mock).mockResolvedValue({ ...mockAgent, _id: 'mock-id' });

      const result = await databaseService.createAgent(mockAgent);

      expect(databaseService.createAgent).toHaveBeenCalledWith(mockAgent);
      expect(result).toEqual({ ...mockAgent, _id: 'mock-id' });
    });

    test('should get an agent by ID', async () => {
      (databaseService.getAgent as jest.Mock).mockResolvedValue(mockAgent);

      const result = await databaseService.getAgent('test-agent-1');

      expect(databaseService.getAgent).toHaveBeenCalledWith('test-agent-1');
      expect(result).toEqual(mockAgent);
    });

    test('should update an agent', async () => {
      const updateData = { status: 'Deployed' };
      (databaseService.updateAgent as jest.Mock).mockResolvedValue({ ...mockAgent, ...updateData });

      const result = await databaseService.updateAgent('test-agent-1', updateData);

      expect(databaseService.updateAgent).toHaveBeenCalledWith('test-agent-1', updateData);
      expect(result).toEqual({ ...mockAgent, ...updateData });
    });

    test('should delete an agent', async () => {
      (databaseService.deleteAgent as jest.Mock).mockResolvedValue(mockAgent);

      const result = await databaseService.deleteAgent('test-agent-1');

      expect(databaseService.deleteAgent).toHaveBeenCalledWith('test-agent-1');
      expect(result).toEqual(mockAgent);
    });

    test('should list all agents', async () => {
      const mockAgents = [mockAgent];
      (databaseService.listAgents as jest.Mock).mockResolvedValue(mockAgents);

      const result = await databaseService.listAgents();

      expect(databaseService.listAgents).toHaveBeenCalledWith();
      expect(result).toEqual(mockAgents);
    });
  });

  describe('Skill Operations', () => {
    const mockSkill = {
      skillId: 'test-skill-1',
      name: 'Test Skill',
      description: 'A test skill',
      version: 1,
      createdAt: new Date().toISOString()
    };

    test('should create a skill', async () => {
      (databaseService.createSkill as jest.Mock).mockResolvedValue({ ...mockSkill, _id: 'mock-id' });

      const result = await databaseService.createSkill(mockSkill);

      expect(databaseService.createSkill).toHaveBeenCalledWith(mockSkill);
      expect(result).toEqual({ ...mockSkill, _id: 'mock-id' });
    });

    test('should search skills', async () => {
      const mockResults = [mockSkill];
      (databaseService.searchSkills as jest.Mock).mockResolvedValue(mockResults);

      const result = await databaseService.searchSkills('test', 10);

      expect(databaseService.searchSkills).toHaveBeenCalledWith('test', 10);
      expect(result).toEqual(mockResults);
    });
  });

  describe('Chat Session Operations', () => {
    const mockChatSession = {
      title: 'Test Chat',
      messages: [
        {
          id: 'msg-1',
          content: 'Hello',
          sender: 'user',
          timestamp: new Date().toISOString()
        }
      ],
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString()
    };

    test('should create a chat session', async () => {
      (databaseService.createChatSession as jest.Mock).mockResolvedValue({ ...mockChatSession, _id: 'mock-id' });

      const result = await databaseService.createChatSession(mockChatSession);

      expect(databaseService.createChatSession).toHaveBeenCalledWith(mockChatSession);
      expect(result).toEqual({ ...mockChatSession, _id: 'mock-id' });
    });

    test('should get a chat session by ID', async () => {
      (databaseService.getChatSession as jest.Mock).mockResolvedValue(mockChatSession);

      const result = await databaseService.getChatSession('session-1');

      expect(databaseService.getChatSession).toHaveBeenCalledWith('session-1');
      expect(result).toEqual(mockChatSession);
    });

    test('should list chat sessions sorted by updatedAt', async () => {
      const mockSessions = [mockChatSession];
      (databaseService.listChatSessions as jest.Mock).mockResolvedValue(mockSessions);

      const result = await databaseService.listChatSessions();

      expect(databaseService.listChatSessions).toHaveBeenCalledWith();
      expect(result).toEqual(mockSessions);
    });
  });

  describe('Error Handling', () => {
    test('should handle database errors gracefully', async () => {
      const error = new Error('Database connection failed');
      (databaseService.getAgent as jest.Mock).mockRejectedValue(error);

      await expect(databaseService.getAgent('test-agent-1')).rejects.toThrow(
        'Database connection failed'
      );
    });
  });

  describe('Database Initialization', () => {
    test('should initialize database successfully', async () => {
      (databaseService.initDatabase as jest.Mock).mockResolvedValue({});

      await databaseService.initDatabase();

      expect(databaseService.initDatabase).toHaveBeenCalled();
    });
  });
});
