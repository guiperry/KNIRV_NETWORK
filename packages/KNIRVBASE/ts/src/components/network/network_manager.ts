import * as net from 'net';
import * as crypto from 'crypto';
import {
  NetworkConfig,
  PeerInfo,
  ProtocolMessage,
  MessageType,
  NetworkStats
} from '../types/types';

export type MessageHandler = (msg: ProtocolMessage) => void;

export interface Network {
  initialize(): Promise<void>;
  createNetwork(cfg: NetworkConfig): Promise<string>;
  joinNetwork(networkID: string, bootstrapPeers: string[]): Promise<void>;
  leaveNetwork(networkID: string): Promise<void>;

  addCollectionToNetwork(networkID: string, collectionName: string): Promise<void>;
  removeCollectionFromNetwork(networkID: string, collectionName: string): Promise<void>;
  getNetworkCollections(networkID: string): string[];

  broadcastMessage(networkID: string, msg: ProtocolMessage): Promise<void>;
  sendToPeer(peerID: string, networkID: string, msg: ProtocolMessage): Promise<void>;
  onMessage(mt: MessageType, handler: MessageHandler): void;

  getNetworkStats(networkID: string): NetworkStats | null;
  getNetworks(): NetworkConfig[];
  getPeerID(): string;
  shutdown(): Promise<void>;
}

export class NetworkManager implements Network {
  private peerID: string;
  private networks: Map<string, NetworkConfig> = new Map();
  private peers: Map<string, PeerInfo> = new Map();
  private connections: Map<string, net.Socket> = new Map();
  private stats: Map<string, NetworkStats> = new Map();
  private handlers: Map<MessageType, MessageHandler[]> = new Map();
  private initialized = false;
  private server?: net.Server;

  constructor() {
    this.peerID = crypto.randomBytes(16).toString('hex');
  }

  async initialize(): Promise<void> {
    if (this.initialized) return;

    return new Promise((resolve, reject) => {
      this.server = net.createServer((socket) => {
        this.handleConnection(socket);
      });

      this.server.listen(0, () => {
        this.initialized = true;
        console.log(`Network manager initialized: ${this.peerID}`);
        resolve();
      });

      this.server.on('error', reject);
    });
  }

  async createNetwork(cfg: NetworkConfig): Promise<string> {
    await this.initialize();

    if (this.networks.has(cfg.networkId)) {
      return cfg.networkId;
    }

    cfg.collections = {} as Record<string, boolean>;
    this.networks.set(cfg.networkId, cfg);
    this.stats.set(cfg.networkId, {
      networkId: cfg.networkId,
      connectedPeers: 0,
      totalPeers: 0,
      collectionsShared: 0,
      operationsSent: 0,
      operationsReceived: 0,
      bytesTransferred: 0,
      averageLatency: 0
    });

    console.log(`Created network ${cfg.networkId}`);
    return cfg.networkId;
  }

  async joinNetwork(networkID: string, bootstrapPeers: string[]): Promise<void> {
    await this.initialize();

    if (!this.networks.has(networkID)) {
      const cfg: NetworkConfig = {
        networkId: networkID,
        name: `Network ${networkID}`,
        collections: {},
        bootstrapPeers: [],
        defaultPostingNetwork: '',
        autoPostClassifications: [],
        privateByDefault: true,
        encryption: { enabled: false, sharedSecret: '' },
        replication: { factor: 1, strategy: 'full' },
        discovery: { mdns: false, bootstrap: false }
      };
      this.networks.set(networkID, cfg);
      this.stats.set(networkID, {
        networkId: networkID,
        connectedPeers: 0,
        totalPeers: 0,
        collectionsShared: 0,
        operationsSent: 0,
        operationsReceived: 0,
        bytesTransferred: 0,
        averageLatency: 0
      });
    }

    // Connect to bootstrap peers
    for (const addr of bootstrapPeers) {
      this.connectToPeer(addr);
    }
  }

  async leaveNetwork(networkID: string): Promise<void> {
    this.networks.delete(networkID);
    this.stats.delete(networkID);
    console.log(`Left network ${networkID}`);
  }

  async addCollectionToNetwork(networkID: string, collectionName: string): Promise<void> {
    const netCfg = this.networks.get(networkID);
    if (!netCfg) throw new Error('network not found');

    netCfg.collections[collectionName] = true;
    const st = this.stats.get(networkID);
    if (st) {
      st.collectionsShared = Object.keys(netCfg.collections).length;
    }

    // Announce collection
    await this.broadcastMessage(networkID, {
      type: MessageType.CollectionAnnounce,
      networkId: networkID,
      senderId: this.getPeerID(),
      timestamp: Date.now(),
      payload: { collection: collectionName }
    });
  }

  async removeCollectionFromNetwork(networkID: string, collectionName: string): Promise<void> {
    const netCfg = this.networks.get(networkID);
    if (!netCfg) return;
    delete netCfg.collections[collectionName];
    const st = this.stats.get(networkID);
    if (st) {
      st.collectionsShared = Object.keys(netCfg.collections).length;
    }
  }

