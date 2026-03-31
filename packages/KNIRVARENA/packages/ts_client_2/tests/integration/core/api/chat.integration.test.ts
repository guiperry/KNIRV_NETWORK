import request from 'supertest';
import express from 'express';
import { NebulaDB } from 'nebuladb';
import * as chatAPI from '../../../../src/core/api/chat';

// Mock NebulaDB for integration tests
jest.mock('nebuladb', () => ({
  NebulaDB: jest.fn()
}));

interface MockDB {
  collection: (name: string) => MockCollection;
  close: jest.Mock;
  defineSchema: jest.Mock;
  defineCollection: jest.Mock;
  connect: jest.Mock;
  disconnect: jest.Mock;
  query: jest.Mock;
  transaction: jest.Mock;
  createDatabase: jest.Mock;
  dropDatabase: jest.Mock;
  listDatabases: jest.Mock;
  createTable: jest.Mock;
  dropTable: jest.Mock;
  listTables: jest.Mock;
}

interface MockCollection {
  insertOne: jest.Mock;
  findOne: jest.Mock;
  updateOne: jest.Mock;
  find: jest.Mock;
  deleteOne: jest.Mock;
  insert: jest.Mock;
  update: jest.Mock;
  delete: jest.Mock;
  count: jest.Mock;
}

