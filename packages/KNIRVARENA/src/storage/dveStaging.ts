/**
 * Per-DVE staging store for the Arena (plan D7.3 / DVE-U24).
 *
 * Each DVE session stages its own errors in browser-local storage under
 * the namespace `knirv.arena.dve.<dveId>.<errorId>`. Staged errors are
 * visible only to that DVE's session and never reach the main Arena
 * until they are promoted.
 *
 * State machine:
 *
 *   staged ──submit──▶ submitted ──registered──▶ promoted
 *      │                    └───────rejected────▶ rejected
 *      └──discard──▶ (removed)
 *
 * This module performs no network calls. Promotion itself (calling
 * registerNetworkErrorNode) is a separate unit; `markPromoted` only
 * records the server-assigned id once that call has succeeded.
 */

export type DveStagedState = 'staged' | 'submitted' | 'promoted' | 'rejected';

export interface DveStagedRecord {
  dveId: string;
  errorId: string;
  state: DveStagedState;
  payload: Record<string, unknown>;
  networkErrorNodeId?: string;
  rejectReason?: string;
  createdAt: string;
  updatedAt: string;
}

export type DveStagingErrorCode =
  | 'INVALID_DVE_ID'
  | 'INVALID_ERROR_ID'
  | 'INVALID_ARGUMENT'
  | 'DUPLICATE'
  | 'NOT_FOUND'
  | 'ILLEGAL_TRANSITION'
  | 'STORAGE_UNAVAILABLE';

export class DveStagingError extends Error {
  readonly code: DveStagingErrorCode;

  constructor(code: DveStagingErrorCode, message: string) {
    super(message);
    this.name = 'DveStagingError';
    this.code = code;
  }
}

/** Minimal Storage surface, so tests can inject an in-memory backend. */
export interface DveStagingStorage {
  readonly length: number;
  key(index: number): string | null;
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
  removeItem(key: string): void;
}

export const DVE_STAGING_NAMESPACE = 'knirv.arena.dve.';

// dveId / errorId may not contain '.', so the namespace prefix
// `knirv.arena.dve.<dveId>.` can never match another DVE's keys.
const ID_PATTERN = /^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$/;

const assertId = (value: unknown, kind: 'dveId' | 'errorId'): string => {
  if (typeof value !== 'string' || !ID_PATTERN.test(value)) {
    throw new DveStagingError(
      kind === 'dveId' ? 'INVALID_DVE_ID' : 'INVALID_ERROR_ID',
      `${kind} must match ${ID_PATTERN.source}`
    );
  }
  return value;
};

const namespaceFor = (dveId: string): string => `${DVE_STAGING_NAMESPACE}${dveId}.`;

const keyFor = (dveId: string, errorId: string): string => `${namespaceFor(dveId)}${errorId}`;

const defaultStorage = (): DveStagingStorage | null => {
  try {
    return typeof localStorage === 'undefined' ? null : localStorage;
  } catch {
    return null;
  }
};

export interface DveStaging {
  stage(dveId: string, errorId: string, payload: Record<string, unknown>): DveStagedRecord;
  submit(dveId: string, errorId: string): DveStagedRecord;
  markPromoted(dveId: string, errorId: string, networkErrorNodeId: string): DveStagedRecord;
  markRejected(dveId: string, errorId: string, reason?: string): DveStagedRecord;
  discard(dveId: string, errorId: string): void;
  get(dveId: string, errorId: string): DveStagedRecord | null;
  list(dveId: string, state?: DveStagedState): DveStagedRecord[];
}

