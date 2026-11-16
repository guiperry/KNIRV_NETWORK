// Runs the NIST ACVP ML-DSA-65 vectors and the KNIRVBASE frame-signing vector
// (testvectors/mldsa/) through the wasm-bindgen core's mldsa.* operations. The
// Rust core and the Go binding read the same files.
import { readFile } from "node:fs/promises";
import { mlDsa65PublicKey, mlDsa65Verify } from "../mldsa.js";

const hex = (s) => Uint8Array.from(s.match(/../g) ?? [], (b) => parseInt(b, 16));
const b64 = (s) => Uint8Array.from(Buffer.from(s, "base64"));
const load = async (name) =>
  JSON.parse(await readFile(new URL(`../../testvectors/mldsa/${name}`, import.meta.url), "utf8"));

const acvp = await load("ml_dsa65_acvp_v1.json");
let checked = 0;
for (const c of acvp.sig_ver) {
  const v = mlDsa65Verify(hex(c.pk), hex(c.message), hex(c.context), hex(c.signature));
  if (v.valid !== c.test_passed) {
    throw new Error(`ACVP sigVer tcId ${c.tc_id}: got ${JSON.stringify(v)}, want valid=${c.test_passed}`);
  }
  checked++;
}
for (const c of acvp.key_gen) {
  const pk = mlDsa65PublicKey(hex(c.seed));
  if (Buffer.from(pk).toString("hex") !== c.pk) {
    throw new Error(`ACVP keyGen tcId ${c.tc_id}: public key differs`);
  }
  checked++;
}

const frame = await load("ml_dsa65_frame_signing_v1.json");
const msg = b64(frame.signed_message_base64);
const pk = b64(frame.public_key_base64);
const sig = b64(frame.signature_base64);
if (!mlDsa65Verify(pk, msg, new Uint8Array(), sig).valid) throw new Error("circl frame signature must verify in the core");
const tampered = Uint8Array.from(msg);
tampered[tampered.length - 1] ^= 1;
if (mlDsa65Verify(pk, tampered, new Uint8Array(), sig).valid) throw new Error("tampered frame payload must fail");
checked += 2;

// Refusals are verdicts with the core's stable code, not exceptions.
const short = mlDsa65Verify(pk.subarray(0, pk.length - 1), msg, new Uint8Array(), sig);
if (short.valid || short.code !== "public_key_length") {
  throw new Error(`short public key: got ${JSON.stringify(short)}`);
}
checked++;

console.log(`mldsa ml-dsa-65 vectors: ${checked} cases`);
