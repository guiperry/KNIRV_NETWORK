/**
 * Wallet Integration Service
 * Integrates with KNIRVWALLET/browser-bridge for wallet functionality
 */

export interface WalletAccount {
  id: string;
  address: string;
  name: string;
  balance: string;
  nrnBalance: string;
  isConnected: boolean;
}

export interface TransactionRequest {
  from: string;
  to: string;
  amount: string;
  memo?: string;
  nrnAmount?: string;
  gasLimit?: number;
  gasPrice?: string;
}

export interface Transaction {
  id: string;
  hash: string;
  from: string;
  to: string;
  amount: string;
  nrnAmount?: string;
  type: 'send' | 'receive';
  status: 'pending' | 'confirmed' | 'failed';
  timestamp: Date;
  blockHeight?: number;
  gasUsed?: number;
  memo?: string;
}

export interface SkillInvocationRequest {
  skillId: string;
  skillName: string;
  nrnCost: string;
  parameters: Record<string, unknown>;
  expectedOutput: Record<string, unknown>;
  timeout: number;
}

export interface WalletConnectionStatus {
  connected: boolean;
  account?: WalletAccount;
  bridgeUrl: string;
  lastSync: Date;
}

export class WalletIntegrationService {
  private currentAccount: WalletAccount | null = null;
  private transactions: Map<string, Transaction> = new Map();
  private bridgeUrl: string;
  private connectionStatus: WalletConnectionStatus;

  constructor() {
    this.bridgeUrl = this.detectBridgeUrl();
    this.connectionStatus = {
      connected: false,
      bridgeUrl: this.bridgeUrl,
      lastSync: new Date()
    };
    this.initializeBridgeConnection();
  }

  /**
   * Detect the KNIRVWALLET browser-bridge URL
   */
  private detectBridgeUrl(): string {
    // Check if we're in a browser environment
    if (typeof window !== 'undefined' && window.location) {
      // Check for local development
      if (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1') {
        return 'http://localhost:3004'; // KNIRVWALLET browser-bridge port
      }
    }

    // Production environment or non-browser environment - use proxy
    return '/wallet';
  }

  /**
   * Initialize connection to KNIRVWALLET browser-bridge
   */
  private async initializeBridgeConnection(): Promise<void> {
    try {
      // Check if bridge is available
      const response = await fetch(`${this.bridgeUrl}/health`, {
        method: 'GET',
        mode: 'cors'
      });

      if (response.ok) {
        console.log('KNIRVWALLET browser-bridge connected');
        await this.syncWalletState();
      } else {
        console.warn('KNIRVWALLET browser-bridge not available, using fallback');
        this.initializeFallbackMode();
      }
    } catch (error) {
      console.warn('Failed to connect to KNIRVWALLET browser-bridge:', error);
      this.initializeFallbackMode();
    }
  }

  /**
   * Initialize fallback mode when bridge is not available
   */
  private initializeFallbackMode(): void {
    // Create a mock account for development/testing
    this.currentAccount = {
      id: 'fallback_account',
      address: 'knirv1fallback...mock',
      name: 'Development Account',
      balance: '1000.00',
      nrnBalance: '500.00',
      isConnected: true
    };

    this.connectionStatus = {
      connected: true,
      account: this.currentAccount,
      bridgeUrl: 'fallback',
      lastSync: new Date()
    };

    console.log('Wallet service running in fallback mode');
  }

