# Password Initialization Implementation Plan

## Phase 1: Configuration Refactoring & Constants

### Migrate Non-Sensitive Keys to constants.go

1. **Identify Non-Sensitive Variables**:
   - Move common configuration values from `.env` or `.key` files to constants
   - Target keys: `CEREBRAS_API_KEY`, `CEREBRAS_BASE_URL`, `GITHUB_TOKEN`, `GITHUB_PUBLIC_KEY`

2. **Implementation in utils/constants.go**:

```go
// Cerebras Configuration (Loaded by Viper from here or overridden by env/config file)
const DEFAULT_CEREBRAS_API_KEY = "your_default_or_public_cerebras_api_key_if_any" 
const DEFAULT_CEREBRAS_BASE_URL = "https://api.cerebras.ai/v1/chat/completions"

// GitHub Configuration (for updates, etc.)
const DEFAULT_GITHUB_TOKEN = "" // Should ideally be set via env for private repos during build/CI
const DEFAULT_GITHUB_PUBLIC_KEY_FOR_UPDATES = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAu5NhSQgUtPRmQwGDpHio
...your_public_key_for_verifying_updates...
-----END PUBLIC KEY-----`
```

3. **Viper Integration in config/viper_loader.go**:

```go
// Add to LoadConfigurationViper function
func LoadConfigurationViper(role Role, cliConfigPath string) (*Config, string, error) {
    // Existing code...
    
    // Set default values from constants if not found in config
    if cfg.Chromem.CerebrasConfig.APIKey == "" {
        cfg.Chromem.CerebrasConfig.APIKey = utils.DEFAULT_CEREBRAS_API_KEY
    }
    if cfg.Chromem.CerebrasConfig.BaseURL == "" {
        cfg.Chromem.CerebrasConfig.BaseURL = utils.DEFAULT_CEREBRAS_BASE_URL
    }
    if cfg.GitHubToken == "" {
        cfg.GitHubToken = utils.DEFAULT_GITHUB_TOKEN
    }
    if cfg.GitHubPublicKey == "" {
        cfg.GitHubPublicKey = utils.DEFAULT_GITHUB_PUBLIC_KEY_FOR_UPDATES
    }
    
    // Existing code...
}
```

### Update root_key.proto

1. **Remove Non-Sensitive Fields**:

```protobuf
syntax = "proto3";

package AGENTCHAIN;

option go_package = "./;proto";

message RootKeyFileContentProto {
  string stripe_secret_key = 1;
  string stripe_webhook_secret = 2;
  string coinbase_api_key = 3;
  string coinbase_webhook_secret = 4;
  string root_private_key_hex = 5;
  // Add any other sensitive root-specific config here
}

message EncryptedRootKeyFile {
  bytes encrypted_content = 1;
  bytes salt = 2;
  bytes nonce = 3;
}
```

2. **Regenerate Protobuf Files**:

```bash
protoc --go_out=. --go_opt=paths=source_relative proto/root_key.proto
```

### Clarify BLOCKCHAIN_KEY and ROOT_PRIVATE_KEY

1. **Update utils/constants.go**:

```go
// Old
// const BLOCKCHAIN_KEY = "ROOT_PRIVATE_KEY"

