// Hostile-input guard for the WASM core. A Rust panic compiles to `unreachable`
// (panic = "abort"), which traps the instance and breaks every later call, so a
// parser that panics on attacker-supplied bytes is a denial of service for any
// Worker verifying untrusted messages. This feeds malformed bytes to every
// parse/verify entry point and fails on any trap (plain errors are expected).
import assert from "node:assert/strict";
import { callLocal, toBase64 } from "../local.js";

let seed = 12345;
const rnd = () => (seed = (seed * 1103515245 + 12345) & 0x7fffffff) / 0x7fffffff;
const randBytes = (n) => Uint8Array.from({ length: n }, () => Math.floor(rnd() * 256));
// Structured seeds reach deeper than noise: valid tags with hostile lengths.
const seeds = [
  [0x0a, 0xff, 0xff, 0xff, 0xff, 0x0f], [0x0a, 0x05, 1], [0x08], [0x12, 0x00], [0x1a, 0x80], [0x09, 1, 2, 3], [0x0d, 1],
  [0x08, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01],
  [0x0a, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01],
  [0xfa, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01],
];
const parsers = [
  ["signing.parse_message_envelope", "bytes_base64"], ["relay.parse", "bytes_base64"],
  ["wasm.assignment.parse", "bytes_base64"], ["wasm.publication.parse", "bytes_base64"],
  ["signing.parse_action_body", "body_base64"], ["signing.parse_sequence", "auth_info_base64"],
];
let calls = 0, traps = 0;
const attempt = (op, payload) => {
  calls++;
  try { callLocal(op, payload); } catch (error) {
    if (error instanceof WebAssembly.RuntimeError) { traps++; console.error("TRAP", op, error.message); }
  }
};
for (let i = 0; i < 1500; i++) {
  const data = i < seeds.length ? Uint8Array.from(seeds[i]) : randBytes(Math.floor(rnd() * 64));
  for (const [op, field] of parsers) attempt(op, { [field]: toBase64(data) });
}
for (let i = 0; i < 400; i++) {
  const junk = toBase64(randBytes(Math.floor(rnd() * 70)));
  attempt("address.encode", { public_key_base64: junk });
  attempt("address.decode", { address: String.fromCharCode(...randBytes(12).map((b) => 33 + (b % 90))) });
  attempt("signing.public_key", { private_key_base64: junk });
  attempt("signing.verify_message", { signed: { envelope: junk, signature: junk, public_key: junk, address: "" }, domain: "d", purpose: "p", chain_id: "c", nonce: "n", now_unix: 1 });
  attempt("signing.verify_transaction", { transaction: { body_bytes: junk, auth_info_bytes: junk, signatures: [junk], public_key: junk, address: "", hash: "" }, chain_id: "c", account_number: 1 });
  attempt("uri.parse", { raw: String.fromCharCode(...randBytes(30).map((b) => b % 128)) });
}
assert.equal(traps, 0, `${traps} WASM traps (Rust panics) on hostile input`);
// The instance must still be usable afterwards.
assert.equal(callLocal("uri.parse", { raw: "knirv://abc.chain/" }).kind, "canonical");
console.log(`no WASM traps across ${calls} hostile calls`);
