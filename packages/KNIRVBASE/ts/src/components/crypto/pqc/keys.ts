import * as crypto from 'crypto';
import { ml_kem768 } from '@noble/post-quantum/ml-kem';
import { ml_dsa65 } from '@noble/post-quantum/ml-dsa';

// PQCKeyPair is a complete PQC key pair: ML-KEM-768 (FIPS 203) for encryption and
// ML-DSA-65 (FIPS 204) for signatures. Interoperable with Go's PQCKeyPair in
// internal/crypto/pqc/keys.go: same algorithms, same serialised key format
// (`knirv-pqc-key/v2`), same ciphertext layout.
//
// Sizes: ML-KEM-768 pk 1184 / sk 2400 / ct 1088 / ss 32;
//        ML-DSA-65  pk 1952 / sk 4032 / sig 3309.
export const ALGORITHM_KEM = 'ML-KEM-768';
export const ALGORITHM_SIG = 'ML-DSA-65';
export const ALGORITHM_SUITE = `${ALGORITHM_KEM}+${ALGORITHM_SIG}`;
export const ENVELOPE_ALGORITHM = `${ALGORITHM_KEM}+AES-256-GCM`;
export const KEY_FORMAT = 'knirv-pqc-key/v2';

const KEM_CT_LEN = 1088;
const GCM_NONCE_LEN = 12;
const GCM_TAG_LEN = 16;

export interface PQCKeyPair {
  id: string;
  name: string;
  purpose: string; // encryption, signature, kex
  algorithm: string;
  createdAt: Date;
  expiresAt?: Date;
  status: string; // active, rotated, revoked, expired

  // ML-KEM-768 keys for encryption
  kyberPublicKeyBytes: Uint8Array;
  kyberPrivateKeyBytes?: Uint8Array;

  // ML-DSA-65 keys for signatures
  dilithiumPublicKeyBytes: Uint8Array;
  dilithiumPrivateKeyBytes?: Uint8Array;
}

// GeneratePQCKeyPair generates a new PQC key pair with real ML-KEM-768 and ML-DSA-65 keys.
// Mirrors Go's GeneratePQCKeyPair.
export function generatePQCKeyPair(name: string, purpose: string): PQCKeyPair {
  const idBytes = crypto.randomBytes(16);
  const id = idBytes.toString('hex');

  // ML-KEM-768 requires 64 bytes of entropy for key generation
  const kemSeed = crypto.randomBytes(64);
  const kemKeys = ml_kem768.keygen(kemSeed);

  // ML-DSA-65 requires 32 bytes of entropy for key generation
  const dsaSeed = crypto.randomBytes(32);
  const dsaKeys = ml_dsa65.keygen(dsaSeed);

  return {
    id,
    name,
    purpose,
    algorithm: ALGORITHM_SUITE,
    createdAt: new Date(),
    status: 'active',
    kyberPublicKeyBytes: kemKeys.publicKey,
    kyberPrivateKeyBytes: kemKeys.secretKey,
    dilithiumPublicKeyBytes: dsaKeys.publicKey,
    dilithiumPrivateKeyBytes: dsaKeys.secretKey,
  };
}

const b64 = (u?: Uint8Array): string | undefined =>
  u && u.length ? Buffer.from(u).toString('base64') : undefined;

function toGoJSON(kp: PQCKeyPair, withPrivate: boolean): string {
  // Key order matches Go's struct field order (json.Marshal).
  return JSON.stringify({
    format: KEY_FORMAT,
    id: kp.id,
    name: kp.name,
    purpose: kp.purpose,
    algorithm: kp.algorithm,
    created_at: kp.createdAt.toISOString(),
    expires_at: kp.expiresAt ? kp.expiresAt.toISOString() : undefined,
    status: kp.status,
    kyber_public_key: b64(kp.kyberPublicKeyBytes),
    kyber_private_key: withPrivate ? b64(kp.kyberPrivateKeyBytes) : undefined,
    dilithium_public_key: b64(kp.dilithiumPublicKeyBytes),
    dilithium_private_key: withPrivate ? b64(kp.dilithiumPrivateKeyBytes) : undefined,
  });
}

// LoadPQCKeyPair deserializes a key pair from the Go-compatible JSON format.
// Legacy (round-3 Kyber/Dilithium) keys are rejected: generate a new key pair.
export function loadPQCKeyPair(data: string): PQCKeyPair {
  const j = JSON.parse(data);
  if (j.format !== KEY_FORMAT || j.algorithm !== ALGORITHM_SUITE) {
    throw new Error(
      `unsupported key format ${JSON.stringify(j.format)} / algorithm ${JSON.stringify(j.algorithm)} ` +
        `(want ${KEY_FORMAT} / ${ALGORITHM_SUITE}): generate a new key pair`
    );
  }
  return {
    id: j.id,
    name: j.name,
    purpose: j.purpose,
    algorithm: j.algorithm,
    createdAt: new Date(j.created_at),
    expiresAt: j.expires_at ? new Date(j.expires_at) : undefined,
    status: j.status,
    kyberPublicKeyBytes: bufferFromField(j.kyber_public_key),
    kyberPrivateKeyBytes: j.kyber_private_key ? bufferFromField(j.kyber_private_key) : undefined,
    dilithiumPublicKeyBytes: bufferFromField(j.dilithium_public_key),
    dilithiumPrivateKeyBytes: j.dilithium_private_key ? bufferFromField(j.dilithium_private_key) : undefined,
  };
}

