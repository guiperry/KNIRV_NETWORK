import { describe, it, expect } from "vitest";
import { RegistryStore, registerNode } from "../index.js";
import { buildTarget, healthURL, parseGatewayPort, validateNode, validationSettings } from "../validation.js";

function createStore() {
  const data = new Map();
  const storage = {
    async get(key) { return data.get(key) ?? null; },
    async put(key, value) { data.set(key, value); },
    async delete(keys) { for (const key of Array.isArray(keys) ? keys : [keys]) data.delete(key); },
    async list({ prefix }) { return new Map([...data].filter(([key]) => prefix === undefined || key.startsWith(prefix))); },
    async setAlarm() {},
  };
  const store = new RegistryStore({ storage });
  store.ctx = { storage };
  return { store, storage };
}

const IP = "8.8.4.4";
const settings = validationSettings({});

function healthyBody() {
  return { status: "healthy", chainId: "chain-1", dht: { peer_id: "peer-1", self_addrs: [`/ip4/${IP}/tcp/4001`] } };
}

// Records every health URL probed so tests can assert which port was used.
function recordingProbes() {
  const urls = [];
  return {
    urls,
    async tcp() { return { ok: true, ms: 10 }; },
    async https(url) { urls.push(url); return { reached: true, status: 200, body: healthyBody(), ms: 10 }; },
  };
}

describe("parseGatewayPort", () => {
  it("treats absent values as unset", () => {
    for (const raw of [undefined, null, ""]) expect(parseGatewayPort(raw)).toEqual({ port: 0 });
  });

  it("accepts integers 1-65535, as numbers or numeric strings", () => {
    expect(parseGatewayPort(1)).toEqual({ port: 1 });
    expect(parseGatewayPort(8443)).toEqual({ port: 8443 });
    expect(parseGatewayPort("65535")).toEqual({ port: 65535 });
  });

  it("rejects 0, out-of-range, non-integer and non-numeric values", () => {
    for (const raw of [0, -1, 65536, 1.5, "443x", "abc", true, {}, []]) {
      expect(parseGatewayPort(raw).error).toMatch(/gatewayPort/);
    }
  });
});

describe("health URL selection", () => {
  const base = { chainID: "chain-1", ip: IP, port: 4001, fqdn: "gw-1.knirv.com", peerID: "peer-1" };

  it("probes the advertised gateway port when present", () => {
    const { target } = buildTarget({ ...base, gatewayPort: 8443 }, settings);
    expect(target.gatewayPort).toBe(8443);
    expect(healthURL(target)).toBe("https://gw-1.knirv.com:8443/health");
  });

  it("falls back to the default 443 URL when absent", () => {
    const { target } = buildTarget({ ...base }, settings);
    expect(target.gatewayPort).toBe(0);
    expect(healthURL(target)).toBe("https://gw-1.knirv.com/health");
  });

  it("keeps the P2P port out of the health URL", async () => {
    const { target } = buildTarget({ ...base, gatewayPort: 8443 }, settings);
    const probes = recordingProbes();
    await validateNode(target, settings, probes);
    expect(probes.urls.every((u) => u === "https://gw-1.knirv.com:8443/health")).toBe(true);
  });

  it("buildTarget rejects an invalid gatewayPort", () => {
    expect(buildTarget({ ...base, gatewayPort: 70000 }, settings).error).toMatch(/gatewayPort/);
  });
});

describe("registration with gatewayPort", () => {
  const payload = { chainID: "chain-1", ip: IP, port: 4001, type: "bootnode", peerID: "peer-1", fqdn: "gw-1.knirv.com" };

  it("probes https://<fqdn>:<gatewayPort>/health and stores the port", async () => {
    const { store, storage } = createStore();
    const probes = recordingProbes();
    const res = await registerNode(store, { ...payload, gatewayPort: 8443 }, IP, 60_000, { BOOTNODE_VALIDATION: "enforce" }, probes);
    expect(res.status).toBe(201);
    expect(probes.urls[0]).toBe("https://gw-1.knirv.com:8443/health");
    const stored = await storage.get("node:chain-1");
    expect(stored.gatewayPort).toBe(8443);
    expect(stored.port).toBe(4001);
  });

  it("probes 443 and stores no gatewayPort when absent", async () => {
    const { store, storage } = createStore();
    const probes = recordingProbes();
    const res = await registerNode(store, payload, IP, 60_000, { BOOTNODE_VALIDATION: "enforce" }, probes);
    expect(res.status).toBe(201);
    expect(probes.urls[0]).toBe("https://gw-1.knirv.com/health");
    expect((await storage.get("node:chain-1")).gatewayPort).toBeUndefined();
  });

  it("rejects an invalid gatewayPort with 400 and stores nothing", async () => {
    const { store, storage } = createStore();
    const probes = recordingProbes();
    for (const gatewayPort of [0, 65536, "nope"]) {
      const res = await registerNode(store, { ...payload, gatewayPort }, IP, 60_000, { BOOTNODE_VALIDATION: "enforce" }, probes);
      expect(res.status).toBe(400);
      expect(res.error).toMatch(/gatewayPort/);
    }
    expect(probes.urls).toHaveLength(0);
    expect(await storage.get("node:chain-1")).toBeNull();
  });
});
