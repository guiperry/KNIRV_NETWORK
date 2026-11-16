import {
  createDveStaging,
  DveStagingError,
  DVE_STAGING_NAMESPACE,
  type DveStagingStorage
} from '../../../src/storage/dveStaging';

// In-memory Storage double; keeps the test independent of jsdom localStorage.
const memoryStorage = (): DveStagingStorage & { data: Map<string, string> } => {
  const data = new Map<string, string>();
  return {
    data,
    get length() {
      return data.size;
    },
    key: (i: number) => Array.from(data.keys())[i] ?? null,
    getItem: (k: string) => (data.has(k) ? (data.get(k) as string) : null),
    setItem: (k: string, v: string) => {
      data.set(k, v);
    },
    removeItem: (k: string) => {
      data.delete(k);
    }
  };
};

const codeOf = (fn: () => unknown): string | undefined => {
  try {
    fn();
    return undefined;
  } catch (err) {
    return err instanceof DveStagingError ? err.code : 'OTHER';
  }
};

describe('dveStaging', () => {
  let storage: ReturnType<typeof memoryStorage>;
  let staging: ReturnType<typeof createDveStaging>;
  const payload = { title: 'boom', detail: 'x' };

  beforeEach(() => {
    storage = memoryStorage();
    staging = createDveStaging(() => storage);
  });

  describe('stage', () => {
    it('creates a staged record under the dve namespace', () => {
      const rec = staging.stage('dve-a', 'err-1', payload);
      expect(rec.state).toBe('staged');
      expect(rec.dveId).toBe('dve-a');
      expect(rec.errorId).toBe('err-1');
      expect(rec.payload).toEqual(payload);
      expect(storage.data.has(`${DVE_STAGING_NAMESPACE}dve-a.err-1`)).toBe(true);
    });

    it('rejects a duplicate errorId for the same dve', () => {
      staging.stage('dve-a', 'err-1', payload);
      expect(codeOf(() => staging.stage('dve-a', 'err-1', payload))).toBe('DUPLICATE');
    });
  });

  describe('transitions', () => {
    it('staged -> submitted', () => {
      staging.stage('dve-a', 'err-1', payload);
      expect(staging.submit('dve-a', 'err-1').state).toBe('submitted');
      expect(staging.get('dve-a', 'err-1')?.state).toBe('submitted');
    });

    it('submitted -> promoted records networkErrorNodeId', () => {
      staging.stage('dve-a', 'err-1', payload);
      staging.submit('dve-a', 'err-1');
      const rec = staging.markPromoted('dve-a', 'err-1', 'node-42');
      expect(rec.state).toBe('promoted');
      expect(rec.networkErrorNodeId).toBe('node-42');
    });

    it('submitted -> rejected records the reason', () => {
      staging.stage('dve-a', 'err-1', payload);
      staging.submit('dve-a', 'err-1');
      const rec = staging.markRejected('dve-a', 'err-1', 'duplicate of known error');
      expect(rec.state).toBe('rejected');
      expect(rec.rejectReason).toBe('duplicate of known error');
    });

    it('staged -> discard removes the entry', () => {
      staging.stage('dve-a', 'err-1', payload);
      staging.discard('dve-a', 'err-1');
      expect(staging.get('dve-a', 'err-1')).toBeNull();
      expect(storage.data.size).toBe(0);
    });
  });

  describe('illegal transitions', () => {
    it('cannot promote a staged (unsubmitted) error', () => {
      staging.stage('dve-a', 'err-1', payload);
      expect(codeOf(() => staging.markPromoted('dve-a', 'err-1', 'node-1'))).toBe('ILLEGAL_TRANSITION');
    });

    it('cannot reject a staged (unsubmitted) error', () => {
      staging.stage('dve-a', 'err-1', payload);
      expect(codeOf(() => staging.markRejected('dve-a', 'err-1'))).toBe('ILLEGAL_TRANSITION');
    });

    it('cannot submit twice', () => {
      staging.stage('dve-a', 'err-1', payload);
      staging.submit('dve-a', 'err-1');
      expect(codeOf(() => staging.submit('dve-a', 'err-1'))).toBe('ILLEGAL_TRANSITION');
    });

    it('cannot discard a submitted error', () => {
      staging.stage('dve-a', 'err-1', payload);
      staging.submit('dve-a', 'err-1');
      expect(codeOf(() => staging.discard('dve-a', 'err-1'))).toBe('ILLEGAL_TRANSITION');
      expect(staging.get('dve-a', 'err-1')?.state).toBe('submitted');
    });

    it('cannot change a promoted error', () => {
      staging.stage('dve-a', 'err-1', payload);
      staging.submit('dve-a', 'err-1');
      staging.markPromoted('dve-a', 'err-1', 'node-1');
      expect(codeOf(() => staging.markRejected('dve-a', 'err-1'))).toBe('ILLEGAL_TRANSITION');
      expect(codeOf(() => staging.submit('dve-a', 'err-1'))).toBe('ILLEGAL_TRANSITION');
      expect(codeOf(() => staging.discard('dve-a', 'err-1'))).toBe('ILLEGAL_TRANSITION');
      expect(staging.get('dve-a', 'err-1')?.state).toBe('promoted');
    });

    it('cannot change a rejected error', () => {
      staging.stage('dve-a', 'err-1', payload);
      staging.submit('dve-a', 'err-1');
      staging.markRejected('dve-a', 'err-1');
      expect(codeOf(() => staging.markPromoted('dve-a', 'err-1', 'node-1'))).toBe('ILLEGAL_TRANSITION');
      expect(codeOf(() => staging.submit('dve-a', 'err-1'))).toBe('ILLEGAL_TRANSITION');
    });

    it('promote requires a non-empty networkErrorNodeId', () => {
      staging.stage('dve-a', 'err-1', payload);
      staging.submit('dve-a', 'err-1');
      expect(codeOf(() => staging.markPromoted('dve-a', 'err-1', ''))).toBe('INVALID_ARGUMENT');
      expect(codeOf(() => staging.markPromoted('dve-a', 'err-1', '   '))).toBe('INVALID_ARGUMENT');
      expect(staging.get('dve-a', 'err-1')?.state).toBe('submitted');
    });

    it('transitions on a missing error throw NOT_FOUND', () => {
      expect(codeOf(() => staging.submit('dve-a', 'nope'))).toBe('NOT_FOUND');
      expect(codeOf(() => staging.discard('dve-a', 'nope'))).toBe('NOT_FOUND');
    });
  });

  describe('dveId and errorId validation', () => {
    it.each(['', 'has.dot', 'has space', '../escape', '-leading', 'a'.repeat(129)])(
      'rejects dveId %p',
      bad => {
        expect(codeOf(() => staging.stage(bad, 'err-1', payload))).toBe('INVALID_DVE_ID');
        expect(codeOf(() => staging.list(bad))).toBe('INVALID_DVE_ID');
      }
    );

    it.each(['', 'err.1', 'err/1'])('rejects errorId %p', bad => {
      expect(codeOf(() => staging.stage('dve-a', bad, payload))).toBe('INVALID_ERROR_ID');
    });

    it('accepts ids made of letters, digits, underscore and hyphen', () => {
      expect(() => staging.stage('DVE_01-x', 'err_9-z', payload)).not.toThrow();
    });
  });

  describe('namespace isolation', () => {
    it('keys for two dveIds never collide, even with the same errorId', () => {
      staging.stage('dve-a', 'err-1', { owner: 'a' });
      staging.stage('dve-b', 'err-1', { owner: 'b' });
      expect(storage.data.has(`${DVE_STAGING_NAMESPACE}dve-a.err-1`)).toBe(true);
      expect(storage.data.has(`${DVE_STAGING_NAMESPACE}dve-b.err-1`)).toBe(true);
      expect(staging.get('dve-a', 'err-1')?.payload).toEqual({ owner: 'a' });
      expect(staging.get('dve-b', 'err-1')?.payload).toEqual({ owner: 'b' });
    });

    it('list is scoped to one dve', () => {
      staging.stage('dve-a', 'a-1', payload);
      staging.stage('dve-a', 'a-2', payload);
      staging.stage('dve-b', 'b-1', payload);
      expect(staging.list('dve-a').map(r => r.errorId).sort()).toEqual(['a-1', 'a-2']);
      expect(staging.list('dve-b').map(r => r.errorId)).toEqual(['b-1']);
    });

    it('a transition on dve-a does not touch dve-b', () => {
      staging.stage('dve-a', 'err-1', payload);
      staging.stage('dve-b', 'err-1', payload);
      staging.submit('dve-a', 'err-1');
      expect(staging.get('dve-b', 'err-1')?.state).toBe('staged');
    });

    it('discard in dve-a leaves dve-b intact', () => {
      staging.stage('dve-a', 'err-1', payload);
      staging.stage('dve-b', 'err-1', payload);
      staging.discard('dve-a', 'err-1');
      expect(staging.get('dve-a', 'err-1')).toBeNull();
      expect(staging.get('dve-b', 'err-1')?.state).toBe('staged');
    });

    it('list can filter by state', () => {
      staging.stage('dve-a', 'err-1', payload);
      staging.stage('dve-a', 'err-2', payload);
      staging.submit('dve-a', 'err-2');
      expect(staging.list('dve-a', 'staged').map(r => r.errorId)).toEqual(['err-1']);
      expect(staging.list('dve-a', 'submitted').map(r => r.errorId)).toEqual(['err-2']);
    });
  });

  describe('storage and network', () => {
    it('reports STORAGE_UNAVAILABLE when no backend exists', () => {
      const none = createDveStaging(() => null);
      expect(codeOf(() => none.stage('dve-a', 'err-1', payload))).toBe('STORAGE_UNAVAILABLE');
    });

    it('makes no network calls', () => {
      const fetchSpy = jest.fn();
      const original = (globalThis as { fetch?: unknown }).fetch;
      (globalThis as { fetch?: unknown }).fetch = fetchSpy;
      try {
        staging.stage('dve-a', 'err-1', payload);
        staging.submit('dve-a', 'err-1');
        staging.markPromoted('dve-a', 'err-1', 'node-1');
        expect(fetchSpy).not.toHaveBeenCalled();
      } finally {
        (globalThis as { fetch?: unknown }).fetch = original;
      }
    });
  });
});
