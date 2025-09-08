import { api } from './api';

// KNIRVCONTROLLER connection status
export interface ControllerConnectionStatus {
  connected: boolean;
  controllerEndpoint?: string;
  walletLinked: boolean;
  lastSync?: Date;
  error?: string;
}

export interface WalletAccount {
  id: string;
  address: string;
  name: string;
  balance: string;
  nrnBalance: string;
  isActive: boolean;
  keyringType: 'hd' | 'private' | 'ledger' | 'web3auth';
}

export interface TransactionRequest {
  from: string;
  to: string;
  amount: string;
  token?: string;
  memo?: string;
  gasLimit?: string;
  chainId?: string;
  skillId?: string; // For skill invocation transactions
  nrnAmount?: string; // NRN consumption for skills
}

export interface WalletTransaction {
  id: string;
  hash?: string;
  from: string;
  to: string;
  amount: string;
  token?: string;
  status: 'pending' | 'signed' | 'broadcast' | 'confirmed' | 'failed';
  timestamp: number;
  gasUsed?: string;
  fee?: string;
  skillId?: string;
  nrnConsumed?: string;
}

export interface NRNBalance {
  available: string;
  staked: string;
  earned: string;
  totalSpent: string;
  lastUpdated: number;
}

export interface WalletBalance {
  nrnBalance: number;
  usdValue: number;
  change24h: number;
  walletAddress: string;
  assets: CryptoAsset[];
}

export interface CryptoAsset {
  symbol: string;
  name: string;
  price: string;
  change: number;
  amount: string;
  value: string;
  icon: string;
}

export interface SkillInvocation {
  skillId: string;
  skillName: string;
  nrnCost: string;
  parameters: unknown;
  expectedOutput: unknown;
  timeout: number;
}

class WalletService {
  private baseUrl: string;
  private controllerConnectionStatus: ControllerConnectionStatus;

  constructor() {
    this.baseUrl = '/api/v1/wallet';
    this.controllerConnectionStatus = {
      connected: false,
      walletLinked: false
    };
  }

  // KNIRVCONTROLLER Connection Management
  async checkControllerConnection(): Promise<ControllerConnectionStatus> {
    try {
      const response = await api.get('/api/v1/controller/status');
      this.controllerConnectionStatus = {
        connected: response.data.connected || false,
        controllerEndpoint: response.data.endpoint,
        walletLinked: response.data.walletLinked || false,
        lastSync: response.data.lastSync ? new Date(response.data.lastSync) : undefined
      };
    } catch (error) {
      this.controllerConnectionStatus = {
        connected: false,
        walletLinked: false,
        error: error instanceof Error ? error.message : 'Connection check failed'
      };
    }
    return this.controllerConnectionStatus;
  }

  async linkWithController(controllerEndpoint: string): Promise<boolean> {
    try {
      const response = await api.post('/api/v1/controller/link', {
        endpoint: controllerEndpoint
      });

      if (response.data.success) {
        this.controllerConnectionStatus = {
          connected: true,
          controllerEndpoint,
          walletLinked: true,
          lastSync: new Date()
        };
        return true;
      }
      return false;
    } catch (error) {
      this.controllerConnectionStatus.error = error instanceof Error ? error.message : 'Link failed';
      return false;
    }
  }

  getControllerConnectionStatus(): ControllerConnectionStatus {
    return this.controllerConnectionStatus;
  }

  // Check if wallet functionality is available (requires controller link)
  private async ensureWalletAvailable(): Promise<void> {
    const status = await this.checkControllerConnection();
    if (!status.connected || !status.walletLinked) {
      throw new Error('Wallet functionality requires active KNIRVCONTROLLER connection');
    }
  }

  // Wallet Management (requires controller connection)
  async createWallet(name: string): Promise<WalletAccount> {
    await this.ensureWalletAvailable();
    const response = await api.post(`${this.baseUrl}/create`, { name });
    return response.data;
  }

  async getWallets(): Promise<WalletAccount[]> {
    await this.ensureWalletAvailable();
    const response = await api.get(`${this.baseUrl}/accounts`);
    return response.data;
  }

  async getWallet(id: string): Promise<WalletAccount> {
    await this.ensureWalletAvailable();
    const response = await api.get(`${this.baseUrl}/accounts/${id}`);
    return response.data;
  }

  async deleteWallet(id: string): Promise<void> {
    await this.ensureWalletAvailable();
    await api.delete(`${this.baseUrl}/accounts/${id}`);
  }

  // Balance and Assets (requires controller connection)
  async getBalance(): Promise<WalletBalance> {
    await this.ensureWalletAvailable();
    const response = await api.get(`${this.baseUrl}/balance`);
    return response.data;
  }

  async getNRNBalance(): Promise<NRNBalance> {
    await this.ensureWalletAvailable();
    const response = await api.get(`${this.baseUrl}/nrn-balance`);
    return response.data;
  }

  async getAssets(): Promise<CryptoAsset[]> {
    const response = await api.get(`${this.baseUrl}/assets`);
    return response.data;
  }

