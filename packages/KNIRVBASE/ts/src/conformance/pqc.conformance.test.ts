// PQC interop: ML-KEM-768 + ML-DSA-65 keys, KEM+AES-GCM ciphertexts,
// signatures and the signed envelope must cross Go <-> TS in both directions.
import { execFileSync, spawnSync } from 'child_process';
import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';
import {
  decrypt, encrypt, generatePQCKeyPair, loadPQCKeyPair, marshalWithPrivateKeys, sign, verify,
} from '../components/crypto/pqc/keys';
import { EncryptionManager } from '../components/crypto/pqc/encryption';

jest.setTimeout(180_000);

const GO_DIR = path.resolve(__dirname, '../../../go');
const BIN = process.env.NRVCONFORM_BIN;
const goAvailable = () => (BIN ? fs.existsSync(BIN) : spawnSync('go', ['version']).status === 0);
const goRun = (...args: string[]): string =>
  (BIN
    ? execFileSync(BIN, args, { maxBuffer: 64 << 20 })
    : execFileSync('go', ['run', './cmd/nrvconform', ...args], { cwd: GO_DIR, maxBuffer: 64 << 20 })
  ).toString().trim();

const maybe = goAvailable() || process.env.NRVCONFORM_REQUIRED ? describe : describe.skip;
const b64 = (u: Uint8Array | Buffer) => Buffer.from(u).toString('base64');

maybe('PQC Go <-> TS', () => {
  let dir: string;
  let goKeyFile: string;
  let tsKeyFile: string;
  beforeAll(() => {
    dir = fs.mkdtempSync(path.join(os.tmpdir(), 'pqcconform-'));
    goKeyFile = path.join(dir, 'go-key.json');
    tsKeyFile = path.join(dir, 'ts-key.json');
    fs.writeFileSync(goKeyFile, goRun('pqc', 'keygen'));
    fs.writeFileSync(tsKeyFile, marshalWithPrivateKeys(generatePQCKeyPair('conform', 'encryption')));
  });
  afterAll(() => fs.rmSync(dir, { recursive: true, force: true }));

  const msg = new TextEncoder().encode('knirvbase pqc interop — ünïcode 🙂');

  it('TS loads a Go-generated key and Go loads a TS-generated key', () => {
    const kp = loadPQCKeyPair(fs.readFileSync(goKeyFile, 'utf8'));
    expect(kp.algorithm).toBe('ML-KEM-768+ML-DSA-65');
    expect(kp.kyberPublicKeyBytes).toHaveLength(1184);
    expect(kp.kyberPrivateKeyBytes).toHaveLength(2400);
    expect(kp.dilithiumPublicKeyBytes).toHaveLength(1952);
    expect(kp.dilithiumPrivateKeyBytes).toHaveLength(4032);
    // Go accepts the TS key (it would error on load otherwise).
    expect(goRun('pqc', 'sign', tsKeyFile, b64(msg))).not.toBe('');
  });

  it('KEM+AES-GCM: Go encrypts -> TS decrypts', () => {
    const kp = loadPQCKeyPair(fs.readFileSync(goKeyFile, 'utf8'));
    const ct = Buffer.from(goRun('pqc', 'encrypt', goKeyFile, b64(msg)), 'base64');
    expect(Buffer.from(decrypt(kp, ct))).toEqual(Buffer.from(msg));
  });

  it('KEM+AES-GCM: TS encrypts -> Go decrypts', () => {
    const kp = loadPQCKeyPair(fs.readFileSync(tsKeyFile, 'utf8'));
    const ct = encrypt(kp, msg);
    expect(goRun('pqc', 'decrypt', tsKeyFile, b64(ct))).toBe(b64(msg));
  });

  it('ML-DSA-65: Go signs -> TS verifies (and rejects tampering)', () => {
    const kp = loadPQCKeyPair(fs.readFileSync(goKeyFile, 'utf8'));
    const sig = Buffer.from(goRun('pqc', 'sign', goKeyFile, b64(msg)), 'base64');
    expect(sig).toHaveLength(3309);
    expect(verify(kp, msg, sig)).toBe(true);
    expect(verify(kp, new TextEncoder().encode('tampered'), sig)).toBe(false);
  });

  it('ML-DSA-65: TS signs -> Go verifies (and rejects tampering)', () => {
    const kp = loadPQCKeyPair(fs.readFileSync(tsKeyFile, 'utf8'));
    const sig = sign(kp, msg);
    expect(goRun('pqc', 'verify', tsKeyFile, b64(msg), b64(sig))).toBe('true');
    expect(goRun('pqc', 'verify', tsKeyFile, b64(Buffer.from('tampered')), b64(sig))).toBe('false');
  });

  it('signed envelope: Go EncryptData -> TS decryptData', async () => {
    const kp = loadPQCKeyPair(fs.readFileSync(goKeyFile, 'utf8'));
    const em = new EncryptionManager();
    em.setMasterKey(kp);
    const enc = goRun('pqc', 'envelope-encrypt', goKeyFile, b64(msg));
    expect(Buffer.from(await em.decryptData(enc))).toEqual(Buffer.from(msg));
  });

  it('signed envelope: TS encryptData -> Go DecryptData', async () => {
    const kp = loadPQCKeyPair(fs.readFileSync(tsKeyFile, 'utf8'));
    const em = new EncryptionManager();
    em.setMasterKey(kp);
    const enc = await em.encryptData(msg, kp.id);
    expect(goRun('pqc', 'envelope-decrypt', tsKeyFile, enc)).toBe(b64(msg));
  });
});
