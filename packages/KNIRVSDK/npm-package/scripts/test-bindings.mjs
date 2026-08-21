import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { SdkClient } from "@knirv/sdk";
import {
  signMessageEnvelope,
  verifyMessage,
  verifyMessagePayload,
} from "@knirv/sdk/signing";

const fixturesURL = new URL("../../bindings-tests/envelopes.json", import.meta.url);
const fixtures = JSON.parse(await readFile(fixturesURL, "utf8"));
assert.equal(fixtures.version, 1, "unsupported binding fixture version");

const sdk = await SdkClient.init();
for (const fixture of fixtures.requests) {
  const response = await sdk.call(fixture.operation, fixture.payload);
  assert.equal(response.version, fixtures.version);
  assert.equal(response.operation, fixture.operation);
  assert.equal(response.error, undefined, `${fixture.operation} failed`);
  if (fixture.response) assert.deepEqual(response.payload, fixture.response, fixture.operation);
  if (fixture.operation === "wasm.manifest") {
    assert.ok(Array.isArray(response.payload.modules));
    assert.equal(response.payload.modules.length, 4);
  }
}

const now = Math.floor(Date.now() / 1000);
const envelope = {
  domain: "knirv.sdk.package-test",
  purpose: "binding-fixture",
  chainId: "knirv-testnet-1",
  nonce: "npm-package-signing",
  issuedAtUnix: now,
  expiresAtUnix: now + 120,
  payload: new TextEncoder().encode("KNIRV"),
};
const signed = await signMessageEnvelope(new Uint8Array(32).fill(7), envelope);
await verifyMessage(signed, envelope.domain, envelope.purpose, envelope.chainId, envelope.nonce, new Date());
await verifyMessagePayload(signed, envelope.domain, envelope.purpose, envelope.chainId, envelope.payload, new Date());

console.log(`validated ${fixtures.requests.length} shared binding fixtures through @knirv/sdk`);
