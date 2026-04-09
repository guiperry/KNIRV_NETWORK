package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/argon2"
)

// Wallet represents a wallet with its metadata
type Wallet struct {
	Name      string `json:"name"`
	Address   string `json:"address"`
	FilePath  string `json:"file_path"`
	Timestamp string `json:"timestamp"`
}

// WalletManager handles wallet operations
type WalletManager struct {
	walletDir string
	logger    *logrus.Logger
}

// NewWalletManager creates a new wallet manager
func NewWalletManager(walletDir string, logger *logrus.Logger) *WalletManager {
	return &WalletManager{
		walletDir: walletDir,
		logger:    logger,
	}
}

// GenerateKeyPair generates a new ECDSA key pair
func (wm *WalletManager) GenerateKeyPair() (*ecdsa.PrivateKey, error) {
	return crypto.GenerateKey()
}

// GetWalletPath returns the path for a wallet file
func (wm *WalletManager) GetWalletPath(address string) string {
	return filepath.Join(wm.walletDir, fmt.Sprintf("%s.json", address))
}

// WalletExists checks if a wallet file exists
func (wm *WalletManager) WalletExists(address string) bool {
	walletPath := wm.GetWalletPath(address)
	_, err := os.Stat(walletPath)
	return err == nil
}

// GetWallet retrieves wallet information by name (address)
func (wm *WalletManager) GetWallet(name string) (*Wallet, error) {
	// For now, treat name as address
	address := name
	walletPath := wm.GetWalletPath(address)

	// Check if wallet exists
	if !wm.WalletExists(address) {
		return nil, fmt.Errorf("wallet not found: %s", name)
	}

	// Get file info for timestamp
	fileInfo, err := os.Stat(walletPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet file info: %w", err)
	}

	wallet := &Wallet{
		Name:      name,
		Address:   address,
		FilePath:  walletPath,
		Timestamp: fileInfo.ModTime().Format(time.RFC3339),
	}

	return wallet, nil
}

// SignMessage signs a message with the specified wallet
func (wm *WalletManager) SignMessage(walletName string, message []byte) (string, error) {
	// Get wallet
	wallet, err := wm.GetWallet(walletName)
	if err != nil {
		return "", fmt.Errorf("failed to get wallet: %w", err)
	}

	// For now, return a mock signature
	// TODO: Implement actual signing with the wallet's private key
	signature := fmt.Sprintf("mock_signature_%s_%x", wallet.Address, message[:min(len(message), 8)])

	return signature, nil
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// WalletFile represents the structure of a wallet file
type WalletFile struct {
	Version   int       `json:"version"`
	Address   string    `json:"address"`
	Crypto    Crypto    `json:"crypto"`
	Timestamp time.Time `json:"timestamp"`
}

// Crypto contains the encrypted private key and encryption parameters
type Crypto struct {
	Cipher       string                 `json:"cipher"`
	CipherText   string                 `json:"ciphertext"`
	CipherParams CipherParams           `json:"cipherparams"`
	KDF          string                 `json:"kdf"`
	KDFParams    map[string]interface{} `json:"kdfparams"`
	MAC          string                 `json:"mac"`
}

// CipherParams contains parameters for the cipher
type CipherParams struct {
	Nonce string `json:"nonce"`
}

// ListWallets returns a list of wallet addresses
func (wm *WalletManager) ListWallets() ([]string, error) {
	var addresses []string

	// Read wallet directory
	files, err := os.ReadDir(wm.walletDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return addresses, nil
		}
		return nil, fmt.Errorf("failed to read wallet directory: %w", err)
	}

	// Filter JSON files
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			// Remove .json extension
			address := file.Name()[:len(file.Name())-5]
			addresses = append(addresses, address)
		}
	}

	return addresses, nil
}

// GetAddressFromPrivateKey returns the Ethereum address for a private key
func GetAddressFromPrivateKey(privateKey *ecdsa.PrivateKey) string {
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return ""
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA)
	return address.Hex()
}

// ParsePrivateKey parses a hex-encoded private key
func ParsePrivateKey(privateKeyHex string) (*ecdsa.PrivateKey, error) {
	// Remove 0x prefix if present
	if len(privateKeyHex) > 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}

	// Parse private key
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	return privateKey, nil
}

// GetPublicKeyHex returns the hex-encoded public key for a private key
func GetPublicKeyHex(publicKey *ecdsa.PublicKey) string {
	publicKeyBytes := crypto.FromECDSAPub(publicKey)
	return "0x" + hex.EncodeToString(publicKeyBytes)
}

// GetPublicKeyFromPrivateKey returns the public key for a private key
func GetPublicKeyFromPrivateKey(privateKey *ecdsa.PrivateKey) *ecdsa.PublicKey {
	return &privateKey.PublicKey
}

