package validationchain

import (
	"context"
	"fmt"
	"os"
	"strings"

	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
	knirvsecrets "github.com/guiperry/knirv-sdk-go/secrets"
	"github.com/ethereum/go-ethereum/crypto"
)

// LoadCheckpointSigner loads the Validation Chain checkpoint-signer identity.
//
// When OpenBao is configured (VALIDATION_CHAIN_BAO_SOCKET set) the identity is
// the node-local OpenBao plugin's checkpoint-signer, so the raw key never
// leaves OpenBao and the same identity is used by both the embedded binary and
// the Go-side checkpoint runtime. Otherwise the explicitly provisioned service
// key (legacy raw-hex handoff) is used. It deliberately never falls back to a
// user's root wallet and never generates a key at runtime.
func LoadCheckpointSigner() (knirvsigning.RemoteSigner, error) {
	if socket := strings.TrimSpace(os.Getenv("VALIDATION_CHAIN_BAO_SOCKET")); socket != "" {
		token := strings.TrimSpace(os.Getenv("VALIDATION_CHAIN_BAO_TOKEN"))
		return knirvsecrets.NewBaoProvider(socket, token).
			Signer(context.Background(), "checkpoint-signer")
	}
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
// from its environment). Task 2a.4 routes the binary through OpenBao's
// checkpoint-signer when configured, in which case the raw key is not needed;
// new code must use LoadCheckpointSigner instead.
func LoadCheckpointPrivateKeyHex() (string, error) {
	// When OpenBao is the signing backend the child binary derives its identity
	// from the same OpenBao checkpoint-signer, so no raw key is handed off.
	if strings.TrimSpace(os.Getenv("VALIDATION_CHAIN_BAO_SOCKET")) != "" {
		return "", nil
	}
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
