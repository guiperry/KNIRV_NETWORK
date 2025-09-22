/**
 * RxDB Migration Scripts for Production
 * Handles database schema migrations and data transformations
 */

import { RxDatabase, RxCollection } from 'rxdb';

export interface MigrationStrategy {
  fromVersion: number;
  toVersion: number;
  migrate: (oldDoc: Record<string, unknown>) => Record<string, unknown>;
  description: string;
}

export interface DatabaseMigration {
  version: number;
  collections: {
    [collectionName: string]: {
      schema: Record<string, unknown>;
      migrationStrategies?: { [version: number]: MigrationStrategy };
    };
  };
  description: string;
  timestamp: string;
}

// Migration strategies for different schema versions
export const migrationStrategies: { [collectionName: string]: { [version: number]: MigrationStrategy } } = {
  // Personal Graph migrations
  personal_graphs: {
    1: {
      fromVersion: 0,
      toVersion: 1,
      migrate: (oldDoc: Record<string, unknown>) => {
        // Add new fields for enhanced graph structure
        return {
          ...oldDoc,
          metadata: {
            ...(oldDoc.metadata as Record<string, unknown>),
            version: '1.0.0',
            lastSync: new Date().toISOString(),
            capabilities: []
          }
        };
      },
      description: 'Add metadata versioning and sync tracking'
    },
    2: {
      fromVersion: 1,
      toVersion: 2,
      migrate: (oldDoc: Record<string, unknown>) => {
        // Add KNIRVANA integration fields
        return {
          ...oldDoc,
          knirvana: {
            gameState: null,
            nrvBalance: 0,
            agentDeployments: [],
            collectiveSync: false,
            lastGameSession: null
          }
        };
      },
      description: 'Add KNIRVANA game integration fields'
    }
  },

  // Error nodes migrations
  error_nodes: {
    1: {
      fromVersion: 0,
      toVersion: 1,
      migrate: (oldDoc: Record<string, unknown>) => {
        // Add enhanced error tracking
        return {
          ...oldDoc,
          tracking: {
            attempts: oldDoc.attempts || 0,
            lastAttempt: oldDoc.lastAttempt || null,
            successRate: 0,
            averageResolutionTime: 0,
            difficulty: oldDoc.difficulty || 1
          }
        };
      },
      description: 'Add enhanced error tracking metrics'
    },
    2: {
      fromVersion: 1,
      toVersion: 2,
      migrate: (oldDoc: Record<string, unknown>) => {
        // Add KNIRVANA game integration
        return {
          ...oldDoc,
          game: {
            bounty: (oldDoc.tracking as any)?.difficulty * 10 || 10,
            isBeingSolved: false,
            solverAgent: null,
            progress: 0,
            gameSessionId: null
          }
        };
      },
      description: 'Add KNIRVANA game mechanics'
    }
  },

  // Skill nodes migrations
  skill_nodes: {
    1: {
      fromVersion: 0,
      toVersion: 1,
      migrate: (oldDoc: Record<string, unknown>) => {
        // Add skill proficiency tracking
        return {
          ...oldDoc,
          proficiency: {
            level: oldDoc.proficiency || 0,
            experience: 0,
            lastUsed: null,
            usageCount: 0,
            successRate: 0
          }
        };
      },
      description: 'Add detailed proficiency tracking'
    },
    2: {
      fromVersion: 1,
      toVersion: 2,
      migrate: (oldDoc: Record<string, unknown>) => {
        // Add collective learning integration
        return {
          ...oldDoc,
          collective: {
            isShared: false,
            sharedAt: null,
            collectiveRating: 0,
            collectiveUsage: 0,
            sourceGraph: oldDoc.graphId || null
          }
        };
      },
      description: 'Add collective learning features'
    }
  },

  // Wallet data migrations
  wallet_data: {
    1: {
      fromVersion: 0,
      toVersion: 1,
      migrate: (oldDoc: Record<string, unknown>) => {
        // Add enhanced wallet tracking
        return {
          ...oldDoc,
          security: {
            encryptionVersion: '1.0',
            lastBackup: null,
            backupVerified: false,
            securityLevel: 'standard'
          }
        };
      },
      description: 'Add wallet security enhancements'
    },
    2: {
      fromVersion: 1,
      toVersion: 2,
      migrate: (oldDoc: Record<string, unknown>) => {
        // Add NRV/NRN token tracking
        return {
          ...oldDoc,
          tokens: {
            nrv: {
              balance: 0,
              earned: 0,
              spent: 0,
              lastUpdate: new Date().toISOString()
            },
            nrn: {
              balance: 0,
              purchased: 0,
              spent: 0,
              lastUpdate: new Date().toISOString()
            }
          }
        };
      },
      description: 'Add NRV/NRN token tracking'
    }
  }
};

