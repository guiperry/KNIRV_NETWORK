import { randomBytes } from 'crypto';
import { Secp256k1 } from '@cosmjs/crypto';
import {
  signMessageEnvelope,
  verifyMessage,
  verifyMessagePayload,
  publicKeyToKNIRVAddress,
  MessageEnvelope,
} from './signing';

async function makeSignedEnvelope(overrides: Partial<MessageEnvelope> = {}) {
  const privateKey = randomBytes(32);
  const { pubkey } = await Secp256k1.makeKeypair(privateKey);
  const publicKey = Secp256k1.compressPubkey(pubkey);
  const nowUnix = Math.floor(Date.now() / 1000);
  const envelope: MessageEnvelope = {
    domain: 'knirv.controller',
    purpose: 'wasm-publication',
    chainId: 'knirv-testnet-1',
    nonce: 'nonce-0001',
    issuedAtUnix: nowUnix,
    expiresAtUnix: nowUnix + 300,
    payload: new TextEncoder().encode('payload'),
    ...overrides,
  };
  const signed = await signMessageEnvelope(privateKey, envelope);
  return { signed, envelope, publicKey };
}

describe('verifyMessage — Go VerifyMessage parity', () => {
  it('accepts a validly signed, in-window envelope', async () => {
    const { signed, envelope } = await makeSignedEnvelope();
    await expect(
      verifyMessage(signed, envelope.domain, envelope.purpose, envelope.chainId, envelope.nonce, new Date()),
    ).resolves.toBeUndefined();
  });

  it('rejects domain/purpose/chainId/nonce mismatch', async () => {
    const { signed, envelope } = await makeSignedEnvelope();
    await expect(
      verifyMessage(signed, 'wrong.domain', envelope.purpose, envelope.chainId, envelope.nonce, new Date()),
    ).rejects.toThrow();
  });

  it('rejects an expired envelope even though the signature is valid', async () => {
    const { signed, envelope } = await makeSignedEnvelope({
      issuedAtUnix: Math.floor(Date.now() / 1000) - 600,
      expiresAtUnix: Math.floor(Date.now() / 1000) - 300,
    });
    await expect(
      verifyMessage(signed, envelope.domain, envelope.purpose, envelope.chainId, envelope.nonce, new Date()),
    ).rejects.toThrow(/validity window/);
  });

  it('rejects a not-yet-valid envelope beyond the 60s clock-skew allowance', async () => {
    const { signed, envelope } = await makeSignedEnvelope({
      issuedAtUnix: Math.floor(Date.now() / 1000) + 3600,
      expiresAtUnix: Math.floor(Date.now() / 1000) + 7200,
    });
    await expect(
      verifyMessage(signed, envelope.domain, envelope.purpose, envelope.chainId, envelope.nonce, new Date()),
    ).rejects.toThrow(/validity window/);
  });

  it('rejects a tampered signature', async () => {
    const { signed, envelope } = await makeSignedEnvelope();
    const tampered = { ...signed, signature: signed.signature.slice(0, -4) + 'AAAA' };
    await expect(
      verifyMessage(tampered, envelope.domain, envelope.purpose, envelope.chainId, envelope.nonce, new Date()),
    ).rejects.toThrow();
  });

  it('rejects a public key that does not match the claimed address', async () => {
    const { signed, envelope } = await makeSignedEnvelope();
    const otherKeypair = await Secp256k1.makeKeypair(randomBytes(32));
    const otherAddress = publicKeyToKNIRVAddress(Secp256k1.compressPubkey(otherKeypair.pubkey));
    const tampered = { ...signed, address: otherAddress };
    await expect(
      verifyMessage(tampered, envelope.domain, envelope.purpose, envelope.chainId, envelope.nonce, new Date()),
    ).rejects.toThrow(/does not match public key/);
  });
});

describe('verifyMessagePayload — Go VerifyMessagePayload parity', () => {
  it('accepts a matching payload and rejects a tampered one', async () => {
    const { signed, envelope } = await makeSignedEnvelope();
    await expect(
      verifyMessagePayload(signed, envelope.domain, envelope.purpose, envelope.chainId, envelope.payload, new Date()),
    ).resolves.toBeUndefined();

    await expect(
      verifyMessagePayload(
        signed,
        envelope.domain,
        envelope.purpose,
        envelope.chainId,
        new TextEncoder().encode('different payload'),
        new Date(),
      ),
    ).rejects.toThrow(/payload does not match/);
  });
});