// New
const ROOT_IDENTITY_KEY_NAME_IN_FILE = "ROOT_PRIVATE_KEY"
const BLOCKCHAIN_PRIVATE_KEY = "0x1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a"
```

## Phase 2: Root Role Initialization and Password Flow

### Startup Flow in main.go

1. **Check for Unencrypted .key File**:

```go
func main() {
    // Parse command line flags
    rootFlag := flag.Bool("root", false, "Run as Root node")
    // Other flags...
    flag.Parse()
    
    // Determine node role from flags
    var nodeRole config.Role
    if *rootFlag {
        nodeRole = config.Root
    }
    // Other role determination...
    
    // Root-specific initialization
    if nodeRole == config.Root {
        // Step 2.1: Check for unencrypted .key file
        openKeyPath := filepath.Join(".", ".key")
        forceRootInstall := false
        rootPrivateKey := ""
        
        if fileExists(openKeyPath) {
            // Parse .key file for ROOT_PRIVATE_KEY
            keyData, err := parseKeyFile(openKeyPath)
            if err == nil && keyData[utils.ROOT_IDENTITY_KEY_NAME_IN_FILE] != "" {
                rootPrivateKey = keyData[utils.ROOT_IDENTITY_KEY_NAME_IN_FILE]
                
                // Step 2.2: Key matching & auto-decryption
                if rootPrivateKey == utils.BLOCKCHAIN_PRIVATE_KEY {
                    // Try to load embedded root.key
                    if len(config.EmbeddedRootKeyData) > 0 {
                        var encryptedFile pb.EncryptedRootKeyFile
                        if err := proto.Unmarshal(config.EmbeddedRootKeyData, &encryptedFile); err == nil {
                            // Derive key from BLOCKCHAIN_PRIVATE_KEY
                            derivedKey, err := utils.DeriveKeyFromPassword(
                                []byte(utils.BLOCKCHAIN_PRIVATE_KEY),
                                encryptedFile.Salt,
                                utils.ScryptN,
                                utils.ScryptR,
                                utils.ScryptP,
                                utils.KeyLen,
                            )
                            
                            if err == nil {
                                // Decrypt data
                                decryptedData, err := utils.Decrypt(encryptedFile.EncryptedContent, derivedKey)
                                if err == nil {
                                    // Unmarshal content
                                    var content pb.RootKeyFileContentProto
                                    if err := proto.Unmarshal(decryptedData, &content); err == nil {
                                        // Set environment variables
                                        os.Setenv("STRIPE_SECRET_KEY", content.StripeSecretKey)
                                        os.Setenv("STRIPE_WEBHOOK_SECRET", content.StripeWebhookSecret)
                                        os.Setenv("COINBASE_API_KEY", content.CoinbaseApiKey)
                                        os.Setenv("COINBASE_WEBHOOK_SECRET", content.CoinbaseWebhookSecret)
                                        os.Setenv("ROOT_PRIVATE_KEY_HEX", content.RootPrivateKeyHex)
                                        
                                        log.Println("Root mode started with pre-configured keys (password-less).")
                                        
                                        // Step 2.4: Prompt for reconfiguration
                                        reader := bufio.NewReader(os.Stdin)
                                        fmt.Print("Root keys are loaded. Do you want to re-configure/update them? (y/N): ")
                                        answer, _ := reader.ReadString('\n')
                                        answer = strings.TrimSpace(strings.ToLower(answer))
                                        if answer == "y" || answer == "yes" {
                                            forceRootInstall = true
                                        }
                                    }
                                } else {
                                    log.Println("Pre-configured key match, but embedded root.key decryption failed. Proceeding to manual password prompt.")
                                }
                            }
                        }
                    }
                } else {
                    log.Printf("Warning: ROOT_PRIVATE_KEY in .key file does not match expected value.")
                }
            }
        }
        
        // Step 2.3: Load user-specific root.key or trigger Peer Node
        if os.Getenv("ROOT_PRIVATE_KEY_HEX") == "" {
            userRootKeyPath, err := config.GetRootKeyPath()
            if err == nil && fileExists(userRootKeyPath) {
                // Prompt for password and load key file
                success := false
                maxRetries := 3
                retries := 0
                
                for !success && retries < maxRetries {
                    password, err := PromptForPassword("Enter password to decrypt root.key: ")
                    if err == nil {
                        content, err := LoadRootKeyFile(userRootKeyPath, password)
                        if err == nil {
                            // Set environment variables
                            os.Setenv("STRIPE_SECRET_KEY", content.StripeSecretKey)
                            os.Setenv("STRIPE_WEBHOOK_SECRET", content.StripeWebhookSecret)
                            os.Setenv("COINBASE_API_KEY", content.CoinbaseApiKey)
                            os.Setenv("COINBASE_WEBHOOK_SECRET", content.CoinbaseWebhookSecret)
                            os.Setenv("ROOT_PRIVATE_KEY_HEX", content.RootPrivateKeyHex)
                            
                            success = true
                            
                            // Step 2.4: Prompt for reconfiguration
                            reader := bufio.NewReader(os.Stdin)
                            fmt.Print("Root keys are loaded. Do you want to re-configure/update them? (y/N): ")
                            answer, _ := reader.ReadString('\n')
                            answer = strings.TrimSpace(strings.ToLower(answer))
                            if answer == "y" || answer == "yes" {
                                forceRootInstall = true
                            }
                        } else {
                            log.Printf("Failed to decrypt root.key: %v", err)
                            retries++
                        }
                    } else {
                        log.Printf("Failed to read password: %v", err)
                        retries++
                    }
                }
                
                if !success {
                    reader := bufio.NewReader(os.Stdin)
                    fmt.Print("Failed to decrypt existing root.key. Do you want to initialize a Peer Node (y/N/r)? ")
                    answer, _ := reader.ReadString('\n')
                    answer = strings.TrimSpace(strings.ToLower(answer))
                    
                    if answer == "r" {
                        forceRootInstall = true
                    } else if answer == "y" || answer == "yes" {
                        nodeRole = config.Peer
                    } else {
                        log.Fatal("Exiting due to root.key decryption failure.")
                    }
                }
            } else {
                log.Println("Root key file not found. A Key file is required to set up Root secrets installation")
                forceRootInstall = false
                nodeRole = config.Client
            }
        }
        
        // Step 2.5: Trigger installation if needed
        // Load configuration
        cfg, configPath, err := config.LoadConfigurationViper(nodeRole, *configPath)
        if err != nil {
            log.Fatalf("Failed to load configuration: %v", err)
        }
        
        if (!cfg.InstallComplete && !*skipInstall) || (nodeRole == config.Root && forceRootInstall) {
            // Trigger installation
            // ...
        }
    }
    
    // Continue with normal startup...
}
```

### Installation Process for Root Role

1. **Pre-fill Values for Update**:

```go
// In install.go
func Install(configPath string, IsBootnode bool, role config.Role, nonInteractive bool, walletPath string) (*config.Config, error) {
    // Existing code...
    
    // For Root role, handle key creation/update
    if role == config.Root {
        // Check if we have existing decrypted content to pre-fill
        var existingContent *pb.RootKeyFileContentProto
        
        // If environment variables are set from previous decryption
        if os.Getenv("ROOT_PRIVATE_KEY_HEX") != "" {
            existingContent = &pb.RootKeyFileContentProto{
                StripeSecretKey:       os.Getenv("STRIPE_SECRET_KEY"),
                StripeWebhookSecret:   os.Getenv("STRIPE_WEBHOOK_SECRET"),
                CoinbaseApiKey:        os.Getenv("COINBASE_API_KEY"),
                CoinbaseWebhookSecret: os.Getenv("COINBASE_WEBHOOK_SECRET"),
                RootPrivateKeyHex:     os.Getenv("ROOT_PRIVATE_KEY_HEX"),
            }
        }
        
        // Get path for root.key
        rootKeyPath, err := config.GetRootKeyPath()
        if err != nil {
            return nil, fmt.Errorf("failed to get root key path: %w", err)
        }
        
        // Prompt for root key creation/update with pre-filled values
        if err := PromptForRootKeyCreation(rootKeyPath, existingContent); err != nil {
            return nil, fmt.Errorf("failed to create/update root key file: %w", err)
        }
    }
    
    // Existing code...
}
```

2. **Modify PromptForRootKeyCreation**:

```go
// In password_prompt.go
func PromptForRootKeyCreation(keyFilePath string, currentContent *pb.RootKeyFileContentProto) error {
    if currentContent == nil {
        fmt.Println("No Root key file found. You need to create one to continue.")
    } else {
        fmt.Println("Updating existing Root key file.")
    }
    fmt.Println("This will store your sensitive Root configuration securely.")
    fmt.Println()
    
    reader := bufio.NewReader(os.Stdin)
    
    // Pre-fill values if available
    stripeSecretKey := ""
    stripeWebhookSecret := ""
    coinbaseAPIKey := ""
    coinbaseWebhookSecret := ""
    rootPrivateKeyHex := utils.BLOCKCHAIN_PRIVATE_KEY // Default to the fixed value
    
    if currentContent != nil {
        stripeSecretKey = currentContent.StripeSecretKey
        stripeWebhookSecret = currentContent.StripeWebhookSecret
        coinbaseAPIKey = currentContent.CoinbaseApiKey
        coinbaseWebhookSecret = currentContent.CoinbaseWebhookSecret
        rootPrivateKeyHex = currentContent.RootPrivateKeyHex
    }
    
    // Prompt for sensitive data with pre-filled values
    if stripeSecretKey != "" {
        fmt.Printf("Enter Stripe Secret Key (current: %s): ", maskString(stripeSecretKey))
    } else {
        fmt.Print("Enter Stripe Secret Key (or leave empty): ")
    }
    input, _ := reader.ReadString('\n')
    input = strings.TrimSpace(input)
    if input != "" {
        stripeSecretKey = input
    }
    
    if stripeWebhookSecret != "" {
        fmt.Printf("Enter Stripe Webhook Secret (current: %s): ", maskString(stripeWebhookSecret))
    } else {
        fmt.Print("Enter Stripe Webhook Secret (or leave empty): ")
    }
    input, _ = reader.ReadString('\n')
    input = strings.TrimSpace(input)
    if input != "" {
        stripeWebhookSecret = input
    }
    
    if coinbaseAPIKey != "" {
        fmt.Printf("Enter Coinbase API Key (current: %s): ", maskString(coinbaseAPIKey))
    } else {
        fmt.Print("Enter Coinbase API Key (or leave empty): ")
    }
    input, _ = reader.ReadString('\n')
    input = strings.TrimSpace(input)
    if input != "" {
        coinbaseAPIKey = input
    }
    
    if coinbaseWebhookSecret != "" {
        fmt.Printf("Enter Coinbase Webhook Secret (current: %s): ", maskString(coinbaseWebhookSecret))
    } else {
        fmt.Print("Enter Coinbase Webhook Secret (or leave empty): ")
    }
    input, _ = reader.ReadString('\n')
    input = strings.TrimSpace(input)
    if input != "" {
        coinbaseWebhookSecret = input
    }
    
    // For Root Private Key, enforce that it matches the expected value
    fmt.Printf("Root Private Key (must match blockchain identity): %s\n", utils.BLOCKCHAIN_PRIVATE_KEY)
    fmt.Print("Press Enter to confirm or type 'override' to use a different key: ")
    input, _ = reader.ReadString('\n')
    input = strings.TrimSpace(input)
    if input == "override" {
        fmt.Print("Enter Root Private Key (hex, required): ")
        input, _ = reader.ReadString('\n')
        input = strings.TrimSpace(input)
        if input != "" {
            rootPrivateKeyHex = input
        }
        
        if rootPrivateKeyHex != utils.BLOCKCHAIN_PRIVATE_KEY {
            fmt.Println("WARNING: Using a non-standard Root Private Key may cause blockchain identity issues.")
            fmt.Print("Are you sure you want to continue? (y/N): ")
            confirm, _ := reader.ReadString('\n')
            confirm = strings.TrimSpace(strings.ToLower(confirm))
            if confirm != "y" && confirm != "yes" {
                rootPrivateKeyHex = utils.BLOCKCHAIN_PRIVATE_KEY
                fmt.Println("Reverting to standard Root Private Key.")
            }
        }
    }
    
    if rootPrivateKeyHex == "" {
        return fmt.Errorf("root private key is required")
    }
    
    // Prompt for password
    password, err := PromptForPassword("Enter password to encrypt key file: ")
    if err != nil {
        return err
    }
    
    confirmPassword, err := PromptForPassword("Confirm password: ")
    if err != nil {
        return err
    }
    
    if string(password) != string(confirmPassword) {
        return fmt.Errorf("passwords do not match")
    }
    
    // Create content
    content := &pb.RootKeyFileContentProto{
        StripeSecretKey:       stripeSecretKey,
        StripeWebhookSecret:   stripeWebhookSecret,
        CoinbaseApiKey:        coinbaseAPIKey,
        CoinbaseWebhookSecret: coinbaseWebhookSecret,
        RootPrivateKeyHex:     rootPrivateKeyHex,
    }
    
    // Create key file
    if err := CreateRootKeyFile(content, password, keyFilePath); err != nil {
        return err
    }
    
    fmt.Printf("Root key file created successfully at %s\n", keyFilePath)
    fmt.Println("IMPORTANT: Securely back up this file and remember your password!")
    
    return nil
}

