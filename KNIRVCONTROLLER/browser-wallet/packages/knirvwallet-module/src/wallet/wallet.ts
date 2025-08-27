// Mock implementation for KnirvWallet
export class KnirvWallet {
  private config: any;
  private isInitialized: boolean = false;
  private accounts: any[] = [];
  private currentAccount: any = null;
  private network: string;

  constructor(config: any = {}) {
    this.config = config;
    this.network = config.network || 'testnet';
  }

  async initialize(): Promise<boolean> {
    this.isInitialized = true;
    return true;
  }

  async createWallet(mnemonic?: string): Promise<any> {
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

  async importWallet(mnemonic: string): Promise<any> {
    return this.createWallet(mnemonic);
  }

  async getAccounts(): Promise<any[]> {
    return this.accounts;
  }

  async getCurrentAccount(): Promise<any> {
    return this.currentAccount;
  }

  async signTransaction(transaction: any): Promise<any> {
    return {
      signature: 'mock-signature',
      signedTransaction: {
        ...transaction,
        signatures: ['mock-signature']
      }
    };
  }

  async sendTransaction(transaction: any): Promise<any> {
    return {
      txHash: 'mock-tx-hash-123',
      success: true,
      gasUsed: 100000,
      gasWanted: 150000
    };
  }

  async getBalance(address: string): Promise<any> {
    return {
      amount: '1000000',
      denom: 'uatom'
    };
  }

  async disconnect(): Promise<void> {
    this.isInitialized = false;
    this.accounts = [];
    this.currentAccount = null;
  }

  isConnected(): boolean {
    return this.isInitialized;
  }
}
