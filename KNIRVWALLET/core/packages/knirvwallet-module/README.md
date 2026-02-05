# KNIRV Wallet Core Module

The core cryptographic and wallet management library for the KNIRV Network ecosystem. This module provides the fundamental building blocks for wallet operations, transaction signing, and key management across all KNIRV Wallet implementations.

## Overview

The KNIRV Wallet Module (`knirvwallet-module`) is a TypeScript library that serves as the foundation for:
- **Browser Extension**: Core wallet functionality for the browser extension
- **Mobile App**: Cryptographic operations for the React Native app
- **Web Applications**: Direct integration for web-based dApps
- **Server Applications**: Backend wallet operations and validation

## Features

### 🔐 Cryptographic Operations
- **BIP39 Mnemonic**: Generate and validate mnemonic phrases
- **HD Wallets**: Hierarchical Deterministic wallet support (BIP32/BIP44)
- **Key Derivation**: Secure key derivation for multiple accounts
- **Digital Signatures**: Transaction and message signing
- **Hash Functions**: SHA-256, RIPEMD-160, and other cryptographic hashes
- **Random Generation**: Cryptographically secure random number generation

### 💼 Wallet Management
- **Multiple Keyring Types**:
  - HD (Hierarchical Deterministic) wallets
  - Private key wallets
  - Ledger hardware wallets
  - Web3Auth social login integration
  - Address-only (watch-only) wallets

### 🔧 Utility Functions
- **Address Validation**: Validate KNIRV Network addresses
- **Encoding/Decoding**: Base64, Hex, Bech32, and ASCII utilities
- **Data Structures**: Efficient handling of blockchain data
- **Type Safety**: Full TypeScript support with comprehensive types

## Installation

```bash
# Using npm
npm install knirvwallet-module

# Using yarn
yarn add knirvwallet-module

# Using pnpm
pnpm add knirvwallet-module
```

## Quick Start

### Basic Wallet Creation

```typescript
import { HDWalletKeyring, Bip39 } from 'knirvwallet-module';

// Generate a new mnemonic
const mnemonic = Bip39.generateMnemonic(128); // 12 words
console.log('Mnemonic:', mnemonic);

// Create HD wallet from mnemonic
const hdWallet = new HDWalletKeyring();
await hdWallet.deserialize({
  mnemonic: mnemonic,
  numberOfAccounts: 1,
  hdPath: "m/44'/118'/0'/0/0" // KNIRV Network derivation path
});

// Get the first account
const accounts = await hdWallet.getAccounts();
console.log('Address:', accounts[0].address);
```

### Transaction Signing

```typescript
import { Wallet } from 'knirvwallet-module';

// Create wallet instance
const wallet = new Wallet();

// Add HD keyring
const keyring = new HDWalletKeyring();
await keyring.deserialize({ mnemonic: 'your mnemonic here' });
wallet.addKeyring(keyring);

// Sign a transaction
const tx = {
  from: 'knirv1...',
  to: 'knirv1...',
  amount: '1000000',
  gas: '200000',
  memo: 'Test transaction'
};

const signedTx = await wallet.signTransaction(tx);
console.log('Signed transaction:', signedTx);
```

## API Reference

### Core Classes

#### `Wallet`
Main wallet class that manages multiple keyrings and provides unified interface.

```typescript
class Wallet {
  // Add a keyring to the wallet
  addKeyring(keyring: Keyring): void;
  
  // Get all accounts from all keyrings
  getAccounts(): Promise<Account[]>;
  
  // Sign a transaction
  signTransaction(tx: Transaction): Promise<SignedTransaction>;
  
  // Sign a message
  signMessage(message: string, address: string): Promise<string>;
  
  // Serialize wallet data
  serialize(): Promise<SerializedWallet>;
  
  // Deserialize wallet data
  deserialize(data: SerializedWallet): Promise<void>;
}
```

#### `HDWalletKeyring`
Hierarchical Deterministic wallet implementation.

```typescript
class HDWalletKeyring implements Keyring {
  // Create accounts from mnemonic
  deserialize(opts: {
    mnemonic: string;
    numberOfAccounts?: number;
    hdPath?: string;
  }): Promise<void>;
  
  // Get all accounts
  getAccounts(): Promise<Account[]>;
  
  // Add new account
  addAccount(): Promise<Account>;
  
  // Sign transaction with specific account
  signTransaction(tx: Transaction, address: string): Promise<SignedTransaction>;
}
```

#### `PrivateKeyKeyring`
Private key-based wallet implementation.

```typescript
class PrivateKeyKeyring implements Keyring {
  // Import from private key
  deserialize(privateKeys: string[]): Promise<void>;
  
  // Add private key
  addAccount(privateKey: string): Promise<Account>;
  
  // Export private key
  exportAccount(address: string): Promise<string>;
}
```

