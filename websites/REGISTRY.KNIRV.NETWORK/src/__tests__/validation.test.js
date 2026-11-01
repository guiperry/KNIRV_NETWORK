import { describe, it, expect } from "vitest";
import { RegistryStore, registerNode, validateRequest } from "../index.js";
import { isPublicIP, buildTarget, validateNode, validationSettings, tokensMatch } from "../validation.js";

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

function healthyBody(overrides = {}) {
  return { status: "healthy", chainId: "chain-1", dht: { peer_id: "peer-1", self_addrs: [`/ip4/${IP}/tcp/4001`] }, ...overrides };
}

function fakeProbes({ tcp = { ok: true, ms: 20 }, https = { reached: true, status: 200, body: healthyBody(), ms: 50 } } = {}) {
  const calls = { tcp: 0, https: 0 };
  return {
    calls,
    async tcp() { calls.tcp++; return tcp; },
    async https() { calls.https++; return https; },
  };
}

const bootPayload = { chainID: "chain-1", ip: IP, port: 4001, type: "bootnode", peerID: "peer-1", fqdn: "gw-1.knirv.com" };

describe("isPublicIP / buildTarget", () => {
  it("accepts public and rejects private, loopback and malformed addresses", () => {
    expect(isPublicIP("8.8.4.4")).toBe(true);
    for (const ip of ["10.0.0.1", "127.0.0.1", "192.168.1.5", "172.16.0.1", "169.254.1.1", "0.0.0.0", "999.1.1.1", "::1", "fd00::1", "localhost", ""]) {
      expect(isPublicIP(ip)).toBe(false);
    }
  });

  it("restricts fqdn to the allowed suffixes", () => {
    expect(buildTarget({ ip: IP, port: 1, fqdn: "gw.knirv.com" }, settings).target.fqdn).toBe("gw.knirv.com");
    expect(buildTarget({ ip: IP, port: 1, fqdn: "evil.example.com" }, settings).error).toMatch(/fqdn/);
    expect(buildTarget({ ip: IP, port: 1, fqdn: "evil.example.com" }, validationSettings({ VALIDATION_FQDN_SUFFIXES: "" })).target).toBeDefined();
  });
});

describe("validateNode", () => {
  const target = { chainID: "chain-1", ip: IP, port: 4001, fqdn: "gw-1.knirv.com", peerID: "peer-1" };

  it("passes when all four checks pass", async () => {
    const result = await validateNode(target, settings, fakeProbes());
    expect(result.passed).toBe(true);
    expect(result.checks).toEqual({ connectivity: true, health: true, security: true, performance: true });
  });

  it("fails connectivity when TCP cannot connect", async () => {
    const result = await validateNode(target, settings, fakeProbes({ tcp: { ok: false, ms: 5000, error: "timeout" } }));
    expect(result.passed).toBe(false);
    expect(result.checks.connectivity).toBe(false);
  });

  it("fails health and security without an fqdn", async () => {
    const result = await validateNode({ ...target, fqdn: "" }, settings, fakeProbes());
    expect(result.checks).toMatchObject({ connectivity: true, health: false, security: false, performance: false });
  });

  it("fails security when TLS cannot be established", async () => {
    const result = await validateNode(target, settings, fakeProbes({ https: { reached: false, ms: 10, error: "tls" } }));
    expect(result.checks.security).toBe(false);
    expect(result.passed).toBe(false);
  });

  it("fails health when the node reports a different chainId", async () => {
    const probes = fakeProbes({ https: { reached: true, status: 200, body: healthyBody({ chainId: "someone-else" }), ms: 10 } });
    const result = await validateNode(target, settings, probes);
    expect(result.checks.health).toBe(false);
    expect(result.details.health.error).toMatch(/chainId/);
  });

  it("fails health when self_addrs do not include the registered ip", async () => {
    const body = healthyBody({ dht: { peer_id: "peer-1", self_addrs: ["/ip4/9.9.9.9/tcp/4001"] } });
    const result = await validateNode(target, settings, fakeProbes({ https: { reached: true, status: 200, body, ms: 10 } }));
    expect(result.checks.health).toBe(false);
  });

  it("fails performance when median latency exceeds the limit", async () => {
    const result = await validateNode(target, settings, fakeProbes({ https: { reached: true, status: 200, body: healthyBody(), ms: 9000 } }));
    expect(result.checks.performance).toBe(false);
    expect(result.checks.health).toBe(true);
  });
});

