// Contract tests for src/models against the shared fixtures in testvectors/models
// (the Go twin runs the same files: go-package/models).
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { initLocal } from "../local.js";

import {
  BYTE_V1, HEAD_PREFIX, HandsPrefix, ModelsClient, NotReadyError, RoleHands, STATUS_PATH, abstainAdvice, adviceFromReasoning, buildAdvisePrompt, checkVerify, encodeByteV1, encodeChoiceSetV1, inputWidth, planText, roles,
} from "../models/index.js";

await initLocal(); // the models logic runs in the core

const vec = async (name) => JSON.parse(await readFile(new URL(`../../testvectors/models/${name}`, import.meta.url), "utf8"));

test("prompt matches the shared vector", async () => {
  const v = await vec("prompt_vector.json");
  assert.equal(buildAdvisePrompt(v.context), v.prompt);
});

test("prompt is redacted and deterministic", async () => {
  const v = (await vec("prompt_redaction.json")).redaction;
  const p = buildAdvisePrompt(v.context);
  assert.equal(p, buildAdvisePrompt(v.context));
  for (const s of v.must_not_contain) assert.ok(!p.includes(s), `secret ${s} reached the model input`);
});

test("byte-v1 vectors", async () => {
  for (const c of await vec("byte_v1.json")) assert.deepEqual(encodeByteV1(c.text, c.width), c.tokens);
});

test("choice-set prompt matches the shared vector", async () => {
	const c = (await vec("prompt_vector.json")).set_prompt;
	assert.deepEqual(encodeChoiceSetV1(c.question, c.state, c.choices, c.width), c.tokens);
	assert.equal(BYTE_V1.OPT_OPEN, 260);
	assert.equal(BYTE_V1.OPT_END, 261);
	assert.equal(BYTE_V1.VOCAB, 262);
});

test("advice cases", async () => {
  for (const c of await vec("advice_cases.json")) {
    if (c.questions) continue; // set-request fixtures carry no advice
    const a = adviceFromReasoning(c.reasoning, c.context);
    assert.equal(a.abstain, c.expect.abstain, c.name);
    assert.equal(a.model_version, c.reasoning.model_version, c.name);
    assert.equal(a.dataset_manifest_hash, c.reasoning.dataset_manifest_hash, c.name);
    assert.ok(!a.missing_evidence || a.missing_evidence.length === 0, c.name);
    if (c.expect.abstain) {
      assert.ok(a.abstain_reason && a.recommended_action === "", c.name);
      continue;
    }
    assert.equal(a.recommended_action, c.expect.recommended_action, c.name);
    assert.equal(a.next_phase, c.expect.next_phase, c.name);
    assert.ok(Math.abs(a.confidence - c.expect.confidence) < 1e-6, c.name);
    assert.equal(a.policy_bundle_hash, c.expect.policy_bundle_hash, c.name);
  }
});

test("wire round trip", async () => {
  assert.equal(inputWidth(await vec("head_manifest.json")), 39);
  assert.equal(abstainAdvice("x", "v", "m").abstain, true);
  assert.equal(BYTE_V1.EOS, 259);
});

test("client advise abstains on every failure and advises on success", async () => {
  const ok = (body, status = 200) => ({ ok: status < 400, status, json: async () => body });
  const manifest = { version: "v1-aaaaaaaa", config: { max_seq_len: 40, summary_slots: 1 } };
  let tokens;
  const good = new ModelsClient("http://x", async (url, init) => {
    if (url.endsWith("/models")) return ok(manifest);
    tokens = JSON.parse(init.body).tokens;
    return ok({ model_version: "v1-aaaaaaaa", dataset_manifest_hash: "dm", policy_bundle_hash: "p", plan_tokens: [], halting_steps: 1, confidence: 1,
      decisions: [{ name: "action", kind: "choice", options: [], probabilities: [], selected: "ask_operator", value: null, confidence: 0.8, abstain: false }] });
  });
  const a = await good.advise({ current_phase: "develop" });
  assert.equal(a.abstain, false);
  assert.equal(a.recommended_action, "ask_operator");
  assert.equal(tokens.length, 39);
  assert.equal(tokens[0], BYTE_V1.BOS);

  const none = new ModelsClient("http://x", async () => ok({}, 503));
  const n = await none.advise({});
  assert.ok(n.abstain && n.model_version === "none");
  const broken = new ModelsClient("http://x", async (url) => (url.endsWith("/models") ? ok(manifest) : ok({}, 500)));
  assert.ok((await broken.advise({})).abstain);
  const down = new ModelsClient("http://x", async () => { throw new Error("ECONNREFUSED"); });
  assert.ok((await down.advise({})).abstain);
});

