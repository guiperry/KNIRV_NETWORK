package validationchain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
