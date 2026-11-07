import { DurableObject } from "cloudflare:workers";
import { sha256 } from "@noble/hashes/sha2.js";
import { hmac } from "@noble/hashes/hmac.js";
import * as secp256k1 from "@noble/secp256k1";
import { buildTarget, defaultProbes, tokensMatch, validateNode, validationKey, validationMode, validationSettings } from "./validation.js";

// Root tx-chain registry: the signed POST /registry/register and signed lookup
// are verified by the Rust-core WASM verifier from the KNIRV SDK (no JS crypto).
// The trusted root identity and the immutable root registry id are read ONLY from
// Worker configuration (env.ROOT_PUBLIC_ADDRESS / env.ROOT_REGISTRY_ID) — never
// from the request or the record.
import { initLocal as sdkInitLocal } from "../../../packages/KNIRVSDK/npm-package/local.js";
import { verifyRegistryRecord as sdkVerifyRegistryRecord } from "../../../packages/KNIRVSDK/npm-package/signing.js";

secp256k1.hashes.sha256 = sha256;
secp256k1.hashes.hmacSha256 = (key, msg) => hmac(sha256, key, msg);

const JSON_HEADERS = { "content-type": "application/json; charset=utf-8" };
const MAX_REGISTRY_BODY_BYTES = 64 * 1024;
const MAX_CHAIN_ID_LEN = 64;

// --- Rust-core WASM verifier (KNIRV SDK) -----------------------------------
let sdkReady = null;
function ensureSdk() {
  if (!sdkReady) sdkReady = sdkInitLocal();
  return sdkReady;
}

// Read and parse a JSON body with a hard size cap and duplicate-key rejection.
// `request.json()` silently keeps the last of duplicate keys, which lets an
// attacker smuggle a canonicalization-ambiguous object; we reject on the spot
// with a small strict parser (the standard parser collapses duplicates first).
async function readJsonBody(request, maxBytes = MAX_REGISTRY_BODY_BYTES) {
  const declared = request.headers.get("content-length");
  if (declared && Number.parseInt(declared, 10) > maxBytes) {
    return { error: json({ error: "Request body too large" }, 413) };
  }
  const text = await request.text();
  if (text.length > maxBytes) return { error: json({ error: "Request body too large" }, 413) };
  try {
    return { value: parseStrict(text) };
  } catch {
    return { error: json({ error: "Invalid JSON request body" }, 400) };
  }
}

// Minimal JSON parser that throws on duplicate object keys (security control).
function parseStrict(str) {
  let i = 0;
  const n = str.length;
  const ws = () => { while (i < n && /\s/.test(str[i])) i++; };
  function strTok() {
    let s = "";
    i++; // opening quote
    while (i < n) {
      const ch = str[i++];
      if (ch === '"') return s;
      if (ch === "\\") {
        const e = str[i++];
        if (e === "u") { i += 4; }
      } else s += ch;
    }
    throw new Error("bad string");
  }
  function numTok() {
    const start = i;
    while (i < n && /[-0-9.eE+]/.test(str[i])) i++;
    return Number(str.slice(start, i));
  }
  function arrTok() {
    i++; const arr = [];
    ws(); if (str[i] === "]") { i++; return arr; }
    for (;;) {
      arr.push(valTok());
      ws();
      const c = str[i++];
      if (c === ",") continue;
      if (c === "]") return arr;
      throw new Error("bad array");
    }
  }
  function objTok() {
    i++; const keys = new Set(); const obj = {};
    ws(); if (str[i] === "}") { i++; return obj; }
    for (;;) {
      ws();
      const key = strTok();
      if (keys.has(key)) throw new Error("duplicate key");
      keys.add(key);
      ws();
      if (str[i++] !== ":") throw new Error("bad object");
      obj[key] = valTok();
      ws();
      const c = str[i++];
      if (c === ",") continue;
      if (c === "}") return obj;
      throw new Error("bad object");
    }
  }
  function valTok() {
    ws();
    const c = str[i];
    if (c === "{") return objTok();
    if (c === "[") return arrTok();
    if (c === '"') return strTok();
    if (c === "t") { i += 4; return true; }
    if (c === "f") { i += 5; return false; }
    if (c === "n") { i += 4; return null; }
    return numTok();
  }
  const result = valTok();
  ws();
  if (i !== n) throw new Error("trailing data");
  return result;
}

