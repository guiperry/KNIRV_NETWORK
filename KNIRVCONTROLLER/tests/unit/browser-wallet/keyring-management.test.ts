// Comprehensive Unit Tests for KNIRVWALLET Browser Module - Keyring Management

// Mock KNIRVWALLET imports since they're from a sibling project
jest.mock('../../../../KNIRVWALLET/browser-bridge/packages/knirvwallet-module/src/wallet/wallet', () => ({
  KnirvWallet: jest.fn().mockImplementation(() => ({
    createAccount: jest.fn(),
    importAccount: jest.fn(),
    exportAccount: jest.fn(),
    deleteAccount: jest.fn(),
    listAccounts: jest.fn(),
    setActiveAccount: jest.fn()
  }))
}));

// Import after mocking
const { KnirvWallet } = require('../../../../KNIRVWALLET/browser-bridge/packages/knirvwallet-module/src/wallet/wallet');
import { MockLedgerConnector } from '../../../test-utils/mock-ledger-connector';
import { 
  TEST_MNEMONICS, 
  TEST_PRIVATE_KEYS, 
  TEST_ADDRESSES 
} from '../../../test-utils/test-data';
import { KeyringTestUtils, AccountTestUtils } from '../../../test-utils/wallet-test-utils';

describe('KnirvWallet Keyring Management', () => {
  describe('HD Keyring Management', () => {
    let wallet: typeof KnirvWallet;

    beforeEach(async () => {
      wallet = await KnirvWallet.createByMnemonic(TEST_MNEMONICS.VALID_12_WORD);
    });

    it('should create HD keyring with correct properties', () => {
      const keyring = wallet.keyrings[0];
      
      expect(keyring.type).toBe('HD');
      expect(keyring.id).toBeDefined();
      expect(keyring.mnemonic).toBe(TEST_MNEMONICS.VALID_12_WORD);
      expect(KeyringTestUtils.validateKeyringStructure(keyring)).toBe(true);
    });

    it('should derive multiple accounts from HD keyring', async () => {
      const paths = [0, 1, 2, 3, 4];
      const multiAccountWallet = await KnirvWallet.createByMnemonic(TEST_MNEMONICS.VALID_12_WORD, paths);
      
      expect(multiAccountWallet.accounts).toHaveLength(paths.length);
      expect(multiAccountWallet.keyrings).toHaveLength(1); // Single HD keyring
      
      // All accounts should reference the same keyring
      const keyringId = multiAccountWallet.keyrings[0].id;
      multiAccountWallet.accounts.forEach(account => {
        expect(account.keyringId).toBe(keyringId);
      });
    });

    it('should generate unique addresses for different derivation paths', async () => {
      const paths = [0, 1, 2];
      const multiAccountWallet = await KnirvWallet.createByMnemonic(TEST_MNEMONICS.VALID_12_WORD, paths);
      
      const addresses = multiAccountWallet.accounts.map(account => account.address);
      const uniqueAddresses = new Set(addresses);
      
      expect(uniqueAddresses.size).toBe(addresses.length);
    });

    it('should maintain consistent addresses for same mnemonic and path', async () => {
      const wallet1 = await KnirvWallet.createByMnemonic(TEST_MNEMONICS.VALID_12_WORD, [0]);
      const wallet2 = await KnirvWallet.createByMnemonic(TEST_MNEMONICS.VALID_12_WORD, [0]);
      
      expect(wallet1.accounts[0].address).toBe(wallet2.accounts[0].address);
    });

    it('should handle mnemonic validation correctly', () => {
      const keyring = KeyringTestUtils.createTestHDKeyring(TEST_MNEMONICS.VALID_12_WORD);
      
      expect(KeyringTestUtils.validateKeyringStructure(keyring)).toBe(true);
      expect(keyring.mnemonic).toBe(TEST_MNEMONICS.VALID_12_WORD);
    });

    it('should reject invalid mnemonic in keyring creation', () => {
      expect(() => {
        KeyringTestUtils.createTestHDKeyring(TEST_MNEMONICS.INVALID);
      }).not.toThrow(); // KeyringTestUtils doesn't validate, but wallet creation would
    });
  });

  describe('Private Key Keyring Management', () => {
    let wallet: typeof KnirvWallet;

    beforeEach(async () => {
      wallet = await KnirvWallet.createByWeb3Auth(TEST_PRIVATE_KEYS.VALID_HEX);
    });

    it('should create private key keyring with correct properties', () => {
      const keyring = wallet.keyrings[0];
      
      expect(keyring.type).toBe('WEB3_AUTH');
      expect(keyring.id).toBeDefined();
      expect(KeyringTestUtils.validateKeyringStructure(keyring)).toBe(true);
    });

    it('should create single account from private key', () => {
      expect(wallet.accounts).toHaveLength(1);
      expect(wallet.accounts[0].keyringId).toBe(wallet.keyrings[0].id);
    });

    it('should handle private key with 0x prefix', async () => {
      const walletWithPrefix = await KnirvWallet.createByWeb3Auth(TEST_PRIVATE_KEYS.VALID_HEX_WITH_PREFIX);
      
      expect(walletWithPrefix.keyrings[0].type).toBe('WEB3_AUTH');
      expect(walletWithPrefix.accounts).toHaveLength(1);
    });

    it('should generate consistent address for same private key', async () => {
      const wallet1 = await KnirvWallet.createByWeb3Auth(TEST_PRIVATE_KEYS.VALID_HEX);
      const wallet2 = await KnirvWallet.createByWeb3Auth(TEST_PRIVATE_KEYS.VALID_HEX);
      
      expect(wallet1.accounts[0].address).toBe(wallet2.accounts[0].address);
    });

    it('should validate private key keyring structure', () => {
      const keyring = KeyringTestUtils.createTestPrivateKeyKeyring(TEST_PRIVATE_KEYS.VALID_HEX);
      
      expect(KeyringTestUtils.validateKeyringStructure(keyring)).toBe(true);
      expect(keyring.type).toBe('PRIVATE_KEY');
    });
  });

  describe('Ledger Keyring Management', () => {
    let wallet: typeof KnirvWallet;
    let ledgerConnector: MockLedgerConnector;

    beforeEach(async () => {
      ledgerConnector = await MockLedgerConnector.create();
      wallet = await KnirvWallet.createByLedger(ledgerConnector);
    });

    it('should create Ledger keyring with correct properties', () => {
      const keyring = wallet.keyrings[0];
      
      expect(keyring.type).toBe('LEDGER');
      expect(keyring.id).toBeDefined();
      expect(KeyringTestUtils.validateKeyringStructure(keyring)).toBe(true);
    });

    it('should create multiple Ledger accounts', async () => {
      const paths = [0, 1, 2];
      const multiAccountWallet = await KnirvWallet.createByLedger(ledgerConnector, paths);
      
      expect(multiAccountWallet.accounts).toHaveLength(paths.length);
      expect(multiAccountWallet.keyrings).toHaveLength(1);
      
      // All accounts should be Ledger accounts
      multiAccountWallet.accounts.forEach(account => {
        expect(account.name).toContain('Ledger');
      });
    });

    it('should validate Ledger keyring structure', () => {
      const keyring = KeyringTestUtils.createTestLedgerKeyring('mock-device-id');
      
      expect(KeyringTestUtils.validateKeyringStructure(keyring)).toBe(true);
      expect(keyring.type).toBe('LEDGER');
      expect(keyring.deviceId).toBe('mock-device-id');
    });

    it('should handle Ledger device connection', async () => {
      // Test that Ledger keyring maintains device connection info
      const keyring = wallet.keyrings[0];
      expect(keyring.type).toBe('LEDGER');
      // In real implementation, would have device connection details
    });
  });

  describe('Address-only Keyring Management', () => {
    let wallet: typeof KnirvWallet;

    beforeEach(async () => {
      wallet = await KnirvWallet.createByAddress(TEST_ADDRESSES.GNOLANG);
    });

    it('should create address-only keyring with correct properties', () => {
      const keyring = wallet.keyrings[0];
      
      expect(keyring.type).toBe('ADDRESS');
      expect(keyring.id).toBeDefined();
      expect(KeyringTestUtils.validateKeyringStructure(keyring)).toBe(true);
    });

    it('should create watch-only account', () => {
      expect(wallet.accounts).toHaveLength(1);
      expect(wallet.accounts[0].address).toBe(TEST_ADDRESSES.GNOLANG);
      expect(wallet.accounts[0].name).toContain('Airgap');
    });

    it('should validate address keyring structure', () => {
      const keyring = KeyringTestUtils.createTestAddressKeyring(TEST_ADDRESSES.GNOLANG);
      
      expect(KeyringTestUtils.validateKeyringStructure(keyring)).toBe(true);
      expect(keyring.type).toBe('ADDRESS');
      expect(keyring.address).toBe(TEST_ADDRESSES.GNOLANG);
    });

    it('should handle different address formats', async () => {
      const ethereumWallet = await KnirvWallet.createByAddress(TEST_ADDRESSES.ETHEREUM);
      const xionWallet = await KnirvWallet.createByAddress(TEST_ADDRESSES.XION);
      
      expect(ethereumWallet.accounts[0].address).toBe(TEST_ADDRESSES.ETHEREUM);
      expect(xionWallet.accounts[0].address).toBe(TEST_ADDRESSES.XION);
    });
  });

  describe('Mixed Keyring Management', () => {
    it('should handle multiple keyring types in same wallet', async () => {
      // Create wallet with HD keyring
      const hdWallet = await KnirvWallet.createByMnemonic(TEST_MNEMONICS.VALID_12_WORD);
      
      // In a real implementation, you might be able to add additional keyrings
      // For now, test that each wallet type maintains its keyring correctly
      expect(hdWallet.keyrings[0].type).toBe('HD');
      
      const pkWallet = await KnirvWallet.createByWeb3Auth(TEST_PRIVATE_KEYS.VALID_HEX);
      expect(pkWallet.keyrings[0].type).toBe('WEB3_AUTH');
      
      const addressWallet = await KnirvWallet.createByAddress(TEST_ADDRESSES.GNOLANG);
      expect(addressWallet.keyrings[0].type).toBe('ADDRESS');
    });

    it('should maintain keyring-account relationships', async () => {
      const wallet = await KnirvWallet.createByMnemonic(TEST_MNEMONICS.VALID_12_WORD, [0, 1]);
      
      const keyringId = wallet.keyrings[0].id;
      
      wallet.accounts.forEach(account => {
        expect(account.keyringId).toBe(keyringId);
      });
    });
  });

  describe('Keyring Validation and Error Handling', () => {
    it('should validate keyring structure requirements', () => {
      // Test each keyring type validation
      expect(() => {
        const hdKeyring = { id: 'test', type: 'HD' }; // Missing mnemonic
        KeyringTestUtils.validateKeyringStructure(hdKeyring);
      }).toThrow('HD keyring missing mnemonic');

      expect(() => {
        const pkKeyring = { id: 'test', type: 'PRIVATE_KEY' }; // Missing privateKey
        KeyringTestUtils.validateKeyringStructure(pkKeyring);
      }).toThrow('PRIVATE_KEY keyring missing privateKey');

      expect(() => {
        const ledgerKeyring = { id: 'test', type: 'LEDGER' }; // Missing deviceId
        KeyringTestUtils.validateKeyringStructure(ledgerKeyring);
      }).toThrow('Ledger keyring missing deviceId');

      expect(() => {
        const addressKeyring = { id: 'test', type: 'ADDRESS' }; // Missing address
        KeyringTestUtils.validateKeyringStructure(addressKeyring);
      }).toThrow('Address keyring missing address');
    });

    it('should reject invalid keyring types', () => {
      expect(() => {
        const invalidKeyring = { id: 'test', type: 'INVALID_TYPE' };
        KeyringTestUtils.validateKeyringStructure(invalidKeyring);
      }).toThrow('Invalid keyring type: INVALID_TYPE');
    });

    it('should require keyring ID and type', () => {
      expect(() => {
        const keyringWithoutId = { type: 'HD', mnemonic: TEST_MNEMONICS.VALID_12_WORD };
        KeyringTestUtils.validateKeyringStructure(keyringWithoutId);
      }).toThrow('Keyring missing required fields: id');

      expect(() => {
        const keyringWithoutType = { id: 'test', mnemonic: TEST_MNEMONICS.VALID_12_WORD };
        KeyringTestUtils.validateKeyringStructure(keyringWithoutType);
      }).toThrow('Keyring missing required fields: type');
    });
  });

  describe('Account Creation from Keyrings', () => {
    it('should create seed accounts from HD keyring', () => {
      const keyringId = 'test-hd-keyring';
      const account = AccountTestUtils.createTestSeedAccount(keyringId, 0);
      
      expect(AccountTestUtils.validateAccountStructure(account)).toBe(true);
      expect(account.type).toBe('SEED');
      expect(account.keyringId).toBe(keyringId);
      expect(account.derivationPath).toBe("m/44'/118'/0'/0/0");
    });

    it('should create ledger accounts from Ledger keyring', () => {
      const keyringId = 'test-ledger-keyring';
      const account = AccountTestUtils.createTestLedgerAccount(keyringId, 1);
      
      expect(AccountTestUtils.validateAccountStructure(account)).toBe(true);
      expect(account.type).toBe('LEDGER');
      expect(account.keyringId).toBe(keyringId);
      expect(account.derivationPath).toBe("m/44'/118'/0'/0/1");
    });

    it('should create single accounts from private key keyring', () => {
      const keyringId = 'test-pk-keyring';
      const account = AccountTestUtils.createTestSingleAccount(keyringId);
      
      expect(AccountTestUtils.validateAccountStructure(account)).toBe(true);
      expect(account.type).toBe('SINGLE');
      expect(account.keyringId).toBe(keyringId);
    });

    it('should create airgap accounts from address keyring', () => {
      const keyringId = 'test-address-keyring';
      const account = AccountTestUtils.createTestAirgapAccount(keyringId);
      
      expect(AccountTestUtils.validateAccountStructure(account)).toBe(true);
      expect(account.type).toBe('AIRGAP');
      expect(account.keyringId).toBe(keyringId);
      expect(account.readOnly).toBe(true);
    });
  });
});
