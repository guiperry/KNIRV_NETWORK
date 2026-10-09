// three-stdlib instantiates its Meshopt WebAssembly decoder on import. The
// test WebAssembly polyfill has no real exports, so the rejected promise
// crashed the Jest process. Tests never decode compressed meshes.
const MeshoptDecoder = {
  supported: false,
  ready: Promise.resolve(),
  useWorkers: () => undefined,
  decodeVertexBuffer: () => undefined,
  decodeIndexBuffer: () => undefined,
  decodeIndexSequence: () => undefined,
  decodeGltfBuffer: () => undefined,
};
module.exports = { MeshoptDecoder };
