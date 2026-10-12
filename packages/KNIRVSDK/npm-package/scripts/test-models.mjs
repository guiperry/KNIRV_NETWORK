// Contract tests for src/models against the shared fixtures in testvectors/models
// (the Go twin runs the same files: go-package/models).
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import {
  BYTE_V1, HEAD_PREFIX, ModelsClient, NotReadyError, abstainAdvice, adviceFromReasoning, buildAdvisePrompt, encodeByteV1, inputWidth, planText,
} from "../models/index.js";

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

test("advice cases", async () => {
  for (const c of await vec("advice_cases.json")) {
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
  const r = await vec("head_reasoning.json");
  assert.equal(planText(r), "hi");
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