// Helper function to mask sensitive strings
func maskString(s string) string {
    if len(s) <= 4 {
        return "****"
    }
    return s[:2] + "****" + s[len(s)-2:]
}
```

## Phase 3: Bootnode Role Initialization and Password Flow

### Bootnode Startup Flow

```go
// In main.go
func main() {
    // Parse command line flags
    bootnodeFlag := flag.Bool("bootnode", false, "Run as Bootnode")
    // Other flags...
    flag.Parse()
    
    // Determine node role from flags
    var nodeRole config.Role
    if *bootnodeFlag {
        nodeRole = config.RoleBootnode
    }
    // Other role determination...
    
    // Bootnode-specific initialization
    if nodeRole == config.RoleBootnode {
        // Step 3.1: Check for unencrypted .key file
        openKeyPath := filepath.Join(".", ".key")
        bootnodeEnabledByOpenKey := false
        masterWalletKey := ""
        forceBootnodeInstall := false
        
        if fileExists(openKeyPath) {
            // Parse .key file for MASTER_WALLET_KEY
            keyData, err := parseKeyFile(openKeyPath)
            if err == nil && keyData[utils.MASTER_WALLET_KEY] != "" {
                masterWalletKey = keyData[utils.MASTER_WALLET_KEY]
                bootnodeEnabledByOpenKey = true
            }
        }
        
        // Step 3.3: Load user-specific bootnode.key or trigger install
        bootnodeKeyPath, err := config.GetBootnodeKeyPath()
        if err == nil && fileExists(bootnodeKeyPath) {
            // Prompt for password and load key file
            success := false
            maxRetries := 3
            retries := 0
            
            for !success && retries < maxRetries {
                password, err := PromptForPassword("Enter password to decrypt bootnode.key: ")
                if err == nil {
                    content, err := LoadBootnodeKeyFile(bootnodeKeyPath, password)
                    if err == nil {
                        // Set environment variables
                        os.Setenv("MASTER_WALLET_KEY", content.MasterWalletKey)
                        
                        success = true
                        
                        // Prompt for reconfiguration
                        reader := bufio.NewReader(os.Stdin)
                        fmt.Print("Bootnode keys are loaded. Do you want to re-configure/update them? (y/N): ")
                        answer, _ := reader.ReadString('\n')
                        answer = strings.TrimSpace(strings.ToLower(answer))
                        if answer == "y" || answer == "yes" {
                            forceBootnodeInstall = true
                        }
                    } else {
                        log.Printf("Failed to decrypt bootnode.key: %v", err)
                        retries++
                    }
                } else {
                    log.Printf("Failed to read password: %v", err)
                    retries++
                }
            }
            
            if !success {
                log.Println("Failed to decrypt bootnode.key after multiple attempts.")
                if bootnodeEnabledByOpenKey {
                    log.Println("Bootnode master key found in open .key. Installation will proceed to set password and save.")
                    forceBootnodeInstall = true
                } else {
                    log.Println("No bootnode key available. Switching to Peer role.")
                    nodeRole = config.Peer
                }
            }
        } else {
            // No bootnode.key found
            if bootnodeEnabledByOpenKey {
                log.Println("Bootnode master key found in open .key. Installation will proceed to set password and save.")
                forceBootnodeInstall = true
            } else {
                log.Println("Bootnode key file not found. .key file required for bootnode installation.")
                log.Println("Switching to Peer role.")
                nodeRole = config.Peer
            }
        }
        
        // Load configuration
        cfg, configPath, err := config.LoadConfigurationViper(nodeRole, *configPath)
        if err != nil {
            log.Fatalf("Failed to load configuration: %v", err)
        }
        
        // Trigger installation if needed
        if (!cfg.InstallComplete && !*skipInstall) || (nodeRole == config.RoleBootnode && forceBootnodeInstall) {
            // Trigger installation
            // ...
        }
    }
    
    // Continue with normal startup...
}
```

### Bootnode Key Management

1. **Create Bootnode Key Proto**:

```protobuf
// In proto/bootnode_key.proto
syntax = "proto3";

