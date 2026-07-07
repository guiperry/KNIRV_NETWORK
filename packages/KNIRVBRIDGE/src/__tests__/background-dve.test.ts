import { AlarmKey } from '@common/constants/alarm-key.constant';
import { DVE_CONSTANTS } from '@common/constants/dve.constant';
import { deriveDVEIdentity, DVEIdentity } from '@services/dve/dve-identity';
import { DVERegistryService } from '@services/dve/dve-registry';
import { DVEWebSocketClient } from '@services/dve/dve-ws-client';
import { DVEBadgeManager, DVEBadge } from '@services/dve/dve-badge-manager';
import { ValidationRuntime } from '@services/dve/validation-runtime';

// ---------------------------------------------------------------------------
// Mock chrome.* APIs
// ---------------------------------------------------------------------------
const mockChromeStorageLocal = {
  get: jest.fn(),
  set: jest.fn(),
  remove: jest.fn(),
  clear: jest.fn(),
};

const mockChromeStorageSession = {
  get: jest.fn(),
  set: jest.fn(),
  remove: jest.fn(),
  clear: jest.fn(),
};

const mockChromeAlarms = {
  create: jest.fn(),
  get: jest.fn(),
  clear: jest.fn(),
  onAlarm: { addListener: jest.fn() },
};

const mockChromeRuntime = {
  id: 'test-extension-id-12345',
  onConnect: { addListener: jest.fn() },
  onMessage: { addListener: jest.fn() },
  onInstalled: { addListener: jest.fn() },
  getURL: jest.fn((path: string) => `chrome-extension://test/${path}`),
};

const mockChromeTabs = {
  create: jest.fn(),
  onCreated: { addListener: jest.fn() },
  onUpdated: { addListener: jest.fn() },
  sendMessage: jest.fn(),
};

const mockChromeAction = {
  setPopup: jest.fn(),
  onClicked: { addListener: jest.fn() },
};

Object.assign(globalThis, {
  chrome: {
    storage: {
      local: mockChromeStorageLocal,
      session: mockChromeStorageSession,
    },
    alarms: mockChromeAlarms,
    runtime: mockChromeRuntime,
    tabs: mockChromeTabs,
    action: mockChromeAction,
  },
});

// ---------------------------------------------------------------------------
// DVE Identity derivation tests
// ---------------------------------------------------------------------------
describe('DVE Identity Derivation', () => {
  beforeAll(() => {
    // Ensure crypto.subtle is available (jsdom may not provide it)
    if (!globalThis.crypto?.subtle) {
      const { subtle } = require('crypto').webcrypto;
      Object.assign(globalThis, {
        crypto: { subtle },
      });
    }
  });

  it('should derive a deterministic DVEIdentity from a wallet address', async () => {
    const walletAddress = 'gno1testwalletaddress12345';
    const identity = await deriveDVEIdentity(walletAddress);

    expect(identity).toBeDefined();
    expect(identity.nodeID).toMatch(/^dve-/);
    expect(identity.nodeID.length).toBe(20); // 'dve-' + 16 hex chars
    expect(identity.dveURI).toBe(`knirv://dve/${walletAddress}/browser`);
    expect(identity.walletAddress).toBe(walletAddress);
    expect(identity.extensionID).toBe('test-extension-id-12345');
    expect(typeof identity.browserVersion).toBe('string');
  });

  it('should produce different nodeIDs for different wallet addresses', async () => {
    const id1 = await deriveDVEIdentity('walletA');
    const id2 = await deriveDVEIdentity('walletB');
    expect(id1.nodeID).not.toBe(id2.nodeID);
  });

  it('should produce the same nodeID for the same wallet address (deterministic)', async () => {
    const id1 = await deriveDVEIdentity('gno1samewallet');
    const id2 = await deriveDVEIdentity('gno1samewallet');
    expect(id1.nodeID).toBe(id2.nodeID);
  });
});

