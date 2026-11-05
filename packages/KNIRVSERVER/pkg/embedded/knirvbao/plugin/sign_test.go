package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/guiperry/knirv-sdk-go/signing"
	"github.com/openbao/openbao/sdk/v2/logical"
)

func TestSignerServiceMatchesSDKAndRejectsPolicyViolations(t *testing.T) {
	ctx := context.Background()
	storage := &logical.InmemStorage{}
	meta, err := storeVersion(ctx, storage, "node", "node-identity", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := commitMetadata(ctx, storage, "node", meta); err != nil {
		t.Fatal(err)
	}
	key, err := readVersion(ctx, storage, "node", 1)
	if err != nil {
		t.Fatal(err)
	}
	policy := Policy{TypeURLs: []string{signing.ActionTypeURL}, ChainIDs: []string{"dev"}, Messages: []MessagePair{{"knirv.scenes", "scene-manifest"}}, MaxFee: "3", RateLimit: 3}
	svc := &signerService{}
	if err := svc.setPolicy(ctx, storage, "node", Policy{}); err == nil {
		t.Fatal("invalid policy accepted")
	}
	if err := svc.setPolicy(ctx, storage, "node", policy); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_750_000_000, 0)
	req := signing.SignRequest{Action: signing.Action{Action: "noop", Sender: "knirv1sender", TimestampUnix: 1}, ChainID: "dev", Fee: signing.Fee{Denom: "unrn", Amount: "3"}}
	want, err := signing.SignTransaction(key.PrivateKey, req)
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.direct(ctx, storage, "node", req, now)
	if err != nil || !reflect.DeepEqual(got, want.Wire()) {
		t.Fatalf("transaction mismatch: %v", err)
	}
	req.ChainID = "other"
	if _, err := svc.direct(ctx, storage, "node", req, now); err == nil {
		t.Fatal("wrong chain signed")
	}
	msg := signing.MessageEnvelope{Domain: "knirv.scenes", Purpose: "scene-manifest", ChainID: "dev", Nonce: "n", IssuedAtUnix: now.Unix(), ExpiresAtUnix: now.Add(time.Minute).Unix(), Payload: []byte("scene")}
	wantMsg, err := signing.SignMessage(key.PrivateKey, msg)
	if err != nil {
		t.Fatal(err)
	}
	gotMsg, err := svc.message(ctx, storage, "node", msg, now)
	if err != nil || !reflect.DeepEqual(gotMsg, wantMsg) {
		t.Fatalf("message mismatch: %v", err)
	}
	msg.Purpose = "other"
	if _, err := svc.message(ctx, storage, "node", msg, now); err == nil {
		t.Fatal("wrong purpose signed")
	}
	msg.Purpose = "scene-manifest"
	if _, err := svc.message(ctx, storage, "node", msg, now); err != nil {
		t.Fatal(err)
	}
	if _, err := (&signerService{}).message(ctx, storage, "node", msg, now); err == nil {
		t.Fatal("restart reset signing rate")
	}
}

func TestConcurrentSigningRespectsRateLimit(t *testing.T) {
	ctx := context.Background()
	storage := &logical.InmemStorage{}
	svc := &signerService{}
	if _, err := svc.createKey(ctx, storage, "concurrent", "service", ""); err != nil {
		t.Fatal(err)
	}
	policy := Policy{ChainIDs: []string{"dev"}, Messages: []MessagePair{{"domain", "purpose"}}, MaxFee: "0", RateLimit: 4}
	if err := svc.setPolicy(ctx, storage, "concurrent", policy); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1750000000, 0)
	msg := signing.MessageEnvelope{Domain: "domain", Purpose: "purpose", ChainID: "dev", Nonce: "n", IssuedAtUnix: now.Unix(), ExpiresAtUnix: now.Add(time.Minute).Unix()}
	var wg sync.WaitGroup
	results := make(chan error, 16)
	for i := 0; i < cap(results); i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := svc.message(ctx, storage, "concurrent", msg, now); results <- err }()
	}
	wg.Wait()
	close(results)
	accepted := 0
	for err := range results {
		if err == nil {
			accepted++
		}
	}
	if accepted != 4 {
		t.Fatalf("accepted %d concurrent requests, want 4", accepted)
	}
}

