import { DurableObject } from "cloudflare:workers";

const JSON_HEADERS = { "content-type": "application/json; charset=utf-8" };
const DEFAULT_NODE_TTL_MS = 60 * 60 * 1000;
const PRUNE_INTERVAL_MS = 15 * 60 * 1000;

function json(value, status = 200, headers = {}) {
  return new Response(JSON.stringify(value), { status, headers: { ...JSON_HEADERS, ...headers } });
}

function positiveInt(value, fallback) {
  const parsed = Number.parseInt(value, 10);
  return Number.isInteger(parsed) && parsed > 0 ? parsed : fallback;
}

function sourceIP(request) {
  return request.headers.get("CF-Connecting-IP") || request.headers.get("X-Forwarded-For")?.split(",")[0]?.trim() || "";
}

function escapeHTML(value) {
  return String(value).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#039;");
}

function statusHTML(status) {
  const rows = Object.entries(status.nodes).map(([chainID, node]) => `<tr><td><code>${escapeHTML(chainID)}</code></td><td>${escapeHTML(node.type)}</td><td>${escapeHTML(node.ip)}</td><td>${node.port}</td><td>${node.active ? "Active" : "Inactive"}</td><td>${escapeHTML(new Date(node.lastSeen).toLocaleString())}</td></tr>`).join("");
  return `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>KNIRV Network Registry</title><style>body{font-family:system-ui,sans-serif;max-width:1100px;margin:2rem auto;padding:0 1rem;color:#172033}table{width:100%;border-collapse:collapse}th,td{padding:.65rem;border-bottom:1px solid #d8dee9;text-align:left}th{background:#f4f7fb}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(160px,1fr));gap:1rem}.card{background:#f4f7fb;border-radius:.5rem;padding:1rem}code{word-break:break-all}</style></head><body><h1>KNIRV Network Registry</h1><div class="grid"><div class="card">Status<br><strong>${escapeHTML(status.status)}</strong></div><div class="card">Registered nodes<br><strong>${status.registeredNodes}</strong></div><div class="card">Active nodes<br><strong>${status.activeNodes}</strong></div><div class="card">Version<br><strong>${escapeHTML(status.version)}</strong></div></div><h2>Registered Nodes</h2><table><thead><tr><th>Chain ID</th><th>Type</th><th>IP</th><th>Port</th><th>Status</th><th>Last seen</th></tr></thead><tbody>${rows || '<tr><td colspan="6">No nodes registered</td></tr>'}</tbody></table></body></html>`;
}

export class RegistryStore extends DurableObject {
  async register(payload, ipFallback, ttlMs) {
    const chainID = typeof payload?.chainID === "string" ? payload.chainID.trim() : "";
    const ip = typeof payload?.ip === "string" && payload.ip.trim() ? payload.ip.trim() : ipFallback;
    const port = typeof payload?.port === "string" || typeof payload?.port === "number" ? Number.parseInt(payload.port, 10) : NaN;
    const type = payload?.type === "bootnode" ? "bootnode" : "client";
    const peerID = typeof payload?.peerID === "string" ? payload.peerID : undefined;
    if (!chainID || !Number.isInteger(port)) return { status: 400, error: "Missing required fields: chainID and port are mandatory." };
    if (port < 1 || port > 65535 || !ip) return { status: 400, error: "Invalid data types or value for chainID, port, or IP" };

    const now = Date.now();
    await this.prune(now, ttlMs);
    await this.ctx.storage.put("nodeTtlMs", ttlMs);
    const node = { ip, port, lastSeen: now, type, ...(peerID ? { peerID } : {}) };
    await this.ctx.storage.put(`node:${chainID}`, node);
    let suggestedBootnode;
    if (type === "client") {
      const bootnodes = await this.bootnodes();
      if (bootnodes.length) {
        const index = (await this.ctx.storage.get("roundRobinIndex")) || 0;
        suggestedBootnode = bootnodes[index % bootnodes.length];
        await this.ctx.storage.put("roundRobinIndex", (index + 1) % bootnodes.length);
      }
    }
    await this.ctx.storage.setAlarm(now + PRUNE_INTERVAL_MS);
    return { status: 201, message: type === "bootnode" ? "Bootnode registered successfully" : "Client node registered successfully", registeredIp: ip, details: { chainID, type, ...(peerID ? { peerID } : {}), port }, ...(suggestedBootnode ? { suggestedBootnode } : {}) };
  }

