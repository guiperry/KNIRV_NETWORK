// Multi-process racer for FileNonceStore concurrency tests. Runs a single consume()
// against a shared nonce file and prints WIN (first) or LOSE (replay). Spawned by
// nonceStore.test.ts with args: <nonce-file> <nonce>.
import { FileNonceStore } from "../nonceStore.js";

const [, , file, nonce] = process.argv;
const store = new FileNonceStore(file);
store
  .consume(nonce)
  .then((ok) => {
    process.stdout.write(ok ? "WIN" : "LOSE");
    process.exit(0);
  })
  .catch((err) => {
    process.stderr.write(String(err));
    process.exit(1);
  });
