

---

**Source**: KNIRVROOT/docs/pending_implementation_plans/Password_Protection_Implementation_Plan.md

# Password Protection Implementation Plan

## Overview

This implementation plan outlines the steps to add password protection for the Root role in KNIRVCHAIN. The goal is to secure sensitive information (payment processor keys and the root private key) by encrypting them in a `.key` file that can only be accessed with a password.

## Phase 1: Define Data Structures

### 1.1 Create Protobuf Definition (root_key.proto)

First, create a new file named `root_key.proto` in the project root directory:

```protobuf
syntax = "proto3";

package main;

option go_package = "./;main";

message RootKeyFileContentProto {
  string stripe_secret_key = 1;
  string stripe_webhook_secret = 2;
  string coinbase_api_key = 3;
  string coinbase_webhook_secret = 4;
  string root_private_key_hex = 5;
  // Add any other sensitive root-specific config here
}
```

### 1.2 Compile the Protobuf Definition

Install the Protocol Buffers compiler (protoc) if you haven't already:

```bash
# For Ubuntu/Debian
sudo apt-get install protobuf-compiler

# For macOS with Homebrew
brew install protobuf

# For Windows, download from https://github.com/protocolbuffers/protobuf/releases
```

Install the Go protobuf plugin:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
```

Compile the protobuf definition to generate Go code:

```bash
# Run this from the project root directory
protoc --go_out=. --go_opt=paths=source_relative root_key.proto
```

This will generate a file named `root_key.pb.go` containing the Go structs and methods for the protobuf message.

### 1.3 Create Key File Structure (root_key_file.go)

Create a new file named `root_key_file.go` in the project root directory:

```go
// root_key_file.go
package main

import (
    "encoding/json"
    "google.golang.org/protobuf/proto"
)

// RootKeyFileContent holds the sensitive data for the Root role.
type RootKeyFileContent struct {
    // Payment Processor Keys
    StripeSecretKey       string
    StripeWebhookSecret   string
    CoinbaseAPIKey        string
    CoinbaseWebhookSecret string
    
    // Root Node Private Key (replace hardcoded constant)
    RootPrivateKeyHex string
}

// ToProto converts RootKeyFileContent to RootKeyFileContentProto.
func (c *RootKeyFileContent) ToProto() *RootKeyFileContentProto {
    return &RootKeyFileContentProto{
        StripeSecretKey:       c.StripeSecretKey,
        StripeWebhookSecret:   c.StripeWebhookSecret,
        CoinbaseApiKey:        c.CoinbaseAPIKey,
        CoinbaseWebhookSecret: c.CoinbaseWebhookSecret,
        RootPrivateKeyHex:     c.RootPrivateKeyHex,
    }
}

// RootKeyFileContentFromProto converts RootKeyFileContentProto to RootKeyFileContent.
func RootKeyFileContentFromProto(p *RootKeyFileContentProto) *RootKeyFileContent {
    return &RootKeyFileContent{
        StripeSecretKey:       p.StripeSecretKey,
        StripeWebhookSecret:   p.StripeWebhookSecret,
        CoinbaseAPIKey:        p.CoinbaseApiKey,
        CoinbaseWebhookSecret: p.CoinbaseWebhookSecret,
        RootPrivateKeyHex:     p.RootPrivateKeyHex,
    }
}

// EncryptedRootKeyFile represents the structure saved to the .key file.
type EncryptedRootKeyFile struct {
    Salt          []byte `json:"salt"`           // Salt for PBKDF
    N             int    `json:"n"`              // N parameter for Scrypt
    R             int    `json:"r"`              // R parameter for Scrypt
    P             int    `json:"p"`              // P parameter for Scrypt
    EncryptedData []byte `json:"encrypted_data"` // Encrypted protobuf bytes
}

// MarshalContent marshals the sensitive content to protobuf bytes.
func (c *RootKeyFileContent) MarshalContent() ([]byte, error) {
    protoMsg := c.ToProto()
    return proto.Marshal(protoMsg)
}

// UnmarshalContent unmarshals protobuf bytes into the sensitive content struct.
func (c *RootKeyFileContent) UnmarshalContent(data []byte) error {
    var protoMsg RootKeyFileContentProto
    if err := proto.Unmarshal(data, &protoMsg); err != nil {
        return err
    }
    *c = *RootKeyFileContentFromProto(&protoMsg)
    return nil
}
```

## Phase 2: Implement Cryptographic Functions

### 2.1 Add Password-Based Key Derivation (crypto_utils.go)

```go
// Add to crypto_utils.go
import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "fmt"
    "io"
    
    "golang.org/x/crypto/scrypt"
)

