// Adapts a DistributedCollection to the P2P consensus EventHandler
// (Go: internal/collection/consensus_handler.go). Inbound operations are applied
// to local storage; sync requests are answered from the operation log.
import { CRDTOperation } from '../types/types';
import { DistributedCollection } from './distributed_collection';
import { EventHandler, OperationEnvelope, PeerInfo, SyncRequest, SyncResponse } from '../p2pconsensus/types';

export class CollectionEventHandler implements EventHandler {
  constructor(private collection: DistributedCollection) {}

  /** Apply a CRDT operation received from a peer. */
  async onOperationReceived(env: OperationEnvelope): Promise<void> {
    const op = JSON.parse(Buffer.from(env.data, 'base64').toString('utf8')) as CRDTOperation;
    // Fall back to the envelope fields if the operation did not carry them.
    if (!op.documentId) op.documentId = env.document_id;
    if (!op.collection) op.collection = env.collection;
    if (!op.peerId) op.peerId = env.peer_id;
    if (!op.vector || Object.keys(op.vector).length === 0) op.vector = env.vector_clock;
    if (!op.timestamp) op.timestamp = env.timestamp;
    await this.collection.applyRemoteOperation(op);
  }

  /** Answer a peer's sync request from the operation log. */
  onSyncRequestReceived(req: SyncRequest): SyncResponse {
    const operations: OperationEnvelope[] = this.collection.computeSyncResponse(req.vector_clock ?? {}).map((op) => ({
      network_id: '',
      collection: op.collection,
      document_id: op.documentId,
      data: Buffer.from(JSON.stringify(op), 'utf8').toString('base64'),
      vector_clock: op.vector,
      timestamp: op.timestamp,
      peer_id: op.peerId,
    }));
    return { network_id: req.network_id, collection: req.collection, operations, more_available: false };
  }

  async onSyncResponseReceived(resp: SyncResponse): Promise<void> {
    for (const op of resp.operations) {
      await this.onOperationReceived(op);
    }
  }

  /** A new peer: proactively sync so this node converges quickly. */
  async onPeerDiscovered(_peer: PeerInfo): Promise<void> {
    if (this.collection.attachedNetworkID() === '') return;
    await this.collection.forceSync();
  }
}
