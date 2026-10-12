package compat

import (
	"os"
	"testing"
)

func TestPromoteLegacyEnv(t *testing.T) {
	t.Setenv("KNIRV_HRM_PRESET", "small")
	t.Setenv("KNIRV_HRM_EPOCHS", "3")
	t.Setenv("KNIRV_HEAD_EPOCHS", "9") // new name wins
	os.Unsetenv("KNIRV_HEAD_PRESET")
	PromoteLegacyEnv()
	if got := os.Getenv("KNIRV_HEAD_PRESET"); got != "small" {
		t.Fatalf("PRESET not promoted: %q", got)
	}
	if got := os.Getenv("KNIRV_HEAD_EPOCHS"); got != "9" {
		t.Fatalf("new name was overwritten: %q", got)
	}
	os.Unsetenv("KNIRV_HEAD_PRESET")
}
