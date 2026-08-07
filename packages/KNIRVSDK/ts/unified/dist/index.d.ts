/**
 * KNIRV SDK - Unified TypeScript/JavaScript SDK for KNIRV Network
 *
 * This package provides comprehensive access to all KNIRV Network services
 * including badges, DVE, treasury, agents, network monitoring, and more.
 */
import { KNIRVClient, type KNIRVClientConfig } from './client';
export { KNIRVClient, type KNIRVClientConfig };
export { BadgeService, DVEService, TreasuryService, AgentService, NetworkService, FactualityService, HealthService, ConfigService, } from './services';
export * from './wallet';
export * from './types';
export * from './errors';
export * from './signing';
export declare function createKNIRVClient(config?: KNIRVClientConfig): KNIRVClient;
export declare function createProductionClient(apiKey?: string): KNIRVClient;
export declare function createTestnetClient(apiKey?: string): KNIRVClient;
export declare function createLocalClient(apiKey?: string): KNIRVClient;
export declare const VERSION = "2.0.0";
//# sourceMappingURL=index.d.ts.map