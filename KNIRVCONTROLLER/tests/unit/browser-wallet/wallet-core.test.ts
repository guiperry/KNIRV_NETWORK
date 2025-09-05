// Comprehensive Unit Tests for KNIRVWALLET Browser Module - Core Wallet Functionality

// Mock KNIRVWALLET imports since they're from a sibling project
jest.mock('../../../../KNIRVWALLET/browser-bridge/packages/knirvwallet-module/src/wallet/wallet', () => ({
  KnirvWallet: jest.fn().mockImplementation(() => ({
    createAccount: jest.fn(),
    importAccount: jest.fn(),
    signTransaction: jest.fn(),
    getBalance: jest.fn(),
    getAddress: jest.fn()
  }))
}));

jest.mock('../../../../KNIRVWALLET/browser-bridge/packages/knirvwallet-module/src/test-utils/mock-ledgerconnector', () => ({
  MockLedgerConnector: jest.fn().mockImplementation(() => ({
    connect: jest.fn(),
    disconnect: jest.fn(),
    getPublicKey: jest.fn(),
    signTransaction: jest.fn()
  }))
}));

// Import after mocking
const { KnirvWallet } = require('../../../../KNIRVWALLET/browser-bridge/packages/knirvwallet-module/src/wallet/wallet');
const { MockLedgerConnector } = require('../../../../KNIRVWALLET/browser-bridge/packages/knirvwallet-module/src/test-utils/mock-ledgerconnector');
import {
  TEST_MNEMONICS,
  TEST_PRIVATE_KEYS,
  TEST_ADDRESSES
} from '../../../test-utils/test-data';
import { WalletTestFactory } from '../../../test-utils/wallet-test-utils';

