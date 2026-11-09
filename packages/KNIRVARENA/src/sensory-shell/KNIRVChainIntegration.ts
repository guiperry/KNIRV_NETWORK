import { EventEmitter } from './EventEmitter';
import { getKnirvGatewayUrl } from '../config/runtimeConfig';
import { submitChainWrite, type ChainWrite } from '../services/ChainWrites';
import { showQR } from '../services/ControllerActuarialHandoff';

/**
 * KNIRV Chain access for the Arena.
 *
 * Every call goes to a KNIRVCHAIN route that KNIRVGATEWAY proxies under
 * `/api/chain/*` (the gateway strips the prefix and forwards to the chain
 * socket, `internal/blockchain/blockchain_server.go`):
 *
 *   GET /api/chain/health                          liveness
 *   GET /api/chain/chain/head                      head: chain id, height, hash
 *   GET /api/chain/mcp/capability/list?type=TOOL   skills (MCP TOOL capabilities)
 *   GET /api/chain/mcp/capability/get?id=<id>      one capability (404 when absent)
 *   GET /api/chain/accounts/balance?address=<a>    NRN balance of an account
 *
 * Writes (register a skill, invoke a skill, transfer NRN) must be signed chain
 * transactions. The Arena cannot sign them yet, so those methods reject with
 * an explicit error instead of returning placeholder data.
 */

/** KNIRVCHAIN capability type that the Arena treats as a skill. */
export const SKILL_CAPABILITY_TYPE = 'TOOL';

export interface ChainConfig {
  /** Chain base URL on the gateway, e.g. `https://gateway.knirv.com/api/chain`. */
  rpcUrl: string;
  chainId: string;
  networkName: string;
  /** Per-request timeout in milliseconds. */
  requestTimeoutMs: number;
  /**
   * Shows the Controller approval URI for a signed write. Defaults to the QR
   * prompt; returns a closer for the prompt.
   */
  onApprovalUri?: ChainWrite['onApprovalUri'];
}

/** KNIRV Chain capability types a skill may be registered as. */
const CHAIN_CAPABILITY_TYPES = ['TOOL', 'RESOURCE', 'PROMPT', 'MEMORY_SERVICE'] as const;

export interface SkillMetadata {
  id: string;
  name: string;
  skillType: string;
  capabilities: string[];
  requirements: Record<string, string>;
  owner: string;
  usageFee: string;
  /**
   * KNIRVCHAIN capabilities carry no validation status, so skills read from the
   * chain are reported as `pending` until the chain exposes one.
   */
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

export interface NetworkConsensus {
  chainId: string;
  blockHeight: number;
  blockHash: string;
  timestamp: number;
}

interface ChainHeadBody {
  chain_id: string;
  height: number;
  block_hash: string;
  timestamp: number;
}

const unsupportedWrite = (operation: string, reason: string): Error =>
  new Error(`${operation} is not available: ${reason}`);

const encode = (value: unknown): Uint8Array => new TextEncoder().encode(JSON.stringify(value));

/** A random identifier from the platform's CSPRNG (available in browsers and jsdom alike). */
const randomHex = (bytes: number): string =>
  Array.from(crypto.getRandomValues(new Uint8Array(bytes)), (b) => b.toString(16).padStart(2, '0')).join('');

export class KNIRVChainIntegration extends EventEmitter {
  private config: ChainConfig;
  private isConnected: boolean = false;
  private skills: Map<string, SkillMetadata> = new Map();
  // KNIRVCHAIN has no LLM-model registry; this map stays empty.
  private llmModels: Map<string, LLMMetadata> = new Map();
  // Invocations are recorded on chain only once signed submission exists.
  private skillInvocations: Map<string, SkillInvocation[]> = new Map();
  private lastBlockHeight: number = 0;
  private monitorTimer: ReturnType<typeof setInterval> | null = null;

  constructor(config: Partial<ChainConfig>) {
    super();
    this.config = {
      rpcUrl: `${getKnirvGatewayUrl()}/api/chain`,
      chainId: 'knirv-1',
      networkName: 'KNIRV Network',
      requestTimeoutMs: 15000,
      onApprovalUri: typeof document !== 'undefined' ? showQR : undefined,
      ...config,
    };
  }

