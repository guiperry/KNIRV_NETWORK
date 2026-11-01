package validationchain

import (
	"fmt"
	"os"
	"strings"

	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
	"github.com/ethereum/go-ethereum/crypto"
)

// LoadCheckpointSigner loads the explicitly provisioned unattended service
// identity used by Validation Chain. It deliberately never falls back to a
// user's root wallet and never generates a key at runtime.
func LoadCheckpointSigner() (knirvsigning.RemoteSigner, error) {
	hexKey, err := loadCheckpointKeyHex()
	if err != nil {
		return nil, err
	}
	key, err := crypto.HexToECDSA(hexKey)
	if err != nil {
		return nil, fmt.Errorf("Validation Chain service key is not valid secp256k1: %w", err)
	}
	return knirvsigning.NewLocalSigner(crypto.FromECDSA(key))
}

// LoadCheckpointPrivateKeyHex returns the provisioned key as hex for the
// legacy child-process handoff (the Validation Chain binary still reads it
// from its environment). Task 2a.4 removes this handoff in favour of a
// signer handle; new code must use LoadCheckpointSigner instead.
func LoadCheckpointPrivateKeyHex() (string, error) {
	hexKey, err := loadCheckpointKeyHex()
	if err != nil {
		return "", err
	}
	return hexKey, nil
}

func loadCheckpointKeyHex() (string, error) {
	hexKey := strings.TrimSpace(os.Getenv("KNIRV_VALIDATION_SERVICE_PRIVATE_KEY"))
	if hexKey == "" {
		keyPath := strings.TrimSpace(os.Getenv("KNIRV_VALIDATION_SERVICE_KEY_FILE"))
		if keyPath == "" {
			return "", fmt.Errorf("set KNIRV_VALIDATION_SERVICE_KEY_FILE (preferred) or KNIRV_VALIDATION_SERVICE_PRIVATE_KEY to an Oracle-registered service key")
		}
		info, err := os.Stat(keyPath)
		if err != nil {
			return "", fmt.Errorf("stat Validation Chain service key: %w", err)
		}
		if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
			return "", fmt.Errorf("Validation Chain service key file must be regular and accessible only by its owner (mode 0600 or stricter)")
		}
		data, err := os.ReadFile(keyPath)
		if err != nil {
			return "", fmt.Errorf("read Validation Chain service key: %w", err)
		}
		hexKey = strings.TrimSpace(string(data))
	}
	hexKey = strings.TrimPrefix(hexKey, "0x")
	if hexKey == "" {
		return "", fmt.Errorf("Validation Chain service key is empty")
	}
	return hexKey, nil
}

func CheckpointAddress(signer knirvsigning.RemoteSigner) string {
	if signer == nil {
		return ""
	}
	return signer.Address()
}
