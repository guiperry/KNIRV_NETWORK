package config

import (
	"fmt"
	"os"
	"path/filepath"

	pb "KNIRVCHAIN/internal/protocol/proto"

	"google.golang.org/protobuf/proto"
)

// GetKNIRVServerRootKeyPath returns the path where KNIRVSERVER extracts root.key at runtime.
// KNIRVSERVER extracts the embedded root.key to: ~/.config/knirv-server/root.key
func GetKNIRVServerRootKeyPath() (string, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config dir: %w", err)
	}
	return filepath.Join(userConfigDir, "knirv-server", "root.key"), nil
}

// GetRootKeyPath returns the default path for the Root key file.
// It first checks the KNIRVSERVER location (where root.key is extracted at runtime),
// then falls back to the KNIRVCHAIN config directory.
func GetRootKeyPath() (string, error) {
	// First check KNIRVSERVER location (where it's extracted at runtime)
	if serverPath, err := GetKNIRVServerRootKeyPath(); err == nil {
		if _, err := os.Stat(serverPath); err == nil {
			return serverPath, nil
		}
	}

	// Fall back to KNIRVCHAIN config directory
	configDir, err := GetConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get config directory: %w", err)
	}

	return filepath.Join(configDir, "root.key"), nil
}

// LoadEncryptedRootKeyFile loads the encrypted Root key file from the specified path.
func LoadEncryptedRootKeyFile(path string) (*pb.EncryptedRootKeyFile, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("key file does not exist at %s", path)
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	// Unmarshal protobuf
	var keyFile pb.EncryptedRootKeyFile
	if err := proto.Unmarshal(data, &keyFile); err != nil {
		return nil, fmt.Errorf("failed to parse key file: %w", err)
	}

	return &keyFile, nil
}

// EncryptedRootKeyFile represents the structure saved to the .key file.
// Deprecated: Use proto.EncryptedRootKeyFile instead
type EncryptedRootKeyFile struct {
	Salt          []byte `json:"salt"`           // Salt for PBKDF
	N             int    `json:"n"`              // N parameter for Scrypt
	R             int    `json:"r"`              // R parameter for Scrypt
	P             int    `json:"p"`              // P parameter for Scrypt
	EncryptedData []byte `json:"encrypted_data"` // Encrypted protobuf bytes
}
