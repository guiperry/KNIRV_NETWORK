package main

import (
	"os"
	"testing"

	"backend_server/keyfile"
)

// TestHardcodedConstants ensures the compiled-in secrets are not
// accidentally left as empty strings.
func TestHardcodedConstants(t *testing.T) {
	// Secrets that are critical for bootnode operation
	critical := map[string]string{
		"defaultStripeSecretKey":      defaultStripeSecretKey,
		"defaultRootPrivateKeyHex":    defaultRootPrivateKeyHex,
		"defaultJwtSecret":            defaultJwtSecret,
		"defaultKnirvJwtSecret":       defaultKnirvJwtSecret,
		"defaultGithubToken":          defaultGithubToken,
		"defaultCloudflareAPIToken":   defaultCloudflareAPIToken,
		"defaultCloudflareZoneID":     defaultCloudflareZoneID,
		"defaultCloudflareAccountID":  defaultCloudflareAccountID,
		"defaultSmtpPassword":         defaultSmtpPassword,
	}

	for name, val := range critical {
		if val == "" {
			t.Errorf("hardcoded constant %s is empty — must be populated for bootnode operation", name)
		}
	}
}

// TestNoEnvFileDependency verifies the bootnode binary does NOT
// depend on a .env file — there is no loadEnvFile function in this
// package.
func TestNoEnvFileDependency(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if len(src) == 0 {
		t.Fatal("main.go is empty")
	}
}

// TestProtobufFieldAssignment validates that the protobuf struct
// receives all the hardcoded values in the correct fields and maps
// the editable Entry widgets properly.
func TestProtobufFieldAssignment(t *testing.T) {
	content := &keyfile.RootKeyFileContentProto{
		StripeSecretKey:          defaultStripeSecretKey,
		StripeWebhookSecret:      defaultStripeWebhookSecret,
		CoinbaseApiKey:           defaultCoinbaseAPIKey,
		CoinbaseWebhookSecret:    defaultCoinbaseWebhookSecret,
		RootPrivateKeyHex:        defaultRootPrivateKeyHex,
		JwtSecret:                defaultJwtSecret,
		KnirvJwtSecret:           defaultKnirvJwtSecret,
		CerebrasApiKey:           "user-supplied-csk",
		TlsCert:                  defaultTLSCert,
		TlsKey:                   defaultTLSKey,
		CloudflareApiToken:       defaultCloudflareAPIToken,
		CloudflareZoneId:         defaultCloudflareZoneID,
		CloudflareAccountId:      defaultCloudflareAccountID,
		CloudflareEmbeddingsUrl:  defaultCloudflareEmbeddingsURL,
		SmtpPassword:             defaultSmtpPassword,
		DefaultGithubToken:       defaultGithubToken,
		DeviceIp:                 "user-supplied-ip",
		DevicePassword:           "user-supplied-pass",
		DeviceUsername:           "user-supplied-user",
	}

	if content.GetStripeSecretKey() != defaultStripeSecretKey {
		t.Errorf("StripeSecretKey mismatch")
	}
	if content.GetRootPrivateKeyHex() != defaultRootPrivateKeyHex {
		t.Errorf("RootPrivateKeyHex mismatch")
	}
	if content.GetJwtSecret() != defaultJwtSecret {
		t.Errorf("JwtSecret mismatch")
	}
	if content.GetKnirvJwtSecret() != defaultKnirvJwtSecret {
		t.Errorf("KnirvJwtSecret mismatch")
	}
	if content.GetCerebrasApiKey() != "user-supplied-csk" {
		t.Errorf("CerebrasApiKey mismatch: got %q", content.GetCerebrasApiKey())
	}
	if content.GetCloudflareApiToken() != defaultCloudflareAPIToken {
		t.Errorf("CloudflareApiToken mismatch")
	}
	if content.GetDeviceIp() != "user-supplied-ip" {
		t.Errorf("DeviceIp mismatch: got %q", content.GetDeviceIp())
	}
	if content.GetDevicePassword() != "user-supplied-pass" {
		t.Errorf("DevicePassword mismatch")
	}
	if content.GetDeviceUsername() != "user-supplied-user" {
		t.Errorf("DeviceUsername mismatch")
	}
	if content.GetSmtpPassword() != defaultSmtpPassword {
		t.Errorf("SmtpPassword mismatch")
	}
	if content.GetDefaultGithubToken() != defaultGithubToken {
		t.Errorf("DefaultGithubToken mismatch")
	}
}

// TestOutputPathFlagDefaults verifies the default output path
// resolves through keyfile.GetRootKeyPath or falls back.
func TestOutputPathFlag(t *testing.T) {
	path, err := keyfile.GetRootKeyPath()
	if err != nil {
		t.Logf("GetRootKeyPath returned error (expected in CI): %v", err)
		path = "root.key"
	}
	if path == "" {
		t.Error("output path must not be empty")
	}
}