// SaveWallet saves a wallet to a file
func (wm *WalletManager) SaveWallet(privateKey *ecdsa.PrivateKey, password string, filePath string) error {
	// Get address from private key
	address := GetAddressFromPrivateKey(privateKey)

	// Convert private key to bytes
	privateKeyBytes := crypto.FromECDSA(privateKey)

	// Create wallet file
	walletFile, err := wm.createWalletFile(privateKeyBytes, address, password)
	if err != nil {
		return fmt.Errorf("failed to create wallet file: %w", err)
	}

	// Marshal wallet file to JSON
	walletJSON, err := json.MarshalIndent(walletFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal wallet file: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0700); err != nil {
		return fmt.Errorf("failed to create wallet directory: %w", err)
	}

	// Write wallet file
	if err := os.WriteFile(filePath, walletJSON, 0600); err != nil {
		return fmt.Errorf("failed to write wallet file: %w", err)
	}

	return nil
}

// createWalletFile creates a wallet file structure
func (wm *WalletManager) createWalletFile(privateKeyBytes []byte, address, password string) (*WalletFile, error) {
	// Generate salt
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key using Argon2id
	memory := uint32(64 * 1024) // 64MB
	iterations := uint32(3)
	parallelism := uint8(4)
	keyLength := uint32(32) // 256 bits

	derivedKey := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, keyLength)

	// Generate nonce for AES-GCM
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt private key
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	ciphertext := aesgcm.Seal(nil, nonce, privateKeyBytes, nil)

	// Calculate MAC
	mac := sha256.Sum256(append(derivedKey[16:32], ciphertext...))

	// Create wallet file
	walletFile := &WalletFile{
		Version:   1,
		Address:   address,
		Timestamp: time.Now(),
		Crypto: Crypto{
			Cipher:     "aes-256-gcm",
			CipherText: base64.StdEncoding.EncodeToString(ciphertext),
			CipherParams: CipherParams{
				Nonce: base64.StdEncoding.EncodeToString(nonce),
			},
			KDF: "argon2id",
			KDFParams: map[string]interface{}{
				"salt":        base64.StdEncoding.EncodeToString(salt),
				"memory":      memory,
				"iterations":  iterations,
				"parallelism": parallelism,
			},
			MAC: hex.EncodeToString(mac[:]),
		},
	}

	return walletFile, nil
}

// LoadWallet loads a wallet from a file
func (wm *WalletManager) LoadWallet(filePath, password string) (*ecdsa.PrivateKey, error) {
	// Read wallet file
	walletJSON, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read wallet file: %w", err)
	}

	// Unmarshal wallet file
	var walletFile WalletFile
	if err := json.Unmarshal(walletJSON, &walletFile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal wallet file: %w", err)
	}

	// Validate wallet file
	if walletFile.Version != 1 {
		return nil, fmt.Errorf("unsupported wallet version: %d", walletFile.Version)
	}

	if walletFile.Crypto.Cipher != "aes-256-gcm" {
		return nil, fmt.Errorf("unsupported cipher: %s", walletFile.Crypto.Cipher)
	}

	if walletFile.Crypto.KDF != "argon2id" {
		return nil, fmt.Errorf("unsupported KDF: %s", walletFile.Crypto.KDF)
	}

	// Get KDF parameters
	saltStr, ok := walletFile.Crypto.KDFParams["salt"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid KDF parameter: salt")
	}

	salt, err := base64.StdEncoding.DecodeString(saltStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	memory, ok := walletFile.Crypto.KDFParams["memory"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid KDF parameter: memory")
	}

	iterations, ok := walletFile.Crypto.KDFParams["iterations"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid KDF parameter: iterations")
	}

	parallelism, ok := walletFile.Crypto.KDFParams["parallelism"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid KDF parameter: parallelism")
	}

	// Derive key using Argon2id
	keyLength := uint32(32) // 256 bits
	derivedKey := argon2.IDKey(
		[]byte(password),
		salt,
		uint32(iterations),
		uint32(memory),
		uint8(parallelism),
		keyLength,
	)

	// Decode ciphertext and nonce
	ciphertext, err := base64.StdEncoding.DecodeString(walletFile.Crypto.CipherText)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(walletFile.Crypto.CipherParams.Nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to decode nonce: %w", err)
	}

	// Verify MAC
	mac := sha256.Sum256(append(derivedKey[16:32], ciphertext...))
	if hex.EncodeToString(mac[:]) != walletFile.Crypto.MAC {
		return nil, fmt.Errorf("invalid password or corrupted wallet file")
	}

	// Decrypt private key
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	privateKeyBytes, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt private key: %w", err)
	}

	// Parse private key
	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return privateKey, nil
}
