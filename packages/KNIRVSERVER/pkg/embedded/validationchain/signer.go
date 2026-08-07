package validationchain

import (
	"crypto/ecdsa"
	"fmt"
	"os"
	"strings"

	knirvsigning "github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/signing"
	"github.com/ethereum/go-ethereum/crypto"
)

// LoadCheckpointSigner loads the explicitly provisioned unattended service
// identity used by Validation Chain. It deliberately never falls back to a
// user's root wallet and never generates a key at runtime.
func LoadCheckpointSigner() (*ecdsa.PrivateKey, error) {
	hexKey := strings.TrimSpace(os.Getenv("KNIRV_VALIDATION_SERVICE_PRIVATE_KEY"))
	if hexKey == "" {
		keyPath := strings.TrimSpace(os.Getenv("KNIRV_VALIDATION_SERVICE_KEY_FILE"))
		if keyPath == "" {
			return nil, fmt.Errorf("set KNIRV_VALIDATION_SERVICE_KEY_FILE (preferred) or KNIRV_VALIDATION_SERVICE_PRIVATE_KEY to an Oracle-registered service key")
		}
		info, err := os.Stat(keyPath)
		if err != nil {
			return nil, fmt.Errorf("stat Validation Chain service key: %w", err)
		}
		if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
			return nil, fmt.Errorf("Validation Chain service key file must be regular and accessible only by its owner (mode 0600 or stricter)")
		}
		data, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("read Validation Chain service key: %w", err)
		}
		hexKey = strings.TrimSpace(string(data))
	}
	hexKey = strings.TrimPrefix(hexKey, "0x")
	key, err := crypto.HexToECDSA(hexKey)
	if err != nil {
		return nil, fmt.Errorf("Validation Chain service key is not valid secp256k1: %w", err)
	}
	return key, nil
}

func CheckpointAddress(pub *ecdsa.PublicKey) string {
	address, err := knirvsigning.Address(crypto.CompressPubkey(pub), knirvsigning.DefaultAddressPrefix)
	if err != nil {
		panic(err)
	}
	return address
}
