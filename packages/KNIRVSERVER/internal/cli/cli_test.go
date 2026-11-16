package cli

import (
	"context"
	"strings"
	"testing"
)

// The CLI must refuse to proceed without an approved registration and an issued
// nonce, and must not read any key before those checks pass.
func TestRunEnrollRequiresRegistrationID(t *testing.T) {
	err := runEnrollRegister([]string{"--socket", "/nonexistent.sock", "--operator", "operator:alice", "--nonce", "enroll:x", "--secrets-backend", "file"})
	if err == nil || !strings.Contains(err.Error(), "--registration-id is required") {
		t.Fatalf("missing --registration-id must error, got %v", err)
	}
}

func TestRunEnrollRejectsClaimsWithoutRegistration(t *testing.T) {
	err := runEnrollRegister([]string{"--socket", "/nonexistent.sock", "--operator", "operator:alice", "--nonce", "enroll:x", "--claims", "/tmp/none.json", "--secrets-backend", "file"})
	if err == nil {
		t.Fatal("--claims without --registration-id must error")
	}
}

func TestRunEnrollRequiresIssuedNonce(t *testing.T) {
	err := runEnrollRegister([]string{"--socket", "/nonexistent.sock", "--operator", "operator:alice", "--registration-id", "reg-1", "--secrets-backend", "file"})
	if err == nil || !strings.Contains(err.Error(), "--nonce is required") {
		t.Fatalf("missing --nonce must error, got %v", err)
	}
}

func TestResolveKeySourceRequiresExplicitChoice(t *testing.T) {
	ctx := context.Background()
	if _, _, err := resolveKeySource(ctx, "", "", "", "", false, "", ""); err == nil {
		t.Fatal("no backend and no --dev-raw-keys must error")
	}
	if _, _, err := resolveKeySource(ctx, "file", "", "", "", true, "", ""); err == nil {
		t.Fatal("--dev-raw-keys and --secrets-backend must be mutually exclusive")
	}
}

func TestResolveKeySourceBaoNeedsSocketRoleAndSecretID(t *testing.T) {
	ctx := context.Background()
	if _, _, err := resolveKeySource(ctx, "bao", "", "role", "/x", false, "", ""); err == nil {
		t.Fatal("bao backend without socket must error")
	}
	if _, _, err := resolveKeySource(ctx, "bao", "/run/bao.sock", "", "/x", false, "", ""); err == nil || !strings.Contains(err.Error(), "--bao-role-id") {
		t.Fatalf("bao backend without role id must error, got %v", err)
	}
	if _, _, err := resolveKeySource(ctx, "bao", "/run/bao.sock", "role", "", false, "", ""); err == nil || !strings.Contains(err.Error(), "--bao-secret-id-file") {
		t.Fatalf("bao backend without secret-id file must error, got %v", err)
	}
}
