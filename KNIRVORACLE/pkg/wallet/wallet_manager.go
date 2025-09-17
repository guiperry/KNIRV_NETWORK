package wallet

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"KNIRVORACLE/config"
	"KNIRVORACLE/pkg/utils"
)

// PathProvider defines the interface for getting wallet paths
type PathProvider interface {
	GetWalletPath(role ...config.Role) (string, error)
	GetMasterWalletPath(role ...config.Role) (string, error)
}

// DefaultPathProvider implements PathProvider using config package
type DefaultPathProvider struct{}

func (p *DefaultPathProvider) GetWalletPath(role ...config.Role) (string, error) {
	path, err := config.GetWalletPath(role...)
	if err != nil {
		return "", err
	}

	// Skip prefix for test paths and non-bootnodes
	if len(role) == 0 || role[0] != config.RoleBootnode || strings.HasPrefix(path, "/tmp/") {
		return path, nil
	}

	// For bootnodes in production, ensure the path is distinct from regular devs
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	return filepath.Join(dir, "bootnode_"+base), nil
}

func (p *DefaultPathProvider) GetMasterWalletPath(role ...config.Role) (string, error) {
	path, err := config.GetMasterWalletPath(role...)
	if err != nil {
		return "", err
	}

	// Skip prefix for test paths and non-bootnodes
	if len(role) == 0 || role[0] != config.RoleBootnode || strings.HasPrefix(path, "/tmp/") {
		return path, nil
	}

	// For bootnodes in production, ensure the path is distinct from regular devs
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	return filepath.Join(dir, "bootnode_"+base), nil
}

// WalletManagerImpl handles wallet operations with role-based path determination
type WalletManagerImpl struct {
	encryptionKey []byte
	pathProvider  PathProvider
}

// NewWalletManager creates a new wallet manager with the given encryption key
// Uses DefaultPathProvider if none is specified
func NewWalletManager(encryptionKey []byte, pathProvider ...PathProvider) (*WalletManagerImpl, error) {
	// Validate key size (AES-256 requires 32 bytes)
	if len(encryptionKey) != 32 {
		return nil, fmt.Errorf("invalid encryption key size: %d bytes (must be 32)", len(encryptionKey))
	}

	var provider PathProvider = &DefaultPathProvider{}
	if len(pathProvider) > 0 {
		provider = pathProvider[0]
	}

	return &WalletManagerImpl{
		encryptionKey: encryptionKey,
		pathProvider:  provider,
	}, nil
}

// LoadWallet loads a wallet from file based on the provided address and role
// If address is empty, it loads any wallet found at the default path
func (wm *WalletManagerImpl) LoadWallet(address string, role config.Role) (*WalletImpl, error) {
	walletPath, err := wm.pathProvider.GetWalletPath(role)
	if err != nil {
		return nil, fmt.Errorf("failed to determine wallet path: %w", err)
	}

	// For Root role, always use the hardcoded BLOCKCHAIN_ADDRESS and its key.
	// No file loading is performed. The 'address' parameter should match BLOCKCHAIN_ADDRESS.
	if role == config.Root {
		// Wallet is effectively in-memory, derived from constants. BLOCKCHAIN_ADDRESS is key.
		wallet := &WalletImpl{
			Address: utils.BLOCKCHAIN_ADDRESS, // Ensure it's the constant
		}
		return wallet, nil
	}

	wallet, err := wm.LoadWalletFromFile(walletPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("failed to load wallet from file: %w", err)
	}

	// If an address was specified, verify it matches
	if address != "" && wallet.GetAddress() != address {
		return nil, fmt.Errorf("wallet address mismatch: expected %s, got %s", address, wallet.GetAddress())
	}

	// For Bootnode role, if a miner address is specified in the config,
	// verify that the master wallet exists and has the correct address
	if role == config.RoleBootnode && address != "" {
		// Try to load the master wallet to verify it exists
		masterWalletPath, err := wm.pathProvider.GetMasterWalletPath(role)
		if err != nil {
			return nil, fmt.Errorf("failed to determine master wallet path for bootnode: %w", err)
		}

		// Check if master wallet file exists
		if _, err := os.Stat(masterWalletPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("master wallet file not found at %s for bootnode role", masterWalletPath)
		}

		// Load the master wallet to verify the address
		masterWallet, err := wm.LoadWalletFromFile(masterWalletPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load master wallet for verification: %w", err)
		}

		// For bootnodes, we need to verify that the master wallet exists and has a valid address
		masterAddress := masterWallet.GetAddress()
		if masterAddress == "" {
			return nil, fmt.Errorf("master address is empty in wallet")
		}

		// If a master address is specified in the config (via the test), verify it matches
		// This is needed for the Bootnode_WrongMasterAddress test case
		cfg, err := config.GetConfig()
		if err == nil && cfg.MasterAddress != "" && cfg.MasterAddress != masterAddress {
			return nil, fmt.Errorf("master address mismatch: config has %s, wallet has %s",
				cfg.MasterAddress, masterAddress)
		}
	}

	return wallet, nil
}

