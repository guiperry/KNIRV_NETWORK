import AsyncStorage from '@react-native-async-storage/async-storage';
import { QRCodeService, TransactionRequest } from './QRCodeService';

export interface TransactionData {
  from: string;
  to: string;
  amount: string;
  token?: string;
  memo?: string;
  gasLimit?: string;
  chainId?: string;
  nonce?: number;
  data?: string;
}

export interface SignedTransaction {
  transaction: TransactionData;
  signature: string;
  hash: string;
  rawTransaction: string;
  timestamp: number;
}

export interface PendingTransaction {
  id: string;
  sessionId: string;
  transaction: TransactionData;
  status: 'pending' | 'signed' | 'broadcast' | 'confirmed' | 'failed';
  createdAt: number;
  signedAt?: number;
  broadcastAt?: number;
  confirmedAt?: number;
  signature?: string;
  hash?: string;
  error?: string;
}

export class CrossPlatformTransactionService {
  private static instance: CrossPlatformTransactionService;
  private qrService: QRCodeService;
  private pendingTransactions: Map<string, PendingTransaction> = new Map();
  private apiBaseUrl: string = 'http://localhost:8083/api/v1';

  private constructor() {
    this.qrService = QRCodeService.getInstance();
    this.loadPendingTransactions();
  }

  public static getInstance(): CrossPlatformTransactionService {
    if (!CrossPlatformTransactionService.instance) {
      CrossPlatformTransactionService.instance = new CrossPlatformTransactionService();
    }
    return CrossPlatformTransactionService.instance;
  }

  /**
   * Initiate a transaction from browser wallet to be signed on mobile
   */
  public async initiateTransactionFromBrowser(
    sessionId: string,
    transaction: TransactionData
  ): Promise<string> {
    const transactionId = this.generateTransactionId();
    
    const pendingTx: PendingTransaction = {
      id: transactionId,
      sessionId,
      transaction,
      status: 'pending',
      createdAt: Date.now(),
    };

    this.pendingTransactions.set(transactionId, pendingTx);
    await this.savePendingTransactions();

    // Generate QR code for mobile scanning
    await this.qrService.generateTransactionRequestQR(
      sessionId,
      transaction.from,
      transaction.to,
      transaction.amount,
      {
        token: transaction.token,
        memo: transaction.memo,
        gasLimit: transaction.gasLimit,
      }
    );

    return transactionId;
  }

  /**
   * Sign a transaction on mobile wallet
   */
  public async signTransactionOnMobile(
    transactionId: string,
    walletAddress: string,
    privateKey?: string
  ): Promise<SignedTransaction> {
    const pendingTx = this.pendingTransactions.get(transactionId);
    if (!pendingTx) {
      throw new Error('Transaction not found');
    }

    if (pendingTx.status !== 'pending') {
      throw new Error(`Transaction is already ${pendingTx.status}`);
    }

    try {
      // Sign the transaction using the backend API
      const signedTx = await this.signTransaction(pendingTx.transaction, walletAddress);
      
      // Update pending transaction
      pendingTx.status = 'signed';
      pendingTx.signedAt = Date.now();
      pendingTx.signature = signedTx.signature;
      pendingTx.hash = signedTx.hash;

      this.pendingTransactions.set(transactionId, pendingTx);
      await this.savePendingTransactions();

      return signedTx;
    } catch (error) {
      pendingTx.status = 'failed';
      pendingTx.error = error.message;
      this.pendingTransactions.set(transactionId, pendingTx);
      await this.savePendingTransactions();
      throw error;
    }
  }

