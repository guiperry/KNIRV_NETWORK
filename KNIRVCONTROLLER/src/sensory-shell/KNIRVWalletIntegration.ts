import { EventEmitter } from './EventEmitter';

export interface WalletAccount {
  id: string;
  address: string;
  name: string;
  balance: string;
  nrnBalance: string; // Neural Reasoning Network tokens
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

export interface WalletConfig {
  apiBaseUrl: string;
  chainId: string;
  rpcUrl: string;
  enableCrossPlatform: boolean;
  autoConnectMobile: boolean;
  qrCodeTimeout: number;
}

export interface SkillInvocation {
  skillId: string;
  skillName: string;
  nrnCost: string;
  parameters: unknown;
  expectedOutput: unknown;
  timeout: number;
}

export interface NRNBalance {
  available: string;
  staked: string;
  earned: string;
  totalSpent: string;
  lastUpdated: number;
}

export class KNIRVWalletIntegration extends EventEmitter {
  private config: WalletConfig;
  private accounts: Map<string, WalletAccount> = new Map();
  private transactions: Map<string, WalletTransaction> = new Map();
  private currentAccount: WalletAccount | null = null;
  private isConnected: boolean = false;
  private walletService: unknown = null; // Will be injected
  private crossPlatformService: unknown = null; // For mobile integration

  constructor(config: Partial<WalletConfig>) {
    super();
    
    this.config = {
      apiBaseUrl: 'http://localhost:8083/api/v1',
      chainId: 'knirv-mainnet-1',
      rpcUrl: 'https://rpc.knirv.com',
      enableCrossPlatform: true,
      autoConnectMobile: false,
      qrCodeTimeout: 300000, // 5 minutes
      ...config,
    };
  }

  public async initialize(): Promise<void> {
    console.log('Initializing KNIRV Wallet Integration...');

    try {
      // Initialize wallet services
      await this.initializeWalletServices();

      // Load existing accounts
      await this.loadAccounts();

      // Set up cross-platform communication if enabled
      if (this.config.enableCrossPlatform) {
        await this.initializeCrossPlatformService();
      }

      this.isConnected = true;
      this.emit('walletInitialized');
      console.log('KNIRV Wallet Integration initialized successfully');

    } catch (error) {
      console.error('Failed to initialize KNIRV Wallet Integration:', error);
      throw error;
    }
  }

  public async disconnect(): Promise<void> {
    console.log('Disconnecting KNIRV Wallet Integration...');
    
    this.isConnected = false;
    this.currentAccount = null;
    this.accounts.clear();
    this.transactions.clear();

    this.emit('walletDisconnected');
    console.log('KNIRV Wallet Integration disconnected');
  }

  private async initializeWalletServices(): Promise<void> {
    // This would initialize the actual KNIRV wallet services
    // For now, we'll simulate the connection
    console.log('Connecting to KNIRV Wallet services...');
    
    // In a real implementation, this would:
    // 1. Import the KNIRV wallet module
    // 2. Initialize the wallet provider
    // 3. Set up event listeners
    
    await new Promise(resolve => setTimeout(resolve, 1000)); // Simulate connection delay
  }

  private async initializeCrossPlatformService(): Promise<void> {
    // Initialize cross-platform transaction service for mobile integration
    console.log('Initializing cross-platform wallet service...');
    
    // This would import and initialize the CrossPlatformTransactionService
    // from the KNIRV-WALLET agentic-wallet module
  }

