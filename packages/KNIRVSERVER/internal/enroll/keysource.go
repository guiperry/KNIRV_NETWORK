package enroll

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/guiperry/knirv-sdk-go/secrets"
)

// KeySource supplies the two private keys a DEC-2 enrollment signs with: the new
// boot node's key (challenge) and the root owner's key (approval). Keys are only
// handed to the SDK signing functions; this package never inspects or derives them.
type KeySource interface {
	BootKey(ctx context.Context) ([]byte, error)
	RootKey(ctx context.Context) ([]byte, error)
}

// ProviderKeySource resolves the enrollment keys through the SDK secrets.Provider
// that every KNIRV service uses (OpenBao in production, or the file backend). It
// reads the same fields the node itself signs with: boot.key's master wallet key
// for the boot-node identity and root.key's root signing key for the approval.
type ProviderKeySource struct {
	Provider secrets.Provider
}

// BootKey returns the boot-node signing key from the provider.
func (k ProviderKeySource) BootKey(ctx context.Context) ([]byte, error) {
	if k.Provider == nil {
		return nil, errors.New("enroll: no secrets provider configured")
	}
	boot, err := k.Provider.Boot(ctx)
	if err != nil {
		return nil, fmt.Errorf("enroll: boot key from secrets provider: %w", err)
	}
	if boot == nil || boot.MasterWalletKeyHex == "" {
		return nil, errors.New("enroll: secrets provider has no boot-node key")
	}
	return decodeSigningKey(boot.MasterWalletKeyHex)
}

// RootKey returns the root owner's signing key from the provider.
func (k ProviderKeySource) RootKey(ctx context.Context) ([]byte, error) {
	if k.Provider == nil {
		return nil, errors.New("enroll: no secrets provider configured")
	}
	root, err := k.Provider.Root(ctx)
	if err != nil {
		return nil, fmt.Errorf("enroll: root key from secrets provider: %w", err)
	}
	if root == nil || root.RootPrivateKeyHex == "" {
		return nil, errors.New("enroll: secrets provider has no root signing key")
	}
	return decodeSigningKey(root.RootPrivateKeyHex)
}

// RawFileKeySource reads raw 32-byte private keys from files. It is a DEV/TEST
// fallback only (the CLI exposes it behind --dev-raw-keys). Production keys come
// from ProviderKeySource.
type RawFileKeySource struct {
	BootPath string
	RootPath string
}

// BootKey reads the boot-node key file.
func (k RawFileKeySource) BootKey(_ context.Context) ([]byte, error) {
	return readRawKey(k.BootPath, "boot")
}

// RootKey reads the root key file.
func (k RawFileKeySource) RootKey(_ context.Context) ([]byte, error) {
	return readRawKey(k.RootPath, "root")
}

func readRawKey(path, role string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("enroll: %s key path is required", role)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("enroll: read %s key: %w", role, err)
	}
	if len(data) != 32 {
		return nil, fmt.Errorf("enroll: %s key file must hold 32 raw bytes", role)
	}
	return data, nil
}

// decodeSigningKey parses a 32-byte private key from hex. It never echoes the value.
func decodeSigningKey(value string) ([]byte, error) {
	raw, err := hex.DecodeString(strings.TrimSpace(strings.TrimPrefix(value, "0x")))
	if err != nil {
		return nil, errors.New("enroll: signing key is not valid hex")
	}
	if len(raw) != 32 {
		return nil, errors.New("enroll: signing key must be 32 bytes")
	}
	return raw, nil
}
