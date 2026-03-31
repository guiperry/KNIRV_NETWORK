// Mock KNIRVWALLET modules for Jest testing

// Mock KnirvWallet class
const KnirvWallet = jest.fn().mockImplementation(() => ({
  createAccount: jest.fn().mockResolvedValue({
    address: 'mock-address-123',
    publicKey: 'mock-public-key',
    privateKey: 'mock-private-key'
  }),
  importAccount: jest.fn().mockResolvedValue({
    address: 'mock-imported-address',
    publicKey: 'mock-imported-public-key'
  }),
  signTransaction: jest.fn().mockResolvedValue({
    signature: 'mock-signature',
    txHash: 'mock-tx-hash'
  }),
  signMessage: jest.fn().mockResolvedValue('mock-message-signature'),
  getBalance: jest.fn().mockResolvedValue('1000.0'),
  sendTransaction: jest.fn().mockResolvedValue({
    txHash: 'mock-tx-hash',
    status: 'success'
  }),
  getTransactionHistory: jest.fn().mockResolvedValue([]),
  exportPrivateKey: jest.fn().mockResolvedValue('mock-exported-private-key'),
  lock: jest.fn(),
  unlock: jest.fn().mockResolvedValue(true),
  isLocked: jest.fn().mockReturnValue(false)
}));

// Mock crypto utilities
const encryptAES = jest.fn().mockResolvedValue('encrypted-data');
const decryptAES = jest.fn().mockResolvedValue('decrypted-data');
const encryptSha256 = jest.fn().mockResolvedValue('hashed-data');
const makeCryptKey = jest.fn().mockResolvedValue('mock-crypt-key');
const sha256 = jest.fn().mockResolvedValue('mock-sha256-hash');

// Mock keyring management
const KeyringManager = jest.fn().mockImplementation(() => ({
  createKeyring: jest.fn().mockResolvedValue('mock-keyring-id'),
  importKeyring: jest.fn().mockResolvedValue('mock-imported-keyring-id'),
  exportKeyring: jest.fn().mockResolvedValue('mock-exported-keyring'),
  deleteKeyring: jest.fn().mockResolvedValue(true),
  listKeyrings: jest.fn().mockResolvedValue([]),
  getKeyring: jest.fn().mockResolvedValue({
    id: 'mock-keyring-id',
    accounts: []
  }),
  addAccount: jest.fn().mockResolvedValue('mock-account-id'),
  removeAccount: jest.fn().mockResolvedValue(true),
  getAccount: jest.fn().mockResolvedValue({
    id: 'mock-account-id',
    address: 'mock-address'
  })
}));

// Mock wallet storage
const WalletStorage = jest.fn().mockImplementation(() => ({
  store: jest.fn().mockResolvedValue(true),
  retrieve: jest.fn().mockResolvedValue(null),
  remove: jest.fn().mockResolvedValue(true),
  clear: jest.fn().mockResolvedValue(true),
  list: jest.fn().mockResolvedValue([])
}));

// Mock transaction utilities
const TransactionBuilder = jest.fn().mockImplementation(() => ({
  buildTransaction: jest.fn().mockResolvedValue({
    to: 'mock-recipient',
    value: '100',
    gas: '21000',
    gasPrice: '20000000000'
  }),
  estimateGas: jest.fn().mockResolvedValue('21000'),
  getGasPrice: jest.fn().mockResolvedValue('20000000000'),
  validateTransaction: jest.fn().mockResolvedValue(true)
}));

// Mock network utilities
const NetworkManager = jest.fn().mockImplementation(() => ({
  connect: jest.fn().mockResolvedValue(true),
  disconnect: jest.fn(),
  isConnected: jest.fn().mockReturnValue(true),
  getNetworkInfo: jest.fn().mockResolvedValue({
    chainId: '1',
    name: 'Mock Network',
    rpcUrl: 'https://mock-rpc.example.com'
  }),
  switchNetwork: jest.fn().mockResolvedValue(true)
}));

// Mock Ledger connector for hardware wallet testing
const MockLedgerConnector = jest.fn().mockImplementation(() => ({
  connect: jest.fn().mockResolvedValue(true),
  disconnect: jest.fn(),
  getPublicKey: jest.fn().mockResolvedValue('mock-ledger-public-key'),
  signTransaction: jest.fn().mockResolvedValue('mock-ledger-signature'),
  isConnected: jest.fn().mockReturnValue(true),
  getDeviceInfo: jest.fn().mockResolvedValue({
    model: 'Nano S',
    version: '1.6.0'
  })
}));

// CommonJS exports for different import patterns
module.exports = {
  KnirvWallet,
  encryptAES,
  decryptAES,
  encryptSha256,
  makeCryptKey,
  sha256,
  KeyringManager,
  WalletStorage,
  TransactionBuilder,
  NetworkManager,
  MockLedgerConnector
};

// Also export individual items for named imports
module.exports.KnirvWallet = KnirvWallet;
module.exports.encryptAES = encryptAES;
module.exports.decryptAES = decryptAES;
module.exports.encryptSha256 = encryptSha256;
module.exports.makeCryptKey = makeCryptKey;
module.exports.sha256 = sha256;
module.exports.KeyringManager = KeyringManager;
module.exports.WalletStorage = WalletStorage;
module.exports.TransactionBuilder = TransactionBuilder;
module.exports.NetworkManager = NetworkManager;
module.exports.MockLedgerConnector = MockLedgerConnector;
