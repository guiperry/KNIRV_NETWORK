/**
 * Unified KNIRV Client - Provides access to all KNIRV Network services
 */

import { KNIRVWallet, WalletResponse } from './wallet';
import {
  KNIRVNetworkInfo,
  Badge,
  SkillBadge,
  CapabilityBadge,
  PropertyBadge,
  DVEEnvironment,
  DVESession,
  TreasuryOperation,
  FaucetRequest,
  Agent,
  AgentWorkflow,
  ConnectivityProof,
  NetworkRoute,
  FactualitySlice,
  FactualityVerification,
  ServiceHealth,
  NetworkHealth,
  NRNToken,
  XIONMetaAccount,
  TreasuryContract
} from './types';

export interface KNIRVClientConfig {
  // Network configuration
  network?: {
    environment?: 'public-testnet' | 'public-production' | 'local-testnet' | 'local-production';
    customEndpoints?: {
      controller?: string;
      router?: string;
      graph?: string;
      chain?: string;
      oracle?: string;
      nexus?: string;
      gateway?: string;
    };
  };

  // Gateway configuration
  gateway?: {
    baseURL?: string;
    economicsURL?: string;
    apiKey?: string;
    timeout?: number;
    retries?: number;
  };

  // Authentication configuration
  auth?: {
    apiKey?: string;
    xionMetaAccount?: XIONMetaAccount;
    treasuryContract?: string;
  };

  // Wallet configuration
  wallet?: {
    provider?: string;
    chainId?: string;
    enableGasless?: boolean;
  };

  // Debug configuration
  debug?: boolean;
  verbose?: boolean;
}

import {
  BadgeService,
  DVEService,
  TreasuryService,
  AgentService,
  NetworkService,
  FactualityService,
  HealthService,
  ConfigService,
} from './services';

/**
 * Unified KNIRV Client providing access to all KNIRV Network services
 */
export class KNIRVClient {
  public readonly wallet: KNIRVWallet;
  public readonly badges: BadgeService;
  public readonly dve: DVEService;
  public readonly treasury: TreasuryService;
  public readonly agents: AgentService;
  public readonly network: NetworkService;
  public readonly factuality: FactualityService;
  public readonly health: HealthService;
  public readonly config: ConfigService;

  private readonly networkConfig: KNIRVNetworkInfo;

  constructor(config: KNIRVClientConfig = {}) {
    // Set up network configuration
    this.networkConfig = this.getNetworkConfig(config.network?.environment || 'public-production');

    // Override with custom endpoints if provided
    if (config.network?.customEndpoints) {
      Object.assign(this.networkConfig.services, config.network.customEndpoints);
    }

    // Initialize wallet
    this.wallet = new KNIRVWallet(config.wallet);

    // Initialize all services with appropriate endpoints
    const apiKey = config.auth?.apiKey || config.gateway?.apiKey;
    const timeout = config.gateway?.timeout || 30000;

    this.badges = new BadgeService(this.networkConfig.services.oracle, apiKey, timeout);
    this.dve = new DVEService(this.networkConfig.services.nexus, apiKey, timeout);
    this.treasury = new TreasuryService(this.networkConfig.services.oracle, apiKey, timeout);
    this.agents = new AgentService(this.networkConfig.services.controller, apiKey, timeout);
    this.network = new NetworkService(this.networkConfig.services.router, apiKey, timeout);
    this.factuality = new FactualityService(this.networkConfig.services.controller, apiKey, timeout);
    this.health = new HealthService(this.networkConfig.services.gateway, apiKey, timeout);
    this.config = new ConfigService(this.networkConfig.services.gateway, apiKey, timeout);
  }
  
  /**
   * Create a client optimized for gateway operations (temporarily disabled)
   */
  // static createGatewayClient(config?: KNIRVClientConfig['gateway']) {
  //   return new KNIRVGatewayClient(config);
  // }
  
  /**
   * Create a client optimized for transaction operations (temporarily disabled)
   */
  // static createTransactionClient(config?: KNIRVClientConfig['transaction']) {
  //   return new KnirvchainTransactionSDK(config);
  // }
  
  /**
   * Create a client optimized for transmission operations (temporarily disabled)
   */
  // static createTransmissionClient(config?: KNIRVClientConfig['transmission']) {
  //   return new KnirvClient(config || {
  //     bootstrapPeers: [],
  //     enableDHT: true,
  //     port: 0
  //   });
  // }
  
