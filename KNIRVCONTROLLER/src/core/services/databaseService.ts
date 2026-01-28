import { knirvbaseService } from '../../services/KNIRVBASEService';

/**
 * Initialize database service
 */
export async function initializeDatabase(): Promise<void> {
  try {
    await knirvbaseService.initialize({
      dataDir: './data/knirvcontroller',
      distributedEnabled: false // Can be enabled later for distributed features
    });
    console.log('KNIRVBASE database service initialized successfully');
  } catch (error) {
    console.error('Failed to initialize database service:', error);
    throw error;
  }
}

/**
 * Get database instance
 */
export function getDatabase() {
  return knirvbaseService.getDatabase();
}

/**
 * Database connection status
 */
export function isDatabaseConnected(): boolean {
  return knirvbaseService.isInitialized();
}

/**
 * Close database connection
 */
export async function closeDatabase(): Promise<void> {
  await knirvbaseService.shutdown();
}

/**
 * Export the KNIRVBASE service as the default database service
 */
export const databaseService = knirvbaseService;
export default knirvbaseService;