test("registry, HEART and HASHER fixtures round-trip through the client", async () => {
  const ok = (body, status = 200) => ({ ok: status < 400, status, json: async () => body });
  const reg = await vec("registry.json");
  const hh = await vec("heart_health.json");
  const hc = await vec("hasher_cases.json");
  assert.equal(HEAD_PREFIX, "/models/head");
  const c = new ModelsClient("http://x", async (url, init) => {
    if (url.endsWith("/models/registry")) return ok(reg);
    if (url.endsWith("/models/health")) return ok({ status: "ok", time: "t", models: 3 });
    if (url.endsWith("/models/heart/health")) return ok(hh.idle);
    if (url.endsWith("/models/heart/generate")) {
      const p = JSON.parse(init.body).prompt;
      return p === "cold" ? ok({ error: "model not trained", ready: false }, 503) : ok({ text: `echo ${p}`, ready: true });
    }
    if (url.endsWith("/models/hasher/validate")) return ok(hc.validate.response);
    if (url.endsWith("/models/hasher/infer")) return ok(hc.infer.response);
    if (url.endsWith("/models/hasher/status")) return ok(hc.status.idle);
    return ok({}, 404);
  });
  const rows = await c.registry();
  assert.deepEqual(rows.map((r) => r.name), ["hasher", "heart", "head"]);
  assert.equal(rows[1].loaded, false);
  assert.equal((await c.health()).models, 3);
  assert.equal((await c.heartHealth()).loaded, false);
  assert.equal(await c.heartGenerate("hi"), "echo hi");
  await assert.rejects(() => c.heartGenerate("cold"), NotReadyError);
  assert.equal((await c.hasherValidate(hc.validate.request)).valid, true);
  assert.equal((await c.hasherInfer("x")).mode, "transformer");
  assert.equal((await c.hasherStatus()).loaded, false);
});

test("hands binding constants mirror the Go SDK (RoleHands / HandsPrefix)", async () => {
  assert.equal(HandsPrefix, "/models/hands");
  assert.equal(RoleHands, "server_tasks");
});

test("hands binding mirrors the Go facade routes and readiness", async () => {
  const ok = (body, status = 200) => ({ ok: status < 400, status, json: async () => body });
  const seen = [];
  const c = new ModelsClient("http://x", async (url, init) => {
    seen.push({ url, method: init?.method ?? "GET" });
    if (url.endsWith("/models/hands/status")) return ok({ name: "hands", role: "server_tasks", ready: false, loaded: false, backend: "software" });
    if (url.endsWith("/models/hands/health")) return ok({ name: "hands", role: "server_tasks", ready: true, loaded: true, backend: "software" });
    if (url.endsWith("/models/hands/models")) return ok({ name: "hands", role: "server_tasks", backend: "software", version: "gorgonite", capabilities: ["generate"], endpoints: ["/status", "/health", "/models", "/weights", "/generate", "/plan", "/act"] });
    if (url.endsWith("/models/hands/weights")) return ok({ error: "HANDS untrained" }, 503);
    if (url.endsWith("/models/hands/generate")) return ok({ text: "did the thing" });
    if (url.endsWith("/models/hands/plan")) return ok({ stage: "hands.plan", frame_hash: "abc", text: "plan" });
    if (url.endsWith("/models/hands/act")) return ok({ stage: "hands.assistant", frame_hash: "def", text: "act" });
    return ok({}, 404);
  });
  const st = await c.handsStatus();
  assert.equal(st.role, "server_tasks");
  assert.equal((await c.handsHealth()).loaded, true);
  assert.deepEqual((await c.handsModels()).endpoints, ["/status", "/health", "/models", "/weights", "/generate", "/plan", "/act"]);
  await assert.rejects(() => c.handsWeights(), NotReadyError);
  assert.equal((await c.handsGenerate("do")).text, "did the thing");
  assert.equal((await c.handsPlan("p")).stage, "hands.plan");
  assert.equal((await c.handsAct("a")).stage, "hands.assistant");
  // The HANDS client must reach only the /models/hands namespace, never the legacy head/llama paths.
  for (const s of seen) assert.ok(s.url.startsWith("http://x/models/hands/"), s.url);
  assert.ok(!seen.some((s) => s.url.includes("/models/head") || s.url.includes("/models/llama")));
});

