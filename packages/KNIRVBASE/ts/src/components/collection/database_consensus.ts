// Database-level consensus wiring (Go: DistributedEventHandler + the consensus
// block of NewDistributedDatabase). Inbound operations and sync requests are
// routed to the collection named in the message.
import { DistributedCollection } from './distributed_collection';
import { CollectionEventHandler } from './consensus_handler';
import { P2PConsensusManager } from '../p2pconsensus/manager';
import {
  ConsensusConfig, ConsensusManager, EventHandler, OperationEnvelope, PeerInfo, SyncRequest, SyncResponse,
} from '../p2pconsensus/types';

export class DistributedEventHandler implements EventHandler {
  constructor(private collections: () => Map<string, DistributedCollection>) {}

  private collection(name: string): DistributedCollection | undefined {
    return this.collections().get(name);
  }

  async onOperationReceived(op: OperationEnvelope): Promise<void> {
    const c = this.collection(op.collection);
    if (!c) throw new Error(`no collection "${op.collection}" for inbound operation`);
    await new CollectionEventHandler(c).onOperationReceived(op);
  }

  onSyncRequestReceived(req: SyncRequest): SyncResponse {
    const c = this.collection(req.collection);
    if (!c) throw new Error(`no collection "${req.collection}" for inbound sync request`);
    return new CollectionEventHandler(c).onSyncRequestReceived(req);
  }

  async onSyncResponseReceived(resp: SyncResponse): Promise<void> {
    const c = this.collection(resp.collection);
    if (!c) throw new Error(`no collection "${resp.collection}" for inbound sync response`);
    await new CollectionEventHandler(c).onSyncResponseReceived(resp);
  }

  /** A new peer: sync every attached collection; report the first failure. */
  async onPeerDiscovered(peer: PeerInfo): Promise<void> {
    let first: unknown;
    for (const c of this.collections().values()) {
      try {
        await new CollectionEventHandler(c).onPeerDiscovered(peer);
      } catch (e) {
        first ??= e;
      }
    }
    if (first) throw first;
  }
}

export type ConsensusFactory = (cfg: ConsensusConfig, handler: EventHandler) => Promise<ConsensusManager>;

/** Create and start a consensus manager for a database, if the config enables one. */
export async function startDatabaseConsensus(
  cfg: ConsensusConfig,
  collections: () => Map<string, DistributedCollection>,
  factory?: ConsensusFactory
): Promise<ConsensusManager | undefined> {
  if (!cfg.enabled) return undefined;
  const handler = new DistributedEventHandler(collections);
  const manager = factory
    ? await factory(cfg, handler)
    : await P2PConsensusManager.create(cfg, cfg.socket_path, handler);
  await manager.start();
  return manager;
}
