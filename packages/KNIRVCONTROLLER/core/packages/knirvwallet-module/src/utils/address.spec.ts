import { Secp256k1 } from '@cosmjs/crypto';

import { publicKeyToOracleAddress } from './address';

describe('oracle address derivation', () => {
  it('derives the oracle address from a secp256k1 public key', async () => {
    const privateKey = new Uint8Array(32);
    privateKey[31] = 1;

    const { pubkey } = await Secp256k1.makeKeypair(privateKey);
    const compressedPublicKey = Secp256k1.compressPubkey(pubkey);

    expect(publicKeyToOracleAddress(compressedPublicKey)).toBe(
      '0x7e5f4552091a69125d5dfcb7b8c2659029395bdf',
    );
  });
});
