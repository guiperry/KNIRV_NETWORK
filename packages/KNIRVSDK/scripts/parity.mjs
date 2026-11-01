#!/usr/bin/env node
// Runs bindings-tests/{parity,network}.json on the TypeScript host and compares
// every response with the Go host's (dumped by internal/core's TestParityDump).
// Usage: parity.mjs <go-dump.json>
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { SdkClient } from "../npm-package/index.js";

const shared = (name) => new URL(`../bindings-tests/${name}`, import.meta.url);
const parity = JSON.parse(await readFile(shared("parity.json"), "utf8"));
const network = JSON.parse(await readFile(shared("network.json"), "utf8"));
const go = JSON.parse(await readFile(process.argv[2], "utf8"));

const ts = {};
const local = await SdkClient.init();
for (const [i, q] of parity.requests.entries()) {
  ts[`local ${i} ${q.operation}`] = await local.call(q.operation, q.payload);
}
for (const c of network.cases) {
  const http = {
    async http_request(json) {
      const req = JSON.parse(json);
      const answer = c.host[`${req.method} ${req.url}`];
      if (answer.error) return JSON.stringify(answer);
      return JSON.stringify({ status: answer.status, headers: {}, body_base64: Buffer.from(JSON.stringify(answer.body)).toString("base64") });
    },
    sleep: async () => {},
  };
  ts[`network ${c.name}`] = await (await SdkClient.init(undefined, { http })).call(c.request.operation, c.request.payload, c.request.config);
}

// A transport failure's message is the host's own wording (Go's net error vs
// fetch's); the code, status and retryable flag are the core's and must match.
const core = (r) => (r?.error?.code === "TRANSPORT" ? { ...r, error: { ...r.error, message: "<host>" } } : r);

// Report every difference rather than the first.
const diffs = [];
for (const key of new Set([...Object.keys(go), ...Object.keys(ts)])) {
  try {
    assert.deepEqual(core(ts[key]), core(go[key]));
  } catch {
    diffs.push(`${key}\n    go: ${JSON.stringify(go[key])}\n    ts: ${JSON.stringify(ts[key])}`);
  }
}
if (diffs.length) {
  console.error(`parity: ${diffs.length} difference(s)\n  ${diffs.join("\n  ")}`);
  process.exit(1);
}
console.log(`parity: ${Object.keys(ts).length} responses identical on Go (WASI) and TypeScript (wasm-bindgen)`);
