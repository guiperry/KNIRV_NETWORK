// XION-specific Test Utilities for KNIRVWALLET
import { TEST_XION_CONFIGS, TEST_ADDRESSES } from './test-data';

// XION Meta Account test utilities
export class XionTestUtils {
  static createTestXionConfig(network: 'testnet' | 'mainnet' = 'testnet') {
    return TEST_XION_CONFIGS[network.toUpperCase() as keyof typeof TEST_XION_CONFIGS];
  }

  static createTestMetaAccount(overrides: Record<string, unknown> = {}) {
    return {
      address: TEST_ADDRESSES.XION,
      chainId: 'xion-testnet-1',
      balance: '1000000',
      nrnBalance: '500000',
      gaslessEnabled: true,
      ...overrides
    };
  }

  static createTestNRNContract() {
    return {
      address: 'xion1nrn_contract_test_address',
      name: 'KNIRV Network Token',
      symbol: 'NRN',
      decimals: 6,
      totalSupply: '1000000000000000'
    };
  }

  static createTestFaucetContract() {
    return {
      address: 'xion1faucet_contract_test_address',
      dailyLimit: '10000000',
      perRequestLimit: '1000000',
      cooldownPeriod: 86400 // 24 hours in seconds
    };
  }

  static createTestXionTransaction(overrides: Record<string, unknown> = {}) {
    return {
      from: TEST_ADDRESSES.XION,
      to: 'xion1234567890abcdef1234567890abcdef12345678',
      amount: '1000000',
      denom: 'uxion',
      memo: 'Test XION transaction',
      gasLimit: '200000',
      gasPrice: '0.025uxion',
      gasless: true,
      ...overrides
    };
  }

  static createTestNRNTransferTransaction(to: string, amount: string) {
    return {
      type: 'nrn_transfer',
      from: TEST_ADDRESSES.XION,
      to,
      amount,
      contractAddress: 'xion1nrn_contract_test_address',
      memo: 'NRN token transfer',
      gasless: true
    };
  }

  static createTestNRNBurnTransaction(skillId: string, amount: string) {
    return {
      type: 'nrn_burn_for_skill',
      from: TEST_ADDRESSES.XION,
      amount,
      skillId,
      contractAddress: 'xion1nrn_contract_test_address',
      memo: `Burn NRN for skill: ${skillId}`,
      gasless: true,
      metadata: {
        skillParameters: {
          input: 'test input',
          model: 'CodeT5',
          maxTokens: 100
        }
      }
    };
  }

  static createTestFaucetRequest(address: string, amount: string = '1000000') {
    return {
      type: 'faucet_request',
      recipient: address,
      amount,
      contractAddress: 'xion1faucet_contract_test_address',
      timestamp: Date.now()
    };
  }

  static validateXionAddress(address: string): boolean {
    return address.startsWith('xion1') && address.length === 44;
  }

  static validateXionTransaction(transaction: Record<string, unknown>): boolean {
    const requiredFields = ['from', 'to', 'amount', 'denom'];
    const missingFields = requiredFields.filter(field => !transaction[field]);
    
    if (missingFields.length > 0) {
      throw new Error(`XION transaction missing required fields: ${missingFields.join(', ')}`);
    }

    if (!this.validateXionAddress((transaction as any).from) || !this.validateXionAddress((transaction as any).to)) {
      throw new Error('Invalid XION address format');
    }

    if (isNaN(Number(transaction.amount))) {
      throw new Error('Invalid amount format');
    }

    return true;
  }

  static createTestWalletManager(config?: Record<string, unknown>) {
    const defaultConfig = this.createTestXionConfig();
    
    return {
      config: { ...defaultConfig, ...config },
      wallets: new Map(),
      currentWallet: null,
      
      async createWallet(name: string) {
        const wallet = {
          id: `wallet-${Date.now()}`,
          name,
          address: `xion1${Math.random().toString(36).substr(2, 38)}`,
          balance: '0',
          nrnBalance: '0',
          createdAt: Date.now()
        };
        
        this.wallets.set(wallet.id, wallet);
        this.currentWallet = wallet as any;
        
        return wallet;
      },
      
      async getWallet(id: string) {
        return this.wallets.get(id);
      },
      
      async listWallets() {
        return Array.from(this.wallets.values());
      },
      
      async deleteWallet(id: string) {
        const deleted = this.wallets.delete(id);
        if ((this.currentWallet as any)?.id === id) {
          this.currentWallet = null;
        }
        return deleted;
      }
    };
  }

  static createTestXionMetaAccount(config?: Record<string, unknown>) {
    const defaultConfig = this.createTestXionConfig();
    
    return {
      config: { ...defaultConfig, ...config },
      address: TEST_ADDRESSES.XION,
      balance: '1000000',
      nrnBalance: '500000',
      
      async getNRNBalance() {
        return this.nrnBalance;
      },
      
      async getBalance() {
        return this.balance;
      },
      
      async transferNRN(to: string, amount: string) {
        const txHash = `0x${Math.random().toString(16).substr(2, 64)}`;
        
        // Simulate balance update
        const currentBalance = parseInt(this.nrnBalance);
        const transferAmount = parseInt(amount);
        
        if (currentBalance < transferAmount) {
          throw new Error('Insufficient NRN balance');
        }
        
        this.nrnBalance = (currentBalance - transferAmount).toString();
        
        return {
          txHash,
          blockHeight: Math.floor(Math.random() * 1000000),
          gasUsed: '0' // Gasless transaction
        };
      },
      
      async burnNRNForSkill(skillId: string, amount: string) {
        const txHash = `0x${Math.random().toString(16).substr(2, 64)}`;
        
        // Simulate balance update
        const currentBalance = parseInt(this.nrnBalance);
        const burnAmount = parseInt(amount);
        
        if (currentBalance < burnAmount) {
          throw new Error('Insufficient NRN balance for skill invocation');
        }
        
        this.nrnBalance = (currentBalance - burnAmount).toString();
        
        return {
          txHash,
          skillId,
          blockHeight: Math.floor(Math.random() * 1000000),
          gasUsed: '0' // Gasless transaction
        };
      },
      
      async requestFromFaucet(amount: string = '1000000') {
        const txHash = `0x${Math.random().toString(16).substr(2, 64)}`;
        
        // Simulate balance update
        const currentBalance = parseInt(this.nrnBalance);
        const faucetAmount = parseInt(amount);
        
        this.nrnBalance = (currentBalance + faucetAmount).toString();
        
        return {
          txHash,
          amount,
          blockHeight: Math.floor(Math.random() * 1000000)
        };
      }
    };
  }
}

