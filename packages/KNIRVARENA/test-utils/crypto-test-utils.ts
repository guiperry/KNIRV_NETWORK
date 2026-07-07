// Cryptographic Test Utilities for KNIRVWALLET
import { TEST_MNEMONICS, TEST_PRIVATE_KEYS, TEST_ENCRYPTION_DATA } from './test-data';

// Mnemonic validation and testing utilities
export class MnemonicTestUtils {
  static readonly VALID_WORD_COUNTS = [12, 15, 18, 21, 24];
  
  static readonly TEST_WORDS = [
    'abandon', 'ability', 'able', 'about', 'above', 'absent', 'absorb', 'abstract',
    'absurd', 'abuse', 'access', 'accident', 'account', 'accuse', 'achieve', 'acid',
    'acoustic', 'acquire', 'across', 'act', 'action', 'actor', 'actress', 'actual'
  ];

  static generateTestMnemonic(wordCount: number = 12): string {
    if (!this.VALID_WORD_COUNTS.includes(wordCount)) {
      throw new Error(`Invalid word count: ${wordCount}. Must be one of: ${this.VALID_WORD_COUNTS.join(', ')}`);
    }

    const words = [];
    for (let i = 0; i < wordCount; i++) {
      const randomIndex = Math.floor(Math.random() * this.TEST_WORDS.length);
      words.push(this.TEST_WORDS[randomIndex]);
    }
    
    return words.join(' ');
  }

  static validateMnemonicFormat(mnemonic: string): boolean {
    if (!mnemonic || typeof mnemonic !== 'string') {
      return false;
    }

    const words = mnemonic.trim().split(/\s+/);
    return this.VALID_WORD_COUNTS.includes(words.length);
  }

  static validateMnemonicWords(mnemonic: string): boolean {
    const words = mnemonic.trim().split(/\s+/);
    return words.every(word => this.TEST_WORDS.includes(word.toLowerCase()));
  }

  static createInvalidMnemonic(): string {
    return 'invalid mnemonic phrase with wrong words count';
  }

  static createMnemonicWithInvalidWords(): string {
    return 'invalid words that are not in bip39 wordlist test phrase';
  }
}

// Private key testing utilities
export class PrivateKeyTestUtils {
  static generateTestPrivateKey(): string {
    const bytes = new Uint8Array(32);
    for (let i = 0; i < 32; i++) {
      bytes[i] = Math.floor(Math.random() * 256);
    }
    return Array.from(bytes, byte => byte.toString(16).padStart(2, '0')).join('');
  }

  static validatePrivateKeyFormat(privateKey: string): boolean {
    if (!privateKey || typeof privateKey !== 'string') {
      return false;
    }

    // Remove 0x prefix if present
    const cleanKey = privateKey.replace(/^0x/, '');
    
    // Check if it's exactly 64 hex characters
    return /^[0-9a-fA-F]{64}$/.test(cleanKey);
  }

  static addHexPrefix(privateKey: string): string {
    return privateKey.startsWith('0x') ? privateKey : `0x${privateKey}`;
  }

  static removeHexPrefix(privateKey: string): string {
    return privateKey.replace(/^0x/, '');
  }

  static createInvalidPrivateKey(): string {
    return 'invalid_private_key_format';
  }

  static createShortPrivateKey(): string {
    return '1234567890abcdef'; // Too short
  }

  static createLongPrivateKey(): string {
    return TEST_PRIVATE_KEYS.VALID_HEX + TEST_PRIVATE_KEYS.VALID_HEX; // Too long
  }
}

// Address derivation testing utilities
export class AddressTestUtils {
  static generateTestAddress(prefix: string = 'g'): string {
    const randomPart = Math.random().toString(36).substr(2, 38);
    return `${prefix}${randomPart}`;
  }

  static validateAddressFormat(address: string, expectedPrefix?: string): boolean {
    if (!address || typeof address !== 'string') {
      return false;
    }

    if (expectedPrefix) {
      return address.startsWith(expectedPrefix) && address.length > expectedPrefix.length;
    }

    // General validation - should start with known prefixes
    const knownPrefixes = ['g', 'xion', '0x', 'cosmos', 'osmo'];
    return knownPrefixes.some(prefix => address.startsWith(prefix));
  }

  static createInvalidAddress(): string {
    return 'invalid_address_format_123';
  }

