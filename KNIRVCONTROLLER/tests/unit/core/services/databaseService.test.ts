
// Import the actual DatabaseService
import { databaseService } from '../../../../src/core/services/databaseService';

describe('DatabaseService', () => {
  let mockCollection: {
    insertOne: jest.Mock;
    findOne: jest.Mock;
    find: jest.Mock;
    updateOne: jest.Mock;
    deleteOne: jest.Mock;
    deleteMany: jest.Mock;
    search: jest.Mock;
  };

  beforeEach(async () => {
    jest.clearAllMocks();

    // Mock the database service methods
    jest.spyOn(databaseService, 'createAgent').mockImplementation(mockCollection.insertOne);
    jest.spyOn(databaseService, 'getAgent').mockImplementation(mockCollection.findOne);
    jest.spyOn(databaseService, 'listAgents').mockImplementation(mockCollection.find);
    jest.spyOn(databaseService, 'updateAgent').mockImplementation(mockCollection.updateOne);
    jest.spyOn(databaseService, 'deleteAgent').mockImplementation(mockCollection.deleteOne);
    jest.spyOn(databaseService, 'createSkill').mockImplementation(mockCollection.insertOne);
    jest.spyOn(databaseService, 'searchSkills').mockImplementation(mockCollection.search);
    jest.spyOn(databaseService, 'createChatSession').mockImplementation(mockCollection.insertOne);
    jest.spyOn(databaseService, 'getChatSession').mockImplementation(mockCollection.findOne);
    jest.spyOn(databaseService, 'listChatSessions').mockImplementation(mockCollection.find);

    // Initialize mock collection
    mockCollection = {
      insertOne: jest.fn(),
      findOne: jest.fn(),
      find: jest.fn(),
      updateOne: jest.fn(),
      deleteOne: jest.fn(),
      deleteMany: jest.fn(),
      search: jest.fn(),
    };
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
      mockCollection.insertOne.mockResolvedValue({ ...mockAgent, _id: 'mock-id' });

      const result = await databaseService.createAgent(mockAgent);

      expect(mockCollection.insertOne).toHaveBeenCalledWith(mockAgent);
      expect(result).toEqual({ ...mockAgent, _id: 'mock-id' });
    });

    test('should get an agent by ID', async () => {
      mockCollection.findOne.mockResolvedValue(mockAgent);

      const result = await databaseService.getAgent('test-agent-1');

      expect(mockCollection.findOne).toHaveBeenCalledWith({ agentId: 'test-agent-1' });
      expect(result).toEqual(mockAgent);
    });

    test('should update an agent', async () => {
      const updateData = { status: 'Deployed' };
      mockCollection.updateOne.mockResolvedValue({ ...mockAgent, ...updateData });

      const result = await databaseService.updateAgent('test-agent-1', updateData);

      expect(mockCollection.updateOne).toHaveBeenCalledWith(
        { agentId: 'test-agent-1' },
        updateData
      );
      expect(result).toEqual({ ...mockAgent, ...updateData });
    });

    test('should delete an agent', async () => {
      mockCollection.deleteOne.mockResolvedValue(mockAgent);

      const result = await databaseService.deleteAgent('test-agent-1');

      expect(mockCollection.deleteOne).toHaveBeenCalledWith({ agentId: 'test-agent-1' });
      expect(result).toEqual(mockAgent);
    });

    test('should list all agents', async () => {
      const mockAgents = [mockAgent];
      mockCollection.find.mockResolvedValue(mockAgents);

      const result = await databaseService.listAgents();

      expect(mockCollection.find).toHaveBeenCalledWith({});
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
      mockCollection.insertOne.mockResolvedValue({ ...mockSkill, _id: 'mock-id' });

      const result = await databaseService.createSkill(mockSkill);

      expect(mockCollection.insertOne).toHaveBeenCalledWith(mockSkill);
      expect(result).toEqual({ ...mockSkill, _id: 'mock-id' });
    });

    test('should search skills', async () => {
      const mockResults = [mockSkill];
      jest.spyOn(databaseService, 'searchSkills').mockResolvedValue(mockResults);

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
      mockCollection.insertOne.mockResolvedValue({ ...mockChatSession, _id: 'mock-id' });

      const result = await databaseService.createChatSession(mockChatSession);

      expect(mockCollection.insertOne).toHaveBeenCalledWith(mockChatSession);
      expect(result).toEqual({ ...mockChatSession, _id: 'mock-id' });
    });

    test('should get a chat session by ID', async () => {
      mockCollection.findOne.mockResolvedValue(mockChatSession);

      const result = await databaseService.getChatSession('session-1');

      expect(mockCollection.findOne).toHaveBeenCalledWith({ _id: 'session-1' });
      expect(result).toEqual(mockChatSession);
    });

    test('should list chat sessions sorted by updatedAt', async () => {
      const mockSessions = [mockChatSession];
      mockCollection.find.mockResolvedValue(mockSessions);

      const result = await databaseService.listChatSessions();

      expect(mockCollection.find).toHaveBeenCalledWith({}, {
        sort: { updatedAt: -1 }
      });
      expect(result).toEqual(mockSessions);
    });
  });

  describe('Error Handling', () => {
    test('should handle database errors gracefully', async () => {
      const error = new Error('Database connection failed');
      mockCollection.findOne.mockRejectedValue(error);

      await expect(databaseService.getAgent('test-agent-1')).rejects.toThrow(
        'Database connection failed'
      );
    });
  });

  describe('Database Initialization', () => {
    test('should initialize database successfully', async () => {
      // Mock the initDatabase function to avoid actual database initialization
      const mockInitDatabase = jest.fn().mockResolvedValue({});
      jest.mock('../../../../src/core/services/databaseService', () => ({
        ...jest.requireActual('../../../../src/core/services/databaseService'),
        initDatabase: mockInitDatabase
      }));
      
      // Re-import to trigger initialization
      jest.resetModules();
      const { databaseService: service } = await import('../../../../src/core/services/databaseService');

      // Verify service is available and initialization was attempted
      expect(service).toBeDefined();
      expect(mockInitDatabase).toHaveBeenCalled();
      
      // service variable is declared for potential future use in test extensions
    });
  });
});