describe("registerNode", () => {
  it("registers a validated bootnode and lists it", async () => {
    const { store } = createStore();
    const result = await registerNode(store, bootPayload, IP, 3600000, {}, fakeProbes());
    expect(result.status).toBe(201);
    expect(result.validation.passed).toBe(true);
    expect((await store.bootnodes(true)).map((n) => n.chainID)).toEqual(["chain-1"]);
    expect((await store.lookup("chain-1")).node.validated).toBe(true);
  });

  it("rejects a bootnode that fails validation and stores nothing", async () => {
    const { store } = createStore();
    const result = await registerNode(store, bootPayload, IP, 3600000, {}, fakeProbes({ tcp: { ok: false, ms: 1, error: "refused" } }));
    expect(result.status).toBe(422);
    expect(result.validation.checks.connectivity).toBe(false);
    expect((await store.lookup("chain-1")).status).toBe(404);
  });

  it("rejects a private ip without probing", async () => {
    const { store } = createStore();
    const probes = fakeProbes();
    const result = await registerNode(store, { ...bootPayload, ip: "10.0.0.5" }, "10.0.0.5", 3600000, {}, probes);
    expect(result.status).toBe(422);
    expect(probes.calls.tcp).toBe(0);
  });

  it("reuses a passing validation instead of re-probing on heartbeat", async () => {
    const { store } = createStore();
    const probes = fakeProbes();
    await registerNode(store, bootPayload, IP, 3600000, {}, probes);
    const second = await registerNode(store, bootPayload, IP, 3600000, {}, probes);
    expect(second.status).toBe(201);
    expect(probes.calls.tcp).toBe(1);
  });

  it("re-validates when the registered address changes", async () => {
    const { store } = createStore();
    const probes = fakeProbes({ https: { reached: true, status: 200, body: healthyBody({ dht: { peer_id: "peer-1", self_addrs: [] } }), ms: 10 } });
    await registerNode(store, bootPayload, IP, 3600000, {}, probes);
    await registerNode(store, { ...bootPayload, port: 4002 }, IP, 3600000, {}, probes);
    expect(probes.calls.tcp).toBe(2);
  });

  it("applies a cooldown after a failed validation", async () => {
    const { store } = createStore();
    const probes = fakeProbes({ tcp: { ok: false, ms: 1, error: "refused" } });
    await registerNode(store, bootPayload, IP, 3600000, {}, probes);
    const again = await registerNode(store, bootPayload, IP, 3600000, {}, probes);
    expect(again.status).toBe(429);
    expect(again.retryAfter).toBeGreaterThan(0);
    expect(probes.calls.tcp).toBe(1);
  });

  it("does not validate clients and never hides them", async () => {
    const { store } = createStore();
    const probes = fakeProbes();
    const result = await registerNode(store, { chainID: "client-1", port: 9090, type: "client" }, IP, 3600000, {}, probes);
    expect(result.status).toBe(201);
    expect(probes.calls.tcp).toBe(0);
    expect(Object.keys(await store.nodes(true))).toContain("client-1");
  });

  it("hides unvalidated legacy bootnodes in enforce mode only", async () => {
    const { store, storage } = createStore();
    await storage.put("node:legacy", { ip: IP, port: 1, lastSeen: Date.now(), type: "bootnode" });
    expect(Object.keys(await store.nodes(true))).not.toContain("legacy");
    expect(Object.keys(await store.nodes(false))).toContain("legacy");
  });

  it("suggests only validated bootnodes to clients in enforce mode", async () => {
    const { store, storage } = createStore();
    await storage.put("node:legacy", { ip: IP, port: 1, lastSeen: Date.now(), type: "bootnode" });
    const noBoot = await registerNode(store, { chainID: "c1", port: 9090, type: "client" }, IP, 3600000, {});
    expect(noBoot.suggestedBootnode).toBeUndefined();
    await registerNode(store, bootPayload, IP, 3600000, {}, fakeProbes());
    const withBoot = await registerNode(store, { chainID: "c2", port: 9090, type: "client" }, IP, 3600000, {});
    expect(withBoot.suggestedBootnode.chainID).toBe("chain-1");
  });

  it("monitor mode records the result but still registers", async () => {
    const { store } = createStore();
    const result = await registerNode(store, bootPayload, IP, 3600000, { BOOTNODE_VALIDATION: "monitor" }, fakeProbes({ tcp: { ok: false, ms: 1, error: "refused" } }));
    expect(result.status).toBe(201);
    expect(result.validation.passed).toBe(false);
    expect((await store.lookup("chain-1")).node.validated).toBeUndefined();
  });

  it("off mode skips validation entirely", async () => {
    const { store } = createStore();
    const probes = fakeProbes();
    const result = await registerNode(store, bootPayload, IP, 3600000, { BOOTNODE_VALIDATION: "off" }, probes);
    expect(result.status).toBe(201);
    expect(probes.calls.tcp).toBe(0);
  });

  it("stores nodeID (sent by KNIRVGATEWAY) as the peer id", async () => {
    const { store } = createStore();
    await registerNode(store, { chainID: "c", port: 1, type: "client", nodeID: "peer-x" }, IP, 3600000, {});
    expect((await store.lookup("c")).node.peerID).toBe("peer-x");
  });
});

