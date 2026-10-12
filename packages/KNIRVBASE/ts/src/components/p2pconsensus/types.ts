// Wire types for P2P consensus. Field names are Go's JSON tags
// (internal/p2pconsensus/types.go, interfaces.go, config.go) so the two
// implementations exchange messages unchanged.

export interface PeerInfo {
  id: string;
  addresses?: string[];
  latency?: string;
  last_seen?: string; // RFC 3339
}

export interface SyncRequest {
  network_id: string;
  collection: string;
  vector_clock: Record<string, number>;
  max_entries?: number;
}

export interface OperationEnvelope {
  network_id: string;
  collection: string;
  document_id: string;
  /** JSON-encoded CRDTOperation, base64 on the wire (Go []byte). */
  data: string;
  vector_clock: Record<string, number>;
  timestamp: number;
  peer_id: string;
}

export interface SyncResponse {
  network_id: string;
  collection: string;
  operations: OperationEnvelope[];
  more_available: boolean;
}

export interface WireMessage {
  type: string;
  network_id: string;
  payload: unknown;
}

export const MsgOperation = 'operation';
export const MsgSyncRequest = 'sync_request';
export const MsgSyncResponse = 'sync_response';
export const MsgHeartbeat = 'heartbeat';
export const MsgCollectionAnnounce = 'collection_announce';

export type ConsensusMode = 'auto' | 'gateway' | 'standalone' | 'disabled';

export interface ConsensusConfig {
  enabled: boolean;
  network_id: string;
  /** Unix socket for gateway callbacks (gateway mode only). */
  socket_path: string;
  /**
   * Pre-shared key. Derived into the service ID, pubsub topic and the HMAC that
   * authenticates gateway-sourced messages. Empty means an open network.
   */
  network_secret: string;
  mode: ConsensusMode;
  gateway_url: string;
  gateway_timeout: string; // Go duration, e.g. "2s"
  listen_addr: string;
  port: number;
  bootstrap_peers: string[];
}

export function defaultConsensusConfig(): ConsensusConfig {
  return {
    enabled: true,
    network_id: 'knirvbase-default',
    socket_path: '',
    network_secret: '',
    mode: 'auto',
    gateway_url: 'http://localhost:8080',
    gateway_timeout: '2s',
    listen_addr: '0.0.0.0',
    port: 4002,
    bootstrap_peers: [],
  };
}

export interface ConsensusStatus {
  mode: 'gateway' | 'standalone' | 'disabled';
  network_id?: string;
  peer_count?: number;
  running: boolean;
}

/** Processes inbound consensus events (Go: EventHandler + SyncResponseHandler). */
export interface EventHandler {
  onOperationReceived(op: OperationEnvelope): Promise<void> | void;
  onSyncRequestReceived(req: SyncRequest): Promise<SyncResponse | null> | SyncResponse | null;
  onPeerDiscovered(peer: PeerInfo): Promise<void> | void;
  onSyncResponseReceived?(resp: SyncResponse): Promise<void> | void;
}

export interface ConsensusManager {
  start(): Promise<void>;
  stop(): Promise<void>;
  status(): ConsensusStatus;
  peers(): Promise<PeerInfo[]>;
  enabled(): boolean;
  setEnabled(enabled: boolean): void;
  publishOperation(op: OperationEnvelope): Promise<void>;
  requestSync?(req: SyncRequest): Promise<void>;
}

/** Parse a Go duration such as "2s", "500ms", "1m30s" into milliseconds (NaN if invalid). */
export function parseGoDuration(s: string): number {
  if (!s) return NaN;
  const re = /(\d+(?:\.\d+)?)(ns|us|µs|ms|s|m|h)/g;
  const unit: Record<string, number> = { ns: 1e-6, us: 1e-3, 'µs': 1e-3, ms: 1, s: 1000, m: 60_000, h: 3_600_000 };
  let total = 0;
  let consumed = 0;
  for (const m of s.matchAll(re)) {
    total += parseFloat(m[1]) * unit[m[2]];
    consumed += m[0].length;
  }
  return consumed === s.length && consumed > 0 ? total : NaN;
}
