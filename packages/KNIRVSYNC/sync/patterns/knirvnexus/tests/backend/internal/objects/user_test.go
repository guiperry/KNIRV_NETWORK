package objects

import (
	"testing"
	"time"
)

func TestHashPassword(t *testing.T) {
	password := "testpassword123"

	hash1, salt1, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	hash2, salt2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password second time: %v", err)
	}

	// Hashes should be different due to different salts
	if hash1 == hash2 {
		t.Error("Expected different hashes for different salts")
	}

	if salt1 == salt2 {
		t.Error("Expected different salts")
	}

	// But verification should work for both
	if !VerifyPassword(password, hash1, salt1) {
		t.Error("Failed to verify first password hash")
	}

	if !VerifyPassword(password, hash2, salt2) {
		t.Error("Failed to verify second password hash")
	}

	// Wrong password should fail
	if VerifyPassword("wrongpassword", hash1, salt1) {
		t.Error("Wrong password should not verify")
	}
}

func TestGenerateSecureToken(t *testing.T) {
	token1, err := GenerateSecureToken(32)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	token2, err := GenerateSecureToken(32)
	if err != nil {
		t.Fatalf("Failed to generate second token: %v", err)
	}

	// Tokens should be different
	if token1 == token2 {
		t.Error("Expected different tokens")
	}

	// Token should be correct length (32 bytes = 64 hex characters)
	if len(token1) != 64 {
		t.Errorf("Expected token length 64, got %d", len(token1))
	}
}