// Verified mirror (twin of Go TestContractMirrorsCore): the shared frames fixture
// both the Go and TS bindings use must already carry the HANDS stages core emits.
test("hands frame stages exist in the shared fixture", async () => {
  const fx = await vec("frames.json");
  const ids = fx.stages.map((s) => s.id);
  assert.ok(ids.includes("hands.assistant"), "hands.assistant stage present");
  assert.ok(ids.includes("hands.plan"), "hands.plan stage present");
});

test("ontology fixtures use the typed routes and authority headers", async () => {  const sync = await vec("ontology_sync.json");
  const check = await vec("ontology_check.json");
  const ok = (body, status = 200) => ({ ok: status < 400, status, json: async () => body });
  const calls = [];
  const c = new ModelsClient("http://x", async (url, init) => {
    calls.push({ url, init });
    if (url.endsWith("/ontology/sync")) return ok(sync.response);
    if (url.includes("/ontology?tenant=acme")) return ok(check.remove_response);
    if (url.endsWith("/ontology/check")) return ok(check.response);
    return ok({}, 404);
  });
  assert.equal((await c.hasherOntologySync(sync.request)).ontology_version, "ont1:fixture");
  assert.equal((await c.hasherOntologyCheck({ ...check.request, tenant: "acme" })).outcome, "supported");
  assert.equal((await c.hasherOntologyRemove("acme")).status, "removed");
  assert.equal(calls[0].init.headers["X-KNIRV-Privilege"], "operator");
  assert.equal(calls[1].init.headers["X-KNIRV-Tenant"], "acme");
  assert.equal(calls[2].init.method, "DELETE");
  assert.equal(calls[2].init.headers["X-KNIRV-Privilege"], "operator");
  assert.ok(!JSON.stringify(calls[1].init.body).includes("tenant"));
});

test("platform status role lines match the shared vectors (same file as the Go twin)", async () => {
  const cases = await vec("platform_status.json");
  assert.equal(Object.keys(cases).length, 3);
  for (const [name, c] of Object.entries(cases)) {
    const got = roles(c.status).map((r) => ({ ...r, detail: r.detail || undefined }));
    const want = c.roles.map((r) => ({ ...r, detail: r.detail || undefined }));
    assert.deepEqual(got, want, name);
  }
  assert.equal(STATUS_PATH, "/api/models/status");
});

// Role frames: the TS builder must reproduce the Go builder's text and hashes.
import {
  CONSTITUTION, CONSTITUTION_VERSION, buildFrame, buildLite, checkShape, frameHash, approxTokens,
} from "../models/index.js";

test("frames match the shared fixture byte for byte", async () => {
  const fx = await vec("frames.json");
  assert.equal(CONSTITUTION_VERSION, fx.constitution_version);
  assert.equal(CONSTITUTION, fx.constitution);
  assert.ok(fx.cases.length > 10);
  for (const c of fx.cases) {
    const f = buildFrame(fx.stages, fx.tools, c.stage, c.facts);
    assert.equal(f.system, c.system, `${c.stage} system text`);
    assert.equal(await frameHash(f.system), c.frame_hash, `${c.stage} hash`);
    const lite = buildLite(fx.stages, c.stage);
    assert.equal(lite, c.lite, `${c.stage} lite`);
    assert.equal(await frameHash(lite), c.lite_hash, `${c.stage} lite hash`);
    assert.ok(approxTokens(f.system) <= 1200, `${c.stage} fits the budget`);
  }
});

