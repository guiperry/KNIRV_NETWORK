package bootkey

import (
	"context"
	"fmt"
	"os"

	"github.com/guiperry/knirv-sdk-go/secrets"
)

// Content holds fields extracted from a decrypted boot.key file.
// Field numbers mirror BootnodeKeyFileContentProto in bootnode_key.proto.
type Content struct {
	RegistrationID               string // field 2
	JWTSecret                    string // field 10
	GeminiAPIKey                 string // field 11
	DeepseekAPIKey               string // field 12
	CerebrasAPIKey               string // field 13
	CloudflareAPIToken           string // field 18
	CloudflareZoneID             string // field 19
	CloudflareAccountID          string // field 20
	CloudflareMainnetTunnelToken string // field 21
	DeviceIP                     string // field 22
	DevicePassword               string // field 23
	DeviceUsername               string // field 24
	CloudflareTestnetTunnelToken string // field 28
	CloudflareOracleTunnelToken  string // field 29; bootnode continuity credential
	MasterWalletKeyHex           string // field 1 — master wallet key; used as the Validation Chain checkpoint signer identity on Bootnodes that have no root.key (see startValidationChain in main.go)
}

// CandidatePaths exposes boot.key search paths for diagnostic error messages.
func CandidatePaths() []string { return secrets.BootSearchPaths() }

// RootKeyCandidatePaths exposes root.key search paths for diagnostic error messages.
func RootKeyCandidatePaths() []string { return secrets.RootSearchPaths() }

// Exists returns true if boot.key is found in any expected location.
func Exists() bool {
	path := secrets.FindBoot()
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// FindRootKey returns the first root.key path found, or empty string.
func FindRootKey() string { return secrets.FindRoot() }

// FindBootKey returns the first boot.key path found, or empty string.
func FindBootKey() string { return secrets.FindBoot() }

// RootKeyExists returns true if root.key is found in any expected location.
func RootKeyExists() bool { return FindRootKey() != "" }

// RootKeyCloudflareCreds holds the Cloudflare fields extracted from a decrypted
// root.key file. Field numbers mirror RootKeyFileContentProto in root_key.proto
// (KNIRV_CORP/packages/server/backend_server/internal/proto/root_key.proto) — NOT the
// same field numbers as BootnodeKeyFileContentProto (see Content above); the two
// protos assign cloudflare_account_id / cloudflare_mainnet_tunnel_token to
// different field numbers.
type RootKeyCloudflareCreds struct {
	CloudflareAPIToken           string // field 18
	CloudflareZoneID             string // field 19
	CloudflareAccountID          string // field 26
	CloudflareMainnetTunnelToken string // field 27
	CloudflareOracleTunnelTok    string // field 28
	CloudflareTestnetTunnelToken string // field 38
	RootPrivateKeyHex            string // field 5 — root node private key; used as the Validation Chain checkpoint signer identity on Root nodes (see startValidationChain in main.go)
}

// LoadRootKeyCloudflareCreds finds root.key, decrypts it using the
// ORACLE_KEY_PASSWORD environment variable, and returns its Cloudflare fields.
// Returns nil if root.key is not found or no password is set.
func LoadRootKeyCloudflareCreds() (*RootKeyCloudflareCreds, error) {
	found := FindRootKey()
	if found == "" {
		return nil, fmt.Errorf("root.key not found in any expected location")
	}
	root, err := secrets.NewFileProvider().Root(context.Background())
	if err != nil {
		return nil, err
	}
	if root == nil {
		return nil, fmt.Errorf("root.key not found in any expected location")
	}
	return &RootKeyCloudflareCreds{
		CloudflareAPIToken:           root.Cloudflare.APIToken,
		CloudflareZoneID:             root.Cloudflare.ZoneID,
		CloudflareAccountID:          root.Cloudflare.AccountID,
		CloudflareMainnetTunnelToken: root.Cloudflare.MainnetTunnelToken,
		CloudflareOracleTunnelTok:    root.Cloudflare.OracleTunnelToken,
		CloudflareTestnetTunnelToken: root.Cloudflare.TestnetTunnelToken,
		RootPrivateKeyHex:            root.RootPrivateKeyHex,
	}, nil
}

// Load finds boot.key, decrypts it using the BOOT_KEY_PASSWORD environment
// variable, and returns its contents. Returns nil if boot.key is not found
// or no password is set.
func Load() (*Content, error) {
	found := secrets.FindBoot()
	if found == "" {
		return nil, fmt.Errorf("boot.key not found in any expected location")
	}
	boot, err := secrets.NewFileProvider().Boot(context.Background())
	if err != nil {
		return nil, err
	}
	if boot == nil {
		return nil, fmt.Errorf("boot.key not found in any expected location")
	}
	return &Content{
		RegistrationID:               boot.RegistrationID,
		JWTSecret:                    boot.JWTSecret,
		GeminiAPIKey:                 boot.GeminiAPIKey,
		DeepseekAPIKey:               boot.DeepseekAPIKey,
		CerebrasAPIKey:               boot.CerebrasAPIKey,
		CloudflareAPIToken:           boot.Cloudflare.APIToken,
		CloudflareZoneID:             boot.Cloudflare.ZoneID,
		CloudflareAccountID:          boot.Cloudflare.AccountID,
		CloudflareMainnetTunnelToken: boot.Cloudflare.MainnetTunnelToken,
		DeviceIP:                     boot.DeviceIP,
		DevicePassword:               boot.DevicePassword,
		DeviceUsername:               boot.DeviceUsername,
		CloudflareTestnetTunnelToken: boot.Cloudflare.TestnetTunnelToken,
		CloudflareOracleTunnelToken:  boot.Cloudflare.OracleTunnelToken,
		MasterWalletKeyHex:           boot.MasterWalletKeyHex,
	}, nil
}
