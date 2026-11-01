/**
 * Wire contract for the KNIRVMODELS daemon. The logic (byte-v1 tokenizer, advice
 * prompt and redaction, advice mapping, role lines) lives in the Rust core
 * (core/src/models.rs) and is reached through `models.*` operations, so this file
 * holds types, the HTTP client and thin calls. The core must be initialised first
 * (`initLocal()` or `SdkClient.init()`); `ModelsClient.advise` does it itself.
 * testvectors/models gate the core and the still-separate Go copy.
 */
import { callLocal, fromBase64, initLocal } from "../local.js";
/** Where the daemon serves each model. One namespace for all three. */
export const HEAD_PREFIX = "/models/head";
export const REGISTRY_PATH = "/models/registry";
export const HEALTH_PATH = "/models/health";
export const HEART_PREFIX = "/models/heart";
export const HASHER_PREFIX = "/models/hasher";
/** The model is up but cannot answer yet (HTTP 503). */
export class NotReadyError extends Error {
    constructor() { super("model not ready"); this.name = "NotReadyError"; }
}
export const BYTE_V1 = { EOS: 259, PAD: 256, BOS: 257 }; // token ids; the tokenizer itself is in the core
export function findDecision(r, name) {
    return r.decisions.find((d) => d.name === name);
}
/** Plan text: bytes up to the first EOS, specials dropped. */
export function planText(r) {
    const { bytes_base64 } = callLocal("models.plan_text", { plan_tokens: r.plan_tokens });
    return new TextDecoder().decode(fromBase64(bytes_base64));
}
/** Tokens the model takes: its sequence length minus the summary slots. */
export function inputWidth(m) {
    return callLocal("models.input_width", { config: m.config }).width;
}
/** BOS + the text's UTF-8 bytes, truncated or PAD-filled to `width`. */
export function encodeByteV1(text, width) {
    return callLocal("models.encode_byte_v1", { text, width }).tokens;
}
const now = () => new Date().toISOString();
/** Canonical model input; the trainer's `prompt` is pinned to the same vector. */
export function buildAdvisePrompt(c) {
    return callLocal("models.build_advise_prompt", c).prompt;
}
export function abstainAdvice(reason, modelVersion, manifest) {
    return callLocal("models.abstain_advice", { reason, model_version: modelVersion, dataset_manifest_hash: manifest, now: now() });
}
/** Maps decision heads onto SupervisorAdvice; every doubtful case abstains. */
export function adviceFromReasoning(r, c) {
    return callLocal("models.advice_from_reasoning", { reasoning: r, context: c, now: now() });
}
export class NoModelError extends Error {
    constructor() { super("no HEAD model has been published yet"); }
}
/** Client for the daemon's HEAD routes; `baseUrl` is the backend or daemon origin. */
export class ModelsClient {
    baseUrl;
    fetchImpl;
    headers;
    constructor(baseUrl, fetchImpl = fetch, headers = {}) {
        this.baseUrl = baseUrl;
        this.fetchImpl = fetchImpl;
        this.headers = headers;
    }
    async json(path, init) {
        const res = await this.fetchImpl(this.baseUrl + path, { ...init, headers: { ...this.headers, ...(init?.headers ?? {}) } });
        if (res.status === 503)
            throw new NoModelError();
        if (!res.ok)
            throw new Error(`head: HTTP ${res.status}`);
        return (await res.json());
    }
    registry() { return this.json(REGISTRY_PATH); }
    health() { return this.json(HEALTH_PATH); }
    heartHealth() { return this.json(`${HEART_PREFIX}/health`); }
    hasherStatus() { return this.json(`${HASHER_PREFIX}/status`); }
    async post(path, body) {
        const res = await this.fetchImpl(this.baseUrl + path, {
            method: "POST", headers: { ...this.headers, "Content-Type": "application/json" }, body: JSON.stringify(body),
        });
        if (res.status === 503)
            throw new NotReadyError();
        if (!res.ok)
            throw new Error(`models: HTTP ${res.status}`);
        return (await res.json());
    }
    /** HEART text; NotReadyError until HEART has a trained generator. */
    async heartGenerate(prompt) {
        const out = await this.post(`${HEART_PREFIX}/generate`, { prompt });
        if (out.error)
            throw new Error(`heart: ${out.error}`);
        return out.text ?? "";
    }
    hasherValidate(req) { return this.post(`${HASHER_PREFIX}/validate`, req); }
    hasherInfer(text) { return this.post(`${HASHER_PREFIX}/infer`, { text }); }
    hasherIndexSearch(req) {
        const { tenant, ...body } = req;
        return this.postWithHeaders(`${HASHER_PREFIX}/index/search`, body, tenant ? { "X-KNIRV-Tenant": tenant } : {});
    }
    /** Sync is operator-only; authority is sent by the client, never in ontology content. */
    hasherOntologySync(config) {
        return this.postWithHeaders(`${HASHER_PREFIX}/ontology/sync`, config, { "X-KNIRV-Privilege": "operator" });
    }
    hasherOntologyRemove(tenant) {
        return this.requestWithHeaders(`${HASHER_PREFIX}/ontology?tenant=${encodeURIComponent(tenant)}`, "DELETE", undefined, { "X-KNIRV-Privilege": "operator" });
    }
    hasherOntologyCheck(claim) {
        const { tenant, ...body } = claim;
        return this.postWithHeaders(`${HASHER_PREFIX}/ontology/check`, body, tenant ? { "X-KNIRV-Tenant": tenant } : {});
    }
    async postWithHeaders(path, body, headers) {
        const res = await this.fetchImpl(this.baseUrl + path, {
            method: "POST", headers: { ...this.headers, ...headers, "Content-Type": "application/json" }, body: JSON.stringify(body),
        });
        if (res.status === 503)
            throw new NotReadyError();
        if (!res.ok)
            throw new Error(`models: HTTP ${res.status}`);
        return (await res.json());
    }
    async requestWithHeaders(path, method, body, headers) {
        const res = await this.fetchImpl(this.baseUrl + path, {
            method, headers: { ...this.headers, ...headers, ...(body === undefined ? {} : { "Content-Type": "application/json" }) },
            ...(body === undefined ? {} : { body: JSON.stringify(body) }),
        });
        if (res.status === 503)
            throw new NotReadyError();
        if (!res.ok)
            throw new Error(`models: HTTP ${res.status}`);
        return (await res.json());
    }
    manifest() { return this.json(`${HEAD_PREFIX}/models`); }
    reason(tokens) {
        return this.json(`${HEAD_PREFIX}/reason`, {
            method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ tokens }),
        });
    }
    /** manifest → encode → reason → advice; every failure is an abstention. */
    async advise(c) {
        await initLocal();
        let m;
        try {
            m = await this.manifest();
        }
        catch {
            return abstainAdvice("no HEAD model available", "none", "");
        }
        let width;
        try {
            width = inputWidth(m);
        }
        catch {
            return abstainAdvice("no HEAD model available", "none", "");
        }
        try {
            return adviceFromReasoning(await this.reason(encodeByteV1(buildAdvisePrompt(c), width)), c);
        }
        catch (e) {
            return abstainAdvice(e instanceof NoModelError ? "no HEAD model available" : "HEAD could not answer", "none", "");
        }
    }
}
// ---- platform status (GET /api/models/status) -------------------------------------------
/** The backend's authenticated route for the platform-wide model view. */
export const STATUS_PATH = "/api/models/status";
/** The fixed list of role lines for a status document (core/src/models.rs). */
export function roles(s) {
    return callLocal("models.platform_roles", { status: s }).roles;
}
export * from "./frame.js";
export const INDEX_STATUS_PATH = `${HASHER_PREFIX}/index/status`;
