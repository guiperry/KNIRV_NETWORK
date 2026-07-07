// Wallet-specific Test Utilities for KNIRVWALLET
import { TEST_MNEMONICS, TEST_PRIVATE_KEYS, TEST_ADDRESSES, TEST_WALLET_CONFIGS } from './test-data';
import { MockCryptoProvider, MockStorageProvider } from './mock-services';

// Wallet creation test utilities
export class WalletTestFactory {
  private cryptoProvider: MockCryptoProvider;
  private storageProvider: MockStorageProvider;

  constructor() {
    this.cryptoProvider = new MockCryptoProvider();
    this.storageProvider = new MockStorageProvider();
  }

  async createTestHDWallet(mnemonic?: string) {
    const testMnemonic = mnemonic || TEST_MNEMONICS.VALID_12_WORD;
    
    return {
      type: 'HD',
      mnemonic: testMnemonic,
      accounts: [
        {
          id: 'account-1',
          name: 'Account 1',
          address: await this.cryptoProvider.deriveAddress('mock-pubkey-1'),
          derivationPath: "m/44'/118'/0'/0/0",
          balance: '1000000'
        }
      ],
      keyrings: [
        {
          id: 'keyring-1',
          type: 'HD',
          mnemonic: testMnemonic
        }
      ],
      currentAccountId: 'account-1'
    };
  }

  async createTestPrivateKeyWallet(privateKey?: string) {
    const testPrivateKey = privateKey || TEST_PRIVATE_KEYS.VALID_HEX;
    
    return {
      type: 'PRIVATE_KEY',
      privateKey: testPrivateKey,
      accounts: [
        {
          id: 'account-1',
          name: 'Account 1',
          address: await this.cryptoProvider.deriveAddress('mock-pubkey-1'),
          balance: '1000000'
        }
      ],
      keyrings: [
        {
          id: 'keyring-1',
          type: 'PRIVATE_KEY',
          privateKey: testPrivateKey
        }
      ],
      currentAccountId: 'account-1'
    };
  }

  async createTestLedgerWallet() {
    return {
      type: 'LEDGER',
      accounts: [
        {
          id: 'account-1',
          name: 'Ledger Account 1',
          address: await this.cryptoProvider.deriveAddress('mock-ledger-pubkey-1'),
          derivationPath: "m/44'/118'/0'/0/0",
          balance: '1000000'
        }
      ],
      keyrings: [
        {
          id: 'keyring-1',
          type: 'LEDGER',
          deviceId: 'mock-ledger-device-1'
        }
      ],
      currentAccountId: 'account-1'
    };
  }

  async createTestWeb3AuthWallet(privateKey?: string) {
    const testPrivateKey = privateKey || TEST_PRIVATE_KEYS.VALID_HEX;
    
    return {
      type: 'WEB3_AUTH',
      privateKey: testPrivateKey,
      accounts: [
        {
          id: 'account-1',
          name: 'Web3Auth Account',
          address: await this.cryptoProvider.deriveAddress('mock-web3auth-pubkey-1'),
          balance: '1000000'
        }
      ],
      keyrings: [
        {
          id: 'keyring-1',
          type: 'WEB3_AUTH',
          privateKey: testPrivateKey
        }
      ],
      currentAccountId: 'account-1'
    };
  }

  async createTestAddressOnlyWallet(address?: string) {
    const testAddress = address || TEST_ADDRESSES.GNOLANG;
    
    return {
      type: 'ADDRESS',
      address: testAddress,
      accounts: [
        {
          id: 'account-1',
          name: 'Watch-only Account',
          address: testAddress,
          balance: '1000000',
          readOnly: true
        }
      ],
      keyrings: [
        {
          id: 'keyring-1',
          type: 'ADDRESS',
          address: testAddress
        }
      ],
      currentAccountId: 'account-1'
    };
  }
}

// Transaction testing utilities
interface TransactionOverrides {
  from?: string;
  to?: string;
  amount?: string;
  token?: string;
  memo?: string;
  gasLimit?: string;
  chainId?: string;
}

export class TransactionTestUtils {
  static createTestTransaction(overrides: TransactionOverrides = {}) {
    return {
      from: TEST_ADDRESSES.GNOLANG,
      to: 'g1234567890abcdef1234567890abcdef12345678',
      amount: '1000000',
      token: 'unrn',
      memo: 'Test transaction',
      gasLimit: '200000',
      gasPrice: '0.025unrn',
      ...overrides
    };
  }

  static createTestNRNBurnTransaction(skillId: string, amount: string = '1000000') {
    return {
      from: TEST_ADDRESSES.GNOLANG,
      to: 'g1skillcontractaddress1234567890abcdef123',
      amount,
      token: 'unrn',
      memo: `Burn NRN for skill: ${skillId}`,
      metadata: {
        type: 'skill_invocation',
        skillId,
        parameters: {
          input: 'test input data',
          model: 'CodeT5'
        }
      }
    };
  }

