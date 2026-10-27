package launcher

import "testing"

func TestResolvePublicURLDeploymentClasses(t *testing.T) {
	t.Setenv("KNIRV_PUBLIC_URL", "")
	tests := []struct {
		name    string
		config  Config
		want    string
		wantErr bool
	}{
		{name: "testnet default", config: Config{NetworkMode: "testnet"}, want: "https://testnet-gateway.knirv.com"},
		{name: "production mainnet", config: Config{NetworkMode: "production"}, want: "https://gateway.knirv.com"},
		{name: "pro devnet", config: Config{NetworkMode: "development", UserIDTag: "User 42"}, want: "https://devnet-user-42.knirv.com"},
		{name: "enterprise", config: Config{NetworkMode: "enterprise", Enterprise: true, UserIDTag: "Acme_Admin"}, want: "https://enterprise-acme-admin.knirv.com"},
		{name: "configured domain devnet", config: Config{NetworkMode: "development", UserIDTag: "User 42", Domain: " Example.ORG. "}, want: "https://devnet-user-42.example.org"},
		{name: "configured domain enterprise", config: Config{NetworkMode: "enterprise", Enterprise: true, UserIDTag: "acme", Domain: "example.org"}, want: "https://enterprise-acme.example.org"},
		{name: "configured domain production", config: Config{NetworkMode: "production", Domain: "example.org"}, want: "https://gateway.example.org"},
		{name: "configured domain testnet", config: Config{NetworkMode: "testnet", Domain: "example.org"}, want: "https://testnet-gateway.example.org"},
		{name: "devnet tag required", config: Config{NetworkMode: "development"}, wantErr: true},
		{name: "enterprise tag required", config: Config{NetworkMode: "enterprise", Enterprise: true}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolvePublicURL(&tt.config)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("resolvePublicURL() = %q, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolvePublicURL(): %v", err)
			}
			if got != tt.want {
				t.Fatalf("resolvePublicURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolvePublicURLExplicitOverride(t *testing.T) {
	t.Setenv("KNIRV_PUBLIC_URL", "https://custom.example/")
	got, err := resolvePublicURL(&Config{NetworkMode: "enterprise"})
	if err != nil {
		t.Fatalf("resolvePublicURL(): %v", err)
	}
	if got != "https://custom.example" {
		t.Fatalf("resolvePublicURL() = %q, want explicit origin", got)
	}
}

func TestSetChildEnvReplacesInheritedRole(t *testing.T) {
	env := []string{"PATH=/bin", "CHAIN_NODE_ROLE=Root", "VALUE=kept"}
	got := setChildEnv(env, "CHAIN_NODE_ROLE", "Client")

	want := []string{"PATH=/bin", "VALUE=kept", "CHAIN_NODE_ROLE=Client"}
	if len(got) != len(want) {
		t.Fatalf("setChildEnv() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("setChildEnv()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
