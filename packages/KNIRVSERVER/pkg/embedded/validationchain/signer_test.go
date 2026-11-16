package validationchain

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestLoadCheckpointSignerProvisioning(t *testing.T) {
	// Synthetic scalar one; never a node or wallet key.
	key := strings.Repeat("0", 63) + "1"
	t.Setenv("KNIRV_VALIDATION_SERVICE_PRIVATE_KEY", "")
	t.Setenv("KNIRV_VALIDATION_SERVICE_KEY_FILE", "")
	if _, err := LoadCheckpointSigner(); err == nil {
		t.Fatal("missing service identity accepted")
	}
	path := filepath.Join(t.TempDir(), "service.key")
	if err := os.WriteFile(path, []byte(key), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KNIRV_VALIDATION_SERVICE_KEY_FILE", path)
	signer, err := LoadCheckpointSigner()
	if err != nil {
		t.Fatal(err)
	}
	if signer.Address() != "knirv1w508d6qejxtdg4y5r3zarvary0c5xw7kaqx36e" || CheckpointAddress(signer) != signer.Address() {
		t.Fatalf("unexpected checkpoint address %q", signer.Address())
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCheckpointSigner(); err == nil {
		t.Fatal("insecure key file accepted")
	}
	// The established environment override takes precedence over the file.
	t.Setenv("KNIRV_VALIDATION_SERVICE_PRIVATE_KEY", key)
	if signer, err := LoadCheckpointSigner(); err != nil || signer.Address() != CheckpointAddress(signer) {
		t.Fatalf("environment override failed: %v", err)
	}
}

// Key validation runs without the SDK core (no address derivation), so these
// tests exercise the bad-input and good-input paths of checkpointKeyBytes directly.
func TestCheckpointKeyBytesRejectsBadHex(t *testing.T) {
	cases := map[string]string{
		"non-hex character": strings.Repeat("0", 62) + "zz",
		"odd length":        strings.Repeat("0", 63),
		"too short":         strings.Repeat("0", 62) + "1",
		"too long":          strings.Repeat("f", 66),
		"zero scalar":       strings.Repeat("0", 64),
		"scalar >= N":       "ffffffffffffffffffffffffffffffffbaaedce6af48a03bbfd25e8cd0364141",
	}
	for name, hexKey := range cases {
		t.Run(name, func(t *testing.T) {
			b, err := checkpointKeyBytes(hexKey)
			if err == nil {
				t.Fatalf("accepted %s: %x", name, b)
			}
			if !strings.Contains(err.Error(), "Validation Chain service key is not valid secp256k1") {
				t.Fatalf("unexpected error text: %v", err)
			}
		})
	}
}

func TestCheckpointKeyBytesAcceptsGoodHex(t *testing.T) {
	// Freshly generated test key; never a node or wallet key.
	priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	want := crypto.FromECDSA(priv)
	got, err := checkpointKeyBytes(hex.EncodeToString(want))
	if err != nil {
		t.Fatalf("valid key rejected: %v", err)
	}
	if !bytes.Equal(got, want) || len(got) != 32 {
		t.Fatalf("bytes mismatch: got %x want %x", got, want)
	}
	// A 0x prefix is stripped by loadCheckpointKeyHex, so the helper receives bare hex;
	// the scalar one must still serialize to 32 bytes with leading zeros kept.
	one, err := checkpointKeyBytes(strings.Repeat("0", 63) + "1")
	if err != nil || len(one) != 32 || one[31] != 1 {
		t.Fatalf("scalar one: %x %v", one, err)
	}
}
