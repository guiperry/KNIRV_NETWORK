/**
 * KNIRV SDK - Unified TypeScript/JavaScript SDK for KNIRV Network
 *
 * This package provides comprehensive access to all KNIRV Network services
 * including badges, DVE, treasury, agents, network monitoring, and more.
 */

// Re-export the main client
export { KNIRVClient, type KNIRVClientConfig } from './client';

// Service classes
export {
  BadgeService,
  DVEService,
  TreasuryService,
  AgentService,
  NetworkService,
  FactualityService,
  HealthService,
  ConfigService,
} from './services';

// Wallet functionality exports (main compatibility layer)
export * from './wallet';

// Common types and utilities
export * from './types';
export * from './errors';

// Convenience factory functions
export function createKNIRVClient(config?: KNIRVClientConfig): KNIRVClient {
  return new KNIRVClient(config);
}

export function createProductionClient(apiKey?: string): KNIRVClient {
  return new KNIRVClient({
    network: { environment: 'public-production' },
    auth: { apiKey },
  });
}

export function createTestnetClient(apiKey?: string): KNIRVClient {
  return new KNIRVClient({
    network: { environment: 'public-testnet' },
    auth: { apiKey },
  });
}

export function createLocalClient(apiKey?: string): KNIRVClient {
  return new KNIRVClient({
    network: { environment: 'local-testnet' },
    auth: { apiKey },
  });
}

// Version information
export const VERSION = '2.0.0';