  /** The gateway origin that chain signing requests and transactions go through. */
  private gatewayOrigin(): string {
    const origin = /^https?:\/\/[^/]+/i.exec(this.config.rpcUrl);
    if (!origin) {
      throw new Error(`KNIRV Chain URL is not absolute: ${this.config.rpcUrl}`);
    }
    return origin[0];
  }

  public async initialize(): Promise<void> {
    console.log('Initializing KNIRV Chain Integration...');

    try {
      await this.connectToChain();
      await this.loadSkillRegistry();

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

    this.stopEventMonitoring();
    this.isConnected = false;
    this.skills.clear();
    this.llmModels.clear();
    this.skillInvocations.clear();

    this.emit('chainDisconnected');
    console.log('KNIRV Chain Integration disconnected');
  }

  /** GET a chain route through the gateway. Returns the raw Response; throws on network failure. */
  private async chainFetch(path: string, query: Record<string, string> = {}): Promise<Response> {
    const base = this.config.rpcUrl.replace(/\/$/, '');
    const search = new URLSearchParams(query).toString();
    const url = `${base}${path}${search ? `?${search}` : ''}`;

    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), this.config.requestTimeoutMs);
    try {
      return await fetch(url, { headers: { Accept: 'application/json' }, signal: controller.signal });
    } finally {
      clearTimeout(timer);
    }
  }

  /** GET a chain route and decode JSON. A 404 resolves with `data: null`; other failures throw. */
  private async chainGet<T>(path: string, query: Record<string, string> = {}): Promise<{ status: number; data: T | null }> {
    const response = await this.chainFetch(path, query);
    if (response.status === 404) {
      return { status: 404, data: null };
    }
    if (!response.ok) {
      throw new Error(`KNIRV Chain ${path} failed: HTTP ${response.status}`);
    }
    return { status: response.status, data: (await response.json()) as T };
  }

  private async connectToChain(): Promise<void> {
    console.log(`Connecting to KNIRV Chain at ${this.config.rpcUrl}...`);

    const health = await this.chainFetch('/health');
    if (!health.ok) {
      throw new Error(`KNIRV Chain health check failed: HTTP ${health.status}`);
    }

    const head = await this.chainGet<ChainHeadBody>('/chain/head');
    if (!head.data) {
      throw new Error('KNIRV Chain returned no head');
    }
    this.lastBlockHeight = head.data.height;
    console.log(`Connected to KNIRV Chain at block height: ${this.lastBlockHeight}`);
  }

  private async loadSkillRegistry(): Promise<void> {
    try {
      console.log('Loading skills from KNIRV Chain...');

      const list = await this.chainGet<{ capabilities?: unknown[] }>('/mcp/capability/list', {
        type: SKILL_CAPABILITY_TYPE,
      });
      for (const capability of list.data?.capabilities ?? []) {
        const skill = this.toSkillMetadata(capability);
        if (skill) {
          this.skills.set(skill.id, skill);
        }
      }

      console.log(`Loaded ${this.skills.size} skills from KNIRV Chain`);
      this.emit('skillsLoaded', Array.from(this.skills.values()));
    } catch (error) {
      console.error('Failed to load skills from KNIRV Chain:', error);
    }
  }

  private toSkillMetadata(raw: unknown): SkillMetadata | null {
    const c = raw as {
      id?: string;
      name?: string;
      capabilityType?: string;
      owner?: string;
      gasFeeNRN?: number;
      timestamp?: number;
      description?: string;
    };
    if (!c || typeof c.id !== 'string' || c.id === '') {
      return null;
    }
    return {
      id: c.id,
      name: c.name || c.id,
      skillType: c.capabilityType || SKILL_CAPABILITY_TYPE,
      capabilities: [],
      requirements: {},
      owner: c.owner || '',
      usageFee: String(c.gasFeeNRN ?? 0),
      validationStatus: 'pending',
      performanceMetrics: {
        successRate: 0,
        averageLatency: 0,
        totalInvocations: 0,
        lastUpdated: Date.now(),
      },
      registeredAt: Number(c.timestamp) || 0,
    };
  }

  /** Whether a skill (capability) exists on chain. The chain has no validation status to report. */
  public async verifySkill(skillId: string): Promise<boolean> {
    try {
      const response = await this.chainGet<{ status?: string; capability?: unknown }>('/mcp/capability/get', {
        id: skillId,
      });
      return response.status === 200 && response.data?.status === 'success' && response.data.capability != null;
    } catch (error) {
      console.error('Failed to verify skill on KNIRV Chain:', error);
      return false;
    }
  }

