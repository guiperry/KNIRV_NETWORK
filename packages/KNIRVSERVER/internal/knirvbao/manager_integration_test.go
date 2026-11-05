//go:build openbao

package knirvbao

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/guiperry/knirv-sdk-go/signing"
)

func TestRealOpenBaoBootstrapRestartAndSign(t *testing.T) {
	seal := make([]byte, 32)
	if _, err := rand.Read(seal); err != nil {
		t.Fatal(err)
	}
	m := NewManager(Config{Enabled: true, DataDir: t.TempDir(), ConfigDir: t.TempDir()}, staticSeal{key: seal})
	// This public test key is generated only for the isolated test server.
	key := make([]byte, 32)
	key[31] = 1
	now := time.Unix(1_750_000_000, 0)
	req := signing.SignRequest{Action: signing.Action{Action: "noop", Sender: "knirv1sender", TimestampUnix: 1}, ChainID: "dev"}
	msg := signing.MessageEnvelope{Domain: "knirv.scenes", Purpose: "scene-manifest", ChainID: "dev", Nonce: "test", IssuedAtUnix: now.Unix(), ExpiresAtUnix: now.Add(time.Minute).Unix()}
	called := false
	m.beforeFinish = func(ctx context.Context, api *baoAPI, token string) error {
		called = true
		if _, err := api.request(ctx, http.MethodPost, "knirv/keys/integration", token, map[string]any{"type": "service", "private_key": hex.EncodeToString(key)}, nil); err != nil {
			return err
		}
		policy := map[string]any{"type_urls": []string{signing.ActionTypeURL}, "chain_ids": []string{"dev"}, "messages": []map[string]string{{"domain": "knirv.scenes", "purpose": "scene-manifest"}}, "max_fee": "0", "rate_limit": 4}
		if _, err := api.request(ctx, http.MethodPost, "knirv/keys/integration/policy", token, policy, nil); err != nil {
			return err
		}
		var tx struct {
			Data signing.WireTransaction `json:"data"`
		}
		if _, err := api.request(ctx, http.MethodPost, "knirv/sign/direct/integration", token, req, &tx); err != nil {
			return err
		}
		wantTx, err := signing.SignTransaction(key, req)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(tx.Data, wantTx.Wire()) {
			t.Error("OpenBao transaction differs from local signer")
		}
		var signed struct {
			Data signing.SignedMessage `json:"data"`
		}
		if _, err := api.request(ctx, http.MethodPost, "knirv/sign/message/integration", token, msg, &signed); err != nil {
			return err
		}
		wantMsg, err := signing.SignMessage(key, msg)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(signed.Data, wantMsg) {
			t.Error("OpenBao message differs from local signer")
		}
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	t.Cleanup(func() {
		stop, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = m.Stop(stop)
	})
	if err := m.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if !called || m.Health().State != Ready {
		t.Fatal("real OpenBao did not initialize and sign")
	}
	if err := m.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if m.Health().State != Ready {
		t.Fatal("real OpenBao did not unseal after restart")
	}
	if err := m.Stop(ctx); err != nil {
		t.Fatal(err)
	}
}