package AGENTCHAIN;

option go_package = "./;proto";

message BootnodeKeyFileContentProto {
  string master_wallet_key = 1;
  // Add any other sensitive bootnode-specific config here
}

message EncryptedBootnodeKeyFile {
  bytes encrypted_content = 1;
  bytes salt = 2;
  bytes nonce = 3;
}
```

2. **Implement Bootnode Key Functions**:

```go
// In config/bootnode_key_config.go
package config

import (
    "fmt"
    "io/ioutil"
    "os"
    "path/filepath"

    pb "AGENTCHAIN/proto"

    "google.golang.org/protobuf/proto"
)

// GetBootnodeKeyPath returns the default path for the Bootnode key file.
func GetBootnodeKeyPath() (string, error) {
    configDir, err := GetDataDir(RoleBootnode)
    if err != nil {
        return "", fmt.Errorf("failed to get bootnode data directory: %w", err)
    }

    return filepath.Join(configDir, "bootnode.key"), nil
}

// LoadEncryptedBootnodeKeyFile loads the encrypted Bootnode key file from the specified path.
func LoadEncryptedBootnodeKeyFile(path string) (*pb.EncryptedBootnodeKeyFile, error) {
    // Check if file exists
    if _, err := os.Stat(path); os.IsNotExist(err) {
        return nil, fmt.Errorf("bootnode key file does not exist at %s", path)
    }

    // Read file
    data, err := ioutil.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read bootnode key file: %w", err)
    }

    // Unmarshal protobuf
    var keyFile pb.EncryptedBootnodeKeyFile
    if err := proto.Unmarshal(data, &keyFile); err != nil {
        return nil, fmt.Errorf("failed to parse bootnode key file: %w", err)
    }

    return &keyFile, nil
}
```

3. **Implement PromptForBootnodeKeyCreation**:

```go
// In password_prompt.go
func PromptForBootnodeKeyCreation(keyFilePath string, currentMasterWalletKey string) error {
    if currentMasterWalletKey == "" {
        fmt.Println("No Bootnode key file found. You need to create one to continue.")
    } else {
        fmt.Println("Updating existing Bootnode key file.")
    }
    fmt.Println("This will store your sensitive Bootnode configuration securely.")
    fmt.Println()
    
    reader := bufio.NewReader(os.Stdin)
    
    // Prompt for master wallet key with pre-filled value if available
    masterWalletKey := currentMasterWalletKey
    if masterWalletKey != "" {
        fmt.Printf("Enter Master Wallet Key (current: %s): ", maskString(masterWalletKey))
    } else {
        fmt.Print("Enter Master Wallet Key (required): ")
    }
    input, _ := reader.ReadString('\n')
    input = strings.TrimSpace(input)
    if input != "" {
        masterWalletKey = input
    }
    
    if masterWalletKey == "" {
        return fmt.Errorf("master wallet key is required")
    }
    
    // Prompt for password
    password, err := PromptForPassword("Enter password to encrypt key file: ")
    if err != nil {
        return err
    }
    
    confirmPassword, err := PromptForPassword("Confirm password: ")
    if err != nil {
        return err
    }
    
    if string(password) != string(confirmPassword) {
        return fmt.Errorf("passwords do not match")
    }
    
    // Create content
    content := &pb.BootnodeKeyFileContentProto{
        MasterWalletKey: masterWalletKey,
    }
    
    // Create key file
    if err := CreateBootnodeKeyFile(content, password, keyFilePath); err != nil {
        return err
    }
    
    fmt.Printf("Bootnode key file created successfully at %s\n", keyFilePath)
    fmt.Println("IMPORTANT: Securely back up this file and remember your password!")
    
    return nil
}

