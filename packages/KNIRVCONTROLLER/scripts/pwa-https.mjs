import { mkdirSync, writeFileSync } from 'node:fs';
import { networkInterfaces, tmpdir } from 'node:os';
import path from 'node:path';
import selfsigned from 'selfsigned';

const CERT_DIR = path.join(tmpdir(), 'knirv-controller-pwa-https');

function collectLocalIpAddresses() {
  const envIps = process.env.KNIRV_HTTPS_IPS?.split(',')
    .map((ip) => ip.trim())
    .filter(Boolean) ?? [];

  const addresses = new Set();
  try {
    const interfaces = networkInterfaces();

    for (const entries of Object.values(interfaces)) {
      for (const entry of entries ?? []) {
        if (entry.family === 'IPv4' && !entry.internal) {
          addresses.add(entry.address);
        }
      }
    }
  } catch {
    // Sandbox or constrained environments can block interface discovery.
  }

  return [...new Set([...addresses, ...envIps])];
}

export async function ensureHttpsMaterial() {
  mkdirSync(CERT_DIR, { recursive: true });

  const altNames = [
    { type: 2, value: 'localhost' },
    { type: 2, value: 'knirv-controller.local' },
    { type: 7, ip: '127.0.0.1' },
    { type: 7, ip: '::1' },
    ...collectLocalIpAddresses().map((ip) => ({ type: 7, ip })),
  ];

  const pems = await selfsigned.generate(
    [{ name: 'commonName', value: 'knirv-controller.local' }],
    {
      algorithm: 'sha256',
      days: 365,
      keySize: 2048,
      extensions: [
        {
          name: 'subjectAltName',
          altNames,
        },
      ],
    },
  );

  const keyPath = path.join(CERT_DIR, 'server.key');
  const certPath = path.join(CERT_DIR, 'server.crt');

  writeFileSync(keyPath, pems.private);
  writeFileSync(certPath, pems.cert);

  return { keyPath, certPath };
}
