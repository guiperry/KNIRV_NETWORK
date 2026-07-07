package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestTokenManager_GenerateAndValidateToken(t *testing.T) {
	tm := NewTokenManager("test-secret-key")

	token, err := tm.GenerateToken("user123", "walletAddr1", []Permission{PermissionRead, PermissionWrite})
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	if token == "" {
		t.Error("Expected non-empty token")
	}

	claims, err := tm.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	if claims.UserID != "user123" {
		t.Errorf("Expected userID 'user123', got '%s'", claims.UserID)
	}
	if claims.WalletAddr != "walletAddr1" {
		t.Errorf("Expected walletAddr 'walletAddr1', got '%s'", claims.WalletAddr)
	}
	if len(claims.Permissions) != 2 {
		t.Errorf("Expected 2 permissions, got %d", len(claims.Permissions))
	}
}

func TestTokenManager_GenerateNodeToken(t *testing.T) {
	tm := NewTokenManager("test-secret-key")

	token, err := tm.GenerateNodeToken("user123", "walletAddr1", []Permission{PermissionMine}, "skill")
	if err != nil {
		t.Fatalf("GenerateNodeToken() error = %v", err)
	}

	claims, err := tm.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	if claims.NodeType != "skill" {
		t.Errorf("Expected nodeType 'skill', got '%s'", claims.NodeType)
	}
	if claims.TokenType != "node" {
		t.Errorf("Expected tokenType 'node', got '%s'", claims.TokenType)
	}
}

func TestTokenManager_ValidateToken_InvalidToken(t *testing.T) {
	tm := NewTokenManager("test-secret-key")

	_, err := tm.ValidateToken("definitely-not-valid")
	if err == nil {
		t.Error("Expected error for invalid token")
	}
}

func TestTokenManager_ValidateToken_WrongSecret(t *testing.T) {
	tm1 := NewTokenManager("secret1")
	tm2 := NewTokenManager("secret2")

	token, _ := tm1.GenerateToken("user123", "wallet1", []Permission{PermissionRead})

	_, err := tm2.ValidateToken(token)
	if err == nil {
		t.Error("Expected error for token signed with different secret")
	}
}

func TestTokenManager_ExpiredToken(t *testing.T) {
	tm := NewTokenManagerWithDuration("test-secret-key", -time.Hour)

	token, err := tm.GenerateToken("user123", "wallet1", []Permission{PermissionRead})
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	_, err = tm.ValidateToken(token)
	if err == nil {
		t.Error("Expected error for expired token")
	}
}

func TestTokenManager_Permissions(t *testing.T) {
	tm := NewTokenManager("test-secret-key")

	tests := []struct {
		name        string
		permissions []Permission
		wantRead    bool
		wantWrite   bool
		wantMint    bool
		wantMine    bool
		wantAdmin   bool
	}{
		{
			name:        "read only",
			permissions: []Permission{PermissionRead},
			wantRead:    true,
		},
		{
			name:        "read and write",
			permissions: []Permission{PermissionRead, PermissionWrite},
			wantRead:    true,
			wantWrite:   true,
		},
		{
			name:        "all permissions",
			permissions: []Permission{PermissionRead, PermissionWrite, PermissionMint, PermissionMine, PermissionAdmin},
			wantRead:    true,
			wantWrite:   true,
			wantMint:    true,
			wantMine:    true,
			wantAdmin:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, _ := tm.GenerateToken("user123", "wallet1", tt.permissions)
			claims, _ := tm.ValidateToken(token)

			if claims.HasPermission(PermissionRead) != tt.wantRead {
				t.Errorf("HasPermission(Read) = %v, want %v", claims.HasPermission(PermissionRead), tt.wantRead)
			}
			if claims.HasPermission(PermissionWrite) != tt.wantWrite {
				t.Errorf("HasPermission(Write) = %v, want %v", claims.HasPermission(PermissionWrite), tt.wantWrite)
			}
			if claims.HasPermission(PermissionMint) != tt.wantMint {
				t.Errorf("HasPermission(Mint) = %v, want %v", claims.HasPermission(PermissionMint), tt.wantMint)
			}
			if claims.HasPermission(PermissionMine) != tt.wantMine {
				t.Errorf("HasPermission(Mine) = %v, want %v", claims.HasPermission(PermissionMine), tt.wantMine)
			}
			if claims.HasPermission(PermissionAdmin) != tt.wantAdmin {
				t.Errorf("HasPermission(Admin) = %v, want %v", claims.HasPermission(PermissionAdmin), tt.wantAdmin)
			}
		})
	}
}

func TestClaims_HasPermission(t *testing.T) {
	claims := &Claims{
		Permissions: []Permission{PermissionRead, PermissionWrite, PermissionMine},
	}

	if !claims.HasPermission(PermissionRead) {
		t.Error("Expected HasPermission(Read) to be true")
	}
	if !claims.HasPermission(PermissionWrite) {
		t.Error("Expected HasPermission(Write) to be true")
	}
	if !claims.HasPermission(PermissionMine) {
		t.Error("Expected HasPermission(Mine) to be true")
	}
	if claims.HasPermission(PermissionMint) {
		t.Error("Expected HasPermission(Mint) to be false")
	}
	if claims.HasPermission(PermissionAdmin) {
		t.Error("Expected HasPermission(Admin) to be false")
	}
}

func TestClaims_Parse(t *testing.T) {
	claims := &Claims{
		UserID:     "user123",
		WalletAddr: "wallet1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	if claims.UserID != "user123" {
		t.Errorf("Expected userID 'user123', got '%s'", claims.UserID)
	}
}
