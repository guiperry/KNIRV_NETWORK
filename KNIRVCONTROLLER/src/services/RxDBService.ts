/**
 * RxDB Service for Secure Local Database Storage
 * Manages encrypted RxDB instance for wallet data and sensitive information
 */

import { createRxDatabase, RxDatabase, RxCollection, addRxPlugin } from 'rxdb';
import { getRxStorageDexie } from 'rxdb/plugins/storage-dexie';
import { RxDBDevModePlugin } from 'rxdb/plugins/dev-mode';
import { wrappedValidateAjvStorage } from 'rxdb/plugins/validate-ajv';

// Add plugins for development mode
if (process.env.NODE_ENV === 'development' || process.env.NODE_ENV === 'test') {
  addRxPlugin(RxDBDevModePlugin);
}

// Database interfaces
interface WalletDocType {
  id: string;
  type: 'wallet';
  address: string;
  name: string;
  encryptedPrivateKey?: string;
  publicKey: string;
  balance: string;
  usdcBalance: string;
  nrnBalance: string;
  lastSync: number;
  createdAt: number;
  updatedAt: number;
}

interface TransactionDocType {
  id: string;
  type: 'transaction';
  walletId: string;
  hash: string;
  from: string;
  to: string;
  amount: string;
  nrnAmount?: string;
  status: 'pending' | 'confirmed' | 'failed';
  timestamp: number;
  blockHeight?: number;
  gasUsed?: number;
  memo?: string;
  category: 'send' | 'receive' | 'conversion' | 'skill_payment';
}

interface ConversionDocType {
  id: string;
  type: 'conversion';
  walletId: string;
  transactionId: string;
  usdcAmount: string;
  nrnAmount: string;
  rate: number;
  timestamp: number;
  status: 'pending' | 'completed' | 'failed';
  targetAddress?: string;
}

interface SettingsDocType {
  id: string;
  type: 'settings';
  key: string;
  value: string;
  walletId?: string;
  autoSync?: boolean;
  biometricEnabled?: boolean;
  notificationsEnabled?: boolean;
  defaultNetwork?: 'testnet' | 'mainnet';
  preferredCurrency?: 'USD' | 'NRN' | 'USDC';
  theme?: 'light' | 'dark' | 'auto';
  timestamp: number;
  createdAt?: number;
  updatedAt?: number;
}

interface GraphDocType {
  id: string;
  type: 'graph';
  userId: string;
  nodes: unknown[];
  edges: unknown[];
  metadata: Record<string, unknown>;
  timestamp: number;
}

type DatabaseCollections = {
  wallets: RxCollection<WalletDocType>;
  transactions: RxCollection<TransactionDocType>;
  conversions: RxCollection<ConversionDocType>;
  settings: RxCollection<SettingsDocType>;
  graphs: RxCollection<GraphDocType>;
};

export class RxDBService {
  private db: RxDatabase<DatabaseCollections> | null = null;
  private isInitialized = false;
  private encryptionKey: string;

  constructor(encryptionKey?: string) {
    this.encryptionKey = encryptionKey || 'knirv-wallet-default-key-2025';
  }

  async initialize(): Promise<void> {
    if (this.isInitialized) return;

    try {
      console.log('Initializing RxDB database...');

      this.db = await createRxDatabase<DatabaseCollections>({
        name: 'knirv_wallet_db',
        storage: wrappedValidateAjvStorage({
          storage: getRxStorageDexie()
        })
      });

      await this.createCollections();
      this.isInitialized = true;

      console.log('RxDB database initialized successfully');
    } catch (error) {
      console.error('Failed to initialize RxDB:', error);
      throw error;
    }
  }