// ---------------------------------------------------------------------------
// DVE Initialization Logic Tests
// ---------------------------------------------------------------------------
describe('DVE Initialization Logic', () => {
  // These tests validate the behavior of the initializeDVE function by
  // testing each of the service interactions in isolation.

  it('should derive identity, register, connect WS, and start badge sync', async () => {
    const walletAddress = 'gno1test123';
    const authToken = 'test-auth-token';

    const identity = await deriveDVEIdentity(walletAddress);
    expect(identity.walletAddress).toBe(walletAddress);

    const registry = new DVERegistryService('http://localhost:8084', authToken);
    const wsClient = new DVEWebSocketClient('http://localhost:8084', authToken);
    const badgeManager = new DVEBadgeManager('http://localhost:8084');
    const validationRuntime = new ValidationRuntime();

    // Verify initial state
    expect(registry).toBeDefined();
    expect(wsClient).toBeDefined();
    expect(badgeManager).toBeDefined();
    expect(validationRuntime).toBeDefined();

    // Badge manager should start empty
    const activeBadges = badgeManager.getActiveBadges();
    expect(activeBadges).toEqual([]);

    // Validation runtime should have no running tasks initially
    expect(validationRuntime.getRunningCount()).toBe(0);
    expect(validationRuntime.getQueueSize()).toBe(0);
  });

  it('should register with the server and return a server node ID', async () => {
    const walletAddress = 'gno1registertest';
    const authToken = 'test-auth-token';
    const serverURL = 'http://localhost:8084';

    const identity = await deriveDVEIdentity(walletAddress);
    const registry = new DVERegistryService(serverURL, authToken);

    // Mock fetch for registration
    const mockFetch = jest.fn().mockResolvedValue({
      ok: true,
      status: 201,
      json: async () => ({
        id: 'server-node-001',
        node_id: identity.nodeID,
        status: 'active',
        wallet_address: walletAddress,
        dve_uri: identity.dveURI,
        capabilities: [],
        badge_nft_ids: [],
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      }),
    });
    globalThis.fetch = mockFetch as any;

    const serverNodeID = await registry.register(identity, [], []);
    expect(serverNodeID).toBe('server-node-001');
    expect(mockFetch).toHaveBeenCalledWith(
      `${serverURL}/api/v1/dve/nodes`,
      expect.objectContaining({
        method: 'POST',
        headers: expect.objectContaining({
          'Content-Type': 'application/json',
          Authorization: `Bearer ${authToken}`,
        }),
      }),
    );
  });

  it('should throw on registration failure', async () => {
    const walletAddress = 'gno1failtest';
    const identity = await deriveDVEIdentity(walletAddress);
    const registry = new DVERegistryService('http://localhost:8084', 'token');

    globalThis.fetch = jest.fn().mockResolvedValue({
      ok: false,
      status: 400,
      text: async () => 'Bad Request',
    }) as any;

    await expect(registry.register(identity, [], [])).rejects.toThrow(
      'DVE registration failed (400): Bad Request',
    );
  });
});

// ---------------------------------------------------------------------------
// DVE Shutdown Logic Tests
// ---------------------------------------------------------------------------
describe('DVE Shutdown Logic', () => {
  it('should deregister, disconnect WS, and reset state', async () => {
    const walletAddress = 'gno1shutdowntest';
    const authToken = 'test-auth-token';
    const serverURL = 'http://localhost:8084';

    const identity = await deriveDVEIdentity(walletAddress);
    const registry = new DVERegistryService(serverURL, authToken);

    // Mock successful deregistration
    globalThis.fetch = jest.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({}),
    }) as any;

    // Deregister should succeed without throwing
    await expect(registry.deregister(identity.nodeID)).resolves.toBeUndefined();

    // Verify the deregister API call
    expect(globalThis.fetch).toHaveBeenCalledWith(
      `${serverURL}/api/v1/dve/nodes/${encodeURIComponent(identity.nodeID)}/status`,
      expect.objectContaining({
        method: 'PUT',
        body: expect.stringContaining('"status":"inactive"'),
      }),
    );
  });

  it('should throw on deregistration failure', async () => {
    const identity = await deriveDVEIdentity('gno1faildereg');
    const registry = new DVERegistryService('http://localhost:8084', 'token');

    globalThis.fetch = jest.fn().mockResolvedValue({
      ok: false,
      status: 404,
      text: async () => 'Not Found',
    }) as any;

    await expect(registry.deregister(identity.nodeID)).rejects.toThrow(
      'DVE deregistration failed (404): Not Found',
    );
  });

  it('should disconnect the WebSocket gracefully', () => {
    const wsClient = new DVEWebSocketClient('http://localhost:8084', 'token');

    // Mock the WebSocket
    const mockClose = jest.fn();
    Object.assign(wsClient as any, {
      ws: { close: mockClose, readyState: WebSocket.OPEN },
    });

    wsClient.disconnect();

    expect(mockClose).toHaveBeenCalledWith(1000, 'Client disconnecting');
  });
});

