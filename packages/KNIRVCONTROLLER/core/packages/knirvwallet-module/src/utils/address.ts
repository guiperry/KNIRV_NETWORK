import { ripemd160, Secp256k1 } from '@cosmjs/crypto';
import { keccak_256 } from '@noble/hashes/sha3';

import { fromBech32, toBech32, toHex } from '../encoding';
import { sha256 } from '../crypto';

// KNIRV-compatible address generation.
// Standard Cosmos/Gno-style derivation: ripemd160(sha256(compressed pubkey)),
// bech32-encoded. This replaces a placeholder that copied raw pubkey bytes
// into the address without hashing them at all.
export async function publicKeyToAddress(
  publicKey: Uint8Array,
  addressPrefix: string = 'knirv',
): Promise<string> {
  const hash = ripemd160(sha256(publicKey));
  return toBech32(addressPrefix, hash);
}

export function publicKeyToOracleAddress(publicKey: Uint8Array): string {
  const uncompressed = Secp256k1.uncompressPubkey(publicKey);
  const hash = keccak_256(uncompressed.slice(1));
  return `0x${toHex(hash.slice(12))}`;
}

export function validateAddress(address: string): boolean {
  try {
    const publicKey = fromBech32(address);
    return Boolean(publicKey?.prefix);
  } catch {
    return false;
  }
}