  static createAddressForNetwork(network: string): string {
    switch (network.toLowerCase()) {
      case 'gnolang':
      case 'gno':
        return this.generateTestAddress('g');
      case 'xion':
        return this.generateTestAddress('xion');
      case 'ethereum':
      case 'eth':
        return this.generateTestAddress('0x');
      case 'cosmos':
        return this.generateTestAddress('cosmos');
      default:
        return this.generateTestAddress('g');
    }
  }
}

// Encryption and decryption testing utilities
export class EncryptionTestUtils {
  static async mockEncrypt(data: string, password: string): Promise<string> {
    // Simple mock encryption for testing (not secure)
    const encoded = Buffer.from(data).toString('base64');
    const passwordHash = this.simpleHash(password);
    return `${encoded}.${passwordHash}`;
  }

  static async mockDecrypt(encryptedData: string, password: string): Promise<string> {
    const [encoded, expectedHash] = encryptedData.split('.');
    const passwordHash = this.simpleHash(password);
    
    if (passwordHash !== expectedHash) {
      throw new Error('Invalid password');
    }
    
    return Buffer.from(encoded, 'base64').toString();
  }

  static generateTestSalt(): string {
    return TEST_ENCRYPTION_DATA.SALT;
  }

  static createTestKDFConfig() {
    return TEST_ENCRYPTION_DATA.KDF_CONFIG;
  }

  static async testEncryptionRoundTrip(data: string, password: string): Promise<boolean> {
    try {
      const encrypted = await this.mockEncrypt(data, password);
      const decrypted = await this.mockDecrypt(encrypted, password);
      return data === decrypted;
    } catch (error) {
      console.error('Encryption round trip test failed:', error);
      return false;
    }
  }

  static createTestEncryptedWallet(): string {
    return TEST_ENCRYPTION_DATA.ENCRYPTED_WALLET_DATA;
  }

  private static simpleHash(input: string): string {
    let hash = 0;
    for (let i = 0; i < input.length; i++) {
      const char = input.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash; // Convert to 32-bit integer
    }
    return Math.abs(hash).toString(16).padStart(8, '0');
  }
}

// Signature testing utilities
export class SignatureTestUtils {
  static generateTestSignature(): string {
    // Generate a mock signature (64 bytes = 128 hex characters)
    const bytes = new Uint8Array(64);
    for (let i = 0; i < 64; i++) {
      bytes[i] = Math.floor(Math.random() * 256);
    }
    return Array.from(bytes, byte => byte.toString(16).padStart(2, '0')).join('');
  }

  static validateSignatureFormat(signature: string): boolean {
    if (!signature || typeof signature !== 'string') {
      return false;
    }

    // Remove 0x prefix if present
    const cleanSignature = signature.replace(/^0x/, '');
    
    // Check if it's exactly 128 hex characters (64 bytes)
    return /^[0-9a-fA-F]{128}$/.test(cleanSignature);
  }

  static async mockSignTransaction(privateKey: string, transaction: Record<string, unknown>): Promise<string> {
    // Simple mock signing for testing
    const txData = JSON.stringify(transaction);
    const combined = privateKey + txData;
    
    let hash = 0;
    for (let i = 0; i < combined.length; i++) {
      const char = combined.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash;
    }
    
    // Convert to 64-byte signature
    const hashHex = Math.abs(hash).toString(16).padStart(16, '0');
    return hashHex.repeat(8); // 128 hex characters
  }

  static async mockVerifySignature(signature: string, transaction: Record<string, unknown>, publicKey: string): Promise<boolean> {
    // Simple mock verification for testing
    // In real implementation, this would use proper cryptographic verification
    return this.validateSignatureFormat(signature) && !!transaction && !!publicKey;
  }

  static createInvalidSignature(): string {
    return 'invalid_signature_format';
  }
}

// Key derivation testing utilities
export class KeyDerivationTestUtils {
  static readonly TEST_DERIVATION_PATHS = {
    BITCOIN: "m/44'/0'/0'/0/0",
    ETHEREUM: "m/44'/60'/0'/0/0",
    COSMOS: "m/44'/118'/0'/0/0",
    GNOLANG: "m/44'/118'/0'/0/0",
    XION: "m/44'/118'/0'/0/0"
  };

  static validateDerivationPath(path: string): boolean {
    // Basic validation for BIP44 derivation paths
    const bip44Regex = /^m\/44'\/\d+'\/\d+'\/\d+\/\d+$/;
    return bip44Regex.test(path);
  }

  static generateDerivationPath(coinType: number = 118, account: number = 0, change: number = 0, index: number = 0): string {
    return `m/44'/${coinType}'/${account}'/${change}/${index}`;
  }

