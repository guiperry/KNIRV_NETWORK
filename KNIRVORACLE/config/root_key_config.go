package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	pb "KNIRVORACLE/proto"

	"google.golang.org/protobuf/proto"
)

// GetRootKeyPath returns the default path for the Root key file.
// For Root role, it returns "root.key" in the config directory.
func GetRootKeyPath() (string, error) {
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
	data, err := ioutil.ReadFile(path)
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