export const createDveStaging = (
  resolveStorage: () => DveStagingStorage | null = defaultStorage
): DveStaging => {
  const requireStorage = (): DveStagingStorage => {
    const storage = resolveStorage();
    if (!storage) {
      throw new DveStagingError('STORAGE_UNAVAILABLE', 'no storage backend available');
    }
    return storage;
  };

  const readRecord = (storage: DveStagingStorage, key: string): DveStagedRecord | null => {
    try {
      const raw = storage.getItem(key);
      return raw ? (JSON.parse(raw) as DveStagedRecord) : null;
    } catch {
      return null;
    }
  };

  const writeRecord = (storage: DveStagingStorage, record: DveStagedRecord): void => {
    try {
      storage.setItem(keyFor(record.dveId, record.errorId), JSON.stringify(record));
    } catch {
      throw new DveStagingError('STORAGE_UNAVAILABLE', 'failed to persist staged record');
    }
  };

  const loadExisting = (storage: DveStagingStorage, dveId: string, errorId: string): DveStagedRecord => {
    const record = readRecord(storage, keyFor(dveId, errorId));
    if (!record) {
      throw new DveStagingError('NOT_FOUND', `no staged error ${errorId} for dve ${dveId}`);
    }
    return record;
  };

  const illegal = (record: DveStagedRecord, action: string): DveStagingError =>
    new DveStagingError(
      'ILLEGAL_TRANSITION',
      `cannot ${action} error ${record.errorId} in state ${record.state}`
    );

  const transition = (
    dveId: string,
    errorId: string,
    from: DveStagedState,
    action: string,
    apply: (record: DveStagedRecord) => DveStagedRecord
  ): DveStagedRecord => {
    const storage = requireStorage();
    const current = loadExisting(storage, assertId(dveId, 'dveId'), assertId(errorId, 'errorId'));
    if (current.state !== from) {
      throw illegal(current, action);
    }
    const next = apply({ ...current, updatedAt: new Date().toISOString() });
    writeRecord(storage, next);
    return next;
  };

  return {
    stage(dveId, errorId, payload) {
      const d = assertId(dveId, 'dveId');
      const e = assertId(errorId, 'errorId');
      const storage = requireStorage();
      if (readRecord(storage, keyFor(d, e))) {
        throw new DveStagingError('DUPLICATE', `error ${e} is already staged for dve ${d}`);
      }
      const now = new Date().toISOString();
      const record: DveStagedRecord = {
        dveId: d,
        errorId: e,
        state: 'staged',
        payload: { ...payload },
        createdAt: now,
        updatedAt: now
      };
      writeRecord(storage, record);
      return record;
    },

    submit(dveId, errorId) {
      return transition(dveId, errorId, 'staged', 'submit', r => ({ ...r, state: 'submitted' }));
    },

    markPromoted(dveId, errorId, networkErrorNodeId) {
      if (typeof networkErrorNodeId !== 'string' || networkErrorNodeId.trim() === '') {
        throw new DveStagingError('INVALID_ARGUMENT', 'networkErrorNodeId is required to promote');
      }
      return transition(dveId, errorId, 'submitted', 'promote', r => ({
        ...r,
        state: 'promoted',
        networkErrorNodeId
      }));
    },

    markRejected(dveId, errorId, reason) {
      return transition(dveId, errorId, 'submitted', 'reject', r => ({
        ...r,
        state: 'rejected',
        rejectReason: reason ?? ''
      }));
    },

    discard(dveId, errorId) {
      const d = assertId(dveId, 'dveId');
      const e = assertId(errorId, 'errorId');
      const storage = requireStorage();
      const current = loadExisting(storage, d, e);
      if (current.state !== 'staged') {
        throw illegal(current, 'discard');
      }
      try {
        storage.removeItem(keyFor(d, e));
      } catch {
        throw new DveStagingError('STORAGE_UNAVAILABLE', 'failed to remove staged record');
      }
    },

    get(dveId, errorId) {
      const storage = requireStorage();
      return readRecord(storage, keyFor(assertId(dveId, 'dveId'), assertId(errorId, 'errorId')));
    },

    list(dveId, state) {
      const d = assertId(dveId, 'dveId');
      const storage = requireStorage();
      const prefix = namespaceFor(d);
      const records: DveStagedRecord[] = [];
      for (let i = 0; i < storage.length; i += 1) {
        const key = storage.key(i);
        if (!key || !key.startsWith(prefix)) continue;
        const record = readRecord(storage, key);
        if (record && record.dveId === d && (state === undefined || record.state === state)) {
          records.push(record);
        }
      }
      return records.sort((a, b) => a.createdAt.localeCompare(b.createdAt));
    }
  };
};

export const dveStaging: DveStaging = createDveStaging();