// CreateBootnodeKeyFile creates a new encrypted bootnode key file
func CreateBootnodeKeyFile(content *pb.BootnodeKeyFileContentProto, password []byte, outputPath string) error {
    // Marshal content to protobuf
    contentBytes, err := proto.Marshal(content)
    if err != nil {
        return fmt.Errorf("failed to marshal content: %w", err)
    }
    
    // Generate salt
    salt, err := utils.GenerateSalt(utils.SaltLen)
    if err != nil {
        return fmt.Errorf("failed to generate salt: %w", err)
    }
    
    // Derive encryption key
    key, err := utils.DeriveKeyFromPassword(password, salt, utils.ScryptN, utils.ScryptR, utils.ScryptP, utils.KeyLen)
    if err != nil {
        return fmt.Errorf("failed to derive key: %w", err)
    }
    
    // Encrypt data
    encryptedData, err := utils.Encrypt(contentBytes, key)
    if err != nil {
        return fmt.Errorf("failed to encrypt data: %w", err)
    }
    
    // Create encrypted file structure
    encryptedFile := &pb.EncryptedBootnodeKeyFile{
        EncryptedContent: encryptedData,
        Salt:             salt,
    }
    
    // Marshal to protobuf
    fileBytes, err := proto.Marshal(encryptedFile)
    if err != nil {
        return fmt.Errorf("failed to marshal file content: %w", err)
    }
    
    // Create directory if it doesn't exist
    dir := filepath.Dir(outputPath)
    if err := os.MkdirAll(dir, 0700); err != nil {
        return fmt.Errorf("failed to create directory %s: %w", dir, err)
    }
    
    // Write file
    if err := ioutil.WriteFile(outputPath, fileBytes, 0600); err != nil {
        return fmt.Errorf("failed to write key file: %w", err)
    }
    
    return nil
}

