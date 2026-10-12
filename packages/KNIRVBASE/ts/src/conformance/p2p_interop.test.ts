// E7: real libp2p interop between the TS StandaloneConsensus and the Go one.
// The libp2p packages are ESM-only, so the exchange runs in a plain Node child
// (scripts/p2p-interop.cjs) against the compiled CommonJS build.
import { execFileSync, spawn, spawnSync } from 'child_process';
import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';

jest.setTimeout(300_000);

const ROOT = path.resolve(__dirname, '../..');
const GO_DIR = path.resolve(ROOT, '../go');
const BIN = process.env.NRVCONFORM_BIN;
const goAvailable = () => (BIN ? fs.existsSync(BIN) : spawnSync('go', ['version']).status === 0);
const libp2pInstalled = () => fs.existsSync(path.join(ROOT, 'node_modules/libp2p/package.json'));
const maybe = (goAvailable() && libp2pInstalled()) || process.env.NRVCONFORM_REQUIRED ? describe : describe.skip;

let bin = BIN ?? '';

function run(args: string[]): Promise<{ code: number | null; json: any; out: string }> {
  return new Promise((resolve) => {
    const child = spawn('node', [path.join(ROOT, 'scripts/p2p-interop.cjs'), bin, ...args], { cwd: ROOT });
    let out = '';
    child.stdout.on('data', (d) => (out += d));
    child.stderr.on('data', (d) => (out += d));
    child.on('exit', (code) => {
      const line = out.trim().split('\n').reverse().find((l) => l.startsWith('{'));
      resolve({ code, json: line ? JSON.parse(line) : null, out });
    });
  });
}

// Distinct ports per run so reruns and parallel suites do not collide.
const base = 20000 + Math.floor(Math.random() * 20000);

maybe('libp2p standalone consensus: TS <-> Go', () => {
  beforeAll(() => {
    if (!bin) {
      bin = path.join(os.tmpdir(), `nrvconform-p2p-${process.pid}`);
      execFileSync('go', ['build', '-o', bin, './cmd/nrvconform'], { cwd: GO_DIR });
    }
    // The child loads the CommonJS build.
    execFileSync('npx', ['tsc', '-p', 'tsconfig.cjs.json'], { cwd: ROOT });
  });
  afterAll(() => { if (!BIN && bin) fs.rmSync(bin, { force: true }); });

  it('exchanges operations (gossipsub) and syncs (direct stream) in both directions', async () => {
    const r = await run([String(base), String(base + 1), 'interop-net', 'shared-secret']);
    expect(r.json?.error).toBeUndefined();
    expect(r.json?.steps).toEqual({ mesh: true, tsToGo: true, goToTs: true, tsSyncsFromGo: true, goSyncsFromTs: true });
    expect(r.code).toBe(0);
  });

  it('a node with a different network secret never joins the mesh', async () => {
    const r = await run([String(base + 2), String(base + 3), 'interop-net', 'secret-A', 'secret-B']);
    expect(r.json?.steps).toEqual({ isolated: true });
    expect(r.code).toBe(0);
  });
});