### Cryptographic Functions

#### `Bip39`
BIP39 mnemonic operations.

```typescript
class Bip39 {
  // Generate mnemonic
  static generateMnemonic(strength?: number): string;
  
  // Validate mnemonic
  static validateMnemonic(mnemonic: string): boolean;
  
  // Convert mnemonic to seed
  static mnemonicToSeed(mnemonic: string, passphrase?: string): Uint8Array;
  
  // Get word list
  static getWordList(): string[];
}
```

#### `Random`
Cryptographically secure random generation.

```typescript
class Random {
  // Generate random bytes
  static getBytes(length: number): Uint8Array;
  
  // Generate random integer
  static getUint32(): number;
  
  // Generate random string
  static getString(length: number): string;
}
```

### Encoding Utilities

#### `Hex`
Hexadecimal encoding/decoding.

```typescript
class Hex {
  static encode(data: Uint8Array): string;
  static decode(hex: string): Uint8Array;
}
```

#### `Base64`
Base64 encoding/decoding.

```typescript
class Base64 {
  static encode(data: Uint8Array): string;
  static decode(base64: string): Uint8Array;
}
```

#### `Bech32`
Bech32 address encoding/decoding for KNIRV Network.

```typescript
class Bech32 {
  static encode(prefix: string, data: Uint8Array): string;
  static decode(address: string): { prefix: string; data: Uint8Array };
}
```

## Advanced Usage

### Custom Keyring Implementation

```typescript
import { Keyring, Account } from 'knirvwallet-module';

class CustomKeyring implements Keyring {
  type = 'Custom Keyring';
  
  async getAccounts(): Promise<Account[]> {
    // Implementation
  }
  
  async signTransaction(tx: Transaction, address: string): Promise<SignedTransaction> {
    // Implementation
  }
  
  async serialize(): Promise<any> {
    // Implementation
  }
  
  async deserialize(data: any): Promise<void> {
    // Implementation
  }
}
```

### Ledger Integration

```typescript
import { LedgerKeyring, LedgerConnector } from 'knirvwallet-module';

// Create Ledger connector
const connector = new LedgerConnector();

// Create Ledger keyring
const ledgerKeyring = new LedgerKeyring(connector);

// Get accounts from Ledger
const accounts = await ledgerKeyring.getAccounts();

// Sign with Ledger
const signedTx = await ledgerKeyring.signTransaction(tx, accounts[0].address);
```

### Web3Auth Integration

```typescript
import { Web3AuthKeyring } from 'knirvwallet-module';

const web3AuthKeyring = new Web3AuthKeyring({
  clientId: 'your-web3auth-client-id',
  network: 'mainnet'
});

// Initialize Web3Auth
await web3AuthKeyring.init();

// Login with social provider
await web3AuthKeyring.login('google');

// Get accounts
const accounts = await web3AuthKeyring.getAccounts();
```

## Configuration

### Network Configuration

```typescript
import { setNetworkConfig } from 'knirvwallet-module';

setNetworkConfig({
  chainId: 'knirv-1',
  rpcUrl: 'https://rpc.knirv.network',
  addressPrefix: 'knirv',
  coinType: 118,
  hdPath: "m/44'/118'/0'/0/0"
});
```

### Security Configuration

```typescript
import { setSecurityConfig } from 'knirvwallet-module';

setSecurityConfig({
  encryptionAlgorithm: 'AES-256-GCM',
  keyDerivationRounds: 100000,
  saltLength: 32
});
```

## Usage in Different Environments

### Browser Extension
```typescript
// In background script
import { Wallet, HDWalletKeyring } from 'knirvwallet-module';

const wallet = new Wallet();
// Extension-specific implementation
```

### React Native App
```typescript
// In mobile app
import { Wallet, PrivateKeyKeyring } from 'knirvwallet-module';

const wallet = new Wallet();
// Mobile-specific implementation
```

### Web Application
```typescript
// In web app
import { Wallet, Web3AuthKeyring } from 'knirvwallet-module';

const wallet = new Wallet();
// Web-specific implementation
```

## Security Considerations

### Best Practices
- **Secure Storage**: Always encrypt sensitive data before storage
- **Memory Management**: Clear sensitive data from memory after use
- **Validation**: Validate all inputs and outputs
- **Randomness**: Use cryptographically secure random generation

### Security Features
- **Constant-time Operations**: Prevent timing attacks
- **Secure Defaults**: Safe configuration out of the box
- **Input Validation**: Comprehensive input sanitization
- **Error Handling**: Secure error messages without information leakage

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

**Note**: This module is the core foundation of the KNIRV Wallet ecosystem. For complete wallet applications, see the [browser extension](../knirvwallet-extension/README.md) and [mobile app](../../agentic-wallet/README.md).
