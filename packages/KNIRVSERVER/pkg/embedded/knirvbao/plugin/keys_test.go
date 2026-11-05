package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/openbao/openbao/sdk/v2/logical"
)

func TestStoreKeyVersion(t *testing.T) {
	ctx := context.Background()
	storage := &logical.InmemStorage{}
	meta, err := storeVersion(ctx, storage, "node-key", "node-identity", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Version != 1 || meta.Type != "node-identity" {
		t.Fatalf("metadata: %+v", meta)
	}
	entry, err := storage.Get(ctx, "keys/node-key/v1")
	if err != nil || entry == nil {
		t.Fatalf("stored version missing: %v", err)
	}
	if _, err := storeVersion(ctx, storage, "node-key", "node-identity", "", 1); err == nil {
		t.Fatal("existing version was overwritten")
	}
	if err := commitMetadata(ctx, storage, "node-key", keyMetadata{Version: 2, Type: "node-identity"}); err == nil {
		t.Fatal("published nonexistent key version")
	}
	if err := commitMetadata(ctx, storage, "node-key", meta); err != nil {
		t.Fatal(err)
	}
	loaded, err := readMetadata(ctx, storage, "node-key")
	if err != nil || loaded != meta {
		t.Fatalf("metadata mismatch: %+v, %v", loaded, err)
	}
	var stored keyVersion
	if err := entry.DecodeJSON(&stored); err != nil {
		t.Fatal(err)
	}
	public, err := describeVersion(ctx, storage, "node-key", 1)
	if err != nil {
		t.Fatal(err)
	}
	if public.Address == "" || len(public.PublicKey) != 33 {
		t.Fatal("missing public metadata")
	}
	encoded, err := json.Marshal(public)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range [][]byte{stored.PrivateKey, []byte(base64.StdEncoding.EncodeToString(stored.PrivateKey)), []byte(hex.EncodeToString(stored.PrivateKey))} {
		if bytes.Contains(encoded, secret) {
			t.Fatal("metadata exposes private key")
		}
	}
	for _, name := range []string{"../other", "x/y", "", "UPPER"} {
		if _, err := storeVersion(ctx, storage, name, "service", "", 1); err == nil {
			t.Errorf("invalid name %q accepted", name)
		}
	}
	if _, err := storeVersion(ctx, storage, "key", "other", "", 1); err == nil {
		t.Fatal("invalid key type accepted")
	}
	if _, err := storeVersion(ctx, storage, "key", "service", "invalid", 1); err == nil {
		t.Fatal("invalid import accepted")
	}
}

func TestRotationPublishesANewAddress(t *testing.T) {
	ctx := context.Background()
	storage := &logical.InmemStorage{}
	svc := &signerService{}
	first, err := svc.createKey(ctx, storage, "node", "node-identity", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.createKey(ctx, storage, "node", "node-identity", ""); err == nil {
		t.Fatal("duplicate key accepted")
	}
	second, err := svc.rotateKey(ctx, storage, "node")
	if err != nil {
		t.Fatal(err)
	}
	if second.Version != 2 || first.Address == second.Address {
		t.Fatal("rotation did not create a new identity")
	}
	old, err := describeVersion(ctx, storage, "node", 1)
	if err != nil || old.Address != first.Address {
		t.Fatal("old public key unavailable")
	}
}
