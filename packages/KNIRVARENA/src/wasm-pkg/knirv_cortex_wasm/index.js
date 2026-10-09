// Placeholder — replaced by the wasm-pack output of rust-wasm (knirv-hrm)
// when `npm run build:hrm` runs. All exports throw so an unbuilt HRM is
// loud instead of silently mock-shaped.
const notBuilt = (name) => () => {
  throw new Error(`knirv_cortex_wasm.${name} is not available: the HRM WASM module has not been built. Run npm run build:hrm, or use POST /api/hrm/reason for server-side inference.`);
};
export const initialize_modules = notBuilt('initialize_modules');
export const process_cognitive_input = notBuilt('process_cognitive_input');
export const load_weights = notBuilt('load_weights');
export const get_model_info = notBuilt('get_model_info');
export default { initialize_modules, process_cognitive_input, load_weights, get_model_info };
