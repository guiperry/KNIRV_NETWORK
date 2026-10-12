import { EventEmitter } from './EventEmitter';
import { getKnirvServerUrl } from '../config/runtimeConfig';
import { authToken } from '../services/serverLLM';
import { BYTE_V1, decodeTokens, encodeSamples, encodeText, padTo } from './headTokenizer';
import { HEADEngine, HEADEngineInfo, HEADReasoning, HEADDecision, HEADWorkerEngine, HEADAdapterApplied } from './headEngine';
import { fetchLatestModel, HEADModelUnavailableError } from './headModelStore';

export type { HEADDecision, HEADReasoning, HEADAdapterApplied } from './headEngine';

/** A KNIRVULORA `bind` response as returned by `POST /api/head/adapters/bind`. */
export interface HEADAdapterBind {
  adapter_rank: number;
  scaling_alpha: number;
  layers: Record<string, { a: number[][]; b: number[][] }>;
  content_hash?: string;
  /** The HEAD version the connector was derived for; must match the loaded model. */
  model_version?: string;
  /** Optional local controls: which stacks to adapt and how strongly. */
  stacks?: Array<'l' | 'h'>;
  strength?: number;
}

export interface HEADCognitiveInput {
  sensory_data: number[];
  context: string;
  task_type: string;
}

export interface HEADCognitiveOutput {
  reasoning_result: string;
  confidence: number;
  processing_time: number;
  l_module_activations: number[];
  h_module_activations: number[];
  /** High-level cycles actually used (ACT). */
  halting_steps: number;
  /** The model's plan, detokenised. */
  plan: string;
  plan_tokens: number[];
  decisions: HEADDecision[];
  model_version: string;
  /** Where the answer was computed. */
  source: 'wasm' | 'server';
}

export interface HEADModelInfo {
  total_parameters: number;
  l_modules: number;
  h_modules: number;
  model_version?: string;
  tokenizer?: string;
}

export interface HEADConfig {
  /** Retained for callers; the real depth comes from the model's own config. */
  l_module_count: number;
  h_module_count: number;
  enable_adaptation: boolean;
  processing_timeout: number;
}

/** No model could answer. Callers must not substitute made-up reasoning. */
export class HEADUnavailableError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'HEADUnavailableError';
  }
}

export interface HEADBridgeDeps {
  engine?: HEADEngine;
  fetchImpl?: typeof fetch;
  fetchModel?: typeof fetchLatestModel;
}

type Mode = 'uninitialized' | 'wasm' | 'server';

export class HEADBridge extends EventEmitter {
  private engine: HEADEngine | null = null;
  private mode: Mode = 'uninitialized';
  private info: HEADEngineInfo | null = null;
  /** The verified base model, kept so an adapter can be merged into (or removed from) it. */
  private base: { configJson: string; weights: Uint8Array } | null = null;
  private config: HEADConfig;
  private readonly deps: HEADBridgeDeps;

  constructor(config: HEADConfig, deps: HEADBridgeDeps = {}) {
    super();
    this.config = config;
    this.deps = deps;
  }

  /**
   * Loads the latest verified model into a worker. If that is impossible
   * (not signed in, no model published, unsupported tokenizer, WASM failure)
   * the bridge uses the server endpoint instead; it never fabricates output.
   */
  public async initialize(): Promise<void> {
    try {
      const files = await (this.deps.fetchModel ?? fetchLatestModel)(this.deps.fetchImpl);
      const tokenizer = String(files.manifest.config.tokenizer ?? '');
      if (tokenizer !== BYTE_V1) {
        throw new HEADModelUnavailableError(`tokenizer ${tokenizer || '(none)'} is not available in the browser`);
      }
      const engine = this.deps.engine ?? new HEADWorkerEngine();
      const configJson = JSON.stringify(files.manifest.config);
      this.info = await engine.load(configJson, files.weights);
      this.base = { configJson, weights: files.weights };
      this.engine = engine;
      this.mode = 'wasm';
      this.emit('initialized');
    } catch (error) {
      this.engine?.dispose();
      this.engine = null;
      this.info = null;
      this.base = null;
      this.mode = 'server';
      this.emit('unavailable', error);
    }
  }

  /** Replaces the loaded model. `configJson` is `head_config.json`. */
  public async loadModelWeights(weightsData: Uint8Array, configJson?: string): Promise<boolean> {
    if (!configJson) return false;
    try {
      const engine = this.engine ?? this.deps.engine ?? new HEADWorkerEngine();
      this.info = await engine.load(configJson, weightsData);
      this.base = { configJson, weights: weightsData };
      this.engine = engine;
      this.mode = 'wasm';
      this.emit('weightsLoaded');
      return true;
    } catch (error) {
      this.emit('error', error);
      return false;
    }
  }

  /**
   * Merges a KNIRVULORA adapter into the in-browser model. Server-side reasoning
   * (`/api/head/reason`) always uses the base model. A rejected adapter leaves the
   * current model exactly as it was.
   */
  public async applyAdapter(bind: HEADAdapterBind): Promise<HEADAdapterApplied> {
    if (this.mode !== 'wasm' || !this.engine || !this.base) {
      throw new HEADUnavailableError('adapters apply only to the model running in this browser');
    }
    if (bind.model_version && bind.model_version !== this.info?.model_version) {
      throw new HEADUnavailableError(`adapter was bound for model ${bind.model_version}, but ${this.info?.model_version ?? 'none'} is loaded`);
    }
    const info = await this.engine.load(this.base.configJson, this.base.weights, JSON.stringify(bind));
    this.info = info;
    this.emit('adapterApplied', info.adapter);
    return info.adapter as HEADAdapterApplied;
  }