func TestSharedAPIContract(t *testing.T) {
	data, err := os.ReadFile("../../../../../KNIRVSDK/testvectors/secrets/knirvbao_api_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Version     int                     `json:"version"`
		KeyType     string                  `json:"key_type"`
		Policy      Policy                  `json:"policy"`
		Transaction signing.SignRequest     `json:"transaction"`
		Message     signing.MessageEnvelope `json:"message"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Version != 1 {
		t.Fatal("unknown API fixture version")
	}
	ctx := context.Background()
	storage := &logical.InmemStorage{}
	svc := &signerService{}
	if _, err := svc.createKey(ctx, storage, "fixture", fixture.KeyType, ""); err != nil {
		t.Fatal(err)
	}
	if err := svc.setPolicy(ctx, storage, "fixture", fixture.Policy); err != nil {
		t.Fatal(err)
	}
	key, err := readVersion(ctx, storage, "fixture", 1)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(fixture.Message.IssuedAtUnix, 0)
	tx, err := svc.direct(ctx, storage, "fixture", fixture.Transaction, now)
	if err != nil {
		t.Fatal(err)
	}
	wantTx, err := signing.SignTransaction(key.PrivateKey, fixture.Transaction)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(tx, wantTx.Wire()) {
		t.Fatal("transaction contract differs from SDK")
	}
	msg, err := svc.message(ctx, storage, "fixture", fixture.Message, now)
	if err != nil {
		t.Fatal(err)
	}
	wantMsg, err := signing.SignMessage(key.PrivateKey, fixture.Message)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(msg, wantMsg) {
		t.Fatal("message contract differs from SDK")
	}
}

func TestSDKDirectExamplesMatch(t *testing.T) {
	ctx := context.Background()
	storage := &logical.InmemStorage{}
	svc := &signerService{}
	// This is the same publicly known test key used by SDK direct_test.go.
	testKey := make([]byte, 32)
	testKey[31] = 1
	if _, err := svc.createKey(ctx, storage, "examples", "service", hex.EncodeToString(testKey)); err != nil {
		t.Fatal(err)
	}
	key, err := readVersion(ctx, storage, "examples", 1)
	if err != nil {
		t.Fatal(err)
	}
	policy := Policy{TypeURLs: []string{signing.ActionTypeURL}, ChainIDs: []string{"dev", "knirv-testnet-1"}, Messages: []MessagePair{{"knirv.controller", "cli-sign"}}, MaxFee: "3", RateLimit: 10}
	if err := svc.setPolicy(ctx, storage, "examples", policy); err != nil {
		t.Fatal(err)
	}
	requests := []signing.SignRequest{
		{Action: signing.Action{Action: "transfer", Sender: "knirv1sender", Recipient: "knirv1recipient", Amount: 42, Payload: []byte(`{"memo":"hello"}`), TimestampUnix: 1_750_000_000}, ChainID: "knirv-testnet-1", AccountNumber: 7, Sequence: 9, Fee: signing.Fee{Denom: "unrn", Amount: "3", GasLimit: 200_000}},
		{Action: signing.Action{Action: "noop", Sender: "knirv1sender", TimestampUnix: 1}, ChainID: "dev"},
	}
	now := time.Unix(1_750_000_000, 0)
	for _, req := range requests {
		want, err := signing.SignTransaction(key.PrivateKey, req)
		if err != nil {
			t.Fatal(err)
		}
		got, err := svc.direct(ctx, storage, "examples", req, now)
		if err != nil || !reflect.DeepEqual(got, want.Wire()) {
			t.Fatalf("SDK transaction differs: %v", err)
		}
	}
	msg := signing.MessageEnvelope{Domain: "knirv.controller", Purpose: "cli-sign", ChainID: "knirv-testnet-1", Nonce: "request-123", IssuedAtUnix: now.Unix(), ExpiresAtUnix: now.Add(5 * time.Minute).Unix(), Payload: []byte("approve transfer")}
	want, err := signing.SignMessage(key.PrivateKey, msg)
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.message(ctx, storage, "examples", msg, now)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("SDK message differs: %v", err)
	}
}
