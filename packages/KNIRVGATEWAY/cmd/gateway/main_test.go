package main

import (
	"testing"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/config"
)

func TestCanStartCloudflareTunnel(t *testing.T) {
	tests := []struct {
		name                  string
		apiToken, zone, token string
		want                  bool
	}{
		{name: "pre-provisioned tunnel token", token: "tunnel-token", want: true},
		{name: "API provisioning credentials", apiToken: "api-token", zone: "zone-id", want: true},
		{name: "API token without zone", apiToken: "api-token", want: false},
		{name: "zone without API token", zone: "zone-id", want: false},
		{name: "no credentials", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canStartCloudflareTunnel(tt.apiToken, tt.zone, tt.token); got != tt.want {
				t.Fatalf("canStartCloudflareTunnel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanOwnCloudflareTunnel(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
		want bool
	}{
		{name: "root production", cfg: config.Config{NetworkMode: "production", ChainNodeRole: "Root"}, want: true},
		{name: "root testnet", cfg: config.Config{NetworkMode: "testnet", ChainNodeRole: "Root"}, want: true},
		{name: "root devnet denied", cfg: config.Config{NetworkMode: "development", ChainNodeRole: "Root"}},
		{name: "bootnode testnet", cfg: config.Config{NetworkMode: "testnet", ChainNodeRole: "Bootnode"}, want: true},
		{name: "bootnode devnet", cfg: config.Config{NetworkMode: "development", ChainNodeRole: "Bootnode"}, want: true},
		{name: "bootnode production denied", cfg: config.Config{NetworkMode: "production", ChainNodeRole: "Bootnode"}},
		{name: "client denied", cfg: config.Config{NetworkMode: "testnet", ChainNodeRole: "Client"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canOwnCloudflareTunnel(&tt.cfg); got != tt.want {
				t.Fatalf("canOwnCloudflareTunnel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolvePublicEndpointDeploymentClasses(t *testing.T) {
	tests := []struct {
		name       string
		config     config.Config
		wantURL    string
		wantTunnel string
		wantErr    bool
	}{
		{name: "testnet", config: config.Config{NetworkMode: "testnet"}, wantURL: "https://testnet-gateway.knirv.network"},
		{name: "production", config: config.Config{NetworkMode: "production"}, wantURL: "https://gateway.knirv.network"},
		{name: "devnet pro", config: config.Config{NetworkMode: "development", UserIDTag: "User 42"}, wantURL: "https://devnet-user-42.knirv.network"},
		{name: "enterprise", config: config.Config{NetworkMode: "enterprise", EnterpriseMode: true, UserIDTag: "Acme_Admin"}, wantURL: "https://enterprise-acme-admin.knirv.network"},
		{name: "devnet tag required", config: config.Config{NetworkMode: "development"}, wantErr: true},
		{name: "enterprise tag required", config: config.Config{NetworkMode: "enterprise", EnterpriseMode: true}, wantErr: true},
		{name: "root testnet", config: config.Config{NetworkMode: "testnet", ChainNodeRole: "Root"}, wantURL: "https://testnet-gateway.knirv.network", wantTunnel: "knirv-testnet-gateway"},
		{name: "root production", config: config.Config{NetworkMode: "production", ChainNodeRole: "Root"}, wantURL: "https://gateway.knirv.network", wantTunnel: "knirv-gateway"},
		{name: "bootnode testnet", config: config.Config{NetworkMode: "testnet", ChainNodeRole: "Bootnode", UserIDTag: "User 42"}, wantURL: "https://testnet-user-42-gateway.knirv.network", wantTunnel: "knirv-testnet-user-42-gateway"},
		{name: "bootnode devnet", config: config.Config{NetworkMode: "development", ChainNodeRole: "Bootnode", UserIDTag: "User 42"}, wantURL: "https://devnet-user-42-gateway.knirv.network", wantTunnel: "knirv-devnet-user-42-gateway"},
		{name: "bootnode tag required", config: config.Config{NetworkMode: "testnet", ChainNodeRole: "Bootnode"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolvePublicEndpoint(&tt.config)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("resolvePublicEndpoint() = %#v, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolvePublicEndpoint(): %v", err)
			}
			if got.URL != tt.wantURL {
				t.Fatalf("resolvePublicEndpoint().URL = %q, want %q", got.URL, tt.wantURL)
			}
			if tt.wantTunnel != "" && got.TunnelName != tt.wantTunnel {
				t.Fatalf("resolvePublicEndpoint().TunnelName = %q, want %q", got.TunnelName, tt.wantTunnel)
			}
		})
	}
}