  private async loadAccounts(): Promise<void> {
    try {
      // Load accounts from wallet service
      // For now, we'll create mock accounts
      const mockAccounts: WalletAccount[] = [
        {
          id: 'account_1',
          address: 'knirv1abc123def456ghi789jkl012mno345pqr678stu',
          name: 'Main Account',
          balance: '1000.50',
          nrnBalance: '500.25',
          isActive: true,
          keyringType: 'hd',
        },
        {
          id: 'account_2',
          address: 'knirv1xyz987wvu654tsr321qpo098nml765kji432hgf',
          name: 'Secondary Account',
          balance: '250.75',
          nrnBalance: '100.00',
          isActive: false,
          keyringType: 'private',
        },
      ];

      for (const account of mockAccounts) {
        this.accounts.set(account.id, account);
        if (account.isActive) {
          this.currentAccount = account;
        }
      }

      this.emit('accountsLoaded', Array.from(this.accounts.values()));

    } catch (error) {
      console.error('Failed to load accounts:', error);
    }
  }

  public getAccounts(): WalletAccount[] {
    return Array.from(this.accounts.values());
  }

  public getCurrentAccount(): WalletAccount | null {
    return this.currentAccount;
  }

  public async switchAccount(accountId: string): Promise<void> {
    const account = this.accounts.get(accountId);
    if (!account) {
      throw new Error('Account not found');
    }

    // Update active status
    if (this.currentAccount) {
      this.currentAccount.isActive = false;
      this.accounts.set(this.currentAccount.id, this.currentAccount);
    }

    account.isActive = true;
    this.currentAccount = account;
    this.accounts.set(accountId, account);

    this.emit('accountSwitched', account);
    console.log(`Switched to account: ${account.name}`);
  }

  public async getBalance(accountId?: string): Promise<{ balance: string; nrnBalance: string }> {
    const account = accountId ? this.accounts.get(accountId) : this.currentAccount;
    if (!account) {
      throw new Error('Account not found');
    }

    try {
      // In a real implementation, this would query the blockchain
      const response = await fetch(`${this.config.apiBaseUrl}/account/${account.address}/balance`);
      const result = await response.json();

      if (result.success) {
        // Update cached balance
        account.balance = result.data.balance;
        account.nrnBalance = result.data.nrnBalance;
        this.accounts.set(account.id, account);

        return {
          balance: result.data.balance,
          nrnBalance: result.data.nrnBalance,
        };
      } else {
        // Return cached balance if API fails
        return {
          balance: account.balance,
          nrnBalance: account.nrnBalance,
        };
      }

    } catch (error) {
      console.error('Failed to fetch balance:', error);
      return {
        balance: account.balance,
        nrnBalance: account.nrnBalance,
      };
    }
  }

  public async getNRNBalance(accountId?: string): Promise<NRNBalance> {
    const account = accountId ? this.accounts.get(accountId) : this.currentAccount;
    if (!account) {
      throw new Error('Account not found');
    }

    try {
      const response = await fetch(`${this.config.apiBaseUrl}/account/${account.address}/nrn-balance`);
      const result = await response.json();

      if (result.success) {
        return result.data;
      } else {
        throw new Error(result.error || 'Failed to fetch NRN balance');
      }

    } catch (error) {
      console.error('Failed to fetch NRN balance:', error);
      // Return mock data
      return {
        available: account.nrnBalance,
        staked: '0',
        earned: '0',
        totalSpent: '0',
        lastUpdated: Date.now(),
      };
    }
  }

  public async createTransaction(request: TransactionRequest): Promise<string> {
    if (!this.currentAccount) {
      throw new Error('No active account');
    }

    const transactionId = this.generateTransactionId();
    
    const transaction: WalletTransaction = {
      id: transactionId,
      from: request.from || this.currentAccount.address,
      to: request.to,
      amount: request.amount,
      token: request.token,
      status: 'pending',
      timestamp: Date.now(),
      skillId: request.skillId,
      nrnConsumed: request.nrnAmount,
    };

    this.transactions.set(transactionId, transaction);

    try {
      // Validate transaction
      const validation = await this.validateTransaction(request);
      if (!validation.isValid) {
        throw new Error(`Transaction validation failed: ${validation.errors.join(', ')}`);
      }

      // Estimate fees
      const feeEstimate = await this.estimateTransactionFee(request);
      transaction.fee = feeEstimate.estimatedFee;

      // If cross-platform is enabled, initiate mobile signing
      if (this.config.enableCrossPlatform && this.crossPlatformService) {
        await this.initiateCrossPlatformTransaction(transactionId, request);
      } else {
        // Sign transaction locally
        await this.signTransaction(transactionId);
      }

      this.emit('transactionCreated', transaction);
      return transactionId;

    } catch (error) {
      transaction.status = 'failed';
      this.transactions.set(transactionId, transaction);
      console.error('Failed to create transaction:', error);
      throw error;
    }
  }

