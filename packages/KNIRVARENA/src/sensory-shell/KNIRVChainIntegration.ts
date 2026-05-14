import { EventEmitter } from './EventEmitter';
import { KNIRVRouterIntegration } from './KNIRVRouterIntegration';

export interface ChainConfig {
  rpcUrl: string;
  chainId: string;
  networkName: string;
  contractAddresses: {
    nrnToken: string;
    llmRegistry: string;
    skillRegistry: string;
  };
  gasPrice: string;
  gasLimit: string;
  // KNIRVROUTER network configuration
  knirvRouterUrl?: string;
  knirvGraphUrl?: string;
  useKnirvRouter?: boolean;
}

export interface SkillMetadata {
  id: string;
  name: string;
  skillType: string;
  capabilities: string[];
  requirements: Record<string, string>;
  owner: string;
  usageFee: string;
  validationStatus: 'pending' | 'validated' | 'rejected';
  performanceMetrics: {
    successRate: number;
    averageLatency: number;
    totalInvocations: number;
    lastUpdated: number;
  };
  registeredAt: number;
}

export interface LLMMetadata {
  id: string;
  name: string;
  version: string;
  modelHash: string;
  capabilities: string[];
  owner: string;
  registrationFee: string;
  usageFee: string;
  ipfsHash?: string;
  validationStatus: 'pending' | 'validated' | 'rejected';
  registeredAt: number;
}

export interface SkillInvocation {
  skillId: string;
  user: string;
  amountBurned: string;
  timestamp: number;
  success: boolean;
  resultHash?: string;
  transactionHash: string;
}

export interface ContractCall {
  contract: string;
  method: string;
  params: unknown;
}

export interface ContractResponse {
  success: boolean;
  data?: unknown;
  error?: string;
  transactionHash?: string;
  gasUsed?: string;
}

export interface NetworkConsensus {
  blockHeight: number;
  blockHash: string;
  validators: string[];
  consensusReached: boolean;
  timestamp: number;
}

// KNIRVROUTER network integration interfaces
export interface ErrorContext {
  errorId: string;
  errorType: string;
  errorMessage: string;
  stackTrace: string;
  userContext: unknown;
  agentId: string;
  timestamp: number;
  severity: 'low' | 'medium' | 'high' | 'critical';
}

export interface SkillNodeURI {
  nodeId: string;
  skillId: string;
  routerAddress: string;
  networkPath: string;
  capabilities: string[];
  confidence: number;
}

export interface KNIRVRouterRequest {
  requestId: string;
  errorContext: ErrorContext;
  requiredCapabilities: string[];
  nrnToken: string;
  agentId: string;
  priority: 'low' | 'normal' | 'high';
  timestamp: number;
}

export interface KNIRVRouterResponse {
  requestId: string;
  status: 'SUCCESS' | 'FAILURE' | 'NOT_FOUND';
  skillNodeUri?: SkillNodeURI;
  loraAdapter?: LoRAAdapterData;
  errorMessage?: string;
  executionTime: number;
  networkLatency: number;
}

export interface LoRAAdapterData {
  adapterId: string;
  adapterName: string;
  description: string;
  baseModelCompatibility: string;
  version: number;
  rank: number;
  alpha: number;
  weightsA: Float32Array;
  weightsB: Float32Array;
  metadata: Record<string, string>;
  createdAt: Date;
  lastUsed: Date;
  usageCount: number;
  networkScore: number;
}

export class KNIRVChainIntegration extends EventEmitter {
  private config: ChainConfig;
  private isConnected: boolean = false;
  private skills: Map<string, SkillMetadata> = new Map();
  private llmModels: Map<string, LLMMetadata> = new Map();
  private skillInvocations: Map<string, SkillInvocation[]> = new Map();
  private knirvRouter: KNIRVRouterIntegration;
  private lastBlockHeight: number = 0;