// Database migration configurations
export const databaseMigrations: DatabaseMigration[] = [
  {
    version: 1,
    collections: {
      personal_graphs: {
        schema: {
          version: 1,
          primaryKey: 'id',
          type: 'object',
          properties: {
            id: { type: 'string', maxLength: 100 },
            userId: { type: 'string' },
            nodes: { type: 'array' },
            edges: { type: 'array' },
            metadata: {
              type: 'object',
              properties: {
                version: { type: 'string' },
                lastSync: { type: 'string' },
                capabilities: { type: 'array' }
              }
            }
          },
          required: ['id', 'userId']
        },
        migrationStrategies: {
          1: migrationStrategies.personal_graphs[1]
        }
      }
    },
    description: 'Initial database schema with basic collections',
    timestamp: '2024-01-01T00:00:00.000Z'
  },
  {
    version: 2,
    collections: {
      personal_graphs: {
        schema: {
          version: 2,
          primaryKey: 'id',
          type: 'object',
          properties: {
            id: { type: 'string', maxLength: 100 },
            userId: { type: 'string' },
            nodes: { type: 'array' },
            edges: { type: 'array' },
            metadata: {
              type: 'object',
              properties: {
                version: { type: 'string' },
                lastSync: { type: 'string' },
                capabilities: { type: 'array' }
              }
            },
            knirvana: {
              type: 'object',
              properties: {
                gameState: { type: ['object', 'null'] },
                nrvBalance: { type: 'number' },
                agentDeployments: { type: 'array' },
                collectiveSync: { type: 'boolean' },
                lastGameSession: { type: ['string', 'null'] }
              }
            }
          },
          required: ['id', 'userId']
        },
        migrationStrategies: {
          1: migrationStrategies.personal_graphs[1],
          2: migrationStrategies.personal_graphs[2]
        }
      },
      error_nodes: {
        schema: {
          version: 2,
          primaryKey: 'id',
          type: 'object',
          properties: {
            id: { type: 'string', maxLength: 100 },
            graphId: { type: 'string' },
            errorType: { type: 'string' },
            message: { type: 'string' },
            tracking: {
              type: 'object',
              properties: {
                attempts: { type: 'number' },
                lastAttempt: { type: ['string', 'null'] },
                successRate: { type: 'number' },
                averageResolutionTime: { type: 'number' },
                difficulty: { type: 'number' }
              }
            },
            game: {
              type: 'object',
              properties: {
                bounty: { type: 'number' },
                isBeingSolved: { type: 'boolean' },
                solverAgent: { type: ['string', 'null'] },
                progress: { type: 'number' },
                gameSessionId: { type: ['string', 'null'] }
              }
            }
          },
          required: ['id', 'graphId', 'errorType']
        },
        migrationStrategies: {
          1: migrationStrategies.error_nodes[1],
          2: migrationStrategies.error_nodes[2]
        }
      }
    },
    description: 'Add KNIRVANA game integration and enhanced tracking',
    timestamp: '2024-02-01T00:00:00.000Z'
  }
];

export class DatabaseMigrationManager {
  private database: RxDatabase;
  private currentVersion: number;

  constructor(database: RxDatabase) {
    this.database = database;
    this.currentVersion = 0;
  }

  async getCurrentVersion(): Promise<number> {
    try {
      // Try to get version from metadata collection
      const metadataCollection = this.database.collections.metadata;
      if (metadataCollection) {
        const versionDoc = await metadataCollection.findOne({ selector: { key: 'database_version' } }).exec();
        return versionDoc ? versionDoc.value : 0;
      }
      return 0;
    } catch {
      console.warn('Could not determine database version, assuming version 0');
      return 0;
    }
  }

  async setCurrentVersion(version: number): Promise<void> {
    try {
      const metadataCollection = this.database.collections.metadata;
      if (metadataCollection) {
        await metadataCollection.upsert({
          id: 'database_version',
          key: 'database_version',
          value: version,
          updatedAt: new Date().toISOString()
        });
      }
    } catch (error) {
      console.error('Failed to set database version:', error);
    }
  }

