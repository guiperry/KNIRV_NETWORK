import { VectorClock, increment, merge } from '../clock/vector_clock';
import { Network } from '../network/network_manager';
import { Storage } from '../storage/storage';
import {
  DistributedDocument,
  CRDTOperation,
  OperationType,
  SyncState,
  ProtocolMessage,
  MessageType,
  EntryType
} from '../types/types';
import { ToDistributed, ToRegular, ApplyOperation } from '../resolver/crdt_resolver';
import { ModalityType } from '../storage/nrv/spec';

export interface Frame {
  id: string;
  vector: Float32Array;
  seed: Uint8Array;
  thermo: {
    tempCelsius: number;
    voltageV: number;
    freqMHz: number;
    fanRPM: number;
  };
  proof: Uint8Array;
}

// A minimal in-memory local collection implementation to keep the example self-contained
export class LocalCollection {
  private name: string;
  private store: Storage;

  constructor(name: string, store: Storage) {
    this.name = name;
    this.store = store;
  }

  async insert(doc: Record<string, any>): Promise<Record<string, any>> {
    const cloned = this.cloneMap(doc);
    await this.store.insert(this.name, cloned);
    return this.cloneMap(cloned);
  }

  async update(id: string, update: Record<string, any>): Promise<number> {
    return await this.store.update(this.name, id, update);
  }

  async delete(id: string): Promise<number> {
    return await this.store.delete(this.name, id);
  }

  async find(id: string): Promise<Record<string, any> | null> {
    return await this.store.find(this.name, id);
  }

  async findAll(): Promise<Record<string, any>[]> {
    return await this.store.findAll(this.name);
  }

  getStore(): Storage {
    return this.store;
  }

  private cloneMap(m: Record<string, any>): Record<string, any> {
    const out: Record<string, any> = {};
    for (const k in m) {
      const v = m[k];
      if (typeof v === 'object' && v !== null && !Array.isArray(v)) {
        out[k] = this.cloneMap(v);
      } else if (Array.isArray(v)) {
        out[k] = this.cloneSlice(v);
      } else {
        out[k] = v;
      }
    }
    return out;
  }

  private cloneSlice(s: any[]): any[] {
    const out: any[] = [];
    for (const e of s) {
      if (typeof e === 'object' && e !== null && !Array.isArray(e)) {
        out.push(this.cloneMap(e));
      } else if (Array.isArray(e)) {
        out.push(this.cloneSlice(e));
      } else {
        out.push(e);
      }
    }
    return out;
  }
}

// DistributedCollection manages local storage plus network synchronization
export class DistributedCollection {
  public name: string;
  private network: Network;
  private networkID: string = '';
  private syncStates: Map<string, SyncState> = new Map();
  private operationLog: CRDTOperation[] = [];
  private maxLogSize: number = 10000;
  private local: LocalCollection;

  constructor(name: string, network: Network, store: Storage) {
    this.name = name;
    this.network = network;
    this.local = new LocalCollection(name, store);
    this.setupMessageHandlers();
  }

  private setupMessageHandlers(): void {
    this.network.onMessage(MessageType.Operation, (msg: ProtocolMessage) => {
      const payload = msg.payload as Record<string, any>;
      if (!payload) return;
      const coll = payload.collection as string;
      if (coll !== this.name) return;
      const opMap = payload.operation as Record<string, any>;
      const op: CRDTOperation = opMap as any;
      this.handleRemoteOperation(op);
    });

    this.network.onMessage(MessageType.SyncRequest, (msg: ProtocolMessage) => {
      const payload = msg.payload as Record<string, any>;
      if (!payload) return;
      const coll = payload.collection as string;
      if (coll !== this.name) return;
      this.handleSyncRequest(msg);
    });

    this.network.onMessage(MessageType.SyncResponse, (msg: ProtocolMessage) => {
      const payload = msg.payload as Record<string, any>;
      if (!payload) return;
      const coll = payload.collection as string;
      if (coll !== this.name) return;
      this.handleSyncResponse(msg);
    });
  }

  async attachToNetwork(networkID: string): Promise<void> {
    if (this.networkID !== '') {
      throw new Error(`collection ${this.name} already attached to ${this.networkID}`);
    }

    await this.network.addCollectionToNetwork(networkID, this.name);

    this.networkID = networkID;
    this.syncStates.set(networkID, {
      collection: this.name,
      networkId: networkID,
      localVector: {},
      lastSync: new Date(),
      pendingOperations: [],
      stagedEntries: [],
      syncInProgress: false
    });

    await this.requestSync();
  }

  async detachFromNetwork(): Promise<void> {
    if (this.networkID === '') return;
    await this.network.removeCollectionFromNetwork(this.networkID, this.name);
    this.syncStates.delete(this.networkID);
    this.networkID = '';
  }

  async insert(doc: Record<string, any>): Promise<Record<string, any>> {
    const id = doc.id as string;
    if (!id) {
      throw new Error("document must contain 'id'");
    }
    const entryType = doc.entryType as EntryType;

    if (entryType === EntryType.Memory) {
      const payload = doc.payload as Record<string, any>;
      if (payload && 'blob' in payload) {
        // Blob will be saved by storage.insert
      }
    }

    const inserted = await this.local.insert(doc);

    if (this.networkID !== '') {
      const opPayload = ToDistributed(inserted, this.network.getPeerID());
      opPayload.entryType = entryType;

      const op: CRDTOperation = {
        id: `${this.network.getPeerID()}-${Date.now()}-${Math.random()}`,
        type: OperationType.Insert,
        collection: this.name,
        documentId: id,
        data: opPayload,
        vector: this.getCurrentVector(),
        timestamp: Date.now(),
        peerId: this.network.getPeerID(),
      };
      this.broadcastOperation(op);
    }

    return inserted;
  }

