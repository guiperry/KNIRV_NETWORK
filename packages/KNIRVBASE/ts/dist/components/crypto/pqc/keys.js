import * as crypto from 'crypto';
import { ml_kem768 } from '@noble/post-quantum/ml-kem';
import { ml_dsa65 } from '@noble/post-quantum/ml-dsa';
// GeneratePQCKeyPair generates a new PQC key pair with real ML-KEM-768 and ML-DSA-65 keys.
// Mirrors Go's GeneratePQCKeyPair.
export function generatePQCKeyPair(name, purpose) {
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
        algorithm: 'Kyber-768+Dilithium-3',
        createdAt: new Date(),
        status: 'active',
        kyberPublicKeyBytes: kemKeys.publicKey,
        kyberPrivateKeyBytes: kemKeys.secretKey,
        dilithiumPublicKeyBytes: dsaKeys.publicKey,
        dilithiumPrivateKeyBytes: dsaKeys.secretKey,
    };
}
// LoadPQCKeyPair deserializes a key pair from a JSON string.
export function loadPQCKeyPair(data) {
    const parsed = JSON.parse(data);
    return {
        ...parsed,
        createdAt: new Date(parsed.createdAt),
        expiresAt: parsed.expiresAt ? new Date(parsed.expiresAt) : undefined,
        kyberPublicKeyBytes: bufferFromField(parsed.kyberPublicKeyBytes),
        kyberPrivateKeyBytes: parsed.kyberPrivateKeyBytes ? bufferFromField(parsed.kyberPrivateKeyBytes) : undefined,
        dilithiumPublicKeyBytes: bufferFromField(parsed.dilithiumPublicKeyBytes),
        dilithiumPrivateKeyBytes: parsed.dilithiumPrivateKeyBytes ? bufferFromField(parsed.dilithiumPrivateKeyBytes) : undefined,
    };
}
// Marshal serializes the key pair to JSON without private keys (for public storage).
export function marshalPublic(kp) {
    const { kyberPrivateKeyBytes, dilithiumPrivateKeyBytes, ...pub } = kp;
    return JSON.stringify({
        ...pub,
        kyberPublicKeyBytes: Array.from(kp.kyberPublicKeyBytes),
        dilithiumPublicKeyBytes: Array.from(kp.dilithiumPublicKeyBytes),
    });
}
// MarshalWithPrivateKeys serializes the key pair to JSON including private keys.
// WARNING: only use for encrypted storage.
export function marshalWithPrivateKeys(kp) {
    return JSON.stringify({
        ...kp,
        kyberPublicKeyBytes: Array.from(kp.kyberPublicKeyBytes),
        kyberPrivateKeyBytes: kp.kyberPrivateKeyBytes ? Array.from(kp.kyberPrivateKeyBytes) : undefined,
        dilithiumPublicKeyBytes: Array.from(kp.dilithiumPublicKeyBytes),
        dilithiumPrivateKeyBytes: kp.dilithiumPrivateKeyBytes ? Array.from(kp.dilithiumPrivateKeyBytes) : undefined,
    });
}
// Encrypt encrypts plaintext using ML-KEM-768 + AES-256-GCM.
// Output format: kem_ciphertext (1088 bytes) || nonce (12 bytes) || aes_ciphertext
// Mirrors Go's KyberEncrypt.
export function encrypt(kp, plaintext) {
    // Encapsulate: generates a shared secret and a KEM ciphertext
    const { cipherText: kemCt, sharedSecret } = ml_kem768.encapsulate(kp.kyberPublicKeyBytes);
    // Derive AES-256 key from the 32-byte shared secret via SHA-256
    const aesKey = crypto.createHash('sha256').update(sharedSecret).digest();
    // AES-256-GCM encrypt
    const nonce = crypto.randomBytes(12);
    const cipher = crypto.createCipheriv('aes-256-gcm', aesKey, nonce);
    const encrypted = Buffer.concat([cipher.update(plaintext), cipher.final()]);
    const authTag = cipher.getAuthTag();
    // Concatenate: kemCt || nonce || authTag || encrypted
    const result = new Uint8Array(kemCt.length + 12 + 16 + encrypted.length);
    result.set(kemCt, 0);
    result.set(nonce, kemCt.length);
    result.set(authTag, kemCt.length + 12);
    result.set(encrypted, kemCt.length + 12 + 16);
    return result;
}
// Decrypt decrypts ciphertext using ML-KEM-768 + AES-256-GCM.
// Mirrors Go's KyberDecrypt.
export function decrypt(kp, ciphertext) {
    if (!kp.kyberPrivateKeyBytes) {
        throw new Error('no Kyber private key available');
    }
    const kemCtLen = 1088; // ML-KEM-768 ciphertext size
    if (ciphertext.length < kemCtLen + 12 + 16) {
        throw new Error('ciphertext too short');
    }
    const kemCt = ciphertext.slice(0, kemCtLen);
    const nonce = ciphertext.slice(kemCtLen, kemCtLen + 12);
    const authTag = ciphertext.slice(kemCtLen + 12, kemCtLen + 12 + 16);
    const encryptedData = ciphertext.slice(kemCtLen + 12 + 16);
    // Decapsulate: recover the shared secret
    const sharedSecret = ml_kem768.decapsulate(kemCt, kp.kyberPrivateKeyBytes);
    // Derive AES-256 key
    const aesKey = crypto.createHash('sha256').update(sharedSecret).digest();
    // AES-256-GCM decrypt
    const decipher = crypto.createDecipheriv('aes-256-gcm', aesKey, nonce);
    decipher.setAuthTag(authTag);
    const decrypted = Buffer.concat([decipher.update(encryptedData), decipher.final()]);
    return new Uint8Array(decrypted);
}
// Sign signs a message using ML-DSA-65 (Dilithium-3).
export function sign(kp, message) {
    if (!kp.dilithiumPrivateKeyBytes) {
        throw new Error('no Dilithium private key available');
    }
    return ml_dsa65.sign(kp.dilithiumPrivateKeyBytes, message);
}
// Verify verifies a ML-DSA-65 signature.
export function verify(kp, message, signature) {
    return ml_dsa65.verify(kp.dilithiumPublicKeyBytes, message, signature);
}
// IsExpired checks if the key pair has expired.
export function isExpired(kp) {
    if (!kp.expiresAt)
        return false;
    return new Date() > kp.expiresAt;
}
// IsActive checks if the key pair is active and not expired.
export function isActive(kp) {
    return kp.status === 'active' && !isExpired(kp);
}
function bufferFromField(field) {
    if (field instanceof Uint8Array)
        return field;
    if (Array.isArray(field))
        return new Uint8Array(field);
    if (typeof field === 'string')
        return Buffer.from(field, 'base64');
    return new Uint8Array();
}
//# sourceMappingURL=keys.js.map