  static createTestXionTransaction(overrides: TransactionOverrides = {}) {
    return {
      from: TEST_ADDRESSES.XION,
      to: 'xion1234567890abcdef1234567890abcdef12345678',
      amount: '1000000',
      token: 'uxion',
      memo: 'XION test transaction',
      gasLimit: '200000',
      gasPrice: '0.025uxion',
      ...overrides
    };
  }

  static validateTransactionStructure(transaction: Record<string, unknown>) {
    const requiredFields = ['from', 'to', 'amount'];
    const missingFields = requiredFields.filter(field => !transaction[field]);
    
    if (missingFields.length > 0) {
      throw new Error(`Transaction missing required fields: ${missingFields.join(', ')}`);
    }

    if (!TEST_ADDRESSES.GNOLANG.startsWith('g') && !TEST_ADDRESSES.XION.startsWith('xion')) {
      throw new Error('Invalid address format');
    }

    if (isNaN(Number(transaction.amount))) {
      throw new Error('Invalid amount format');
    }

    return true;
  }
}

// Keyring testing utilities
export class KeyringTestUtils {
  static createTestHDKeyring(mnemonic?: string) {
    return {
      id: `keyring-${Date.now()}`,
      type: 'HD',
      mnemonic: mnemonic || TEST_MNEMONICS.VALID_12_WORD,
      accounts: [],
      derivationPath: "m/44'/118'/0'/0"
    };
  }

  static createTestPrivateKeyKeyring(privateKey?: string) {
    return {
      id: `keyring-${Date.now()}`,
      type: 'PRIVATE_KEY',
      privateKey: privateKey || TEST_PRIVATE_KEYS.VALID_HEX,
      accounts: []
    };
  }

  static createTestLedgerKeyring(deviceId?: string) {
    return {
      id: `keyring-${Date.now()}`,
      type: 'LEDGER',
      deviceId: deviceId || 'mock-ledger-device',
      accounts: [],
      derivationPath: "m/44'/118'/0'/0"
    };
  }

  static createTestWeb3AuthKeyring(privateKey?: string) {
    return {
      id: `keyring-${Date.now()}`,
      type: 'WEB3_AUTH',
      privateKey: privateKey || TEST_PRIVATE_KEYS.VALID_HEX,
      accounts: []
    };
  }

  static createTestAddressKeyring(address?: string) {
    return {
      id: `keyring-${Date.now()}`,
      type: 'ADDRESS',
      address: address || TEST_ADDRESSES.GNOLANG,
      accounts: []
    };
  }

  static validateKeyringStructure(keyring: Record<string, unknown>) {
    const requiredFields = ['id', 'type'];
    const missingFields = requiredFields.filter(field => !keyring[field]);
    
    if (missingFields.length > 0) {
      throw new Error(`Keyring missing required fields: ${missingFields.join(', ')}`);
    }

    const validTypes = ['HD', 'PRIVATE_KEY', 'LEDGER', 'WEB3_AUTH', 'ADDRESS'];
    if (!validTypes.includes((keyring as any).type)) {
      throw new Error(`Invalid keyring type: ${keyring.type}`);
    }

    // Type-specific validation
    switch (keyring.type) {
      case 'HD':
        if (!keyring.mnemonic) {
          throw new Error('HD keyring missing mnemonic');
        }
        break;
      case 'PRIVATE_KEY':
      case 'WEB3_AUTH':
        if (!keyring.privateKey) {
          throw new Error(`${keyring.type} keyring missing privateKey`);
        }
        break;
      case 'LEDGER':
        if (!keyring.deviceId) {
          throw new Error('Ledger keyring missing deviceId');
        }
        break;
      case 'ADDRESS':
        if (!keyring.address) {
          throw new Error('Address keyring missing address');
        }
        break;
    }

    return true;
  }
}

// Account testing utilities
interface AccountOverrides {
  id?: string;
  name?: string;
  address?: string;
  keyringId?: string;
  balance?: string;
  nrnBalance?: string;
  isActive?: boolean;
}

export class AccountTestUtils {
  static createTestAccount(overrides: AccountOverrides = {}) {
    return {
      id: `account-${Date.now()}`,
      name: 'Test Account',
      address: TEST_ADDRESSES.GNOLANG,
      balance: '1000000',
      keyringId: 'keyring-1',
      derivationPath: "m/44'/118'/0'/0/0",
      ...overrides
    };
  }

  static createTestSeedAccount(keyringId: string, index: number = 0) {
    return {
      id: `seed-account-${index}`,
      name: `Seed Account ${index + 1}`,
      address: `g1${Math.random().toString(36).substr(2, 38)}`,
      balance: '0',
      keyringId,
      derivationPath: `m/44'/118'/0'/0/${index}`,
      type: 'SEED'
    };
  }