  // Transactions
  async getTransactions(limit?: number, offset?: number): Promise<WalletTransaction[]> {
    const params = new URLSearchParams();
    if (limit) params.append('limit', limit.toString());
    if (offset) params.append('offset', offset.toString());
    
    const response = await api.get(`${this.baseUrl}/transactions?${params.toString()}`);
    return response.data;
  }

  async getTransaction(id: string): Promise<WalletTransaction> {
    const response = await api.get(`${this.baseUrl}/transactions/${id}`);
    return response.data;
  }

  async sendTransaction(request: TransactionRequest): Promise<WalletTransaction> {
    const response = await api.post(`${this.baseUrl}/send`, request);
    return response.data;
  }

  async signTransaction(transactionData: any): Promise<string> {
    const response = await api.post(`${this.baseUrl}/sign`, transactionData);
    return response.data.signature;
  }

  // NRN Operations
  async transferNRN(to: string, amount: string, memo?: string): Promise<WalletTransaction> {
    const response = await api.post(`${this.baseUrl}/nrn/transfer`, {
      to,
      amount,
      memo
    });
    return response.data;
  }

  async burnNRNForSkill(skillId: string, amount: string): Promise<WalletTransaction> {
    const response = await api.post(`${this.baseUrl}/nrn/burn`, {
      skillId,
      amount
    });
    return response.data;
  }

  async invokeSkill(invocation: SkillInvocation): Promise<any> {
    const response = await api.post(`${this.baseUrl}/skills/invoke`, invocation);
    return response.data;
  }

  // Faucet Operations (for testnet)
  async requestFaucet(address: string, amount?: string): Promise<WalletTransaction> {
    const response = await api.post(`${this.baseUrl}/faucet`, {
      address,
      amount: amount || '100'
    });
    return response.data;
  }

  // Wallet Connection and Authentication
  async connectWallet(type: 'metamask' | 'walletconnect' | 'xion'): Promise<WalletAccount> {
    const response = await api.post(`${this.baseUrl}/connect`, { type });
    return response.data;
  }

  async disconnectWallet(): Promise<void> {
    await api.post(`${this.baseUrl}/disconnect`);
  }

  async isWalletConnected(): Promise<boolean> {
    try {
      const response = await api.get(`${this.baseUrl}/status`);
      return response.data.connected;
    } catch {
      return false;
    }
  }

  // Cross-platform operations
  async generateQRCode(data: any): Promise<string> {
    const response = await api.post(`${this.baseUrl}/qr/generate`, data);
    return response.data.qrCode;
  }

  async scanQRCode(qrData: string): Promise<any> {
    const response = await api.post(`${this.baseUrl}/qr/scan`, { qrData });
    return response.data;
  }

  // Sync with mobile wallet
  async syncWithMobile(deviceId: string): Promise<void> {
    await api.post(`${this.baseUrl}/sync/mobile`, { deviceId });
  }

  // Wallet export/import
  async exportWallet(id: string, password: string): Promise<string> {
    const response = await api.post(`${this.baseUrl}/export`, { id, password });
    return response.data.keystore;
  }

  async importWallet(keystore: string, password: string, name: string): Promise<WalletAccount> {
    const response = await api.post(`${this.baseUrl}/import`, {
      keystore,
      password,
      name
    });
    return response.data;
  }

  // Network operations
  async switchNetwork(chainId: string): Promise<void> {
    await api.post(`${this.baseUrl}/network/switch`, { chainId });
  }

  async addNetwork(networkConfig: any): Promise<void> {
    await api.post(`${this.baseUrl}/network/add`, networkConfig);
  }

  // Utility methods
  async validateAddress(address: string): Promise<boolean> {
    try {
      const response = await api.post(`${this.baseUrl}/validate-address`, { address });
      return response.data.valid;
    } catch {
      return false;
    }
  }

  async estimateGas(transaction: Partial<TransactionRequest>): Promise<string> {
    const response = await api.post(`${this.baseUrl}/estimate-gas`, transaction);
    return response.data.gasEstimate;
  }

  async getGasPrice(): Promise<string> {
    const response = await api.get(`${this.baseUrl}/gas-price`);
    return response.data.gasPrice;
  }

  // Real-time updates
  subscribeToBalanceUpdates(callback: (balance: WalletBalance) => void): () => void {
    // WebSocket subscription for real-time balance updates
    const ws = new WebSocket(`ws://localhost:8081/api/v1/wallet/ws/balance`);
    
    ws.onmessage = (event) => {
      const balance = JSON.parse(event.data);
      callback(balance);
    };

    return () => {
      ws.close();
    };
  }

  subscribeToTransactionUpdates(callback: (transaction: WalletTransaction) => void): () => void {
    // WebSocket subscription for real-time transaction updates
    const ws = new WebSocket(`ws://localhost:8081/api/v1/wallet/ws/transactions`);
    
    ws.onmessage = (event) => {
      const transaction = JSON.parse(event.data);
      callback(transaction);
    };

    return () => {
      ws.close();
    };
  }
}

export const walletService = new WalletService();
export default walletService;
