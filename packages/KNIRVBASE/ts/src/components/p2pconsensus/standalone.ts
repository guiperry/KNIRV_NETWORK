// Direct libp2p consensus used when no gateway is present (Go: standalone.go).
//
// Wire-compatible with the Go node: same gossipsub topic (Topic(net, secret)),
// same direct-stream protocol (/knirvbase/consensus/1.0.0, newline-delimited
// JSON), same Kademlia protocol (/knirvbase/<net>/kad/1.0.0) and the same
// rendezvous namespace (knirvbase/<net>).
//
// The libp2p packages are ESM-only and optional (install them to use
// standalone mode), so they are loaded lazily with a real dynamic import.
import { ConsensusConfig, ConsensusStatus, EventHandler, OperationEnvelope, PeerInfo, SyncRequest, SyncResponse, WireMessage, MsgSyncRequest } from './types';
import { topic as topicName } from './secret';

export const CONSENSUS_PROTOCOL = '/knirvbase/consensus/1.0.0';

// eslint-disable-next-line @typescript-eslint/no-implied-eval
const esmImport = new Function('s', 'return import(s)') as (s: string) => Promise<any>;

async function load(name: string): Promise<any> {
  try {
    return await esmImport(name);
  } catch (e: any) {
    throw new Error(
      `standalone consensus needs the optional package "${name}" (npm i libp2p@2 @libp2p/tcp@10 @chainsafe/libp2p-noise@16 ` +
        `@chainsafe/libp2p-yamux@7 @chainsafe/libp2p-gossipsub@14 @libp2p/identify@3 @libp2p/ping@2 @libp2p/kad-dht@15 @multiformats/multiaddr@12 multiformats@13): ${e.message}`
    );
  }
}

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

export class StandaloneConsensus {
  private node: any;
  private peerMap = new Map<string, PeerInfo>();
  private running = false;
  private abort = new AbortController();
  private topic: string;
  private discoveryTimer?: ReturnType<typeof setTimeout>;
  private mod: { multiaddr: any; CID: any; sha256: any; raw: any } | null = null;

  constructor(private config: ConsensusConfig, private handler: EventHandler | null = null) {
    this.topic = topicName(config.network_id, config.network_secret);
  }

  /** libp2p peer id once started. */
  get peerId(): string | undefined {
    return this.node?.peerId?.toString();
  }

  get listenAddrs(): string[] {
    return this.node ? this.node.getMultiaddrs().map((a: any) => a.toString()) : [];
  }

  async start(): Promise<void> {
    if (this.running) throw new Error('already running');
    const [{ createLibp2p }, { tcp }, { noise }, { yamux }, { gossipsub }, { identify }, { ping }, { kadDHT, passthroughMapper }, ma, cidMod, shaMod, rawMod] =
      await Promise.all([
        load('libp2p'), load('@libp2p/tcp'), load('@chainsafe/libp2p-noise'), load('@chainsafe/libp2p-yamux'),
        load('@chainsafe/libp2p-gossipsub'), load('@libp2p/identify'), load('@libp2p/ping'), load('@libp2p/kad-dht'),
        load('@multiformats/multiaddr'), load('multiformats/cid'), load('multiformats/hashes/sha2'), load('multiformats/codecs/raw'),
      ]);
    this.mod = { multiaddr: ma.multiaddr, CID: cidMod.CID, sha256: shaMod.sha256, raw: rawMod };

    const listen = `/ip4/${this.config.listen_addr || '0.0.0.0'}/tcp/${this.config.port}`;
    this.node = await createLibp2p({
      addresses: { listen: [listen] },
      transports: [tcp()],
      connectionEncrypters: [noise()],
      streamMuxers: [yamux()],
      services: {
        identify: identify(),
        ping: ping(),
        pubsub: gossipsub({ emitSelf: false, allowPublishToZeroTopicPeers: false }),
        dht: kadDHT({
          protocol: `/knirvbase/${this.config.network_id}/kad/1.0.0`,
          clientMode: false,
          peerInfoMapper: passthroughMapper,
        }),
      },
    });
    this.abort = new AbortController();

    await this.node.handle(CONSENSUS_PROTOCOL, (data: any) => void this.handleStream(data.stream ?? data));
    const pubsub = this.node.services.pubsub;
    pubsub.subscribe(this.topic);
    pubsub.addEventListener('message', (evt: any) => void this.onPubsub(evt.detail));
    this.running = true;

    for (const addr of this.config.bootstrap_peers) {
      void this.connect(addr);
    }
    void this.discoveryLoop();
  }

  async stop(): Promise<void> {
    if (!this.running) return;
    this.running = false;
    this.abort.abort();
    if (this.discoveryTimer) clearTimeout(this.discoveryTimer);
    await this.node?.stop();
    this.node = undefined;
  }

  status(): ConsensusStatus {
    return { mode: 'standalone', network_id: this.config.network_id, peer_count: this.peerMap.size, running: this.running };
  }

  peers(): PeerInfo[] {
    return [...this.peerMap.values()];
  }

  /** Subscribers of our topic (peers that will receive a broadcast). */
  topicPeers(): string[] {
    return this.node ? this.node.services.pubsub.getSubscribers(this.topic).map((p: any) => p.toString()) : [];
  }

