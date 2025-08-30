import { NebulaDB } from 'nebuladb';

// Mock NebulaDB for testing
jest.mock('nebuladb', () => {
  const mockCollection = {
    insertOne: jest.fn(),
    findOne: jest.fn(),
    find: jest.fn(),
    updateOne: jest.fn(),
    deleteOne: jest.fn(),
    deleteMany: jest.fn(),
    search: jest.fn(),
  };

  const mockDB = {
    defineSchema: jest.fn(() => ({
      addPlugin: jest.fn(() => ({}))
    })),
    defineCollection: jest.fn(() => mockCollection),
  };

  return {
    NebulaDB: jest.fn(() => mockDB),
    fullTextSearch: jest.fn(),
  };
});

// Mock fs and path modules
jest.mock('fs', () => ({
  existsSync: jest.fn(() => true),
  mkdirSync: jest.fn(),
}));

jest.mock('path', () => ({
  resolve: jest.fn(() => '/mock/path'),
  join: jest.fn(() => '/mock/path/file.db'),
}));

describe('DatabaseService', () => {
  let databaseService: any;
  let mockCollection: any;

  beforeEach(() => {
    jest.clearAllMocks();
    
    // Re-import to get fresh instance
    delete require.cache[require.resolve('../../../../src/core/services/databaseService')];
    const { databaseService: service } = require('../../../../src/core/services/databaseService');
    databaseService = service;
    
    // Get mock collection reference
    const { NebulaDB } = require('nebuladb');
    const mockDB = new NebulaDB();
    mockCollection = mockDB.defineCollection();
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
      createdAt: new Date()
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
      createdAt: new Date()
    };

    test('should create a skill', async () => {
      mockCollection.insertOne.mockResolvedValue({ ...mockSkill, _id: 'mock-id' });

      const result = await databaseService.createSkill(mockSkill);

      expect(mockCollection.insertOne).toHaveBeenCalledWith(mockSkill);
      expect(result).toEqual({ ...mockSkill, _id: 'mock-id' });
    });

    test('should search skills', async () => {
      const mockResults = [mockSkill];
      mockCollection.search.mockResolvedValue(mockResults);

      const result = await databaseService.searchSkills('test', 10);

      expect(mockCollection.search).toHaveBeenCalledWith({
        term: 'test',
        limit: 10,
      });
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
          timestamp: new Date()
        }
      ],
      createdAt: new Date(),
      updatedAt: new Date()
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
    test('should initialize database with correct configuration', () => {
      const { NebulaDB } = require('nebuladb');
      
      expect(NebulaDB).toHaveBeenCalledWith({
        filePath: '/mock/path/file.db',
        autoload: true,
        autosave: true,
        autosaveInterval: 4000,
      });
    });

    test('should create data directory if it does not exist', () => {
      const fs = require('fs');
      
      // Mock fs.existsSync to return false
      fs.existsSync.mockReturnValue(false);
      
      // Re-import to trigger directory creation
      delete require.cache[require.resolve('../../../../src/core/services/databaseService')];
      require('../../../../src/core/services/databaseService');
      
      expect(fs.mkdirSync).toHaveBeenCalledWith('/mock/path', { recursive: true });
    });
  });
});