  public async invokeSkill(skillInvocation: SkillInvocation): Promise<string> {
    if (!this.currentAccount) {
      throw new Error('No active account');
    }

    console.log(`Invoking skill: ${skillInvocation.skillName}`);

    // Check NRN balance
    const nrnBalance = await this.getNRNBalance();
    if (parseFloat(nrnBalance.available) < parseFloat(skillInvocation.nrnCost)) {
      throw new Error('Insufficient NRN balance for skill invocation');
    }

    // Create transaction for skill invocation
    const transactionRequest: TransactionRequest = {
      from: this.currentAccount.address,
      to: 'knirv1skillcontract000000000000000000000000', // Skill contract address
      amount: '0', // No token transfer, only NRN consumption
      nrnAmount: skillInvocation.nrnCost,
      skillId: skillInvocation.skillId,
      memo: JSON.stringify({
        skillName: skillInvocation.skillName,
        parameters: skillInvocation.parameters,
        expectedOutput: skillInvocation.expectedOutput,
      }),
    };

    const transactionId = await this.createTransaction(transactionRequest);

    this.emit('skillInvoked', {
      skillInvocation,
      transactionId,
      timestamp: Date.now(),
    });

    return transactionId;
  }

  public async getTransaction(transactionId: string): Promise<WalletTransaction | null> {
    return this.transactions.get(transactionId) || null;
  }

  public getTransactions(): WalletTransaction[] {
    return Array.from(this.transactions.values()).sort((a, b) => b.timestamp - a.timestamp);
  }

  public async checkTransactionStatus(transactionId: string): Promise<WalletTransaction> {
    const transaction = this.transactions.get(transactionId);
    if (!transaction) {
      throw new Error('Transaction not found');
    }

    if (transaction.status === 'broadcast' && transaction.hash) {
      try {
        const response = await fetch(`${this.config.apiBaseUrl}/transaction/status/${transaction.hash}`);
        const result = await response.json();

        if (result.success) {
          transaction.status = result.data.status;
          if (result.data.gasUsed) {
            transaction.gasUsed = result.data.gasUsed;
          }
          this.transactions.set(transactionId, transaction);
          this.emit('transactionStatusUpdated', transaction);
        }

      } catch (error) {
        console.error('Failed to check transaction status:', error);
      }
    }

    return transaction;
  }

  private async validateTransaction(request: TransactionRequest): Promise<{
    isValid: boolean;
    errors: string[];
  }> {
    const errors: string[] = [];

    if (!request.to) {
      errors.push('Recipient address is required');
    }

    if (!request.amount || parseFloat(request.amount) < 0) {
      errors.push('Amount must be non-negative');
    }

    // Check balance
    if (this.currentAccount) {
      const balance = parseFloat(this.currentAccount.balance);
      const amount = parseFloat(request.amount);
      
      if (amount > balance) {
        errors.push('Insufficient balance');
      }

      // Check NRN balance for skill invocations
      if (request.nrnAmount) {
        const nrnBalance = parseFloat(this.currentAccount.nrnBalance);
        const nrnAmount = parseFloat(request.nrnAmount);
        
        if (nrnAmount > nrnBalance) {
          errors.push('Insufficient NRN balance');
        }
      }
    }

    return {
      isValid: errors.length === 0,
      errors,
    };
  }