  constructor(config: Partial<ChainConfig>) {
    super();

    this.config = {
      rpcUrl: 'http://localhost:8080',
      chainId: 'knirv-chain-1',
      networkName: 'KNIRV Network',
      contractAddresses: {
        nrnToken: '0x1234567890123456789012345678901234567890',
        llmRegistry: '0x2345678901234567890123456789012345678901',
        skillRegistry: '0x3456789012345678901234567890123456789012',
      },
      gasPrice: '20000000000', // 20 gwei
      gasLimit: '500000',
      // KNIRVROUTER network defaults
      knirvRouterUrl: 'http://localhost:5000/knirv-router',
      knirvGraphUrl: 'http://localhost:5001/knirv-graph',
      useKnirvRouter: true,
      ...config,
    };

    // Initialize KNIRVROUTER integration
    this.knirvRouter = new KNIRVRouterIntegration({
      routerUrl: this.config.knirvRouterUrl || 'http://localhost:5000',
      graphUrl: this.config.knirvGraphUrl || 'http://localhost:5001',
      oracleUrl: 'http://localhost:5002',
      timeout: 30000,
      retryAttempts: 3,
      enableP2P: true,
      enableWASM: true
    });

    // Set up KNIRVROUTER event handlers
    this.setupKNIRVRouterEventHandlers();
  }

  /**
   * Set up KNIRVROUTER event handlers
   */
  private setupKNIRVRouterEventHandlers(): void {
    this.knirvRouter.on('connected', (data) => {
      console.log('KNIRVROUTER connected:', data);
      this.emit('knirvRouterConnected', data);
    });

    this.knirvRouter.on('disconnected', (data) => {
      console.log('KNIRVROUTER disconnected:', data);
      this.emit('knirvRouterDisconnected', data);
    });

    this.knirvRouter.on('skillResolved', (data) => {
      console.log('Skill resolved via KNIRVROUTER:', data);
      this.emit('skillResolvedViaKNIRVRouter', data);
    });

    this.knirvRouter.on('skillNodeDiscovered', (data) => {
      console.log('Skill node discovered:', data);
      this.emit('skillNodeDiscovered', data);
    });

    this.knirvRouter.on('p2pConnected', (data) => {
      console.log('P2P connection established:', data);
      this.emit('p2pConnected', data);
    });

    this.knirvRouter.on('wasmSkillExecuted', (data) => {
      console.log('WASM skill executed:', data);
      this.emit('wasmSkillExecuted', data);
    });

    this.knirvRouter.on('loraAdapterRegistered', (data) => {
      console.log('LoRA adapter registered:', data);
      this.emit('loraAdapterRegistered', data);
    });
  }

  public async initialize(): Promise<void> {
    console.log('Initializing KNIRV Chain Integration...');

    try {
      // Connect to KNIRV Chain RPC
      await this.connectToChain();

      // Load existing skills and LLM models
      await this.loadSkillRegistry();
      await this.loadLLMRegistry();

      // Start monitoring blockchain events
      this.startEventMonitoring();

      this.isConnected = true;
      this.emit('chainInitialized');
      console.log('KNIRV Chain Integration initialized successfully');

    } catch (error) {
      console.error('Failed to initialize KNIRV Chain Integration:', error);
      throw error;
    }
  }

  public async disconnect(): Promise<void> {
    console.log('Disconnecting KNIRV Chain Integration...');
    
    this.isConnected = false;
    this.skills.clear();
    this.llmModels.clear();
    this.skillInvocations.clear();

    this.emit('chainDisconnected');
    console.log('KNIRV Chain Integration disconnected');
  }

  private async connectToChain(): Promise<void> {
    try {
      console.log(`Connecting to KNIRV Chain at ${this.config.rpcUrl}...`);
      
      // Test connection with a simple RPC call
      const response = await fetch(`${this.config.rpcUrl}/status`);
      const result = await response.json();

      if (result.success) {
        this.lastBlockHeight = result.data.block_height || 0;
        console.log(`Connected to KNIRV Chain at block height: ${this.lastBlockHeight}`);
      } else {
        throw new Error('Failed to connect to KNIRV Chain');
      }

    } catch (error) {
      console.error('Chain connection failed:', error);
      throw error;
    }
  }

  private async loadSkillRegistry(): Promise<void> {
    try {
      console.log('Loading skill registry from chain...');
      
      const response = await this.executeContractCall({
        contract: 'skill_registry',
        method: 'get_all_skills',
        params: {},
      });

      if (response.success && response.data) {
        const skills = (response.data as { skills?: unknown[] }).skills || [];
        
        for (const skill of skills) {
          const skillMetadata = skill as SkillMetadata;
          this.skills.set(skillMetadata.id, skillMetadata);
        }

        console.log(`Loaded ${this.skills.size} skills from registry`);
        this.emit('skillsLoaded', Array.from(this.skills.values()));
      }

    } catch (error) {
      console.error('Failed to load skill registry:', error);
    }
  }