  async lookup(chainID) {
    const node = await this.ctx.storage.get(`node:${chainID}`);
    return node ? { status: 200, node } : { status: 404, error: "Node not found" };
  }

  async nodes() {
    const entries = await this.ctx.storage.list({ prefix: "node:" });
    return Object.fromEntries([...entries].map(([key, value]) => [key.slice(5), value]));
  }

  async bootnodes() {
    const nodes = await this.nodes();
    return Object.entries(nodes).filter(([, node]) => node.type === "bootnode").map(([chainID, node]) => ({ chainID, ...node }));
  }

  async status(ttlMs, version) {
    const now = Date.now();
    await this.prune(now, ttlMs);
    const nodes = await this.nodes();
    const withActivity = Object.fromEntries(Object.entries(nodes).map(([chainID, node]) => [chainID, { ...node, active: now - node.lastSeen < ttlMs }]));
    return { status: "online", version, registeredNodes: Object.keys(nodes).length, activeNodes: Object.values(withActivity).filter((node) => node.active).length, timestamp: new Date(now).toISOString(), nodes: withActivity };
  }

  async prune(now, ttlMs) {
    const entries = await this.ctx.storage.list({ prefix: "node:" });
    const expired = [...entries].filter(([, node]) => now - node.lastSeen > ttlMs).map(([key]) => key);
    if (expired.length) await this.ctx.storage.delete(expired);
  }

  async alarm() {
    const ttlMs = (await this.ctx.storage.get("nodeTtlMs")) || DEFAULT_NODE_TTL_MS;
    await this.prune(Date.now(), ttlMs);
    await this.ctx.storage.setAlarm(Date.now() + PRUNE_INTERVAL_MS);
  }
}

export default {
  async fetch(request, env) {
    const url = new URL(request.url);
    const path = url.pathname.replace(/\/$/, "") || "/";
    const ttlMs = positiveInt(env.NODE_TTL_SECONDS, DEFAULT_NODE_TTL_MS / 1000) * 1000;
    const store = env.REGISTRY.getByName("network-registry");
    if (request.method === "OPTIONS") return new Response(null, { headers: { "access-control-allow-origin": "*", "access-control-allow-methods": "GET, POST, OPTIONS", "access-control-allow-headers": "content-type" } });
    if (path === "/") return Response.redirect(new URL("/status", url), 308);
    if (path === "/register" && request.method === "GET") return json({ error: "Method Not Allowed. Use POST to /register for node registration.", hint: "See /status for registry information." }, 405, { allow: "POST" });
    if (path === "/register" && request.method === "POST") {
      let payload;
      try { payload = await request.json(); } catch { return json({ error: "Invalid JSON request body" }, 400); }
      const result = await store.register(payload, sourceIP(request), ttlMs);
      return result.error ? json({ error: result.error }, result.status) : json(result, result.status);
    }
    if (path.startsWith("/lookup/") && request.method === "GET") {
      const result = await store.lookup(decodeURIComponent(path.slice(8)));
      return result.error ? json({ error: result.error }, result.status) : json(result.node);
    }
    if (path === "/nodes" && request.method === "GET") return json(await store.nodes());
    if (path === "/bootnodes" && request.method === "GET") return json(await store.nodes());
    if (path === "/status" && request.method === "GET") {
      const status = await store.status(ttlMs, env.REGISTRY_VERSION || "2.0.0");
      return request.headers.get("accept")?.includes("text/html") ? new Response(statusHTML(status), { headers: { "content-type": "text/html; charset=utf-8" } }) : json(status);
    }
    if (path === "/stun" && request.method === "GET") {
      const host = env.STUN_HOST?.trim();
      const udpPort = positiveInt(env.STUN_PORT, 3478);
      const tcpPort = positiveInt(env.TURN_TCP_PORT, 3479);
      return json({ protocols: { udp: { enabled: Boolean(host), port: udpPort, address: host || "not configured" }, tcp: { enabled: Boolean(host), port: tcpPort, address: host || "not configured" } }, connectionStrings: host ? { udp: `stun:${host}:${udpPort}`, tcp: `stun:${host}:${tcpPort}?transport=tcp`, turn_udp: `turn:${host}:${udpPort}?transport=udp`, turn_tcp: `turn:${host}:${tcpPort}?transport=tcp` } : {}, serverSoftware: "external TURN/STUN required", timestamp: new Date().toISOString() });
    }
    return json({ error: "Not found" }, 404);
  },
};
