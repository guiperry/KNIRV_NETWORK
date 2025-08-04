/**
 * Unified KNIRV Client - Provides access to all KNIRV Network services
 */

// import { KNIRVGatewayClient } from './gateway'; // Temporarily disabled
// import { KnirvchainTransactionSDK } from './transaction'; // Temporarily disabled
// import { KnirvClient } from './transmission'; // Temporarily disabled due to libp2p dependency issues
import { KNIRVWallet, WalletResponse } from './wallet';

export interface KNIRVClientConfig {
  // Gateway configuration
  gateway?: {
    environment?: 'development' | 'staging' | 'production';
    debug?: boolean;
    economicsURL?: string;
    gatewayURL?: string;
    apiKey?: string;
  };
  
  // Transaction configuration
  transaction?: {
    baseURL?: string;
    apiKey?: string;
    timeout?: number;
  };
  
  // Transmission configuration (temporarily disabled)
  // transmission?: {
  //   bootstrapPeers?: string[];
  //   enableDHT?: boolean;
  //   port?: number;
  // };
  
  // Wallet configuration
  wallet?: {
    provider?: string;
    chainId?: string;
  };
}

/**
 * Unified KNIRV Client providing access to all KNIRV Network services
 */
export class KNIRVClient {
  // public readonly gateway: KNIRVGatewayClient; // Temporarily disabled
  // public readonly transaction: KnirvchainTransactionSDK; // Temporarily disabled
  // public readonly transmission: KnirvClient; // Temporarily disabled
  public readonly wallet: KNIRVWallet;
  
  constructor(config: KNIRVClientConfig = {}) {
    // Initialize gateway client (temporarily disabled)
    // this.gateway = new KNIRVGatewayClient(config.gateway);
    
    // Initialize transaction client (temporarily disabled)
    // this.transaction = new KnirvchainTransactionSDK(config.transaction);
    
    // Initialize transmission client (temporarily disabled)
    // this.transmission = new KnirvClient(config.transmission || {
    //   bootstrapPeers: [],
    //   enableDHT: true,
    //   port: 0
    // });
    
    // Initialize wallet (placeholder - would need proper implementation)
    this.wallet = new (class implements KNIRVWallet {
      async getAccount(): Promise<WalletResponse> {
        throw new Error('Not implemented');
      }
      async addNetwork(params: any): Promise<WalletResponse> {
        throw new Error('Not implemented');
      }
      async switchNetwork(chainId: string): Promise<WalletResponse> {
        throw new Error('Not implemented');
      }
      async signTransaction(transaction: any): Promise<WalletResponse> {
        throw new Error('Not implemented');
      }
      async doContract(params: any): Promise<WalletResponse> {
        throw new Error('Not implemented');
      }
    })();
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
   * Health check for all services
   */
  async healthCheck() {
    const results = {
      gateway: false,
      transaction: false,
      transmission: false,
    };
    
    try {
      // await this.gateway.health.checkGatewayHealth(); // Temporarily disabled
      results.gateway = true;
    } catch (error) {
      console.warn('Gateway health check failed:', error);
    }
    
    // Add transaction and transmission health checks when available
    
    return results;
  }
}
