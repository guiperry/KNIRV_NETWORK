// Isomorphic core (`@knirvcorp/knirvbase-ts`): safe in browsers and Node.
// Types, vector clocks, CRDT resolution and the .nrv wire format (spec,
// bracket layout, frame codec, Go-schema registry serialisation).
export * from '../components/types/types';
export * from '../components/clock/vector_clock';
export * from '../components/resolver/crdt_resolver';
export * from '../components/storage/nrv/spec';
export * from '../components/storage/nrv/bracket';
export {
  Header, Frame, ThermoData, FrameIndexEntry, PQCManifest, Registry,
  encodeHeader, decodeHeader, encodeFrame, decodeFrame, createDefaultRegistry,
} from '../components/storage/nrv/codec';
export * from '../components/storage/nrv/registry_json';