// Scrypt parameters
const (
    ScryptN = 32768 // CPU/memory cost factor (must be power of 2 > 1)
    ScryptR = 8     // Block size parameter
    ScryptP = 1     // Parallelization parameter
    SaltLen = 16    // Length of the salt in bytes
    KeyLen  = 32    // Desired key length in bytes (for AES-256)
)

// DeriveKeyFromPassword uses Scrypt to derive a key from a password and salt.
func DeriveKeyFromPassword(password, salt []byte, n, r, p, keyLen int) ([]byte, error) {
    key, err := scrypt.Key(password, salt, n, r, p, keyLen)
    if err != nil {
        return nil, fmt.Errorf("failed to derive key using scrypt: %w", err)
    }
    return key, nil
}

// GenerateSalt generates a random salt of a specified length.
func GenerateSalt(length int) ([]byte, error) {
    salt := make([]byte, length)
    if _, err := io.ReadFull(rand.Reader, salt); err != nil {
        return nil, fmt.Errorf("failed to generate random salt: %w", err)
    }
    return salt, nil
}

// Encrypt encrypts data using AES-GCM with the provided key.
func Encrypt(data, key []byte) ([]byte, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, fmt.Errorf("failed to create cipher block: %w", err)
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, fmt.Errorf("failed to create GCM: %w", err)
    }
    
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, fmt.Errorf("failed to generate nonce: %w", err)
    }
    
    ciphertext := gcm.Seal(nonce, nonce, data, nil)
    return ciphertext, nil
}

// Decrypt decrypts data using AES-GCM with the provided key.
func Decrypt(data, key []byte) ([]byte, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, fmt.Errorf("failed to create cipher block: %w", err)
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, fmt.Errorf("failed to create GCM: %w", err)
    }
    
    nonceSize := gcm.NonceSize()
    if len(data) < nonceSize {
        return nil, fmt.Errorf("ciphertext too short")
    }
    
    nonce, ciphertext := data[:nonceSize], data[nonceSize:]
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to decrypt: %w", err)
    }
    
    return plaintext, nil
}
```

## Phase 3: Create Encryption Tool

### 3.1 Install Fyne Dependencies

The encryption tool uses the Fyne GUI toolkit. Install the required dependencies:

```bash
# For Ubuntu/Debian
sudo apt-get install gcc libgl1-mesa-dev xorg-dev

# For macOS with Homebrew
brew install gcc

# For Windows
# Install MinGW-w64 or MSYS2 with the appropriate packages
```

Install the Fyne package:

```bash
go get fyne.io/fyne/v2
```

### 3.2 Implement Encryption GUI Script (encrypt_root_key.go)

Create a new file named `encrypt_root_key.go` in the project root directory:

```go
// encrypt_root_key.go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "os"
    "path/filepath"
    
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/dialog"
    "fyne.io/fyne/v2/widget"
)

