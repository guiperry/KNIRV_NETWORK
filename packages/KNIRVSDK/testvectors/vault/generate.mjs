#!/usr/bin/env node
// Captures golden vectors from the controller's CURRENT JavaScript vault worker
// logic (KNIRV_CORP/packages/controller/src/react-app/wasm/vaultCrypto.worker.ts),
// with every random input injected, so the Rust vault can be held to bit-for-bit
// compatibility. The primitives are the ones the worker imports (@cosmjs/crypto,
// WebCrypto AES-GCM, crypto-js), resolved from the controller's node_modules.
//
//   VAULT_CONTROLLER_DIR=../../../../KNIRV_CORP/packages/controller node generate.mjs
//
// Output: vectors.json next to this file. Regenerate only if the JS worker's
// observable behaviour is meant to change; the Rust tests pin it.
import { createRequire } from "node:module";
import { webcrypto } from "node:crypto";
import { writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const controller = resolve(here, process.env.VAULT_CONTROLLER_DIR ?? "../../../../../KNIRV_CORP/packages/controller");
const require = createRequire(`${controller}/package.json`);
const { Bip39, EnglishMnemonic, Slip10, Slip10Curve, Slip10RawIndex, Secp256k1, sha256, ripemd160, Argon2id } = require("@cosmjs/crypto");
const { bech32 } = require("bech32");
const CryptoJS = require("crypto-js");
const signing = await import(pathToFileURL(resolve(here, "../../npm-package/signing.js")).href);
await signing.initLocal();

const KEY_BYTES = 32;
const ARGON = { outputLength: KEY_BYTES, opsLimit: 3, memLimitKib: 64 * 1024 };
const hex = (b) => Buffer.from(b).toString("hex");
const b64 = (b) => Buffer.from(b).toString("base64");
const fixed = (n, start) => Uint8Array.from({ length: n }, (_, i) => (start + i * 7) & 255);

const parse = (path) => path.split("/").filter((p) => p && p !== "m").map((p) => {
  const h = p.endsWith("'");
  const v = Number.parseInt(h ? p.slice(0, -1) : p, 10);
  return h ? Slip10RawIndex.hardened(v) : Slip10RawIndex.normal(v);
});
async function derive(seed, index) {
  const { privkey } = Slip10.derivePath(Slip10Curve.Secp256k1, seed, parse(signing.KNIRV_HD_PATH.replace("i", String(index))));
  const { pubkey } = await Secp256k1.makeKeypair(privkey);
  return { privkey, pub: Secp256k1.compressPubkey(pubkey) };
}
const address = (pub) => bech32.encode("knirv", bech32.toWords(ripemd160(sha256(pub))));

async function seal(payload, password, salt, nonce, createdAt) {
  const key = await Argon2id.execute(password, salt, ARGON);
  const ck = await webcrypto.subtle.importKey("raw", key, "AES-GCM", false, ["encrypt"]);
  const ct = new Uint8Array(await webcrypto.subtle.encrypt({ name: "AES-GCM", iv: nonce }, ck, new TextEncoder().encode(payload)));
  const envelope = {
    version: 2, migration_version: 1, algorithm: "argon2id+aes-256-gcm",
    kdf: { name: "argon2id", output_length: KEY_BYTES, ops_limit: ARGON.opsLimit, mem_limit_kib: ARGON.memLimitKib },
    salt: b64(salt), nonce: b64(nonce), ciphertext: b64(ct), created_at: createdAt,
  };
  return `knirv-vault-v2.${Buffer.from(JSON.stringify(envelope)).toString("base64")}`;
}

const tx = {
  action: { action: "transfer", sender: "knirv1sender", recipient: "knirv1recipient", amount: 5n, payload: Uint8Array.from([1, 2, 3]), timestampUnix: 1700000000n },
  chainId: "knirv-test-1", accountNumber: 7n, sequence: 3n, fee: { denom: "nrn", amount: "10", gasLimit: 200000n },
};
const message = { domain: "knirv.vault.vector", purpose: "golden", chainId: "knirv-test-1", nonce: "n-1", issuedAtUnix: 1700000000n, expiresAtUnix: 1700003600n, payload: Uint8Array.from([9, 8, 7]) };
// JSON-safe copies recorded in the vectors, in the SDK's wire (snake_case) form.
const wire = {
  tx: { action: { action: "transfer", sender: "knirv1sender", recipient: "knirv1recipient", amount: 5, payload: [1, 2, 3], timestamp_unix: 1700000000 }, chain_id: "knirv-test-1", account_number: 7, sequence: 3, fee: { denom: "nrn", amount: "10", gas_limit: 200000 } },
  message: { domain: "knirv.vault.vector", purpose: "golden", chain_id: "knirv-test-1", nonce: "n-1", issued_at_unix: 1700000000, expires_at_unix: 1700003600, payload: [9, 8, 7] },
};

const mnemonics = [
  { name: "abandon-12", mnemonic: "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about" },
  { name: "entropy-16", entropy: hex(fixed(16, 3)) },
  { name: "entropy-32", entropy: hex(fixed(32, 11)) },
];
const out = { hd_path: signing.KNIRV_HD_PATH, argon2id: ARGON, accounts: [], seal: [], errors: [] };

for (const m of mnemonics) {
  const mnemonic = m.mnemonic ?? Bip39.encode(Uint8Array.from(Buffer.from(m.entropy, "hex"))).toString();
  const seed = await Bip39.mnemonicToSeed(new EnglishMnemonic(mnemonic));
  const entry = { name: m.name, entropy: m.entropy, mnemonic, seed: hex(seed), derived: [] };
  for (const index of [0, 1, 2, 5]) {
    const { privkey, pub } = await derive(seed, index);
    const d = { index, public_key: b64(pub), address: address(pub) };
    if (index === 0 || index === 1) {
      d.sign_tx = await signing.signDirectTransaction(privkey, tx);
      d.sign_msg = await signing.signMessageEnvelope(privkey, message);
    }
    entry.derived.push(d);
  }
  out.accounts.push(entry);
}
out.sign_inputs = wire;

const password = "correct horse battery staple";
for (const [i, a] of out.accounts.entries()) {
  const salt = fixed(16, 40 + i), nonce = fixed(12, 90 + i), created_at = `2026-10-3${i}T00:00:00.000Z`;
  const payload = JSON.stringify({ mnemonic: a.mnemonic });
  const serialized = await seal(payload, password, salt, nonce, created_at);
  out.seal.push({ account: a.name, password, salt: hex(salt), nonce: hex(nonce), created_at, payload, serialized });
}
// An unusual-but-accepted password (non-ASCII) pins UTF-8 handling.
{
  const salt = fixed(16, 200), nonce = fixed(12, 210), created_at = "2026-10-31T12:00:00.000Z";
  const payload = JSON.stringify({ mnemonic: out.accounts[0].mnemonic });
  out.seal.push({ account: "abandon-12", password: "pässwörd-密码-🔐", salt: hex(salt), nonce: hex(nonce), created_at, payload, serialized: await seal(payload, "pässwörd-密码-🔐", salt, nonce, created_at) });
}

// Error cases the Rust vault must also reject.
let invalid;
try { new EnglishMnemonic("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon"); } catch (e) { invalid = e.message; }
out.errors.push({ case: "bad-checksum-mnemonic", mnemonic: "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon", js_error: invalid });
try { new EnglishMnemonic("notaword ".repeat(12).trim()); } catch (e) { invalid = e.message; }
out.errors.push({ case: "unknown-words-mnemonic", mnemonic: "notaword ".repeat(12).trim(), js_error: invalid });
out.errors.push({ case: "wrong-password", serialized: out.seal[0].serialized, password: "wrong" });
{
  const salt = fixed(16, 1), nonce = fixed(12, 2);
  const key = await Argon2id.execute(password, salt, ARGON);
  const ck = await webcrypto.subtle.importKey("raw", key, "AES-GCM", false, ["decrypt"]);
  let rejected = false;
  try { await webcrypto.subtle.decrypt({ name: "AES-GCM", iv: nonce }, ck, fixed(40, 0)); } catch { rejected = true; }
  out.errors.push({ case: "tampered-ciphertext-rejected-by-aes-gcm", rejected });
}
// Whitespace normalisation the worker applies to imported mnemonics.
out.normalisation = { input: "  abandon  abandon abandon abandon\tabandon abandon abandon abandon abandon abandon abandon   about\n", normalised: out.accounts[0].mnemonic };

// One legacy crypto-js vault (random salt, so a fixed captured string). The
// legacy path stays in the controller's JS; this pins its migration input.
const legacyPayload = JSON.stringify({ wallet: { mnemonic: out.accounts[0].mnemonic } });
out.legacy = [{ password, payload: legacyPayload, serialized: CryptoJS.AES.encrypt(legacyPayload, password).toString() }];
out.legacy[0].roundtrip = CryptoJS.AES.decrypt(out.legacy[0].serialized, password).toString(CryptoJS.enc.Utf8) === legacyPayload;

writeFileSync(resolve(here, "vectors.json"), `${JSON.stringify(out, null, 2)}\n`);
console.log(`vault vectors: ${out.accounts.length} accounts, ${out.seal.length} sealed vaults, ${out.errors.length} error cases, ${out.legacy.length} legacy`);
