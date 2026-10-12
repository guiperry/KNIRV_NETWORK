// Client that proxies P2P operations through KNIRVGATEWAY (Go: gateway_proxy.go).
import { PeerInfo, OperationEnvelope, SyncRequest } from './types';
import { signMessage, topic } from './secret';

export class GatewayProxy {
  private closed = false;

  constructor(
    private gatewayURL: string,
    private networkID: string,
    private callbackPath: string,
    private networkSecret = '',
    private timeoutMs = 10_000
  ) {}

  private async request(method: string, path: string, body?: string, extra: Record<string, string> = {}, timeout = this.timeoutMs): Promise<Response> {
    const headers: Record<string, string> = { ...extra };
    if (body !== undefined) headers['Content-Type'] = 'application/json';
    return fetch(this.gatewayURL + path, { method, body, headers, signal: AbortSignal.timeout(timeout) });
  }

  private signed(body: string): Record<string, string> {
    const sig = signMessage(this.networkID, this.networkSecret, body);
    return sig ? { 'X-KNIRV-Signature': sig } : {};
  }

  /** Tell KNIRVGATEWAY where to deliver CRDT operations. */
  async register(): Promise<void> {
    const body = JSON.stringify({
      socket_path: this.callbackPath,
      network_id: this.networkID,
      topic: topic(this.networkID, this.networkSecret),
    });
    let resp: Response;
    try {
      resp = await this.request('POST', '/knirvbase/register-callback', body);
    } catch (e: any) {
      throw new Error(`register callback: ${e.message}`);
    }
    if (resp.status !== 200) throw new Error(`register-callback returned ${resp.status}`);
  }

  async publishOperation(op: OperationEnvelope): Promise<void> {
    const body = JSON.stringify(op);
    let resp: Response;
    try {
      resp = await this.request('POST', '/knirvbase/publish-op', body, this.signed(body));
    } catch (e: any) {
      throw new Error(`publish op: ${e.message}`);
    }
    if (resp.status !== 200) throw new Error(`publish-op returned ${resp.status}`);
  }

  async discoverPeers(query = ''): Promise<PeerInfo[]> {
    let path = `/knirvbase/discover?network_id=${encodeURIComponent(this.networkID)}`;
    if (query) path += `&query=${encodeURIComponent(query)}`;
    let resp: Response;
    try {
      resp = await this.request('GET', path);
    } catch (e: any) {
      throw new Error(`discover peers: ${e.message}`);
    }
    try {
      const peers = await resp.json();
      return Array.isArray(peers) ? peers : [];
    } catch (e: any) {
      throw new Error(`decode peers: ${e.message}`);
    }
  }

  async requestSync(req: SyncRequest): Promise<void> {
    if (!req.network_id) req = { ...req, network_id: this.networkID };
    const body = JSON.stringify(req);
    let resp: Response;
    try {
      resp = await this.request('POST', '/knirvbase/sync-request', body, this.signed(body));
    } catch (e: any) {
      throw new Error(`sync request: ${e.message}`);
    }
    if (resp.status !== 200) throw new Error(`sync request returned ${resp.status}`);
  }

  async health(): Promise<boolean> {
    try {
      return (await this.request('GET', '/p2p/health', undefined, {}, 2000)).status === 200;
    } catch {
      return false;
    }
  }

  close(): void {
    this.closed = true;
  }

  get isClosed(): boolean {
    return this.closed;
  }
}

/** Probe `<gatewayURL>/p2p/health`. */
export async function detectGateway(gatewayURL: string, timeoutMs: number): Promise<boolean> {
  if (!gatewayURL) return false;
  try {
    const r = await fetch(gatewayURL + '/p2p/health', { signal: AbortSignal.timeout(timeoutMs) });
    return r.status === 200;
  } catch {
    return false;
  }
}
