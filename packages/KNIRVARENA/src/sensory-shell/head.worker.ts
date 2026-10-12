/// <reference lib="webworker" />
// Hosts the knirv-head WASM module (built by `npm run build:head`).
import init, { HEADCognitive } from '../wasm-pkg/knirv_head/knirv_head.js';

let model: HEADCognitive | null = null;
let ready: Promise<unknown> | null = null;

self.onmessage = async (ev: MessageEvent) => {
  const msg = ev.data as
    | { id: number; type: 'load'; configJson: string; weights: Uint8Array; adapterJson?: string }
    | { id: number; type: 'reason'; tokens: number[] };
  try {
    ready ??= init();
    await ready;
    if (msg.type === 'load') {
      const next = new HEADCognitive();
      // A failed adapter leaves the previous model in place (nothing is swapped until load succeeds).
      const applied = msg.adapterJson
        ? JSON.parse(next.load_model_with_adapter(msg.configJson, msg.weights, msg.adapterJson))
        : (next.load_model(msg.configJson, msg.weights), undefined);
      model?.free();
      model = next;
      const info = JSON.parse(next.get_model_info());
      info.adaptable_tensors = next.adaptable_tensors();
      if (applied) info.adapter = applied;
      (self as unknown as Worker).postMessage({ id: msg.id, ok: true, result: info });
    } else {
      if (!model) throw new Error('HEAD model not loaded');
      (self as unknown as Worker).postMessage({
        id: msg.id, ok: true, result: JSON.parse(model.reason(new Uint32Array(msg.tokens))),
      });
    }
  } catch (e) {
    (self as unknown as Worker).postMessage({ id: msg.id, ok: false, error: e instanceof Error ? e.message : String(e) });
  }
};