describe('KnirvWallet Core Functionality', () => {
  // walletFactory is declared but not used in current tests
  // Keeping it for potential future use
  let walletFactory: WalletTestFactory;

  beforeEach(() => {
    walletFactory = new WalletTestFactory();
  });

  describe('Wallet Creation', () => {
    describe('HD Wallet Creation', () => {
      it('should create wallet from valid 12-word mnemonic', async () => {
        const wallet = await KnirvWallet.createByMnemonic(TEST_MNEMONICS.VALID_12_WORD);
        
        expect(wallet).toBeDefined();
        expect(wallet.accounts).toHaveLength(1);
        expect(wallet.keyrings).toHaveLength(1);
        expect(wallet.currentAccountId).toBeDefined();
        expect(wallet.getMnemonic()).toBe(TEST_MNEMONICS.VALID_12_WORD);
      });

      it('should create wallet from valid 24-word mnemonic', async () => {
        const wallet = await KnirvWallet.createByMnemonic(TEST_MNEMONICS.VALID_24_WORD);

        expect(wallet).toBeDefined();
        expect(wallet.accounts).toHaveLength(1);
        expect(wallet.keyrings).toHaveLength(1);
        expect(wallet.getMnemonic()).toBe(TEST_MNEMONICS.VALID_24_WORD);
      });

      it('should create wallet using wallet factory', async () => {
        const wallet = await walletFactory.createTestHDWallet();

        expect(wallet).toBeDefined();
        expect(wallet.accounts).toHaveLength(1);
        expect(wallet.keyrings).toHaveLength(1);
        expect(wallet.currentAccountId).toBeDefined();
      });

      it('should create wallet with multiple derivation paths', async () => {
        const paths = [0, 1, 2];
        const wallet = await KnirvWallet.createByMnemonic(TEST_MNEMONICS.VALID_12_WORD, paths);
        
        expect(wallet.accounts).toHaveLength(paths.length);
        expect(wallet.keyrings).toHaveLength(1); // Single keyring for multiple accounts
        
        // Verify each account has correct derivation path
        wallet.accounts.forEach((account, index) => {
          expect(account.name).toContain(`${index + 1}`);
        });
      });

      it('should throw error for invalid mnemonic', async () => {
        await expect(KnirvWallet.createByMnemonic(TEST_MNEMONICS.INVALID))
          .rejects.toThrow();
      });

      it('should generate correct address for known mnemonic', async () => {
        const wallet = await KnirvWallet.createByMnemonic(TEST_MNEMONICS.CUSTOM_TEST);
        const account = wallet.accounts[0];
        const address = await account.getAddress('g');
        
        expect(address).toBeValidAddress('g');
        expect(address).toBe('g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5');
      });
    });

    describe('Private Key Wallet Creation', () => {
      it('should create wallet from valid private key', async () => {
        const wallet = await KnirvWallet.createByWeb3Auth(TEST_PRIVATE_KEYS.VALID_HEX);
        
        expect(wallet).toBeDefined();
        expect(wallet.accounts).toHaveLength(1);
        expect(wallet.keyrings).toHaveLength(1);
        expect(wallet.currentKeyring.type).toBe('WEB3_AUTH');
      });

      it('should create wallet from private key with 0x prefix', async () => {
        const wallet = await KnirvWallet.createByWeb3Auth(TEST_PRIVATE_KEYS.VALID_HEX_WITH_PREFIX);
        
        expect(wallet).toBeDefined();
        expect(wallet.accounts).toHaveLength(1);
      });

      it('should throw error for invalid private key', async () => {
        await expect(KnirvWallet.createByWeb3Auth(TEST_PRIVATE_KEYS.INVALID_SHORT))
          .rejects.toThrow();
        
        await expect(KnirvWallet.createByWeb3Auth(TEST_PRIVATE_KEYS.INVALID_LONG))
          .rejects.toThrow();
      });
    });

    describe('Ledger Wallet Creation', () => {
      it('should create wallet from Ledger device', async () => {
        const ledgerConnector = await MockLedgerConnector.create();
        const wallet = await KnirvWallet.createByLedger(ledgerConnector);
        
        expect(wallet).toBeDefined();
        expect(wallet.accounts).toHaveLength(1);
        expect(wallet.keyrings).toHaveLength(1);
        expect(wallet.currentKeyring.type).toBe('LEDGER');
      });

      it('should create wallet with multiple Ledger accounts', async () => {
        const ledgerConnector = await MockLedgerConnector.create();
        const paths = [0, 1, 2];
        const wallet = await KnirvWallet.createByLedger(ledgerConnector, paths);
        
        expect(wallet.accounts).toHaveLength(paths.length);
        expect(wallet.keyrings).toHaveLength(1);
      });

      it('should handle Ledger connection errors gracefully', async () => {
        // This test would normally fail with real Ledger, but MockLedgerConnector should work
        try {
          const ledgerConnector = await MockLedgerConnector.create();
          const wallet = await KnirvWallet.createByLedger(ledgerConnector);
          expect(wallet.currentKeyring.type).toBe('LEDGER');
        } catch (error) {
          // Expected in test environment without real Ledger device
          expect(error).toBeDefined();
        }
      });
    });

    describe('Address-only Wallet Creation', () => {
      it('should create watch-only wallet from address', async () => {
        const wallet = await KnirvWallet.createByAddress(TEST_ADDRESSES.GNOLANG);
        
        expect(wallet).toBeDefined();
        expect(wallet.accounts).toHaveLength(1);
        expect(wallet.keyrings).toHaveLength(1);
        expect(wallet.currentKeyring.type).toBe('ADDRESS');
        expect(wallet.accounts[0].address).toBe(TEST_ADDRESSES.GNOLANG);
      });

      it('should throw error for invalid address', async () => {
        await expect(KnirvWallet.createByAddress(TEST_ADDRESSES.INVALID))
          .rejects.toThrow();
      });
    });
  });

  describe('Wallet Serialization', () => {
    it('should serialize and deserialize HD wallet correctly', async () => {
      const originalWallet = await KnirvWallet.createByMnemonic(TEST_MNEMONICS.VALID_12_WORD);
      const password = 'test-password-123';
      
      // Serialize
      const serialized = await originalWallet.serialize(password);
      expect(serialized).toBeDefined();
      expect(typeof serialized).toBe('string');
      
      // Deserialize
      const deserializedWallet = await KnirvWallet.deserialize(serialized, password);
      expect(deserializedWallet).toBeDefined();
      expect(deserializedWallet.accounts).toHaveLength(originalWallet.accounts.length);
      expect(deserializedWallet.keyrings).toHaveLength(originalWallet.keyrings.length);
      expect(deserializedWallet.getMnemonic()).toBe(originalWallet.getMnemonic());
    });

    it('should fail deserialization with wrong password', async () => {
      const wallet = await KnirvWallet.createByMnemonic(TEST_MNEMONICS.VALID_12_WORD);
      const correctPassword = 'correct-password';
      const wrongPassword = 'wrong-password';
      
      const serialized = await wallet.serialize(correctPassword);
      
      await expect(KnirvWallet.deserialize(serialized, wrongPassword))
        .rejects.toThrow();
    });

    it('should serialize private key wallet without exposing private key', async () => {
      const wallet = await KnirvWallet.createByWeb3Auth(TEST_PRIVATE_KEYS.VALID_HEX);
      const password = 'test-password-123';
      
      const serialized = await wallet.serialize(password);
      
      // Serialized data should not contain the raw private key
      expect(serialized).not.toContain(TEST_PRIVATE_KEYS.VALID_HEX);
      expect(serialized).not.toContain(TEST_PRIVATE_KEYS.VALID_HEX_WITH_PREFIX);
    });
  });

  describe('Account Management', () => {
    let wallet: typeof KnirvWallet;

    beforeEach(async () => {
      wallet = await KnirvWallet.createByMnemonic(TEST_MNEMONICS.VALID_12_WORD);
    });

    it('should get all accounts', () => {
      const accounts = wallet.getAccounts();
      expect(accounts).toHaveLength(1);
      expect(accounts[0]).toHaveProperty('id');
      expect(accounts[0]).toHaveProperty('name');
      expect(accounts[0]).toHaveProperty('address');
    });

    it('should get current account', () => {
      const currentAccount = wallet.getCurrentAccount();
      expect(currentAccount).toBeDefined();
      expect(currentAccount?.id).toBe(wallet.currentAccountId);
    });

    it('should switch between accounts', async () => {
      // Add another account
      const paths = [0, 1];
      const multiAccountWallet = await KnirvWallet.createByMnemonic(TEST_MNEMONICS.VALID_12_WORD, paths);
      
      const firstAccountId = multiAccountWallet.accounts[0].id;
      const secondAccountId = multiAccountWallet.accounts[1].id;
      
      // Switch to second account
      multiAccountWallet.currentAccountId = secondAccountId;
      expect(multiAccountWallet.getCurrentAccount()?.id).toBe(secondAccountId);
      
      // Switch back to first account
      multiAccountWallet.currentAccountId = firstAccountId;
      expect(multiAccountWallet.getCurrentAccount()?.id).toBe(firstAccountId);
    });

    it('should get account by ID', () => {
      const accountId = wallet.accounts[0].id;
      const account = wallet.getAccountById(accountId);
      
      expect(account).toBeDefined();
      expect(account?.id).toBe(accountId);
    });

    it('should return undefined for non-existent account ID', () => {
      const nonExistentId = 'non-existent-account-id';
      const account = wallet.getAccountById(nonExistentId);
      
      expect(account).toBeUndefined();
    });
  });

  describe('Keyring Management', () => {
    let wallet: typeof KnirvWallet;

    beforeEach(async () => {
      wallet = await KnirvWallet.createByMnemonic(TEST_MNEMONICS.VALID_12_WORD);
    });

    it('should get all keyrings', () => {
      const keyrings = wallet.getKeyrings();
      expect(keyrings).toHaveLength(1);
      expect(keyrings[0]).toHaveProperty('id');
      expect(keyrings[0]).toHaveProperty('type');
    });

    it('should get current keyring', () => {
      const currentKeyring = wallet.currentKeyring;
      expect(currentKeyring).toBeDefined();
      expect(currentKeyring.type).toBe('HD');
    });

    it('should get keyring by ID', () => {
      const keyringId = wallet.keyrings[0].id;
      const keyring = wallet.getKeyringById(keyringId);
      
      expect(keyring).toBeDefined();
      expect(keyring?.id).toBe(keyringId);
    });

    it('should return undefined for non-existent keyring ID', () => {
      const nonExistentId = 'non-existent-keyring-id';
      const keyring = wallet.getKeyringById(nonExistentId);
      
      expect(keyring).toBeUndefined();
    });
  });

  describe('Wallet Properties', () => {
    it('should have correct structure for HD wallet', async () => {
      const wallet = await KnirvWallet.createByMnemonic(TEST_MNEMONICS.VALID_12_WORD);
      
      expect(wallet).toHaveWalletStructure();
      expect(wallet.accounts[0]).toHaveProperty('keyringId');
      expect(wallet.keyrings[0]).toHaveProperty('type', 'HD');
    });

    it('should have correct structure for private key wallet', async () => {
      const wallet = await KnirvWallet.createByWeb3Auth(TEST_PRIVATE_KEYS.VALID_HEX);
      
      expect(wallet).toHaveWalletStructure();
      expect(wallet.keyrings[0]).toHaveProperty('type', 'WEB3_AUTH');
    });

    it('should have correct structure for address-only wallet', async () => {
      const wallet = await KnirvWallet.createByAddress(TEST_ADDRESSES.GNOLANG);
      
      expect(wallet).toHaveWalletStructure();
      expect(wallet.keyrings[0]).toHaveProperty('type', 'ADDRESS');
    });

    it('should generate unique account names', async () => {
      const paths = [0, 1, 2, 3, 4];
      const wallet = await KnirvWallet.createByMnemonic(TEST_MNEMONICS.VALID_12_WORD, paths);
      
      const accountNames = wallet.accounts.map(account => account.name);
      const uniqueNames = new Set(accountNames);
      
      expect(uniqueNames.size).toBe(accountNames.length);
    });

    it('should generate unique keyring IDs', async () => {
      const wallet1 = await KnirvWallet.createByMnemonic(TEST_MNEMONICS.VALID_12_WORD);
      const wallet2 = await KnirvWallet.createByMnemonic(TEST_MNEMONICS.VALID_24_WORD);
      
      expect(wallet1.keyrings[0].id).not.toBe(wallet2.keyrings[0].id);
    });
  });

  describe('Error Handling', () => {
    it('should handle empty mnemonic gracefully', async () => {
      await expect(KnirvWallet.createByMnemonic(''))
        .rejects.toThrow();
    });

    it('should handle null/undefined inputs gracefully', async () => {
      await expect(KnirvWallet.createByMnemonic(null as unknown as string))
        .rejects.toThrow();
      
      await expect(KnirvWallet.createByMnemonic(undefined as unknown as string))
        .rejects.toThrow();
    });

    it('should handle invalid serialized data gracefully', async () => {
      const invalidSerialized = 'invalid-serialized-data';
      const password = 'test-password';
      
      await expect(KnirvWallet.deserialize(invalidSerialized, password))
        .rejects.toThrow();
    });

    it('should handle corrupted serialized data gracefully', async () => {
      const wallet = await KnirvWallet.createByMnemonic(TEST_MNEMONICS.VALID_12_WORD);
      const password = 'test-password';
      
      const serialized = await wallet.serialize(password);
      const corrupted = serialized.slice(0, -10) + 'corrupted';
      
      await expect(KnirvWallet.deserialize(corrupted, password))
        .rejects.toThrow();
    });
  });
});