  private async estimateTransactionFee(request: TransactionRequest): Promise<{
    gasLimit: string;
    gasPrice: string;
    estimatedFee: string;
  }> {
    try {
      const response = await fetch(`${this.config.apiBaseUrl}/transaction/estimate-fee`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(request),
      });

      const result = await response.json();
      
      if (result.success) {
        return result.data;
      } else {
        throw new Error(result.error);
      }

    } catch (error) {
      console.error('Failed to estimate fee:', error);
      // Return default values
      return {
        gasLimit: '200000',
        gasPrice: '0.025',
        estimatedFee: '0.005',
      };
    }
  }

  private async signTransaction(transactionId: string): Promise<void> {
    const transaction = this.transactions.get(transactionId);
    if (!transaction) {
      throw new Error('Transaction not found');
    }

    try {
      // In a real implementation, this would use the wallet service to sign
      console.log(`Signing transaction ${transactionId}...`);
      
      // Simulate signing delay
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      transaction.status = 'signed';
      transaction.hash = this.generateTransactionHash();
      this.transactions.set(transactionId, transaction);

      // Auto-broadcast if signed successfully
      await this.broadcastTransaction(transactionId);

    } catch (error) {
      transaction.status = 'failed';
      this.transactions.set(transactionId, transaction);
      throw error;
    }
  }

  private async broadcastTransaction(transactionId: string): Promise<void> {
    const transaction = this.transactions.get(transactionId);
    if (!transaction || transaction.status !== 'signed') {
      throw new Error('Transaction not ready for broadcast');
    }

    try {
      console.log(`Broadcasting transaction ${transactionId}...`);
      
      // Simulate broadcast
      await new Promise(resolve => setTimeout(resolve, 500));
      
      transaction.status = 'broadcast';
      this.transactions.set(transactionId, transaction);

      this.emit('transactionBroadcast', transaction);

      // Start monitoring for confirmation
      this.monitorTransactionConfirmation(transactionId);

    } catch (error) {
      transaction.status = 'failed';
      this.transactions.set(transactionId, transaction);
      throw error;
    }
  }

  private async initiateCrossPlatformTransaction(transactionId: string, request: TransactionRequest): Promise<void> {
    // This would use the CrossPlatformTransactionService for mobile signing
    console.log(`Initiating cross-platform transaction ${transactionId}...`);
    
    // Generate QR code for mobile scanning
    this.emit('qrCodeGenerated', {
      transactionId,
      qrData: {
        type: 'transaction',
        id: transactionId,
        request,
        timestamp: Date.now(),
      },
    });
  }

  private monitorTransactionConfirmation(transactionId: string): void {
    const checkConfirmation = async () => {
      try {
        const transaction = await this.checkTransactionStatus(transactionId);
        
        if (transaction.status === 'confirmed') {
          this.emit('transactionConfirmed', transaction);
        } else if (transaction.status === 'failed') {
          this.emit('transactionFailed', transaction);
        } else {
          // Continue monitoring
          setTimeout(checkConfirmation, 5000); // Check every 5 seconds
        }

      } catch (error) {
        console.error('Error monitoring transaction:', error);
      }
    };

    // Start monitoring after a short delay
    setTimeout(checkConfirmation, 2000);
  }

  private generateTransactionId(): string {
    return `tx_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  private generateTransactionHash(): string {
    return `0x${Array.from({ length: 64 }, () => Math.floor(Math.random() * 16).toString(16)).join('')}`;
  }

  public isWalletConnected(): boolean {
    return this.isConnected;
  }

  public getConfig(): WalletConfig {
    return { ...this.config };
  }

  public updateConfig(newConfig: Partial<WalletConfig>): void {
    this.config = { ...this.config, ...newConfig };
    this.emit('configUpdated', this.config);
  }

  public getStatus(): unknown {
    return {
      isConnected: this.isConnected,
      currentAccount: this.currentAccount,
      accountsCount: this.accounts.size,
      transactionsCount: this.transactions.size,
      config: this.config,
    };
  }
}
