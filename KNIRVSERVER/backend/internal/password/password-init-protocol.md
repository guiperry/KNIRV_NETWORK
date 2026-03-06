# Password Initialization Protocol for KNIRV-SERVER

## Overview

KNIRV-SERVER implements encrypted key storage for sensitive configuration data using scrypt-based key derivation and AES-GCM encryption. This document outlines the implementation of password-protected key file creation and loading.

## Architecture

### Core Components

1. **Crypto Utilities** (`utils/crypto_utils.go`)
   - Scrypt key derivation
   - AES-GCM encryption/decryption
   - Salt generation

2. **Password Prompting** (`password_prompt.go`)
   - Secure password input (no echo)
   - Key file creation and loading
   - Protobuf-based data serialization

3. **Protobuf Definitions** (`proto/`)
   - `root_key.proto`: Defines encrypted key file structure
   - `bootnode_key.proto`: Defines bootnode-specific key structure

### Key File Structure

Encrypted key files use the following protobuf structure:

```protobuf
message RootKeyFileContentProto {
  string stripe_secret_key = 1;
  string stripe_webhook_secret = 2;
  string coinbase_api_key = 3;
  string coinbase_webhook_secret = 4;
  string root_private_key_hex = 5;
}

message EncryptedRootKeyFile {
  bytes encrypted_content = 1;
  bytes salt = 2;
}
```

## Implementation Details

### Password-Based Key Derivation

The system uses scrypt with the following parameters:
- N: 32768 (CPU/memory cost)
- R: 8 (block size)
- P: 1 (parallelization)
- Key length: 32 bytes
- Salt length: 32 bytes

### Encryption Flow

1. **Key Creation**:
   - Collect sensitive data from user
   - Prompt for password (twice for confirmation)
   - Generate random salt
   - Derive encryption key using scrypt
   - Encrypt data with AES-GCM
   - Store encrypted data + salt in protobuf format

2. **Key Loading**:
   - Load encrypted file
   - Extract salt
   - Prompt for password
   - Derive encryption key using scrypt
   - Decrypt data with AES-GCM
   - Parse decrypted protobuf content

### Security Considerations

- Passwords are read without echo using `golang.org/x/term`
- Files are created with restrictive permissions (0600)
- Salt is randomly generated for each key file
- AES-GCM provides authenticated encryption
- Scrypt provides memory-hard key derivation

## Usage

### Creating a Key File

```go
// Prompt user for key creation
err := PromptForKeyCreation("path/to/keyfile.key")
if err != nil {
    log.Fatal(err)
}
```

### Loading a Key File

```go
// Load and decrypt key file
content, err := LoadEncryptedKeyFile("path/to/keyfile.key", password)
if err != nil {
    log.Fatal(err)
}

// Access decrypted data
apiKey := content.CoinbaseApiKey
secretKey := content.RootPrivateKeyHex
```

## File Locations

- Default key file location: User-specific application data directory
- Permissions: 0600 (owner read/write only)
- Format: Binary protobuf (not human-readable)

## Error Handling

- Wrong password: Clear error message without revealing file contents
- File not found: Prompt for key creation
- Corrupted file: Clear error with recovery suggestions
- Permission issues: Informative error messages

## Future Enhancements

- Key file backup and recovery mechanisms
- Hardware security module (HSM) integration
- Multi-factor authentication support
- Key rotation capabilities