// Explicit CORS: never a wildcard for the write endpoint. Only an origin set in
// configuration is reflected; otherwise no CORS header is emitted (same-origin only).
function corsHeaders(env, request) {
  const allow = env.CORS_ALLOW_ORIGIN?.trim();
  if (!allow) return {};
  const origin = request.headers.get("origin");
  if (origin && (allow === "*" || allow.split(",").map((o) => o.trim()).includes(origin))) {
    return {
      "access-control-allow-origin": origin,
      "access-control-allow-methods": "GET, POST, OPTIONS",
      "access-control-allow-headers": "content-type",
    };
  }
  return {};
}
const DEFAULT_NODE_TTL_MS = 60 * 60 * 1000;
const PRUNE_INTERVAL_MS = 15 * 60 * 1000;
const HEARTBEAT_TTL_MS = 45 * 1000;
const HEARTBEAT_PRUNE_INTERVAL_MS = 30 * 1000;
const VOTE_TTL_MS = 24 * 60 * 60 * 1000;
const VALIDATION_RECORD_TTL_MS = 24 * 60 * 60 * 1000;

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
  const rows = Object.entries(status.nodes).map(([chainID, node]) => `<tr><td><code>${escapeHTML(chainID)}</code></td><td>${escapeHTML(node.type)}</td><td>${escapeHTML(node.ip)}</td><td>${node.port}</td><td>${node.type === "bootnode" ? (node.validated ? "Yes" : "No") : "-"}</td><td>${node.active ? "Active" : "Inactive"}</td><td>${escapeHTML(new Date(node.lastSeen).toLocaleString())}</td></tr>`).join("");
  const validatorRows = status.validators && Object.entries(status.validators).map(([validatorID, v]) => `<tr><td><code>${escapeHTML(validatorID)}</code></td><td>${escapeHTML(v.publicKey || "")}</td><td>${v.active ? "Active" : "Jailed/Demoted"}</td></tr>`).join("");
  return `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>KNIRV Network Registry</title><style>body{font-family:system-ui,sans-serif;max-width:1100px;margin:2rem auto;padding:0 1rem;color:#172033}table{width:100%;border-collapse:collapse}th,td{padding:.65rem;border-bottom:1px solid #d8dee9;text-align:left}th{background:#f4f7fb}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(160px,1fr));gap:1rem}.card{background:#f4f7fb;border-radius:.5rem;padding:1rem}code{word-break:break-all}</style></head><body><h1>KNIRV Network Registry</h1><div class="grid"><div class="card">Status<br><strong>${escapeHTML(status.status)}</strong></div><div class="card">Registered nodes<br><strong>${status.registeredNodes}</strong></div><div class="card">Active nodes<br><strong>${status.activeNodes}</strong></div><div class="card">Version<br><strong>${escapeHTML(status.version)}</strong></div></div><h2>Registered Nodes</h2><table><thead><tr><th>Chain ID</th><th>Type</th><th>IP</th><th>Port</th><th>Validated</th><th>Status</th><th>Last seen</th></tr></thead><tbody>${rows || '<tr><td colspan="7">No nodes registered</td></tr>'}</tbody></table>${validatorRows ? '<h2>Validators</h2><table><thead><tr><th>Validator ID</th><th>Public Key</th><th>Status</th></tr></thead><tbody>' + validatorRows + '</tbody></table>' : ''}</</body></html>`;
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
  async register(payload, ipFallback, ttlMs, validation, validatedOnly = false) {
    const chainID = typeof payload?.chainID === "string" ? payload.chainID.trim() : "";
    const ip = typeof payload?.ip === "string" && payload.ip.trim() ? payload.ip.trim() : ipFallback;
    const port = typeof payload?.port === "string" || typeof payload?.port === "number" ? Number.parseInt(payload.port, 10) : NaN;
    const type = payload?.type === "bootnode" ? "bootnode" : "client";
    const peerRaw = payload?.peerID ?? payload?.nodeID;
    const peerID = typeof peerRaw === "string" && peerRaw ? peerRaw : undefined;
    const txChainRaw = payload?.txChainAddress;
    // A tx-chain address is a payment-route field and may ONLY be published through
    // the signed POST /registry/register path (verified by the Rust-core WASM). The
    // unsigned discovery path never persists or returns one — it rejects outright so
    // it can never overwrite the root's payment identity.
    if (typeof txChainRaw === "string" && txChainRaw.trim()) {
      return { status: 400, error: "A tx-chain address requires a signed registry record." };
    }
    if (!chainID || !Number.isInteger(port)) return { status: 400, error: "Missing required fields: chainID and port are mandatory." };
    if (port < 1 || port > 65535 || !ip) return { status: 400, error: "Invalid data types or value for chainID, port, or IP" };

    const now = Date.now();
    await this.prune(now, ttlMs);
    await this.ctx.storage.put("nodeTtlMs", ttlMs);
    const node = { ip, port, lastSeen: now, type, ...(peerID ? { peerID } : {}), ...(validation ? { validated: true, validatedAt: validation.validatedAt, ...(validation.fqdn ? { fqdn: validation.fqdn } : {}) } : {}) };
    await this.ctx.storage.put(`node:${chainID}`, node);
    let suggestedBootnode;
    if (type === "client") {
      const bootnodes = await this.bootnodes(validatedOnly);
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
    // Quarantined (legacy unsigned) tx-chain entries are never returned to clients.
    if (!node || node.quarantined) return { status: 404, error: "Node not found" };
    return { status: 200, node };
  }

  async allNodes() {
    const entries = await this.ctx.storage.list({ prefix: "node:" });
    return Object.fromEntries([...entries].map(([key, value]) => [key.slice(5), value]));
  }

  // validatedOnly hides bootnodes that have not passed validation, so clients
  // are never pointed at them. Non-bootnode entries are never filtered. Quarantined
  // (legacy unsigned) tx-chain entries are always excluded from client views.
  async nodes(validatedOnly = false) {
    const nodes = await this.allNodes();
    return Object.fromEntries(
      Object.entries(nodes).filter(
        ([, node]) =>
          !node.quarantined && (!validatedOnly || node.type !== "bootnode" || node.validated === true),
      ),
    );
  }

  async bootnodes(validatedOnly = false) {
    const nodes = await this.nodes(validatedOnly);
    return Object.entries(nodes).filter(([, node]) => node.type === "bootnode").map(([chainID, node]) => ({ chainID, ...node }));
  }

  // Signed root tx-chain registry record publication. The envelope is verified by
  // the Rust-core WASM verifier; the trusted root identity and immutable root
  // registry id come from env (never the request). The signed envelope and the
  // highest nonce are persisted in ONE Durable Object storage transaction (or, where
  // transactions are unavailable, a single combined put) so a crash between writes
  // leaves no inconsistent state. The per-instance mutex serializes concurrent
  // submissions so the read-modify-write cannot race; a nonce <= the stored
  // high-water mark is rejected.
  async registerSigned(payload, env) {
    const run = () => this.registerSignedInner(payload, env);
    const prev = this._regMutex || Promise.resolve();
    const next = prev.then(run);
    // Keep the chain alive even if a run rejects; the rejection still propagates to
    // the immediate caller (the HTTP handler turns it into a 500).
    this._regMutex = next.catch(() => {});
    return next;
  }

  async registerSignedInner(payload, env) {
    const signed = payload?.signed;
    if (!signed || typeof signed !== "object" || !signed.envelope || !signed.signature) {
      return { status: 400, error: "Signed envelope is required." };
    }
    // Strict outer schema: only the `signed` field is accepted.
    const bodyKeys = payload && typeof payload === "object" ? Object.keys(payload) : [];
    if (bodyKeys.length !== 1 || bodyKeys[0] !== "signed") {
      return { status: 400, error: "Unknown field in signed registration body." };
    }
    const rootAddress = env.ROOT_PUBLIC_ADDRESS?.trim();
    const rootRegistryId = env.ROOT_REGISTRY_ID?.trim();
    if (!rootAddress || !rootRegistryId) {
      return { status: 500, error: "Registry trust configuration is missing." };
    }
    await ensureSdk();
    let record;
    try {
      record = await sdkVerifyRegistryRecord(signed, rootAddress, rootRegistryId, new Date());
    } catch {
      // Generic external error: never leak verifier/WASM internals.
      return { status: 401, error: "Invalid registry record signature." };
    }
    const chainId = record.chain_id;
    if (typeof chainId !== "string" || chainId.length === 0 || chainId.length > MAX_CHAIN_ID_LEN) {
      return { status: 400, error: "Invalid registry record chain id." };
    }
    if (typeof record.nonce !== "number" || !Number.isSafeInteger(record.nonce) || record.nonce < 0) {
      return { status: 400, error: "Invalid registry record nonce." };
    }
    const nonce = BigInt(record.nonce);
    const storedNonceRaw = await this.ctx.storage.get(`registryNonce:${chainId}`);
    const storedNonce = storedNonceRaw === null || storedNonceRaw === undefined ? -1n : BigInt(storedNonceRaw);
    if (nonce <= storedNonce) {
      return { status: 409, error: "Stale or replayed registry record (nonce not greater than stored)." };
    }
    // Atomic persistence: both keys commit together, or neither does.
    const storage = this.ctx.storage;
    const writeAtomic = async (txn) => {
      await txn.put(`signedEnvelope:${chainId}`, signed);
      await txn.put(`registryNonce:${chainId}`, record.nonce);
    };
    try {
      if (typeof storage.transaction === "function") {
        await storage.transaction(writeAtomic);
      } else {
        // Fallback: a single put of one object holding both values is atomic at the
        // storage layer.
        await storage.put(`registryRecord:${chainId}`, { signed, nonce: record.nonce });
      }
    } catch {
      return { status: 500, error: "Failed to persist registry record." };
    }
    return { status: 201, message: "Registry record published.", chainID: chainId, record, signed };
  }

  // Signed lookup: returns the originally published signed envelope (not a decoded
  // or bare record), so clients can re-verify it themselves.
  async lookupSigned(chainID) {
    const stored =
      (await this.ctx.storage.get(`signedEnvelope:${chainID}`)) ??
      (await this.ctx.storage.get(`registryRecord:${chainID}`))?.signed;
    if (!stored) return { status: 404, error: "Registry record not found" };
    return { status: 200, signed: stored };
  }

  async validationGet(chainID) {
    return (await this.ctx.storage.get(`validation:${chainID}`)) || null;
  }

  async validationPut(chainID, record) {
    await this.ctx.storage.put(`validation:${chainID}`, record);
    await this.ctx.storage.setAlarm(Date.now() + PRUNE_INTERVAL_MS);
  }

  async status(ttlMs, version) {
    const now = Date.now();
    await this.prune(now, ttlMs);
    const nodes = await this.allNodes();
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

    // Replica telemetry is additive to the original heartbeat contract.  It
    // has its own signature so an authenticated legacy heartbeat cannot be
    // paired with forged mode/lag fields during a rolling deployment.
    let replica;
    if (payload?.replica !== undefined || payload?.replicaSignature !== undefined) {
      const raw = payload?.replica;
      const replicaSignature = typeof payload?.replicaSignature === "string" ? payload.replicaSignature.trim() : "";
      const safeUInt = (value) => typeof value === "number" && Number.isSafeInteger(value) && value >= 0;
      const safeInt = (value) => typeof value === "number" && Number.isSafeInteger(value) && value >= 0;
      const safeCount = (value) => Number.isSafeInteger(value) && value >= 0 && value <= 6;
      if (!raw || !replicaSignature || !["active", "standby", "off"].includes(raw.mode) ||
        !safeUInt(raw.headSeq) || !safeUInt(raw.lagSeq) || !safeInt(raw.lagSeconds) ||
        typeof raw.diverged !== "boolean" || typeof raw.paused !== "boolean" || !safeCount(raw.unverifiedClasses)) {
        return { status: 400, error: "Invalid replica telemetry." };
      }
      const replicaMessage = canonicalize(chainID, validatorID, "replica-v1", raw.mode,
        String(raw.headSeq), String(raw.lagSeq), String(raw.lagSeconds), String(raw.diverged), String(raw.paused), String(raw.unverifiedClasses));
      if (!await verifySignature(validator.publicKey, replicaMessage, replicaSignature)) {
        return { status: 403, error: "Invalid replica telemetry signature." };
      }
      replica = { mode: raw.mode, headSeq: raw.headSeq, lagSeq: raw.lagSeq, lagSeconds: raw.lagSeconds,
        diverged: raw.diverged, paused: raw.paused, unverifiedClasses: raw.unverifiedClasses };
    }

    const ttlMs = positiveInt(env.HEARTBEAT_TTL_SECONDS, HEARTBEAT_TTL_MS / 1000) * 1000;
    const now = Date.now();
    await pruneHeartbeats(this, now, ttlMs);
    const heartbeat = { validatorID, rootObservedHealthy, rootLastSeenMs, syncHeight, lastSeen: now, timestamp: now, ...(replica ? { replica } : {}) };
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

  async pruneValidations(now) {
    const entries = await this.ctx.storage.list({ prefix: "validation:" });
    const expired = [...entries].filter(([, record]) => now - (record.attemptedAt || 0) > VALIDATION_RECORD_TTL_MS).map(([key]) => key);
    if (expired.length) await this.ctx.storage.delete(expired);
  }

  async alarm() {
    const now = Date.now();
    const ttlMs = (await this.ctx.storage.get("nodeTtlMs")) || DEFAULT_NODE_TTL_MS;
    await this.prune(now, ttlMs);
    const heartbeatTtlMs = positiveInt(await this.ctx.storage.get("heartbeatTtlMs"), HEARTBEAT_TTL_MS / 1000) * 1000;
    await pruneHeartbeats(this, now, heartbeatTtlMs);
    await pruneVotes(this, now, VOTE_TTL_MS);
    await this.pruneValidations(now);
    const nextAlarm = Math.min(now + PRUNE_INTERVAL_MS, now + HEARTBEAT_PRUNE_INTERVAL_MS);
    await this.ctx.storage.setAlarm(nextAlarm);
  }
}

// registerNode applies the bootnode validation requirement, then registers.
// Clients are registered exactly as before. A bootnode must hold a passing,
// current validation for its ip/port/fqdn; BOOTNODE_VALIDATION=monitor records
// results without blocking, =off disables the requirement.
export async function registerNode(store, payload, ipFallback, ttlMs, env, probes = defaultProbes) {
  const mode = validationMode(env);
  const type = payload?.type === "bootnode" ? "bootnode" : "client";
  const validatedOnly = mode === "enforce";
  if (type !== "bootnode" || mode === "off") return store.register(payload, ipFallback, ttlMs, undefined, validatedOnly);

  const chainID = typeof payload?.chainID === "string" ? payload.chainID.trim() : "";
  const ip = typeof payload?.ip === "string" && payload.ip.trim() ? payload.ip.trim() : ipFallback;
  const port = typeof payload?.port === "string" || typeof payload?.port === "number" ? Number.parseInt(payload.port, 10) : NaN;
  // Let register() produce its usual 400 for malformed basics.
  if (!chainID || !Number.isInteger(port) || port < 1 || port > 65535 || !ip) return store.register(payload, ipFallback, ttlMs, undefined, validatedOnly);

  const settings = validationSettings(env);
  const peerRaw = payload?.peerID ?? payload?.nodeID;
  const built = buildTarget({ chainID, ip, port, fqdn: payload?.fqdn, peerID: typeof peerRaw === "string" ? peerRaw : "" }, settings);
  const now = Date.now();
  let result;
  if (built.error) {
    result = { passed: false, checks: { connectivity: false, health: false, security: false, performance: false }, details: { error: built.error }, key: "", validatedAt: now };
  } else {
    const key = validationKey(built.target);
    const previous = await store.validationGet(chainID);
    if (previous?.key === key && previous.passed && now - previous.validatedAt < settings.ttlMs) {
      result = previous;
    } else if (previous?.key === key && !previous.passed && now - previous.attemptedAt < settings.failCooldownMs) {
      const retryAfter = Math.ceil((settings.failCooldownMs - (now - previous.attemptedAt)) / 1000);
      if (mode === "enforce") return { status: 429, error: "Bootnode validation failed recently; retry later.", retryAfter, validation: { passed: false, checks: previous.checks, details: previous.details } };
      result = previous;
    } else {
      result = await validateNode(built.target, settings, probes);
      await store.validationPut(chainID, { ...result, attemptedAt: now });
    }
  }

  const validation = { passed: result.passed, checks: result.checks, details: result.details };
  if (!result.passed && mode === "enforce") return { status: 422, error: "Bootnode failed validation and was not registered.", validation };
  const registered = await store.register(payload, ipFallback, ttlMs, result.passed ? { validatedAt: result.validatedAt, fqdn: built.target?.fqdn } : undefined, validatedOnly);
  return registered.error ? registered : { ...registered, validation };
}

// validateRequest implements the validation-service contract used by
// ONBOARDING.KNIRV.COM's run-tests.ts: POST { registrationId, fqdn, publicIps,
// expectedPorts } -> { checks, details }. It is read-only and only probes
// public IPs and allowlisted hostnames, so it needs no secret by default. If
// VALIDATION_SERVICE_TOKEN is set it is enforced as an optional extra lock.
export async function validateRequest(request, env, probes = defaultProbes) {
  if (env.VALIDATION_SERVICE_TOKEN) {
    const header = request.headers.get("authorization") || "";
    if (!header.startsWith("Bearer ") || !tokensMatch(header.slice(7), env.VALIDATION_SERVICE_TOKEN)) return json({ error: "Unauthorized" }, 401);
  }
  let body;
  try { body = await request.json(); } catch { return json({ error: "Invalid JSON request body" }, 400); }
  const ips = Array.isArray(body?.publicIps) ? body.publicIps : [];
  const ports = Array.isArray(body?.expectedPorts) ? body.expectedPorts : [];
  const ip = typeof ips[0] === "string" ? ips[0] : "";
  const port = Number.parseInt(ports[0], 10);
  const settings = validationSettings(env);
  const built = buildTarget({ chainID: "", ip, port, fqdn: body?.fqdn, peerID: "" }, settings);
  if (built.error) return json({ checks: { connectivity: false, health: false, security: false, performance: false }, details: { error: built.error } });
  const result = await validateNode(built.target, settings, probes);
  return json({ checks: result.checks, details: { ...result.details, registrationId: typeof body?.registrationId === "string" ? body.registrationId : undefined, identityChecked: false } });
}

export default {
  async fetch(request, env) {
    const url = new URL(request.url);
    const path = url.pathname.replace(/\/$/, "") || "/";
    const ttlMs = positiveInt(env.NODE_TTL_SECONDS, DEFAULT_NODE_TTL_MS / 1000) * 1000;
    const store = env.REGISTRY.getByName("network-registry");
    if (request.method === "OPTIONS") {
      if (path === "/registry/register") return new Response(null, { status: 204, headers: corsHeaders(env, request) });
      return new Response(null, { status: 204, headers: { "access-control-allow-origin": "*", "access-control-allow-methods": "GET, POST, OPTIONS", "access-control-allow-headers": "content-type, authorization" } });
    }
    if (path === "/") return Response.redirect(new URL("/status", url), 308);
    if (path === "/register" && request.method === "GET") return json({ error: "Method Not Allowed. Use POST to /register for node registration.", hint: "See /status for registry information." }, 405, { allow: "POST" });
    if (path === "/register" && request.method === "POST") {
      let payload;
      try { payload = await request.json(); } catch { return json({ error: "Invalid JSON request body" }, 400); }
      const result = await registerNode(store, payload, sourceIP(request), ttlMs, env);
      if (result.error) return json({ error: result.error, ...(result.validation ? { validation: result.validation } : {}) }, result.status, result.retryAfter ? { "retry-after": String(result.retryAfter) } : {});
      return json(result, result.status);
    }
    if (path.startsWith("/lookup/") && request.method === "GET") {
      const result = await store.lookup(decodeURIComponent(path.slice(8)));
      return result.error ? json({ error: result.error }, result.status) : json(result.node);
    }
    // Signed root tx-chain registry record endpoints (verified by Rust-core WASM).
    if (path === "/registry/register" && request.method === "POST") {
      const body = await readJsonBody(request);
      if (body.error) return body.error;
      const result = await store.registerSigned(body.value, env);
      return result.error
        ? json({ error: result.error }, result.status, corsHeaders(env, request))
        : json(result, result.status, corsHeaders(env, request));
    }
    if (path.startsWith("/registry/lookup/") && request.method === "GET") {
      const result = await store.lookupSigned(decodeURIComponent(path.slice("/registry/lookup/".length)));
      return result.error ? json({ error: result.error }, result.status) : json(result.signed);
    }
    const validatedOnly = validationMode(env) === "enforce";
    if (path === "/nodes" && request.method === "GET") return json(await store.nodes(validatedOnly));
    if (path === "/bootnodes" && request.method === "GET") return json(await store.bootnodes(validatedOnly));
    if (path === "/validate" && request.method === "POST") return validateRequest(request, env);
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
