import axios, { AxiosInstance, AxiosResponse } from 'axios';
import {
  KNIRVOracleConfig,
  AgentMintRequest,
  AgentMintResponse,
  CapabilityRegistrationRequest,
  CapabilityRegistrationResponse,
  FaucetRequest,
  FaucetResponse,
  WalletBalanceResponse,
  TransactionRequest,
  TransactionResponse,
  AgentCapability,
  KNIRVOracleError,
  KNIRVORACLE_ENDPOINTS,
  KNIRVORACLE_TIMEOUTS,
  isKNIRVOracleError,
} from '../types/knirvoracle';

// Re-export types for convenience
export type {
  KNIRVOracleConfig,
  AgentMintRequest,
  AgentMintResponse,
  CapabilityRegistrationRequest,
  CapabilityRegistrationResponse,
  FaucetRequest,
  FaucetResponse,
  WalletBalanceResponse,
  TransactionRequest,
  TransactionResponse,
  AgentCapability,
} from '../types/knirvoracle';

/**
 * KNIRVOracleService provides TypeSafe integration with KNIRVORACLE
 * for agent minting, capability registration, wallet operations, and network interactions
 */
export class KNIRVOracleService {
  private client: AxiosInstance;
  private config: KNIRVOracleConfig;

  constructor(config: KNIRVOracleConfig) {
    this.config = config;
    this.client = axios.create({
      baseURL: config.baseURL,
      timeout: config.timeout || KNIRVORACLE_TIMEOUTS.TRANSACTION,
      headers: {
        'Content-Type': 'application/json',
        'User-Agent': 'KNIRVENGINE-Desktop-Client/1.0',
        ...(config.apiKey && { Authorization: `Bearer ${config.apiKey}` }),
      },
    });

    // Add response interceptor for error handling
    this.client.interceptors.response.use(
      (response) => response,
      (error) => {
        const knirvoracleError: KNIRVOracleError = new Error(
          `KNIRVORACLE API Error: ${error.response?.data?.message || error.message}`
        ) as KNIRVOracleError;

        knirvoracleError.isKNIRVOracleError = true;
        knirvoracleError.code = error.response?.data?.error_code;
        knirvoracleError.status = error.response?.status;
        knirvoracleError.response = error.response?.data;

        console.error('KNIRVORACLE API Error:', knirvoracleError);
        return Promise.reject(knirvoracleError);
      }
    );
  }

  /**
   * Mint a new agent as an NFT on KNIRVORACLE
   */
  async mintAgent(request: AgentMintRequest): Promise<AgentMintResponse> {
    try {
      const response: AxiosResponse<AgentMintResponse> = await this.client.post(
        KNIRVORACLE_ENDPOINTS.AGENT_MINT,
        request,
        { timeout: KNIRVORACLE_TIMEOUTS.AGENT_MINT }
      );
      return response.data;
    } catch (error) {
      if (isKNIRVOracleError(error)) {
        throw error;
      }
      throw new Error(`Failed to mint agent: ${error}`);
    }
  }

  /**
   * Register a capability with KNIRVORACLE
   */
  async registerCapability(request: CapabilityRegistrationRequest): Promise<CapabilityRegistrationResponse> {
    try {
      const response: AxiosResponse<CapabilityRegistrationResponse> = await this.client.post(
        KNIRVORACLE_ENDPOINTS.CAPABILITY_REGISTER,
        request,
        { timeout: KNIRVORACLE_TIMEOUTS.CAPABILITY_REGISTER }
      );
      return response.data;
    } catch (error) {
      if (isKNIRVOracleError(error)) {
        throw error;
      }
      throw new Error(`Failed to register capability: ${error}`);
    }
  }

  /**
   * Request tokens from the KNIRVORACLE faucet
   */
  async requestFaucet(request: FaucetRequest): Promise<FaucetResponse> {
    try {
      const response: AxiosResponse<FaucetResponse> = await this.client.post(
        '/api/mint/nrv',
        request
      );
      return response.data;
    } catch (error) {
      throw new Error(`Failed to request faucet: ${error}`);
    }
  }