  async broadcastOperation(op: OperationEnvelope, timeoutMs = 10_000): Promise<void> {
    if (!this.running || !this.node) throw new Error('standalone consensus is not running');
    if (!op.network_id) op = { ...op, network_id: this.config.network_id };
    if (op.network_id !== this.config.network_id) {
      throw new Error(`operation network_id "${op.network_id}" does not match "${this.config.network_id}"`);
    }
    // Like Go: wait until at least one peer subscribes to the topic.
    const deadline = Date.now() + timeoutMs;
    while (this.topicPeers().length === 0) {
      if (Date.now() > deadline) throw new Error('no topic peers');
      await sleep(50);
    }
    await this.node.services.pubsub.publish(this.topic, new TextEncoder().encode(JSON.stringify(op)));
  }

  async requestSync(req: SyncRequest): Promise<void> {
    if (!this.running || !this.node) throw new Error('standalone consensus is not running');
    if (!req.network_id) req = { ...req, network_id: this.config.network_id };
    const data = new TextEncoder().encode(JSON.stringify({ type: MsgSyncRequest, payload: req }) + '\n');
    for (const conn of this.node.getConnections()) {
      try {
        const stream = await conn.newStream(CONSENSUS_PROTOCOL);
        await stream.sink((async function* () { yield data; })());
        const resp = await readJSON(stream.source);
        await stream.close?.();
        if (resp && this.handler?.onSyncResponseReceived) await this.handler.onSyncResponseReceived(resp as SyncResponse);
      } catch {
        // peer does not speak the protocol or went away
      }
    }
  }

  // ---- inbound ---------------------------------------------------------------

  private async handleStream(stream: any): Promise<void> {
    try {
      const msg = (await readJSON(stream.source)) as { type?: string; payload?: SyncRequest } | null;
      if (msg && this.handler && msg.type === MsgSyncRequest && msg.payload) {
        const resp = await this.handler.onSyncRequestReceived(msg.payload);
        if (resp) {
          const bytes = new TextEncoder().encode(JSON.stringify(resp) + '\n');
          await stream.sink((async function* () { yield bytes; })());
        }
      }
    } catch {
      // malformed request: drop the stream
    } finally {
      try { await stream.close?.(); } catch { /* already closed */ }
    }
  }

  private async onPubsub(m: any): Promise<void> {
    if (m.topic !== this.topic) return;
    let parsed: any;
    try {
      parsed = JSON.parse(new TextDecoder().decode(m.data));
    } catch {
      return;
    }
    const from = m.from?.toString?.() as string | undefined;
    if (parsed.network_id === this.config.network_id && parsed.document_id !== undefined) {
      if (from) this.peerMap.set(from, { id: from, last_seen: new Date().toISOString() });
      await this.handler?.onOperationReceived(parsed as OperationEnvelope);
      return;
    }
    const wire = parsed as WireMessage;
    if (wire.type !== MsgSyncRequest || wire.network_id !== this.config.network_id || !this.handler) return;
    const resp = await this.handler.onSyncRequestReceived(wire.payload as SyncRequest);
    if (resp) {
      for (const op of resp.operations) {
        try { await this.broadcastOperation(op); } catch { /* best effort */ }
      }
    }
  }

  // ---- discovery ---------------------------------------------------------------

  private record(id: string, addrs: string[]): void {
    this.peerMap.set(id, { id, addresses: addrs, last_seen: new Date().toISOString() });
  }

  private async connect(addr: string): Promise<void> {
    try {
      const conn = await this.node.dial(this.mod!.multiaddr(addr), { signal: this.abort.signal });
      this.record(conn.remotePeer.toString(), [conn.remoteAddr.toString()]);
    } catch {
      // unreachable bootstrap peer; discovery will retry through the DHT
    }
  }

  /** CID of the rendezvous namespace, as go-libp2p's RoutingDiscovery computes it. */
  private async nsCid(): Promise<any> {
    const { CID, sha256, raw } = this.mod!;
    const digest = await sha256.digest(new TextEncoder().encode(`knirvbase/${this.config.network_id}`));
    return CID.createV1(raw.code, digest);
  }

  private async discoveryLoop(): Promise<void> {
    const cid = await this.nsCid();
    while (this.running) {
      try {
        // Advertise can fail until the routing table has peers.
        await this.node.contentRouting.provide(cid, { signal: this.abort.signal }).catch(() => undefined);
        for await (const p of this.node.contentRouting.findProviders(cid, { signal: this.abort.signal })) {
          if (p.id.equals(this.node.peerId)) continue;
          try {
            await this.node.dial(p.id, { signal: this.abort.signal });
            const addrs = p.multiaddrs.map((a: any) => a.toString());
            this.record(p.id.toString(), addrs);
            await this.handler?.onPeerDiscovered({ id: p.id.toString(), addresses: addrs, last_seen: new Date().toISOString() });
          } catch { /* skip unreachable provider */ }
        }
      } catch { /* aborted or no providers yet */ }
      if (!this.running) return;
      await new Promise<void>((r) => { this.discoveryTimer = setTimeout(r, 30_000); });
    }
  }
}

/** Read bytes from a stream source until they parse as one JSON value (or the stream ends). */
async function readJSON(source: AsyncIterable<any>): Promise<unknown | null> {
  let buf = '';
  const dec = new TextDecoder();
  for await (const chunk of source) {
    buf += dec.decode(chunk.subarray ? chunk.subarray() : chunk, { stream: true });
    try {
      return JSON.parse(buf);
    } catch {
      // need more bytes
    }
  }
  try {
    return JSON.parse(buf);
  } catch {
    return null;
  }
}
