package main

import (
	"context"
	"encoding/hex"
	"testing"

	"github.com/openbao/openbao/sdk/v2/logical"
)

func TestGoldenSDKAddress(t *testing.T) {
	// Public test key and address from the SDK direct-signing vectors.
	key := make([]byte, 32)
	key[31] = 1
	service := &signerService{}
	pub, err := service.createKey(context.Background(), &logical.InmemStorage{}, "golden", "service", hex.EncodeToString(key))
	if err != nil {
		t.Fatal(err)
	}
	if pub.Address != "knirv1w508d6qejxtdg4y5r3zarvary0c5xw7kaqx36e" {
		t.Fatalf("address differs from SDK vector: %s", pub.Address)
	}
}
