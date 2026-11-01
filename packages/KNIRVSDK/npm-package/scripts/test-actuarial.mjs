// The actuarial client goes through the Rust core: the endpoint table is checked
// by the recorded requests the core sends to the caller's fetch.
import assert from "node:assert/strict";
import { ActuarialClient } from "@knirv/sdk/actuarial";

const seen = [];
const fetcher = async (url, init) => {
  seen.push({ url: String(url), method: init?.method ?? "GET", headers: Object.fromEntries(new Headers(init?.headers)), body: init?.body ? Buffer.from(init.body).toString() : "" });
  if (String(url).endsWith("/pools/missing")) return new Response(JSON.stringify({ error: "no such pool" }), { status: 404 });
  return new Response(JSON.stringify({ ok: true, url: String(url) }), { status: 200 });
};
const client = new ActuarialClient("https://backend.test", fetcher).setHeaders({ authorization: "Bearer t" });

await client.pools();
await client.pool("a b/c");
await client.submissions({ status: "open" });
await client.exposure("w1");
await client.createStake("p1", { amount: 5 });
await client.claimSubmission("s1", { resolver_wallet: "w" });

assert.deepEqual(seen.map((r) => `${r.method} ${r.url}`), [
  "GET https://backend.test/api/v1/actuarial/pools",
  "GET https://backend.test/api/v1/actuarial/pools/a%20b%2Fc",
  "GET https://backend.test/api/v1/actuarial/submissions?status=open",
  "GET https://backend.test/api/v1/actuarial/me/exposure?wallet=w1",
  "POST https://backend.test/api/v1/actuarial/pools/p1/stakes",
  "POST https://backend.test/api/v1/actuarial/submissions/s1/claim",
]);
assert.equal(seen[0].headers.authorization, "Bearer t");
assert.equal(seen[4].body, '{"amount":5}');
assert.equal(seen[4].headers["content-type"], "application/json");

await assert.rejects(client.pool("missing"), /no such pool/);
console.log(`actuarial: ${seen.length + 1} requests through the core`);
