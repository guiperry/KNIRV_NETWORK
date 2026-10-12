// Placeholder — replaced by the wasm-pack output of rust-wasm (knirv-head)
// when `npm run build:head` runs. All exports throw so an unbuilt HEAD is
// loud instead of silently mock-shaped.
const notBuilt = (name) => () => {
  throw new Error(`knirv_head.${name} is not available: the HEAD WASM module has not been built. Run npm run build:head, or use POST /api/head/reason for server-side inference.`);
};
export const initialize_modules = notBuilt('initialize_modules');
export const process_cognitive_input = notBuilt('process_cognitive_input');
export const load_weights = notBuilt('load_weights');
export const get_model_info = notBuilt('get_model_info');
export default { initialize_modules, process_cognitive_input, load_weights, get_model_info };
