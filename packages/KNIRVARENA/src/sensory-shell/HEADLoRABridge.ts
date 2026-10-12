import { EventEmitter } from './EventEmitter';
import { getKnirvServerUrl } from '../config/runtimeConfig';
import { authToken } from '../services/serverLLM';
import type { HEADAdapterApplied, HEADAdapterBind } from './HEADBridge';

/**
 * Connects the real HEAD (KNIRVMODELS/head, `knirv-head/v1`) to KNIRVULORA.
 *
 * A uLoRA bundle holds a portable canonical core. The server binds it to this
 * HEAD version with the model's own connector (`POST /api/head/adapters/bind`),
 * giving per-role low-rank factors; the engine merges them into the attention
 * projections of the in-browser model (`W' = W + alpha/r · B·A`). There is no
 * per-frame weight "syncing": an adapter is applied once, and re-applied only
 * when the HEAD itself is replaced by a newly promoted version.
 *
 * Server-side `/api/head/reason` uses the base model; adapters affect only the
 * model running in this browser.
 *
 * The arena's EnhancedLoRAAdapter is a separate, small MLP adapter with its own
 * layers; its tensors have no correspondence to the HEAD's, so nothing is mapped
 * between them. `processWithLoRA` still runs the HEAD and then that adapter.
 */

/** A uLoRA role and the HEAD tensors it changes. */
export interface HEADLoRAMapping {
  role: string;
  headTensors: string[];
  applied: boolean;
}

export interface WeightSyncConfig {
  /** How often to check that the applied adapter still matches the loaded model (ms). */
  syncFrequency: number;
  /** Multiplier on the compiled adapter strength; 0 disables it. */
  strength: number;
  /** Which HEAD stacks to adapt. */
  stacks: Array<'l' | 'h'>;
}

/** The roles an HEAD can take, and the per-block tensor each one adapts. */
const ROLE_TENSOR: Record<string, string> = {
  'self_attn.q_proj': 'attn.q',
  'self_attn.k_proj': 'attn.k',
  'self_attn.v_proj': 'attn.v',
  'self_attn.o_proj': 'attn.o',
};

/** The subset of HEADBridge this class uses. */
export interface HEADAdapterHost {
  applyAdapter(bind: HEADAdapterBind): Promise<HEADAdapterApplied>;
  clearAdapter(): Promise<void>;
  getAdapter(): HEADAdapterApplied | null;
  getAdaptableTensors(): string[];
  getModelInfo(): { model_version?: string } | null;
  isLocal?(): boolean;
}

export interface HEADLoRABridgeDeps {
  fetchImpl?: typeof fetch;
}

export class HEADLoRABridge extends EventEmitter {
  private headBridge: HEADAdapterHost | null = null;
  private enhancedLoraAdapter: unknown = null;
  private mappings = new Map<string, HEADLoRAMapping>();
  private syncConfig: WeightSyncConfig;
  private isRunning = false;
  private timer: ReturnType<typeof setInterval> | null = null;
  private bundleId: string | null = null;
  private bound: { bundleId: string; contentHash: string; modelVersion: string } | null = null;
  private busy: Promise<void> | null = null;
  private readonly fetchImpl?: typeof fetch;

  constructor(syncConfig?: Partial<WeightSyncConfig>, deps: HEADLoRABridgeDeps = {}) {
    super();
    this.fetchImpl = deps.fetchImpl;
    this.syncConfig = { syncFrequency: 30_000, strength: 1, stacks: ['l', 'h'], ...syncConfig };
  }

  public setHEADBridge(headBridge: unknown): void {
    this.headBridge = headBridge as HEADAdapterHost;
  }

  public setEnhancedLoRAAdapter(enhancedLoraAdapter: unknown): void {
    this.enhancedLoraAdapter = enhancedLoraAdapter;
  }

  /** Chooses the uLoRA bundle to apply, and applies it now if the bridge is running. */
  public async setBundle(bundleId: string | null): Promise<void> {
    this.bundleId = bundleId;
    if (!bundleId) {
      await this.clear();
      return;
    }
    if (this.isRunning) await this.forceSyncNow();
  }

  public async start(): Promise<void> {
    if (!this.headBridge) throw new Error('HEAD bridge must be connected');
    this.refreshMappings();
    this.isRunning = true;
    this.timer = setInterval(() => {
      this.forceSyncNow().catch(() => { /* reported through 'syncError' */ });
    }, this.syncConfig.syncFrequency);
    this.emit('bridgeStarted');
    if (this.bundleId) await this.forceSyncNow().catch(() => { /* reported through 'syncError' */ });
  }

  public async stop(): Promise<void> {
    this.isRunning = false;
    if (this.timer) clearInterval(this.timer);
    this.timer = null;
    this.emit('bridgeStopped');
  }

  /**
   * Applies the chosen bundle if it is not already applied to the loaded model
   * version. Safe to call often: it does nothing when the adapter is current.
   */
  public async forceSyncNow(): Promise<void> {
    if (!this.isRunning) throw new Error('Bridge not running');
    if (this.busy) return this.busy;
    this.busy = this.sync().finally(() => { this.busy = null; });
    return this.busy;
  }

  private async sync(): Promise<void> {
    const host = this.headBridge;
    if (!host || !this.bundleId) return;
    const version = host.getModelInfo()?.model_version ?? '';
    if (host.isLocal && !host.isLocal()) return; // nothing to adapt: reasoning is on the server
    if (this.bound && this.bound.bundleId === this.bundleId && this.bound.modelVersion === version && host.getAdapter()) return;
    try {
      const bind = await this.fetchBind(this.bundleId);
      await this.applyBind(bind);
    } catch (error) {
      this.emit('syncError', error);
      throw error;
    }
  }

