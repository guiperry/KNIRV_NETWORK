export interface NonceStore {
    /** Returns true if `nonce` was not seen before and is now consumed; false on replay. */
    consume(nonce: string): Promise<boolean>;
}
/** In-process (single instance) store. Safe for a single Node process. */
export declare class MemoryNonceStore implements NonceStore {
    private readonly seen;
    consume(nonce: string): Promise<boolean>;
}
/**
 * File-backed store with an advisory O_EXCL lock file. `fs.openSync(path, "wx")` atomically
 * creates the lock iff it does not already exist (O_EXCL), so exactly one caller — across
 * processes AND across async tasks — ever holds it; a stale lock (older than `staleMs`,
 * e.g. from a crashed holder) is removed before retrying. The consumed set is written to a
 * temp file and atomically renamed into place, so a crash mid-write never corrupts it.
 */
export declare class FileNonceStore implements NonceStore {
    private readonly path;
    private readonly lockPath;
    private readonly staleMs;
    constructor(path: string);
    consume(nonce: string): Promise<boolean>;
    /** Atomically acquire the O_EXCL lock, removing a stale holder first if needed. */
    private acquireLock;
}
