/**
 * KNIRV SDK - Unified TypeScript/JavaScript SDK for KNIRV Network
 *
 * This package provides wallet functionality compatible with existing wallet implementations
 * and re-exports essential blockchain functionality.
 */

// Re-export the main client
export { KNIRVClient, type KNIRVClientConfig } from './client';

// Wallet functionality exports (main compatibility layer)
export * from './wallet';

// Common types and utilities
export * from './types';
export * from './errors';

// Version information
export const VERSION = '1.0.0';