// LoadBootnodeKeyFile loads and decrypts the Bootnode key file
func LoadBootnodeKeyFile(keyFilePath string, password []byte) (*pb.BootnodeKeyFileContentProto, error) {
    // Load encrypted key file
    encryptedFile, err := config.LoadEncryptedBootnodeKeyFile(keyFilePath)
    if err != nil {
        return nil, fmt.Errorf("failed to load bootnode key file: %w", err)
    }
    
    // Derive encryption key from password
    key, err := utils.DeriveKeyFromPassword(
        password,
        encryptedFile.Salt,
        utils.ScryptN,
        utils.ScryptR,
        utils.ScryptP,
        utils.KeyLen,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to derive key from password: %w", err)
    }
    
    // Decrypt data
    decryptedData, err := utils.Decrypt(encryptedFile.EncryptedContent, key)
    if err != nil {
        return nil, fmt.Errorf("failed to decrypt bootnode key file (incorrect password?): %w", err)
    }
    
    // Unmarshal content
    var content pb.BootnodeKeyFileContentProto
    if err := proto.Unmarshal(decryptedData, &content); err != nil {
        return nil, fmt.Errorf("failed to unmarshal bootnode key file content: %w", err)
    }
    
    return &content, nil
}
```

## Phase 4: General Role Selection and Non-Root/Non-Bootnode Behavior

### Role Selection Prompt

```go
// In main.go
func main() {
    // Parse command line flags
    preRootFlag := flag.Bool("root", false, "Run as Root node")
    preBootnodeFlag := flag.Bool("bootnode", false, "Run as Bootnode")
    prePeerFlag := flag.Bool("dev", false, "Run as Peer node")
    preClientOnlyFlag := flag.Bool("client-only", false, "Run as Client only")
    preRoleFlag := flag.String("role", "", "Node role (root, bootnode, dev, client)")
    // Other flags...
    flag.Parse()
    
    // Determine node role from flags
    var nodeRole config.Role
    
    // Check if any role flag is set
    roleSpecified := *rootFlag || *bootnodeFlag || *devFlag || *clientOnlyFlag || *roleFlag != ""
    
    if !roleSpecified {
        // No role flag set, prompt user
        fmt.Println("Please select a role to run:")
        fmt.Println("1. Root")
        fmt.Println("2. Bootnode")
        fmt.Println("3. Peer")
        fmt.Println("4. Client")
        
        reader := bufio.NewReader(os.Stdin)
        fmt.Print("Enter selection (1-4): ")
        input, _ := reader.ReadString('\n')
        input = strings.TrimSpace(input)
        
        switch input {
        case "1":
            nodeRole = config.Root
        case "2":
            nodeRole = config.RoleBootnode
        case "3":
            nodeRole = config.Peer
        case "4":
            nodeRole = config.Client
        default:
            fmt.Println("Invalid selection. Defaulting to Client role.")
            nodeRole = config.Client
        }
    } else {
        // Use role from flags (existing logic)
        if *rootFlag {
            nodeRole = config.Root
        } else if *bootnodeFlag {
            nodeRole = config.RoleBootnode
        } else if *devFlag {
            nodeRole = config.Peer
        } else if *clientOnlyFlag {
            nodeRole = config.Client
        } else if *roleFlag != "" {
            // Parse role from string
            switch strings.ToLower(*roleFlag) {
            case "root":
                nodeRole = config.Root
            case "bootnode":
                nodeRole = config.RoleBootnode
            case "dev":
                nodeRole = config.Peer
            case "client":
                nodeRole = config.Client
            default:
                fmt.Printf("Unknown role: %s. Defaulting to Client.\n", *roleFlag)
                nodeRole = config.Client
            }
        }
    }
    
    // Continue with role-specific initialization...
}
```

### Non-Root/Non-Bootnode Roles

For Peer and Client roles, the implementation is simpler as they don't need to handle encrypted key files:

```go
// In main.go
func main() {
    // Role determination code...
    
    // For Peer and Client roles, just load configuration normally
    if nodeRole == config.Peer || nodeRole == config.Client {
        // Load .env file for general environment variables
        if err := godotenv.Load(".env"); err != nil {
            log.Printf("Warning: No .env file found or error loading it: %v", err)
        }
        
        // Load configuration
        cfg, configPath, err := config.LoadConfigurationViper(nodeRole, *configPath)
        if err != nil {
            log.Fatalf("Failed to load configuration: %v", err)
        }
        
        // Continue with normal startup...
    }
    
    // Continue with role-specific initialization...
}
```

## Phase 5: Embedding Pre-configured root.key (PHASE COMPLETED!)

### Create Default Encrypted root.key

1. **Create and Encrypt Default root.key**:

```go
// In a separate tool or script (encrypt_root_key.go)
package main

