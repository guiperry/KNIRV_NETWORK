// Mock implementation for KnirvWallet
class KnirvWallet {
  constructor(config = {}) {
    this.config = config;
    this.isInitialized = false;
    this.accounts = [];
    this.currentAccount = null;
    this.network = config.network || 'testnet';
  }

  async initialize() {
    this.isInitialized = true;
    return true;
  }

  async createWallet(mnemonic) {
    const mockAccount = {
      address: 'cosmos1test123',
      publicKey: 'mock-public-key',
      mnemonic: mnemonic || 'test mnemonic words here',
      derivationPath: "m/44'/118'/0'/0/0"
    };
    this.accounts.push(mockAccount);
    this.currentAccount = mockAccount;
    return mockAccount;
  }

  async importWallet(mnemonic) {
    return this.createWallet(mnemonic);
  }

  async getAccounts() {
    return this.accounts;
  }

  async getCurrentAccount() {
    return this.currentAccount;
  }

  async signTransaction(transaction) {
    return {
      signature: 'mock-signature',
      signedTransaction: {
        ...transaction,
        signatures: ['mock-signature']
      }
    };
  }

  async sendTransaction(transaction) {
    return {
      txHash: 'mock-tx-hash-123',
      success: true,
      gasUsed: 100000,
      gasWanted: 150000
    };
  }

  async getBalance(address) {
    return {
      amount: '1000000',
      denom: 'uatom'
    };
  }

  async disconnect() {
    this.isInitialized = false;
    this.accounts = [];
    this.currentAccount = null;
  }

  isConnected() {
    return this.isInitialized;
  }
}

module.exports = { KnirvWallet };