  static parseDerivationPath(path: string): { coinType: number; account: number; change: number; index: number } | null {
    const match = path.match(/^m\/44'\/(\d+)'\/(\d+)'\/(\d+)\/(\d+)$/);
    if (!match) return null;

    return {
      coinType: parseInt(match[1]),
      account: parseInt(match[2]),
      change: parseInt(match[3]),
      index: parseInt(match[4])
    };
  }

  static async mockDeriveKey(mnemonic: string, derivationPath: string): Promise<string> {
    // Simple mock key derivation for testing
    const combined = mnemonic + derivationPath;
    
    let hash = 0;
    for (let i = 0; i < combined.length; i++) {
      const char = combined.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash;
    }
    
    // Convert to 32-byte private key
    const hashHex = Math.abs(hash).toString(16).padStart(16, '0');
    return hashHex.repeat(4); // 64 hex characters
  }

  static async mockDerivePublicKey(privateKey: string): Promise<string> {
    // Simple mock public key derivation for testing
    let hash = 0;
    for (let i = 0; i < privateKey.length; i++) {
      const char = privateKey.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash;
    }
    
    // Convert to 33-byte compressed public key
    const hashHex = Math.abs(hash).toString(16).padStart(16, '0');
    return '02' + hashHex.repeat(4); // 66 hex characters (compressed)
  }

  static createInvalidDerivationPath(): string {
    return "invalid/derivation/path";
  }
}

// Comprehensive crypto test suite
export class CryptoTestSuite {
  static async runAllTests(): Promise<{ passed: number; failed: number; results: Array<{ name: string; passed: boolean; error?: string }> }> {
    const tests = [
      { name: 'Mnemonic Generation', test: () => this.testMnemonicGeneration() },
      { name: 'Private Key Generation', test: () => this.testPrivateKeyGeneration() },
      { name: 'Address Derivation', test: () => this.testAddressDerivation() },
      { name: 'Encryption Round Trip', test: () => this.testEncryptionRoundTrip() },
      { name: 'Signature Generation', test: () => this.testSignatureGeneration() },
      { name: 'Key Derivation', test: () => this.testKeyDerivation() }
    ];

    const results = [];
    let passed = 0;
    let failed = 0;

    for (const test of tests) {
      try {
        await test.test();
        results.push({ name: test.name, passed: true });
        passed++;
      } catch (error) {
        results.push({ name: test.name, passed: false, error: error instanceof Error ? error.message : String(error) });
        failed++;
      }
    }

    return { passed, failed, results };
  }

  private static async testMnemonicGeneration(): Promise<void> {
    const mnemonic = MnemonicTestUtils.generateTestMnemonic(12);
    if (!MnemonicTestUtils.validateMnemonicFormat(mnemonic)) {
      throw new Error('Generated mnemonic has invalid format');
    }
  }

  private static async testPrivateKeyGeneration(): Promise<void> {
    const privateKey = PrivateKeyTestUtils.generateTestPrivateKey();
    if (!PrivateKeyTestUtils.validatePrivateKeyFormat(privateKey)) {
      throw new Error('Generated private key has invalid format');
    }
  }

  private static async testAddressDerivation(): Promise<void> {
    const address = AddressTestUtils.generateTestAddress('g');
    if (!AddressTestUtils.validateAddressFormat(address, 'g')) {
      throw new Error('Generated address has invalid format');
    }
  }

  private static async testEncryptionRoundTrip(): Promise<void> {
    const testData = 'test encryption data';
    const password = 'test-password';
    
    const success = await EncryptionTestUtils.testEncryptionRoundTrip(testData, password);
    if (!success) {
      throw new Error('Encryption round trip failed');
    }
  }

  private static async testSignatureGeneration(): Promise<void> {
    const signature = SignatureTestUtils.generateTestSignature();
    if (!SignatureTestUtils.validateSignatureFormat(signature)) {
      throw new Error('Generated signature has invalid format');
    }
  }

  private static async testKeyDerivation(): Promise<void> {
    const mnemonic = TEST_MNEMONICS.VALID_12_WORD;
    const path = KeyDerivationTestUtils.TEST_DERIVATION_PATHS.GNOLANG;
    
    const privateKey = await KeyDerivationTestUtils.mockDeriveKey(mnemonic, path);
    if (!PrivateKeyTestUtils.validatePrivateKeyFormat(privateKey)) {
      throw new Error('Derived private key has invalid format');
    }
  }
}
