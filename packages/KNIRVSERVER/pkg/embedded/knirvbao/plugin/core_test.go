package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/guiperry/knirv-sdk-go/signing"
)

func TestEmbeddedCoreMatchesSDKPin(t *testing.T) {
	stamp, err := os.ReadFile("../../../../../KNIRVSDK/go-package/internal/core/stamp.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(stamp), `Version = "`+embeddedCoreVersion+`"`) || !strings.Contains(string(stamp), `Digest = "`+embeddedCoreDigest+`"`) {
		t.Fatal("embedded core pin differs from SDK; rebuild plugin assets")
	}
}

func TestEmbeddedCoreWorksOffline(t *testing.T) {
	if os.Getenv("KNIRVBAO_CORE_TEST_CHILD") == "1" {
		cleanup, err := installEmbeddedCore()
		if err != nil {
			t.Fatal(err)
		}
		defer cleanup()
		key := make([]byte, 32)
		key[31] = 1 // public test key
		if _, err := signing.NewLocalSigner(key); err != nil {
			t.Fatal(err)
		}
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestEmbeddedCoreWorksOffline$")
	cmd.Env = append(os.Environ(), "KNIRVBAO_CORE_TEST_CHILD=1", "KNIRV_SDK_BASE_URL=http://127.0.0.1:1")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("offline signer: %v: %s", err, output)
	}
}
