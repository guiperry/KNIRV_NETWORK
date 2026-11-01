// The five network operations through the host seam, answered from the recorded
// responses in bindings-tests/network.json (the Rust and Go hosts run the same file).
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { SdkClient } from "@knirv/sdk";

const fixtures = JSON.parse(await readFile(new URL("../../bindings-tests/network.json", import.meta.url), "utf8"));

function subset(want, got) {
  if (want && typeof want === "object") return got && typeof got === "object" && Object.entries(want).every(([k, v]) => subset(v, got[k]));
  return want === got;
}

for (const c of fixtures.cases) {
  const http = {
    async http_request(json) {
      const req = JSON.parse(json);
      const answer = c.host[`${req.method} ${req.url}`];
      assert.ok(answer, `${c.name}: unexpected host request ${req.method} ${req.url}`);
      if (answer.error) return JSON.stringify(answer);
      return JSON.stringify({ status: answer.status, headers: {}, body_base64: Buffer.from(JSON.stringify(answer.body)).toString("base64") });
    },
    sleep: async () => {},
  };
  const sdk = await SdkClient.init(undefined, { http });
  const response = await sdk.call(c.request.operation, c.request.payload, c.request.config);
  assert.ok(subset(c.expect, response), `${c.name}: ${JSON.stringify(response)} does not contain ${JSON.stringify(c.expect)}`);
}

// The default host is global fetch: a refused connection is a TRANSPORT error, not a hang.
const sdk = await SdkClient.init(undefined, { config: { network: "local-testnet", gateway_url: "http://127.0.0.1:1", timeout_ms: 2000 } });
const refused = await sdk.call("gateway.health");
assert.equal(refused.error?.code, "TRANSPORT", JSON.stringify(refused));

console.log(`network operations: ${fixtures.cases.length} recorded cases + default fetch host`);
