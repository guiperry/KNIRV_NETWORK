// @knirv/sdk/vault against the golden vectors captured from the controller's
// original JavaScript vault worker (testvectors/vault/vectors.json).
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { Vault, VaultError } from "@knirv/sdk/vault";

const v = JSON.parse(await readFile(new URL("../../testvectors/vault/vectors.json", import.meta.url), "utf8"));
const unhex = (h) => Uint8Array.from(h.match(/../g).map((x) => parseInt(x, 16)));

/** A host that replays chosen salt/nonce/entropy and ids. */
function host({ salt, nonce, entropy, createdAt = "2026-10-31T00:00:00.000Z" } = {}) {
  let n = 0;
  return {
    randomBytes: (len) => (len === 16 && entropy && !host.used ? ((host.used = true), entropy) : len === 16 ? salt : nonce),
    nowIso: () => createdAt,
    sessionId: () => `s${n++}`,
  };
}

const sdkTx = {
  action: { action: "transfer", sender: "knirv1sender", recipient: "knirv1recipient", amount: 5n, payload: Uint8Array.from([1, 2, 3]), timestampUnix: 1700000000n },
  chainId: "knirv-test-1", accountNumber: 7n, sequence: 3n, fee: { denom: "nrn", amount: "10", gasLimit: 200000n },
};
const sdkMsg = { domain: "knirv.vault.vector", purpose: "golden", chainId: "knirv-test-1", nonce: "n-1", issuedAtUnix: 1700000000n, expiresAtUnix: 1700003600n, payload: Uint8Array.from([9, 8, 7]) };

const vault = await Vault.load();

// derivation, addresses, signatures
for (const account of v.accounts) {
  const imported = vault.importMnemonic(account.mnemonic, "pw");
  assert.equal(imported.account.address, account.derived[0].address, account.name);
  for (const d of account.derived) {
    const got = vault.account(imported.sessionId, d.index);
    assert.deepEqual(got, { address: d.address, public_key: d.public_key }, `${account.name} #${d.index}`);
    assert.equal(vault.addressFromPublicKey(d.public_key), d.address);
    if (d.sign_tx) {
      assert.deepEqual(vault.signTransaction(imported.sessionId, sdkTx, d.index), d.sign_tx);
      assert.deepEqual(vault.signMessage(imported.sessionId, sdkMsg, d.index), d.sign_msg);
    }
  }
  vault.closeSession(imported.sessionId);
}

// sealing reproduces the JS envelope exactly, and JS envelopes open
for (const s of v.seal) {
  const account = v.accounts.find((a) => a.name === s.account);
  const h = host({ salt: unhex(s.salt), nonce: unhex(s.nonce), createdAt: s.created_at });
  const sealed = (await Vault.load({ host: h })).importMnemonic(account.mnemonic, s.password);
  assert.equal(sealed.envelope, s.serialized, `seal ${s.account}`);
  const opened = vault.unlock(s.serialized, s.password);
  assert.equal(opened.account.address, account.derived[0].address);
  assert.equal(opened.migrated, false);
}

// create: entropy -> the JS mnemonic
{
  const a = v.accounts.find((x) => x.entropy);
  host.used = false;
  const created = (await Vault.load({ host: host({ entropy: unhex(a.entropy), salt: new Uint8Array(16), nonce: new Uint8Array(12) }) })).createVault("pw");
  assert.equal(created.recoveryPhrase, a.mnemonic);
}

// errors
assert.throws(() => vault.importMnemonic(v.errors[0].mnemonic, "pw"), (e) => e instanceof VaultError && e.message === v.errors[0].js_error);
assert.throws(() => vault.unlock(v.seal[0].serialized, "wrong"), /Incorrect password/);
assert.throws(() => vault.unlock("garbage", "pw"), /Invalid vault envelope/);
assert.throws(() => vault.account("nope"), /Session not found/);

// migrate returns a verified v2 envelope
{
  const m = vault.migrate(v.accounts[0].mnemonic, "pw");
  assert.equal(m.migrated, true);
  assert.equal(vault.unlock(m.envelope, "pw").account.address, v.accounts[0].derived[0].address);
}

// sessions close and zeroize drops them all
vault.zeroize();
assert.equal(vault.sessionCount(), 0);
{
  const s = vault.importMnemonic(v.accounts[0].mnemonic, "pw");
  assert.equal(vault.sessionCount(), 1);
  vault.closeSession(s.sessionId);
  assert.equal(vault.sessionCount(), 0);
  assert.throws(() => vault.account(s.sessionId));
}

// Secrets do not linger in linear memory after use: the password and mnemonic
// bytes are wiped by the module. (A string that was ever a JS string cannot be
// erased from JS; this checks the module's own copies.) The canary mnemonic is
// NOT the "abandon ... about" phrase, which the module's self-test embeds as a
// constant.
{
  const password = "lingering-password-canary-0123456789";
  const canary = v.accounts[1];
  const fresh = await Vault.load();
  const s = fresh.importMnemonic(canary.mnemonic, password);
  fresh.signMessage(s.sessionId, sdkMsg);
  const duringSession = Buffer.from(new Uint8Array(fresh.wasm.memory.buffer));
  assert.equal(duringSession.includes(Buffer.from(canary.seed.slice(0, 64), "hex")), true, "sanity: the seed is held while the session is open");
  fresh.closeSession(s.sessionId);
  const memory = Buffer.from(new Uint8Array(fresh.wasm.memory.buffer));
  assert.equal(memory.includes(Buffer.from(password)), false, "password remained in module memory");
  assert.equal(memory.includes(Buffer.from(canary.mnemonic.split(" ").slice(0, 4).join(" "))), false, "mnemonic remained in module memory");
  assert.equal(memory.includes(Buffer.from(canary.seed.slice(0, 64), "hex")), false, "seed remained in module memory after close");
}

// the digest of what was loaded is checked
await assert.rejects(Vault.fromBytes(new Uint8Array([0, 97, 115, 109, 1, 0, 0, 0])), /digest/);

console.log(`vault: ${v.accounts.length} accounts, ${v.seal.length} sealed vaults, errors, migrate, zeroize, memory canaries`);
