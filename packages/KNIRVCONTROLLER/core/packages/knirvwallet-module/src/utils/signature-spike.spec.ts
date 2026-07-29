import { Secp256k1 } from '@cosmjs/crypto';
import { keccak_256 } from '@noble/hashes/sha3';
import { toHex } from '../encoding/hex';
import { publicKeyToOracleAddress } from './address';

describe('signature interop spike', () => {
  it('signs a test message and outputs results for Go verification', async () => {
    const privateKey = new Uint8Array(32);
    privateKey[31] = 1;

    const { pubkey } = await Secp256k1.makeKeypair(privateKey);
    const compressedPublicKey = Secp256k1.compressPubkey(pubkey);

    const oracleAddress = publicKeyToOracleAddress(compressedPublicKey);

    const message = 'register:0xTESTADDR:controller-test';
    const messageHash = keccak_256(new TextEncoder().encode(message));
    const signature = await Secp256k1.createSignature(messageHash, privateKey);
    const signatureHex = '0x' + toHex(signature.toFixedLength());

    console.log('oracleAddress:', oracleAddress);
    console.log('message:', message);
    console.log('signature:', signatureHex);
    console.log('privateKey:', '0x' + toHex(privateKey));

    expect(oracleAddress).toBeTruthy();
    expect(signatureHex).toBeTruthy();
  });
});