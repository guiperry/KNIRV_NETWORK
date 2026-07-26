export interface EncryptionOptions {
  iterations?: number;
  keyLength?: number;
}

export interface EncryptedData {
  data: string; // Base64 encoded encrypted data
  nonce: string; // Base64 encoded nonce
}

export interface KeyDerivationOptions {
  salt: string; // Base64 encoded salt
  iterations?: number;
  keyLength?: number;
}