  /** Reloads the base model without any adapter. */
  public async clearAdapter(): Promise<void> {
    if (this.mode !== 'wasm' || !this.engine || !this.base) return;
    this.info = await this.engine.load(this.base.configJson, this.base.weights);
    this.emit('adapterCleared');
  }

  /** The adapter currently merged into the model, if any. */
  public getAdapter(): HEADAdapterApplied | null {
    return this.info?.adapter ?? null;
  }

  /** Weight names an adapter can change in the loaded model. */
  public getAdaptableTensors(): string[] {
    return this.info?.adaptable_tensors ?? [];
  }

  public async processCognitiveInput(input: HEADCognitiveInput): Promise<HEADCognitiveOutput> {
    if (this.mode === 'uninitialized') throw new HEADUnavailableError('HEAD bridge not initialized');
    const started = performance.now();
    const maxLen = this.info?.max_input_len ?? 512;
    // The model is trained on a fixed layout: BOS + content, PAD-filled to the full width.
    const tokens = padTo(input.task_type === 'text_processing' || input.sensory_data.length === 0
      ? encodeText(input.context, maxLen)
      : encodeSamples(input.sensory_data, maxLen), maxLen);

    let raw: HEADReasoning;
    let source: 'wasm' | 'server';
    try {
      if (this.mode === 'wasm' && this.engine) {
        raw = await this.engine.reason(tokens);
        source = 'wasm';
      } else {
        raw = await this.reasonOnServer(tokens);
        source = 'server';
      }
    } catch (error) {
      this.emit('error', error);
      throw error instanceof HEADUnavailableError ? error : new HEADUnavailableError(error instanceof Error ? error.message : String(error));
    }

    const output = this.toOutput(raw, source, performance.now() - started);
    this.emit('inputProcessed', { input, output, processingTime: output.processing_time });
    return output;
  }

  private async reasonOnServer(tokens: number[]): Promise<HEADReasoning> {
    const token = authToken();
    if (!token) throw new HEADUnavailableError('HEAD unavailable: sign in to use server-side reasoning');
    const res = await (this.deps.fetchImpl ?? fetch)(`${getKnirvServerUrl()}/api/head/reason`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({ tokens, tokenizer: BYTE_V1 }),
    });
    let body: { success?: boolean; data?: HEADReasoning; error?: string } = {};
    try { body = await res.json(); } catch { /* non-JSON error body */ }
    if (!res.ok || body.success === false || !body.data) {
      throw new HEADUnavailableError(body.error ?? `HEAD unavailable (HTTP ${res.status})`);
    }
    return body.data;
  }

  private toOutput(raw: HEADReasoning, source: 'wasm' | 'server', ms: number): HEADCognitiveOutput {
    const plan = decodeTokens(raw.plan_tokens);
    const parts = raw.decisions.map((d) =>
      d.kind === 'score'
        ? `${d.name}=${(d.value ?? 0).toFixed(3)}`
        : `${d.name}=${d.abstain ? 'abstain' : d.selected} (${(d.confidence ?? 0).toFixed(2)})`);
    return {
      reasoning_result: parts.join('; '),
      confidence: raw.confidence,
      processing_time: ms,
      l_module_activations: raw.l_module_activations,
      h_module_activations: raw.h_module_activations,
      halting_steps: raw.halting_steps,
      plan,
      plan_tokens: raw.plan_tokens,
      decisions: raw.decisions,
      model_version: raw.model_version,
      source,
    };
  }

  public getModelInfo(): HEADModelInfo | null {
    if (!this.info) return null;
    const { total_parameters, l_modules, h_modules, model_version, tokenizer } = this.info;
    return { total_parameters, l_modules, h_modules, model_version, tokenizer };
  }

  /** True when a model can answer: loaded locally, or the server endpoint will be tried. */
  public isReady(): boolean {
    return this.mode === 'wasm' || this.mode === 'server';
  }

  /** True only when the model runs in this browser. */
  public isLocal(): boolean {
    return this.mode === 'wasm';
  }

  public async destroy(): Promise<void> {
    this.engine?.dispose();
    this.engine = null;
    this.info = null;
    this.base = null;
    this.mode = 'uninitialized';
    this.emit('destroyed');
  }

  public async processTextInput(text: string, context?: unknown): Promise<HEADCognitiveOutput> {
    return this.processCognitiveInput({
      sensory_data: [],
      context: text || JSON.stringify(context ?? {}),
      task_type: 'text_processing',
    });
  }

  public async processVoiceInput(audioData: number[], context?: unknown): Promise<HEADCognitiveOutput> {
    return this.processCognitiveInput({ sensory_data: audioData, context: JSON.stringify(context ?? {}), task_type: 'voice_processing' });
  }

  public async processVisualInput(visualData: number[], context?: unknown): Promise<HEADCognitiveOutput> {
    return this.processCognitiveInput({ sensory_data: visualData, context: JSON.stringify(context ?? {}), task_type: 'visual_processing' });
  }

  public getConfig(): HEADConfig {
    return { ...this.config };
  }

  public updateConfig(newConfig: Partial<HEADConfig>): void {
    this.config = { ...this.config, ...newConfig };
    this.emit('configUpdated', this.config);
  }

  // Required by the AdaptiveLearningPipeline interface.
  public async process(data: unknown): Promise<unknown> {
    if (typeof data === 'object' && data !== null && 'sensory_data' in data) {
      return this.processCognitiveInput(data as HEADCognitiveInput);
    }
    if (Array.isArray(data)) return this.processCognitiveInput({ sensory_data: data as number[], context: '', task_type: 'general' });
    return this.processTextInput(typeof data === 'string' ? data : JSON.stringify(data));
  }

  public isConnected(): boolean {
    return this.isReady();
  }
}