  /**
   * Get wallet balance from KNIRVORACLE
   */
  async getWalletBalance(address: string): Promise<WalletBalanceResponse> {
    try {
      const response: AxiosResponse<WalletBalanceResponse> = await this.client.get(
        `/balance/${address}`
      );
      return response.data;
    } catch (error) {
      throw new Error(`Failed to get wallet balance: ${error}`);
    }
  }

  /**
   * Send a transaction through KNIRVORACLE
   */
  async sendTransaction(request: TransactionRequest): Promise<TransactionResponse> {
    try {
      const response: AxiosResponse<TransactionResponse> = await this.client.post(
        '/transactions',
        request
      );
      return response.data;
    } catch (error) {
      throw new Error(`Failed to send transaction: ${error}`);
    }
  }

  /**
   * Get capabilities for a specific agent
   */
  async getAgentCapabilities(agentId: string): Promise<AgentCapability[]> {
    try {
      const response: AxiosResponse<{ success: boolean; capabilities: AgentCapability[]; message: string }> = 
        await this.client.get(`/agent/capabilities/${agentId}`);
      
      if (!response.data.success) {
        throw new Error(response.data.message);
      }
      
      return response.data.capabilities;
    } catch (error) {
      throw new Error(`Failed to get agent capabilities: ${error}`);
    }
  }

  /**
   * Invoke a capability through KNIRVORACLE
   */
  async invokeCapability(capabilityId: string, inputData: Record<string, unknown>): Promise<Record<string, unknown>> {
    try {
      const requestBody = {
        capability_id: capabilityId,
        interaction_type: 'invoke',
        input_data: inputData,
        timestamp: Date.now(),
      };

      const response: AxiosResponse<Record<string, unknown>> = await this.client.post(
        '/wallet/mcp/create_invoke_capability',
        requestBody
      );
      
      return response.data;
    } catch (error) {
      throw new Error(`Failed to invoke capability: ${error}`);
    }
  }

  /**
   * Get treasury status from KNIRVORACLE
   */
  async getTreasuryStatus(): Promise<Record<string, unknown>> {
    try {
      const response: AxiosResponse<Record<string, unknown>> = await this.client.get('/api/treasury/status');
      return response.data;
    } catch (error) {
      throw new Error(`Failed to get treasury status: ${error}`);
    }
  }

  /**
   * Perform a health check against KNIRVORACLE
   */
  async healthCheck(): Promise<boolean> {
    try {
      const response = await this.client.get(
        KNIRVORACLE_ENDPOINTS.HEALTH,
        { timeout: KNIRVORACLE_TIMEOUTS.HEALTH_CHECK }
      );
      return response.status === 200;
    } catch (error) {
      console.error('KNIRVORACLE health check failed:', error);
      return false;
    }
  }

  /**
   * Update the service configuration
   */
  updateConfig(newConfig: Partial<KNIRVOracleConfig>): void {
    this.config = { ...this.config, ...newConfig };
    
    // Update axios instance with new config
    if (newConfig.baseURL) {
      this.client.defaults.baseURL = newConfig.baseURL;
    }
    
    if (newConfig.timeout) {
      this.client.defaults.timeout = newConfig.timeout;
    }
    
    if (newConfig.apiKey) {
      this.client.defaults.headers.Authorization = `Bearer ${newConfig.apiKey}`;
    }
  }

  /**
   * Get current configuration
   */
  getConfig(): KNIRVOracleConfig {
    return { ...this.config };
  }
}

// Create a default instance for use throughout the application
export const knirvoracleService = new KNIRVOracleService({
  baseURL: import.meta.env.VITE_KNIRVORACLE_URL || 'http://localhost:8080',
  apiKey: import.meta.env.VITE_KNIRVORACLE_API_KEY,
  timeout: 30000,
});

export default knirvoracleService;