  /**
   * Records a skill invocation on chain. The invoker signs it in the Controller and
   * pays the usage fee as the network fee.
   */
  public async invokeSkillOnChain(
    skillId: string,
    userAddress: string,
    nrnAmount: string,
    _parameters: unknown
  ): Promise<string> {
    const timestamp = Math.floor(Date.now() / 1000);
    const invocation = {
      contextRecord: {
        id: '',
        capabilityID: skillId,
        interactionType: 'Invoke',
        initiator: userAddress,
        status: 'pending',
        timestamp,
        timestampInitiated: timestamp,
      },
    };
    const transactionHash = await submitChainWrite(
      {
        type: 'invoke_capability_txn',
        sender: userAddress,
        data: encode(invocation),
        fee: Number(nrnAmount) || 0,
        onApprovalUri: this.config.onApprovalUri,
      },
      { gatewayUrl: this.gatewayOrigin() },
    );

    const record: SkillInvocation = {
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
    this.skillInvocations.get(skillId)!.push(record);
    this.emit('skillInvokedOnChain', record);
    return transactionHash;
  }

  /**
   * Registers a skill as a KNIRV Chain capability. The owner signs the registration in
   * the Controller; the chain verifies it and stores the capability.
   */
  public async registerSkill(skillMetadata: Omit<SkillMetadata, 'id' | 'registeredAt'>): Promise<string> {
    if (!(CHAIN_CAPABILITY_TYPES as readonly string[]).includes(skillMetadata.skillType)) {
      throw new Error(`skill type ${skillMetadata.skillType} is not a KNIRV Chain capability type`);
    }
    const skillId = `skill-${randomHex(16)}`;
    const registration = {
      capabilityID: skillId,
      name: skillMetadata.name,
      owner: skillMetadata.owner,
      version: '1',
      description: '',
      capabilityType: skillMetadata.skillType,
      gasFeeNRN: Number(skillMetadata.usageFee) || 0,
      capabilityDescriptor: { inputSchemaJSON: '{}', outputSchemaJSON: '{}' },
    };
    await submitChainWrite(
      {
        type: 'register_capability_txn',
        sender: skillMetadata.owner,
        data: encode(registration),
        onApprovalUri: this.config.onApprovalUri,
      },
      { gatewayUrl: this.gatewayOrigin() },
    );

    const registered: SkillMetadata = {
      ...skillMetadata,
      id: skillId,
      registeredAt: Date.now(),
      validationStatus: 'pending',
      performanceMetrics: { successRate: 0, averageLatency: 0, totalInvocations: 0, lastUpdated: Date.now() },
    };
    this.skills.set(skillId, registered);
    this.emit('skillRegistered', registered);
    return skillId;
  }

  public async registerLLMModel(_llmMetadata: Omit<LLMMetadata, 'id' | 'registeredAt'>): Promise<string> {
    throw unsupportedWrite('Registering an LLM model', 'KNIRV Chain has no LLM model registry');
  }

  public async transferNRN(_from: string, _to: string, _amount: string): Promise<string> {
    throw unsupportedWrite(
      'Transferring NRN',
      'KNIRV Chain does not accept plain transfers; they belong to the Transaction Chain, which has no transfer route yet',
    );
  }

  /** Current head of the chain, from `GET /chain/head`. */
  public async getNetworkConsensus(): Promise<NetworkConsensus> {
    const head = await this.chainGet<ChainHeadBody>('/chain/head');
    if (!head.data) {
      throw new Error('KNIRV Chain returned no head');
    }
    return {
      chainId: head.data.chain_id,
      blockHeight: head.data.height,
      blockHash: head.data.block_hash,
      timestamp: Date.now(),
    };
  }

  /** NRN balance of one account, from `GET /accounts/balance`. Throws on failure. */
  public async getNRNBalance(address: string): Promise<string> {
    const response = await this.chainGet<{ balance?: number }>('/accounts/balance', { address });
    return String(response.data?.balance ?? 0);
  }

  private startEventMonitoring(): void {
    this.stopEventMonitoring();
    this.monitorTimer = setInterval(() => {
      void this.checkForNewBlocks();
    }, 10000);
  }

  private stopEventMonitoring(): void {
    if (this.monitorTimer) {
      clearInterval(this.monitorTimer);
      this.monitorTimer = null;
    }
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

export default KNIRVChainIntegration;