  static createTestLedgerAccount(keyringId: string, index: number = 0) {
    return {
      id: `ledger-account-${index}`,
      name: `Ledger Account ${index + 1}`,
      address: `g1${Math.random().toString(36).substr(2, 38)}`,
      balance: '0',
      keyringId,
      derivationPath: `m/44'/118'/0'/0/${index}`,
      type: 'LEDGER'
    };
  }

  static createTestSingleAccount(keyringId: string) {
    return {
      id: 'single-account',
      name: 'Single Account',
      address: `g1${Math.random().toString(36).substr(2, 38)}`,
      balance: '0',
      keyringId,
      type: 'SINGLE'
    };
  }

  static createTestAirgapAccount(keyringId: string) {
    return {
      id: 'airgap-account',
      name: 'Airgap Account',
      address: `g1${Math.random().toString(36).substr(2, 38)}`,
      balance: '0',
      keyringId,
      type: 'AIRGAP',
      readOnly: true
    };
  }

  static validateAccountStructure(account: Record<string, unknown>) {
    const requiredFields = ['id', 'name', 'address', 'keyringId'];
    const missingFields = requiredFields.filter(field => !account[field]);
    
    if (missingFields.length > 0) {
      throw new Error(`Account missing required fields: ${missingFields.join(', ')}`);
    }

    if (!(account as any).address.startsWith('g') && !(account as any).address.startsWith('xion')) {
      throw new Error('Invalid account address format');
    }

    return true;
  }
}

// Wallet serialization testing utilities
export class WalletSerializationTestUtils {
  static async createTestSerializedWallet(password: string = 'test-password') {
    const mockCrypto = new MockCryptoProvider();
    
    const walletData = {
      accounts: [AccountTestUtils.createTestAccount()],
      keyrings: [KeyringTestUtils.createTestHDKeyring()],
      currentAccountId: 'account-1',
      version: '1.0.0',
      timestamp: Date.now()
    };

    const serialized = JSON.stringify(walletData);
    const encrypted = await mockCrypto.encryptData(serialized, password);
    
    return {
      encrypted,
      original: walletData,
      password
    };
  }

  static validateSerializedWallet(serializedWallet: string) {
    if (!serializedWallet || typeof serializedWallet !== 'string') {
      throw new Error('Invalid serialized wallet format');
    }

    // Check if it looks like encrypted data
    if (!serializedWallet.includes('.')) {
      throw new Error('Serialized wallet does not appear to be encrypted');
    }

    return true;
  }

  static async testWalletSerialization(wallet: Record<string, unknown>, password: string) {
    const mockCrypto = new MockCryptoProvider();
    
    // Serialize
    const serialized = JSON.stringify(wallet);
    const encrypted = await mockCrypto.encryptData(serialized, password);
    
    // Deserialize
    const decrypted = await mockCrypto.decryptData(encrypted, password);
    const deserialized = JSON.parse(decrypted);
    
    // Validate round-trip
    expect(deserialized).toEqual(wallet);
    
    return {
      encrypted,
      deserialized
    };
  }
}

// Cross-platform sync testing utilities
export class SyncTestUtils {
  static createTestSyncSession() {
    return {
      sessionId: `sync-${Date.now()}`,
      mobileDeviceId: `mobile-${Math.random().toString(36).substr(2, 9)}`,
      browserInstanceId: `browser-${Math.random().toString(36).substr(2, 9)}`,
      encryptionKey: Math.random().toString(36).substr(2, 32),
      createdAt: Date.now(),
      expiresAt: Date.now() + (24 * 60 * 60 * 1000), // 24 hours
      status: 'active'
    };
  }

  static createTestQRCodeData(sessionId: string, encryptionKey: string) {
    return `knirv://sync?session=${sessionId}&key=${encryptionKey}`;
  }

  static createTestSyncMessage(type: string, data: unknown, sessionId: string) {
    return {
      type,
      sessionId,
      data,
      timestamp: Date.now(),
      messageId: `msg-${Math.random().toString(36).substr(2, 9)}`
    };
  }

  static validateSyncSession(session: Record<string, unknown>) {
    const requiredFields = ['sessionId', 'mobileDeviceId', 'browserInstanceId', 'encryptionKey'];
    const missingFields = requiredFields.filter(field => !session[field]);

    if (missingFields.length > 0) {
      throw new Error(`Sync session missing required fields: ${missingFields.join(', ')}`);
    }

    if ((session as any).expiresAt <= Date.now()) {
      throw new Error('Sync session has expired');
    }

    return true;
  }

  // Method to get test wallet configurations
  static getTestWalletConfigs() {
    return TEST_WALLET_CONFIGS;
  }

  // Method to create wallet with test configuration
  static createWalletWithTestConfig(configIndex: number = 0) {
    const configs = Object.values(TEST_WALLET_CONFIGS);
    if (configIndex >= configs.length) {
      throw new Error(`Test config index ${configIndex} out of range`);
    }
    return configs[configIndex];
  }
}
