package authority

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/guiperry/knirv-sdk-go/secrets"
	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
)

func assembleWith(t *testing.T, p secrets.Provider, dir string, poster OraclePoster) (*Service, *Anchor, *logBuf) {
	t.Helper()
	logs := &logBuf{}
	svc, anchor := Assemble(context.Background(), AssembleConfig{Dir: dir, Mode: ModeObserve, Provider: p, Poster: poster, Logf: logs.Logf})
	if svc == nil {
		t.Fatal("Assemble returned no service")
	}
	if st := svc.Store(); st != nil {
		t.Cleanup(func() { st.Close() })
	}
	return svc, anchor, logs
}

func TestAssembleRootAndBootNodesProduceVerifyingRecords(t *testing.T) {
	for _, tc := range []struct {
		kind     string
		provider *fakeProvider
		address  func() string
	}{
		{KindRoot, &fakeProvider{root: &secrets.RootSecrets{RootPrivateKeyHex: "00"}, signers: map[string]knirvsigning.RemoteSigner{"root": testSigner(t, 1)}}, nil},
		{KindBoot, &fakeProvider{bootS: &secrets.BootSecrets{MasterWalletKeyHex: "00"}, signers: map[string]knirvsigning.RemoteSigner{"boot": testSigner(t, 2)}}, nil},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			svc, anchor, _ := assembleWith(t, tc.provider, t.TempDir(), &flakyPoster{})
			if anchor == nil || svc.Status().State != "ok" || svc.Status().Kind != tc.kind {
				t.Fatalf("status %+v anchor %v", svc.Status(), anchor)
			}
			res, err := svc.Authorize(context.Background(), act(1, "evidence.commit", "", `{"ok":true}`))
			if err != nil || !res.Stored.Verified {
				t.Fatalf("%v %+v", err, res)
			}
		})
	}
}

func TestAssembleDegradesInsteadOfFailing(t *testing.T) {
	rootOK := &fakeProvider{root: &secrets.RootSecrets{RootPrivateKeyHex: "00"}, signers: map[string]knirvsigning.RemoteSigner{"root": testSigner(t, 1)}}

	t.Run("client node with no key", func(t *testing.T) {
		svc, anchor, logs := assembleWith(t, &fakeProvider{}, t.TempDir(), &flakyPoster{})
		if st := svc.Status(); st.State != "degraded" || !strings.Contains(st.Reason, "no root.key or boot.key") || anchor != nil {
			t.Fatalf("%+v", st)
		}
		if strings.Contains(logs.String(), "WARN") {
			t.Fatalf("a keyless client node is normal, not a warning: %q", logs.String())
		}
	})
	t.Run("locked root.key", func(t *testing.T) {
		svc, _, logs := assembleWith(t, &fakeProvider{rootErr: errors.New("ORACLE_KEY_PASSWORD is not set"), bootS: &secrets.BootSecrets{MasterWalletKeyHex: "00"}}, t.TempDir(), nil)
		if st := svc.Status(); st.State != "degraded" || !strings.Contains(st.Reason, "no fallback") {
			t.Fatalf("%+v", st)
		}
		if !strings.Contains(logs.String(), "WARN") {
			t.Fatal("a locked root.key must be a warning")
		}
	})
	t.Run("invalid policy.json is not silently replaced", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "policy.json"), []byte(`{"version":1,"epoch":0}`), 0o600)
		svc, _, _ := assembleWith(t, rootOK, dir, nil)
		if st := svc.Status(); st.State != "degraded" || !strings.Contains(st.Reason, "policy") {
			t.Fatalf("%+v", st)
		}
	})
	t.Run("corrupt store refuses to append", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, recordsFile), []byte("garbage\n{\"record\":{}}\n"), 0o600)
		svc, _, _ := assembleWith(t, rootOK, dir, nil)
		if st := svc.Status(); st.State != "degraded" || !strings.Contains(st.Reason, "corrupt") {
			t.Fatalf("%+v", st)
		}
		if _, err := svc.Authorize(context.Background(), act(1, "evidence.commit", "", `{}`)); !errors.Is(err, ErrDegraded) {
			t.Fatalf("want ErrDegraded, got %v", err)
		}
	})
	t.Run("no poster means no anchoring but a working service", func(t *testing.T) {
		svc, anchor, _ := assembleWith(t, rootOK, t.TempDir(), nil)
		if anchor != nil || svc.Status().State != "ok" {
			t.Fatalf("%+v %v", svc.Status(), anchor)
		}
	})
}

func rootProvider(t *testing.T) *fakeProvider {
	return &fakeProvider{root: &secrets.RootSecrets{RootPrivateKeyHex: "00"}, signers: map[string]knirvsigning.RemoteSigner{"root": testSigner(t, 1)}}
}

func TestAssembleRunsTheFirstCeremony(t *testing.T) {
	svc, _, _ := assembleWith(t, rootProvider(t), t.TempDir(), nil)
	st := svc.Status()
	if st.Session == nil || st.Session.State != "active" || st.Session.Epoch != 1 || st.Session.Entropy != EntropyCSPRNG {
		t.Fatalf("%+v", st.Session)
	}
	res, err := svc.Authorize(context.Background(), act(1, "evidence.commit", "", `{}`))
	if err != nil || bytes.Equal(res.Record.SessionCert.SessionPub, svc.Signer().PublicKey()) {
		t.Fatalf("records must be signed by the session key: %v", err)
	}
}

func TestAssembleStartsFailClosedWhenTheFirstCeremonyFails(t *testing.T) {
	logs := &logBuf{}
	svc, _ := Assemble(context.Background(), AssembleConfig{Dir: t.TempDir(), Mode: ModeObserve, Provider: rootProvider(t), Entropy: ASIC{}, Logf: logs.Logf})
	defer svc.Store().Close()
	st := svc.Status()
	if st.State != "ok" || st.Session == nil || st.Session.State != "none" || st.Session.LastError == "" {
		t.Fatalf("the service must start, with no session: %+v", st)
	}
	if !strings.Contains(logs.String(), "first session ceremony failed") {
		t.Fatalf("no warning: %q", logs.String())
	}
	_, err := svc.Authorize(context.Background(), act(1, "evidence.commit", "", `{}`))
	var refusal *Refusal
	if !errors.As(err, &refusal) || refusal.Code != RefusedSession {
		t.Fatalf("want a session refusal, got %v", err)
	}
	if svc.Store().Total() != 0 {
		t.Fatal("a record was issued without a session")
	}
}