func main() {
    a := app.New()
    w := a.NewWindow("KNIRVCHAIN Root Key Encryptor")
    w.Resize(fyne.NewSize(600, 400))
    
    // Input Fields for Sensitive Data
    stripeSecretEntry := widget.NewEntry()
    stripeSecretEntry.SetPlaceHolder("Stripe Secret Key")
    
    stripeWebhookSecretEntry := widget.NewEntry()
    stripeWebhookSecretEntry.SetPlaceHolder("Stripe Webhook Secret")
    
    coinbaseAPIKeyEntry := widget.NewEntry()
    coinbaseAPIKeyEntry.SetPlaceHolder("Coinbase API Key")
    
    coinbaseWebhookSecretEntry := widget.NewEntry()
    coinbaseWebhookSecretEntry.SetPlaceHolder("Coinbase Webhook Secret")
    
    rootPrivateKeyEntry := widget.NewEntry()
    rootPrivateKeyEntry.SetPlaceHolder("Root Private Key (Hex)")
    rootPrivateKeyEntry.Password = true // Mask the input
    
    // Password Input
    passwordEntry := widget.NewPasswordEntry()
    passwordEntry.SetPlaceHolder("Enter Password to Encrypt Key File")
    
    confirmPasswordEntry := widget.NewPasswordEntry()
    confirmPasswordEntry.SetPlaceHolder("Confirm Password")
    
    // Output File Path
    outputFileEntry := widget.NewEntry()
    outputFileEntry.SetPlaceHolder("Output .key file path (e.g., root.key)")
    
    // Suggest a default path
    defaultKeyPath, err := config.GetRootKeyPath()
    if err == nil {
        outputFileEntry.SetText(defaultKeyPath)
    } else {
        log.Printf("Warning: Could not determine default key path: %v", err)
        outputFileEntry.SetText("root.key")
    }
    
    // Encrypt Button
    encryptButton := widget.NewButton("Encrypt Key File", func() {
        password := []byte(passwordEntry.Text)
        confirmPassword := []byte(confirmPasswordEntry.Text)
        outputPath := outputFileEntry.Text
        
        if len(password) == 0 {
            dialog.ShowError(fmt.Errorf("password cannot be empty"), w)
            return
        }
        if !bytes.Equal(password, confirmPassword) {
            dialog.ShowError(fmt.Errorf("passwords do not match"), w)
            return
        }
        if outputPath == "" {
            dialog.ShowError(fmt.Errorf("output file path cannot be empty"), w)
            return
        }
        
        // Gather Sensitive Data
        content := RootKeyFileContent{
            StripeSecretKey:       stripeSecretEntry.Text,
            StripeWebhookSecret:   stripeWebhookSecretEntry.Text,
            CoinbaseAPIKey:        coinbaseAPIKeyEntry.Text,
            CoinbaseWebhookSecret: coinbaseWebhookSecretEntry.Text,
            RootPrivateKeyHex:     rootPrivateKeyEntry.Text,
        }
        
        // Marshal to protobuf
        contentBytes, err := content.MarshalContent()
        if err != nil {
            dialog.ShowError(fmt.Errorf("failed to marshal content to protobuf: %v", err), w)
            return
        }
        
        // Generate Salt and Derive Key
        salt, err := GenerateSalt(SaltLen)
        if err != nil {
            dialog.ShowError(fmt.Errorf("failed to generate salt: %v", err), w)
            return
        }
        
        encryptionKey, err := DeriveKeyFromPassword(password, salt, ScryptN, ScryptR, ScryptP, KeyLen)
        if err != nil {
            dialog.ShowError(fmt.Errorf("failed to derive encryption key: %v", err), w)
            return
        }
        
        // Encrypt Data
        encryptedData, err := Encrypt(contentBytes, encryptionKey)
        if err != nil {
            dialog.ShowError(fmt.Errorf("failed to encrypt data: %v", err), w)
            return
        }
        
        // Prepare File Content
        encryptedFileContent := EncryptedRootKeyFile{
            Salt:          salt,
            N:             ScryptN,
            R:             ScryptR,
            P:             ScryptP,
            EncryptedData: encryptedData,
        }
        
        fileBytes, err := json.MarshalIndent(encryptedFileContent, "", "  ")
        if err != nil {
            dialog.ShowError(fmt.Errorf("failed to marshal file content: %v", err), w)
            return
        }
        
        // Save File
        dir := filepath.Dir(outputPath)
        if err := os.MkdirAll(dir, 0700); err != nil {
            dialog.ShowError(fmt.Errorf("failed to create directory %s: %v", dir, err), w)
            return
        }
        if err := ioutil.WriteFile(outputPath, fileBytes, 0600); err != nil {
            dialog.ShowError(fmt.Errorf("failed to write key file %s: %v", outputPath, err), w)
            return
        }
        
        dialog.ShowInformation("Success", fmt.Sprintf("Root key file encrypted and saved to:\n%s\n\nIMPORTANT: Securely back up this file and remember your password!", outputPath), w)
    })
    
    // Layout
    contentForm := container.NewVBox(
        widget.NewLabel("Enter Sensitive Root Node Configuration:"),
        stripeSecretEntry,
        stripeWebhookSecretEntry,
        coinbaseAPIKeyEntry,
        coinbaseWebhookSecretEntry,
        rootPrivateKeyEntry,
        widget.NewSeparator(),
        widget.NewLabel("Set a Password for Encryption:"),
        passwordEntry,
        confirmPasswordEntry,
        widget.NewSeparator(),
        widget.NewLabel("Output File Path:"),
        outputFileEntry,
        encryptButton,
    )
    
    w.SetContent(container.NewVScroll(contentForm))
    w.ShowAndRun()
}
```

### 3.3 Build the Encryption Tool

To build the encryption tool as a standalone executable:

```bash
# Run this from the project root directory
go build -o encrypt_root_key encrypt_root_key.go root_key_file.go root_key.pb.go crypto_utils.go
```

This will create an executable named `encrypt_root_key` that you can run to create and encrypt your root key file.

## Phase 4: Update Configuration System

### 4.1 Add Root Key Path to Config (config/config.go)

```go
// Add to Config struct in config/config.go
type Config struct {
    // Existing fields...
    RootKeyPath string `json:"root_key_path,omitempty"` // Path to encrypted root key file
    // Other fields...
}
```

### 4.2 Add Helper Function for Root Key Path (config/paths.go)

```go
// Add to config/paths.go
// GetRootKeyPath returns the default path for the encrypted root key file.
func GetRootKeyPath() (string, error) {
    // Use the data directory for the Root role
    dataDir, err := GetDataDir(Root)
    if err != nil {
        return "", fmt.Errorf("failed to get data directory for root key: %w", err)
    }
    return filepath.Join(dataDir, "root.key"), nil
}
```

## Phase 5: Update Installer

### 5.1 Modify install.go for Root Role

Update the `install.go` file to verify the root key file during installation:

```go
// Modify install.go
if role == config.Root {
    fmt.Println("\n--- Root Key File Verification ---")
    fmt.Println("The Root node requires an encrypted 'root.key' file.")
    fmt.Println("This file should have been created using the encryption tool and placed in the application's data directory.")
    
    rootKeyPath, err := config.GetRootKeyPath()
    if err != nil {
        return fmt.Errorf("failed to determine default root key path: %w", err)
    }
    fmt.Printf("Looking for key file at: %s\n", rootKeyPath)
    
    encryptedFileBytes, err := ioutil.ReadFile(rootKeyPath)
    if err != nil {
        return fmt.Errorf("failed to read root key file %s: %w. Please ensure the file exists", rootKeyPath, err)
    }
    
    var encryptedFile EncryptedRootKeyFile
    if err := json.Unmarshal(encryptedFileBytes, &encryptedFile); err != nil {
        return fmt.Errorf("failed to parse root key file %s: %w. Ensure it's a valid encrypted key file", rootKeyPath, err)
    }
    
    fmt.Print("Enter password to verify root key file: ")
    passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
    fmt.Println()
    if err != nil {
        return fmt.Errorf("failed to read password: %w", err)
    }
    password := []byte(strings.TrimSpace(string(passwordBytes)))
    
    // Derive decryption key from password and salt/params
    decryptionKey, err := DeriveKeyFromPassword(password, encryptedFile.Salt, encryptedFile.N, encryptedFile.R, encryptedFile.P, KeyLen)
    if err != nil {
        return fmt.Errorf("failed to derive decryption key: %w", err)
    }
    
    // Decrypt the data
    decryptedBytes, err := Decrypt(encryptedFile.EncryptedData, decryptionKey)
    if err != nil {
        return fmt.Errorf("failed to decrypt root key file: %w. Incorrect password or corrupted file", err)
    }
    
    // Verify the decrypted data can be unmarshaled as protobuf
    var protoMsg RootKeyFileContentProto
    if err := proto.Unmarshal(decryptedBytes, &protoMsg); err != nil {
        return fmt.Errorf("failed to unmarshal decrypted content as protobuf: %w. Corrupted file content", err)
    }
    
    fmt.Println("Root key file successfully verified.")
    
    // Store the root key path in config
    configToSave.RootKeyPath = rootKeyPath
    configToSave.PaymentProcessor.Enabled = true
}
```

## Phase 6: Update Main Application

### 6.1 Modify main.go for Root Role

Update the `main.go` file to load and decrypt the root key file at startup:

```go
// Add to main.go
// Add flag for root key file path
rootKeyPathFlag := flag.String("root-key", "", "Path to the encrypted root key file (.key)")