import (
    "flag"
    "fmt"
    "log"
    "os"

    pb "AGENTCHAIN/proto"
    "AGENTCHAIN/utils"

    "google.golang.org/protobuf/proto"
)

func main() {
    outputPath := flag.String("output", "config/embedded/default_root.key", "Output path for encrypted key file")
    flag.Parse()

    // Create default content
    content := &pb.RootKeyFileContentProto{
        StripeSecretKey:       "sk_test_default",
        StripeWebhookSecret:   "whsec_default",
        CoinbaseApiKey:        "coinbase_api_default",
        CoinbaseWebhookSecret: "coinbase_webhook_default",
        RootPrivateKeyHex:     utils.BLOCKCHAIN_PRIVATE_KEY,
    }

    // Marshal content to protobuf
    contentBytes, err := proto.Marshal(content)
    if err != nil {
        log.Fatalf("Failed to marshal content: %v", err)
    }

    // Generate salt
    salt, err := utils.GenerateSalt(utils.SaltLen)
    if err != nil {
        log.Fatalf("Failed to generate salt: %v", err)
    }

    // Use BLOCKCHAIN_PRIVATE_KEY as password
    password := []byte(utils.BLOCKCHAIN_PRIVATE_KEY)

    // Derive encryption key
    key, err := utils.DeriveKeyFromPassword(password, salt, utils.ScryptN, utils.ScryptR, utils.ScryptP, utils.KeyLen)
    if err != nil {
        log.Fatalf("Failed to derive key: %v", err)
    }

    // Encrypt data
    encryptedData, err := utils.Encrypt(contentBytes, key)
    if err != nil {
        log.Fatalf("Failed to encrypt data: %v", err)
    }

    // Create encrypted file structure
    encryptedFile := &pb.EncryptedRootKeyFile{
        EncryptedContent: encryptedData,
        Salt:             salt,
    }

    // Marshal to protobuf
    fileBytes, err := proto.Marshal(encryptedFile)
    if err != nil {
        log.Fatalf("Failed to marshal file content: %v", err)
    }

    // Create directory if it doesn't exist
    dir := filepath.Dir(*outputPath)
    if err := os.MkdirAll(dir, 0700); err != nil {
        log.Fatalf("Failed to create directory %s: %v", dir, err)
    }

    // Write file
    if err := ioutil.WriteFile(*outputPath, fileBytes, 0600); err != nil {
        log.Fatalf("Failed to write key file: %v", err)
    }

    fmt.Printf("Default encrypted root.key created at %s\n", *outputPath)
}
```

2. **Embed the File**:

```go
// In config/embedded_assets.go
package config

