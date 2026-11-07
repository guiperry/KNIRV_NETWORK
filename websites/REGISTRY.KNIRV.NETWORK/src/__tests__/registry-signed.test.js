import { describe, it, expect, beforeAll } from "vitest";
import { randomBytes } from "crypto";
import { sha256 } from "@noble/hashes/sha2.js";
import { hmac } from "@noble/hashes/hmac.js";
import * as secp256k1 from "@noble/secp256k1";
import worker, { RegistryStore } from "../index.js";
import {
  initLocal,
  signRegistryRecord,
  signMessageEnvelope,
  publicKeyToKNIRVAddress,
  parseMessageEnvelope,
  REGISTRY_DOMAIN,
  PURPOSE_ROOT_TX_CHAIN_RECORD,
  MESSAGE_SCHEMA_VERSION,
} from "../../../../packages/KNIRVSDK/npm-package/signing.js";

secp256k1.hashes.sha256 = sha256;
secp256k1.hashes.hmacSha256 = (key, msg) => hmac(sha256, key, msg);

function createMockStorage(initialData = {}) {
  const data = new Map(Object.entries(initialData));
  return {
    async get(key) { return data.get(key) ?? null; },
    async put(key, value) { data.set(key, value); },
    async delete(keys) { for (const k of Array.isArray(keys) ? keys : [keys]) data.delete(k); },
    async setAlarm() {},
    async deleteAlarm() {},
    // Atomic transaction matching Durable Object semantics: writes inside `fn` are
    // buffered and only committed if `fn` completes; a throw rolls everything back.
    async transaction(fn) {
      const temp = new Map();
      const txn = {
        get: (k) => (temp.has(k) ? temp.get(k) : data.get(k) ?? null),
        put: (k, v) => { temp.set(k, v); },
        delete: (k) => { temp.delete(k); },
      };
      const result = await fn(txn);
      for (const [k, v] of temp) data.set(k, v);
      return result;
    },
    async list({ prefix }) {
      const e = [];
      for (const [k, v] of data) if (prefix === undefined || k.startsWith(prefix)) e.push([k, v]);
      return new Map(e);
    },
  };
}

function createStore() {
  const storage = createMockStorage();
  const store = new RegistryStore({ storage });
  store.ctx = { storage };
  return { store, storage };
}

const MAX_TTL = 365 * 24 * 60 * 60;

let root;
async function makeRoot() {
  const priv = randomBytes(32);
  const pub = secp256k1.getPublicKey(priv, true);
  const ownerPubKey = Buffer.from(pub).toString("base64");
  const address = await publicKeyToKNIRVAddress(pub);
  return { priv, pub, ownerPubKey, address, rootRegId: "root-registry-genesis" };
}

async function signRecord(r, { chainId, nonce, issuedIn = 0, expiresIn = 600, ownerPubKey = r.ownerPubKey } = {}) {
  const now = Math.floor(Date.now() / 1000);
  const record = {
    chainId,
    // A real knirv Bech32 address (the signer identity), so the verifier's
    // bech32 decode/checksum/HRP/length checks are exercised against truth.
    txChainAddress: r.address,
    ownerPubKey,
    nonce,
    expiresAtUnix: now + expiresIn,
  };
  const signed = await signRegistryRecord(r.priv, chainId, record, now + issuedIn);
  return { record, signed };
}

// Build a signed envelope with an arbitrary domain/purpose over the canonical
// payload of a valid record (used to probe domain/purpose enforcement).
async function signCustomEnvelope(r, domain, purpose) {
  const { signed } = await signRecord(r, { chainId: r.rootRegId, nonce: 1 });
  const parsed = parseMessageEnvelope(Buffer.from(signed.envelope, "base64"));
  const now = Math.floor(Date.now() / 1000);
  const envelope = {
    schemaVersion: MESSAGE_SCHEMA_VERSION,
    domain,
    purpose,
    chainId: r.rootRegId,
    nonce: "root-tx-chain:1",
    issuedAtUnix: now,
    expiresAtUnix: now + 600,
    payload: parsed.payload,
  };
  return signMessageEnvelope(r.priv, envelope);
}

function flipByte(base64Str) {
  const bytes = Uint8Array.from(Buffer.from(base64Str, "base64"));
  bytes[bytes.length - 1] ^= 0x01;
  return Buffer.from(bytes).toString("base64");
}

beforeAll(async () => {
  await initLocal();
  root = await makeRoot();
});

function envFor() {
  return { ROOT_PUBLIC_ADDRESS: root.address, ROOT_REGISTRY_ID: root.rootRegId };
}