  /**
   * Broadcast a signed transaction
   */
  public async broadcastTransaction(transactionId: string): Promise<string> {
    const pendingTx = this.pendingTransactions.get(transactionId);
    if (!pendingTx) {
      throw new Error('Transaction not found');
    }

    if (pendingTx.status !== 'signed') {
      throw new Error('Transaction must be signed before broadcasting');
    }

    try {
      const response = await fetch(`${this.apiBaseUrl}/transaction/broadcast`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          transaction_id: transactionId,
          signed_transaction: pendingTx.signature,
        }),
      });

      const result = await response.json();
      
      if (!result.success) {
        throw new Error(result.error || 'Failed to broadcast transaction');
      }

      const txHash = result.data.tx_hash;

      // Update pending transaction
      pendingTx.status = 'broadcast';
      pendingTx.broadcastAt = Date.now();
      pendingTx.hash = txHash;

      this.pendingTransactions.set(transactionId, pendingTx);
      await this.savePendingTransactions();

      return txHash;
    } catch (error) {
      pendingTx.status = 'failed';
      pendingTx.error = error.message;
      this.pendingTransactions.set(transactionId, pendingTx);
      await this.savePendingTransactions();
      throw error;
    }
  }

  /**
   * Check transaction confirmation status
   */
  public async checkTransactionStatus(transactionId: string): Promise<PendingTransaction> {
    const pendingTx = this.pendingTransactions.get(transactionId);
    if (!pendingTx) {
      throw new Error('Transaction not found');
    }

    if (pendingTx.status === 'broadcast' && pendingTx.hash) {
      try {
        const response = await fetch(`${this.apiBaseUrl}/transaction/status/${pendingTx.hash}`);
        const result = await response.json();

        if (result.success && result.data.status === 'confirmed') {
          pendingTx.status = 'confirmed';
          pendingTx.confirmedAt = Date.now();
          this.pendingTransactions.set(transactionId, pendingTx);
          await this.savePendingTransactions();
        }
      } catch (error) {
        console.error('Failed to check transaction status:', error);
      }
    }

    return pendingTx;
  }

  /**
   * Get all pending transactions
   */
  public getPendingTransactions(): PendingTransaction[] {
    return Array.from(this.pendingTransactions.values());
  }

  /**
   * Get transaction by ID
   */
  public getTransaction(transactionId: string): PendingTransaction | undefined {
    return this.pendingTransactions.get(transactionId);
  }

  /**
   * Cancel a pending transaction
   */
  public async cancelTransaction(transactionId: string): Promise<void> {
    const pendingTx = this.pendingTransactions.get(transactionId);
    if (!pendingTx) {
      throw new Error('Transaction not found');
    }

    if (pendingTx.status !== 'pending') {
      throw new Error('Can only cancel pending transactions');
    }

    pendingTx.status = 'failed';
    pendingTx.error = 'Cancelled by user';
    this.pendingTransactions.set(transactionId, pendingTx);
    await this.savePendingTransactions();
  }

  /**
   * Clear completed transactions
   */
  public async clearCompletedTransactions(): Promise<void> {
    const completedStatuses = ['confirmed', 'failed'];
    const now = Date.now();
    const maxAge = 24 * 60 * 60 * 1000; // 24 hours

    for (const [id, tx] of this.pendingTransactions.entries()) {
      if (completedStatuses.includes(tx.status) && (now - tx.createdAt) > maxAge) {
        this.pendingTransactions.delete(id);
      }
    }

    await this.savePendingTransactions();
  }

  /**
   * Estimate transaction fee
   */
  public async estimateTransactionFee(transaction: TransactionData): Promise<{
    gasLimit: string;
    gasPrice: string;
    estimatedFee: string;
  }> {
    try {
      const response = await fetch(`${this.apiBaseUrl}/transaction/estimate-fee`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(transaction),
      });

      const result = await response.json();
      
      if (!result.success) {
        throw new Error(result.error || 'Failed to estimate fee');
      }

      return result.data;
    } catch (error) {
      console.error('Failed to estimate transaction fee:', error);
      // Return default values
      return {
        gasLimit: '21000',
        gasPrice: '20000000000', // 20 gwei
        estimatedFee: '0.00042', // 21000 * 20 gwei
      };
    }
  }

  /**
   * Validate transaction data
   */
  public validateTransaction(transaction: TransactionData): {
    isValid: boolean;
    errors: string[];
  } {
    const errors: string[] = [];

    if (!transaction.from) {
      errors.push('From address is required');
    }

    if (!transaction.to) {
      errors.push('To address is required');
    }

    if (!transaction.amount || parseFloat(transaction.amount) <= 0) {
      errors.push('Amount must be greater than 0');
    }

    // Validate address formats (simplified)
    if (transaction.from && !this.isValidAddress(transaction.from)) {
      errors.push('Invalid from address format');
    }

    if (transaction.to && !this.isValidAddress(transaction.to)) {
      errors.push('Invalid to address format');
    }

    return {
      isValid: errors.length === 0,
      errors,
    };
  }

  // Private helper methods

  private async signTransaction(
    transaction: TransactionData,
    walletAddress: string
  ): Promise<SignedTransaction> {
    try {
      const response = await fetch(`${this.apiBaseUrl}/transaction/sign`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          transaction,
          wallet_address: walletAddress,
        }),
      });

      const result = await response.json();
      
      if (!result.success) {
        throw new Error(result.error || 'Failed to sign transaction');
      }

      return {
        transaction,
        signature: result.data.signature,
        hash: result.data.hash,
        rawTransaction: result.data.raw_transaction,
        timestamp: Date.now(),
      };
    } catch (error) {
      throw new Error(`Transaction signing failed: ${error.message}`);
    }
  }

  private generateTransactionId(): string {
    return `tx_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  private isValidAddress(address: string): boolean {
    // Simplified address validation
    if (address.startsWith('0x') && address.length === 42) {
      return true; // Ethereum-style address
    }
    if (address.startsWith('knirv1') && address.length > 20) {
      return true; // KNIRV Network address
    }
    if (address.startsWith('bc1') || address.startsWith('1') || address.startsWith('3')) {
      return true; // Bitcoin address
    }
    return false;
  }

  private async loadPendingTransactions(): Promise<void> {
    try {
      const stored = await AsyncStorage.getItem('pending_transactions');
      if (stored) {
        const transactions = JSON.parse(stored);
        this.pendingTransactions = new Map(Object.entries(transactions));
      }
    } catch (error) {
      console.error('Failed to load pending transactions:', error);
    }
  }

  private async savePendingTransactions(): Promise<void> {
    try {
      const transactions = Object.fromEntries(this.pendingTransactions);
      await AsyncStorage.setItem('pending_transactions', JSON.stringify(transactions));
    } catch (error) {
      console.error('Failed to save pending transactions:', error);
    }
  }
}
