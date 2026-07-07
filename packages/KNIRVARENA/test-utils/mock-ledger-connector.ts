/**
 * Mock Ledger Connector for testing purposes
 */

interface LedgerDeviceInfo {
  model: string;
  version: string;
  serialNumber: string;
  isLocked: boolean;
}

interface LedgerAccount {
  address: string;
  publicKey: string;
  derivationPath: string;
  balance?: string;
}

interface TransactionSignature {
  signature: string;
  publicKey: string;
  address: string;
}

export class MockLedgerConnector {
  private isConnected: boolean = false;
  private deviceInfo: LedgerDeviceInfo | null = null;

  static async create(): Promise<MockLedgerConnector> {
    const connector = new MockLedgerConnector();
    await connector.connect();
    return connector;
  }

  private accounts: LedgerAccount[] = [];

  async connect(): Promise<void> {
    // Simulate connection delay
    await new Promise(resolve => setTimeout(resolve, 100));

    this.isConnected = true;
    this.deviceInfo = {
      version: '2.1.0',
      model: 'Nano S Plus',
      serialNumber: 'MOCK-SERIAL-123',
      isLocked: false
    };

    // Create mock accounts
    this.accounts = [
      {
        address: 'g1mock1ledger1account1address1234567890',
        publicKey: '02' + '0'.repeat(64),
        derivationPath: "m/44'/118'/0'/0/0",

      },
      {
        address: 'g1mock2ledger2account2address1234567890',
        publicKey: '03' + '1'.repeat(64),
        derivationPath: "m/44'/118'/0'/0/1",

      }
    ];
  }

  async disconnect(): Promise<void> {
    this.isConnected = false;
    this.deviceInfo = null;
    this.accounts = [];
  }

  async getAccount(index: number = 0): Promise<LedgerAccount> {
    if (!this.isConnected) {
      throw new Error('Ledger device not connected');
    }

    if (index >= this.accounts.length) {
      throw new Error(`Account index ${index} not found`);
    }

    return this.accounts[index];
  }

  async getAccounts(count: number = 1): Promise<LedgerAccount[]> {
    if (!this.isConnected) {
      throw new Error('Ledger device not connected');
    }

    return this.accounts.slice(0, count);
  }

  async signTransaction(transaction: Record<string, unknown>, derivationPath: string): Promise<TransactionSignature> {
    if (!this.isConnected) {
      throw new Error('Ledger device not connected');
    }

    // Validate transaction structure
    if (!transaction || typeof transaction !== 'object') {
      throw new Error('Invalid transaction provided');
    }

    // Validate derivation path format
    if (!derivationPath || !derivationPath.match(/^m\/\d+'?\/\d+'?\/\d+'?$/)) {
      throw new Error(`Invalid derivation path: ${derivationPath}`);
    }

    // Simulate user confirmation delay
    await new Promise(resolve => setTimeout(resolve, 200));

    return {
      signature: 'mock-ledger-signature-' + Math.random().toString(36).substr(2, 9),
      publicKey: this.accounts[0].publicKey,
      address: this.accounts[0].address
    };
  }

  async getPublicKey(derivationPath: string): Promise<string> {
    if (!this.isConnected) {
      throw new Error('Ledger device not connected');
    }

    const pathIndex = parseInt(derivationPath.split('/').pop() || '0');
    return this.accounts[pathIndex]?.publicKey || this.accounts[0].publicKey;
  }

  async getAddress(derivationPath: string): Promise<string> {
    if (!this.isConnected) {
      throw new Error('Ledger device not connected');
    }

    const pathIndex = parseInt(derivationPath.split('/').pop() || '0');
    return this.accounts[pathIndex]?.address || this.accounts[0].address;
  }

  getDeviceInfo(): LedgerDeviceInfo | null {
    return this.deviceInfo;
  }

  isDeviceConnected(): boolean {
    return this.isConnected;
  }

  // Simulate connection errors for testing
  static async createWithError(): Promise<MockLedgerConnector> {
    throw new Error('Failed to connect to Ledger device');
  }

  // Simulate timeout for testing
  static async createWithTimeout(): Promise<MockLedgerConnector> {
    return new Promise((_, reject) => {
      setTimeout(() => {
        reject(new Error('Connection timeout'));
      }, 5000);
    });
  }
}

// Export for test utilities
export default MockLedgerConnector;
