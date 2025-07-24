// agent-tunnel-registry/tests/registry-manager.test.js
const axios = require('axios');
const MockAdapter = require('axios-mock-adapter');
const registryManager = require('../registry/registryManager');
const config = require('../config');

// Create a mock for axios
const mockAxios = new MockAdapter(axios);

describe('Registry Manager', () => {
  beforeEach(() => {
    // Clear all mocks before each test
    mockAxios.reset();
    
    // Clear the registry state
    registryManager.nodes.clear();
    registryManager.chainIdToPeerId.clear();
    registryManager.tunneledResourceIds.clear();
  });

  describe('registerNodeViaApi', () => {
    it('should register a node and announce it on the DHT', () => {
      // Mock the DHT announcement endpoint
      mockAxios.onPost(`http://localhost:${config.goInternalApiPort}/internal/dht/announceResource`).reply(200, {
        status: 'success',
        message: 'Resource announced on DHT'
      });

      // Test data
      const devId = 'QmTestPeerId123';
      const chainId = 'test-chain-1';
      const publicIp = '192.168.1.1';
      const publicP2pPort = 4001;
      const type = 'dev';

      // Register the node
      const nodeInfo = registryManager.registerNodeViaApi(devId, chainId, publicIp, publicP2pPort, type);

      // Verify the node was registered
      expect(registryManager.nodes.has(devId)).toBe(true);
      expect(registryManager.chainIdToPeerId.get(chainId)).toBe(devId);
      
      // Verify the node info
      expect(nodeInfo.devId).toBe(devId);
      expect(nodeInfo.chainId).toBe(chainId);
      expect(nodeInfo.publicIp).toBe(publicIp);
      expect(nodeInfo.publicP2pPort).toBe(publicP2pPort);
      expect(nodeInfo.type).toBe(type);
      expect(nodeInfo.isTunneled).toBe(false);
      
      // Verify the DHT announcement was made
      expect(mockAxios.history.post.length).toBe(1);
      expect(mockAxios.history.post[0].url).toBe(`http://localhost:${config.goInternalApiPort}/internal/dht/announceResource`);
      
      const requestData = JSON.parse(mockAxios.history.post[0].data);
      expect(requestData.id).toBe(devId);
      expect(requestData.type).toBe(type);
      expect(requestData.multiaddress).toBe(`/ip4/${publicIp}/tcp/${publicP2pPort}/p2p/${devId}`);
    });

    it('should use devId as chainId if chainId is not provided', () => {
      // Mock the DHT announcement endpoint
      mockAxios.onPost(`http://localhost:${config.goInternalApiPort}/internal/dht/announceResource`).reply(200, {
        status: 'success',
        message: 'Resource announced on DHT'
      });

      // Test data
      const devId = 'QmTestPeerId456';
      const publicIp = '192.168.1.2';
      const publicP2pPort = 4002;
      const type = 'dev';

      // Register the node without a chainId
      const nodeInfo = registryManager.registerNodeViaApi(devId, null, publicIp, publicP2pPort, type);

      // Verify the node was registered
      expect(registryManager.nodes.has(devId)).toBe(true);
      expect(registryManager.chainIdToPeerId.get(devId)).toBe(devId);
      
      // Verify the node info
      expect(nodeInfo.devId).toBe(devId);
      expect(nodeInfo.chainId).toBe(devId); // chainId should be set to devId
    });

    it('should handle DHT announcement failure gracefully', () => {
      // Mock the DHT announcement endpoint to fail
      mockAxios.onPost(`http://localhost:${config.goInternalApiPort}/internal/dht/announceResource`).reply(500, {
        error: 'Internal server error'
      });

      // Test data
      const devId = 'QmTestPeerId789';
      const chainId = 'test-chain-2';
      const publicIp = '192.168.1.3';
      const publicP2pPort = 4003;
      const type = 'dev';

      // Register the node
      const nodeInfo = registryManager.registerNodeViaApi(devId, chainId, publicIp, publicP2pPort, type);

      // Verify the node was still registered locally despite DHT announcement failure
      expect(registryManager.nodes.has(devId)).toBe(true);
      expect(registryManager.chainIdToPeerId.get(chainId)).toBe(devId);
    });
  });

  describe('registerNodeViaControlSocket', () => {
    it('should register a tunneled node', () => {
      // Test data
      const devId = 'QmTestPeerId123';
      const chainId = 'test-chain-1';
      const internalIp = '10.0.0.1';
      const internalP2pPort = 4001;
      const type = 'dev';
      const controlSocketId = 'socket-123';
      const serverPublicHost = 'relay.example.com';
      const publicRelayPort = 4000;

      // Register the node
      const nodeInfo = registryManager.registerNodeViaControlSocket(
        devId, chainId, internalIp, internalP2pPort, type, controlSocketId, serverPublicHost, publicRelayPort
      );

      // Verify the node was registered
      expect(registryManager.nodes.has(devId)).toBe(true);
      expect(registryManager.chainIdToPeerId.get(chainId)).toBe(devId);
      
      // Verify the node info
      expect(nodeInfo.devId).toBe(devId);
      expect(nodeInfo.chainId).toBe(chainId);
      expect(nodeInfo.internalIp).toBe(internalIp);
      expect(nodeInfo.internalP2pPort).toBe(internalP2pPort);
      expect(nodeInfo.type).toBe(type);
      expect(nodeInfo.controlSocketId).toBe(controlSocketId);
      expect(nodeInfo.isTunneled).toBe(true);
      expect(nodeInfo.publicRelayUrl).toBe(`tcp://${serverPublicHost}:${publicRelayPort}/p2p_tunnel/${devId}`);
    });
  });

  describe('mapTunneledResource and getPeerIdForTunneledResource', () => {
    it('should map and retrieve a tunneled resource ID', () => {
      // Test data
      const uniqueId = 'resource-123';
      const devId = 'QmTestPeerId123';

      // Map the resource
      registryManager.mapTunneledResource(uniqueId, devId);

      // Verify the mapping
      expect(registryManager.tunneledResourceIds.get(uniqueId)).toBe(devId);
      
      // Retrieve the mapping
      const retrievedDevId = registryManager.getPeerIdForTunneledResource(uniqueId);
      expect(retrievedDevId).toBe(devId);
    });

    it('should return undefined for non-existent resource ID', () => {
      // Retrieve a non-existent mapping
      const retrievedDevId = registryManager.getPeerIdForTunneledResource('non-existent-id');
      expect(retrievedDevId).toBeUndefined();
    });
  });

  describe('getNodeByPeerId and getNodeByChainId', () => {
    beforeEach(() => {
      // Register a test node
      registryManager.registerNodeViaApi(
        'QmTestPeerId123',
        'test-chain-1',
        '192.168.1.1',
        4001,
        'dev'
      );
    });

    it('should retrieve a node by peer ID', () => {
      const nodeInfo = registryManager.getNodeByPeerId('QmTestPeerId123');
      expect(nodeInfo).toBeDefined();
      expect(nodeInfo.devId).toBe('QmTestPeerId123');
      expect(nodeInfo.chainId).toBe('test-chain-1');
    });

    it('should retrieve a node by chain ID', () => {
      const nodeInfo = registryManager.getNodeByChainId('test-chain-1');
      expect(nodeInfo).toBeDefined();
      expect(nodeInfo.devId).toBe('QmTestPeerId123');
      expect(nodeInfo.chainId).toBe('test-chain-1');
    });

    it('should return undefined for non-existent peer ID', () => {
      const nodeInfo = registryManager.getNodeByPeerId('non-existent-id');
      expect(nodeInfo).toBeUndefined();
    });

    it('should return undefined for non-existent chain ID', () => {
      const nodeInfo = registryManager.getNodeByChainId('non-existent-chain');
      expect(nodeInfo).toBeUndefined();
    });
  });

  describe('deregisterNodeByControlSocket', () => {
    beforeEach(() => {
      // Register a test node via control socket
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
    });

    it('should deregister a node by control socket ID', () => {
      // Verify the node exists
      expect(registryManager.nodes.has('QmTestPeerId123')).toBe(true);
      expect(registryManager.chainIdToPeerId.get('test-chain-1')).toBe('QmTestPeerId123');

      // Deregister the node
      registryManager.deregisterNodeByControlSocket('socket-123');

      // Verify the node was deregistered
      expect(registryManager.nodes.has('QmTestPeerId123')).toBe(false);
      expect(registryManager.chainIdToPeerId.get('test-chain-1')).toBeUndefined();
    });

    it('should not deregister a node with a different control socket ID', () => {
      // Verify the node exists
      expect(registryManager.nodes.has('QmTestPeerId123')).toBe(true);

      // Try to deregister with a different socket ID
      registryManager.deregisterNodeByControlSocket('different-socket-id');

      // Verify the node still exists
      expect(registryManager.nodes.has('QmTestPeerId123')).toBe(true);
    });
  });

  describe('getAllNodes', () => {
    beforeEach(() => {
      // Register multiple test nodes
      registryManager.registerNodeViaApi(
        'QmTestPeerId1',
        'test-chain-1',
        '192.168.1.1',
        4001,
        'dev'
      );
      registryManager.registerNodeViaApi(
        'QmTestPeerId2',
        'test-chain-2',
        '192.168.1.2',
        4002,
        'dev'
      );
    });

    it('should return all registered nodes', () => {
      const nodes = registryManager.getAllNodes();
      expect(nodes.length).toBe(2);
      
      // Verify the nodes are in the result
      const nodeIds = nodes.map(node => node.devId);
      expect(nodeIds).toContain('QmTestPeerId1');
      expect(nodeIds).toContain('QmTestPeerId2');
    });

    it('should return an empty array when no nodes are registered', () => {
      // Clear all nodes
      registryManager.nodes.clear();
      
      const nodes = registryManager.getAllNodes();
      expect(nodes.length).toBe(0);
    });
  });
});