import { fromBech32, toBech32 } from '../encoding';

// KNIRV-compatible address generation
export async function publicKeyToAddress(
  publicKey: Uint8Array,
  addressPrefix: string = 'knirv',
): Promise<string> {
  // This is a simplified implementation
  // In a real implementation, this would hash the public key and encode it properly
  const hash = new Uint8Array(20); // Placeholder 20-byte address
  // Copy some bytes from the public key for uniqueness
  for (let i = 0; i < Math.min(publicKey.length, 20); i++) {
    hash[i] = publicKey[i];
  }
  return toBech32(addressPrefix, hash);
}

export function validateAddress(address: string): boolean {
  try {
    const publicKey = fromBech32(address);
    return Boolean(publicKey?.prefix);
  } catch {
    return false;
  }
}