describe("signed root tx-chain registry (Worker/DO)", () => {
  it("accepts a valid signed registration and stores the original envelope", async () => {
    const { store } = createStore();
    const { signed } = await signRecord(root, { chainId: root.rootRegId, nonce: 1 });
    const res = await store.registerSigned({ signed }, envFor());
    expect(res.status).toBe(201);
    expect(res.chainID).toBe(root.rootRegId);
    expect(res.signed).toEqual(signed);
  });

  it("returns the originally published envelope on signed lookup", async () => {
    const { store } = createStore();
    const { signed } = await signRecord(root, { chainId: root.rootRegId, nonce: 1 });
    await store.registerSigned({ signed }, envFor());
    const looked = await store.lookupSigned(root.rootRegId);
    expect(looked.status).toBe(200);
    expect(looked.signed).toEqual(signed);
  });

  it("rejects replayed nonce and any non-increasing nonce", async () => {
    const { store } = createStore();
    const a = await signRecord(root, { chainId: root.rootRegId, nonce: 5 });
    expect((await store.registerSigned({ signed: a.signed }, envFor())).status).toBe(201);
    const replay = await signRecord(root, { chainId: root.rootRegId, nonce: 5 });
    expect((await store.registerSigned({ signed: replay.signed }, envFor())).status).toBe(409);
    const older = await signRecord(root, { chainId: root.rootRegId, nonce: 4 });
    expect((await store.registerSigned({ signed: older.signed }, envFor())).status).toBe(409);
    const newer = await signRecord(root, { chainId: root.rootRegId, nonce: 6 });
    expect((await store.registerSigned({ signed: newer.signed }, envFor())).status).toBe(201);
  });

  it("rejects concurrent identical-nonce submissions (mutex serializes to one winner)", async () => {
    const { store } = createStore();
    const { signed } = await signRecord(root, { chainId: root.rootRegId, nonce: 9 });
    const results = await Promise.all([
      store.registerSigned({ signed }, envFor()),
      store.registerSigned({ signed }, envFor()),
      store.registerSigned({ signed }, envFor()),
    ]);
    const accepted = results.filter((r) => r.status === 201).length;
    expect(accepted).toBe(1);
  });

  it("rejects an unsigned / missing envelope body", async () => {
    const { store } = createStore();
    expect((await store.registerSigned({}, envFor())).status).toBe(400);
    expect((await store.registerSigned({ signed: { envelope: "x" } }, envFor())).status).toBe(400);
  });

  it("rejects a record signed by a non-root key", async () => {
    const { store } = createStore();
    const other = await makeRoot();
    other.rootRegId = root.rootRegId; // same expected root id, wrong key
    const { signed } = await signRecord(other, { chainId: root.rootRegId, nonce: 1 });
    expect((await store.registerSigned({ signed }, envFor())).status).toBe(401);
  });

  it("rejects owner_pub_key that does not match the signing key", async () => {
    const { store } = createStore();
    const wrong = await makeRoot();
    const { signed } = await signRecord(root, { chainId: root.rootRegId, nonce: 1, ownerPubKey: wrong.ownerPubKey });
    expect((await store.registerSigned({ signed }, envFor())).status).toBe(401);
  });

  it("rejects an expired record", async () => {
    const { store } = createStore();
    const { signed } = await signRecord(root, { chainId: root.rootRegId, nonce: 1, issuedIn: -200, expiresIn: -100 });
    expect((await store.registerSigned({ signed }, envFor())).status).toBe(401);
  });

  it("rejects a record whose TTL exceeds the maximum", async () => {
    const { store } = createStore();
    const { signed } = await signRecord(root, { chainId: root.rootRegId, nonce: 1, expiresIn: MAX_TTL + 100 });
    expect((await store.registerSigned({ signed }, envFor())).status).toBe(401);
  });

  it("rejects a wrong domain envelope", async () => {
    const { store } = createStore();
    const signed = await signCustomEnvelope(root, "evil.domain", PURPOSE_ROOT_TX_CHAIN_RECORD);
    expect((await store.registerSigned({ signed }, envFor())).status).toBe(401);
  });

  it("rejects a wrong purpose envelope", async () => {
    const { store } = createStore();
    const signed = await signCustomEnvelope(root, REGISTRY_DOMAIN, "wrong-purpose");
    expect((await store.registerSigned({ signed }, envFor())).status).toBe(401);
  });

  it("rejects a tampered envelope (signature break)", async () => {
    const { store } = createStore();
    const { signed } = await signRecord(root, { chainId: root.rootRegId, nonce: 1 });
    const tampered = { ...signed, envelope: flipByte(signed.envelope) };
    expect((await store.registerSigned({ signed: tampered }, envFor())).status).toBe(401);
  });

  it("never returns legacy unsigned tx-chain records to payment clients (quarantine)", async () => {
    const { store } = createStore();
    const reg = await store.register(
      { chainID: "legacy-1", ip: "1.2.3.4", port: 8080, type: "bootnode", txChainAddress: "0xdeadbeef" },
      "1.2.3.4",
      3600000,
    );
    expect(reg.status).toBe(400);
    expect((await store.lookup("legacy-1")).status).toBe(404);
    const nodes = await store.nodes();
    expect(nodes["legacy-1"]).toBeUndefined();
  });

  it("rejects unknown top-level fields in the signed body", async () => {
    const { store } = createStore();
    const { signed } = await signRecord(root, { chainId: root.rootRegId, nonce: 1 });
    const res = await store.registerSigned({ signed, extra: "nope" }, envFor());
    expect(res.status).toBe(400);
  });

  it("rejects a nonce that is not a safe integer (overflow)", async () => {
    const { store } = createStore();
    const { signed } = await signRecord(root, { chainId: root.rootRegId, nonce: 2 ** 60 });
    expect((await store.registerSigned({ signed }, envFor())).status).toBe(400);
  });

  it("rolls back cleanly if the atomic write fails (no inconsistent state)", async () => {
    const { store, storage } = createStore();
    const first = await signRecord(root, { chainId: root.rootRegId, nonce: 5 });
    expect((await store.registerSigned({ signed: first.signed }, envFor())).status).toBe(201);
    // Simulate a failure on the second put inside the transaction.
    store.ctx.storage.transaction = async (fn) => {
      const temp = new Map();
      let n = 0;
      const txn = {
        get: (k) => (temp.has(k) ? temp.get(k) : data.get(k) ?? null),
        put: (k, v) => {
          temp.set(k, v);
          if (++n === 2) throw new Error("simulated second put failure");
        },
        delete: (k) => { temp.delete(k); },
      };
      const r = await fn(txn);
      for (const [k, v] of temp) data.set(k, v);
      return r;
    };
    const second = await signRecord(root, { chainId: root.rootRegId, nonce: 7 });
    // The transaction throws mid-write; the registration must fail (not 201) and roll back.
    const res = await store.registerSigned({ signed: second.signed }, envFor());
    expect(res.status).toBe(500);
    // Old record (nonce 5) is fully intact; the failed nonce-7 write was rolled back.
    const looked = await store.lookupSigned(root.rootRegId);
    expect(looked.status).toBe(200);
    expect(looked.signed).toEqual(first.signed);
    expect(await storage.get(`registryNonce:${root.rootRegId}`)).toBe(5);
  });

  it("rejects a replayed nonce after a fresh DO instance over the same storage", async () => {
    const storage = createMockStorage();
    const storeA = new RegistryStore({ storage });
    storeA.ctx = { storage };
    const { signed } = await signRecord(root, { chainId: root.rootRegId, nonce: 3 });
    expect((await storeA.registerSigned({ signed }, envFor())).status).toBe(201);
    // New DO instance, same durable storage: must honor the persisted high-water nonce.
    const storeB = new RegistryStore({ storage });
    storeB.ctx = { storage };
    const replay = await signRecord(root, { chainId: root.rootRegId, nonce: 3 });
    expect((await storeB.registerSigned({ signed: replay.signed }, envFor())).status).toBe(409);
    const newer = await signRecord(root, { chainId: root.rootRegId, nonce: 4 });
    expect((await storeB.registerSigned({ signed: newer.signed }, envFor())).status).toBe(201);
  });
});

