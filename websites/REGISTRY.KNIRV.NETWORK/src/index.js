import { DurableObject } from "cloudflare:workers";
import { sha256 } from "@noble/hashes/sha2.js";
import { hmac } from "@noble/hashes/hmac.js";
import * as secp256k1 from "@noble/secp256k1";

secp256k1.hashes.sha256 = sha256;
secp256k1.hashes.hmacSha256 = (key, msg) => hmac(sha256, key, msg);

const JSON_HEADERS = { "content-type": "application/json; charset=utf-8" };
const DEFAULT_NODE_TTL_MS = 60 * 60 * 1000;
const PRUNE_INTERVAL_MS = 15 * 60 * 1000;
const HEARTBEAT_TTL_MS = 45 * 1000;
const HEARTBEAT_PRUNE_INTERVAL_MS = 30 * 1000;
const VOTE_TTL_MS = 24 * 60 * 60 * 1000;

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
  const validatorRows = status.validators && Object.entries(status.validators).map(([validatorID, v]) => `<tr><td><code>${escapeHTML(validatorID)}</code></td><td>${escapeHTML(v.publicKey || "")}</td><td>${v.active ? "Active" : "Jailed/Demoted"}</td></tr>`).join("");
  return `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>KNIRV Network Registry</title><style>body{font-family:system-ui,sans-serif;max-width:1100px;margin:2rem auto;padding:0 1rem;color:#172033}table{width:100%;border-collapse:collapse}th,td{padding:.65rem;border-bottom:1px solid #d8dee9;text-align:left}th{background:#f4f7fb}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(160px,1fr));gap:1rem}.card{background:#f4f7fb;border-radius:.5rem;padding:1rem}code{word-break:break-all}</style></head><body><h1>KNIRV Network Registry</h1><div class="grid"><div class="card">Status<br><strong>${escapeHTML(status.status)}</strong></div><div class="card">Registered nodes<br><strong>${status.registeredNodes}</strong></div><div class="card">Active nodes<br><strong>${status.activeNodes}</strong></div><div class="card">Version<br><strong>${escapeHTML(status.version)}</strong></div></div><h2>Registered Nodes</h2><table><thead><tr><th>Chain ID</th><th>Type</th><th>IP</th><th>Port</th><th>Status</th><th>Last seen</th></tr></thead><tbody>${rows || '<tr><td colspan="6">No nodes registered</td></tr>'}</tbody></table>${validatorRows ? '<h2>Validators</h2><table><thead><tr><th>Validator ID</th><th>Public Key</th><th>Status</th></tr></thead><tbody>' + validatorRows + '</tbody></table>' : ''}</</body></html>`;
}

function canonicalize(...parts) {
  return parts.join("|");
}

async function verifySignature(publicKeyHex, message, signatureHex) {
  try {
    const messageBytes = new TextEncoder().encode(message);
    const signatureBytes = new Uint8Array(signatureHex.match(/.{2}/g).map(b => parseInt(b, 16)));
    const publicKeyBytes = new Uint8Array(publicKeyHex.match(/.{2}/g).map(b => parseInt(b, 16)));
    return secp256k1.verify(signatureBytes, messageBytes, publicKeyBytes);
  } catch {
    return false;
  }
}

async function getAuthorizedPublisher(store, chainID) {
  const raw = await store.ctx.storage.get(`authorizedPublisher:${chainID}`);
  if (typeof raw === "string") return raw;
  return null;
}

async function setAuthorizedPublisher(store, chainID, validatorID) {
  await store.ctx.storage.put(`authorizedPublisher:${chainID}`, validatorID);
}

async function getValidators(store, chainID) {
  const raw = await store.ctx.storage.get(`validators:${chainID}`);
  if (!raw) return null;
  return raw;
}

async function pruneHeartbeats(store, now, ttlMs) {
  const entries = await store.ctx.storage.list({ prefix: "heartbeat:" });
  const expired = [];
  for (const [key, value] of entries) {
    if (now - value.lastSeen > ttlMs) {
      expired.push(key);
    }
  }
  if (expired.length > 0) {
    await store.ctx.storage.delete(expired);
  }
}

