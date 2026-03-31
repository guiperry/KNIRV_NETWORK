export interface EncryptionOptions {
    iterations?: number;
    keyLength?: number;
}
export interface EncryptedData {
    data: string;
    nonce: string;
}
export interface KeyDerivationOptions {
    salt: string;
    iterations?: number;
    keyLength?: number;
}
//# sourceMappingURL=types.d.ts.map