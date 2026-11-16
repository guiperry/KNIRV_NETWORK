// Stub for the knirv_head package entry, used when the HEAD WASM has not been built.
//
// Tracked source: scripts/templates/knirv_head/index.js. scripts/build-head.sh copies
// this file into src/wasm-pkg/knirv_head/ after wasm-pack runs. src/wasm-pkg/ is
// gitignored build output, so the copy is never committed; this template is.
//
// wasm-pack does NOT write this file. It writes knirv_head.js, knirv_head_bg.wasm and
// knirv_head.d.ts, and package.json declares "main": "knirv_head.js". The browser
// worker imports knirv_head.js directly (src/sensory-shell/head.worker.ts). Nothing
// imports this stub, so it only documents the unbuilt state for anyone who reads it.
//
// All exports throw so an unbuilt HEAD is loud instead of silently mock-shaped.
const notBuilt = (name) => () => {
  throw new Error(`knirv_head.${name} is not available: the HEAD WASM module has not been built. Run npm run build:head, or use POST /api/heart/decide for server-side inference.`);
};
export const initialize_modules = notBuilt('initialize_modules');
export const process_cognitive_input = notBuilt('process_cognitive_input');
export const load_weights = notBuilt('load_weights');
export const get_model_info = notBuilt('get_model_info');
export default { initialize_modules, process_cognitive_input, load_weights, get_model_info };
