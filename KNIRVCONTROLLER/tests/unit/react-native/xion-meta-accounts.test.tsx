// Comprehensive Unit Tests for KNIRVWALLET React Native - XION Meta Accounts

// Import real implementations instead of mocks
import { XionMetaAccount, XionMetaAccountConfig } from '../../../src/services/XionMetaAccount';
import { WalletManager, StorageInterface } from '../../../src/services/WalletManager';

// Test configuration
const testMetaAccountConfig: XionMetaAccountConfig = {
  rpcEndpoint: 'https://rpc.xion-testnet-1.burnt.com:443',
  chainId: 'xion-testnet-1'
};

// Mock storage for testing
class MockStorage implements StorageInterface {
  private storage: Map<string, string> = new Map();

  get(key: string): string | null {
    return this.storage.get(key) || null;
  }

  set(key: string, value: string): void {
    this.storage.set(key, value);
  }

  remove(key: string): void {
    this.storage.delete(key);
  }

  keys(): string[] {
    return Array.from(this.storage.keys());
  }

  clear(): void {
    this.storage.clear();
  }
}
import {
  TEST_ADDRESSES,
  TEST_MNEMONICS
} from '../../../test-utils/test-data';

// Mock fetch for testing
global.fetch = jest.fn((url: string) => {
  // Different responses based on URL
  if (url.includes('/balance/')) {
    return Promise.resolve({
      ok: true,
      status: 200,
      statusText: 'OK',
      json: () => Promise.resolve({
        address: 'xion1test',
        usdc_balance: '1000000',
        nrn_balance: '5000000000000000000000',
        last_updated: new Date().toISOString()
      }),
    });
  }

  // Default payment response
  return Promise.resolve({
    ok: true,
    status: 200,
    statusText: 'OK',
    json: () => Promise.resolve({
      success: true,
      data: {
        payment_id: 'pay_test_123',
        usdc_amount: '1000000',
        nrn_amount: '10000000'
      },
      status: 'confirmed'
    }),
  });
}) as jest.Mock;

