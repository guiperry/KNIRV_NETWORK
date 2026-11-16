package launcher

import (
	"context"
	"encoding/hex"
	"testing"

	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
)

// legacyRootSignerForTest builds the legacy in-process signer from a fixed test key.
func legacyRootSignerForTest(t *testing.T) knirvsigning.RemoteSigner {
	t.Helper()
	priv, err := hex.DecodeString("0101010101010101010101010101010101010101010101010101010101010101")
	if err != nil {
		t.Fatal(err)
	}
	s, err := knirvsigning.NewLocalSigner(priv)
	if err != nil {
		t.Fatalf("legacy signer: %v", err)
	}
	return s
}

// TestRootRegistrySignerFallsBackWithoutBao: with no OpenBao manager the root
// registry record must be signed by the named legacy-local source, never silently.
func TestRootRegistrySignerFallsBackWithoutBao(t *testing.T) {
	legacy := legacyRootSignerForTest(t)
	app := &ServerApp{}
	got, source := app.rootRegistrySigner(context.Background(), legacy)
	if source != "legacy-local" {
		t.Fatalf("source = %q, want legacy-local", source)
	}
	if got != legacy {
		t.Fatal("expected the legacy signer to be returned on fallback")
	}
}

// TestRootRegistrySignerNilApp covers a nil receiver (defensive: launcher paths may
// run before the app is fully wired) and still falls back to the legacy signer.
func TestRootRegistrySignerNilApp(t *testing.T) {
	legacy := legacyRootSignerForTest(t)
	var app *ServerApp
	got, source := app.rootRegistrySigner(context.Background(), legacy)
	if source != "legacy-local" || got != legacy {
		t.Fatalf("nil app: source=%q got=%v", source, got)
	}
}
