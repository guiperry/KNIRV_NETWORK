// E7: gateway proxy, socket server, shared-secret handling and the manager,
// checked against the Go implementation (cmd/nrvconform p2p ...).
import { ChildProcessWithoutNullStreams, execFile, execFileSync, spawn, spawnSync } from 'child_process';
import * as fs from 'fs';
import * as http from 'http';
import * as os from 'os';
import * as path from 'path';
import * as readline from 'readline';
import {
  GatewayProxy, P2PConsensusManager, SocketServer, defaultConsensusConfig, signMessage, serviceId, topic, verifyMessage,
  OperationEnvelope, EventHandler, SyncRequest, SyncResponse, parseGoDuration, Standalone,
} from '../components/p2pconsensus';

jest.setTimeout(180_000);

const GO_DIR = path.resolve(__dirname, '../../../go');
const BIN = process.env.NRVCONFORM_BIN;
const goAvailable = () => (BIN ? fs.existsSync(BIN) : spawnSync('go', ['version']).status === 0);
let goBin = BIN ?? '';
const goArgs = (args: string[]) => (goBin ? { cmd: goBin, args } : { cmd: 'go', args: ['run', './cmd/nrvconform', ...args] });
const goRun = (...args: string[]): string => {
  const g = goArgs(args);
  return execFileSync(g.cmd, g.args, { cwd: GO_DIR, maxBuffer: 64 << 20 }).toString().trim();
};
/** Async variant: required when the TS side of the exchange lives in this process. */
const goRunAsync = (...args: string[]): Promise<string> =>
  new Promise((resolve, reject) => {
    const g = goArgs(args);
    execFile(g.cmd, g.args, { cwd: GO_DIR, maxBuffer: 64 << 20 }, (err, out) => (err ? reject(err) : resolve(out.trim())));
  });
function goSpawn(args: string[]): ChildProcessWithoutNullStreams {
  const g = goArgs(args);
  return spawn(g.cmd, g.args, { cwd: GO_DIR });
}
/** Collect JSON lines from a child process. */
function lines(child: ChildProcessWithoutNullStreams) {
  const events: any[] = [];
  const waiters: Array<() => void> = [];
  readline.createInterface({ input: child.stdout }).on('line', (l) => {
    try { events.push(JSON.parse(l)); } catch { /* log noise */ }
    waiters.splice(0).forEach((w) => w());
  });
  return {
    events,
    async next(pred: (e: any) => boolean, ms = 30_000): Promise<any> {
      const deadline = Date.now() + ms;
      for (;;) {
        const hit = events.find(pred);
        if (hit) return hit;
        if (Date.now() > deadline) throw new Error('timed out waiting for event');
        await new Promise<void>((r) => { waiters.push(r); setTimeout(r, 200); });
      }
    },
  };
}

const op = (id: string): OperationEnvelope => ({
  network_id: 'net-1', collection: 'c', document_id: id,
  data: Buffer.from('{"type":"insert"}').toString('base64'),
  vector_clock: { a: 1 }, timestamp: 1700000000000, peer_id: 'ts',
});

describe('shared secret (pure)', () => {
  it('open network: no signature required', () => {
    expect(signMessage('n', '', 'x')).toBe('');
    expect(verifyMessage('n', '', '', 'x')).toBe(true);
  });
  it('rejects missing/wrong/other-network signatures', () => {
    const sig = signMessage('n', 's', 'payload');
    expect(verifyMessage('n', 's', sig, 'payload')).toBe(true);
    expect(verifyMessage('n', 's', sig.toUpperCase(), 'payload')).toBe(true);
    expect(verifyMessage('n', 's', '', 'payload')).toBe(false);
    expect(verifyMessage('n', 's', sig, 'tampered')).toBe(false);
    expect(verifyMessage('other', 's', sig, 'payload')).toBe(false);
    expect(topic('n', 's1')).not.toBe(topic('n', 's2'));
  });
  it('parses Go durations', () => {
    expect(parseGoDuration('2s')).toBe(2000);
    expect(parseGoDuration('1m30s')).toBe(90_000);
    expect(parseGoDuration('500ms')).toBe(500);
    expect(Number.isNaN(parseGoDuration('soon'))).toBe(true);
  });
});

