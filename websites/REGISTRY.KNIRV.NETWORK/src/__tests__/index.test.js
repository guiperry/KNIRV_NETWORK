import { describe, it, expect, beforeEach } from "vitest";
import { sha256 } from "@noble/hashes/sha2.js";
import { hmac } from "@noble/hashes/hmac.js";
import * as secp256k1 from "@noble/secp256k1";
import { RegistryStore } from "../index.js";

secp256k1.hashes.sha256 = sha256;
secp256k1.hashes.hmacSha256 = (key, msg) => hmac(sha256, key, msg);

function createMockStorage(initialData = {}) {
  const data = new Map(Object.entries(initialData));
  const listPrefixes = [];

  return {
    async get(key) {
      return data.get(key) ?? null;
    },
    async put(key, value) {
      data.set(key, value);
    },
    async delete(keys) {
      for (const key of Array.isArray(keys) ? keys : [keys]) {
        data.delete(key);
      }
    },
    async list({ prefix }) {
      listPrefixes.push(prefix);
      const entries = [];
      for (const [key, value] of data) {
        if (prefix === undefined || key.startsWith(prefix)) {
          entries.push([key, value]);
        }
      }
      return new Map(entries);
    },
    async setAlarm(timestamp) {
      return;
    },
    _data: data,
    _listPrefixes: listPrefixes,
  };
}

function createStore(storageData = {}) {
  const storage = createMockStorage(storageData);
  const store = new RegistryStore({ storage });
  store.ctx = { storage };
  return { store, storage };
}