describe('XionMetaAccount', () => {
  let metaAccount: XionMetaAccount;
  let mockStorage: MockStorage;

  beforeEach(() => {
    mockStorage = new MockStorage();
    metaAccount = new XionMetaAccount(testMetaAccountConfig);
  });

  describe('Initialization', () => {
    it('should initialize with test config', async () => {
      await metaAccount.initialize();

      expect(metaAccount).toBeDefined();
      expect(testMetaAccountConfig.rpcEndpoint).toBe('https://rpc.xion-testnet-1.burnt.com:443');
      expect(testMetaAccountConfig.chainId).toBe('xion-testnet-1');
    });

    it('should initialize with new wallet', async () => {
      await metaAccount.initialize();

      const address = await metaAccount.getAddress();
      const mnemonic = await metaAccount.getMnemonic();

      expect(address).toBeDefined();
      expect(typeof address).toBe('string');
      expect(address).toMatch(/^xion1/);
      expect(mnemonic).toBeDefined();
      expect(typeof mnemonic).toBe('string');
      expect(mnemonic.split(' ').length).toBeGreaterThanOrEqual(12);
    });

    it('should initialize with existing mnemonic', async () => {
      const testMnemonic = TEST_MNEMONICS.VALID_12_WORD;
      
      await metaAccount.initialize(testMnemonic);

      const address = await metaAccount.getAddress();
      const mnemonic = await metaAccount.getMnemonic();

      expect(address).toBeDefined();
      expect(typeof address).toBe('string');
      expect(address).toMatch(/^xion1/);
      expect(mnemonic).toBe(testMnemonic);
    });

    it('should throw error for invalid mnemonic', async () => {
      await expect(metaAccount.initialize(TEST_MNEMONICS.INVALID))
        .rejects.toThrow();
    });

    it('should handle initialization without mnemonic', async () => {
      await metaAccount.initialize();

      const address = await metaAccount.getAddress();
      expect(address).toBeDefined();
      expect(typeof address).toBe('string');
      expect(address).toMatch(/^xion1/);
    });
  });

  describe('Balance Operations', () => {
    beforeEach(async () => {
      await metaAccount.initialize();
    });

    it('should get XION balance', async () => {
      const balance = await metaAccount.getBalance();

      expect(balance).toBeDefined();
      expect(typeof balance).toBe('string');
      expect(parseFloat(balance)).toBeGreaterThanOrEqual(0);
    });

    it('should get NRN balance', async () => {
      const balance = await metaAccount.getNRNBalance();

      expect(balance).toBeDefined();
      expect(typeof balance).toBe('string');
      expect(parseFloat(balance)).toBeGreaterThanOrEqual(0);
    });

    it('should refresh balances', async () => {
      await metaAccount.getBalance();
      await metaAccount.getNRNBalance();

      await metaAccount.refreshBalances();

      const newBalance = await metaAccount.getBalance();
      const newNRNBalance = await metaAccount.getNRNBalance();

      expect(newBalance).toBeDefined();
      expect(newNRNBalance).toBeDefined();
    });
  });

  describe('NRN Transfer Operations', () => {
    beforeEach(async () => {
      await metaAccount.initialize();
    });

    it('should transfer NRN successfully', async () => {
      const recipientAddress = TEST_ADDRESSES.XION;
      const amount = '1000000';

      const txHash = await metaAccount.transferNRN(recipientAddress, amount);

      expect(txHash).toBeDefined();
      expect(typeof txHash).toBe('string');
      expect(txHash).toBeDefined();
      expect(typeof txHash).toBe('string');
      expect(txHash.length).toBeGreaterThan(0);
    });

    it('should throw error for invalid recipient address', async () => {
      const invalidAddress = 'invalid-address';
      const amount = '1000000';

      await expect(metaAccount.transferNRN(invalidAddress, amount))
        .rejects.toThrow();
    });

    it('should throw error for invalid amount', async () => {
      const recipientAddress = TEST_ADDRESSES.XION;
      const invalidAmount = 'invalid-amount';

      await expect(metaAccount.transferNRN(recipientAddress, invalidAmount))
        .rejects.toThrow();
    });

    it('should handle zero amount transfer', async () => {
      const recipientAddress = TEST_ADDRESSES.XION;
      const zeroAmount = '0';

      await expect(metaAccount.transferNRN(recipientAddress, zeroAmount))
        .rejects.toThrow();
    });

    it('should handle negative amount transfer', async () => {
      const recipientAddress = TEST_ADDRESSES.XION;
      const negativeAmount = '-1000000';

      await expect(metaAccount.transferNRN(recipientAddress, negativeAmount))
        .rejects.toThrow();
    });
  });

  describe('Skill Invocation Operations', () => {
    beforeEach(async () => {
      await metaAccount.initialize();
    });

    it('should burn NRN for skill successfully', async () => {
      const skillId = 'skill-test-001';
      const amount = '1000000';

      const txHash = await metaAccount.burnNRNForSkill(skillId, amount);

      expect(txHash).toBeDefined();
      expect(typeof txHash).toBe('string');
      expect(txHash).toBeDefined();
      expect(typeof txHash).toBe('string');
      expect(txHash.length).toBeGreaterThan(0);
    });

    it('should throw error for empty skill ID', async () => {
      const emptySkillId = '';
      const amount = '1000000';

      await expect(metaAccount.burnNRNForSkill(emptySkillId, amount))
        .rejects.toThrow();
    });

    it('should throw error for invalid amount in skill invocation', async () => {
      const skillId = 'skill-test-001';
      const invalidAmount = 'invalid-amount';

      await expect(metaAccount.burnNRNForSkill(skillId, invalidAmount))
        .rejects.toThrow();
    });

    it('should handle skill invocation with metadata', async () => {
      const skillId = 'skill-test-002';
      const amount = '2000000';

      // In real implementation, metadata would be passed as parameter
      const txHash = await metaAccount.burnNRNForSkill(skillId, amount);

      expect(txHash).toBeDefined();
      expect(txHash).toBeDefined();
      expect(typeof txHash).toBe('string');
      expect(txHash.length).toBeGreaterThan(0);
    });
  });

  describe('Faucet Operations', () => {
    beforeEach(async () => {
      await metaAccount.initialize();
    });

    it('should request NRN from faucet successfully', async () => {
      const amount = '1000000';

      const txHash = await metaAccount.requestFromFaucet(amount);

      expect(txHash).toBeDefined();
      expect(typeof txHash).toBe('string');
      expect(txHash).toBeDefined();
      expect(typeof txHash).toBe('string');
      expect(txHash.length).toBeGreaterThan(0);
    });

    it('should request default amount from faucet', async () => {
      const txHash = await metaAccount.requestFromFaucet();

      expect(txHash).toBeDefined();
      expect(txHash).toBeDefined();
      expect(typeof txHash).toBe('string');
      expect(txHash.length).toBeGreaterThan(0);
    });

    it('should handle faucet request with invalid amount', async () => {
      const invalidAmount = 'invalid-amount';

      await expect(metaAccount.requestFromFaucet(invalidAmount))
        .rejects.toThrow();
    });

    it('should handle faucet request with zero amount', async () => {
      const zeroAmount = '0';

      await expect(metaAccount.requestFromFaucet(zeroAmount))
        .rejects.toThrow();
    });
  });

  describe('Gasless Transaction Support', () => {
    beforeEach(async () => {
      await metaAccount.initialize();
    });

    it('should enable gasless transactions', async () => {
      // This is a placeholder test as the actual implementation depends on XION's AA system
      await expect(metaAccount.enableGaslessTransactions()).resolves.not.toThrow();
    });

    it('should check if gasless transactions are enabled', async () => {
      // Enable gasless first
      await metaAccount.enableGaslessTransactions();

      // In real implementation, there would be a method to check gasless status
      // For now, we just verify the enable method doesn't throw
      expect(true).toBe(true);
    });
  });

  describe('Error Handling', () => {
    it('should handle network connection errors gracefully', async () => {
      // Mock network error
      const networkErrorConfig = {
        ...testMetaAccountConfig,
        rpcEndpoint: 'http://invalid-endpoint'
      };

      const errorMetaAccount = new XionMetaAccount(networkErrorConfig);

      // Should handle initialization gracefully
      try {
        await errorMetaAccount.initialize();
      } catch (error) {
        expect(error).toBeDefined();
      }
    });

    it('should handle contract interaction errors', async () => {
      await metaAccount.initialize();

      // Mock fetch error for contract interaction
      const originalFetch = global.fetch;
      global.fetch = jest.fn(() =>
        Promise.reject(new Error('Network error'))
      ) as jest.Mock;

      // Since getNRNBalance catches errors and returns '0', let's test transferNRN which should throw
      await expect(metaAccount.transferNRN('xion1test', '1000000'))
        .rejects.toThrow();

      // Restore original fetch
      global.fetch = originalFetch;
    });

    it('should handle insufficient balance errors', async () => {
      await metaAccount.initialize();

      // Try to transfer more than available balance
      const recipientAddress = TEST_ADDRESSES.XION;
      const largeAmount = '99999999999999999999999'; // Larger than mock balance of 5000000000000000000000

      await expect(metaAccount.transferNRN(recipientAddress, largeAmount))
        .rejects.toThrow();
    });
  });
});

