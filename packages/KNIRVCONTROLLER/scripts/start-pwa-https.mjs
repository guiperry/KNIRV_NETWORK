#!/usr/bin/env node

import { spawn } from 'node:child_process';
import { ensureHttpsMaterial } from './pwa-https.mjs';

const viteBin = process.platform === 'win32' ? 'vite.cmd' : 'vite';
const host = process.env.KNIRV_PWA_HOST ?? '0.0.0.0';
const port = process.env.KNIRV_PWA_PORT ?? '4174';

async function main() {
  const { keyPath, certPath } = await ensureHttpsMaterial();

  const child = spawn(viteBin, ['--host', host, '--port', port], {
    env: {
      ...process.env,
      KNIRV_HTTPS: '1',
      KNIRV_HTTPS_KEY: keyPath,
      KNIRV_HTTPS_CERT: certPath,
    },
    stdio: 'inherit',
  });

  process.on('SIGINT', () => {
    child.kill('SIGINT');
  });

  process.on('SIGTERM', () => {
    child.kill('SIGTERM');
  });

  child.on('exit', (code, signal) => {
    process.exit(signal ? 1 : (code ?? 0));
  });
}

main().catch((error) => {
  console.error(error instanceof Error ? error.message : error);
  process.exit(1);
});