describe('Chat API Integration Tests', () => {
  let app: express.Application;
  let mockDB: MockDB;
  let mockChatSessions: MockCollection;

  beforeEach(() => {
    // Setup Express app
    app = express();
    app.use(express.json());
    
    // Setup routes
    app.get('/api/chat/sessions', chatAPI.getSessions);
    app.get('/api/chat/sessions/:sessionId', chatAPI.getSession);
    app.post('/api/chat/sessions', chatAPI.createSession);
    app.put('/api/chat/sessions/:sessionId', chatAPI.updateSession);
    app.delete('/api/chat/sessions/:sessionId', chatAPI.deleteSession);
    app.post('/api/chat/sessions/:sessionId/messages', chatAPI.addMessage);
    app.get('/api/chat/sessions/:sessionId/messages', chatAPI.getMessages);
    app.get('/api/chat/search', chatAPI.searchSessions);

    // Setup NebulaDB mocks
    mockChatSessions = {
      find: jest.fn(),
      findOne: jest.fn(),
      insertOne: jest.fn(),
      updateOne: jest.fn(),
      deleteOne: jest.fn(),
      insert: jest.fn(),
      update: jest.fn(),
      delete: jest.fn(),
      count: jest.fn(),
    };

    mockDB = {
      collection: jest.fn(() => mockChatSessions),
      close: jest.fn(),
      defineSchema: jest.fn(() => ({})),
      defineCollection: jest.fn(() => mockChatSessions),
      connect: jest.fn(),
      disconnect: jest.fn(),
      query: jest.fn(),
      transaction: jest.fn(),
      createDatabase: jest.fn(),
      dropDatabase: jest.fn(),
      listDatabases: jest.fn(),
      createTable: jest.fn(),
      dropTable: jest.fn(),
      listTables: jest.fn()
    };

    (NebulaDB as jest.MockedClass<typeof NebulaDB>).mockImplementation(() => mockDB as any);

    // Reset mocks
    jest.clearAllMocks();
  });

  describe('GET /api/chat/sessions', () => {
    test('should return all chat sessions', async () => {
      const mockSessions = [
        {
          _id: 'session-1',
          title: 'Test Session 1',
          createdAt: new Date(),
          updatedAt: new Date(),
          messages: []
        }
      ];

      mockChatSessions.find.mockResolvedValue(mockSessions);

      const response = await request(app)
        .get('/api/chat/sessions')
        .expect(200);

      expect(response.body).toEqual({ sessions: mockSessions });
      expect(mockChatSessions.find).toHaveBeenCalledWith({}, {
        sort: { updatedAt: -1 }
      });
    });

    test('should handle database errors', async () => {
      mockChatSessions.find.mockRejectedValue(new Error('Database error'));

      const response = await request(app)
        .get('/api/chat/sessions')
        .expect(500);

      expect(response.body).toEqual({ error: 'Failed to fetch chat sessions' });
    });
  });

  describe('POST /api/chat/sessions', () => {
    test('should create a new chat session', async () => {
      const newSession = {
        title: 'New Session',
        messages: []
      };

      const createdSession = {
        _id: 'new-session-id',
        ...newSession,
        createdAt: new Date(),
        updatedAt: new Date()
      };

      mockChatSessions.insertOne.mockResolvedValue(createdSession);

      const response = await request(app)
        .post('/api/chat/sessions')
        .send(newSession)
        .expect(201);

      expect(response.body).toEqual({
        sessionId: 'new-session-id',
        session: createdSession
      });
    });

    test('should create session with default empty messages', async () => {
      const newSession = {
        title: 'Session without messages'
      };

      const createdSession = {
        _id: 'new-session-id',
        title: 'Session without messages',
        messages: [],
        createdAt: new Date(),
        updatedAt: new Date()
      };

      mockChatSessions.insertOne.mockResolvedValue(createdSession);

      await request(app)
        .post('/api/chat/sessions')
        .send(newSession)
        .expect(201);

      expect(mockChatSessions.insertOne).toHaveBeenCalledWith(
        expect.objectContaining({
          title: 'Session without messages',
          messages: [],
          updatedAt: expect.any(Date)
        })
      );
    });
  });

  describe('POST /api/chat/sessions/:sessionId/messages', () => {
    test('should add a message to existing session', async () => {
      const existingSession = {
        _id: 'session-1',
        title: 'Test Session',
        messages: [],
        createdAt: new Date(),
        updatedAt: new Date()
      };

      const updatedSession = {
        ...existingSession,
        messages: [
          {
            id: expect.any(String),
            content: 'Hello world',
            sender: 'user',
            timestamp: expect.any(Date)
          }
        ],
        updatedAt: expect.any(Date)
      };

      mockChatSessions.findOne.mockResolvedValue(existingSession);
      mockChatSessions.updateOne.mockResolvedValue(updatedSession);

      const response = await request(app)
        .post('/api/chat/sessions/session-1/messages')
        .send({
          content: 'Hello world',
          sender: 'user'
        })
        .expect(200);

      expect(response.body.message).toEqual({
        id: expect.any(String),
        content: 'Hello world',
        sender: 'user',
        timestamp: expect.any(String)
      });
    });

    test('should validate message data', async () => {
      await request(app)
        .post('/api/chat/sessions/session-1/messages')
        .send({
          content: 'Hello world'
          // Missing sender
        })
        .expect(400);

      await request(app)
        .post('/api/chat/sessions/session-1/messages')
        .send({
          sender: 'user'
          // Missing content
        })
        .expect(400);

      await request(app)
        .post('/api/chat/sessions/session-1/messages')
        .send({
          content: 'Hello world',
          sender: 'invalid-sender'
        })
        .expect(400);
    });

    test('should return 404 for non-existent session', async () => {
      mockChatSessions.findOne.mockResolvedValue(null);

      await request(app)
        .post('/api/chat/sessions/non-existent/messages')
        .send({
          content: 'Hello world',
          sender: 'user'
        })
        .expect(404);
    });
  });

  describe('GET /api/chat/search', () => {
    test('should search chat sessions', async () => {
      const mockSessions = [
        {
          _id: 'session-1',
          title: 'JavaScript Tutorial',
          messages: [
            {
              id: 'msg-1',
              content: 'How to use JavaScript?',
              sender: 'user',
              timestamp: new Date()
            }
          ]
        }
      ];

      mockChatSessions.find.mockResolvedValue(mockSessions);

      const response = await request(app)
        .get('/api/chat/search?query=javascript')
        .expect(200);

      expect(response.body.sessions).toHaveLength(1);
      expect(response.body.query).toBe('javascript');
    });

    test('should require search query', async () => {
      await request(app)
        .get('/api/chat/search')
        .expect(400);

      await request(app)
        .get('/api/chat/search?query=')
        .expect(400);
    });

    test('should search in both title and message content', async () => {
      const mockSessions = [
        {
          _id: 'session-1',
          title: 'Python Tutorial',
          messages: [
            {
              id: 'msg-1',
              content: 'How to use Python for data science?',
              sender: 'user',
              timestamp: new Date()
            }
          ]
        },
        {
          _id: 'session-2',
          title: 'Data Science Guide',
          messages: [
            {
              id: 'msg-2',
              content: 'JavaScript is also useful',
              sender: 'ai',
              timestamp: new Date()
            }
          ]
        }
      ];

      mockChatSessions.find.mockResolvedValue(mockSessions);

      const response = await request(app)
        .get('/api/chat/search?query=data')
        .expect(200);

      // Should find both sessions (one by title, one by message content)
      expect(response.body.sessions).toHaveLength(2);
    });
  });

  describe('Error Handling', () => {
    test('should handle database connection errors', async () => {
      mockChatSessions.find.mockRejectedValue(new Error('Connection timeout'));

      await request(app)
        .get('/api/chat/sessions')
        .expect(500);
    });

    test('should handle malformed JSON', async () => {
      await request(app)
        .post('/api/chat/sessions')
        .send('invalid json')
        .set('Content-Type', 'application/json')
        .expect(400);
    });
  });
});