  async needsMigration(): Promise<boolean> {
    const currentVersion = await this.getCurrentVersion();
    const latestVersion = Math.max(...databaseMigrations.map(m => m.version));
    return currentVersion < latestVersion;
  }

  async runMigrations(): Promise<void> {
    const currentVersion = await this.getCurrentVersion();
    const applicableMigrations = databaseMigrations.filter(m => m.version > currentVersion);

    if (applicableMigrations.length === 0) {
      console.log('Database is up to date, no migrations needed');
      return;
    }

    console.log(`Running ${applicableMigrations.length} database migrations...`);

    for (const migration of applicableMigrations.sort((a, b) => a.version - b.version)) {
      console.log(`Applying migration ${migration.version}: ${migration.description}`);
      
      try {
        await this.applyMigration(migration);
        await this.setCurrentVersion(migration.version);
        console.log(`Migration ${migration.version} completed successfully`);
      } catch (error) {
        console.error(`Migration ${migration.version} failed:`, error);
        throw new Error(`Database migration ${migration.version} failed: ${error}`);
      }
    }

    console.log('All database migrations completed successfully');
  }

  private async applyMigration(migration: DatabaseMigration): Promise<void> {
    // Create backup before migration
    await this.createBackup(migration.version);

    // Apply migration to each collection
    for (const [collectionName, config] of Object.entries(migration.collections)) {
      const collection = this.database.collections[collectionName];
      
      if (!collection) {
        console.warn(`Collection ${collectionName} not found, skipping migration`);
        continue;
      }

      await this.migrateCollection(collection, config);
    }
  }

  private async migrateCollection(collection: RxCollection, config: Record<string, unknown>): Promise<void> {
    const documents = await collection.find().exec();
    
    console.log(`Migrating ${documents.length} documents in collection ${collection.name}`);

    for (const doc of documents) {
      const currentVersion = doc.get('_version') || 0;
      const targetVersion = (config.schema as any).version;

      if (currentVersion < targetVersion) {
        let migratedData = doc.toJSON();

        // Apply migration strategies sequentially
        for (let version = currentVersion + 1; version <= targetVersion; version++) {
          const strategy = (config.migrationStrategies as any)?.[version];
          if (strategy) {
            migratedData = strategy.migrate(migratedData);
            migratedData._version = version;
          }
        }

        // Update the document
        await doc.update({
          $set: migratedData
        });
      }
    }
  }

  private async createBackup(migrationVersion: number): Promise<void> {
    try {
      const backupData: Record<string, unknown[]> = {};
      
      // Export all collections
      for (const [name, collection] of Object.entries(this.database.collections)) {
        const documents = await (collection as RxCollection).find().exec();
        backupData[name] = documents.map(doc => doc.toJSON());
      }

      // Store backup in IndexedDB or localStorage
      const backupKey = `knirv_backup_pre_migration_${migrationVersion}_${Date.now()}`;
      localStorage.setItem(backupKey, JSON.stringify(backupData));
      
      console.log(`Database backup created: ${backupKey}`);
    } catch (error) {
      console.error('Failed to create backup:', error);
      // Don't fail migration due to backup failure, but warn
    }
  }

  async rollbackToBackup(backupKey: string): Promise<void> {
    try {
      const backupData = localStorage.getItem(backupKey);
      if (!backupData) {
        throw new Error(`Backup ${backupKey} not found`);
      }

      const data = JSON.parse(backupData);
      
      // Clear current data and restore from backup
      for (const [collectionName, documents] of Object.entries(data)) {
        const collection = this.database.collections[collectionName];
        if (collection) {
          await collection.remove();
          await collection.bulkInsert(documents as Record<string, unknown>[]);
        }
      }

      console.log(`Database restored from backup: ${backupKey}`);
    } catch (error) {
      console.error('Failed to rollback to backup:', error);
      throw error;
    }
  }

  async cleanupOldBackups(keepCount: number = 5): Promise<void> {
    try {
      const backupKeys = Object.keys(localStorage)
        .filter(key => key.startsWith('knirv_backup_'))
        .sort()
        .reverse();

      if (backupKeys.length > keepCount) {
        const keysToDelete = backupKeys.slice(keepCount);
        keysToDelete.forEach(key => localStorage.removeItem(key));
        console.log(`Cleaned up ${keysToDelete.length} old backups`);
      }
    } catch (error) {
      console.error('Failed to cleanup old backups:', error);
    }
  }
}
