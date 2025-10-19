// db.ts - Database operations

import * as fs from 'fs';
import * as path from 'path';
import { BigchainDB } from './types';
import { Mutex, logger } from './utils';

// Global database variables
export const db: BigchainDB = { blocks: [], assets: {}, transactions: {} };
export const dbMutex = new Mutex(); // Global mutex for database access
export const dbPath: string = path.join(process.cwd(), "blockchain.json");

// Load database from file
export function loadDatabase(): void {
    logger.log(`Loading database from ${dbPath}...`);
    
    try {
        if (fs.existsSync(dbPath)) {
            const data = fs.readFileSync(dbPath, 'utf8');
            const loadedDb = JSON.parse(data);
            
            // Validate and merge the loaded data
            if (loadedDb.blocks && Array.isArray(loadedDb.blocks)) {
                db.blocks = loadedDb.blocks;
            }
            
            if (loadedDb.assets && typeof loadedDb.assets === 'object') {
                db.assets = loadedDb.assets;
            }
            
            if (loadedDb.transactions && typeof loadedDb.transactions === 'object') {
                db.transactions = loadedDb.transactions;
            }
            
            logger.log(`Database loaded successfully: ${db.blocks.length} blocks, ${Object.keys(db.assets).length} assets, ${Object.keys(db.transactions).length} transactions`);
        } else {
            logger.log("Database file not found. Starting with empty database.");
        }
    } catch (error: unknown) {
        if (error instanceof Error) {
            logger.warn(`Failed to load database: ${error.message}. Starting with empty database.`);
        } else {
            logger.warn(`Failed to load database. Starting with empty database.`);
        }
    }
}

// Save database to file
export async function saveDatabase(): Promise<void> {
    logger.log(`Saving database to ${dbPath}...`);
    
    try {
        await dbMutex.lock();
        try {
            const data = JSON.stringify(db, null, 2);
            fs.writeFileSync(dbPath, data, 'utf8');
            logger.log('Database saved successfully');
        } finally {
            dbMutex.unlock();
        }
    } catch (error: unknown) {
        if (error instanceof Error) {
            logger.error(`Failed to save database: ${error.message}`);
        } else {
            logger.error('Failed to save database');
        }
    }
}