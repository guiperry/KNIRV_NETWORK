/**
 * Host-side NonceStore for bootnode enrollment (DEC-2 / 2d.2 replay prevention).
 *
 * The Rust core no longer takes a caller-supplied `[]string` of consumed nonces. Instead
 * it authenticates the full approval->challenge->root binding, then returns the single
 * nonce it consumed; the host persists that nonce through an atomic store. This removes
 * the previous read-modify-write race where concurrent senders could both "win" and
 * double-spend an enrollment.
 */
import * as fs from "node:fs";
/** In-process (single instance) store. Safe for a single Node process. */
export class MemoryNonceStore {
    seen = new Set();
    async consume(nonce) {
        if (this.seen.has(nonce))
            return false;
        this.seen.add(nonce);
        return true;
    }
}
const delay = (ms) => new Promise((r) => setTimeout(r, ms));
/**
 * File-backed store with an advisory O_EXCL lock file. `fs.openSync(path, "wx")` atomically
 * creates the lock iff it does not already exist (O_EXCL), so exactly one caller — across
 * processes AND across async tasks — ever holds it; a stale lock (older than `staleMs`,
 * e.g. from a crashed holder) is removed before retrying. The consumed set is written to a
 * temp file and atomically renamed into place, so a crash mid-write never corrupts it.
 */
export class FileNonceStore {
    path;
    lockPath;
    staleMs = 10_000;
    constructor(path) {
        this.path = path;
        this.lockPath = `${path}.lock`;
    }
    async consume(nonce) {
        const dir = this.path.slice(0, this.path.lastIndexOf("/"));
        if (dir)
            fs.mkdirSync(dir, { recursive: true });
        await this.acquireLock();
        try {
            let consumed = [];
            if (fs.existsSync(this.path)) {
                try {
                    consumed = JSON.parse(fs.readFileSync(this.path, "utf8"));
                }
                catch {
                    consumed = [];
                }
            }
            if (!Array.isArray(consumed))
                consumed = [];
            if (consumed.includes(nonce))
                return false;
            consumed.push(nonce);
            const tmp = `${this.path}.tmp`;
            fs.writeFileSync(tmp, JSON.stringify(consumed));
            fs.renameSync(tmp, this.path);
            return true;
        }
        finally {
            try {
                fs.unlinkSync(this.lockPath);
            }
            catch {
                /* best-effort release */
            }
        }
    }
    /** Atomically acquire the O_EXCL lock, removing a stale holder first if needed. */
    async acquireLock() {
        for (;;) {
            try {
                const fd = fs.openSync(this.lockPath, "wx"); // fail if already exists
                fs.closeSync(fd);
                return;
            }
            catch (err) {
                const code = err?.code;
                if (code === "EEXIST" || code === "EACCES") {
                    try {
                        const ageMs = Date.now() - fs.statSync(this.lockPath).mtimeMs;
                        if (ageMs > this.staleMs) {
                            fs.unlinkSync(this.lockPath);
                        }
                    }
                    catch {
                        /* lock vanished between stat and unlink; next pass will acquire */
                    }
                    await delay(2);
                    continue;
                }
                throw err;
            }
        }
    }
}
