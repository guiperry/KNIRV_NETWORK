// Runs the shared Ed25519 evidence vectors (testvectors/evidence/ed25519_v1.json)
// through the wasm-bindgen core's evidence.* operations. The Rust core and the Go
// binding read the same file; Go's crypto/ed25519 produced the expected results.
import { readFile } from "node:fs/promises";
import { callLocal } from "../local.js";

const fixture = JSON.parse(
  await readFile(new URL("../../testvectors/evidence/ed25519_v1.json", import.meta.url), "utf8"),
);

let checked = 0;
for (const c of fixture.verify) {
  const verdict = callLocal("evidence.ed25519_verify", {
    public_key_base64: c.public_key_base64,
    message_base64: c.message_base64,
    signature_base64: c.signature_base64,
  });
  if (verdict.valid !== c.valid || verdict.code !== c.code) {
    throw new Error(`ed25519 verify ${c.name}: got ${JSON.stringify(verdict)}, want valid=${c.valid} code=${c.code}`);
  }
  checked++;
}
for (const c of fixture.public_key) {
  const verdict = callLocal("evidence.ed25519_public_key", { public_key_base64: c.public_key_base64 });
  if (verdict.valid !== c.valid || verdict.code !== c.code) {
    throw new Error(`ed25519 public key ${c.name}: got ${JSON.stringify(verdict)}, want valid=${c.valid} code=${c.code}`);
  }
  if (c.canonical_base64 && verdict.public_key_base64 !== c.canonical_base64) {
    throw new Error(`ed25519 public key ${c.name}: canonical form differs`);
  }
  checked++;
}
console.log(`evidence ed25519 vectors: ${checked} cases`);