  /**
   * Connect to wallet via KNIRVWALLET browser-bridge
   */
  async connectWallet(): Promise<WalletAccount> {
    try {
      const response = await fetch(`${this.bridgeUrl}/api/wallet/connect`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include'
      });

      if (!response.ok) {
        throw new Error(`Wallet connection failed: ${response.statusText}`);
      }

      const accountData = await response.json();
      
      this.currentAccount = {
        id: accountData.id,
        address: accountData.address,
        name: accountData.name || 'KNIRV Account',
        balance: accountData.balance || '0.00',
        nrnBalance: accountData.nrnBalance || '0.00',
        isConnected: true
      };

      this.connectionStatus = {
        connected: true,
        account: this.currentAccount,
        bridgeUrl: this.bridgeUrl,
        lastSync: new Date()
      };

      return this.currentAccount;
    } catch (error) {
      console.error('Wallet connection failed:', error);
      throw new Error(`Failed to connect wallet: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  /**
   * Disconnect wallet
   */
  async disconnectWallet(): Promise<void> {
    try {
      await fetch(`${this.bridgeUrl}/api/wallet/disconnect`, {
        method: 'POST',
        credentials: 'include'
      });
    } catch (error) {
      console.error('Wallet disconnection error:', error);
    }

    this.currentAccount = null;
    this.connectionStatus = {
      connected: false,
      bridgeUrl: this.bridgeUrl,
      lastSync: new Date()
    };
  }

  /**
   * Get current wallet account
   */
  getCurrentAccount(): WalletAccount | null {
    return this.currentAccount;
  }

  /**
   * Get wallet connection status
   */
  getConnectionStatus(): WalletConnectionStatus {
    return this.connectionStatus;
  }

  /**
   * Get account balance
   */
  async getAccountBalance(accountId: string): Promise<{ balance: string; nrnBalance: string }> {
    if (!this.currentAccount || this.currentAccount.id !== accountId) {
      throw new Error('Account not found or not connected');
    }

    try {
      const response = await fetch(`${this.bridgeUrl}/api/wallet/balance/${accountId}`, {
        credentials: 'include'
      });

      if (!response.ok) {
        throw new Error(`Failed to get balance: ${response.statusText}`);
      }

      const balanceData = await response.json();
      
      // Update current account balance
      this.currentAccount.balance = balanceData.balance;
      this.currentAccount.nrnBalance = balanceData.nrnBalance;

      return {
        balance: balanceData.balance,
        nrnBalance: balanceData.nrnBalance
      };
    } catch (error) {
      console.error('Failed to get account balance:', error);
      // Return cached balance if available
      return {
        balance: this.currentAccount.balance,
        nrnBalance: this.currentAccount.nrnBalance
      };
    }
  }

  /**
   * Create a new transaction
   */
  async createTransaction(request: TransactionRequest): Promise<string> {
    if (!this.currentAccount) {
      throw new Error('No wallet connected');
    }

    try {
      const response = await fetch(`${this.bridgeUrl}/api/wallet/transaction`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify(request)
      });

      if (!response.ok) {
        throw new Error(`Transaction creation failed: ${response.statusText}`);
      }

      const result = await response.json();
      
      // Create local transaction record
      const transaction: Transaction = {
        id: result.transactionId,
        hash: result.hash || '',
        type: 'send',
        from: request.from,
        to: request.to,
        amount: request.amount,
        nrnAmount: request.nrnAmount,
        status: 'pending',
        timestamp: new Date(),
        memo: request.memo
      };

      this.transactions.set(transaction.id, transaction);
      return transaction.id;
    } catch (error) {
      console.error('Transaction creation failed:', error);
      throw new Error(`Failed to create transaction: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  /**
   * Invoke a skill with NRN payment
   */
  async invokeSkill(request: SkillInvocationRequest): Promise<string> {
    if (!this.currentAccount) {
      throw new Error('No wallet connected');
    }

    try {
      const response = await fetch(`${this.bridgeUrl}/api/wallet/skill-invoke`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify(request)
      });

      if (!response.ok) {
        throw new Error(`Skill invocation failed: ${response.statusText}`);
      }

      const result = await response.json();
      
      // Create transaction record for skill invocation
      const transaction: Transaction = {
        id: result.transactionId,
        hash: result.hash || '',
        type: 'send',
        from: this.currentAccount.address,
        to: 'skill_network',
        amount: '0',
        nrnAmount: request.nrnCost,
        status: 'pending',
        timestamp: new Date(),
        memo: `Skill invocation: ${request.skillName}`
      };

      this.transactions.set(transaction.id, transaction);
      return transaction.id;
    } catch (error) {
      console.error('Skill invocation failed:', error);
      throw new Error(`Failed to invoke skill: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  /**
   * Check transaction status
   */
  async checkTransactionStatus(transactionId: string): Promise<Transaction | null> {
    const localTransaction = this.transactions.get(transactionId);
    if (!localTransaction) {
      return null;
    }

    try {
      const response = await fetch(`${this.bridgeUrl}/api/wallet/transaction/${transactionId}`, {
        credentials: 'include'
      });

      if (response.ok) {
        const statusData = await response.json();
        
        // Update local transaction
        if (localTransaction) {
          localTransaction.status = statusData.status;
          localTransaction.hash = statusData.hash || localTransaction.hash;
          localTransaction.blockHeight = statusData.blockHeight;
          localTransaction.gasUsed = statusData.gasUsed;

          this.transactions.set(transactionId, localTransaction);
        }
      }
    } catch (error) {
      console.error('Failed to check transaction status:', error);
    }

    return localTransaction;
  }

  /**
   * Get transaction history
   */
  async getTransactionHistory(): Promise<Transaction[]> {
    if (!this.currentAccount) {
      return [];
    }

    try {
      const response = await fetch(`${this.bridgeUrl}/api/wallet/transactions`, {
        credentials: 'include'
      });

      if (response.ok) {
        const transactions = await response.json();
        
        // Update local transaction cache
        transactions.forEach((tx: Transaction) => {
          this.transactions.set(tx.id, tx);
        });
        
        return transactions;
      }
    } catch (error) {
      console.error('Failed to get transaction history:', error);
    }

    // Return local transactions if remote fetch fails
    return Array.from(this.transactions.values());
  }

  /**
   * Sync wallet state with bridge
   */
  private async syncWalletState(): Promise<void> {
    try {
      const response = await fetch(`${this.bridgeUrl}/api/wallet/state`, {
        credentials: 'include'
      });

      if (response.ok) {
        const state = await response.json();
        
        if (state.account) {
          this.currentAccount = state.account;
          this.connectionStatus = {
            connected: true,
            account: this.currentAccount || undefined,
            bridgeUrl: this.bridgeUrl,
            lastSync: new Date()
          };
        }
      }
    } catch (error) {
      console.error('Failed to sync wallet state:', error);
    }
  }

  /**
   * Open KNIRVWALLET interface in new window
   */
  openWalletInterface(): void {
    const walletUrl = `${this.bridgeUrl}/wallet`;
    window.open(walletUrl, 'knirvwallet', 'width=400,height=600,scrollbars=yes,resizable=yes');
  }
}

// Export singleton instance
export const walletIntegrationService = new WalletIntegrationService();
