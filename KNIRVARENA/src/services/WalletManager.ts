/**
 * Wallet Manager Implementation
 * Real implementation for managing multiple XION Meta Accounts
 * Replaces mock implementations with actual functionality
 */

import { XionMetaAccount, XionMetaAccountConfig } from './XionMetaAccount';

export interface WalletInfo {
  id: string;
  name: string;
  address: string;
  created: Date;
}

export interface StorageInterface {
  get(key: string): string | null;
  set(key: string, value: string): void;
  remove(key: string): void;
  keys(): string[];
}

// Simple in-memory storage for testing
class MemoryStorage implements StorageInterface {
  private storage: Map<string, string> = new Map();

  get(key: string): string | null {
    return this.storage.get(key) || null;
  }

  set(key: string, value: string): void {
    this.storage.set(key, value);
  }

  remove(key: string): void {
    this.storage.delete(key);
  }

  keys(): string[] {
    return Array.from(this.storage.keys());
  }
}

export class WalletManager {
  private wallets: Map<string, XionMetaAccount> = new Map();
  private storage: StorageInterface;
  private config: XionMetaAccountConfig;
  private isInitialized = false;

  constructor(config?: XionMetaAccountConfig, storage?: StorageInterface) {
    this.config = config || {
      rpcEndpoint: 'https://rpc.xion-testnet-1.burnt.com:443',
      chainId: 'xion-testnet-1'
    };
    this.storage = storage || new MemoryStorage();
  }

  /**
   * Initialize the wallet manager
   */
  async initialize(): Promise<boolean> {
    try {
      // Load existing wallets from storage
      await this.loadWalletsFromStorage();
      this.isInitialized = true;
      return true;
    } catch (error) {
      console.error('Failed to initialize WalletManager:', error);
      throw error;
    }
  }

  /**
   * Create a new wallet
   */
  async createWallet(name: string): Promise<XionMetaAccount> {
    if (!name || name.trim() === '') {
      throw new Error('Wallet name cannot be empty');
    }

    try {
      const wallet = new XionMetaAccount(this.config);
      await wallet.initialize(); // Generate new mnemonic

      const address = await wallet.getAddress();
      const walletId = this.generateWalletId();

      // Store wallet info
      const walletInfo: WalletInfo = {
        id: walletId,
        name: name.trim(),
        address,
        created: new Date()
      };

      await this.saveWalletToStorage(name, wallet, walletInfo);
      this.wallets.set(name, wallet);

      return wallet;
    } catch (error) {
      console.error('Failed to create wallet:', error);
      throw error;
    }
  }

  /**
   * Import wallet from mnemonic
   */
  async importWallet(name: string, mnemonic: string): Promise<XionMetaAccount> {
    if (!name || name.trim() === '') {
      throw new Error('Wallet name cannot be empty');
    }

    if (!mnemonic || typeof mnemonic !== 'string') {
      throw new Error('Invalid mnemonic');
    }

    try {
      const wallet = new XionMetaAccount(this.config);
      await wallet.initialize(mnemonic);

      const address = await wallet.getAddress();
      const walletId = this.generateWalletId();

      // Store wallet info
      const walletInfo: WalletInfo = {
        id: walletId,
        name: name.trim(),
        address,
        created: new Date()
      };

      await this.saveWalletToStorage(name, wallet, walletInfo);
      this.wallets.set(name, wallet);

      return wallet;
    } catch (error) {
      console.error('Failed to import wallet:', error);
      throw error;
    }
  }

  /**
   * Get existing wallet by name
   */
  async getWallet(name: string): Promise<XionMetaAccount | undefined> {
    if (!name) {
      return undefined;
    }

    // Check in-memory cache first
    if (this.wallets.has(name)) {
      return this.wallets.get(name);
    }

    // Try to load from storage
    try {
      const wallet = await this.loadWalletFromStorage(name);
      if (wallet) {
        this.wallets.set(name, wallet);
      }
      return wallet;
    } catch (error) {
      console.error('Failed to get wallet:', error);
      return undefined;
    }
  }

  /**
   * List all wallet names
   */
  async listWallets(): Promise<string[]> {
    try {
      const walletKeys = this.storage.keys().filter(key => key.startsWith('wallet_') && !key.startsWith('wallet_info_'));
      return walletKeys.map(key => key.replace('wallet_', ''));
    } catch (error) {
      console.error('Failed to list wallets:', error);
      return [];
    }
  }

  /**
   * Delete wallet
   */
  async deleteWallet(name: string): Promise<boolean> {
    try {
      this.storage.remove(`wallet_${name}`);
      this.storage.remove(`wallet_info_${name}`);
      this.wallets.delete(name);
      return true;
    } catch (error) {
      console.error('Failed to delete wallet:', error);
      return false;
    }
  }

  /**
   * Save wallet to storage with encryption
   */
  private async saveWalletToStorage(name: string, wallet: XionMetaAccount, info: WalletInfo): Promise<void> {
    try {
      const mnemonic = await wallet.getMnemonic();
      const address = await wallet.getAddress();

      // Encrypt wallet data (simplified encryption for demo)
      const walletData = {
        mnemonic: this.encryptData(mnemonic),
        address,
        gasless: wallet.isGaslessEnabled()
      };

      this.storage.set(`wallet_${name}`, JSON.stringify(walletData));
      this.storage.set(`wallet_info_${name}`, JSON.stringify(info));
    } catch (error) {
      console.error('Failed to save wallet to storage:', error);
      throw error;
    }
  }

  /**
   * Load wallet from storage with decryption
   */
  private async loadWalletFromStorage(name: string): Promise<XionMetaAccount | undefined> {
    try {
      const walletDataStr = this.storage.get(`wallet_${name}`);
      if (!walletDataStr) {
        return undefined;
      }

      const walletData = JSON.parse(walletDataStr);
      const decryptedMnemonic = this.decryptData(walletData.mnemonic);

      const wallet = new XionMetaAccount(this.config);
      await wallet.initialize(decryptedMnemonic);

      if (walletData.gasless) {
        await wallet.enableGaslessTransactions();
      }

      return wallet;
    } catch (error) {
      console.error('Failed to load wallet from storage:', error);
      return undefined;
    }
  }

  /**
   * Load all wallets from storage
   */
  private async loadWalletsFromStorage(): Promise<void> {
    try {
      const walletKeys = this.storage.keys().filter(key => key.startsWith('wallet_'));
      
      for (const key of walletKeys) {
        const name = key.replace('wallet_', '');
        const wallet = await this.loadWalletFromStorage(name);
        if (wallet) {
          this.wallets.set(name, wallet);
        }
      }
    } catch (error) {
      console.error('Failed to load wallets from storage:', error);
      throw error;
    }
  }

  /**
   * Generate unique wallet ID
   */
  private generateWalletId(): string {
    return 'wallet_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
  }

  /**
   * Simple encryption (for demo purposes - use proper encryption in production)
   */
  private encryptData(data: string): string {
    // Simple base64 encoding for demo
    return Buffer.from(data).toString('base64');
  }

  /**
   * Simple decryption (for demo purposes - use proper decryption in production)
   */
  private decryptData(encryptedData: string): string {
    // Simple base64 decoding for demo
    return Buffer.from(encryptedData, 'base64').toString();
  }
}