  async update(id: string, update: Record<string, any>): Promise<number> {
    const affected = await this.local.update(id, update);
    if (this.networkID !== '' && affected > 0) {
      const doc = await this.local.find(id);
      const op: CRDTOperation = {
        id: `${this.network.getPeerID()}-${Date.now()}`,
        type: OperationType.Update,
        collection: this.name,
        documentId: id,
        data: ToDistributed(doc!, this.network.getPeerID()),
        vector: this.getCurrentVector(),
        timestamp: Date.now(),
        peerId: this.network.getPeerID()
      };
      this.broadcastOperation(op);
    }
    return affected;
  }

  async delete(id: string): Promise<number> {
    const affected = await this.local.delete(id);
    if (this.networkID !== '' && affected > 0) {
      const op: CRDTOperation = {
        id: `${this.network.getPeerID()}-${Date.now()}`,
        type: OperationType.Delete,
        collection: this.name,
        documentId: id,
        data: { id, _deleted: true } as any,
        vector: this.getCurrentVector(),
        timestamp: Date.now(),
        peerId: this.network.getPeerID()
      };
      this.broadcastOperation(op);
    }
    return affected;
  }

  async find(id: string): Promise<Record<string, any> | null> {
    return await this.local.find(id);
  }

  async findAll(): Promise<Record<string, any>[]> {
    return await this.local.findAll();
  }

  getSyncState(): SyncState | null {
    if (this.networkID === '') return null;
    return this.syncStates.get(this.networkID) || null;
  }

  async forceSync(): Promise<void> {
    await this.requestSync();
  }

  streamFrames(modality?: ModalityType): AsyncGenerator<Frame, void, unknown> {
    const store = this.local.getStore() as any;
    if (store.streamFrames) {
      return store.streamFrames(this.name, modality);
    } else {
      throw new Error(`collection "${this.name}": storage backend does not support NRV streaming`);
    }
  }

  getStore(): Storage {
    return this.local.getStore();
  }

  getOperationLog(): CRDTOperation[] {
    return [...this.operationLog];
  }

  private broadcastOperation(op: CRDTOperation): void {
    if (this.networkID === '') return;

    this.operationLog.push(op);
    this.pruneOperationLog();
    const syncState = this.syncStates.get(this.networkID)!;
    syncState.localVector = increment(syncState.localVector, this.network.getPeerID());

    this.network.broadcastMessage(this.networkID, {
      type: MessageType.Operation,
      networkId: this.networkID,
      senderId: this.network.getPeerID(),
      timestamp: Date.now(),
      payload: { collection: this.name, operation: op }
    });
  }

  private async handleRemoteOperation(op: CRDTOperation): Promise<void> {
    const existing = await this.local.find(op.documentId);
    let existingDist: DistributedDocument | null = null;
    if (existing) {
      existingDist = ToDistributed(existing, op.peerId);
    }

    const result = ApplyOperation(existingDist, op);

    if (result === null) {
      await this.local.delete(op.documentId);
    } else if (result._deleted) {
      await this.local.delete(op.documentId);
    } else {
      const regular = ToRegular(result);
      if (regular) {
        await this.local.insert(regular);
      }
    }

    if (this.networkID !== '') {
      const syncState = this.syncStates.get(this.networkID)!;
      syncState.localVector = merge(syncState.localVector, op.vector);
    }
  }

  private async requestSync(): Promise<void> {
    if (this.networkID === '') {
      throw new Error('not attached to network');
    }
    const syncState = this.syncStates.get(this.networkID)!;
    if (syncState.syncInProgress) return;
    syncState.syncInProgress = true;

    await this.network.broadcastMessage(this.networkID, {
      type: MessageType.SyncRequest,
      networkId: this.networkID,
      senderId: this.network.getPeerID(),
      timestamp: Date.now(),
      payload: { collection: this.name, vector: syncState.localVector }
    });

    setTimeout(() => {
      syncState.syncInProgress = false;
    }, 10000);
  }

  private handleSyncRequest(msg: ProtocolMessage): void {
    const payload = msg.payload as Record<string, any>;
    const remoteVector = payload.vector as VectorClock;

    const missing: CRDTOperation[] = [];
    for (const op of this.operationLog) {
      const remoteClock = remoteVector[op.peerId] || 0;
      const opClock = op.vector[op.peerId] || 0;
      if (opClock > remoteClock) {
        missing.push(op);
      }
    }

    this.network.sendToPeer(msg.senderId, this.networkID, {
      type: MessageType.SyncResponse,
      networkId: this.networkID,
      senderId: this.network.getPeerID(),
      timestamp: Date.now(),
      payload: { collection: this.name, operations: missing, vector: this.syncStates.get(this.networkID)!.localVector }
    });
  }

  private handleSyncResponse(msg: ProtocolMessage): void {
    const payload = msg.payload as Record<string, any>;
    const opsIface = payload.operations as CRDTOperation[];

    for (const oi of opsIface) {
      this.handleRemoteOperation(oi);
    }

    if (this.networkID !== '') {
      const syncState = this.syncStates.get(this.networkID)!;
      syncState.syncInProgress = false;
      syncState.lastSync = new Date();
    }
  }

  private getCurrentVector(): VectorClock {
    if (this.networkID === '') return {};
    const s = this.syncStates.get(this.networkID);
    return s ? { ...s.localVector } : {};
  }

  private pruneOperationLog(): void {
    if (this.operationLog.length > this.maxLogSize) {
      this.operationLog = this.operationLog.slice(this.operationLog.length - this.maxLogSize);
    }
  }
}