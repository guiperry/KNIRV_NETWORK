// Multi-process racer for FileNonceStore concurrency tests. Runs a single consume()
// against a shared nonce file and prints WIN (first) or LOSE (replay). Spawned by
// nonceStore.test.ts with args: <nonce-file> <nonce>.
import { writeSync } from "node:fs";
import { FileNonceStore } from "../nonceStore.js";

const [, , file, nonce] = process.argv;
const store = new FileNonceStore(file);
store
  .consume(nonce)
  .then((ok) => {
    // Synchronous output preserves the result when the parent captures stdout.
    writeSync(1, ok ? "WIN" : "LOSE");
    process.exit(0);
  })
  .catch((err) => {
    process.stderr.write(String(err));
    process.exit(1);
  });
