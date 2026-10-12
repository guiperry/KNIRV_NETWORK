/** Raw `Reasoning` JSON produced by the knirv-hrm crate (KNIRVINFERENCER/hrm/src/model.rs). */
export interface HRMDecision {
  name: string;
  kind: 'choice' | 'score';
  options: string[];
  probabilities: number[];
  selected: string | null;
  value: number | null;
  confidence: number | null;
  abstain: boolean;
}

export interface HRMReasoning {
  plan_tokens: number[];
  halting_steps: number;
  max_steps: number;
  halt_probability: number;
  confidence: number;
  decisions: HRMDecision[];
  l_module_activations: number[];
  h_module_activations: number[];
  model_version: string;
  dataset_manifest_hash: string;
  policy_bundle_hash: string;
}

export interface HRMEngineInfo {
  loaded: boolean;
  total_parameters: number;
  l_modules: number;
  h_modules: number;
  max_input_len?: number;
  vocab_size?: number;
  tokenizer?: string;
  model_version?: string;
  /** Weights a uLoRA adapter can change (e.g. `h.0.attn.q.weight`). */
  adaptable_tensors?: string[];
  /** Present when a uLoRA adapter was merged into the loaded weights. */
  adapter?: HRMAdapterApplied;
}

/** What the model did with an adapter (KNIRVINFERENCER/hrm/src/adapter.rs `Applied`). */
export interface HRMAdapterApplied {
  content_hash: string;
  tensors_changed: number;
  roles_applied: string[];
  roles_skipped: string[];
}

/** Runs the model somewhere that cannot block the 3D scene. */
export interface HRMEngine {
  /** `adapterJson` is a KNIRVULORA bind response; it is merged into the weights at load. */
  load(configJson: string, weights: Uint8Array, adapterJson?: string): Promise<HRMEngineInfo>;
  reason(tokens: number[]): Promise<HRMReasoning>;
  dispose(): void;
}

type WorkerRequest =
  | { id: number; type: 'load'; configJson: string; weights: Uint8Array; adapterJson?: string }
  | { id: number; type: 'reason'; tokens: number[] };
type WorkerResponse = { id: number; ok: true; result: unknown } | { id: number; ok: false; error: string };

/** Web Worker engine: the WASM module and model live off the main thread. */
export class HRMWorkerEngine implements HRMEngine {
  private worker: Worker | null = null;
  private seq = 0;
  private pending = new Map<number, { resolve: (v: unknown) => void; reject: (e: Error) => void }>();

  private ensure(): Worker {
    if (this.worker) return this.worker;
    const w = new Worker(new URL('./hrm.worker.ts', import.meta.url), { type: 'module' });
    w.onmessage = (ev: MessageEvent<WorkerResponse>) => {
      const p = this.pending.get(ev.data.id);
      if (!p) return;
      this.pending.delete(ev.data.id);
      if (ev.data.ok) p.resolve(ev.data.result); else p.reject(new Error(ev.data.error));
    };
    w.onerror = (ev) => this.failAll(new Error(ev.message || 'HRM worker crashed'));
    this.worker = w;
    return w;
  }

  private failAll(err: Error): void {
    this.pending.forEach((p) => p.reject(err));
    this.pending.clear();
    this.worker?.terminate();
    this.worker = null;
  }

  private call<T>(msg: DistributiveOmit<WorkerRequest, 'id'>, transfer: Transferable[] = []): Promise<T> {
    const id = ++this.seq;
    const w = this.ensure();
    return new Promise<T>((resolve, reject) => {
      this.pending.set(id, { resolve: resolve as (v: unknown) => void, reject });
      w.postMessage({ ...msg, id }, transfer);
    });
  }

  load(configJson: string, weights: Uint8Array, adapterJson?: string): Promise<HRMEngineInfo> {
    // Copy: the caller keeps its buffer (it may be cached).
    const copy = weights.slice();
    return this.call<HRMEngineInfo>({ type: 'load', configJson, weights: copy, adapterJson }, [copy.buffer]);
  }

  reason(tokens: number[]): Promise<HRMReasoning> {
    return this.call<HRMReasoning>({ type: 'reason', tokens });
  }

  dispose(): void {
    this.failAll(new Error('HRM engine disposed'));
  }
}

type DistributiveOmit<T, K extends keyof never> = T extends unknown ? Omit<T, K> : never;
