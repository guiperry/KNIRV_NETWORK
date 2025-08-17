// agent-tunnel-registry/tests/uri-routes.test.js
const request = require('supertest');
const express = require('express');
const axios = require('axios');
const MockAdapter = require('axios-mock-adapter');
const uriRoutes = require('../api/uriRoutes');
const registryManager = require('../registry/registryManager');
const config = require('../config');

// Create a mock for axios
const mockAxios = new MockAdapter(axios);

// Create an Express app for testing
const app = express();
app.use(express.json());
app.use('/api/uri', uriRoutes);

describe('URI Routes', () => {
  beforeEach(() => {
    // Clear all mocks before each test
    mockAxios.reset();
    
    // Clear the registry state
    registryManager.nodes.clear();
    registryManager.chainIdToPeerId.clear();
    registryManager.tunneledResourceIds.clear();
    
    // Mock the UUID generation to return a predictable value
    jest.spyOn(require('uuid'), 'v4').mockReturnValue('test-uuid-123');
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  describe('POST /api/uri/generate', () => {
    it('should generate a URI for a tunneled node', async () => {
      // Register a tunneled node
      registryManager.registerNodeViaControlSocket(
        'QmTestPeerId123',
        'test-chain-1',
        '10.0.0.1',
        4001,
        'dev',
        'socket-123',
        'relay.example.com',
        4000
      );

      // Mock the ID existence check
      mockAxios.onGet(`http://localhost:${config.goInternalApiPort}/internal/db/idExists?id=test-uuid-123`).reply(200, {
        exists: false
      });

      // Mock the DHT announcement
      mockAxios.onPost(`http://localhost:${config.goInternalApiPort}/internal/dht/announceResource`).reply(200, {
        status: 'success',
        message: 'Resource announced on DHT'
      });

      // Make the request
      const response = await request(app)
        .post('/api/uri/generate')
        .send({
          devId: 'QmTestPeerId123',
          resourceType: 'chain',
          subPath: 'test/path'
        });

      // Verify the response
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('uri');
      expect(response.body.uri).toBe(`agent://${config.serverPublicHost}/test-uuid-123.chain/test/path`);
      expect(response.body).toHaveProperty('resourceId', 'test-uuid-123');
      expect(response.body).toHaveProperty('resourceType', 'chain');
      expect(response.body).toHaveProperty('subPath', 'test/path');
      
      // Verify the resource was mapped
      expect(registryManager.getPeerIdForTunneledResource('test-uuid-123')).toBe('QmTestPeerId123');
      
      // Verify the DHT announcement was made
      expect(mockAxios.history.post.length).toBe(1);
      expect(mockAxios.history.post[0].url).toBe(`http://localhost:${config.goInternalApiPort}/internal/dht/announceResource`);
    });

    it('should generate a URI for a direct node', async () => {
      // Register a direct node
      registryManager.registerNodeViaApi(
        'QmTestPeerId456',
        'test-chain-2',
        '192.168.1.1',
        4001,
        'dev'
      );

      // Mock the ID existence check
      mockAxios.onGet(`http://localhost:${config.goInternalApiPort}/internal/db/idExists?id=test-uuid-123`).reply(200, {
        exists: false
      });

      // Mock the DHT announcement
      mockAxios.onPost(`http://localhost:${config.goInternalApiPort}/internal/dht/announceResource`).reply(200, {
        status: 'success',
        message: 'Resource announced on DHT'
      });

      // Make the request
      const response = await request(app)
        .post('/api/uri/generate')
        .send({
          devId: 'QmTestPeerId456',
          resourceType: 'dev'
        });

      // Verify the response
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('uri');
      expect(response.body.uri).toBe(`agent://192.168.1.1/test-uuid-123.dev`);
      expect(response.body).toHaveProperty('directInfo');
      expect(response.body.directInfo).toHaveProperty('ip', '192.168.1.1');
      expect(response.body.directInfo).toHaveProperty('port', 4001);
    });

    it('should return 400 if devId is missing', async () => {
      const response = await request(app)
        .post('/api/uri/generate')
        .send({
          resourceType: 'chain'
        });

      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('error');
    });

    it('should return 404 if node is not registered', async () => {
      const response = await request(app)
        .post('/api/uri/generate')
        .send({
          devId: 'non-existent-id',
          resourceType: 'chain'
        });

      expect(response.status).toBe(404);
      expect(response.body).toHaveProperty('error');
    });

    it('should retry if the generated ID already exists', async () => {
      // Register a node
      registryManager.registerNodeViaApi(
        'QmTestPeerId789',
        'test-chain-3',
        '192.168.1.2',
        4002,
        'dev'
      );

      // Mock the ID existence check to return true first, then false
      mockAxios.onGet(`http://localhost:${config.goInternalApiPort}/internal/db/idExists?id=test-uuid-123`).replyOnce(200, {
        exists: true
      }).onGet(`http://localhost:${config.goInternalApiPort}/internal/db/idExists?id=test-uuid-123`).reply(200, {
        exists: false
      });

      // Mock the DHT announcement
      mockAxios.onPost(`http://localhost:${config.goInternalApiPort}/internal/dht/announceResource`).reply(200, {
        status: 'success',
        message: 'Resource announced on DHT'
      });

      // Make the request
      const response = await request(app)
        .post('/api/uri/generate')
        .send({
          devId: 'QmTestPeerId789',
          resourceType: 'chain'
        });

      // Verify the response
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('uri');
    });
  });

  describe('GET /api/uri/resolve', () => {
    it('should resolve a tunneled resource URI', async () => {
      // Register a tunneled node
      registryManager.registerNodeViaControlSocket(
        'QmTestPeerId123',
        'test-chain-1',
        '10.0.0.1',
        4001,
        'dev',
        'socket-123',
        'relay.example.com',
        4000
      );

      // Map a tunneled resource
      registryManager.mapTunneledResource('resource-123', 'QmTestPeerId123');

      // Make the request
      const response = await request(app)
        .get('/api/uri/resolve')
        .query({ uri: 'agent://relay.example.com/resource-123.chain/test/path' });

      // Verify the response
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('connectionType', 'TUNNELED');
      expect(response.body).toHaveProperty('targetPeerId', 'QmTestPeerId123');
      expect(response.body).toHaveProperty('tunnelServerHost', config.serverPublicHost);
      expect(response.body).toHaveProperty('tunnelServerPort', config.publicRelayPort);
      expect(response.body).toHaveProperty('originalUri', 'agent://relay.example.com/resource-123.chain/test/path');
      expect(response.body).toHaveProperty('resolvedIdentifier', 'resource-123');
      expect(response.body).toHaveProperty('resourceType', 'chain');
      expect(response.body).toHaveProperty('subPathWithQuery', '/test/path');
      expect(response.body).toHaveProperty('authority', 'relay.example.com');
    });

    it('should resolve a direct node URI by peer ID', async () => {
      // Register a direct node
      registryManager.registerNodeViaApi(
        'QmTestPeerId456',
        'test-chain-2',
        '192.168.1.1',
        4001,
        'dev'
      );

      // Make the request
      const response = await request(app)
        .get('/api/uri/resolve')
        .query({ uri: 'agent://example.com/QmTestPeerId456.dev' });

      // Verify the response
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('connectionType', 'DIRECT_P2P');
      expect(response.body).toHaveProperty('targetPeerId', 'QmTestPeerId456');
      expect(response.body).toHaveProperty('multiaddress', '/ip4/192.168.1.1/tcp/4001/p2p/QmTestPeerId456');
      expect(response.body).toHaveProperty('originalUri', 'agent://example.com/QmTestPeerId456.dev');
      expect(response.body).toHaveProperty('resolvedIdentifier', 'QmTestPeerId456');
      expect(response.body).toHaveProperty('resourceType', 'dev');
      expect(response.body).toHaveProperty('authority', 'example.com');
    });

    it('should resolve a direct node URI by chain ID', async () => {
      // Register a direct node
      registryManager.registerNodeViaApi(
        'QmTestPeerId789',
        'test-chain-3',
        '192.168.1.2',
        4002,
        'dev'
      );

      // Make the request
      const response = await request(app)
        .get('/api/uri/resolve')
        .query({ uri: 'agent://example.com/test-chain-3.chain' });

      // Verify the response
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('connectionType', 'DIRECT_P2P');
      expect(response.body).toHaveProperty('targetPeerId', 'QmTestPeerId789');
      expect(response.body).toHaveProperty('multiaddress', '/ip4/192.168.1.2/tcp/4002/p2p/QmTestPeerId789');
      expect(response.body).toHaveProperty('originalUri', 'agent://example.com/test-chain-3.chain');
      expect(response.body).toHaveProperty('resolvedIdentifier', 'test-chain-3');
      expect(response.body).toHaveProperty('resourceType', 'chain');
      expect(response.body).toHaveProperty('authority', 'example.com');
    });

    it('should resolve a resource URI using DHT lookup', async () => {
      // Mock the DHT lookup
      mockAxios.onGet(`http://localhost:${config.goInternalApiPort}/internal/dht/findResource?id=resource-456&type=chain`).reply(200, [
        {
          ID: 'QmTestPeerId456',
          Addrs: ['/ip4/192.168.1.1/tcp/4001/p2p/QmTestPeerId456']
        }
      ]);

      // Make the request
      const response = await request(app)
        .get('/api/uri/resolve')
        .query({ uri: 'agent://example.com/resource-456.chain' });

      // Verify the response
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('connectionType', 'DIRECT_P2P');
      expect(response.body).toHaveProperty('targetPeerId', 'QmTestPeerId456');
      expect(response.body).toHaveProperty('multiaddress', '/ip4/192.168.1.1/tcp/4001/p2p/QmTestPeerId456');
      expect(response.body).toHaveProperty('originalUri', 'agent://example.com/resource-456.chain');
      expect(response.body).toHaveProperty('resolvedIdentifier', 'resource-456');
      expect(response.body).toHaveProperty('resourceType', 'chain');
      expect(response.body).toHaveProperty('authority', 'example.com');
    });

    it('should resolve a capability URI using blockchain lookup', async () => {
      // Mock the DHT lookup to return no results
      mockAxios.onGet(`http://localhost:${config.goInternalApiPort}/internal/dht/findResource?id=capability-123&type=capability`).reply(200, []);

      // Mock the blockchain lookup
      mockAxios.onGet(`http://localhost:${config.goInternalApiPort}/internal/db/getCapability?id=capability-123`).reply(200, {
        ID: 'capability-123',
        ProviderID: 'QmTestPeerId789'
      });

      // Mock the provider DHT lookup
      mockAxios.onGet(`http://localhost:${config.goInternalApiPort}/internal/dht/findResource?id=QmTestPeerId789&type=dev`).reply(200, [
        {
          ID: 'QmTestPeerId789',
          Addrs: ['/ip4/192.168.1.2/tcp/4002/p2p/QmTestPeerId789']
        }
      ]);

      // Make the request
      const response = await request(app)
        .get('/api/uri/resolve')
        .query({ uri: 'agent://example.com/capability-123.capability' });

      // Verify the response
      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('connectionType', 'DIRECT_P2P');
      expect(response.body).toHaveProperty('targetPeerId', 'QmTestPeerId789');
      expect(response.body).toHaveProperty('multiaddress', '/ip4/192.168.1.2/tcp/4002/p2p/QmTestPeerId789');
      expect(response.body).toHaveProperty('originalUri', 'agent://example.com/capability-123.capability');
      expect(response.body).toHaveProperty('resolvedIdentifier', 'capability-123');
      expect(response.body).toHaveProperty('resourceType', 'capability');
      expect(response.body).toHaveProperty('authority', 'example.com');
    });

    it('should return 400 if URI is missing', async () => {
      const response = await request(app)
        .get('/api/uri/resolve');

      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('error');
    });

    it('should return 400 if URI scheme is invalid', async () => {
      const response = await request(app)
        .get('/api/uri/resolve')
        .query({ uri: 'http://example.com/resource-123.chain' });

      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('error');
    });

    it('should return 400 if URI format is invalid', async () => {
      const response = await request(app)
        .get('/api/uri/resolve')
        .query({ uri: 'agent://example.com' });

      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('error');
    });

    it('should return 400 if resource type is missing', async () => {
      const response = await request(app)
        .get('/api/uri/resolve')
        .query({ uri: 'agent://example.com/resource-123' });

      expect(response.status).toBe(400);
      expect(response.body).toHaveProperty('error');
    });

    it('should return 404 if resource is not found', async () => {
      // Mock the DHT lookup to return no results
      mockAxios.onGet(`http://localhost:${config.goInternalApiPort}/internal/dht/findResource?id=non-existent&type=chain`).reply(200, []);

      // Mock the blockchain lookup to return no results
      mockAxios.onGet(`http://localhost:${config.goInternalApiPort}/internal/db/getCapability?id=non-existent`).reply(404, {
        error: 'Capability not found'
      });

      const response = await request(app)
        .get('/api/uri/resolve')
        .query({ uri: 'agent://example.com/non-existent.chain' });

      expect(response.status).toBe(404);
      expect(response.body).toHaveProperty('error');
    });
  });
});