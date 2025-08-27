// Mock implementation for MockLedgerConnector
export class MockLedgerConnector {
  private isConnected: boolean = false;
  private deviceInfo: any;

  constructor() {
    this.deviceInfo = {
      model: 'Nano S',
      version: '2.1.0'
    };
  }

  async connect(): Promise<any> {
    this.isConnected = true;
    return {
      success: true,
      deviceInfo: this.deviceInfo
    };
  }

  async disconnect(): Promise<any> {
    this.isConnected = false;
    return { success: true };
  }

  async getPublicKey(derivationPath: string = "m/44'/118'/0'/0/0"): Promise<any> {
    if (!this.isConnected) {
      throw new Error('Ledger not connected');
    }
    
    return {
      publicKey: 'mock-ledger-public-key',
      address: 'cosmos1ledgertest123',
      derivationPath
    };
  }

  async signTransaction(transaction: any, derivationPath: string = "m/44'/118'/0'/0/0"): Promise<any> {
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

  async getVersion(): Promise<any> {
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

  isDeviceConnected(): boolean {
    return this.isConnected;
  }

  async getDeviceInfo(): Promise<any> {
    return this.deviceInfo;
  }
}
