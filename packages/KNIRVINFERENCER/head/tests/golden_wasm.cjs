// Runs the golden fixture through the wasm-bindgen build (Node) and compares to native output.
// Usage: node tests/golden_wasm.cjs <path-to-wasm-pack-nodejs-pkg>
const fs = require('fs');
const path = require('path');
const pkg = require(path.resolve(process.argv[2]));
const dir = __dirname;
const want = JSON.parse(fs.readFileSync(path.join(dir, 'golden.json'), 'utf8'));
const m = new pkg.HEADCognitive();
if (m.is_loaded()) throw new Error('loaded before load_model');
try { m.reason(new Uint32Array([1])); throw new Error('reason before load must fail'); } catch (e) { if (!/not loaded/.test(String(e.message || e))) throw e; }
m.load_model(fs.readFileSync(path.join(dir, 'golden_config.json'), 'utf8'), fs.readFileSync(path.join(dir, 'golden.safetensors')));
const got = JSON.parse(m.reason(new Uint32Array([3, 1, 4, 1, 5, 9, 2, 6])));
const close = (a, b, what) => {
  if (a.length !== b.length) throw new Error(`${what}: length ${a.length} vs ${b.length}`);
  a.forEach((x, i) => { if (Math.abs(x - b[i]) > 1e-5 * Math.max(1, Math.abs(b[i]))) throw new Error(`${what}[${i}]: ${x} vs ${b[i]}`); });
};
if (JSON.stringify(got.plan_tokens) !== JSON.stringify(want.plan_tokens)) throw new Error('plan_tokens differ');
if (got.halting_steps !== want.halting_steps) throw new Error('halting_steps differ');
close(got.l_module_activations, want.l_module_activations, 'l_act');
close(got.h_module_activations, want.h_module_activations, 'h_act');
close([got.confidence, got.halt_probability], [want.confidence, want.halt_probability], 'scalars');
got.decisions.forEach((d, i) => close(d.probabilities.concat(d.value ?? 0), want.decisions[i].probabilities.concat(want.decisions[i].value ?? 0), `decision ${d.name}`));
try { m.load_model('{}', new Uint8Array(4)); throw new Error('bad config must fail'); } catch (e) { if (/must fail/.test(String(e.message))) throw e; }
console.log('wasm golden OK; params', JSON.parse(m.get_model_info()).total_parameters);
