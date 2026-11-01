// @knirv/sdk/wallet against the single protocol file (core/data/wallet_protocol.json).
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import {
  WalletMessageInfo, WalletResponseExecuteType, WalletResponseFailureType, WalletResponseRejectType, WalletResponseSuccessType,
} from "@knirv/sdk/wallet";

const protocol = JSON.parse(await readFile(new URL("../../core/data/wallet_protocol.json", import.meta.url), "utf8"));
const objects = { execute: WalletResponseExecuteType, success: WalletResponseSuccessType, failure: WalletResponseFailureType, reject: WalletResponseRejectType };

let count = 0;
for (const [family, object] of Object.entries(objects)) {
  assert.deepEqual(Object.keys(object), Object.keys(protocol[family]), family);
  for (const [key, [code, message]] of Object.entries(protocol[family])) {
    assert.equal(object[key], key);
    assert.deepEqual({ ...WalletMessageInfo[key] }, { code, message, type: key });
    count++;
  }
}
assert.equal(Object.keys(WalletMessageInfo).length, count);
assert.equal(new Set(Object.values(WalletMessageInfo).map((m) => m.code)).size, count, "codes are unique");
console.log(`wallet protocol: ${count} messages match core/data/wallet_protocol.json`);