test("wrapper-side shape check accepts frames and rejects edits", async () => {
  const fx = await vec("frames.json");
  const c = fx.cases.find((x) => x.stage === "heart.chat");
  assert.equal(await checkShape(fx.stages, fx.tools, "heart.chat", c.system, c.frame_hash), "ok");
  const edited = c.system.replace("Forbidden:", "Allowed: everything\nForbidden:");
  assert.equal(await checkShape(fx.stages, fx.tools, "heart.chat", edited, await frameHash(edited)), "stale");
  assert.equal(await checkShape(fx.stages, fx.tools, "heart.chat", c.system, "0".repeat(64)), "hash_mismatch");
  assert.equal(await checkShape(fx.stages, fx.tools, "", "", ""), "unframed");
  assert.equal(await checkShape(fx.stages, fx.tools, "ghost", c.system, c.frame_hash), "unknown_stage");
});

test("index status matches the shared fixture", async () => {
  const fx = await vec("index_status.json");
  for (const [name, st] of Object.entries(fx)) {
    assert.equal(typeof st.index_version, "string", name);
    assert.ok(Array.isArray(st.embedder_ids), name);
    assert.ok(["lexical", "hybrid"].includes(st.retrieval), name);
    for (const k of ["segments", "spans", "chunk_words", "searches", "hit_rate"]) assert.equal(typeof st[k], "number", `${name}.${k}`);
  }
  assert.equal(fx.built.tiers.reference, 1100);
});

test("verify fixture satisfies the contract and a broken answer does not", async () => {
  const fx = await vec("verify.json");
  const cases = Object.entries(fx).filter(([, v]) => v && v.response);
  assert.equal(cases.length, 4);
  for (const [name, c] of cases) assert.equal(checkVerify(c.response), null, name);
  const good = fx.inline_supported.response;
  const clone = () => JSON.parse(JSON.stringify(good));
  for (const [what, mutate] of Object.entries({
    gate: (r) => { r.gate = "accepted"; },
    refused: (r) => { r.refused = true; },
    is_accurate: (r) => { r.is_accurate = true; },
    citation_range: (r) => { r.citations = [9]; },
    cited_not_in_evidence: (r) => { r.claims[0].cited = ["sha256:nope"]; },
    verdict_vs_gate: (r) => { r.claims[0].verdict = "NEI"; },
    pending_shape: (r) => { r.status = "pending"; },
    degraded_chargeable: (r) => { r.degraded = true; },
    confidence: (r) => { r.confidence = 0.5; },
  })) {
    const r = clone();
    mutate(r);
    assert.notEqual(checkVerify(r), null, `${what} passed`);
  }
});

test("hasherVerify sends the tenant as a header and not in the body, and rejects an empty request", async () => {
  const fx = await vec("verify.json");
  let seen;
  const client = new ModelsClient("http://models", async (url, init) => {
    seen = { url: String(url), init };
    return new Response(JSON.stringify(fx.inline_supported.response), { status: 200 });
  });
  const out = await client.hasherVerify({ text: "Paris is the capital of France.", delivery: "inline", tenant: "acme" });
  assert.equal(seen.url, "http://models/models/hasher/verify");
  assert.equal(seen.init.headers["X-KNIRV-Tenant"], "acme");
  assert.ok(!("tenant" in JSON.parse(seen.init.body)));
  assert.equal(out.gate, "seen");
  await client.hasherVerifyJob("abc 1", "acme");
  assert.equal(seen.url, "http://models/models/hasher/verify/jobs/abc%201");
  assert.equal(seen.init.method, "GET");
  await assert.rejects(client.hasherVerify({ text: "  " }), /text or claims/);
  const down = new ModelsClient("http://models", async () => new Response("{}", { status: 503 }));
  await assert.rejects(down.hasherVerify({ text: "x" }), NotReadyError);
});