describe("RegistryStore", () => {
  describe("register", () => {
    it("registers a bootnode and returns 201", async () => {
      const { store, storage } = createStore();
      const result = await store.register({ chainID: "chain-1", ip: "1.2.3.4", port: 8080, type: "bootnode" }, "1.2.3.4", 3600000);
      expect(result.status).toBe(201);
      expect(result.details.chainID).toBe("chain-1");
      expect(result.details.type).toBe("bootnode");
      expect(await storage.get("node:chain-1")).toEqual({ ip: "1.2.3.4", port: 8080, lastSeen: expect.any(Number), type: "bootnode" });
    });

    it("registers a client and suggests a bootnode", async () => {
      const { store, storage } = createStore();
      await storage.put("node:boot-1", { ip: "1.2.3.4", port: 8080, lastSeen: Date.now(), type: "bootnode" });
      const result = await store.register({ chainID: "client-1", ip: "5.6.7.8", port: 9090, type: "client" }, "5.6.7.8", 3600000);
      expect(result.status).toBe(201);
      expect(result.suggestedBootnode).toBeDefined();
    });

    it("rejects missing chainID", async () => {
      const { store } = createStore();
      const result = await store.register({}, "1.2.3.4", 3600000);
      expect(result.status).toBe(400);
    });

    it("rejects invalid port", async () => {
      const { store } = createStore();
      const result = await store.register({ chainID: "chain-1", ip: "1.2.3.4", port: "abc" }, "1.2.3.4", 3600000);
      expect(result.status).toBe(400);
    });
  });

  describe("bootnodes", () => {
    it("returns only bootnodes", async () => {
      const { store, storage } = createStore();
      await storage.put("node:boot-1", { ip: "1.2.3.4", port: 8080, lastSeen: Date.now(), type: "bootnode" });
      await storage.put("node:client-1", { ip: "5.6.7.8", port: 9090, lastSeen: Date.now(), type: "client" });
      const result = await store.bootnodes();
      expect(result).toHaveLength(1);
      expect(result[0].chainID).toBe("boot-1");
      expect(result[0].type).toBe("bootnode");
    });
  });

  describe("lookup", () => {
    it("returns node when found", async () => {
      const { store, storage } = createStore();
      await storage.put("node:chain-1", { ip: "1.2.3.4", port: 8080, lastSeen: Date.now(), type: "bootnode" });
      const result = await store.lookup("chain-1");
      expect(result.status).toBe(200);
      expect(result.node.ip).toBe("1.2.3.4");
    });

    it("returns 404 when not found", async () => {
      const { store } = createStore();
      const result = await store.lookup("missing");
      expect(result.status).toBe(404);
    });
  });

  describe("prune", () => {
    it("removes expired nodes", async () => {
      const { store, storage } = createStore();
      const now = Date.now();
      await storage.put("node:old", { ip: "1.2.3.4", port: 8080, lastSeen: now - 7200000, type: "bootnode" });
      await storage.put("node:new", { ip: "5.6.7.8", port: 9090, lastSeen: now, type: "bootnode" });
      await store.prune(now, 3600000);
      expect(await storage.get("node:old")).toBeNull();
      expect(await storage.get("node:new")).not.toBeNull();
    });
  });

  describe("heartbeat", () => {
    it("rejects when validator set not initialized", async () => {
      const { store } = createStore();
      const result = await store.heartbeat({ chainID: "chain-1", validatorID: "val-1", rootObservedHealthy: true, rootLastSeenMs: 1000, syncHeight: 10, signature: "abcd" }, {});
      expect(result.status).toBe(403);
    });

    it("rejects unenrolled validator", async () => {
      const { store, storage } = createStore();
      await storage.put("validators:chain-1", { chainID: "chain-1", version: 1, validators: [{ validatorID: "val-2", publicKey: "02" + "ab".repeat(32), active: true }] });
      const result = await store.heartbeat({ chainID: "chain-1", validatorID: "val-1", rootObservedHealthy: true, rootLastSeenMs: 1000, syncHeight: 10, signature: "abcd" }, {});
      expect(result.status).toBe(403);
    });

    it("rejects jailed validator", async () => {
      const { store, storage } = createStore();
      await storage.put("validators:chain-1", { chainID: "chain-1", version: 1, validators: [{ validatorID: "val-1", publicKey: "02" + "ab".repeat(32), active: false }] });
      const result = await store.heartbeat({ chainID: "chain-1", validatorID: "val-1", rootObservedHealthy: true, rootLastSeenMs: 1000, syncHeight: 10, signature: "abcd" }, {});
      expect(result.status).toBe(403);
    });

    it("records heartbeat for valid validator with valid signature", async () => {
      const { store, storage } = createStore();
      const privateKey = secp256k1.utils.randomSecretKey();
      const publicKey = secp256k1.getPublicKey(privateKey);
      const pubKeyHex = Buffer.from(publicKey).toString("hex");
      await storage.put("validators:chain-1", { chainID: "chain-1", version: 1, validators: [{ validatorID: "val-1", publicKey: pubKeyHex, active: true }] });
      const message = `chain-1|val-1|true|1000|10`;
      const signature = Buffer.from(secp256k1.sign(new TextEncoder().encode(message), privateKey)).toString("hex");
      const result = await store.heartbeat({ chainID: "chain-1", validatorID: "val-1", rootObservedHealthy: true, rootLastSeenMs: 1000, syncHeight: 10, signature }, { HEARTBEAT_TTL_SECONDS: "45" });
      expect(result.status).toBe(200);
    });
  });

  describe("vote", () => {
    it("rejects when validator set not initialized", async () => {
      const { store } = createStore();
      const result = await store.vote({ chainID: "chain-1", round: 1, validatorID: "val-1", candidateID: "cand-1", signature: "abcd" }, {});
      expect(result.status).toBe(403);
    });

    it("records vote for valid validator with valid signature", async () => {
      const { store, storage } = createStore();
      const privateKey = secp256k1.utils.randomSecretKey();
      const publicKey = secp256k1.getPublicKey(privateKey);
      const pubKeyHex = Buffer.from(publicKey).toString("hex");
      await storage.put("validators:chain-1", { chainID: "chain-1", version: 1, validators: [{ validatorID: "val-1", publicKey: pubKeyHex, active: true }] });
      const message = `chain-1|1|cand-1`;
      const signature = Buffer.from(secp256k1.sign(new TextEncoder().encode(message), privateKey)).toString('hex');
      const result = await store.vote({ chainID: "chain-1", round: 1, validatorID: "val-1", candidateID: "cand-1", signature }, {});
      expect(result.status).toBe(200);
    });
  });

  describe("state", () => {
    it("rejects when validator set not initialized", async () => {
      const { store } = createStore();
      const result = await store.state({ chainID: "chain-1", phase: "VOTING", currentRootID: "val-1", since: Date.now(), signature: "abcd" }, {});
      expect(result.status).toBe(403);
    });

    it("rejects invalid phase", async () => {
      const { store, storage } = createStore();
      const privateKey = secp256k1.utils.randomSecretKey();
      const publicKey = secp256k1.getPublicKey(privateKey);
      const pubKeyHex = Buffer.from(publicKey).toString("hex");
      await storage.put("validators:chain-1", { chainID: "chain-1", version: 1, validators: [{ validatorID: "val-1", publicKey: pubKeyHex, active: true }] });
      const since = Date.now();
      const message = `chain-1|INVALID|val-1|${since}`;
      const signature = Buffer.from(secp256k1.sign(new TextEncoder().encode(message), privateKey)).toString("hex");
      const result = await store.state({ chainID: "chain-1", phase: "INVALID", currentRootID: "val-1", since, signature }, {});
      expect(result.status).toBe(400);
    });

    it("records state and rotates publisher on CONFIRMED", async () => {
      const { store, storage } = createStore();
      const privateKey = secp256k1.utils.randomSecretKey();
      const publicKey = secp256k1.getPublicKey(privateKey);
      const pubKeyHex = Buffer.from(publicKey).toString("hex");
      await storage.put("validators:chain-1", { chainID: "chain-1", version: 1, validators: [{ validatorID: "val-1", publicKey: pubKeyHex, active: true }] });
      const since = Date.now();
      const message = `chain-1|CONFIRMED|val-1|${since}`;
      const signature = Buffer.from(secp256k1.sign(new TextEncoder().encode(message), privateKey)).toString("hex");
      const result = await store.state({ chainID: "chain-1", phase: "CONFIRMED", currentRootID: "val-1", since, signature }, {});
      expect(result.status).toBe(200);
      const publisher = await storage.get("authorizedPublisher:chain-1");
      expect(publisher).toBe("val-1");
    });
  });

  describe("validators", () => {
    it("rejects when no ROOT_PUBLIC_KEY and no existing publisher", async () => {
      const { store } = createStore();
      const result = await store.validators({ chainID: "chain-1", version: 1, validators: [{ validatorID: "val-1", publicKey: "02" + "ab".repeat(32), active: true }], signature: "abcd" }, {});
      expect(result.status).toBe(500);
    });

    it("accepts validators signed by ROOT_PUBLIC_KEY at genesis", async () => {
      const { store, storage } = createStore();
      const rootPrivateKey = secp256k1.utils.randomSecretKey();
      const rootPublicKey = secp256k1.getPublicKey(rootPrivateKey);
      const rootPubHex = Buffer.from(rootPublicKey).toString("hex");
      const validators = [{ validatorID: "val-1", publicKey: rootPubHex, active: true }];
      const message = `chain-1|1|${JSON.stringify(validators)}`;
      const signature = Buffer.from(secp256k1.sign(new TextEncoder().encode(message), rootPrivateKey)).toString("hex");
      const result = await store.validators({ chainID: "chain-1", version: 1, validators, signature }, { ROOT_PUBLIC_KEY: rootPubHex });
      expect(result.status).toBe(200);
      expect(result.version).toBe(1);
    });

    it("rejects stale version", async () => {
      const { store, storage } = createStore();
      const rootPrivateKey = secp256k1.utils.randomSecretKey();
      const rootPublicKey = secp256k1.getPublicKey(rootPrivateKey);
      const rootPubHex = Buffer.from(rootPublicKey).toString("hex");
      await storage.put("validators:chain-1", { chainID: "chain-1", version: 2, validators: [] });
      const validators = [{ validatorID: "val-1", publicKey: rootPubHex, active: true }];
      const message = `chain-1|1|${JSON.stringify(validators)}`;
      const signature = Buffer.from(secp256k1.sign(new TextEncoder().encode(message), rootPrivateKey)).toString("hex");
      const result = await store.validators({ chainID: "chain-1", version: 1, validators, signature }, { ROOT_PUBLIC_KEY: rootPubHex });
      expect(result.status).toBe(409);
    });
  });

  describe("heartbeats", () => {
    it("returns heartbeats for a chain", async () => {
      const { store, storage } = createStore();
      const now = Date.now();
      await storage.put("heartbeat:chain-1:val-1", { validatorID: "val-1", rootObservedHealthy: true, rootLastSeenMs: 1000, syncHeight: 10, lastSeen: now, timestamp: now });
      const result = await store.heartbeats("chain-1");
      expect(result["val-1"].validatorID).toBe("val-1");
    });
  });

  describe("votes", () => {
    it("returns votes for a round", async () => {
      const { store, storage } = createStore();
      const now = Date.now();
      await storage.put("vote:chain-1:1:val-1", { round: 1, validatorID: "val-1", candidateID: "cand-1", timestamp: now });
      const result = await store.votes("chain-1", 1);
      expect(result["val-1"].candidateID).toBe("cand-1");
    });
  });
});