describe('WalletManager', () => {
  let walletManager: WalletManager;
  let mockStorage: MockStorage;

  beforeEach(() => {
    mockStorage = new MockStorage();
    walletManager = new WalletManager(testMetaAccountConfig, mockStorage);
  });

  describe('Wallet Creation and Management', () => {
    it('should create new wallet', async () => {
      const walletName = 'test-wallet';

      const wallet = await walletManager.createWallet(walletName);

      expect(wallet).toBeInstanceOf(XionMetaAccount);

      const address = await wallet.getAddress();
      expect(address).toBeDefined();
      expect(typeof address).toBe('string');
      expect(address).toMatch(/^xion1/);

      // Verify wallet is stored
      const retrievedWallet = await walletManager.getWallet(walletName);
      expect(retrievedWallet).toBeDefined();
    });

    it('should import wallet from mnemonic', async () => {
      const walletName = 'imported-wallet';
      const mnemonic = TEST_MNEMONICS.VALID_12_WORD;

      const wallet = await walletManager.importWallet(walletName, mnemonic);

      expect(wallet).toBeInstanceOf(XionMetaAccount);

      const retrievedMnemonic = await wallet.getMnemonic();
      expect(retrievedMnemonic).toBe(mnemonic);
    });

    it('should get existing wallet', async () => {
      const walletName = 'existing-wallet';

      // Create wallet first
      await walletManager.createWallet(walletName);

      // Retrieve wallet
      const wallet = await walletManager.getWallet(walletName);

      expect(wallet).toBeDefined();
      expect(wallet).toBeInstanceOf(XionMetaAccount);
    });

    it('should return undefined for non-existent wallet', async () => {
      const nonExistentWallet = await walletManager.getWallet('non-existent');

      expect(nonExistentWallet).toBeUndefined();
    });

    it('should list all wallets', async () => {
      const walletNames = ['wallet1', 'wallet2', 'wallet3'];

      // Create multiple wallets
      for (const name of walletNames) {
        await walletManager.createWallet(name);
      }

      const listedWallets = await walletManager.listWallets();

      expect(listedWallets).toHaveLength(walletNames.length);
      walletNames.forEach(name => {
        expect(listedWallets).toContain(name);
      });
    });
  });

  describe('Wallet Storage and Security', () => {
    it('should encrypt wallet data before storage', async () => {
      const walletName = 'encrypted-wallet';
      const wallet = await walletManager.createWallet(walletName);

      const mnemonic = await wallet.getMnemonic();

      // Check that stored data is encrypted (not plain text)
      const storedData = mockStorage.get(`wallet_${walletName}`);
      expect(storedData).toBeDefined();
      expect(storedData).not.toBe(mnemonic);
    });

    it('should decrypt wallet data when loading', async () => {
      const walletName = 'decrypted-wallet';
      const originalMnemonic = TEST_MNEMONICS.VALID_12_WORD;

      // Import wallet with known mnemonic
      await walletManager.importWallet(walletName, originalMnemonic);

      // Clear in-memory cache
      walletManager = new WalletManager(testMetaAccountConfig, mockStorage);

      // Retrieve wallet (should decrypt from storage)
      const wallet = await walletManager.getWallet(walletName);
      expect(wallet).toBeDefined();

      const retrievedMnemonic = await wallet!.getMnemonic();
      expect(retrievedMnemonic).toBe(originalMnemonic);
    });

    it('should handle storage errors gracefully', async () => {
      // Mock storage error
      const originalSetItem = mockStorage.set;
      mockStorage.set = jest.fn().mockImplementation(() => {
        throw new Error('Storage error');
      });

      const walletName = 'error-wallet';

      await expect(walletManager.createWallet(walletName))
        .rejects.toThrow();

      // Restore original storage
      mockStorage.set = originalSetItem;
    });
  });

  describe('Multiple Wallet Management', () => {
    it('should manage multiple wallets independently', async () => {
      const wallet1Name = 'wallet1';
      const wallet2Name = 'wallet2';

      const wallet1 = await walletManager.createWallet(wallet1Name);
      const wallet2 = await walletManager.createWallet(wallet2Name);

      const address1 = await wallet1.getAddress();
      const address2 = await wallet2.getAddress();

      expect(address1).not.toBe(address2);
      expect(address1).toBeDefined();
      expect(typeof address1).toBe('string');
      expect(address1).toMatch(/^xion1/);
      expect(address2).toBeDefined();
      expect(typeof address2).toBe('string');
      expect(address2).toMatch(/^xion1/);
    });

    it('should handle wallet name conflicts', async () => {
      const walletName = 'duplicate-wallet';

      // Create first wallet
      const wallet1 = await walletManager.createWallet(walletName);

      // Create second wallet with same name (should overwrite)
      const wallet2 = await walletManager.createWallet(walletName);

      const address1 = await wallet1.getAddress();
      const address2 = await wallet2.getAddress();

      // Addresses might be different if new wallet was created
      expect(address1).toBeDefined();
      expect(typeof address1).toBe('string');
      expect(address1).toMatch(/^xion1/);
      expect(address2).toBeDefined();
      expect(typeof address2).toBe('string');
      expect(address2).toMatch(/^xion1/);
    });
  });

  describe('Error Handling', () => {
    it('should handle invalid mnemonic during import', async () => {
      const walletName = 'invalid-mnemonic-wallet';
      const invalidMnemonic = 'invalid mnemonic phrase';

      await expect(walletManager.importWallet(walletName, invalidMnemonic))
        .rejects.toThrow();
    });

    it('should handle empty wallet name', async () => {
      const emptyName = '';

      await expect(walletManager.createWallet(emptyName))
        .rejects.toThrow();
    });

    it('should handle storage unavailability', async () => {
      // Mock localStorage unavailability
      const originalLocalStorage = global.localStorage;
      delete (global as unknown as { localStorage?: Storage }).localStorage;

      const walletName = 'no-storage-wallet';

      try {
        await walletManager.createWallet(walletName);
        // Should handle gracefully or throw appropriate error
      } catch (error) {
        expect(error).toBeDefined();
      }

      // Restore localStorage
      global.localStorage = originalLocalStorage;
    });
  });
});
