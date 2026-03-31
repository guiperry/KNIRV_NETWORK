// Mock implementation for MockLedgerConnector
class MockLedgerConnector {
  constructor() {
    this.isConnected = false;
    this.deviceInfo = {
      model: 'Nano S',
      version: '2.1.0'
    };
  }

  async connect() {
    this.isConnected = true;
    return {
      success: true,
      deviceInfo: this.deviceInfo
    };
  }

  async disconnect() {
    this.isConnected = false;
    return { success: true };
  }

  async getPublicKey(derivationPath = "m/44'/118'/0'/0/0") {
    if (!this.isConnected) {
      throw new Error('Ledger not connected');
    }
    
    return {
      publicKey: 'mock-ledger-public-key',
      address: 'cosmos1ledgertest123',
      derivationPath
    };
  }

  async signTransaction(transaction, derivationPath = "m/44'/118'/0'/0/0") {
    if (!this.isConnected) {
      throw new Error('Ledger not connected');
    }

    return {
      signature: 'mock-ledger-signature',
      signedTransaction: {
        ...transaction,
        signatures: ['mock-ledger-signature']
      }
    };
  }

  async getVersion() {
    if (!this.isConnected) {
      throw new Error('Ledger not connected');
    }
    
    return {
      major: 2,
      minor: 1,
      patch: 0,
      deviceLocked: false
    };
  }

  isDeviceConnected() {
    return this.isConnected;
  }

  async getDeviceInfo() {
    return this.deviceInfo;
  }
}

module.exports = { MockLedgerConnector };