describe('GatewayProxy against a mock gateway', () => {
  let server: http.Server;
  let url: string;
  const seen: Array<{ method: string; url: string; body: string; sig?: string }> = [];
  let healthy = true;
  beforeAll(async () => {
    server = http.createServer((req, res) => {
      const chunks: Buffer[] = [];
      req.on('data', (c) => chunks.push(c));
      req.on('end', () => {
        seen.push({ method: req.method!, url: req.url!, body: Buffer.concat(chunks).toString(), sig: req.headers['x-knirv-signature'] as string });
        if (req.url === '/p2p/health') { res.writeHead(healthy ? 200 : 503); return res.end(); }
        if (req.url!.startsWith('/knirvbase/discover')) { res.writeHead(200); return res.end(JSON.stringify([{ id: 'p1' }])); }
        res.writeHead(200); res.end('{}');
      });
    });
    await new Promise<void>((r) => server.listen(0, '127.0.0.1', r));
    url = `http://127.0.0.1:${(server.address() as any).port}`;
  });
  afterAll(() => new Promise<void>((r) => server.close(() => r())));

  it('registers, publishes (signed), syncs (signed), discovers and checks health', async () => {
    const gp = new GatewayProxy(url, 'net-1', '/tmp/cb.sock', 'sekret');
    await gp.register();
    await gp.publishOperation(op('d1'));
    await gp.requestSync({ network_id: '', collection: 'c', vector_clock: {} });
    expect(await gp.discoverPeers()).toEqual([{ id: 'p1' }]);
    expect(await gp.health()).toBe(true);

    const reg = seen.find((s) => s.url === '/knirvbase/register-callback')!;
    expect(JSON.parse(reg.body)).toEqual({ socket_path: '/tmp/cb.sock', network_id: 'net-1', topic: topic('net-1', 'sekret') });
    const pub = seen.find((s) => s.url === '/knirvbase/publish-op')!;
    expect(pub.sig).toBe(signMessage('net-1', 'sekret', pub.body));
    const sync = seen.find((s) => s.url === '/knirvbase/sync-request')!;
    expect(JSON.parse(sync.body).network_id).toBe('net-1'); // defaulted
    expect(sync.sig).toBe(signMessage('net-1', 'sekret', sync.body));
    expect(seen.find((s) => s.url.startsWith('/knirvbase/discover'))!.url).toContain('network_id=net-1');
  });

  it('open network sends no signature header', async () => {
    seen.length = 0;
    await new GatewayProxy(url, 'net-1', '/tmp/cb.sock').publishOperation(op('d2'));
    expect(seen[0].sig).toBeUndefined();
  });

  it('manager: auto picks gateway, registers, and fails over to standalone when the gateway dies', async () => {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'p2pmgr-'));
    const sock = path.join(dir, 'cb.sock');
    const received: OperationEnvelope[] = [];
    const handler: EventHandler = {
      onOperationReceived: (o) => { received.push(o); },
      onSyncRequestReceived: () => null,
      onPeerDiscovered: () => undefined,
    };
    let standaloneStarted = false;
    const stub: Standalone = {
      start: async () => { standaloneStarted = true; },
      stop: async () => undefined,
      status: () => ({ mode: 'standalone', running: true, peer_count: 0 }),
      peers: () => [],
      broadcastOperation: async () => undefined,
      requestSync: async () => undefined,
    };
    healthy = true;
    const cfg = { ...defaultConsensusConfig(), network_id: 'net-1', gateway_url: url, mode: 'auto' as const };
    const m = await P2PConsensusManager.create(cfg, sock, handler, { monitorIntervalMs: 50, failuresBeforeFailover: 2, standaloneFactory: () => stub });
    expect(m.currentMode()).toBe('gateway');
    await m.start();
    expect(fs.statSync(sock).mode & 0o777).toBe(0o600);
    expect(m.status().mode).toBe('gateway');

    healthy = false;
    for (let i = 0; i < 100 && m.currentMode() === 'gateway'; i++) await new Promise((r) => setTimeout(r, 50));
    expect(m.currentMode()).toBe('standalone');
    expect(standaloneStarted).toBe(true);
    await m.stop();
    fs.rmSync(dir, { recursive: true, force: true });
  });

  it('manager: unreachable gateway in auto mode resolves to standalone; disabled refuses to publish', async () => {
    const cfg = { ...defaultConsensusConfig(), gateway_url: 'http://127.0.0.1:1', gateway_timeout: '200ms' };
    expect((await P2PConsensusManager.create(cfg, '', null)).currentMode()).toBe('standalone');
    const off = await P2PConsensusManager.create({ ...cfg, enabled: false }, '', null);
    expect(off.currentMode()).toBe('disabled');
    await expect(off.publishOperation(op('x'))).rejects.toThrow('consensus disabled');
  });
});

const maybe = goAvailable() || process.env.NRVCONFORM_REQUIRED ? describe : describe.skip;

