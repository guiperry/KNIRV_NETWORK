package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/guiperry/knirv-sdk-go/signing"
	"github.com/openbao/openbao/sdk/v2/logical"
)

// keyVersion is stored only in OpenBao's encrypted storage. It must never be
// returned from a logical endpoint or included in errors or audit data.
type keyVersion struct {
	PrivateKey []byte `json:"private_key"`
	Type       string `json:"type"`
}

type keyMetadata struct {
	Version int    `json:"version"`
	Type    string `json:"type"`
}

// publicMetadata is the only key description allowed out of the plugin.
type publicMetadata struct {
	Version   int    `json:"version"`
	Type      string `json:"type"`
	Address   string `json:"address"`
	PublicKey []byte `json:"public_key"`
}

func readVersion(ctx context.Context, storage logical.Storage, name string, version int) (keyVersion, error) {
	if !validName(name) || version < 1 {
		return keyVersion{}, errors.New("invalid key name or version")
	}
	entry, err := storage.Get(ctx, keyPath(name, version))
	if err != nil {
		return keyVersion{}, err
	}
	if entry == nil {
		return keyVersion{}, errors.New("key version not found")
	}
	var key keyVersion
	if err := entry.DecodeJSON(&key); err != nil {
		return keyVersion{}, errors.New("invalid stored key")
	}
	return key, nil
}

func describeVersion(ctx context.Context, storage logical.Storage, name string, version int) (publicMetadata, error) {
	key, err := readVersion(ctx, storage, name, version)
	if err != nil {
		return publicMetadata{}, err
	}
	signer, err := signing.NewLocalSigner(key.PrivateKey)
	if err != nil {
		return publicMetadata{}, errors.New("invalid stored key")
	}
	return publicMetadata{Version: version, Type: key.Type, Address: signer.Address(), PublicKey: signer.PublicKey()}, nil
}

func readMetadata(ctx context.Context, storage logical.Storage, name string) (keyMetadata, error) {
	if !validName(name) {
		return keyMetadata{}, errors.New("invalid key name")
	}
	entry, err := storage.Get(ctx, "keys/"+name+"/metadata")
	if err != nil {
		return keyMetadata{}, err
	}
	if entry == nil {
		return keyMetadata{}, errors.New("key not found")
	}
	var metadata keyMetadata
	if err := entry.DecodeJSON(&metadata); err != nil {
		return keyMetadata{}, errors.New("invalid key metadata")
	}
	if metadata.Version < 1 || (metadata.Type != "node-identity" && metadata.Type != "service") {
		return keyMetadata{}, errors.New("invalid key metadata")
	}
	return metadata, nil
}

// commitMetadata publishes a fully written version. If this write fails, the
// previous metadata remains authoritative and the new version is not active.
func commitMetadata(ctx context.Context, storage logical.Storage, name string, metadata keyMetadata) error {
	if !validName(name) || metadata.Version < 1 || (metadata.Type != "node-identity" && metadata.Type != "service") {
		return errors.New("invalid key metadata")
	}
	key, err := readVersion(ctx, storage, name, metadata.Version)
	if err != nil {
		return err
	}
	if key.Type != metadata.Type {
		return errors.New("key type does not match metadata")
	}
	entry, err := logical.StorageEntryJSON("keys/"+name+"/metadata", metadata)
	if err != nil {
		return err
	}
	return storage.Put(ctx, entry)
}

func keyPath(name string, version int) string {
	return fmt.Sprintf("keys/%s/v%d", name, version)
}

func validName(name string) bool {
	if len(name) == 0 || len(name) > 128 {
		return false
	}
	for _, c := range name {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
			return false
		}
	}
	return true
}

// storeVersion validates an imported or generated key before persisting it.
// Its caller must serialize version allocation and storage writes.
func storeVersion(ctx context.Context, storage logical.Storage, name, kind, imported string, version int) (keyMetadata, error) {
	if !validName(name) || (kind != "node-identity" && kind != "service") || version < 1 {
		return keyMetadata{}, errors.New("invalid key name, type or version")
	}
	existing, err := storage.Get(ctx, keyPath(name, version))
	if err != nil {
		return keyMetadata{}, err
	}
	if existing != nil {
		return keyMetadata{}, errors.New("key version already exists")
	}
	var key []byte
	if imported != "" {
		var err error
		key, err = hex.DecodeString(strings.TrimSpace(imported))
		if err != nil {
			return keyMetadata{}, errors.New("invalid private key")
		}
	} else {
		key = make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return keyMetadata{}, err
		}
	}
	if _, err := signing.NewLocalSigner(key); err != nil {
		return keyMetadata{}, errors.New("invalid private key")
	}
	entry, err := logical.StorageEntryJSON(keyPath(name, version), keyVersion{PrivateKey: key, Type: kind})
	if err != nil {
		return keyMetadata{}, err
	}
	if err := storage.Put(ctx, entry); err != nil {
		return keyMetadata{}, err
	}
	return keyMetadata{Version: version, Type: kind}, nil
}

func (s *signerService) createKey(ctx context.Context, storage logical.Storage, name, kind, imported string) (publicMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !validName(name) {
		return publicMetadata{}, errors.New("invalid key name")
	}
	entry, err := storage.Get(ctx, "keys/"+name+"/metadata")
	if err != nil {
		return publicMetadata{}, err
	}
	if entry != nil {
		return publicMetadata{}, errors.New("key already exists")
	}
	metadata, err := storeVersion(ctx, storage, name, kind, imported, 1)
	if err != nil {
		return publicMetadata{}, err
	}
	if err := commitMetadata(ctx, storage, name, metadata); err != nil {
		return publicMetadata{}, err
	}
	return describeVersion(ctx, storage, name, metadata.Version)
}

func (s *signerService) rotateKey(ctx context.Context, storage logical.Storage, name string) (publicMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	metadata, err := readMetadata(ctx, storage, name)
	if err != nil {
		return publicMetadata{}, err
	}
	if metadata.Version == int(^uint(0)>>1) {
		return publicMetadata{}, errors.New("key version exhausted")
	}
	metadata, err = storeVersion(ctx, storage, name, metadata.Type, "", metadata.Version+1)
	if err != nil {
		return publicMetadata{}, err
	}
	if err := commitMetadata(ctx, storage, name, metadata); err != nil {
		return publicMetadata{}, err
	}
	return describeVersion(ctx, storage, name, metadata.Version)
}
