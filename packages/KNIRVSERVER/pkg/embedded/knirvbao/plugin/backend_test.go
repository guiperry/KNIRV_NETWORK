package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/guiperry/knirv-sdk-go/signing"
	"github.com/openbao/openbao/sdk/v2/logical"
)

func TestCandidateRoutesDenyAndNeverExportKey(t *testing.T) {
	ctx := context.Background()
	store := &logical.InmemStorage{}
	b := newBackend()
	b.Paths = candidatePaths(&signerService{})
	if err := b.Setup(ctx, logical.TestBackendConfig()); err != nil {
		t.Fatal(err)
	}
	call := func(op logical.Operation, path string, data map[string]any) *logical.Response {
		t.Helper()
		response, err := b.HandleRequest(ctx, &logical.Request{Operation: op, Path: path, Data: data, Storage: store})
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if response == nil {
			t.Fatalf("%s returned nil", path)
		}
		return response
	}
	created := call(logical.UpdateOperation, "keys/node", map[string]any{"type": "node-identity"})
	if created.IsError() {
		t.Fatal("create rejected")
	}
	stored, err := readVersion(ctx, store, "node", 1)
	if err != nil {
		t.Fatal(err)
	}
	if response, err := b.HandleRequest(ctx, &logical.Request{Operation: logical.ReadOperation, Path: "keys/node/v1", Storage: store}); err == nil && response != nil && !response.IsError() {
		t.Fatal("storage version is reachable as an API path")
	}
	for _, path := range []string{"keys/node", "address/node"} {
		response := call(logical.ReadOperation, path, nil)
		encoded, err := json.Marshal(response.Data)
		if err != nil {
			t.Fatal(err)
		}
		for _, secret := range [][]byte{stored.PrivateKey, []byte(base64.StdEncoding.EncodeToString(stored.PrivateKey)), []byte(hex.EncodeToString(stored.PrivateKey))} {
			if bytes.Contains(encoded, secret) {
				t.Fatalf("%s exported private key", path)
			}
		}
	}
	now := time.Now().Unix()
	msg := map[string]any{"Domain": "knirv.scenes", "Purpose": "scene-manifest", "ChainID": "dev", "Nonce": "n", "IssuedAtUnix": now, "ExpiresAtUnix": now + 60, "Payload": base64.StdEncoding.EncodeToString([]byte("scene"))}
	if !call(logical.UpdateOperation, "sign/message/node", msg).IsError() {
		t.Fatal("signing without policy accepted")
	}
	policy := map[string]any{"type_urls": []string{signing.ActionTypeURL}, "chain_ids": []string{"dev"}, "messages": []map[string]string{{"domain": "knirv.scenes", "purpose": "scene-manifest"}}, "max_fee": "3", "rate_limit": 2}
	if call(logical.UpdateOperation, "keys/node/policy", policy).IsError() {
		t.Fatal("policy rejected")
	}
	if call(logical.UpdateOperation, "sign/message/node", msg).IsError() {
		t.Fatal("allowlisted signing rejected")
	}
	request := signing.SignRequest{Action: signing.Action{Action: "noop", Sender: "knirv1sender", TimestampUnix: 1}, ChainID: "dev", Fee: signing.Fee{Denom: "unrn", Amount: "3"}}
	requestBytes, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	var requestMap map[string]any
	if err := json.Unmarshal(requestBytes, &requestMap); err != nil {
		t.Fatal(err)
	}
	result := call(logical.UpdateOperation, "sign/direct/node", requestMap)
	want, err := signing.SignTransaction(stored.PrivateKey, request)
	if err != nil {
		t.Fatal(err)
	}
	actualBytes, err := json.Marshal(result.Data)
	if err != nil {
		t.Fatal(err)
	}
	wantBytes, err := json.Marshal(want.Wire())
	if err != nil {
		t.Fatal(err)
	}
	var actual, expected map[string]any
	if err := json.Unmarshal(actualBytes, &actual); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(wantBytes, &expected); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatal("direct wire output differs from SDK")
	}
	msg["Purpose"] = "other"
	denied := call(logical.UpdateOperation, "sign/message/node", msg)
	if !denied.IsError() {
		t.Fatal("disallowed message signed")
	}
	if denied.Data["data"] == nil {
		t.Fatal("denial lacks audit field")
	}
	rotated := call(logical.UpdateOperation, "rotate/node", nil)
	if rotated.IsError() {
		t.Fatal("rotation rejected")
	}
	second, err := readVersion(ctx, store, "node", 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, response := range []*logical.Response{created, result, denied, rotated, call(logical.ReadOperation, "keys/node/policy", nil)} {
		encoded, err := json.Marshal(response.Data)
		if err != nil {
			t.Fatal(err)
		}
		for _, secret := range [][]byte{stored.PrivateKey, second.PrivateKey} {
			for _, form := range [][]byte{secret, []byte(base64.StdEncoding.EncodeToString(secret)), []byte(hex.EncodeToString(secret))} {
				if bytes.Contains(encoded, form) {
					t.Fatal("endpoint response exported private key")
				}
			}
		}
	}
}

func TestDecodeRequestRejectsImpreciseFloat(t *testing.T) {
	var request signing.SignRequest
	if err := decodeRequest(map[string]any{"Action": map[string]any{"Amount": float64(1 << 53)}}, &request); err == nil {
		t.Fatal("imprecise amount accepted")
	}
	if err := decodeRequest(map[string]any{"Sequence": json.Number("9007199254740993")}, &request); err != nil || request.Sequence != 9007199254740993 {
		t.Fatal("exact JSON number was not preserved")
	}
}