func TestUser_IsValid(t *testing.T) {
	tests := []struct {
		name string
		user User
		want bool
	}{
		{
			name: "valid user",
			user: User{
				Username: "testuser",
				Email:    "test@example.com",
				Role:     "user",
			},
			want: true,
		},
		{
			name: "missing username",
			user: User{
				Email: "test@example.com",
				Role:  "user",
			},
			want: false,
		},
		{
			name: "missing email",
			user: User{
				Username: "testuser",
				Role:     "user",
			},
			want: false,
		},
		{
			name: "missing role",
			user: User{
				Username: "testuser",
				Email:    "test@example.com",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.IsValid(); got != tt.want {
				t.Errorf("User.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_IsAccountActive(t *testing.T) {
	tests := []struct {
		name string
		user User
		want bool
	}{
		{
			name: "active verified user",
			user: User{
				Status:        "active",
				EmailVerified: true,
			},
			want: true,
		},
		{
			name: "inactive user",
			user: User{
				Status:        "inactive",
				EmailVerified: true,
			},
			want: false,
		},
		{
			name: "unverified user",
			user: User{
				Status:        "active",
				EmailVerified: false,
			},
			want: false,
		},
		{
			name: "suspended user",
			user: User{
				Status:        "suspended",
				EmailVerified: true,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.IsAccountActive(); got != tt.want {
				t.Errorf("User.IsAccountActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_IsLocked(t *testing.T) {
	now := time.Now()
	future := now.Add(time.Hour)

	tests := []struct {
		name string
		user User
		want bool
	}{
		{
			name: "not locked",
			user: User{LockedUntil: nil},
			want: false,
		},
		{
			name: "locked",
			user: User{LockedUntil: &future},
			want: true,
		},
		{
			name: "lock expired",
			user: User{LockedUntil: &now},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.IsLocked(); got != tt.want {
				t.Errorf("User.IsLocked() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_CanLogin(t *testing.T) {
	now := time.Now()
	future := now.Add(time.Hour)

	tests := []struct {
		name string
		user User
		want bool
	}{
		{
			name: "can login",
			user: User{
				Status:        "active",
				EmailVerified: true,
				LockedUntil:   nil,
			},
			want: true,
		},
		{
			name: "cannot login - inactive",
			user: User{
				Status:        "inactive",
				EmailVerified: true,
				LockedUntil:   nil,
			},
			want: false,
		},
		{
			name: "cannot login - locked",
			user: User{
				Status:        "active",
				EmailVerified: true,
				LockedUntil:   &future,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.CanLogin(); got != tt.want {
				t.Errorf("User.CanLogin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_HasPermission(t *testing.T) {
	rolePermissions := map[string][]string{
		"admin": {"read", "write", "delete"},
		"user":  {"read", "write"},
		"guest": {"read"},
	}

	tests := []struct {
		name     string
		user     User
		permission string
		want     bool
	}{
		{
			name: "admin has write permission",
			user: User{Role: "admin"},
			permission: "write",
			want: true,
		},
		{
			name: "user has write permission",
			user: User{Role: "user"},
			permission: "write",
			want: true,
		},
		{
			name: "user does not have delete permission",
			user: User{Role: "user"},
			permission: "delete",
			want: false,
		},
		{
			name: "guest does not have write permission",
			user: User{Role: "guest"},
			permission: "write",
			want: false,
		},
		{
			name: "unknown role",
			user: User{Role: "unknown"},
			permission: "read",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.HasPermission(tt.permission, rolePermissions); got != tt.want {
				t.Errorf("User.HasPermission() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUser_GetFullName(t *testing.T) {
	tests := []struct {
		name string
		user User
		want string
	}{
		{
			name: "full name",
			user: User{
				Username:  "testuser",
				FirstName: "John",
				LastName:  "Doe",
			},
			want: "John Doe",
		},
		{
			name: "only first name",
			user: User{
				Username:  "testuser",
				FirstName: "John",
			},
			want: "testuser",
		},
		{
			name: "only last name",
			user: User{
				Username: "testuser",
				LastName: "Doe",
			},
			want: "testuser",
		},
		{
			name: "no names",
			user: User{Username: "testuser"},
			want: "testuser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.GetFullName(); got != tt.want {
				t.Errorf("User.GetFullName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserSession_IsExpired(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name    string
		session UserSession
		want    bool
	}{
		{
			name: "not expired",
			session: UserSession{ExpiresAt: future},
			want: false,
		},
		{
			name: "expired",
			session: UserSession{ExpiresAt: past},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.session.IsExpired(); got != tt.want {
				t.Errorf("UserSession.IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserSession_IsSessionActive(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name    string
		session UserSession
		want    bool
	}{
		{
			name: "active session",
			session: UserSession{
				IsActive:  true,
				ExpiresAt: future,
			},
			want: true,
		},
		{
			name: "inactive session",
			session: UserSession{
				IsActive:  false,
				ExpiresAt: future,
			},
			want: false,
		},
		{
			name: "expired session",
			session: UserSession{
				IsActive:  true,
				ExpiresAt: past,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.session.IsSessionActive(); got != tt.want {
				t.Errorf("UserSession.IsSessionActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserSession_UpdateActivity(t *testing.T) {
	session := &UserSession{}
	oldActivity := session.LastActivity

	// Wait a bit to ensure time difference
	time.Sleep(time.Millisecond)

	session.UpdateActivity()

	if !session.LastActivity.After(oldActivity) {
		t.Error("LastActivity should be updated to a later time")
	}
}

func TestRateLimit_IsLocked(t *testing.T) {
	now := time.Now()
	future := now.Add(time.Hour)

	tests := []struct {
		name      string
		rateLimit RateLimit
		want      bool
	}{
		{
			name: "not locked",
			rateLimit: RateLimit{LockedUntil: nil},
			want: false,
		},
		{
			name: "locked",
			rateLimit: RateLimit{LockedUntil: &future},
			want: true,
		},
		{
			name: "lock expired",
			rateLimit: RateLimit{LockedUntil: &now},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rateLimit.IsLocked(); got != tt.want {
				t.Errorf("RateLimit.IsLocked() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRateLimit_CanAttempt(t *testing.T) {
	now := time.Now()
	future := now.Add(time.Hour)

	tests := []struct {
		name      string
		rateLimit RateLimit
		want      bool
	}{
		{
			name: "can attempt",
			rateLimit: RateLimit{LockedUntil: nil},
			want: true,
		},
		{
			name: "cannot attempt - locked",
			rateLimit: RateLimit{LockedUntil: &future},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rateLimit.CanAttempt(); got != tt.want {
				t.Errorf("RateLimit.CanAttempt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRateLimit_RecordAttempt(t *testing.T) {
	rateLimit := &RateLimit{}

	// First attempt
	rateLimit.RecordAttempt(3, time.Hour)
	if rateLimit.Attempts != 1 {
		t.Errorf("Expected attempts to be 1, got %d", rateLimit.Attempts)
	}
	if rateLimit.LockedUntil != nil {
		t.Error("Should not be locked after first attempt")
	}

	// Second attempt
	rateLimit.RecordAttempt(3, time.Hour)
	if rateLimit.Attempts != 2 {
		t.Errorf("Expected attempts to be 2, got %d", rateLimit.Attempts)
	}

	// Third attempt - should lock
	rateLimit.RecordAttempt(3, time.Hour)
	if rateLimit.Attempts != 3 {
		t.Errorf("Expected attempts to be 3, got %d", rateLimit.Attempts)
	}
	if rateLimit.LockedUntil == nil {
		t.Error("Should be locked after third attempt")
	}
}

func TestRateLimit_Reset(t *testing.T) {
	future := time.Now().Add(time.Hour)
	rateLimit := &RateLimit{
		Attempts:    5,
		LockedUntil: &future,
	}

	rateLimit.Reset()

	if rateLimit.Attempts != 0 {
		t.Errorf("Expected attempts to be 0, got %d", rateLimit.Attempts)
	}
	if rateLimit.LockedUntil != nil {
		t.Error("LockedUntil should be nil after reset")
	}
}