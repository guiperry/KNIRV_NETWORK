import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';
import { DB, DistributedDatabase } from '../node';
import { FileStorage } from '../../components/storage/storage';
import { defaultConsensusConfig, EventHandler, OperationEnvelope } from '../../components/p2pconsensus';
import { EntryType } from '../../components/types/types';

function fakeManager() {
  const published: OperationEnvelope[] = [];
  const state = { started: false, stopped: false, handler: undefined as EventHandler | undefined };
  const manager: any = {
    start: async () => { state.started = true; },
    stop: async () => { state.stopped = true; },
    enabled: () => true,
    publishOperation: async (op: OperationEnvelope) => { published.push(op); },
  };
  const factory = async (_cfg: unknown, handler: EventHandler) => { state.handler = handler; return manager; };
  return { published, state, factory };
}

describe('DB consensus lifecycle (Go: Options.Consensus)', () => {
  let dir: string;
  const open: Array<{ shutdown(): Promise<void> }> = [];
  beforeEach(() => { dir = fs.mkdtempSync(path.join(os.tmpdir(), 'dbc-')); });
  afterEach(async () => {
    // Always release sockets, even when an assertion failed, so jest can exit.
    await Promise.all(open.splice(0).map((d) => d.shutdown().catch(() => undefined)));
    fs.rmSync(dir, { recursive: true, force: true });
  });

  it('is off unless explicitly enabled', async () => {
    const db = new DB({ dataDir: dir, distributedEnabled: false });
    open.push(db);
    expect(db.getConsensusConfig().enabled).toBe(false);
    await db.initialize();
    expect(db.getConsensusManager()).toBeUndefined();
    await db.shutdown();
  });

  it('starts the manager, routes outbound ops through it and inbound ops to the named collection', async () => {
    const f = fakeManager();
    const db = new DB({
      dataDir: dir, distributedEnabled: false,
      consensus: { ...defaultConsensusConfig(), enabled: true, network_id: 'net-1' },
      consensusFactory: f.factory,
    });
    open.push(db);
    const early = db.collection('early'); // created before initialize()
    await db.initialize();
    expect(f.state.started).toBe(true);
    await db.createNetwork({ networkId: 'net-1', name: 'n', collections: {}, bootstrapPeers: [] } as any);

    const late = db.collection('late'); // created after
    await early.attachToNetwork('net-1');
    await late.attachToNetwork('net-1');
    await early.insert({ id: 'e1', entryType: EntryType.Auth, payload: { v: 1 } });
    await late.insert({ id: 'l1', entryType: EntryType.Auth, payload: { v: 2 } });
    expect(f.published.map((p) => [p.collection, p.document_id])).toEqual([['early', 'e1'], ['late', 'l1']]);

    // Inbound: an op for "late" lands in "late" only; an unknown collection is an error.
    const op = {
      id: 'o', type: 0, collection: 'late', documentId: 'l2', vector: { x: 1 }, timestamp: 1, peerId: 'x',
      data: { id: 'l2', entryType: EntryType.Auth, payload: { id: 'l2', entryType: EntryType.Auth, v: 3 }, _vector: { x: 1 }, _timestamp: 1, _peerId: 'x' },
    };
    const env = (collection: string): OperationEnvelope => ({
      network_id: 'net-1', collection, document_id: 'l2', peer_id: 'x', timestamp: 1, vector_clock: { x: 1 },
      data: Buffer.from(JSON.stringify(op)).toString('base64'),
    });
    await f.state.handler!.onOperationReceived(env('late'));
    expect(await late.find('l2')).toMatchObject({ v: 3 });
    expect(await early.find('l2')).toBeNull();
    await expect(f.state.handler!.onOperationReceived(env('nope'))).rejects.toThrow('no collection "nope"');

    await db.shutdown();
    expect(f.state.stopped).toBe(true);
  });

  it('DB mirrors the Go network facade (add/remove collection, stats)', async () => {
    const db = new DB({ dataDir: dir, distributedEnabled: false });
    open.push(db);
    await db.initialize();
    expect(db.networkStats('n')).toBeNull();
    await db.createNetwork({ networkId: 'n', name: 'n', collections: {}, bootstrapPeers: [] } as any);
    await db.addCollectionToNetwork('n', 'docs');
    expect(db.networkStats('n')).not.toBeNull();
    await db.removeCollectionFromNetwork('docs');
    await db.removeCollectionFromNetwork('never-created'); // no-op, like Go's detach of an unknown collection
  });

  it('DistributedDatabase.ready() starts consensus and reports startup failures', async () => {
    const f = fakeManager();
    const ddb = new DistributedDatabase(null, {
      distributed: { enabled: false },
      consensus: { ...defaultConsensusConfig(), enabled: true },
      consensusFactory: f.factory,
    }, new FileStorage(dir));
    open.push(ddb);
    await ddb.ready();
    expect(ddb.getConsensusManager()).toBeDefined();
    expect(f.state.started).toBe(true);
    await ddb.shutdown();

    const bad = new DistributedDatabase(null, {
      distributed: { enabled: false },
      consensus: { ...defaultConsensusConfig(), enabled: true },
      consensusFactory: async () => { throw new Error('boom'); },
    }, new FileStorage(dir));
    await expect(bad.ready()).rejects.toThrow('boom');
  });
});
