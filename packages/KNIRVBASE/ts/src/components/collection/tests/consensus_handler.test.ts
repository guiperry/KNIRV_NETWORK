import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';
import { DistributedCollection } from '../distributed_collection';
import { CollectionEventHandler } from '../consensus_handler';
import { FileStorage } from '../../storage/storage';
import { Network } from '../../network/network_manager';
import { EntryType } from '../../types/types';

function stubNetwork(peerID: string): Network {
  return {
    initialize: async () => undefined, createNetwork: async () => 'n', joinNetwork: async () => undefined,
    leaveNetwork: async () => undefined, addCollectionToNetwork: async () => undefined,
    removeCollectionFromNetwork: async () => undefined, getNetworkCollections: () => [],
    broadcastMessage: async () => undefined, sendToPeer: async () => undefined, onMessage: () => undefined,
    getNetworkStats: () => null, getNetworks: () => [], getPeerID: () => peerID, shutdown: async () => undefined,
  } as Network;
}

describe('CollectionEventHandler', () => {
  let dirs: string[] = [];
  const make = async (peer: string) => {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'ceh-'));
    dirs.push(dir);
    const c = new DistributedCollection('docs', stubNetwork(peer), new FileStorage(dir));
    await c.attachToNetwork('net-1');
    return c;
  };
  afterEach(() => { dirs.forEach((d) => fs.rmSync(d, { recursive: true, force: true })); dirs = []; });

  it('replicates an insert from A to B through sync request/response envelopes', async () => {
    const a = await make('peer-a');
    const b = await make('peer-b');
    await a.insert({ id: 'd1', entryType: EntryType.Auth, payload: { key: 'k', value: 'v' } });

    const ha = new CollectionEventHandler(a);
    const hb = new CollectionEventHandler(b);

    const resp = ha.onSyncRequestReceived({ network_id: 'net-1', collection: 'docs', vector_clock: {} });
    expect(resp.operations).toHaveLength(1);
    expect(resp.operations[0]).toMatchObject({ collection: 'docs', document_id: 'd1', peer_id: 'peer-a' });
    // Wire format: Data is base64 of the JSON CRDTOperation (Go []byte).
    expect(JSON.parse(Buffer.from(resp.operations[0].data, 'base64').toString()).documentId).toBe('d1');

    await hb.onSyncResponseReceived(resp);
    expect((await b.find('d1'))?.payload).toEqual({ key: 'k', value: 'v' });
    // The first operation of a peer must be syncable (regression: vector was stamped before the increment).
    expect(resp.operations[0].vector_clock).toEqual({ 'peer-a': 1 });
  });

  it('a requester that is up to date gets nothing', async () => {
    const a = await make('peer-a');
    await a.insert({ id: 'd1', entryType: EntryType.Auth, payload: { x: 1 } });
    const resp = new CollectionEventHandler(a).onSyncRequestReceived({
      network_id: 'net-1', collection: 'docs', vector_clock: { 'peer-a': 99 },
    });
    expect(resp.operations).toEqual([]);
  });

  it('fills missing operation fields from the envelope (Go parity)', async () => {
    const b = await make('peer-b');
    const op = { id: 'o1', type: 0, data: { id: 'd9', entryType: EntryType.Auth, payload: { id: 'd9', entryType: EntryType.Auth, z: 1 }, _vector: { x: 1 }, _timestamp: 5, _peerId: 'x' } };
    await new CollectionEventHandler(b).onOperationReceived({
      network_id: 'net-1', collection: 'docs', document_id: 'd9', peer_id: 'x', timestamp: 5, vector_clock: { x: 1 },
      data: Buffer.from(JSON.stringify(op)).toString('base64'),
    });
    // DistributedDocument.payload carries the whole document (see ToDistributed).
    expect((await b.find('d9'))?.z).toBe(1);
  });
});

describe('DistributedCollection -> consensus', () => {
  it('publishes inserts through the consensus manager as Go-shaped envelopes', async () => {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'ceh-pub-'));
    const published: any[] = [];
    const c = new DistributedCollection('docs', stubNetwork('peer-a'), new FileStorage(dir));
    c.setConsensus({
      enabled: () => true, publishOperation: async (e: any) => { published.push(e); },
    } as any);
    await c.attachToNetwork('net-1');
    await c.insert({ id: 'd1', entryType: EntryType.Auth, payload: { a: 1 } });
    expect(published).toHaveLength(1);
    expect(published[0]).toMatchObject({ network_id: 'net-1', collection: 'docs', document_id: 'd1', peer_id: 'peer-a', vector_clock: { 'peer-a': 1 } });
    expect(JSON.parse(Buffer.from(published[0].data, 'base64').toString()).type).toBe(0);
    fs.rmSync(dir, { recursive: true, force: true });
  });
});