async function pruneVotes(store, now, ttlMs) {
  const entries = await store.ctx.storage.list({ prefix: "vote:" });
  const expired = [];
  for (const [key, value] of entries) {
    if (now - value.timestamp > ttlMs) {
      expired.push(key);
    }
  }
  if (expired.length > 0) {
    await store.ctx.storage.delete(expired);
  }
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
    const validators = await getValidators(this, "knirvoracle-1");
    const validatorMap = validators && validators.validators ? Object.fromEntries(validators.validators.map(v => [v.validatorID, v])) : {};
    return { status: "online", version, registeredNodes: Object.keys(nodes).length, activeNodes: Object.values(withActivity).filter((node) => node.active).length, timestamp: new Date(now).toISOString(), nodes: withActivity, validators: validatorMap };
  }

  async prune(now, ttlMs) {
    const entries = await this.ctx.storage.list({ prefix: "node:" });
    const expired = [...entries].filter(([, node]) => now - node.lastSeen > ttlMs).map(([key]) => key);
    if (expired.length) await this.ctx.storage.delete(expired);
  }

  async heartbeat(payload, env) {
    const chainID = typeof payload?.chainID === "string" ? payload.chainID.trim() : "";
    const validatorID = typeof payload?.validatorID === "string" ? payload.validatorID.trim() : "";
    const rootObservedHealthy = typeof payload?.rootObservedHealthy === "boolean" ? payload.rootObservedHealthy : false;
    const rootLastSeenMs = typeof payload?.rootLastSeenMs === "number" ? payload.rootLastSeenMs : 0;
    const syncHeight = typeof payload?.syncHeight === "number" ? payload.syncHeight : 0;
    const signature = typeof payload?.signature === "string" ? payload.signature.trim() : "";
    if (!chainID || !validatorID || !signature) return { status: 400, error: "Missing required fields: chainID, validatorID, and signature are mandatory." };

    const validators = await getValidators(this, chainID);
    if (!validators || !validators.validators) return { status: 403, error: "Validator set not initialized." };
    const validator = validators.validators.find(v => v.validatorID === validatorID);
    if (!validator) return { status: 403, error: "Validator not found in current set." };
    if (!validator.active) return { status: 403, error: "Validator is not active." };

    const message = canonicalize(chainID, validatorID, String(rootObservedHealthy), String(rootLastSeenMs), String(syncHeight));
    const isValid = await verifySignature(validator.publicKey, message, signature);
    if (!isValid) return { status: 403, error: "Invalid signature." };

    const ttlMs = positiveInt(env.HEARTBEAT_TTL_SECONDS, HEARTBEAT_TTL_MS / 1000) * 1000;
    const now = Date.now();
    await pruneHeartbeats(this, now, ttlMs);
    const heartbeat = { validatorID, rootObservedHealthy, rootLastSeenMs, syncHeight, lastSeen: now, timestamp: now };
    await this.ctx.storage.put(`heartbeat:${chainID}:${validatorID}`, heartbeat);
    await this.ctx.storage.setAlarm(now + HEARTBEAT_PRUNE_INTERVAL_MS);
    return { status: 200, message: "Heartbeat recorded." };
  }

  async heartbeats(chainID) {
    const entries = await this.ctx.storage.list({ prefix: `heartbeat:${chainID}:` });
    return Object.fromEntries([...entries].map(([key, value]) => [key.slice(`heartbeat:${chainID}:`.length), value]));
  }

  // consensus is read-only: it turns the already signature-verified
  // heartbeats and votes into a deterministic decision.  No Worker endpoint
  // can promote a node merely because it can reach the Durable Object.
  async consensus(chainID, round, env) {
    const validators = await getValidators(this, chainID);
    const active = (validators?.validators || []).filter(v => v.active);
    const now = Date.now();
    const ttlMs = positiveInt(env.HEARTBEAT_TTL_SECONDS, HEARTBEAT_TTL_MS / 1000) * 1000;
    const heartbeats = await this.heartbeats(chainID);
    const missing = [];
    const healthy = [];
    const unhealthy = [];
    for (const validator of active) {
      const heartbeat = heartbeats[validator.validatorID];
      if (!heartbeat || now - heartbeat.lastSeen > ttlMs) { missing.push(validator.validatorID); continue; }
      (heartbeat.rootObservedHealthy ? healthy : unhealthy).push(validator.validatorID);
    }
    const rootDown = active.length > 0 && missing.length === 0 && healthy.length === 0;
    const votes = Number.isInteger(round) && round > 0 ? await this.votes(chainID, round) : {};
    const count = new Map();
    for (const id of active.map(v => v.validatorID)) {
      const vote = votes[id];
      if (vote?.candidateID) count.set(vote.candidateID, (count.get(vote.candidateID) || 0) + 1);
    }
    const ranked = [...count.entries()].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]));
    // All active validators participate. A tie is intentionally unresolved so
    // the orchestrator can use its signed 50/25/25 score fallback and submit a
    // new round rather than selecting arbitrarily at the registry layer.
    const winner = ranked.length && ranked[0][1] === active.length && (ranked.length === 1 || ranked[0][1] > ranked[1][1]) ? ranked[0][0] : null;
    return { chainID, round: round || 0, activeValidators: active.map(v => v.validatorID), rootDown, missing, healthy, unhealthy, votes: Object.keys(votes).length, winner };
  }

  async vote(payload, env) {
    const chainID = typeof payload?.chainID === "string" ? payload.chainID.trim() : "";
    const round = typeof payload?.round === "number" ? payload.round : 0;
    const validatorID = typeof payload?.validatorID === "string" ? payload.validatorID.trim() : "";
    const candidateID = typeof payload?.candidateID === "string" ? payload.candidateID.trim() : "";
    const signature = typeof payload?.signature === "string" ? payload.signature.trim() : "";
    if (!chainID || !validatorID || !candidateID || !signature) return { status: 400, error: "Missing required fields: chainID, round, validatorID, candidateID, and signature are mandatory." };

    const validators = await getValidators(this, chainID);
    if (!validators || !validators.validators) return { status: 403, error: "Validator set not initialized." };
    const validator = validators.validators.find(v => v.validatorID === validatorID);
    if (!validator) return { status: 403, error: "Validator not found in current set." };
    if (!validator.active) return { status: 403, error: "Validator is not active." };

    const message = canonicalize(chainID, String(round), candidateID);
    const isValid = await verifySignature(validator.publicKey, message, signature);
    if (!isValid) return { status: 403, error: "Invalid signature." };

    const now = Date.now();
    await pruneVotes(this, now, VOTE_TTL_MS);
    const vote = { round, validatorID, candidateID, timestamp: now };
    await this.ctx.storage.put(`vote:${chainID}:${round}:${validatorID}`, vote);
    return { status: 200, message: "Vote recorded." };
  }

  async votes(chainID, round) {
    const entries = await this.ctx.storage.list({ prefix: `vote:${chainID}:${round}:` });
    return Object.fromEntries([...entries].map(([key, value]) => [key.slice(`vote:${chainID}:${round}:`.length), value]));
  }

  async state(payload, env) {
    const chainID = typeof payload?.chainID === "string" ? payload.chainID.trim() : "";
    const phase = typeof payload?.phase === "string" ? payload.phase.trim() : "";
    const currentRootID = typeof payload?.currentRootID === "string" ? payload.currentRootID.trim() : "";
    const since = typeof payload?.since === "number" ? payload.since : 0;
    const signature = typeof payload?.signature === "string" ? payload.signature.trim() : "";
    if (!chainID || !phase || !currentRootID || !since || !signature) return { status: 400, error: "Missing required fields: chainID, phase, currentRootID, since, and signature are mandatory." };

    const validators = await getValidators(this, chainID);
    if (!validators || !validators.validators) return { status: 403, error: "Validator set not initialized." };
    const validator = validators.validators.find(v => v.validatorID === currentRootID);
    if (!validator) return { status: 403, error: "Current root ID not found in validator set." };
    if (!validator.active) return { status: 403, error: "Current root is not active." };

    const message = canonicalize(chainID, phase, currentRootID, String(since));
    const isValid = await verifySignature(validator.publicKey, message, signature);
    if (!isValid) return { status: 403, error: "Invalid signature." };

    const allowedPhases = ["NORMAL", "VOTING", "ACTING_ROOT", "CONFIRMED", "RECLAIMED"];
    if (!allowedPhases.includes(phase)) return { status: 400, error: `Invalid phase. Allowed: ${allowedPhases.join(", ")}` };

    const oldState = await this.ctx.storage.get(`state:${chainID}`);
    await this.ctx.storage.put(`state:${chainID}`, { chainID, phase, currentRootID, since });

    if (phase === "CONFIRMED" && (!oldState || oldState.phase !== "CONFIRMED")) {
      await setAuthorizedPublisher(this, chainID, currentRootID);
    }

    return { status: 200, message: "State updated.", previous: oldState || null };
  }

  async stateGet(chainID) {
    const state = await this.ctx.storage.get(`state:${chainID}`);
    return state || { chainID, phase: "NORMAL", currentRootID: "", since: 0 };
  }

  async validators(payload, env) {
    const chainID = typeof payload?.chainID === "string" ? payload.chainID.trim() : "";
    const version = typeof payload?.version === "number" ? payload.version : 0;
    const validators = Array.isArray(payload?.validators) ? payload.validators : [];
    const signature = typeof payload?.signature === "string" ? payload.signature.trim() : "";
    if (!chainID || !version || !validators.length || !signature) return { status: 400, error: "Missing required fields: chainID, version, validators, and signature are mandatory." };

    const publisherID = await getAuthorizedPublisher(this, chainID);
    let publisherPubKey;
    if (!publisherID) {
      const rootPubKey = env.ROOT_PUBLIC_KEY?.trim();
      if (!rootPubKey) return { status: 500, error: "ROOT_PUBLIC_KEY not configured and no authorized publisher set." };
      publisherPubKey = rootPubKey;
    } else {
      const currentValidators = await getValidators(this, chainID);
      if (!currentValidators || !currentValidators.validators) return { status: 403, error: "No existing validator set to verify publisher." };
      const publisher = currentValidators.validators.find(v => v.validatorID === publisherID);
      if (!publisher) return { status: 403, error: "Authorized publisher not found in current validator set." };
      publisherPubKey = publisher.publicKey;
    }

    const message = canonicalize(chainID, String(version), JSON.stringify(validators));
    const isValid = await verifySignature(publisherPubKey, message, signature);
    if (!isValid) return { status: 403, error: "Invalid publisher signature." };

    const existing = await getValidators(this, chainID);
    if (existing && existing.version >= version) {
      return { status: 409, error: `Stale version. Current: ${existing.version}, received: ${version}` };
    }

    await this.ctx.storage.put(`validators:${chainID}`, { chainID, version, validators });
    return { status: 200, message: "Validator set updated.", version };
  }

  async validatorsGet(chainID) {
    const validators = await getValidators(this, chainID);
    return validators || { chainID, version: 0, validators: [] };
  }

  async alarm() {
    const now = Date.now();
    const ttlMs = (await this.ctx.storage.get("nodeTtlMs")) || DEFAULT_NODE_TTL_MS;
    await this.prune(now, ttlMs);
    const heartbeatTtlMs = positiveInt(await this.ctx.storage.get("heartbeatTtlMs"), HEARTBEAT_TTL_MS / 1000) * 1000;
    await pruneHeartbeats(this, now, heartbeatTtlMs);
    await pruneVotes(this, now, VOTE_TTL_MS);
    const nextAlarm = Math.min(now + PRUNE_INTERVAL_MS, now + HEARTBEAT_PRUNE_INTERVAL_MS);
    await this.ctx.storage.setAlarm(nextAlarm);
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
    if (path === "/bootnodes" && request.method === "GET") return json(await store.bootnodes());
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
    if (path === "/heartbeat" && request.method === "POST") {
      let payload;
      try { payload = await request.json(); } catch { return json({ error: "Invalid JSON request body" }, 400); }
      const result = await store.heartbeat(payload, env);
      return result.error ? json({ error: result.error }, result.status) : json(result, result.status);
    }
    if (path.startsWith("/heartbeats/") && request.method === "GET") {
      const chainID = decodeURIComponent(path.slice(12));
      return json(await store.heartbeats(chainID));
    }
    if (path.startsWith("/consensus/") && request.method === "GET") {
      const parts = path.slice(11).split("/");
      if (parts.length > 2 || !parts[0]) return json({ error: "Usage: GET /consensus/:chainID/:round" }, 400);
      const round = parts[1] === undefined ? 0 : Number.parseInt(parts[1], 10);
      if (!Number.isInteger(round) || round < 0) return json({ error: "round must be a non-negative integer" }, 400);
      return json(await store.consensus(decodeURIComponent(parts[0]), round, env));
    }
    if (path === "/vote" && request.method === "POST") {
      let payload;
      try { payload = await request.json(); } catch { return json({ error: "Invalid JSON request body" }, 400); }
      const result = await store.vote(payload, env);
      return result.error ? json({ error: result.error }, result.status) : json(result, result.status);
    }
    if (path.startsWith("/votes/") && request.method === "GET") {
      const parts = path.slice(7).split("/");
      if (parts.length !== 2) return json({ error: "Usage: GET /votes/:chainID/:round" }, 400);
      const chainID = decodeURIComponent(parts[0]);
      const round = Number.parseInt(parts[1], 10);
      if (!Number.isInteger(round) || round < 1) return json({ error: "round must be a positive integer" }, 400);
      return json(await store.votes(chainID, round));
    }
    if (path === "/state" && request.method === "POST") {
      let payload;
      try { payload = await request.json(); } catch { return json({ error: "Invalid JSON request body" }, 400); }
      const result = await store.state(payload, env);
      return result.error ? json({ error: result.error }, result.status) : json(result, result.status);
    }
    if (path.startsWith("/state/") && request.method === "GET") {
      const chainID = decodeURIComponent(path.slice(7));
      return json(await store.stateGet(chainID));
    }
    if (path === "/validators" && request.method === "POST") {
      let payload;
      try { payload = await request.json(); } catch { return json({ error: "Invalid JSON request body" }, 400); }
      const result = await store.validators(payload, env);
      return result.error ? json({ error: result.error }, result.status) : json(result, result.status);
    }
    if (path.startsWith("/validators/") && request.method === "GET") {
      const chainID = decodeURIComponent(path.slice(11));
      return json(await store.validatorsGet(chainID));
    }
    return json({ error: "Not found" }, 404);
  },
};