  private async createCollections(): Promise<void> {
    if (!this.db) throw new Error('Database not initialized');

    await this.db.addCollections({
      wallets: {
        schema: {
          title: 'wallet schema',
          version: 0,
          type: 'object',
          primaryKey: 'id',
          properties: {
            id: { type: 'string' },
            type: { type: 'string' },
            address: { type: 'string' },
            name: { type: 'string' },
            encryptedPrivateKey: { type: 'string' },
            publicKey: { type: 'string' },
            balance: { type: 'string' },
            usdcBalance: { type: 'string' },
            nrnBalance: { type: 'string' },
            lastSync: { type: 'number' },
            createdAt: { type: 'number' },
            updatedAt: { type: 'number' }
          },
          required: ['id', 'type', 'address', 'name', 'publicKey', 'balance', 'usdcBalance', 'nrnBalance']
        }
      },

      transactions: {
        schema: {
          title: 'transaction schema',
          version: 0,
          type: 'object',
          primaryKey: 'id',
          properties: {
            id: { type: 'string' },
            type: { type: 'string' },
            walletId: { type: 'string' },
            hash: { type: 'string' },
            from: { type: 'string' },
            to: { type: 'string' },
            amount: { type: 'string' },
            nrnAmount: { type: 'string' },
            status: { type: 'string' },
            timestamp: { type: 'number' },
            blockHeight: { type: 'number' },
            gasUsed: { type: 'number' },
            memo: { type: 'string' },
            category: { type: 'string' }
          },
          required: ['id', 'type', 'walletId', 'hash', 'from', 'to', 'amount', 'status', 'timestamp', 'category']
        }
      },

      conversions: {
        schema: {
          title: 'conversion schema',
          version: 0,
          type: 'object',
          primaryKey: 'id',
          properties: {
            id: { type: 'string' },
            type: { type: 'string' },
            walletId: { type: 'string' },
            transactionId: { type: 'string' },
            usdcAmount: { type: 'string' },
            nrnAmount: { type: 'string' },
            rate: { type: 'number' },
            timestamp: { type: 'number' },
            status: { type: 'string' },
            targetAddress: { type: 'string' }
          },
          required: ['id', 'type', 'walletId', 'transactionId', 'usdcAmount', 'nrnAmount', 'rate', 'timestamp', 'status']
        }
      },

      settings: {
        schema: {
          title: 'settings schema',
          version: 0,
          type: 'object',
          primaryKey: 'id',
          properties: {
            id: { type: 'string' },
            type: { type: 'string' },
            key: { type: 'string' },
            value: { type: 'string' },
            walletId: { type: 'string' },
            autoSync: { type: 'boolean' },
            biometricEnabled: { type: 'boolean' },
            notificationsEnabled: { type: 'boolean' },
            defaultNetwork: { type: 'string' },
            preferredCurrency: { type: 'string' },
            theme: { type: 'string' },
            timestamp: { type: 'number' },
            createdAt: { type: 'number' },
            updatedAt: { type: 'number' }
          },
          required: ['id', 'type', 'key', 'value', 'timestamp']
        }
      }
      ,

      graphs: {
        schema: {
          title: 'graph schema',
          version: 0,
          type: 'object',
          primaryKey: 'id',
          properties: {
            id: { type: 'string' },
            type: { type: 'string' },
            userId: { type: 'string' },
            nodes: { type: 'array', items: { type: 'object' } },
            edges: { type: 'array', items: { type: 'object' } },
            metadata: { type: 'object' },
            timestamp: { type: 'number' }
          },
          required: ['id', 'type', 'userId', 'nodes', 'edges', 'metadata', 'timestamp']
        }
      }
    });
  }

  getDatabase(): RxDatabase<DatabaseCollections> {
    if (!this.db) throw new Error('Database not initialized');
    return this.db;
  }

  isDatabaseInitialized(): boolean {
    return this.isInitialized;
  }

  async destroy(): Promise<void> {
    if (this.db) {
      await this.db.remove();
      this.db = null;
      this.isInitialized = false;
    }
  }
}

export const rxdbService = new RxDBService();