maybe('Go <-> TS', () => {
  beforeAll(() => {
    // Build once: `go run` per call is slow, and node children need a stable binary.
    if (!goBin) {
      goBin = path.join(os.tmpdir(), `nrvconform-${process.pid}`);
      execFileSync('go', ['build', '-o', goBin, './cmd/nrvconform'], { cwd: GO_DIR });
    }
  });
  afterAll(() => { if (!BIN && goBin) fs.rmSync(goBin, { force: true }); });

  it.each([['net-1', 'sekret', 'hello'], ['net-1', '', 'hello'], ['üñï', 's', '{"a":1}'], ['', 's', '']])(
    'secret derivation matches Go: net=%j secret=%j',
    (net, secret, payload) => {
      const go = JSON.parse(goRun('p2p', 'secret', net, secret, payload));
      expect(serviceId(net, secret)).toBe(go.service_id);
      expect(topic(net, secret)).toBe(go.topic);
      expect(signMessage(net, secret, payload)).toBe(go.sig);
    }
  );

  describe('socket server', () => {
    let dir: string;
    beforeAll(() => { dir = fs.mkdtempSync(path.join(os.tmpdir(), 'p2psock-')); });
    afterAll(() => fs.rmSync(dir, { recursive: true, force: true }));

    it('TS server accepts a Go client: auth, network check, routing', async () => {
      const sock = path.join(dir, 'ts.sock');
      const ops: OperationEnvelope[] = [];
      const syncs: SyncRequest[] = [];
      const resp: SyncResponse = { network_id: 'net-1', collection: 'c', operations: [op('s1')], more_available: false };
      const srv = new SocketServer(sock, {
        handleOperation: (o) => { ops.push(o); },
        handleSyncRequest: (r) => { syncs.push(r); return resp; },
      }, 'net-1', 'sekret');
      await srv.serve();
      try {
        expect(fs.statSync(sock).mode & 0o777).toBe(0o600);
        const post = async (p: string, net: string, secret: string, body: unknown) =>
          JSON.parse(await goRunAsync('p2p', 'post-socket', sock, p, net, secret, JSON.stringify(body)));

        const ok = await post('/internal/p2p/received-op', 'net-1', 'sekret', op('d1'));
        expect(ok.status).toBe(200);
        expect(JSON.parse(ok.body)).toEqual({ status: 'ok' });
        expect(ops.map((o) => o.document_id)).toEqual(['d1']);

        expect((await post('/internal/p2p/received-op', 'net-1', 'WRONG', op('d2'))).status).toBe(401);
        expect((await post('/internal/p2p/received-op', 'net-1', '', op('d2'))).status).toBe(401);
        expect((await post('/internal/p2p/received-op', 'net-1', 'sekret', { ...op('d3'), network_id: 'other' })).status).toBe(400);
        expect((await post('/nope', 'net-1', 'sekret', op('d4'))).status).toBe(404);
        expect(ops).toHaveLength(1);

        const s = await post('/internal/p2p/received-sync-request', 'net-1', 'sekret', { network_id: 'net-1', collection: 'c', vector_clock: { a: 3 } });
        expect(s.status).toBe(200);
        expect(JSON.parse(s.body).operations[0].document_id).toBe('s1');
        expect(syncs[0].vector_clock).toEqual({ a: 3 });
      } finally {
        await srv.stop();
      }
      expect(fs.existsSync(sock)).toBe(false);
    });

    it('Go server accepts a TS client', async () => {
      const sock = path.join(dir, 'go.sock');
      const child = goSpawn(['p2p', 'socket-serve', sock, 'net-1', 'sekret']);
      const out = lines(child);
      try {
        await out.next((e) => e.event === 'ready');
        const send = (secret: string, body: unknown): Promise<number> =>
          new Promise((resolve, reject) => {
            const data = JSON.stringify(body);
            const headers: Record<string, string> = { 'Content-Type': 'application/json' };
            const sig = signMessage('net-1', secret, data);
            if (sig) headers['X-KNIRV-Signature'] = sig;
            const req = http.request({ socketPath: sock, path: '/internal/p2p/received-op', method: 'POST', headers }, (r) => {
              r.resume(); r.on('end', () => resolve(r.statusCode!));
            });
            req.on('error', reject);
            req.end(data);
          });
        expect(await send('sekret', op('from-ts'))).toBe(200);
        expect(await send('bad', op('rejected'))).toBe(401);
        const ev = await out.next((e) => e.event === 'op');
        expect(ev.op.document_id).toBe('from-ts');
        expect(ev.op.data).toBe(op('x').data); // []byte <-> base64 round trip
        expect(out.events.filter((e) => e.event === 'op')).toHaveLength(1);
      } finally {
        child.stdin.end();
        await new Promise((r) => child.once('exit', r));
      }
    });
  });
});
