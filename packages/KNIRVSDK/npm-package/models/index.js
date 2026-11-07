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
export const HEART_DECISION_PREFIX = "/models/heart/decide";
export const HASHER_PREFIX = "/models/hasher";
/**
 * HANDS (server_tasks) namespace. Typed transport mirror of the Go SDK's
 * `sdk.HandsPrefix` and `sdk.RoleHands`; the Rust core (core/src/models.rs) is
 * the source of truth, so these are wire-value aliases plus thin HTTP calls.
 */
export const HandsPrefix = "/models/hands";
export const RoleHands = "server_tasks";
// Parity type test: this assignment only compiles while "server_tasks" is a
// member of ModelRole, i.e. while the core contract still reports the HANDS
// server_tasks role. If the core drops it, the build fails here first.
export function serverTasksRole() {
    return RoleHands;
}
/** The model is up but cannot answer yet (HTTP 503). */
export class NotReadyError extends Error {
    constructor() { super("model not ready"); this.name = "NotReadyError"; }
}
export const BYTE_V1 = { EOS: 259, PAD: 256, BOS: 257, SEP: 258, OPT_OPEN: 260, OPT_END: 261, VOCAB: 262 }; // token ids; the tokenizer itself is in the core
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
/**
 * Leakage-safe choice-set HEAD input after the model-owned summary slot:
 * BOS question SEP state SEP (OPT_OPEN choice OPT_END)*.
 */