  /**
   * Get network configuration for the specified environment
   */
  private getNetworkConfig(environment: string): KNIRVNetworkInfo {
    const configs: Record<string, KNIRVNetworkInfo> = {
      'public-production': {
        chainId: 'knirv-1',
        networkName: 'KNIRV Production Network',
        rpcUrl: 'https://rpc.knirv.com',
        currency: { name: 'NRN', symbol: 'NRN', decimals: 18 },
        environment: 'public-production',
        services: {
          controller: 'https://controller.knirv.com',
          router: 'https://router.knirv.com',
          graph: 'https://graph.knirv.com',
          chain: 'https://chain.knirv.com',
          oracle: 'https://oracle.knirv.com',
          nexus: 'https://nexus.knirv.com',
          gateway: 'https://gateway.knirv.com',
        },
      },
      'public-testnet': {
        chainId: 'knirv-testnet-1',
        networkName: 'KNIRV Testnet',
        rpcUrl: 'https://testnet-rpc.knirv.com',
        currency: { name: 'NRN', symbol: 'NRN', decimals: 18 },
        environment: 'public-testnet',
        services: {
          controller: 'https://testnet-controller.knirv.com',
          router: 'https://testnet-router.knirv.com',
          graph: 'https://testnet-graph.knirv.com',
          chain: 'https://testnet-chain.knirv.com',
          oracle: 'https://testnet-oracle.knirv.com',
          nexus: 'https://testnet-nexus.knirv.com',
          gateway: 'https://testnet-gateway.knirv.com',
        },
      },
      'local-testnet': {
        chainId: 'knirv-local-testnet',
        networkName: 'KNIRV Local Testnet',
        rpcUrl: 'http://localhost:26657',
        currency: { name: 'NRN', symbol: 'NRN', decimals: 18 },
        environment: 'local-testnet',
        services: {
          controller: 'http://localhost:3000',
          router: 'http://localhost:8085',
          graph: 'http://localhost:8081',
          chain: 'http://localhost:8080',
          oracle: 'http://localhost:8086',
          nexus: 'http://localhost:8090',
          gateway: 'http://localhost:8087',
        },
      },
      'local-production': {
        chainId: 'knirv-local-production',
        networkName: 'KNIRV Local Production',
        rpcUrl: 'http://localhost:26657',
        currency: { name: 'NRN', symbol: 'NRN', decimals: 18 },
        environment: 'local-production',
        services: {
          controller: 'http://localhost:3000',
          router: 'http://localhost:8085',
          graph: 'http://localhost:8081',
          chain: 'http://localhost:8080',
          oracle: 'http://localhost:8086',
          nexus: 'http://localhost:8090',
          gateway: 'http://localhost:8087',
        },
      },
    };

    return configs[environment] || configs['public-production'];
  }

  /**
   * Get the current network configuration
   */
  getNetworkInfo(): KNIRVNetworkInfo {
    return this.networkConfig;
  }

  /**
   * Switch to a different network environment
   */
  async switchNetwork(environment: string): Promise<KNIRVNetworkInfo> {
    const newConfig = this.getNetworkConfig(environment);
    Object.assign(this.networkConfig, newConfig);
    return this.networkConfig;
  }

  /**
   * Get the current wallet instance
   */
  getWallet(): KNIRVWallet {
    return this.wallet;
  }

  /**
   * Check if the client is properly configured and connected
   */
  async isConnected(): Promise<boolean> {
    try {
      const healthStatus = await this.health.getNetworkHealth();
      return healthStatus.overall === 'healthy' || healthStatus.overall === 'degraded';
    } catch (error) {
      return false;
    }
  }

  /**
   * Get version information for the SDK
   */
  getVersion(): string {
    return '2.0.0';
  }

  /**
   * Health check for all services
   */
  async healthCheck() {
    try {
      return await this.health.getNetworkHealth();
    } catch (error) {
      console.warn('Network health check failed:', error);
      return {
        overall: 'unhealthy' as const,
        services: {},
        timestamp: new Date().toISOString(),
        summary: { total: 0, healthy: 0, unhealthy: 0 }
      };
    }
  }

  /**
   * Clean up resources and close connections
   */
  async close(): Promise<void> {
    // Clean up any open connections or resources
    // Implementation would depend on the specific services being used
  }
}