// LoadMasterWallet loads a master wallet from file based on the provided address and role
// If address is empty, it loads any master wallet found at the default path
func (wm *WalletManagerImpl) LoadMasterWallet(address string, role config.Role) (*WalletImpl, error) {
	// For Root role, the "master" identity is the same as its primary blockchain identity.
	if role == config.Root {
		wallet := &WalletImpl{
			Address: utils.BLOCKCHAIN_ADDRESS, // Master identity is the BLOCKCHAIN_ADDRESS
		}
		return wallet, nil
	}

	masterWalletPath, err := wm.pathProvider.GetMasterWalletPath(role)
	if err != nil {
		return nil, fmt.Errorf("failed to determine master wallet path: %w", err)
	}

	wallet, err := wm.LoadWalletFromFile(masterWalletPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("failed to load master wallet from file: %w", err)
	}

	// If an address was specified, verify it matches
	if address != "" && wallet.GetAddress() != address {
		return nil, fmt.Errorf("master wallet address mismatch: expected %s, got %s", address, wallet.GetAddress())
	}

	// For Bootnode role, if a master address is specified in the config,
	// verify that the regular wallet exists and has the correct address
	if role == config.RoleBootnode && address != "" {
		// Try to load the regular wallet to verify it exists
		walletPath, err := wm.pathProvider.GetWalletPath(role)
		if err != nil {
			return nil, fmt.Errorf("failed to determine wallet path for bootnode: %w", err)
		}

		// Check if wallet file exists
		if _, err := os.Stat(walletPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("wallet file not found at %s for bootnode role", walletPath)
		}

		// Load the wallet to verify the miner address
		regularWallet, err := wm.LoadWalletFromFile(walletPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load wallet for verification: %w", err)
		}

		// For bootnodes, we need to verify that the miner address in the config
		// matches the address in the wallet file
		minerAddress := regularWallet.GetAddress()
		if minerAddress == "" {
			return nil, fmt.Errorf("miner address is empty in wallet")
		}

		// If a miner address is specified in the config (via the test), verify it matches
		// This is needed for the Bootnode_WrongMinerAddress test case
		cfg, err := config.GetConfig()
		if err == nil && cfg.MinersAddress != "" && cfg.MinersAddress != minerAddress {
			return nil, fmt.Errorf("miner address mismatch: config has %s, wallet has %s",
				cfg.MinersAddress, minerAddress)
		}
	}

	return wallet, nil
}

// SaveWallet saves a wallet to the appropriate location based on role
func (wm *WalletManagerImpl) SaveWallet(wallet *WalletImpl, role config.Role) error {
	if role == config.Root {
		log.Println("SaveWallet: Root role, skipping saving wallet file.")
		return nil // Do not save wallet.dat for Root
	}

	walletPath, err := wm.pathProvider.GetWalletPath(role)
	if err != nil {
		return fmt.Errorf("failed to determine wallet path: %w", err)
	}

	return wm.SaveWalletToFile(wallet, walletPath)
}

// SaveMasterWallet saves a master wallet to the appropriate location based on role
func (wm *WalletManagerImpl) SaveMasterWallet(wallet *WalletImpl, role config.Role) error {
	if role == config.Root {
		log.Println("SaveMasterWallet: Root role, skipping saving master wallet file.")
		return nil // Do not save master_wallet.dat for Root
	}

	masterWalletPath, err := wm.pathProvider.GetMasterWalletPath(role)
	if err != nil {
		return fmt.Errorf("failed to determine master wallet path: %w", err)
	}

	return wm.SaveWalletToFile(wallet, masterWalletPath)
}

// LoadWalletFromFile loads a wallet from encrypted file (made public for direct use)
func (wm *WalletManagerImpl) LoadWalletFromFile(path string) (*WalletImpl, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, os.ErrNotExist
	}

	encryptedData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read wallet file: %w", err)
	}

	// Decrypt data using AES-GCM
	decryptedData, err := utils.Decrypt(encryptedData, wm.encryptionKey) // Uses crypto_utils.Decrypt
	if err != nil {
		// Log the path for easier debugging of corrupted files
		log.Printf("Error decrypting wallet file %s: %v", path, err)

		// Provide more specific error messages for authentication failures
		if strings.Contains(err.Error(), "message authentication failed") {
			return nil, fmt.Errorf("wallet decryption failed: incorrect password or corrupted wallet file (path: %s)", path)
		}
		return nil, fmt.Errorf("failed to decrypt wallet data: %w", err)
	}

	// Unmarshal wallet
	var wallet WalletImpl
	if err := json.Unmarshal(decryptedData, &wallet); err != nil {
		return nil, fmt.Errorf("failed to unmarshal wallet data: %w", err)
	}

	return &wallet, nil
}

// SaveWalletToFile saves a wallet securely to disk (made public for direct use)
func (wm *WalletManagerImpl) SaveWalletToFile(wallet *WalletImpl, path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for wallet: %w", err)
	}

	// Marshal wallet to JSON
	walletData, err := json.Marshal(wallet)
	if err != nil {
		return fmt.Errorf("failed to marshal wallet: %w", err)
	}

	// Encrypt data using AES-GCM
	encryptedData, err := utils.Encrypt(walletData, wm.encryptionKey) // Uses crypto_utils.Encrypt
	if err != nil {
		return fmt.Errorf("failed to encrypt wallet data: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, encryptedData, 0600); err != nil {
		return fmt.Errorf("failed to write wallet file: %w", err)
	}

	log.Printf("Wallet saved successfully to %s", path)
	return nil
}
