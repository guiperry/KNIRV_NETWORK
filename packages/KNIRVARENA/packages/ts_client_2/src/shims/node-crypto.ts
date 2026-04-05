/**
 * Browser shim for Node.js 'crypto' module.
 * Covers all APIs used by knirvbase-ts dist files.
 * Server-only blocking APIs (scryptSync, createCipheriv, etc.) throw at call-time —
 * they are never invoked by the arena's browser code paths.
 */

// ── Web Crypto pass-throughs (used via `import * as crypto` in keys.js / network_manager.js) ──
export const subtle = globalThis.crypto.subtle;
export const getRandomValues = <T extends ArrayBufferView>(arr: T): T =>
  globalThis.crypto.getRandomValues(arr);

// ── randomBytes ────────────────────────────────────────────────────────────────────────────────
export function randomBytes(size: number): Uint8Array {
  const bytes = new Uint8Array(size);
  globalThis.crypto.getRandomValues(bytes);
  return bytes;
}

// ── timingSafeEqual ────────────────────────────────────────────────────────────────────────────
export function timingSafeEqual(
  a: ArrayBufferLike | { buffer: ArrayBufferLike },
  b: ArrayBufferLike | { buffer: ArrayBufferLike }
): boolean {
  const aView = new Uint8Array('buffer' in a ? (a as { buffer: ArrayBufferLike }).buffer : a);
  const bView = new Uint8Array('buffer' in b ? (b as { buffer: ArrayBufferLike }).buffer : b);
  if (aView.length !== bView.length) return false;
  let diff = 0;
  for (let i = 0; i < aView.length; i++) diff |= aView[i] ^ bView[i];
  return diff === 0;
}

// ── Node-only stubs (server-side; not called in browser paths) ─────────────────────────────────
function notAvailable(name: string): never {
  throw new Error(`crypto.${name} is not available in browser environments`);
}

export function scryptSync(..._args: unknown[]): never {
  return notAvailable('scryptSync');
}

export function createCipheriv(..._args: unknown[]): never {
  return notAvailable('createCipheriv');
}

export function createDecipheriv(..._args: unknown[]): never {
  return notAvailable('createDecipheriv');
}

export function createHmac(..._args: unknown[]): never {
  return notAvailable('createHmac');
}