  getNetworkCollections(networkID: string): string[] {
    const netCfg = this.networks.get(networkID);
    return netCfg ? Object.keys(netCfg.collections) : [];
  }

  async broadcastMessage(networkID: string, msg: ProtocolMessage): Promise<void> {
    if (!this.initialized) throw new Error('not initialized');

    const data = JSON.stringify(msg);
    const conns = Array.from(this.connections.values());
    const st = this.stats.get(networkID);

    for (const conn of conns) {
      conn.write(data + '\n');
      if (st) {
        st.operationsSent++;
        st.bytesTransferred += data.length;
      }
    }
  }

  async sendToPeer(peerID: string, networkID: string, msg: ProtocolMessage): Promise<void> {
    if (!this.initialized) throw new Error('not initialized');

    const conn = this.connections.get(peerID);
    if (!conn) throw new Error('peer not connected');

    const data = JSON.stringify(msg);
    conn.write(data + '\n');

    const st = this.stats.get(networkID);
    if (st) {
      st.operationsSent++;
      st.bytesTransferred += data.length;
    }
  }

  onMessage(mt: MessageType, handler: MessageHandler): void {
    const handlers = this.handlers.get(mt) || [];
    handlers.push(handler);
    this.handlers.set(mt, handlers);
  }

  getNetworkStats(networkID: string): NetworkStats | null {
    const st = this.stats.get(networkID);
    if (st) {
      st.connectedPeers = this.connections.size;
    }
    return st || null;
  }

  getNetworks(): NetworkConfig[] {
    return Array.from(this.networks.values());
  }

  getPeerID(): string {
    return this.peerID;
  }

  async shutdown(): Promise<void> {
    if (this.server) {
      this.server.close();
    }
    for (const conn of this.connections.values()) {
      conn.destroy();
    }
    this.connections.clear();
    this.initialized = false;
  }

  /** TCP port the legacy transport listens on (0 before initialize()). */
  listenPort(): number {
    const addr = this.server?.address();
    return addr && typeof addr === 'object' ? addr.port : 0;
  }

  /**
   * Newline-delimited framing. TCP delivers arbitrary chunks, so a line may
   * arrive in pieces or several lines in one chunk; only complete lines are
   * dispatched (Go uses bufio.Scanner for the same reason).
   */
  private onLines(socket: net.Socket, onLine: (line: string) => void): void {
    let buf = '';
    socket.setEncoding('utf8');
    socket.on('data', (chunk: string) => {
      buf += chunk;
      let nl: number;
      while ((nl = buf.indexOf('\n')) >= 0) {
        const line = buf.slice(0, nl).trim();
        buf = buf.slice(nl + 1);
        if (line !== '') onLine(line);
      }
    });
  }

  private registerPeer(peerID: string, socket: net.Socket, addr: string): void {
    this.connections.set(peerID, socket);
    this.peers.set(peerID, {
      peerId: peerID,
      addrs: [addr],
      protocols: [],
      latency: 0,
      lastSeen: new Date(),
      collections: [],
    });
    socket.on('close', () => {
      if (this.connections.get(peerID) === socket) this.connections.delete(peerID);
    });
  }

  private dispatchLine(line: string): void {
    try {
      this.handleMessage(JSON.parse(line) as ProtocolMessage);
    } catch {
      // not a protocol message
    }
  }

  // Handshake (shared with Go's legacy_tcp transport): each side sends "KNIRV:<peerID>\n".
  private handleConnection(socket: net.Socket): void {
    let shaken = false;
    socket.on('error', () => undefined);
    this.onLines(socket, (line) => {
      if (!shaken) {
        const parts = line.split(':');
        if (parts.length !== 2 || parts[0] !== 'KNIRV') {
          socket.destroy();
          return;
        }
        shaken = true;
        socket.write(`KNIRV:${this.peerID}\n`);
        this.registerPeer(parts[1], socket, `${socket.remoteAddress}:${socket.remotePort}`);
        return;
      }
      this.dispatchLine(line);
    });
  }

  private connectToPeer(address: string): void {
    const idx = address.lastIndexOf(':');
    const host = address.slice(0, idx);
    const port = parseInt(address.slice(idx + 1), 10);
    const socket = net.createConnection({ host, port }, () => {
      socket.write(`KNIRV:${this.peerID}\n`);
    });
    socket.on('error', () => undefined);
    let shaken = false;
    this.onLines(socket, (line) => {
      if (!shaken) {
        const parts = line.split(':');
        if (parts.length !== 2 || parts[0] !== 'KNIRV') {
          socket.destroy();
          return;
        }
        shaken = true;
        this.registerPeer(parts[1], socket, address);
        return;
      }
      this.dispatchLine(line);
    });
  }

  private handleMessage(msg: ProtocolMessage): void {
    const handlers = this.handlers.get(msg.type) || [];
    for (const h of handlers) {
      h(msg);
    }
  }
}