export function encodeChoiceSetV1(question, state, choices, width) {
    return callLocal("models.encode_choice_set_v1", { question, state, choices: [...choices], width }).tokens;
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
    /** Reads a non-2xx, non-503 response body so transport errors surface the
     *  server's contract text instead of a bare status — e.g. the Rust core's
     *  "no choice-set heads" rejection of a v1 /sets choice-set request. Reads the
     *  body once (as text), then optionally JSON-parses it to prefer the `error`
     *  field; falls back to the raw text so a non-JSON body is not lost. */
    async errorDetail(res) {
        let text;
        try {
            text = await res.text();
        }
        catch {
            return "";
        }
        if (!text || text.length === 0)
            return "";
        try {
            const body = JSON.parse(text);
            if (body && typeof body.error === "string" && body.error.length > 0)
                return body.error;
        }
        catch { /* body was not JSON; fall through to raw text */ }
        return text.slice(0, 240);
    }
    async json(path, init) {
        const res = await this.fetchImpl(this.baseUrl + path, { ...init, headers: { ...this.headers, ...(init?.headers ?? {}) } });
        if (res.status === 503)
            throw new NoModelError();
        if (!res.ok) {
            const detail = await this.errorDetail(res);
            throw new Error(detail ? `head: HTTP ${res.status}: ${detail}` : `head: HTTP ${res.status}`);
        }
        return (await res.json());
    }
    registry() { return this.json(REGISTRY_PATH); }
    health() { return this.json(HEALTH_PATH); }
    heartHealth() { return this.json(`${HEART_PREFIX}/health`); }
    hasherStatus() { return this.json(`${HASHER_PREFIX}/status`); }
    // ---- HANDS (server_tasks) -------------------------------------------------
    // Thin transport mirror of the Go SDK's HandsPrefix/RoleHands and the
    // KNIRVMODELS HANDS facade (pkg/models/hands.go). Untrained HANDS returns 503
    // (NotReadyError) for generate/plan/act/weights; status/health/models stay 200.
    handsStatus() { return this.json(`${HandsPrefix}/status`); }
    handsHealth() { return this.json(`${HandsPrefix}/health`); }
    handsModels() { return this.json(`${HandsPrefix}/models`); }
    /** 503 -> NotReadyError until HANDS has a checkpoint. */
    handsWeights() { return this.requestWithHeaders(`${HandsPrefix}/weights`, "GET", undefined, {}); }
    handsGenerate(prompt, opts = {}) {
        return this.post(`${HandsPrefix}/generate`, { prompt, ...opts });
    }
    handsPlan(prompt, opts = {}) {
        return this.post(`${HandsPrefix}/plan`, { prompt, ...opts });
    }
    handsAct(prompt, opts = {}) {
        return this.post(`${HandsPrefix}/act`, { prompt, ...opts });
    }
    // ---- HEAD VLM (vision / multimodal) ----------------------------------------
    // Transport mirror of the Go SDK HeadVision / HeadGenerate / HeadStatus. The four
    // vision operations each reach their own route; an unknown operation is rejected
    // client-side before any request leaves the process.
    headStatus() { return this.json(`${HEAD_PREFIX}/status`); }
    headCapabilities() { return this.json(`${HEAD_PREFIX}/capabilities`); }
    headGenerate(req) {
        return this.post(`${HEAD_PREFIX}/generate`, req);
    }
    headVision(req) {
        if (!["embed", "describe", "ground", "visual-reason"].includes(req.operation)) {
            return Promise.reject(new Error(`head: unsupported vision operation ${JSON.stringify(req.operation)} (want one of embed, describe, ground, visual-reason)`));
        }
        return this.post(`${HEAD_PREFIX}/${req.operation}`, req);
    }
    async post(path, body) {
        const res = await this.fetchImpl(this.baseUrl + path, {
            method: "POST", headers: { ...this.headers, "Content-Type": "application/json" }, body: JSON.stringify(body),
        });
        if (res.status === 503)
            throw new NotReadyError();
        if (!res.ok) {
            const detail = await this.errorDetail(res);
            throw new Error(detail ? `models: HTTP ${res.status}: ${detail}` : `models: HTTP ${res.status}`);
        }
        return (await res.json());
    }
    /** HEART text; NotReadyError until HEART has a trained generator. */
    async heartGenerate(prompt) {
        const out = await this.post(`${HEART_PREFIX}/generate`, { prompt });
        if (out.error)
            throw new Error(`heart: ${out.error}`);
        return out.text ?? "";
    }
    /** Decision-aware HEART generate. When resolve is true HEART consults the
     *  decision core; otherwise it degrades to plain generation and never fails
     *  just because the core is unavailable. Returns the full response. */
    async heartGenerateWithDecision(prompt, resolve, decision) {
        const body = { prompt };
        if (resolve)
            body.with_decision = true;
        if (decision)
            body.decision = decision;
        const out = await this.post(`${HEART_PREFIX}/generate`, body);
        if (out.error)
            throw new Error(`heart: ${out.error}`);
        return out;
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
    /** Verify a text or claims against the evidence index. A degraded answer (judge down) is a normal result. */
    hasherVerify(req) {
        const bad = callLocal("models.verify_request_check", { request: req }).error;
        if (bad)
            return Promise.reject(new Error(bad));
        const { tenant, ...body } = req;
        return this.postWithHeaders(VERIFY_PATH, body, tenant ? { "X-KNIRV-Tenant": tenant } : {});
    }
    /** Read a deferred job; only the tenant that submitted it can. */
    hasherVerifyJob(id, tenant) {
        return this.requestWithHeaders(VERIFY_JOBS_PATH + encodeURIComponent(id), "GET", undefined, tenant ? { "X-KNIRV-Tenant": tenant } : {});
    }
    async postWithHeaders(path, body, headers) {
        const res = await this.fetchImpl(this.baseUrl + path, {
            method: "POST", headers: { ...this.headers, ...headers, "Content-Type": "application/json" }, body: JSON.stringify(body),
        });
        if (res.status === 503)
            throw new NotReadyError();
        if (!res.ok) {
            const detail = await this.errorDetail(res);
            throw new Error(detail ? `models: HTTP ${res.status}: ${detail}` : `models: HTTP ${res.status}`);
        }
        return (await res.json());
    }
    async requestWithHeaders(path, method, body, headers) {
        const res = await this.fetchImpl(this.baseUrl + path, {
            method, headers: { ...this.headers, ...headers, ...(body === undefined ? {} : { "Content-Type": "application/json" }) },
            ...(body === undefined ? {} : { body: JSON.stringify(body) }),
        });
        if (res.status === 503)
            throw new NotReadyError();
        if (!res.ok) {
            const detail = await this.errorDetail(res);
            throw new Error(detail ? `models: HTTP ${res.status}: ${detail}` : `models: HTTP ${res.status}`);
        }
        return (await res.json());
    }
    /** HEAD VLM manifest (vision / mirrored text). Narrow, VLM-specific contract. */
    manifest() { return this.json(`${HEAD_PREFIX}/models`); }
    /** HEART decision core manifest; the manifest Advise needs. */
    decisionManifest() { return this.json(`${HEART_DECISION_PREFIX}/models`); }
    reason(tokens) {
        return this.json(`${HEART_DECISION_PREFIX}/reason`, {
            method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ tokens }),
        });
    }
    /** Choice-set questions (knirv-heart/v2 models). A v1 model answers with a
     *  "no choice-set heads" error, rejecting the request just like every other runtime. */
    reasonSets(questions) {
        return this.json(`${HEART_DECISION_PREFIX}/sets`, {
            method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ questions }),
        });
    }
    /** decisionManifest → encode → reason → advice; every failure is an abstention. */
    async advise(c) {
        await initLocal();
        let m;
        try {
            m = await this.decisionManifest();
        }
        catch {
            return abstainAdvice("no decision model available", "none", "");
        }
        let width;
        try {
            width = inputWidth(m);
        }
        catch {
            return abstainAdvice("no decision model available", "none", "");
        }
        try {
            return adviceFromReasoning(await this.reason(encodeByteV1(buildAdvisePrompt(c), width)), c);
        }
        catch (e) {
            return abstainAdvice(e instanceof NoModelError ? "no decision model available" : "decision model could not answer", "none", "");
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
/** POST /models/hasher/verify (Task 3.4); twin of Go VerifyRequest. */
export const VERIFY_PATH = `${HASHER_PREFIX}/verify`;
export const VERIFY_JOBS_PATH = `${HASHER_PREFIX}/verify/jobs/`;
export const GATE_NAMES = ["refused", "discouraged", "uncommitted", "seen", "attested", "accepted", "committed"];
/** The first way a response breaks the contract, or null. The invariants live in the SDK core (models.verify_check); twin of Go VerifyResponse.Check. */
export function checkVerify(r) {
    return callLocal("models.verify_check", { response: r }).error;
}
