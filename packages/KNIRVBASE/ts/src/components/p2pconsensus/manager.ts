// Top-level consensus orchestrator (Go: manager.go). Picks gateway or
// standalone mode, fails over when the gateway disappears and moves back when
// it returns.
import {
  ConsensusConfig, ConsensusManager, ConsensusStatus, EventHandler, OperationEnvelope, PeerInfo, SyncRequest, parseGoDuration,
} from './types';
import { GatewayProxy, detectGateway } from './gateway_proxy';
import { SocketServer } from './socket_server';
import { StandaloneConsensus } from './standalone';

/** Resolve "auto" to a concrete mode by probing the gateway. */
export async function resolveMode(cfg: ConsensusConfig): Promise<'gateway' | 'standalone' | 'disabled'> {
  if (!cfg.enabled) return 'disabled';
  switch (cfg.mode) {
    case 'gateway': return 'gateway';
    case 'standalone': return 'standalone';
    case 'disabled': return 'disabled';
    default: {
      let timeout = parseGoDuration(cfg.gateway_timeout);
      if (Number.isNaN(timeout)) timeout = 2000;
      return cfg.gateway_url && (await detectGateway(cfg.gateway_url, timeout)) ? 'gateway' : 'standalone';
    }
  }
}

export interface ManagerOptions {
  /** Health-check period in gateway mode (Go: 5s). */
  monitorIntervalMs?: number;
  /** Consecutive failed checks before failing over to standalone (Go: 3). */
  failuresBeforeFailover?: number;
  /** Build the standalone transport (tests inject a stub; default is libp2p). */
  standaloneFactory?: (cfg: ConsensusConfig, handler: EventHandler | null) => Standalone;
}

/** What the manager needs from a standalone transport. */
export interface Standalone {
  start(): Promise<void>;
  stop(): Promise<void>;
  status(): ConsensusStatus;
  peers(): PeerInfo[];
  broadcastOperation(op: OperationEnvelope): Promise<void>;
  requestSync(req: SyncRequest): Promise<void>;
}

export class P2PConsensusManager implements ConsensusManager {
  private mode: 'gateway' | 'standalone' | 'disabled' = 'disabled';
  private gatewayProxy?: GatewayProxy;
  private standalone?: Standalone;
  private socketServer?: SocketServer;
  private running = false;
  private monitor?: ReturnType<typeof setInterval>;
  private failures = 0;
  private config: ConsensusConfig;

  private constructor(cfg: ConsensusConfig, socketPath: string, private handler: EventHandler | null, private opts: ManagerOptions) {
    this.config = { ...cfg, socket_path: socketPath };
  }

  /** Create a manager and resolve its mode (probes the gateway in "auto"). */
  static async create(cfg: ConsensusConfig, socketPath: string, handler: EventHandler | null, opts: ManagerOptions = {}): Promise<P2PConsensusManager> {
    const m = new P2PConsensusManager(cfg, socketPath, handler, opts);
    m.mode = await resolveMode(cfg);
    if (m.mode === 'gateway') {
      m.gatewayProxy = new GatewayProxy(cfg.gateway_url, cfg.network_id, socketPath, cfg.network_secret);
      if (handler && socketPath) {
        m.socketServer = new SocketServer(socketPath, m.bridge(handler), cfg.network_id, cfg.network_secret);
      }
    } else if (m.mode === 'standalone') {
      m.standalone = m.newStandalone();
    }
    return m;
  }

  private newStandalone(): Standalone {
    return this.opts.standaloneFactory
      ? this.opts.standaloneFactory(this.config, this.handler)
      : new StandaloneConsensus(this.config, this.handler);
  }

  private bridge(handler: EventHandler) {
    return {
      handleOperation: (op: OperationEnvelope) => handler.onOperationReceived(op),
      handleSyncRequest: (req: SyncRequest) => handler.onSyncRequestReceived(req),
    };
  }

  currentMode(): 'gateway' | 'standalone' | 'disabled' {
    return this.mode;
  }