  private async fetchBind(bundleId: string): Promise<HEADAdapterBind> {
    const token = authToken();
    if (!token) throw new Error('sign in to bind an adapter');
    const res = await (this.fetchImpl ?? fetch)(`${getKnirvServerUrl()}/api/head/adapters/bind`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({ bundle_id: bundleId }),
    });
    let body: { success?: boolean; data?: HEADAdapterBind; error?: string } = {};
    try { body = await res.json(); } catch { /* non-JSON error body */ }
    if (!res.ok || body.success === false || !body.data) {
      throw new Error(body.error ?? `adapter bind failed (HTTP ${res.status})`);
    }
    return body.data;
  }

  private async applyBind(bind: HEADAdapterBind): Promise<HEADAdapterApplied> {
    const host = this.headBridge;
    if (!host) throw new Error('HEAD bridge must be connected');
    const applied = await host.applyAdapter({ ...bind, stacks: this.syncConfig.stacks, strength: this.syncConfig.strength });
    this.bound = {
      bundleId: this.bundleId ?? '',
      contentHash: applied.content_hash || bind.content_hash || '',
      modelVersion: host.getModelInfo()?.model_version ?? bind.model_version ?? '',
    };
    this.refreshMappings(applied);
    for (const role of applied.roles_applied) this.emit('mappingSynced', { mapping: this.mappings.get(role), timestamp: new Date() });
    this.emit('weightsSynced', { timestamp: new Date(), mappingsCount: applied.roles_applied.length, contentHash: this.bound.contentHash });
    return applied;
  }

  /** Removes any applied adapter and forgets the bundle. */
  public async clear(): Promise<void> {
    this.bound = null;
    this.bundleId = null;
    await this.headBridge?.clearAdapter();
    this.refreshMappings();
  }

  private refreshMappings(applied?: HEADAdapterApplied): void {
    this.mappings.clear();
    const tensors = this.headBridge?.getAdaptableTensors() ?? [];
    const stacks = new Set<string>(this.syncConfig.stacks);
    for (const [role, target] of Object.entries(ROLE_TENSOR)) {
      const names = tensors.filter((t) => t.includes(`.${target}.`) && stacks.has(t.split('.')[0]));
      if (names.length) this.mappings.set(role, { role, headTensors: names, applied: !!applied?.roles_applied.includes(role) });
    }
  }

  public getMappings(): Map<string, HEADLoRAMapping> {
    return new Map(this.mappings);
  }

  public getSyncConfig(): WeightSyncConfig {
    return { ...this.syncConfig, stacks: [...this.syncConfig.stacks] };
  }

  public updateSyncConfig(newConfig: Partial<WeightSyncConfig>): void {
    this.syncConfig = { ...this.syncConfig, ...newConfig };
    if (newConfig.syncFrequency && this.timer) {
      clearInterval(this.timer);
      this.timer = setInterval(() => { this.forceSyncNow().catch(() => { /* 'syncError' */ }); }, this.syncConfig.syncFrequency);
    }
    // Strength and stacks change the merged weights, so re-merge on the next sync.
    if (newConfig.strength !== undefined || newConfig.stacks) this.bound = null;
    this.emit('syncConfigUpdated', this.syncConfig);
  }

  public getStatus(): {
    isRunning: boolean;
    isConnected: boolean;
    headBridgeReady: boolean;
    bundleId: string | null;
    applied: HEADAdapterApplied | null;
    boundModelVersion: string | null;
    mappingsCount: number;
    syncConfig: WeightSyncConfig;
  } {
    return {
      isRunning: this.isRunning,
      isConnected: !!this.headBridge,
      headBridgeReady: !!this.headBridge && (this.headBridge.getModelInfo() !== null),
      bundleId: this.bundleId,
      applied: this.headBridge?.getAdapter() ?? null,
      boundModelVersion: this.bound?.modelVersion ?? null,
      mappingsCount: this.mappings.size,
      syncConfig: this.getSyncConfig(),
    };
  }

  // Required by AdaptiveLearningPipeline.
  public async processWithLoRA(data: unknown, loraConfig: Record<string, unknown>): Promise<unknown> {
    if (!this.headBridge || !this.enhancedLoraAdapter) throw new Error('HEAD Bridge or Enhanced LoRA Adapter not initialized');
    const headResult = await (this.headBridge as unknown as { process?: (d: unknown) => Promise<unknown> }).process?.(data);
    return (this.enhancedLoraAdapter as { adapt?: (r: unknown, c: unknown) => Promise<unknown> }).adapt?.(headResult, loraConfig);
  }

  /**
   * Applies already-bound uLoRA factors (`{adapter_rank, scaling_alpha, layers}`)
   * directly. Anything else is not HEAD weights and is refused.
   */
  public updateLoRAWeights(weights: Record<string, unknown>): void {
    const w = weights as Partial<HEADAdapterBind>;
    if (!w || typeof w.adapter_rank !== 'number' || typeof w.scaling_alpha !== 'number' || typeof w.layers !== 'object') {
      this.emit('syncError', new Error('updateLoRAWeights expects a KNIRVULORA bind response'));
      return;
    }
    this.applyBind(w as HEADAdapterBind).catch((error) => this.emit('syncError', error));
  }
}