// heartGenerateWithDecision must send `with_decision` on the wire (matching the Rust
// core / Go server contract) and reach decision resolution — not silently degrade to
// plain generation. This is the TS twin of the Go httptest binding check and the Rust
// core round-trip; all three consume the same shared heart_generate.json vector.
test("heartGenerateWithDecision sends with_decision and resolves a decision", async () => {
  const v = await vec("heart_generate.json");
  // The shared vector request uses the snake_case wire field.
  assert.equal(v.request.with_decision, true);
  assert.equal(v.request.prompt, "restart the failing service");

  const ok = (body, status = 200) => ({ ok: status < 400, status, json: async () => body });
  let seen;
  const c = new ModelsClient("http://x", async (url, init) => {
    seen = { url: String(url), body: JSON.parse(init.body) };
    if (url.endsWith("/models/heart/generate")) return ok(v.response);
    return ok({}, 404);
  });
  const out = await c.heartGenerateWithDecision("restart the failing service", true);
  assert.equal(seen.body.with_decision, true, "wire body must carry with_decision");
  assert.ok(!("withDecision" in seen.body), "camelCase alias must not be sent");
  assert.equal(out.text, "restarting service");
  assert.ok(out.decision && Array.isArray(out.decision.decisions), "decision resolved and returned");
  assert.equal(out.abstained, false);

  // Plain generation (resolve=false) must not send with_decision.
  let plain;
  const cp = new ModelsClient("http://x", async (url, init) => {
    plain = JSON.parse(init.body);
    return ok({ text: "ok", ready: true, progress: 1 });
  });
  await cp.heartGenerateWithDecision("hi", false);
  assert.ok(!("with_decision" in plain), "plain generation must omit with_decision");
});

// reasonSets is the TS twin of the Go SDK DecideSets / Rust reason_sets path. It
// must hit /models/heart/decide/sets with a {"questions":[...]} body and decode
// per-candidate probabilities; and a v1 model's "no choice-set heads" rejection
// must reach the caller verbatim (matching the Rust/Go v1 contract, see
// pkg/heart/decision/contract_test.go TestSetRequestRejectedOnV1Model).
test("reasonSets POSTs questions to /models/heart/decide/sets and decodes candidates", async () => {
  const ok = (body, status = 200) => ({ ok: status < 400, status, json: async () => body, text: async () => "" });
  const seen = [];
  const c = new ModelsClient("http://x", async (url, init) => {
    seen.push({ url: String(url), method: init?.method ?? "GET", body: init?.body ? JSON.parse(init.body) : undefined });
    if (url.endsWith("/models/heart/decide/sets")) {
      return ok({
        plan_tokens: [], halting_steps: 1, confidence: 1, model_version: "v1-aaaaaaaa",
        dataset_manifest_hash: "dm", policy_bundle_hash: "p",
        decisions: [{
          name: "route", kind: "choice-set", options: [], probabilities: [], selected: null,
          value: null, confidence: null, abstain: false,
          candidates: [
            { id: "a", text: "route-a", probability: 0.7 },
            { id: "b", text: "route-b", probability: 0.3 },
          ],
        }],
      });
    }
    return ok({}, 404);
  });
  const questions = [
    { name: "route", question: "which route?", state: "develop", choices: [{ id: "a", text: "route-a" }, { id: "b", text: "route-b" }] },
    { name: "approve", choices: [] },
  ];
  const r = await c.reasonSets(questions);
  assert.equal(seen[0].url, "http://x/models/heart/decide/sets");
  assert.equal(seen[0].method, "POST");
  assert.deepEqual(seen[0].body, { questions });
  assert.equal(r.decisions.length, 1);
  assert.deepEqual(r.decisions[0].candidates, [
    { id: "a", text: "route-a", probability: 0.7 },
    { id: "b", text: "route-b", probability: 0.3 },
  ]);
});

test("reasonSets surfaces the v1 'no choice-set heads' rejection verbatim", async () => {
  const err = (msg, status = 400) => ({ ok: false, status, json: async () => ({ error: msg }), text: async () => msg });
  const c = new ModelsClient("http://x", async (url) =>
    url.endsWith("/models/heart/decide/sets") ? err("no choice-set heads") : err("no choice-set heads"));
  await assert.rejects(() => c.reasonSets([{ name: "x", choices: [] }]), /no choice-set heads/);
});

test("errorDetail surfaces a non-JSON (plain-text) error body", async () => {
  // The body is plain text, not JSON: errorDetail must read it once and surface it
  // verbatim rather than returning an empty fallback (the old json()-then-text() bug).
  const err = (msg, status = 400) => ({ ok: false, status, json: async () => { throw new Error("not json"); }, text: async () => msg });
  const c = new ModelsClient("http://x", async (url) =>
    url.endsWith("/models/heart/decide/sets") ? err("plain text failure reason") : err("plain text failure reason"));
  await assert.rejects(() => c.reasonSets([{ name: "x", choices: [] }]), /plain text failure reason/);
});
