// Imported first by suites that sign with @cosmjs/crypto: its libsodium
// instantiates WebAssembly at import time, and tests/polyfills.ts replaces the
// global WebAssembly with a stub. Restore the real runtime for this suite.
const real = (global as unknown as { __nodeWebAssembly?: typeof WebAssembly }).__nodeWebAssembly;
if (real) Object.defineProperty(global, 'WebAssembly', { value: real, configurable: true, writable: true });
export {};
