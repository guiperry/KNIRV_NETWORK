package controller

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestHasherHostReuseRequiresMatchingJitter(t *testing.T) {
	fixture := filepath.Join("..", "..", "..", "..", "KNIRVSDK", "testvectors", "models", "hasher_host_health.json")
	raw, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	var mode hasherHostMode
	if err := json.Unmarshal(raw, &mode); err != nil {
		t.Fatal(err)
	}
	if !mode.matches(true, false) || mode.matches(true, true) || mode.matches(false, false) {
		t.Fatal("fixture mode did not match only direct mode with jitter disabled")
	}
	var old hasherHostMode
	if err := json.Unmarshal([]byte(`{"direct_mode":true}`), &old); err != nil {
		t.Fatal(err)
	}
	if old.matches(true, false) || old.matches(true, true) {
		t.Fatal("reused a direct host whose jitter mode is unknown")
	}
	if err := json.Unmarshal([]byte(`{"direct_mode":false}`), &old); err != nil {
		t.Fatal(err)
	}
	if !old.matches(false, false) {
		t.Fatal("ordinary legacy host should still be reusable")
	}
}
