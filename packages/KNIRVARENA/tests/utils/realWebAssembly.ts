// Imported first by suites that run real WebAssembly: @cosmjs/crypto's libsodium
// instantiates it at import time, and the SDK's Rust core (signing, relay
// envelopes, model status) is a wasm-bindgen module that needs a working
// TextDecoder (config/jest.setup.js replaces it with a stub; tests/polyfills.ts replaces
// WebAssembly). Restore the real ones; TextEncoder is left alone because swapping it creates Uint8Arrays in another realm and breaks toEqual.
import { TextDecoder as NodeTextDecoder } from 'util';

const real = (global as unknown as { __nodeWebAssembly?: typeof WebAssembly }).__nodeWebAssembly;
if (real) Object.defineProperty(global, 'WebAssembly', { value: real, configurable: true, writable: true });
Object.defineProperty(global, 'TextDecoder', { value: NodeTextDecoder, configurable: true, writable: true });
export {};
