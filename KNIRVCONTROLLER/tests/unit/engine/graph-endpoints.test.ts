import request from 'supertest';
import app from '../server/api-server';

// Mock apiKeyService to bypass authentication
jest.mock('../services/ApiKeyService', () => ({
  apiKeyService: {
    validateApiKey: jest.fn(async (key: string) => ({ id: 'testkey', key, permissions: ['write:graph', 'admin:all'], isActive: true })),
    checkRateLimit: jest.fn(async () => ({ allowed: true })),
    recordUsage: jest.fn(async () => {}),
    hasPermission: jest.fn((apiKey: { permissions: string[] }, perm: string) => apiKey.permissions.includes(perm)),
    getAvailablePermissions: jest.fn(() => ['write:graph'])
  }
}));

// Mock rxdbService and PersonalKNIRVGRAPHService
jest.mock('../services/RxDBService', () => ({
  rxdbService: {
    isDatabaseInitialized: jest.fn(() => true),
    getDatabase: jest.fn(() => ({
      graphs: {
        findOne: jest.fn(() => ({ exec: async () => null })),
        upsert: jest.fn(async () => ({}))
      }
    }))
  }
}));

jest.mock('../services/PersonalKNIRVGRAPHService', () => ({
  personalKNIRVGRAPHService: {
    addErrorNode: jest.fn(async (data: Record<string, unknown>) => ({ id: `error_${(data as { errorId: string }).errorId}`, ...data })),
    addContextNode: jest.fn(async (data: Record<string, unknown>) => ({ id: `capability_${(data as { contextId: string }).contextId}`, ...data })),
    addIdeaNode: jest.fn(async (data: Record<string, unknown>) => ({ id: `property_${(data as { ideaId: string }).ideaId}`, ...data }))
  }
}));
// Note: Use the real PersonalKNIRVGRAPHService with rxdbService mocked above to reduce over-mocking

describe('Graph endpoints', () => {
  it('creates an error node', async () => {
    const res = await request(app)
      .post('/api/graph/error')
      .set('X-API-Key', 'test')
      .send({ errorId: 'e1', description: 'Test error', context: { log: 'err' } });

    expect(res.status).toBe(200);
    expect(res.body).toHaveProperty('node');
    expect(res.body.node.id).toBe('error_e1');
  });

  it('creates a context node', async () => {
    const res = await request(app)
      .post('/api/graph/context')
      .set('X-API-Key', 'test')
      .send({ contextId: 'c1', contextName: 'MCP Server', mcpServerInfo: { cap: true } });

    expect(res.status).toBe(200);
    expect(res.body.node.id).toBe('capability_c1');
  });

  it('creates an idea node', async () => {
    const res = await request(app)
      .post('/api/graph/idea')
      .set('X-API-Key', 'test')
      .send({ ideaId: 'i1', ideaName: 'New Property', description: 'An idea' });

    expect(res.status).toBe(200);
    expect(res.body.node.id).toBe('property_i1');
  });
});