  private async loadLLMRegistry(): Promise<void> {
    try {
      console.log('Loading LLM registry from chain...');
      
      const response = await this.executeContractCall({
        contract: 'llm_registry',
        method: 'get_all_models',
        params: {},
      });

      if (response.success && response.data) {
        const models = (response.data as { models?: unknown[] }).models || [];
        
        for (const model of models) {
          const modelMetadata = model as LLMMetadata;
          this.llmModels.set(modelMetadata.id, modelMetadata);
        }

        console.log(`Loaded ${this.llmModels.size} LLM models from registry`);
        this.emit('llmModelsLoaded', Array.from(this.llmModels.values()));
      }

    } catch (error) {
      console.error('Failed to load LLM registry:', error);
    }
  }

  public async executeContractCall(call: ContractCall): Promise<ContractResponse> {
    if (!this.isConnected) {
      throw new Error('Not connected to KNIRV Chain');
    }

    try {
      console.log(`Executing contract call: ${call.contract}.${call.method}`);
      
      const response = await fetch(`${this.config.rpcUrl}/contract/call`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          contract: call.contract,
          method: call.method,
          params: call.params,
          gas_price: this.config.gasPrice,
          gas_limit: this.config.gasLimit,
        }),
      });

      const result = await response.json();
      
      if (result.success) {
        this.emit('contractCallExecuted', {
          call,
          response: result.data,
          timestamp: Date.now(),
        });

        return {
          success: true,
          data: result.data,
          transactionHash: result.transaction_hash,
          gasUsed: result.gas_used,
        };
      } else {
        return {
          success: false,
          error: result.error || 'Contract call failed',
        };
      }

    } catch (error) {
      console.error('Contract call failed:', error);
      return {
        success: false,
        error: error instanceof Error ? error.message : String(error),
      };
    }
  }

  public async verifySkill(skillId: string): Promise<boolean> {
    try {
      const response = await this.executeContractCall({
        contract: 'skill_registry',
        method: 'get_skill',
        params: { skill_id: skillId },
      });

      if (response.success && response.data) {
        const skill = (response.data as { skill: { validation_status?: string } }).skill;
        return skill.validation_status === 'validated';
      }

      return false;

    } catch (error) {
      console.error('Failed to verify skill:', error);
      return false;
    }
  }

  /**
   * Revolutionary ErrorContext → KNIRVGRAPH → KNIRVROUTER skill invocation
   * This replaces embedded skill invocation with external network integration
   */
  public async invokeSkillOnChain(
    skillId: string,
    userAddress: string,
    nrnAmount: string,
    parameters: unknown
  ): Promise<string> {
    try {
      console.log(`Invoking skill ${skillId} via KNIRVROUTER network...`);

      // Use KNIRVROUTER if enabled
      if (this.config.useKnirvRouter) {
        return await this.invokeSkillViaKNIRVRouter(skillId, userAddress, nrnAmount, parameters);
      }

      // Fallback to traditional blockchain invocation
      return await this.invokeSkillOnTraditionalChain(skillId, userAddress, nrnAmount, parameters);

    } catch (error) {
      console.error('Failed to invoke skill via KNIRVROUTER:', error);
      throw error;
    }
  }

  /**
   * Invoke skill via KNIRVROUTER network (revolutionary ErrorContext → KNIRVGRAPH → SkillNode approach)
   */
  private async invokeSkillViaKNIRVRouter(
    skillId: string,
    userAddress: string,
    nrnAmount: string,
    parameters: unknown
  ): Promise<string> {
    // Generate ErrorContext for the skill request
    const errorContext: ErrorContext = {
      errorId: `err_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
      errorType: 'skill_invocation_request',
      errorMessage: `Skill invocation requested: ${skillId}`,
      stackTrace: `Skill: ${skillId}, User: ${userAddress}, NRN: ${nrnAmount}`,
      userContext: { userAddress, nrnAmount, parameters },
      agentId: (parameters as { agentId?: string }).agentId || 'unknown-agent',
      timestamp: Date.now(),
      severity: (parameters as { priority?: string }).priority === 'high' ? 'high' : 'medium'
    };

    // Use the KNIRVRouterIntegration for skill resolution
    const routerResponse = await this.knirvRouter.resolveSkillViaErrorContext(
      errorContext,
      ((parameters as { capabilities?: unknown[] }).capabilities || []) as string[],
      {
        priority: ((parameters as { priority?: string }).priority as 'low' | 'normal' | 'high') || 'normal',
        useP2P: (parameters as { useP2P?: boolean }).useP2P !== false,
        useWASM: (parameters as { useWASM?: boolean }).useWASM !== false,
        nrnToken: nrnAmount
      }
    );

    if (routerResponse.status !== 'SUCCESS') {
      throw new Error(`Skill resolution failed: ${routerResponse.errorMessage}`);
    }

    // Emit the event for backward compatibility
    this.emit('skillResolvedViaKNIRVRouter', {
      requestId: routerResponse.requestId,
      skillId,
      skillNodeUri: routerResponse.skillNodeUri,
      userAddress,
      nrnAmount,
      executionTime: routerResponse.executionTime,
      networkLatency: routerResponse.networkLatency,
      timestamp: Date.now(),
    });

    return routerResponse.requestId;
  }



  /**
   * Invoke skill on traditional blockchain (fallback)
   */
  private async invokeSkillOnTraditionalChain(
    skillId: string,
    userAddress: string,
    nrnAmount: string,
    parameters: unknown
  ): Promise<string> {
    // First, burn NRN tokens for skill usage
    const burnResponse = await this.executeContractCall({
      contract: 'nrn_token',
      method: 'burn_for_skill',
      params: {
        from: userAddress,
        skill_id: skillId,
        amount: nrnAmount,
      },
    });

    if (!burnResponse.success) {
      throw new Error(`Failed to burn NRN: ${burnResponse.error}`);
    }

    // Record skill invocation
    const invocationResponse = await this.executeContractCall({
      contract: 'skill_registry',
      method: 'record_invocation',
      params: {
        skill_id: skillId,
        user: userAddress,
        amount_burned: nrnAmount,
        parameters: JSON.stringify(parameters),
      },
    });

    if (!invocationResponse.success) {
      throw new Error(`Failed to record invocation: ${invocationResponse.error}`);
    }

    const transactionHash = invocationResponse.transactionHash || '';

    // Update local cache
    const invocation: SkillInvocation = {
      skillId,
      user: userAddress,
      amountBurned: nrnAmount,
      timestamp: Date.now(),
      success: true,
      transactionHash,
    };

    if (!this.skillInvocations.has(skillId)) {
      this.skillInvocations.set(skillId, []);
    }
    this.skillInvocations.get(skillId)!.push(invocation);

    this.emit('skillInvokedOnChain', invocation);
    return transactionHash;
  }

  public async registerSkill(skillMetadata: Omit<SkillMetadata, 'id' | 'registeredAt'>): Promise<string> {
    try {
      console.log(`Registering skill: ${skillMetadata.name}`);

      const response = await this.executeContractCall({
        contract: 'skill_registry',
        method: 'register_skill',
        params: {
          name: skillMetadata.name,
          skill_type: skillMetadata.skillType,
          capabilities: skillMetadata.capabilities,
          requirements: skillMetadata.requirements,
          owner: skillMetadata.owner,
          usage_fee: skillMetadata.usageFee,
        },
      });

      if (response.success && response.data) {
        const skillId = (response.data as { skill_id?: string }).skill_id;
        
        // Update local cache
        const fullSkillMetadata: SkillMetadata = {
          ...skillMetadata,
          id: skillId || 'unknown-skill-id',
          registeredAt: Date.now(),
          validationStatus: 'pending' as 'pending' | 'validated' | 'rejected',
          performanceMetrics: {
            successRate: 0,
            averageLatency: 0,
            totalInvocations: 0,
            lastUpdated: Date.now(),
          },
        };

        this.skills.set(skillId || 'unknown-skill-id', fullSkillMetadata);
        this.emit('skillRegistered', fullSkillMetadata);

        return skillId || 'unknown-skill-id';
      } else {
        throw new Error(response.error || 'Failed to register skill');
      }

    } catch (error) {
      console.error('Failed to register skill:', error);
      throw error;
    }
  }

  public async registerLLMModel(llmMetadata: Omit<LLMMetadata, 'id' | 'registeredAt'>): Promise<string> {
    try {
      console.log(`Registering LLM model: ${llmMetadata.name}`);

      const response = await this.executeContractCall({
        contract: 'llm_registry',
        method: 'register_model',
        params: {
          name: llmMetadata.name,
          version: llmMetadata.version,
          model_hash: llmMetadata.modelHash,
          capabilities: llmMetadata.capabilities,
          owner: llmMetadata.owner,
          registration_fee: llmMetadata.registrationFee,
          usage_fee: llmMetadata.usageFee,
          ipfs_hash: llmMetadata.ipfsHash,
        },
      });

      if (response.success && response.data) {
        const modelId = (response.data as { model_id?: string }).model_id;
        
        // Update local cache
        const fullLLMMetadata: LLMMetadata = {
          ...llmMetadata,
          id: modelId || 'unknown-model-id',
          registeredAt: Date.now(),
          validationStatus: 'pending' as 'pending' | 'validated' | 'rejected',
        };

        this.llmModels.set(modelId || 'unknown-model-id', fullLLMMetadata);
        this.emit('llmModelRegistered', fullLLMMetadata);

        return modelId || 'unknown-model-id';
      } else {
        throw new Error(response.error || 'Failed to register LLM model');
      }

    } catch (error) {
      console.error('Failed to register LLM model:', error);
      throw error;
    }
  }

  public async getNetworkConsensus(): Promise<NetworkConsensus> {
    try {
      const response = await fetch(`${this.config.rpcUrl}/consensus/status`);
      const result = await response.json();

      if (result.success) {
        return {
          blockHeight: result.data.block_height,
          blockHash: result.data.block_hash,
          validators: result.data.validators || [],
          consensusReached: result.data.consensus_reached,
          timestamp: Date.now(),
        };
      } else {
        throw new Error('Failed to get network consensus');
      }

    } catch (error) {
      console.error('Failed to get network consensus:', error);
      throw error;
    }
  }

  public async getNRNBalance(address: string): Promise<string> {
    try {
      const response = await this.executeContractCall({
        contract: 'nrn_token',
        method: 'balance_of',
        params: { address },
      });

      if (response.success && response.data) {
        return (response.data as { balance?: string }).balance || '0';
      }

      return '0';

    } catch (error) {
      console.error('Failed to get NRN balance:', error);
      return '0';
    }
  }

  public async transferNRN(from: string, to: string, amount: string): Promise<string> {
    try {
      const response = await this.executeContractCall({
        contract: 'nrn_token',
        method: 'transfer',
        params: { from, to, amount },
      });

      if (response.success) {
        this.emit('nrnTransferred', {
          from,
          to,
          amount,
          transactionHash: response.transactionHash,
          timestamp: Date.now(),
        });

        return response.transactionHash || '';
      } else {
        throw new Error(response.error || 'Transfer failed');
      }

    } catch (error) {
      console.error('Failed to transfer NRN:', error);
      throw error;
    }
  }

  /**
   * Programmatic LoRA adapter filtering system that traverses skill chains to find relevant adapters
   */
  public async findSkillsWithFiltering(filter: {
    skillType?: string;
    baseModel?: string;
    minNetworkScore?: number;
    maxRank?: number;
    capabilities?: string[];
    excludeSkills?: string[];
  }): Promise<LoRAAdapterData[]> {
    if (!this.config.useKnirvRouter) {
      console.warn('LoRA adapter filtering only available with KNIRVROUTER');
      return [];
    }

    try {
      const response = await fetch(`${this.config.knirvRouterUrl}/lora-adapters/filter`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(filter),
      });

      if (!response.ok) {
        throw new Error(`Filter request failed: ${response.statusText}`);
      }

      const result = await response.json();
      return result.adapters || [];

    } catch (error) {
      console.error('Failed to filter LoRA adapters:', error);
      throw error;
    }
  }

  /**
   * Create skill chain via KNIRVROUTER network using LoRA adapter composition
   */
  public async createSkillChain(skillIds: string[]): Promise<unknown> {
    if (!this.config.useKnirvRouter) {
      throw new Error('Skill chains only available with KNIRVROUTER');
    }

    try {
      const response = await fetch(`${this.config.knirvRouterUrl}/skill-chains`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ skill_ids: skillIds }),
      });

      if (!response.ok) {
        throw new Error(`Skill chain creation failed: ${response.statusText}`);
      }

      const skillChain = await response.json();

      this.emit('skillChainCreated', {
        chainId: skillChain.chain_id,
        skillIds,
        adapterCount: skillChain.adapters?.length || 0,
        timestamp: Date.now(),
      });

      return skillChain;

    } catch (error) {
      console.error('Failed to create skill chain via KNIRVROUTER:', error);
      throw error;
    }
  }

  /**
   * Get all available LoRA adapter skills from KNIRVROUTER network
   */
  public async getLoRAAdapterSkills(filter?: unknown): Promise<LoRAAdapterData[]> {
    if (!this.config.useKnirvRouter) {
      console.warn('LoRA adapter skills only available with KNIRVROUTER');
      return [];
    }

    try {
      return await this.knirvRouter.getLoRAAdapters(filter as any);
    } catch (error) {
      console.error('Failed to get LoRA adapters from KNIRVROUTER:', error);
      throw error;
    }
  }

  /**
   * Register a new LoRA adapter skill via KNIRVROUTER network
   */
  public async registerLoRAAdapterSkill(skill: Omit<LoRAAdapterData, 'createdAt' | 'lastUsed' | 'usageCount' | 'networkScore' | 'routerNodes'>): Promise<string> {
    if (!this.config.useKnirvRouter) {
      throw new Error('LoRA adapter skill registration only available with KNIRVROUTER');
    }

    try {
      return await this.knirvRouter.registerLoRAAdapter(skill);
    } catch (error) {
      console.error('Failed to register LoRA adapter via KNIRVROUTER:', error);
      throw error;
    }
  }

  private startEventMonitoring(): void {
    // Monitor blockchain events
    const monitorEvents = async () => {
      while (this.isConnected) {
        try {
          await this.checkForNewBlocks();
          await this.checkForSkillUpdates();
          
          // Wait before next check
          await new Promise(resolve => setTimeout(resolve, 10000)); // 10 seconds

        } catch (error) {
          console.error('Error monitoring blockchain events:', error);
        }
      }
    };

    monitorEvents();
  }

  private async checkForNewBlocks(): Promise<void> {
    try {
      const consensus = await this.getNetworkConsensus();
      
      if (consensus.blockHeight > this.lastBlockHeight) {
        const newBlocks = consensus.blockHeight - this.lastBlockHeight;
        this.lastBlockHeight = consensus.blockHeight;

        this.emit('newBlocks', {
          newBlockCount: newBlocks,
          currentHeight: consensus.blockHeight,
          blockHash: consensus.blockHash,
        });
      }

    } catch (error) {
      console.error('Error checking for new blocks:', error);
    }
  }

  private async checkForSkillUpdates(): Promise<void> {
    try {
      // Check for skill validation updates
      this.skills.forEach(async (skill, skillId) => {
        if (skill.validationStatus === 'pending') {
          const response = await this.executeContractCall({
            contract: 'skill_registry',
            method: 'get_skill',
            params: { skill_id: skillId },
          });

          if (response.success && response.data) {
            const updatedSkill = (response.data as { skill: { validation_status?: string } }).skill;
            if (updatedSkill.validation_status !== skill.validationStatus) {
              skill.validationStatus = (updatedSkill.validation_status as 'pending' | 'validated' | 'rejected') || 'pending';
              this.skills.set(skillId, skill);
              
              this.emit('skillValidationUpdated', {
                skillId,
                newStatus: skill.validationStatus,
                skill,
              });
            }
          }
        }
      });

    } catch (error) {
      console.error('Error checking for skill updates:', error);
    }
  }

  public getSkills(): SkillMetadata[] {
    return Array.from(this.skills.values());
  }

  public getSkill(skillId: string): SkillMetadata | null {
    return this.skills.get(skillId) || null;
  }

  public getLLMModels(): LLMMetadata[] {
    return Array.from(this.llmModels.values());
  }

  public getLLMModel(modelId: string): LLMMetadata | null {
    return this.llmModels.get(modelId) || null;
  }

  public getSkillInvocations(skillId: string): SkillInvocation[] {
    return this.skillInvocations.get(skillId) || [];
  }

  public isChainConnected(): boolean {
    return this.isConnected;
  }

  public getConfig(): ChainConfig {
    return { ...this.config };
  }

  public updateConfig(newConfig: Partial<ChainConfig>): void {
    this.config = { ...this.config, ...newConfig };
    this.emit('configUpdated', this.config);
  }

  public getStatus(): unknown {
    return {
      isConnected: this.isConnected,
      chainId: this.config.chainId,
      networkName: this.config.networkName,
      lastBlockHeight: this.lastBlockHeight,
      skillsCount: this.skills.size,
      llmModelsCount: this.llmModels.size,
      config: this.config,
    };
  }
}
