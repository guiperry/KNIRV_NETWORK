import { describe, it, expect, beforeEach, jest } from '@jest/globals';

const mockAxiosInstance = {
  get: jest.fn(),
  post: jest.fn(),
  delete: jest.fn(),
  put: jest.fn(),
  defaults: {
    headers: { common: {} },
  },
  interceptors: {
    response: { use: jest.fn((fn) => fn) },
    request: { use: jest.fn((fn) => fn) },
  },
};

jest.mock('axios', () => ({
  create: jest.fn<() => typeof mockAxiosInstance>(() => mockAxiosInstance),
  isAxiosError: jest.fn<() => boolean>(() => false),
  default: jest.fn<() => typeof mockAxiosInstance>(() => mockAxiosInstance),
}));

import { KNIRVSERVERClient } from '../KNIRVSERVERClient';

describe('KNIRVSERVERClient', () => {
  let client: KNIRVSERVERClient;

  beforeEach(() => {
    jest.clearAllMocks();
    client = new KNIRVSERVERClient({ baseUrl: 'http://localhost:8082' });
  });

  describe('Initialization', () => {
    it('should create a KNIRVSERVERClient instance', () => {
      expect(client).toBeInstanceOf(KNIRVSERVERClient);
    });

    it('should set default config', () => {
      const config = client.getConfig();
      expect(config.baseUrl).toBe('http://localhost:8082');
      expect(config.timeout).toBe(30000);
      expect(config.retryAttempts).toBe(3);
    });

    it('should accept custom config', () => {
      const customClient = new KNIRVSERVERClient({ baseUrl: 'http://custom:8082', timeout: 60000 });
      const config = customClient.getConfig();
      expect(config.baseUrl).toBe('http://custom:8082');
      expect(config.timeout).toBe(60000);
    });
  });

  describe('Health Check', () => {
    it('should return true when health check succeeds', async () => {
      (mockAxiosInstance.get as any).mockResolvedValue({ status: 200 } as any);

      const result = await client.healthCheck();

      expect(result).toBe(true);
      expect(mockAxiosInstance.get).toHaveBeenCalledWith('/api/dve-nodes/health');
    });

    it('should return false when health check fails', async () => {
      (mockAxiosInstance.get as any).mockRejectedValue(new Error('Connection refused'));

      const result = await client.healthCheck();

      expect(result).toBe(false);
    });
  });

  describe('DVE Methods', () => {
    it('should validate with DVE', async () => {
      const mockResult = {
        success: true,
        score: 0.85,
        passed: true,
        output: 'Validation passed',
        executionTime: 100,
      };
      (mockAxiosInstance.post as any).mockResolvedValue({ data: mockResult } as any);

      const result = await client.validateWithDVE({
        skillCode: 'test_code',
        failureContext: 'Test context',
      });

      expect(result).toEqual(mockResult);
      expect(mockAxiosInstance.post).toHaveBeenCalledWith('/api/dve-nodes/tasks/validate', {
        skillCode: 'test_code',
        failureContext: 'Test context',
      });
    });

    it('should handle DVE validation error gracefully', async () => {
      (mockAxiosInstance.post as any).mockRejectedValue(new Error('Network error'));

      const result = await client.validateWithDVE({
        skillCode: 'test',
        failureContext: 'context',
      });

      expect(result.success).toBe(false);
      expect(result.score).toBe(0.5);
    });

    it('should get DVE tasks', async () => {
      const mockTasks = [
        { id: 'task1', type: 'validation', status: 'pending' },
        { id: 'task2', type: 'skill', status: 'completed' },
      ];
      (mockAxiosInstance.get as any).mockResolvedValue({ data: mockTasks });

      const result = await client.getDVETasks();

      expect(result).toEqual(mockTasks);
      expect(mockAxiosInstance.get).toHaveBeenCalledWith('/api/dve-nodes/tasks', expect.objectContaining({ params: {} }));
    });

    it('should filter DVE tasks by status', async () => {
      (mockAxiosInstance.get as any).mockResolvedValue({ data: [] });

      await client.getDVETasks('pending');

      expect(mockAxiosInstance.get).toHaveBeenCalledWith('/api/dve-nodes/tasks', expect.objectContaining({ params: { status: 'pending' } }));
    });

    it('should get DVE nodes', async () => {
      const mockNodes = [{ id: 'node1', name: 'DVE Node 1', status: 'online' }];
      (mockAxiosInstance.get as any).mockResolvedValue({ data: mockNodes });

      const result = await client.getDVENodes();

      expect(result).toEqual(mockNodes);
    });

    it('should submit validation result', async () => {
      (mockAxiosInstance.post as any).mockResolvedValue({ status: 200 });

      const result = await client.submitValidationResult('task1', {
        success: true,
        score: 0.9,
        passed: true,
        output: 'Test output',
        executionTime: 50,
      });

      expect(result).toBe(true);
    });
  });

  describe('CDE Methods', () => {
    it('should create CDE environment', async () => {
      const mockEnv = { id: 'env1', name: 'Test Environment', status: 'creating' };
      (mockAxiosInstance.post as any).mockResolvedValue({ data: mockEnv });

      const result = await client.createCDEEnvironment('Test Environment', 'general');

      expect(result).toEqual(mockEnv);
      expect(mockAxiosInstance.post).toHaveBeenCalledWith('/cde/environments', {
        name: 'Test Environment',
        environmentType: 'general',
        config: undefined,
      });
    });

    it('should get CDE environments', async () => {
      const mockEnvs = [
        { id: 'env1', name: 'Env 1', status: 'running' },
        { id: 'env2', name: 'Env 2', status: 'stopped' },
      ];
      (mockAxiosInstance.get as any).mockResolvedValue({ data: mockEnvs });

      const result = await client.getCDEEnvironments();

      expect(result).toEqual(mockEnvs);
    });

    it('should get CDE environment by ID', async () => {
      const mockEnv = { id: 'env1', name: 'Test', status: 'running' };
      (mockAxiosInstance.get as any).mockResolvedValue({ data: mockEnv });

      const result = await client.getCDEEnvironment('env1');

      expect(result).toEqual(mockEnv);
      expect(mockAxiosInstance.get).toHaveBeenCalledWith('/cde/environments/env1');
    });

    it('should delete CDE environment', async () => {
      (mockAxiosInstance.delete as any).mockResolvedValue({ status: 200 });

      const result = await client.deleteCDEEnvironment('env1');

      expect(result).toBe(true);
      expect(mockAxiosInstance.delete).toHaveBeenCalledWith('/cde/environments/env1');
    });

    it('should start CDE environment', async () => {
      (mockAxiosInstance.post as any).mockResolvedValue({ status: 200 });

      const result = await client.startCDEEnvironment('env1');

      expect(result).toBe(true);
      expect(mockAxiosInstance.post).toHaveBeenCalledWith('/cde/environments/env1/start');
    });

    it('should stop CDE environment', async () => {
      (mockAxiosInstance.post as any).mockResolvedValue({ status: 200 });

      const result = await client.stopCDEEnvironment('env1');

      expect(result).toBe(true);
      expect(mockAxiosInstance.post).toHaveBeenCalledWith('/cde/environments/env1/stop');
    });

    it('should create CDE session', async () => {
      const mockSession = {
        id: 'sess1',
        environmentId: 'env1',
        status: 'active',
        connectionType: 'ssh',
      };
      (mockAxiosInstance.post as any).mockResolvedValue({ data: mockSession });

      const result = await client.createCDESession('env1', 'ssh');

      expect(result).toEqual(mockSession);
    });

    it('should get CDE sessions', async () => {
      const mockSessions = [{ id: 'sess1', status: 'active' }];
      (mockAxiosInstance.get as any).mockResolvedValue({ data: mockSessions });

      const result = await client.getCDESessions();

      expect(result).toEqual(mockSessions);
    });

    it('should validate with CDE', async () => {
      const mockResult = {
        success: true,
        output: 'Code executed',
        executionTime: 150,
        constraintsSatisfied: true,
      };
      (mockAxiosInstance.post as any).mockResolvedValue({ data: mockResult });

      const result = await client.validateWithCDE({
        code: 'console.log("test")',
        language: 'javascript',
      });

      expect(result).toEqual(mockResult);
    });

    it('should handle CDE validation error gracefully', async () => {
      (mockAxiosInstance.post as any).mockRejectedValue(new Error('Network error'));

      const result = await client.validateWithCDE({
        code: 'test',
        language: 'javascript',
      });

      expect(result.success).toBe(true);
      expect(result.constraintsSatisfied).toBe(true);
    });
  });

  describe('Cognitive Integration', () => {
    it('should report cognitive metrics', async () => {
      (mockAxiosInstance.post as any).mockResolvedValue({ status: 200 });

      const result = await client.reportCognitiveMetrics({
        confidence: 0.85,
        processingTime: 100,
      });

      expect(result).toBe(true);
    });

    it('should get cognitive state', async () => {
      const mockState = { learningRate: 0.01, patterns: [] };
      (mockAxiosInstance.get as any).mockResolvedValue({ data: mockState });

      const result = await client.getCognitiveState();

      expect(result).toEqual(mockState);
    });

    it('should submit guardrail violation', async () => {
      (mockAxiosInstance.post as any).mockResolvedValue({ status: 200 });

      const result = await client.submitGuardrailViolation({
        type: 'prompt_injection',
        severity: 'high',
      });

      expect(result).toBe(true);
    });
  });

  describe('Config Updates', () => {
    it('should update config', () => {
      client.updateConfig({ timeout: 60000, apiKey: 'test-key' });

      const config = client.getConfig();
      expect(config.timeout).toBe(60000);
      expect(config.apiKey).toBe('test-key');
    });
  });

  describe('Connection Status', () => {
    it('should report initial connection status', () => {
      const isConnected = client.isServerConnected();
      expect(typeof isConnected).toBe('boolean');
    });
  });
});