// Marshal serializes the key pair without private keys (for public storage).
export function marshalPublic(kp: PQCKeyPair): string {
  return toGoJSON(kp, false);
}

// MarshalWithPrivateKeys serializes the key pair including private keys.
// WARNING: only use for encrypted storage.
export function marshalWithPrivateKeys(kp: PQCKeyPair): string {
  return toGoJSON(kp, true);
}

// Encrypt encrypts plaintext using ML-KEM-768 + AES-256-GCM.
// Output: kem_ciphertext (1088) || nonce (12) || aes_ciphertext || gcm_tag (16),
// identical to Go's KyberEncrypt. The 32-byte KEM shared secret is the AES key.
export function encrypt(kp: PQCKeyPair, plaintext: Uint8Array): Uint8Array {
  const { cipherText: kemCt, sharedSecret } = ml_kem768.encapsulate(kp.kyberPublicKeyBytes);

  const nonce = crypto.randomBytes(GCM_NONCE_LEN);
  const cipher = crypto.createCipheriv('aes-256-gcm', sharedSecret, nonce);
  const encrypted = Buffer.concat([cipher.update(plaintext), cipher.final()]);
  const authTag = cipher.getAuthTag();

  const result = new Uint8Array(kemCt.length + GCM_NONCE_LEN + encrypted.length + GCM_TAG_LEN);
  result.set(kemCt, 0);
  result.set(nonce, kemCt.length);
  result.set(encrypted, kemCt.length + GCM_NONCE_LEN);
  result.set(authTag, kemCt.length + GCM_NONCE_LEN + encrypted.length);
  return result;
}

// Decrypt decrypts ciphertext produced by encrypt() or Go's KyberEncrypt.
export function decrypt(kp: PQCKeyPair, ciphertext: Uint8Array): Uint8Array {
  if (!kp.kyberPrivateKeyBytes) {
    throw new Error('no Kyber private key available');
  }
  if (ciphertext.length < KEM_CT_LEN + GCM_NONCE_LEN + GCM_TAG_LEN) {
    throw new Error('ciphertext too short');
  }

  const kemCt = ciphertext.subarray(0, KEM_CT_LEN);
  const nonce = ciphertext.subarray(KEM_CT_LEN, KEM_CT_LEN + GCM_NONCE_LEN);
  const sealed = ciphertext.subarray(KEM_CT_LEN + GCM_NONCE_LEN);
  const authTag = sealed.subarray(sealed.length - GCM_TAG_LEN);
  const encryptedData = sealed.subarray(0, sealed.length - GCM_TAG_LEN);

  const sharedSecret = ml_kem768.decapsulate(kemCt, kp.kyberPrivateKeyBytes);

  const decipher = crypto.createDecipheriv('aes-256-gcm', sharedSecret, nonce);
  decipher.setAuthTag(authTag);
  return new Uint8Array(Buffer.concat([decipher.update(encryptedData), decipher.final()]));
}

// Sign signs a message using ML-DSA-65 (Dilithium-3).
export function sign(kp: PQCKeyPair, message: Uint8Array): Uint8Array {
  if (!kp.dilithiumPrivateKeyBytes) {
    throw new Error('no Dilithium private key available');
  }
  return ml_dsa65.sign(kp.dilithiumPrivateKeyBytes, message);
}

// Verify verifies a ML-DSA-65 signature.
export function verify(kp: PQCKeyPair, message: Uint8Array, signature: Uint8Array): boolean {
  return ml_dsa65.verify(kp.dilithiumPublicKeyBytes, message, signature);
}

// IsExpired checks if the key pair has expired.
export function isExpired(kp: PQCKeyPair): boolean {
  if (!kp.expiresAt) return false;
  return new Date() > kp.expiresAt;
}

// IsActive checks if the key pair is active and not expired.
export function isActive(kp: PQCKeyPair): boolean {
  return kp.status === 'active' && !isExpired(kp);
}

function bufferFromField(field: unknown): Uint8Array {
  if (field instanceof Uint8Array) return field;
  if (Array.isArray(field)) return new Uint8Array(field);
  if (typeof field === 'string') return new Uint8Array(Buffer.from(field, 'base64'));
  return new Uint8Array();
}
