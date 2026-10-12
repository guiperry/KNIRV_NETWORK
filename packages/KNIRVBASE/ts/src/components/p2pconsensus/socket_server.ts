// HTTP server on a Unix socket that the gateway calls into (Go: socket_server.go).
import * as fs from 'fs';
import * as http from 'http';
import * as path from 'path';
import { OperationEnvelope, SyncRequest, SyncResponse } from './types';
import { verifyMessage } from './secret';

export interface ConsensusHandler {
  handleOperation(op: OperationEnvelope): Promise<void> | void;
  handleSyncRequest(req: SyncRequest): Promise<SyncResponse | null> | SyncResponse | null;
}

export class SocketServer {
  private server?: http.Server;
  private started = false;

  constructor(
    private socketPath: string,
    private handler: ConsensusHandler | null,
    private networkID: string,
    private networkSecret: string
  ) {}

  /**
   * Bind the socket with owner-only permissions (0600; directory 0700) and
   * remove a stale socket left by a crash. Resolves once listening.
   */
  async serve(): Promise<void> {
    if (this.started) throw new Error('socket server already started');
    this.started = true;
    try {
      const dir = path.dirname(this.socketPath);
      if (dir !== '.') fs.mkdirSync(dir, { recursive: true, mode: 0o700 });
      fs.rmSync(this.socketPath, { force: true });

      this.server = http.createServer((req, res) => void this.route(req, res));
      await new Promise<void>((resolve, reject) => {
        this.server!.once('error', reject);
        this.server!.listen(this.socketPath, () => resolve());
      });
      fs.chmodSync(this.socketPath, 0o600);
    } catch (e) {
      this.started = false;
      throw e;
    }
  }

  async stop(): Promise<void> {
    const server = this.server;
    this.server = undefined;
    this.started = false;
    if (server) {
      await new Promise<void>((resolve) => server.close(() => resolve()));
    }
    fs.rmSync(this.socketPath, { force: true });
  }

  private async route(req: http.IncomingMessage, res: http.ServerResponse): Promise<void> {
    const fail = (code: number, msg: string) => {
      res.writeHead(code, { 'Content-Type': 'text/plain; charset=utf-8' });
      res.end(msg + '\n');
    };
    const url = (req.url ?? '').split('?')[0];
    if (url !== '/internal/p2p/received-op' && url !== '/internal/p2p/received-sync-request') {
      return fail(404, '404 page not found');
    }
    if (req.method !== 'POST') return fail(405, 'method not allowed');

    const chunks: Buffer[] = [];
    for await (const c of req) chunks.push(c as Buffer);
    const body = Buffer.concat(chunks);

    if (this.networkSecret !== '') {
      const sig = String(req.headers['x-knirv-signature'] ?? '');
      if (!verifyMessage(this.networkID, this.networkSecret, sig, body)) return fail(401, 'unauthorized');
    }

    let msg: any;
    try {
      msg = JSON.parse(body.toString('utf8'));
    } catch (e: any) {
      return fail(400, `bad request: ${e.message}`);
    }
    if (this.networkID !== '' && msg.network_id && msg.network_id !== this.networkID) {
      return fail(400, 'network mismatch');
    }
    if (!this.handler) return fail(500, 'no handler registered');

    try {
      if (url === '/internal/p2p/received-op') {
        await this.handler.handleOperation(msg as OperationEnvelope);
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ status: 'ok' }) + '\n');
      } else {
        const resp = await this.handler.handleSyncRequest(msg as SyncRequest);
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify(resp) + '\n');
      }
    } catch (e: any) {
      fail(500, `handler error: ${e.message}`);
    }
  }
}