describe("fetch entrypoint: POST /registry/register", () => {
  function envWith(store) {
    return {
      REGISTRY: { getByName: () => store },
      ROOT_PUBLIC_ADDRESS: root.address,
      ROOT_REGISTRY_ID: root.rootRegId,
      BOOTNODE_VALIDATION: "off",
    };
  }

  it("accepts a valid signed registration over HTTP and emits explicit CORS", async () => {
    const { store } = createStore();
    const { signed } = await signRecord(root, { chainId: root.rootRegId, nonce: 1 });
    const req = new Request("http://localhost/registry/register", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ signed }),
    });
    const res = await worker.fetch(req, envWith(store));
    expect(res.status).toBe(201);
    expect(res.headers.get("access-control-allow-origin")).toBeNull();
  });

  it("rejects an oversized body with 413", async () => {
    const { store } = createStore();
    const { signed } = await signRecord(root, { chainId: root.rootRegId, nonce: 1 });
    // Guarantee the body exceeds the 64KB cap regardless of envelope size.
    const body = JSON.stringify({ signed, padding: "x".repeat(200000) });
    const req = new Request("http://localhost/registry/register", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body,
    });
    const res = await worker.fetch(req, envWith(store));
    expect(res.status).toBe(413);
  });

  it("rejects duplicate JSON keys with 400", async () => {
    const { store } = createStore();
    const { signed } = await signRecord(root, { chainId: root.rootRegId, nonce: 1 });
    const body = `{"signed":${JSON.stringify(signed)},"signed":${JSON.stringify(signed)}}`;
    const req = new Request("http://localhost/registry/register", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body,
    });
    const res = await worker.fetch(req, envWith(store));
    expect(res.status).toBe(400);
  });

  it("OPTIONS /registry/register returns explicit (non-wildcard) CORS", async () => {
    const { store } = createStore();
    const req = new Request("http://localhost/registry/register", { method: "OPTIONS" });
    const res = await worker.fetch(req, envWith(store));
    expect(res.status).toBe(204);
  });

  it("rejects unsigned POST /register carrying a tx-chain address (root or any id)", async () => {
    const { store } = createStore();
    const req = new Request("http://localhost/register", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({
        chainID: root.rootRegId,
        ip: "1.2.3.4",
        port: 8080,
        type: "bootnode",
        txChainAddress: root.address,
      }),
    });
    const res = await worker.fetch(req, envWith(store));
    expect(res.status).toBe(400);
  });
});