// XION Mock Client for testing
export class MockXionClient {
  private accounts: Map<string, Record<string, unknown>> = new Map();
  private contracts: Map<string, Record<string, unknown>> = new Map();
  private transactions: Array<Record<string, unknown>> = [];

  constructor() {
    // Initialize with test data
    this.accounts.set(TEST_ADDRESSES.XION, {
      address: TEST_ADDRESSES.XION,
      balance: '1000000',
      nrnBalance: '500000'
    });

    this.contracts.set('xion1nrn_contract_test_address', XionTestUtils.createTestNRNContract());
    this.contracts.set('xion1faucet_contract_test_address', XionTestUtils.createTestFaucetContract());
  }

  async getAccount(address: string) {
    return this.accounts.get(address) || null;
  }

  async getBalance(address: string, denom: string = 'uxion') {
    const account = this.accounts.get(address);
    if (!account) return '0';
    
    return denom === 'uxion' ? account.balance : account.nrnBalance || '0';
  }

  async sendTransaction(transaction: Record<string, unknown>) {
    const txHash = `0x${Math.random().toString(16).substr(2, 64)}`;
    const blockHeight = Math.floor(Math.random() * 1000000);
    
    // Simulate transaction processing
    const processedTx = {
      ...transaction,
      txHash,
      blockHeight,
      timestamp: Date.now(),
      status: 'success'
    };
    
    this.transactions.push(processedTx);
    
    // Update balances if it's a transfer
    if (transaction.type === 'transfer') {
      await this.updateBalanceForTransfer(transaction);
    }
    
    return processedTx;
  }

  async executeContract(contractAddress: string, method: string, params: Record<string, unknown>) {
    const contract = this.contracts.get(contractAddress);
    if (!contract) {
      throw new Error(`Contract not found: ${contractAddress}`);
    }

    const txHash = `0x${Math.random().toString(16).substr(2, 64)}`;
    
    // Simulate contract execution
    const result = {
      txHash,
      contractAddress,
      method,
      params,
      blockHeight: Math.floor(Math.random() * 1000000),
      gasUsed: '0' // Gasless
    };

    this.transactions.push(result);
    
    return result;
  }

  async queryContract(contractAddress: string, query: Record<string, unknown>) {
    const contract = this.contracts.get(contractAddress);
    if (!contract) {
      throw new Error(`Contract not found: ${contractAddress}`);
    }

    // Simulate contract queries
    if (query.balance) {
      return { balance: '500000' };
    }
    
    if (query.token_info) {
      return {
        name: contract.name,
        symbol: contract.symbol,
        decimals: contract.decimals,
        total_supply: contract.totalSupply
      };
    }

    return {};
  }

  getTransactionHistory(address?: string) {
    if (address) {
      return this.transactions.filter(tx => 
        tx.from === address || tx.to === address
      );
    }
    return this.transactions;
  }

  private async updateBalanceForTransfer(transaction: Record<string, unknown>) {
    const fromAccount = this.accounts.get((transaction as any).from);
    const toAccount = this.accounts.get((transaction as any).to);
    
    if (fromAccount) {
      const currentBalance = parseInt((fromAccount as any).balance);
      const transferAmount = parseInt((transaction as any).amount);
      fromAccount.balance = (currentBalance - transferAmount).toString();
    }
    
    if (toAccount) {
      const currentBalance = parseInt((toAccount as any).balance);
      const transferAmount = parseInt((transaction as any).amount);
      toAccount.balance = (currentBalance + transferAmount).toString();
    } else {
      // Create new account for recipient
      this.accounts.set((transaction as any).to, {
        address: transaction.to,
        balance: transaction.amount,
        nrnBalance: '0'
      });
    }
  }

  // Test utilities
  setAccountBalance(address: string, balance: string, nrnBalance?: string) {
    const account = this.accounts.get(address) || { address };
    account.balance = balance;
    if (nrnBalance !== undefined) {
      account.nrnBalance = nrnBalance;
    }
    this.accounts.set(address, account);
  }

  addContract(address: string, contract: Record<string, unknown>) {
    this.contracts.set(address, contract);
  }

  clearTransactionHistory() {
    this.transactions = [];
  }

  reset() {
    this.accounts.clear();
    this.contracts.clear();
    this.transactions = [];
    
    // Re-initialize with test data
    this.accounts.set(TEST_ADDRESSES.XION, {
      address: TEST_ADDRESSES.XION,
      balance: '1000000',
      nrnBalance: '500000'
    });

    this.contracts.set('xion1nrn_contract_test_address', XionTestUtils.createTestNRNContract());
    this.contracts.set('xion1faucet_contract_test_address', XionTestUtils.createTestFaucetContract());
  }
}