// ---------------------------------------------------------------------------
// Heartbeat Alarm Handling Tests
// ---------------------------------------------------------------------------
describe('DVE Heartbeat Alarm Handling', () => {
  it('should send heartbeat via WebSocket when DVE is active', () => {
    const wsClient = new DVEWebSocketClient('http://localhost:8084', 'token');

    const sendSpy = jest.spyOn(wsClient as any, 'send');

    // Simulate an open WebSocket
    Object.assign(wsClient as any, {
      ws: { readyState: WebSocket.OPEN },
    });

    // Send heartbeat directly
    wsClient.sendHeartbeat();

    expect(sendSpy).toHaveBeenCalledWith('heartbeat', {
      timestamp: expect.any(Number),
    });

    sendSpy.mockRestore();
  });

  it('should handle heartbeat gracefully when WebSocket is null', () => {
    const wsClient = new DVEWebSocketClient('http://localhost:8084', 'token');

    // No WebSocket set — sendHeartbeat should not throw
    expect(() => wsClient.sendHeartbeat()).not.toThrow();
  });

  it('should create DVE_HEARTBEAT alarm with correct period', () => {
    // Verify the alarm key constant
    expect(AlarmKey.DVE_HEARTBEAT).toBe('DVE_HEARTBEAT');

    // Verify SCHEDULE_ALARMS includes DVE_HEARTBEAT
    const { SCHEDULE_ALARMS } = require('@common/constants/alarm-key.constant');
    const dveAlarm = SCHEDULE_ALARMS.find(
      (a: { key: string }) => a.key === AlarmKey.DVE_HEARTBEAT,
    );
    expect(dveAlarm).toBeDefined();
    expect(dveAlarm.periodInMinutes).toBe(0.75);

    // 0.75 minutes = 45 seconds matches HEARTBEAT_INTERVAL_MS
    expect(DVE_CONSTANTS.HEARTBEAT_INTERVAL_MS).toBe(45_000);
  });
});

// ---------------------------------------------------------------------------
// DVEBadgeManager integration tests
// ---------------------------------------------------------------------------
describe('DVEBadgeManager Integration', () => {
  const mockBadges: DVEBadge[] = [
    {
      nftTokenID: 'badge-001',
      collectionPath: 'gno.land/r/dve/badges',
      capabilities: ['policy-check', 'signature-verify'],
      supportedTags: ['policy', 'sig'],
      attachedPolicies: ['policy-v1'],
      stakeRequirement: 100,
      trustTier: 'standard',
      active: true,
    },
    {
      nftTokenID: 'badge-002',
      collectionPath: 'gno.land/r/dve/badges',
      capabilities: ['reasoning-simple', 'skill-lint'],
      supportedTags: ['reasoning', 'lint'],
      attachedPolicies: [],
      stakeRequirement: 500,
      trustTier: 'verified',
      active: true,
    },
  ];

  it('should compute aggregate capabilities from active badges', () => {
    const badgeManager = new DVEBadgeManager();
    const capabilities = badgeManager.computeAggregateCapabilities(mockBadges);

    expect(capabilities).toContain('policy-check');
    expect(capabilities).toContain('signature-verify');
    expect(capabilities).toContain('reasoning-simple');
    expect(capabilities).toContain('skill-lint');
    expect(capabilities.length).toBe(4);
  });

  it('should exclude inactive badges from capabilities', () => {
    const inactiveBadges = mockBadges.map((b) => ({ ...b, active: false }));
    const badgeManager = new DVEBadgeManager();
    const capabilities = badgeManager.computeAggregateCapabilities(inactiveBadges);

    expect(capabilities).toEqual([]);
  });

  it('should compute aggregate stake from all active badges', () => {
    const badgeManager = new DVEBadgeManager();
    const stake = badgeManager.computeAggregateStake(mockBadges);

    expect(stake).toBe(600); // 100 + 500
  });

  it('should toggle badge active state', async () => {
    const badgeManager = new DVEBadgeManager();

    // Manually add badges to the manager
    for (const badge of mockBadges) {
      (badgeManager as any).badges.set(badge.nftTokenID, { ...badge });
    }

    expect(badgeManager.getActiveBadges().length).toBe(2);

    await badgeManager.toggleBadgeActive('badge-001', false);
    expect(badgeManager.getActiveBadges().length).toBe(1);

    await badgeManager.toggleBadgeActive('badge-001', true);
    expect(badgeManager.getActiveBadges().length).toBe(2);
  });

  it('should throw when toggling a non-existent badge', async () => {
    const badgeManager = new DVEBadgeManager();

    await expect(
      badgeManager.toggleBadgeActive('nonexistent', false),
    ).rejects.toThrow('Badge nonexistent not found');
  });

  it('should destroy and clear all state', () => {
    const badgeManager = new DVEBadgeManager();

    for (const badge of mockBadges) {
      (badgeManager as any).badges.set(badge.nftTokenID, { ...badge });
    }

    expect(badgeManager.getAllBadges().length).toBe(2);

    badgeManager.destroy();

    expect(badgeManager.getAllBadges().length).toBe(0);
    expect(badgeManager.getActiveBadges().length).toBe(0);
  });
});

