package validationchain

import (
	"crypto/ecdsa"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"

	"knirv-server/internal/bootkey"
)

const checkpointSignerFilename = "validationchain-checkpoint-signer.key"

// LoadOrCreateCheckpointSigner returns the secp256k1 identity Validation
// Chain uses to register with and post checkpoints to KNIRVORACLE.
//
// root.key reserves a field (root_private_key_hex, proto field 5) for
// exactly this purpose, but nothing in the onboarding pipeline populates it
// yet (KNIRVCONTROLLER's "Sign to Enter" step is currently a UI stub). If an
// operator has manually filled it in via root_key_encryptor, that key is
// used and the identity is stable across the whole KNIRV deployment.
// Otherwise a dedicated key is generated once and persisted under the app
// data dir, so the identity is at least stable across restarts of this node
// — and the moment a real onboarding-derived key lands in root.key, it takes
// over automatically with no code change.
func LoadOrCreateCheckpointSigner(appDataDir string) (*ecdsa.PrivateKey, error) {
	if creds, err := bootkey.LoadRootKeyCloudflareCreds(); err == nil && creds != nil {
		if hexKey := strings.TrimSpace(strings.TrimPrefix(creds.RootPrivateKeyHex, "0x")); hexKey != "" {
			key, err := crypto.HexToECDSA(hexKey)
			if err != nil {
				return nil, fmt.Errorf("root.key root_private_key_hex is not a valid secp256k1 key: %w", err)
			}
			return key, nil
		}
	}

	keyDir := appDataDir
	if strings.TrimSpace(keyDir) == "" {
		var err error
		keyDir, err = os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolve key directory: %w", err)
		}
	}
	keyPath := filepath.Join(keyDir, checkpointSignerFilename)

	if data, err := os.ReadFile(keyPath); err == nil {
		key, err := crypto.HexToECDSA(strings.TrimSpace(string(data)))
		if err != nil {
			return nil, fmt.Errorf("parse persisted checkpoint signer at %s: %w", keyPath, err)
		}
		return key, nil
	}

	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("generate checkpoint signer: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
		return nil, fmt.Errorf("create key directory: %w", err)
	}
	hexKey := fmt.Sprintf("%x", crypto.FromECDSA(key))
	if err := os.WriteFile(keyPath, []byte(hexKey), 0o600); err != nil {
		return nil, fmt.Errorf("persist checkpoint signer: %w", err)
	}
	return key, nil
}

// CheckpointAddress derives the Oracle-facing address for a checkpoint
// signer, using the same scheme KNIRVORACLE and KNIRVCHAIN already use:
// 0x + keccak256(uncompressed pubkey without the 0x04 prefix)[12:].
func CheckpointAddress(pub *ecdsa.PublicKey) string {
	uncompressed := crypto.FromECDSAPub(pub)
	sum := crypto.Keccak256(uncompressed[1:])
	return "0x" + fmt.Sprintf("%x", sum[12:])
}
