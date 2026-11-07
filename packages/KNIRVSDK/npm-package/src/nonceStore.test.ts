/**
 * Durable, atomic NonceStore tests (DEC-2 / 2d.2 replay prevention, audit item 4).
 *
 * These exercise the host-side store directly — no WASM core required — so they run
 * non-skipped: MemoryNonceStore (in-process) and FileNonceStore (file-backed, O_EXCL
 * lock, persistent across restarts) must honor the same replay contract, and concurrent
 * callers on a FileNonceStore must resolve atomically (exactly one winner) — including
 * across separate OS processes.
 */
import * as fs from "node:fs";
import { fileURLToPath } from "node:url";
import { spawn } from "node:child_process";
import { MemoryNonceStore, FileNonceStore } from "./nonceStore.js";

async function storeContract(
  name: string,
  make: () => { consume: (n: string) => Promise<boolean> },
): Promise<void> {
  const s = make();
  if (!(await s.consume("a"))) throw new Error(`${name}: first consume of a should win`);
  if (await s.consume("a")) throw new Error(`${name}: replay of a must be rejected`);
  if (!(await s.consume("b"))) throw new Error(`${name}: first consume of b should win`);
}

/** Runs the O_EXCL-lock racer child once; resolves with its stdout ("WIN"/"LOSE"). */
function runChild(script: string, file: string, nonce: string): Promise<string> {
  return new Promise((resolve, reject) => {
    const p = spawn("node", [script, file, nonce]);
    let out = "";
    p.stdout.on("data", (d: Uint8Array | string) => (out += d.toString()));
    p.on("close", () => resolve(out.trim()));
    p.on("error", reject);
  });
}

export async function runNonceStoreTests(): Promise<{ passed: boolean; skipped: boolean; reason?: string }> {
  try {
    await storeContract("memory", () => new MemoryNonceStore());

    const dir = fs.mkdtempSync("/tmp/knirv-nonce-");
    try {
      const path = `${dir}/nonces.json`;
      const f1 = new FileNonceStore(path);
      if (!(await f1.consume("enroll:restart"))) throw new Error("first consume must win");
      // New instance, same file: replay must be rejected (durable state).
      const f2 = new FileNonceStore(path);
      if (await f2.consume("enroll:restart")) throw new Error("restarted store must remember consumed nonce");

      await storeContract("file", () => new FileNonceStore(`${fs.mkdtempSync("/tmp/knirv-nonce-")}/n.json`));

      // In-process concurrency: 32 racing consumers of one nonce -> exactly one wins.
      const cdir = fs.mkdtempSync("/tmp/knirv-nonce-");
      const store = new FileNonceStore(`${cdir}/nonces.json`);
      let wins = 0;
      await Promise.all(
        Array.from({ length: 32 }, async () => {
          if (await store.consume("enroll:shared")) wins++;
        }),
      );
      if (wins !== 1) throw new Error(`expected exactly one in-process concurrent consumer to win, got ${wins}`);
      if (await new FileNonceStore(`${cdir}/nonces.json`).consume("enroll:shared")) {
        throw new Error("post-race replay must be rejected");
      }

      // Cross-process atomicity: >=4 OS processes race the SAME nonce/lock file. The
      // O_EXCL lock guarantees exactly one wins; the rest observe a replay.
      const mdir = fs.mkdtempSync("/tmp/knirv-nonce-mp-");
      const mpFile = `${mdir}/nonces.json`;
      const childScript = fileURLToPath(new URL("./scripts/mp_nonce_child.mjs", import.meta.url));
      const results = await Promise.all(
        Array.from({ length: 4 }, () => runChild(childScript, mpFile, "enroll:shared")),
      );
      const mpWins = results.filter((r) => r === "WIN").length;
      if (mpWins !== 1) {
        throw new Error(`expected exactly one cross-process winner, got ${mpWins} (${results.join(",")})`);
      }
      // After the race, a fresh process must be rejected (durable replay prevention).
      const after = await runChild(childScript, mpFile, "enroll:shared");
      if (after !== "LOSE") throw new Error(`post-race cross-process replay must be rejected, got ${after}`);
    } finally {
      fs.rmSync(dir, { recursive: true, force: true });
    }
    return { passed: true, skipped: false };
  } catch (e) {
    return { passed: false, skipped: false, reason: e instanceof Error ? e.message : String(e) };
  }
}