  async start(): Promise<void> {
    if (this.running) throw new Error('already running');
    if (this.mode === 'gateway') {
      if (!this.gatewayProxy) throw new Error('gateway proxy not initialized');
      // Listen before registering so the gateway never publishes into a dead socket.
      await this.socketServer?.serve();
      try {
        await this.gatewayProxy.register();
      } catch (e: any) {
        console.warn(`[p2pconsensus] gateway registration failed: ${e.message} (falling back to standalone)`);
        await this.socketServer?.stop();
        this.mode = 'standalone';
        this.standalone = this.newStandalone();
        await this.standalone.start();
        this.running = true;
        return;
      }
    } else if (this.mode === 'standalone') {
      this.standalone ??= this.newStandalone();
      await this.standalone.start();
    }
    this.running = true;
    if (this.mode === 'gateway') {
      this.monitor = setInterval(() => void this.monitorTick(), this.opts.monitorIntervalMs ?? 5000);
      this.monitor.unref?.();
    }
  }

  async stop(): Promise<void> {
    if (!this.running) return;
    this.running = false;
    if (this.monitor) clearInterval(this.monitor);
    this.monitor = undefined;
    await this.socketServer?.stop();
    this.gatewayProxy?.close();
    await this.standalone?.stop();
  }

  private async monitorTick(): Promise<void> {
    if (!this.running) return;
    if (this.mode === 'gateway') {
      if (this.gatewayProxy && (await this.gatewayProxy.health())) {
        this.failures = 0;
        return;
      }
      if (++this.failures < (this.opts.failuresBeforeFailover ?? 3)) return;
      this.failures = 0;
      if (this.mode === 'gateway') {
        this.mode = 'standalone';
        this.standalone = this.newStandalone();
        try {
          await this.standalone.start();
        } catch (e: any) {
          console.warn(`[p2pconsensus] standalone failover failed: ${e.message}`);
        }
      }
    } else if (this.mode === 'standalone' && this.config.gateway_url && (await detectGateway(this.config.gateway_url, 2000))) {
      const proxy = new GatewayProxy(this.config.gateway_url, this.config.network_id, this.config.socket_path, this.config.network_secret);
      try {
        await proxy.register();
      } catch {
        return;
      }
      await this.standalone?.stop();
      this.gatewayProxy = proxy;
      this.mode = 'gateway';
    }
  }

  status(): ConsensusStatus {
    if (this.mode === 'gateway') {
      return this.standalone && this.standalone.status().running
        ? this.standalone.status()
        : { mode: 'gateway', network_id: this.config.network_id, peer_count: 0, running: this.running };
    }
    if (this.mode === 'standalone' && this.standalone) return this.standalone.status();
    return { mode: 'disabled', running: false };
  }

  async peers(): Promise<PeerInfo[]> {
    if (this.mode === 'gateway' && this.gatewayProxy) {
      try {
        return await this.gatewayProxy.discoverPeers('');
      } catch {
        return [];
      }
    }
    return this.mode === 'standalone' && this.standalone ? this.standalone.peers() : [];
  }

  enabled(): boolean {
    return this.config.enabled;
  }

  setEnabled(enabled: boolean): void {
    this.config.enabled = enabled;
  }

  async requestSync(req: SyncRequest): Promise<void> {
    if (!this.config.enabled) throw new Error('consensus disabled');
    if (!req.network_id) req = { ...req, network_id: this.config.network_id };
    if (this.mode === 'standalone' && this.standalone) return this.standalone.requestSync(req);
    if (this.mode === 'gateway' && this.gatewayProxy) return this.gatewayProxy.requestSync(req);
    throw new Error('consensus transport unavailable');
  }

  async publishOperation(op: OperationEnvelope): Promise<void> {
    if (!this.config.enabled) throw new Error('consensus disabled');
    if (!op.network_id) op = { ...op, network_id: this.config.network_id };
    switch (this.mode) {
      case 'gateway':
        if (!this.gatewayProxy) throw new Error('gateway proxy not initialized');
        return this.gatewayProxy.publishOperation(op);
      case 'standalone':
        if (!this.standalone) throw new Error('standalone not initialized');
        return this.standalone.broadcastOperation(op);
      default:
        throw new Error('consensus disabled');
    }
  }
}