describe("POST /validate", () => {
  const req = (body, token) => new Request("https://registry/validate", { method: "POST", headers: token ? { authorization: `Bearer ${token}` } : {}, body: JSON.stringify(body) });
  const body = { registrationId: "reg-1", fqdn: "gw-1.knirv.com", publicIps: [IP], expectedPorts: [4001] };

  it("works without a token when none is configured", async () => {
    const res = await validateRequest(req(body), {}, fakeProbes());
    expect(res.status).toBe(200);
    expect((await res.json()).checks.connectivity).toBe(true);
  });

  it("rejects a missing or wrong token when one is configured", async () => {
    expect((await validateRequest(req(body), { VALIDATION_SERVICE_TOKEN: "s3cret" }, fakeProbes())).status).toBe(401);
    expect((await validateRequest(req(body, "nope"), { VALIDATION_SERVICE_TOKEN: "s3cret" }, fakeProbes())).status).toBe(401);
  });

  it("returns the checks contract run-tests.ts expects", async () => {
    const res = await validateRequest(req(body, "s3cret"), { VALIDATION_SERVICE_TOKEN: "s3cret" }, fakeProbes());
    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json.checks).toEqual({ connectivity: true, health: true, security: true, performance: true });
  });

  it("reports failed checks for an unsafe target", async () => {
    const res = await validateRequest(req({ ...body, publicIps: ["10.0.0.1"] }, "s3cret"), { VALIDATION_SERVICE_TOKEN: "s3cret" }, fakeProbes());
    const json = await res.json();
    expect(Object.values(json.checks).every((v) => v === false)).toBe(true);
  });
});

describe("tokensMatch", () => {
  it("compares tokens exactly", () => {
    expect(tokensMatch("abc", "abc")).toBe(true);
    expect(tokensMatch("abd", "abc")).toBe(false);
    expect(tokensMatch("abc", "")).toBe(false);
  });
});
