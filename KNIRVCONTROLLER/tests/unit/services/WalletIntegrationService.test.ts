/**
 * Tests for WalletIntegrationService
 */

import { walletIntegrationService, WalletAccount, TransactionRequest, WalletIntegrationService } from '../../../src/services/WalletIntegrationService';

// Mock fetch globally
global.fetch = jest.fn();

// Mock window.location
Object.defineProperty(window, 'location', {
  value: {
    hostname: 'localhost',
    href: 'http://localhost:3000'
  },
  writable: true
});

describe('WalletIntegrationService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    // Reset service state
    walletIntegrationService['currentAccount'] = null;
    walletIntegrationService['transactions'].clear();
    walletIntegrationService['connectionStatus'] = {
      connected: false,
      bridgeUrl: 'http://localhost:3004',
      lastSync: new Date()
    };
  });

  describe('detectBridgeUrl', () => {
    it('should detect localhost URL for development', () => {
      const service = new (walletIntegrationService.constructor as typeof WalletIntegrationService)();
      expect(service['bridgeUrl']).toBe('http://localhost:3004');
    });

    it('should use production URL for non-localhost', () => {
      Object.defineProperty(window, 'location', {
        value: { hostname: 'app.knirv.com' },
        writable: true
      });
      
      const service = new (walletIntegrationService.constructor as typeof WalletIntegrationService)();
      expect(service['bridgeUrl']).toBe('https://wallet.knirv.com');
    });
  });

  describe('connectWallet', () => {
    it('should connect wallet successfully', async () => {
      const mockAccount = {
        id: 'test-account-123',
        address: 'knirv1test123...',
        name: 'Test Account',
        balance: '1000.00',
        nrnBalance: '500.00',
        isConnected: true
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => mockAccount
      });

      const account = await walletIntegrationService.connectWallet();

      expect(account).toEqual(mockAccount);
      expect(walletIntegrationService.getCurrentAccount()).toEqual(mockAccount);
      expect(walletIntegrationService.getConnectionStatus().connected).toBe(true);
    });

    it('should handle connection failure', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: false,
        statusText: 'Unauthorized'
      });

      await expect(walletIntegrationService.connectWallet())
        .rejects.toThrow('Failed to connect wallet: Wallet connection failed: Unauthorized');
    });

    it('should handle network error', async () => {
      (fetch as jest.Mock).mockRejectedValueOnce(new Error('Network error'));

      await expect(walletIntegrationService.connectWallet())
        .rejects.toThrow('Failed to connect wallet: Network error');
    });
  });

  describe('disconnectWallet', () => {
    beforeEach(async () => {
      // Set up connected state
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          id: 'test-account',
          address: 'knirv1test...',
          name: 'Test Account',
          balance: '1000.00',
          nrnBalance: '500.00',
          isConnected: true
        })
      });
      await walletIntegrationService.connectWallet();
    });

    it('should disconnect wallet successfully', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({ ok: true });

      await walletIntegrationService.disconnectWallet();

      expect(walletIntegrationService.getCurrentAccount()).toBeNull();
      expect(walletIntegrationService.getConnectionStatus().connected).toBe(false);
    });

    it('should handle disconnection API failure gracefully', async () => {
      (fetch as jest.Mock).mockRejectedValueOnce(new Error('Network error'));

      await walletIntegrationService.disconnectWallet();

      // Should still disconnect locally even if API call fails
      expect(walletIntegrationService.getCurrentAccount()).toBeNull();
      expect(walletIntegrationService.getConnectionStatus().connected).toBe(false);
    });
  });

  describe('getAccountBalance', () => {
    let testAccount: WalletAccount;

    beforeEach(async () => {
      testAccount = {
        id: 'test-account',
        address: 'knirv1test...',
        name: 'Test Account',
        balance: '1000.00',
        nrnBalance: '500.00',
        isConnected: true
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => testAccount
      });
      await walletIntegrationService.connectWallet();
    });

    it('should get account balance successfully', async () => {
      const updatedBalance = {
        balance: '1200.00',
        nrnBalance: '600.00'
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => updatedBalance
      });

      const balance = await walletIntegrationService.getAccountBalance(testAccount.id);

      expect(balance).toEqual(updatedBalance);
      expect(walletIntegrationService.getCurrentAccount()?.balance).toBe('1200.00');
      expect(walletIntegrationService.getCurrentAccount()?.nrnBalance).toBe('600.00');
    });

    it('should fail for non-existent account', async () => {
      await expect(walletIntegrationService.getAccountBalance('non-existent'))
        .rejects.toThrow('Account not found or not connected');
    });

    it('should return cached balance on API failure', async () => {
      (fetch as jest.Mock).mockRejectedValueOnce(new Error('Network error'));

      const balance = await walletIntegrationService.getAccountBalance(testAccount.id);

      expect(balance).toEqual({
        balance: testAccount.balance,
        nrnBalance: testAccount.nrnBalance
      });
    });
  });

  describe('createTransaction', () => {
    let testAccount: WalletAccount;

    beforeEach(async () => {
      testAccount = {
        id: 'test-account',
        address: 'knirv1test...',
        name: 'Test Account',
        balance: '1000.00',
        nrnBalance: '500.00',
        isConnected: true
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => testAccount
      });
      await walletIntegrationService.connectWallet();
    });

    it('should create transaction successfully', async () => {
      const transactionRequest: TransactionRequest = {
        from: testAccount.address,
        to: 'knirv1recipient...',
        amount: '100.00',
        memo: 'Test transaction'
      };

      const mockResponse = {
        transactionId: 'tx-123',
        hash: 'hash-456'
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse
      });

      const transactionId = await walletIntegrationService.createTransaction(transactionRequest);

      expect(transactionId).toBe('tx-123');
      
      // Check that transaction was stored locally
      const transactions = walletIntegrationService['transactions'];
      expect(transactions.has('tx-123')).toBe(true);
      
      const storedTx = transactions.get('tx-123');
      expect(storedTx?.from).toBe(transactionRequest.from);
      expect(storedTx?.to).toBe(transactionRequest.to);
      expect(storedTx?.amount).toBe(transactionRequest.amount);
      expect(storedTx?.status).toBe('pending');
    });

    it('should fail when no wallet connected', async () => {
      await walletIntegrationService.disconnectWallet();

      const transactionRequest: TransactionRequest = {
        from: 'knirv1test...',
        to: 'knirv1recipient...',
        amount: '100.00'
      };

      await expect(walletIntegrationService.createTransaction(transactionRequest))
        .rejects.toThrow('No wallet connected');
    });

    it('should handle transaction creation failure', async () => {
      const transactionRequest: TransactionRequest = {
        from: testAccount.address,
        to: 'knirv1recipient...',
        amount: '100.00'
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: false,
        statusText: 'Insufficient funds'
      });

      await expect(walletIntegrationService.createTransaction(transactionRequest))
        .rejects.toThrow('Failed to create transaction: Transaction creation failed: Insufficient funds');
    });
  });

  describe('invokeSkill', () => {
    let testAccount: WalletAccount;

    beforeEach(async () => {
      testAccount = {
        id: 'test-account',
        address: 'knirv1test...',
        name: 'Test Account',
        balance: '1000.00',
        nrnBalance: '500.00',
        isConnected: true
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => testAccount
      });
      await walletIntegrationService.connectWallet();
    });

    it('should invoke skill successfully', async () => {
      const skillRequest = {
        skillId: 'analysis-skill',
        skillName: 'Data Analysis',
        nrnCost: '50.00',
        parameters: { data: 'test' },
        expectedOutput: { result: 'analysis' },
        timeout: 30000
      };

      const mockResponse = {
        transactionId: 'skill-tx-123',
        hash: 'skill-hash-456'
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse
      });

      const transactionId = await walletIntegrationService.invokeSkill(skillRequest);

      expect(transactionId).toBe('skill-tx-123');
      
      // Check that skill transaction was stored
      const transactions = walletIntegrationService['transactions'];
      const storedTx = transactions.get('skill-tx-123');
      expect(storedTx?.nrnAmount).toBe('50.00');
      expect(storedTx?.memo).toContain('Data Analysis');
    });

    it('should fail when no wallet connected', async () => {
      await walletIntegrationService.disconnectWallet();

      const skillRequest = {
        skillId: 'test-skill',
        skillName: 'Test',
        nrnCost: '10.00',
        parameters: {},
        expectedOutput: {},
        timeout: 30000
      };

      await expect(walletIntegrationService.invokeSkill(skillRequest))
        .rejects.toThrow('No wallet connected');
    });
  });

  describe('checkTransactionStatus', () => {
    it('should check transaction status successfully', async () => {
      // Create a transaction first
      const testAccount = {
        id: 'test-account',
        address: 'knirv1test...',
        name: 'Test Account',
        balance: '1000.00',
        nrnBalance: '500.00',
        isConnected: true
      };

      (fetch as jest.Mock)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => testAccount
        })
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ transactionId: 'tx-123', hash: 'hash-456' })
        });

      await walletIntegrationService.connectWallet();
      const txId = await walletIntegrationService.createTransaction({
        from: testAccount.address,
        to: 'knirv1recipient...',
        amount: '100.00'
      });

      // Mock status check response
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          status: 'confirmed',
          hash: 'updated-hash',
          blockHeight: 12345,
          gasUsed: 21000
        })
      });

      const transaction = await walletIntegrationService.checkTransactionStatus(txId);

      expect(transaction!.status).toBe('confirmed');
      expect(transaction!.blockHeight).toBe(12345);
      expect(transaction!.gasUsed).toBe(21000);
    });

    it('should return null for non-existent transaction', async () => {
      const result = await walletIntegrationService.checkTransactionStatus('non-existent');
      expect(result).toBeNull();
    });
  });

  describe('getTransactionHistory', () => {
    it('should get transaction history when connected', async () => {
      const testAccount = {
        id: 'test-account',
        address: 'knirv1test...',
        name: 'Test Account',
        balance: '1000.00',
        nrnBalance: '500.00',
        isConnected: true
      };

      const mockHistory = [
        {
          id: 'tx-1',
          hash: 'hash-1',
          from: testAccount.address,
          to: 'knirv1recipient1...',
          amount: '50.00',
          status: 'confirmed',
          timestamp: new Date()
        },
        {
          id: 'tx-2',
          hash: 'hash-2',
          from: testAccount.address,
          to: 'knirv1recipient2...',
          amount: '75.00',
          status: 'pending',
          timestamp: new Date()
        }
      ];

      (fetch as jest.Mock)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => testAccount
        })
        .mockResolvedValueOnce({
          ok: true,
          json: async () => mockHistory
        });

      await walletIntegrationService.connectWallet();
      const history = await walletIntegrationService.getTransactionHistory();

      expect(history).toEqual(mockHistory);
    });

    it('should return empty array when not connected', async () => {
      const history = await walletIntegrationService.getTransactionHistory();
      expect(history).toEqual([]);
    });
  });

  describe('openWalletInterface', () => {
    it('should open wallet interface in new window', () => {
      const mockOpen = jest.fn();
      global.window.open = mockOpen;

      walletIntegrationService.openWalletInterface();

      expect(mockOpen).toHaveBeenCalledWith(
        'http://localhost:3004/wallet',
        'knirvwallet',
        'width=400,height=600,scrollbars=yes,resizable=yes'
      );
    });
  });
});
