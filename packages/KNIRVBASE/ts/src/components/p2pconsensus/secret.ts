// Pre-shared network key handling — byte-for-byte identical to Go's secret.go.
import * as crypto from 'crypto';

/** sha256("knirvbase-network-v1" 0 networkID 0 secret). */
export function deriveNetworkKey(networkID: string, secret: string): Buffer {
  return crypto
    .createHash('sha256')
    .update(Buffer.concat([
      Buffer.from('knirvbase-network-v1'), Buffer.from([0]),
      Buffer.from(networkID), Buffer.from([0]),
      Buffer.from(secret),
    ]))
    .digest();
}

/** Stable identifier for the DHT service / rendezvous point. */
export function serviceId(networkID: string, secret: string): string {
  return `knirv-${deriveNetworkKey(networkID, secret).subarray(0, 8).toString('hex')}`;
}

/** Pubsub topic bound to the secret so same-named networks never cross-talk. */
export function topic(networkID: string, secret: string): string {
  return `knirvbase.${deriveNetworkKey(networkID, secret).subarray(0, 8).toString('hex')}.ops`;
}

/** HMAC-SHA256 hex over the payload; empty string on an open network. */
export function signMessage(networkID: string, secret: string, payload: Uint8Array | string): string {
  if (secret === '') return '';
  return crypto.createHmac('sha256', deriveNetworkKey(networkID, secret)).update(payload).digest('hex');
}

/** Open networks accept anything; otherwise the HMAC must match (constant time). */
export function verifyMessage(networkID: string, secret: string, signature: string, payload: Uint8Array | string): boolean {
  if (secret === '') return true;
  if (!signature) return false;
  const expected = Buffer.from(signMessage(networkID, secret, payload));
  const got = Buffer.from(signature.toLowerCase());
  return expected.length === got.length && crypto.timingSafeEqual(expected, got);
}