// Declare variable to hold decrypted root keys
var rootKeyContent *RootKeyFileContent

// Root Role: Load and Decrypt Sensitive Keys
if nodeRole == config.Root {
    log.Println("Root role detected. Loading and decrypting sensitive keys...")
    
    // Determine the path to the root key file
    rootKeyPath := *rootKeyPathFlag
    if rootKeyPath == "" {
        // If flag not set, use the path from config or default
        if cfg.RootKeyPath != "" {
            rootKeyPath = cfg.RootKeyPath
        } else {
            defaultKeyPath, err := config.GetRootKeyPath()
            if err != nil {
                log.Fatalf("FATAL: Failed to determine default root key path: %v", err)
            }
            rootKeyPath = defaultKeyPath
        }
        log.Printf("Using root key path: %s", rootKeyPath)
    }
    
    // Read the encrypted key file
    encryptedFileBytes, err := ioutil.ReadFile(rootKeyPath)
    if err != nil {
        log.Fatalf("FATAL: Failed to read root key file %s: %v. Ensure the file exists and is accessible.", rootKeyPath, err)
    }
    
    // Unmarshal the encrypted file structure
    var encryptedFile EncryptedRootKeyFile
    if err := json.Unmarshal(encryptedFileBytes, &encryptedFile); err != nil {
        log.Fatalf("FATAL: Failed to parse root key file %s: %v. Ensure it's a valid encrypted key file.", rootKeyPath, err)
    }
    
    // Check for password in environment variable (for failover scenarios)
    var password []byte
    envPassword := os.Getenv("KNIRVCHAIN_CREATOR_PASSWORD")
    if envPassword != "" {
        log.Println("Using root key password from environment variable.")
        password = []byte(envPassword)
    } else {
        // Prompt for password securely
        fmt.Print("Enter password for root key file: ")
        passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
        fmt.Println() // Print newline after password input
        if err != nil {
            log.Fatalf("FATAL: Failed to read password: %v", err)
        }
        password = passwordBytes
    }
    
    // Derive decryption key from password and salt/params
    decryptionKey, err := DeriveKeyFromPassword(password, encryptedFile.Salt, encryptedFile.N, encryptedFile.R, encryptedFile.P, KeyLen)
    if err != nil {
        log.Fatalf("FATAL: Failed to derive decryption key: %v", err)
    }
    
    // Decrypt the sensitive data
    decryptedContentBytes, err := Decrypt(encryptedFile.EncryptedData, decryptionKey)
    if err != nil {
        log.Fatalf("FATAL: Failed to decrypt root key file: %v. Incorrect password or corrupted file.", err)
    }
    
    // Unmarshal the decrypted protobuf content
    var protoMsg RootKeyFileContentProto
    if err := proto.Unmarshal(decryptedContentBytes, &protoMsg); err != nil {
        log.Fatalf("FATAL: Failed to unmarshal decrypted content as protobuf: %v. Corrupted file content.", err)
    }
    
    // Convert from protobuf to our internal structure
    rootKeyContent = RootKeyFileContentFromProto(&protoMsg)
    
    log.Println("Root key file decrypted successfully.")
    
    // Override payment processor config with decrypted keys
    if rootKeyContent != nil {
        log.Println("Overriding payment processor config with decrypted keys...")
        cfg.PaymentProcessor.StripeSecretKey = rootKeyContent.StripeSecretKey
        cfg.PaymentProcessor.StripeWebhookSecret = rootKeyContent.StripeWebhookSecret
        cfg.PaymentProcessor.CoinbaseAPIKey = rootKeyContent.CoinbaseAPIKey
        cfg.PaymentProcessor.CoinbaseWebhookSecret = rootKeyContent.CoinbaseWebhookSecret
    }
    
    // Initialize Root wallet with decrypted private key
    if rootKeyContent != nil && rootKeyContent.RootPrivateKeyHex != "" {
        log.Println("Initializing root node wallet from decrypted key.")
        wallet = NewWalletFromPrivateKeyHex(rootKeyContent.RootPrivateKeyHex)
        log.Printf("Root node wallet initialized. Address: %s", wallet.GetAddress())
    }
}
```

### 6.2 Remove Hardcoded Private Key (constants.go)

```diff
// constants.go
const (
    MINING_DIFFICULTY      = 1
    MINING_REWARD          = 1200 * DECIMAL
    CURRENCY_NAME          = "NRN"
    DECIMAL                = 100
    BLOCKCHAIN_ADDRESS     = "KNIRVCHAINb53c1e30b8a578c091dd40612bfd1433991b4e09"
-   BLOCKCHAIN_PRIVATE_KEY = "0x1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a"
    ADDRESS_PREFIX         = "KNIRVCHAIN"
    WALLET_ENCRYPTION_KEY  = "KNIRVCHAIN_wallet_encryption_key_v1"
    ROOTCHAIN_URL          = "https://localhost:5000"
    // Other constants...
)
```

## Phase 7: Failover Mechanism

### 7.1 Environment Variable for Non-Interactive Failover

For automated failover scenarios, the application will check for the `KNIRVCHAIN_CREATOR_PASSWORD` environment variable. If set, it will use this password instead of prompting interactively.

```bash
# Example of how to set the environment variable for a failover scenario
export KNIRVCHAIN_CREATOR_PASSWORD="your-secure-password"
go run . --root
```

## Implementation Notes

1. **Security Considerations**:
   - The `.key` file should have restrictive permissions (0600)
   - The password should never be stored in plain text
   - For production, consider using a secure secrets management system
   - The protobuf serialization adds an extra layer of security and forward compatibility

2. **Backward Compatibility**:
   - The `BLOCKCHAIN_ADDRESS` constant remains for compatibility
   - The `BLOCKCHAIN_PRIVATE_KEY` constant is removed to prevent accidental use

3. **Failover Mechanism**:
   - Environment variable provides a non-interactive option for automated failover
   - No command-line flag for password to avoid exposure in process lists

4. **Installation Process**:
   - The installer verifies an existing `.key` file rather than creating one
   - The encryption tool is provided but can be removed after initial setup

5. **Protobuf Benefits**:
   - Forward and backward compatibility for the key file format
   - More efficient serialization than JSON
   - Better type safety and validation
   - Easier to extend with new fields in the future

## Testing Plan

1. Test the protobuf compilation:
   - Verify the `protoc` command successfully generates the Go code
   - Verify the generated code can be imported and used in the application

2. Test the encryption tool:
   - Verify it correctly encrypts and saves the `.key` file
   - Verify password validation works
   - Verify the protobuf serialization works correctly

3. Test the installer:
   - Verify it correctly detects and validates the `.key` file
   - Verify password validation works
   - Verify it correctly validates the protobuf format

4. Test the main application:
   - Verify it correctly loads and decrypts the `.key` file
   - Verify it correctly deserializes the protobuf content
   - Verify it correctly applies the decrypted keys to the configuration
   - Verify it correctly initializes the Root wallet with the decrypted private key

5. Test the failover mechanism:
   - Verify the application correctly uses the password from the environment variable
   - Verify the application falls back to interactive prompt if no environment variable is set

## Deployment Checklist

1. Install Protocol Buffers compiler and Go plugin
2. Create and compile the protobuf definition
3. Implement the key file structure and cryptographic functions
4. Build the encryption tool
5. Create and encrypt the root key file
6. Update the installer and main application
7. Remove the hardcoded private key from constants.go
8. Test all components thoroughly

##  How it works:

**The encryption process as detailed in the plan actually works like this:**

1.  Collect sensitive data (Stripe keys, Coinbase keys, Root Private Key)
2.  Serialize the sensitive data using Protocol Buffers (protobuf) - This converts the structured data into a compact binary format
3.  Generate a random salt for the password-based key derivation
4.  Derive an encryption key from the user's password and the salt using Scrypt
5.  Encrypt the protobuf-serialized data using AES-GCM with the derived key
6.  Create a container structure (EncryptedRootKeyFile) that includes:
    *   The salt
    *   Scrypt parameters (N, R, P)
    *   The encrypted protobuf data
    *   Serialize this container to JSON and save it to the .key file

**The decryption process is the reverse:**

1.  Read the .key file and parse the JSON to get the EncryptedRootKeyFile structure
2.  Extract the salt and Scrypt parameters
3.  Prompt for the password and derive the decryption key using the same Scrypt parameters
4.  Decrypt the encrypted data using AES-GCM
5.  Deserialize the decrypted data using Protocol Buffers to get the original sensitive information
6.  Use the decrypted sensitive data in the application

So to be clear:

We're not hashing the sensitive data - we're encrypting it
The protobuf serialization happens before encryption, not after
The .key file contains JSON with the salt, parameters, and encrypted protobuf bytes
This approach provides both security (through encryption) and forward compatibility (through protobuf).

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
