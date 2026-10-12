import * as net from 'net';
import { NetworkManager } from '../network_manager';
import { MessageType, ProtocolMessage } from '../../types/types';

// Plays the Go legacy_tcp peer: "KNIRV:<id>\n" handshake, then newline-delimited JSON.
describe('legacy TCP transport (Go legacy_tcp wire protocol)', () => {
  let nm: NetworkManager;
  beforeEach(async () => { nm = new NetworkManager(); await nm.initialize(); });
  afterEach(async () => { await nm.shutdown(); });

  const connect = () =>
    new Promise<net.Socket>((resolve) => {
      const s = net.createConnection({ host: '127.0.0.1', port: nm.listenPort() }, () => resolve(s));
    });
  const readLine = (s: net.Socket) =>
    new Promise<string>((resolve) => s.once('data', (d) => resolve(d.toString().trim())));
  const msg = (id: string): ProtocolMessage => ({
    type: MessageType.Heartbeat, networkId: 'n', senderId: 'go-peer', timestamp: 1, payload: { id },
  });

  it('answers the handshake with its own peer id', async () => {
    const s = await connect();
    s.write('KNIRV:go-peer\n');
    expect(await readLine(s)).toBe(`KNIRV:${nm.getPeerID()}`);
    s.destroy();
  });

  it('reassembles messages split across TCP chunks and handles several per chunk', async () => {
    const got: string[] = [];
    nm.onMessage(MessageType.Heartbeat, (m) => got.push((m.payload as any).id));
    const s = await connect();
    s.write('KNIRV:go-peer\n');
    await readLine(s);

    const a = JSON.stringify(msg('a'));
    const b = JSON.stringify(msg('b'));
    const c = JSON.stringify(msg('c'));
    s.write(a.slice(0, 10));
    await new Promise((r) => setTimeout(r, 30));
    s.write(a.slice(10) + '\n' + b + '\n' + c.slice(0, 5));
    await new Promise((r) => setTimeout(r, 30));
    s.write(c.slice(5) + '\n');
    await new Promise((r) => setTimeout(r, 50));
    expect(got).toEqual(['a', 'b', 'c']);
    s.destroy();
  });

  it('rejects a bad handshake and ignores garbage lines', async () => {
    const got: string[] = [];
    nm.onMessage(MessageType.Heartbeat, (m) => got.push((m.payload as any).id));
    const bad = await connect();
    bad.write('HELLO:x\n' + JSON.stringify(msg('never')) + '\n');
    await new Promise<void>((r) => bad.once('close', () => r()));
    expect(got).toEqual([]);

    const ok = await connect();
    ok.write('KNIRV:p\n');
    await readLine(ok);
    ok.write('not json\n' + JSON.stringify(msg('fine')) + '\n');
    await new Promise((r) => setTimeout(r, 50));
    expect(got).toEqual(['fine']);
    ok.destroy();
  });

  it('two managers exchange messages after joinNetwork', async () => {
    const other = new NetworkManager();
    await other.initialize();
    try {
      const got: string[] = [];
      other.onMessage(MessageType.Heartbeat, (m) => got.push((m.payload as any).id));
      await nm.createNetwork({ networkId: 'n' } as any);
      await nm.joinNetwork('n', [`127.0.0.1:${other.listenPort()}`]);
      for (let i = 0; i < 50 && !(nm as any).connections.has(other.getPeerID()); i++) await new Promise((r) => setTimeout(r, 20));
      await nm.sendToPeer(other.getPeerID(), 'n', msg('hello'));
      for (let i = 0; i < 50 && got.length === 0; i++) await new Promise((r) => setTimeout(r, 20));
      expect(got).toEqual(['hello']);
    } finally {
      await other.shutdown();
    }
  });
});
