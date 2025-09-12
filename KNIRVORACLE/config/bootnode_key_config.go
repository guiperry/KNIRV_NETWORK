package config

import (
	"fmt"
	"os"
	"path/filepath"

	pb "KNIRVORACLE/proto"

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
	data, err := os.ReadFile(path)
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
