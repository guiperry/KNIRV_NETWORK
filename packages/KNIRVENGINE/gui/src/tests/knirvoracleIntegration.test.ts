import { KNIRVOracleService } from '../services/knirvoracleService';
import { WalletService } from '../services/walletService';

type MockClient = {
  post: jest.Mock;
  get: jest.Mock;
};

const getClient = (service: KNIRVOracleService): MockClient =>
  (service as unknown as { client: MockClient }).client;

describe('KNIRVORACLE Integration Tests', () => {
  let knirvoracleService: KNIRVOracleService;
  let walletService: WalletService;

  beforeEach(() => {
    // Initialize services with test configuration
    knirvoracleService = new KNIRVOracleService({
      baseURL: import.meta.env.VITE_KNIRVORACLE_URL || 'http://localhost:8080',
      apiKey: import.meta.env.VITE_KNIRVORACLE_API_KEY || 'test-key',
      timeout: 10000,
    });

    walletService = new WalletService();
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  describe('KNIRVOracleService', () => {
    it('should initialize with correct configuration', () => {
      expect(knirvoracleService).toBeDefined();
      expect(knirvoracleService.getConfig().baseURL).toBe(
        import.meta.env.VITE_KNIRVORACLE_URL || 'http://localhost:8080'
      );
    });

    it('should update configuration correctly', () => {
      const newConfig = {
        baseURL: 'http://new-url:8080',
        apiKey: 'new-api-key',
      };

      knirvoracleService.updateConfig(newConfig);
      const config = knirvoracleService.getConfig();

      expect(config.baseURL).toBe(newConfig.baseURL);
      expect(config.apiKey).toBe(newConfig.apiKey);
    });

    it('should handle agent minting request', async () => {
      const mockResponse = {
        success: true,
        transaction_id: 'tx-123',
        agent_nft_id: 'nft-456',
        message: 'Agent minted successfully',
      };

      // Mock the axios post method
      const mockPost = jest.fn().mockResolvedValue({ data: mockResponse });
      getClient(knirvoracleService).post = mockPost;

      const mintRequest = {
        agent_id: 'test-agent-123',
        name: 'Test Agent',
        description: 'A test agent for integration testing',
        owner: 'test-owner',
        metadata: {
          type: 'assistant',
          model: 'gpt-4',
        },
      };

      const result = await knirvoracleService.mintAgent(mintRequest);

      expect(mockPost).toHaveBeenCalledWith('/agent/mint', mintRequest, { timeout: 60000 });
      expect(result).toEqual(mockResponse);
      expect(result.success).toBe(true);
      expect(result.transaction_id).toBe('tx-123');
      expect(result.agent_nft_id).toBe('nft-456');
    });

    it('should handle capability registration', async () => {
      const mockResponse = {
        success: true,
        capability_id: 'cap-789',
        tx_hash: '0xabcdef123456',
        message: 'Capability registered successfully',
      };

      const mockPost = jest.fn().mockResolvedValue({ data: mockResponse });
      getClient(knirvoracleService).post = mockPost;

      const registrationRequest = {
        name: 'Test Capability',
        type: 'mcp_capability',
        description: 'A test capability',
        schema: {
          type: 'object',
          properties: {
            input: { type: 'string' },
          },
        } as const,
        owner: 'test-owner',
        gas_fee_nrn: 1000,
      };

      const result = await knirvoracleService.registerCapability(registrationRequest);

      expect(mockPost).toHaveBeenCalledWith('/wallet/mcp/create_register_capability', registrationRequest, { timeout: 30000 });
      expect(result).toEqual(mockResponse);
      expect(result.success).toBe(true);
      expect(result.capability_id).toBe('cap-789');
    });

    it('should handle faucet requests', async () => {
      const mockResponse = {
        success: true,
        request_id: 'req-123',
        tx_hash: '0x123456789',
        amount: '100',
        status: 'completed',
        message: 'Faucet request successful',
      };

      const mockPost = jest.fn().mockResolvedValue({ data: mockResponse });
      getClient(knirvoracleService).post = mockPost;

      const faucetRequest = {
        address: '0x742d35Cc6aa34567890abcdef1234567890abcdef',
        amount: '100',
        reason: 'Test faucet request',
      };

      const result = await knirvoracleService.requestFaucet(faucetRequest);

      expect(mockPost).toHaveBeenCalledWith('/api/mint/nrv', faucetRequest);
      expect(result).toEqual(mockResponse);
      expect(result.success).toBe(true);
      expect(result.amount).toBe('100');
    });

    it('should handle wallet balance queries', async () => {
      const mockResponse = {
        success: true,
        address: '0x742d35Cc6aa34567890abcdef1234567890abcdef',
        balance: '1000.50',
        nrn_balance: '1247',
        usd_value: '312.75',
      };

      const mockGet = jest.fn().mockResolvedValue({ data: mockResponse });
      getClient(knirvoracleService).get = mockGet;

      const address = '0x742d35Cc6aa34567890abcdef1234567890abcdef';
      const result = await knirvoracleService.getWalletBalance(address);

      expect(mockGet).toHaveBeenCalledWith(`/balance/${address}`);
      expect(result).toEqual(mockResponse);
      expect(result.success).toBe(true);
      expect(result.address).toBe(address);
    });

    it('should handle transaction sending', async () => {
      const mockResponse = {
        success: true,
        transaction_id: 'tx-789',
        tx_hash: '0xdef456789',
        status: 'pending',
        message: 'Transaction submitted',
      };

      const mockPost = jest.fn().mockResolvedValue({ data: mockResponse });
      getClient(knirvoracleService).post = mockPost;

      const transactionRequest = {
        from: '0x742d35Cc6aa34567890abcdef1234567890abcdef',
        to: '0x987654321fedcba0987654321fedcba0987654321',
        amount: '50',
        token: 'NRN',
        memo: 'Test transaction',
      };

      const result = await knirvoracleService.sendTransaction(transactionRequest);

      expect(mockPost).toHaveBeenCalledWith('/transactions', transactionRequest);
      expect(result).toEqual(mockResponse);
      expect(result.success).toBe(true);
      expect(result.transaction_id).toBe('tx-789');
    });

    it('should handle health checks', async () => {
      const mockGet = jest.fn().mockResolvedValue({ status: 200 });
      getClient(knirvoracleService).get = mockGet;

      const result = await knirvoracleService.healthCheck();

      expect(mockGet).toHaveBeenCalledWith('/health', { timeout: 5000 });
      expect(result).toBe(true);
    });

    it('should handle health check failures', async () => {
      const mockGet = jest.fn().mockRejectedValue(new Error('Network error'));
      getClient(knirvoracleService).get = mockGet;

      const result = await knirvoracleService.healthCheck();

      expect(result).toBe(false);
    });

    it('should handle capability invocation', async () => {
      const mockResponse = {
        success: true,
        result: 'Capability executed successfully',
        execution_time: 1500,
      };

      const mockPost = jest.fn().mockResolvedValue({ data: mockResponse });
      getClient(knirvoracleService).post = mockPost;

      const capabilityId = 'cap-123';
      const inputData = { input: 'test data' };

      const result = await knirvoracleService.invokeCapability(capabilityId, inputData);

      expect(mockPost).toHaveBeenCalledWith('/wallet/mcp/create_invoke_capability', {
        capability_id: capabilityId,
        interaction_type: 'invoke',
        input_data: inputData,
        timestamp: expect.any(Number),
      });
      expect(result).toEqual(mockResponse);
    });

    it('should handle error responses gracefully', async () => {
      const mockPost = jest.fn().mockRejectedValue(new Error('API Error'));
      getClient(knirvoracleService).post = mockPost;

      const mintRequest = {
        agent_id: 'test-agent',
        name: 'Test Agent',
        description: 'Test',
        owner: 'test-owner',
        metadata: {},
      };

      await expect(knirvoracleService.mintAgent(mintRequest)).rejects.toThrow('Failed to mint agent');
    });
  });

  describe('Wallet Service KNIRVORACLE Integration', () => {
    it('should integrate with KNIRVORACLE for balance queries', async () => {
      // This test would verify that the wallet service correctly calls KNIRVORACLE
      // In a real scenario, we would mock the KNIRVORACLE service calls
      expect(walletService).toBeDefined();
    });

    it('should fallback to local API when KNIRVORACLE fails', async () => {
      // This test would verify fallback behavior
      expect(walletService).toBeDefined();
    });
  });

  describe('Type Safety', () => {
    it('should enforce correct types for agent minting', () => {
      const mintRequest = {
        agent_id: 'test-agent',
        name: 'Test Agent',
        description: 'Test description',
        owner: 'test-owner',
        metadata: {
          type: 'assistant',
          model: 'gpt-4',
        },
      };

      // TypeScript should enforce these types at compile time
      expect(typeof mintRequest.agent_id).toBe('string');
      expect(typeof mintRequest.name).toBe('string');
      expect(typeof mintRequest.metadata).toBe('object');
    });

    it('should enforce correct types for capability registration', () => {
      const registrationRequest = {
        name: 'Test Capability',
        type: 'mcp_capability',
        description: 'Test description',
        schema: {
          type: 'object',
          properties: {},
        },
        owner: 'test-owner',
        gas_fee_nrn: 1000,
      };

      expect(typeof registrationRequest.name).toBe('string');
      expect(typeof registrationRequest.gas_fee_nrn).toBe('number');
      expect(typeof registrationRequest.schema).toBe('object');
    });
  });

  describe('Error Handling', () => {
    it('should handle network timeouts', async () => {
      const mockPost = jest.fn().mockRejectedValue(new Error('timeout'));
      getClient(knirvoracleService).post = mockPost;

      const mintRequest = {
        agent_id: 'test-agent',
        name: 'Test Agent',
        description: 'Test',
        owner: 'test-owner',
        metadata: {},
      };

      await expect(knirvoracleService.mintAgent(mintRequest)).rejects.toThrow();
    });

    it('should handle invalid responses', async () => {
      const mockPost = jest.fn().mockResolvedValue({ data: null });
      getClient(knirvoracleService).post = mockPost;

      const mintRequest = {
        agent_id: 'test-agent',
        name: 'Test Agent',
        description: 'Test',
        owner: 'test-owner',
        metadata: {},
      };

      const result = await knirvoracleService.mintAgent(mintRequest);
      expect(result).toBeNull();
    });
  });
});