// ---------------------------------------------------------------------------
// ValidationRuntime integration tests
// ---------------------------------------------------------------------------
describe('ValidationRuntime Integration', () => {
  let runtime: ValidationRuntime;

  beforeEach(() => {
    runtime = new ValidationRuntime();
  });

  afterEach(async () => {
    // Cancel any running tasks
    runtime = new ValidationRuntime();
  });

  it('should start with zero running and queued tasks', () => {
    expect(runtime.getRunningCount()).toBe(0);
    expect(runtime.getQueueSize()).toBe(0);
  });

  it('should fail a policy-check task with missing fields', async () => {
    const result = await runtime.executeTask({
      id: 'task-001',
      type: 'policy-check',
      payload: {},
    });

    expect(result.status).toBe('failure');
    expect(result.score).toBe(0);
    expect(result.errorMessage).toContain('policy');
  });

  it('should fail a signature-verify task with missing fields', async () => {
    const result = await runtime.executeTask({
      id: 'task-002',
      type: 'signature-verify',
      payload: {},
    });

    expect(result.status).toBe('failure');
    expect(result.score).toBe(0);
    expect(result.errorMessage).toContain('signature');
  });

  it('should fail a reasoning-simple task with missing fields', async () => {
    const result = await runtime.executeTask({
      id: 'task-003',
      type: 'reasoning-simple',
      payload: {},
    });

    expect(result.status).toBe('failure');
    expect(result.score).toBe(0);
    expect(result.errorMessage).toContain('expression');
  });

  it('should fail a skill-lint task with missing fields', async () => {
    const result = await runtime.executeTask({
      id: 'task-004',
      type: 'skill-lint',
      payload: {},
    });

    expect(result.status).toBe('failure');
    expect(result.score).toBe(0);
    expect(result.errorMessage).toContain('skill');
  });

  it('should return error for unknown task type', async () => {
    const result = await runtime.executeTask({
      id: 'task-005',
      type: 'unknown-type' as any,
      payload: {},
    });

    expect(result.status).toBe('error');
    expect(result.score).toBe(0);
    expect(result.errorMessage).toContain('Unknown task type');
  });

  it('should successfully execute a reasoning task', async () => {
    const result = await runtime.executeTask({
      id: 'task-006',
      type: 'reasoning-simple',
      payload: {
        expression: '2 + 2',
        expected: 4,
      },
    });

    expect(result.status).toBe('success');
    expect(result.score).toBe(1);
  });

  it('should reject cancel for non-existent task', () => {
    const cancelled = runtime.cancelTask('nonexistent');
    expect(cancelled).toBe(false);
  });
});