import (
    _ "embed"
)

//go:embed embedded/default_root.key
var EmbeddedRootKeyData []byte
```

## Phase 6: Code Implementation Details & Refinements

### Helper Functions

```go
// In main.go or utils package

// fileExists checks if a file exists and is not a directory
func fileExists(filename string) bool {
    info, err := os.Stat(filename)
    if os.IsNotExist(err) {
        return false
    }
    return !info.IsDir()
}

// parseKeyFile parses a plain text key file into a map
func parseKeyFile(path string) (map[string]string, error) {
    data, err := ioutil.ReadFile(path)
    if err != nil {
        return nil, err
    }
    
    result := make(map[string]string)
    lines := strings.Split(string(data), "\n")
    
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        
        parts := strings.SplitN(line, "=", 2)
        if len(parts) != 2 {
            continue
        }
        
        key := strings.TrimSpace(parts[0])
        value := strings.TrimSpace(parts[1])
        
        // Remove quotes if present
        if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
            value = value[1 : len(value)-1]
        }
        
        result[key] = value
    }
    
    return result, nil
}
```

### encrypt_root_key.go GUI Tool Update

```go
// In encrypt_root_key.go (GUI version)

// Update the fields to match the new RootKeyFileContentProto
func createRootKeyContent() *pb.RootKeyFileContentProto {
    // Get values from GUI inputs
    stripeSecretKey := stripeSecretKeyInput.Text
    stripeWebhookSecret := stripeWebhookSecretInput.Text
    coinbaseApiKey := coinbaseApiKeyInput.Text
    coinbaseWebhookSecret := coinbaseWebhookSecretInput.Text
    rootPrivateKeyHex := rootPrivateKeyHexInput.Text
    
    // Validate root private key
    if rootPrivateKeyHex != utils.BLOCKCHAIN_PRIVATE_KEY {
        dialog.ShowInformation("Warning", 
            "The Root Private Key does not match the expected blockchain identity key.\n"+
            "This may cause issues with blockchain operations.", window)
    }
    
    // Create content
    return &pb.RootKeyFileContentProto{
        StripeSecretKey:       stripeSecretKey,
        StripeWebhookSecret:   stripeWebhookSecret,
        CoinbaseApiKey:        coinbaseApiKey,
        CoinbaseWebhookSecret: coinbaseWebhookSecret,
        RootPrivateKeyHex:     rootPrivateKeyHex,
    }
}

// Save to user's appData directory
func saveKeyFile() {
    // Get user's appData path
    keyPath, err := config.GetRootKeyPath()
    if err != nil {
        dialog.ShowError(fmt.Errorf("failed to get root key path: %w", err), window)
        return
    }
    
    // Create content
    content := createRootKeyContent()
    
    // Get password
    password := []byte(passwordInput.Text)
    if len(password) == 0 {
        dialog.ShowError(fmt.Errorf("password cannot be empty"), window)
        return
    }
    
    // Create key file
    if err := CreateRootKeyFile(content, password, keyPath); err != nil {
        dialog.ShowError(fmt.Errorf("failed to create key file: %w", err), window)
        return
    }
    
    dialog.ShowInformation("Success", 
        fmt.Sprintf("Root key file created successfully at %s\n"+
            "IMPORTANT: Securely back up this file and remember your password!", keyPath), 
        window)
}
```

The key to this implementation is clearly separating the concerns of initial role enablement (via the unencrypted `.key` in the executable directory) from the secure storage and password-protected access of operational secrets (in encrypted files within user-specific appData or embedded in the binary).