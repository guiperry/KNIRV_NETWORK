package launcher

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"sync"
	"testing"

	"knirv-server/internal/knirvbao"

	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
)

// fakeRootSigner is an in-memory RemoteSigner. It records every SignMessage call so
// tests can prove which signer produced a root registry record. It never touches
// the Rust core, a socket, or a vault.
type fakeRootSigner struct {
	name string
	pub  []byte
	mu   sync.Mutex
	// signs counts SignMessage calls.
	signs int
}

func (f *fakeRootSigner) Address() string   { return "fake-" + f.name }
func (f *fakeRootSigner) PublicKey() []byte { return f.pub }
func (f *fakeRootSigner) SignDirect(context.Context, knirvsigning.SignRequest) (knirvsigning.SignedTransaction, error) {
	return knirvsigning.SignedTransaction{}, errors.New("fakeRootSigner: SignDirect not used")
}
func (f *fakeRootSigner) SignMessage(_ context.Context, env knirvsigning.MessageEnvelope) (knirvsigning.SignedMessage, error) {
	f.mu.Lock()
	f.signs++
	f.mu.Unlock()
	return knirvsigning.SignedMessage{
		Envelope:  env.Payload,
		Signature: []byte("sig-by-" + f.name),
		PublicKey: f.pub,
		Address:   f.Address(),
	}, nil
}
func (f *fakeRootSigner) signCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.signs
}

// fakeRootBao implements baoManager without a socket or a child process.
type fakeRootBao struct {
	status knirvbao.Status
	socket string
}

func (b *fakeRootBao) Start(context.Context) error { return nil }
func (b *fakeRootBao) Stop(context.Context) error  { return nil }
func (b *fakeRootBao) Health() knirvbao.Status     { return b.status }
func (b *fakeRootBao) Socket() string              { return b.socket }

// captureRootSignerLog redirects the standard logger into a buffer for the test.
func captureRootSignerLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prevOut := log.Writer()
	prevFlags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
	})
	return &buf
}

// stubBaoSignerSeam replaces the OpenBao signer constructor for the test and reports
// how many times it was called.
func stubBaoSignerSeam(t *testing.T, fn func(ctx context.Context, socket, token string) (knirvsigning.RemoteSigner, error)) *int {
	t.Helper()
	calls := 0
	prev := openRootOwnerBaoSigner
	openRootOwnerBaoSigner = func(ctx context.Context, socket, token string) (knirvsigning.RemoteSigner, error) {
		calls++
		return fn(ctx, socket, token)
	}
	t.Cleanup(func() { openRootOwnerBaoSigner = prev })
	return &calls
}

func readyBao() *fakeRootBao {
	return &fakeRootBao{status: knirvbao.Status{State: knirvbao.Ready}, socket: "/nonexistent/bao.sock"}
}

// rootKeyPair returns a legacy fake signer and a remote fake that shares its public
// key, mirroring a healthy OpenBao root-owner identity.
func rootKeyPair() (legacy *fakeRootSigner, remote *fakeRootSigner) {
	pub := []byte("root-owner-public-key")
	return &fakeRootSigner{name: "legacy", pub: pub}, &fakeRootSigner{name: "bao", pub: pub}
}

// TestRootRegistrySigner_BaoUp_UsesRemoteSigner: with OpenBao ready, a token, and a
// matching root-owner key, the record is signed via the remote signer and the local
// key is not used. Logs no WARN.
func TestRootRegistrySigner_BaoUp_UsesRemoteSigner(t *testing.T) {
	t.Setenv("KNIRV_BAO_TOKEN", "test-token-not-real")
	legacy, remote := rootKeyPair()
	calls := stubBaoSignerSeam(t, func(_ context.Context, socket, token string) (knirvsigning.RemoteSigner, error) {
		if socket != "/nonexistent/bao.sock" || token != "test-token-not-real" {
			t.Errorf("seam got socket=%q token-len=%d", socket, len(token))
		}
		return remote, nil
	})
	buf := captureRootSignerLog(t)

	app := &ServerApp{bao: readyBao()}
	got, source := app.rootRegistrySigner(context.Background(), legacy)
	if source != "openbao" {
		t.Fatalf("source = %q, want openbao", source)
	}
	if got != knirvsigning.RemoteSigner(remote) {
		t.Fatal("expected the OpenBao remote signer to be returned")
	}
	if *calls != 1 {
		t.Fatalf("bao signer constructor calls = %d, want 1", *calls)
	}

	// The record is actually signed by the remote signer, not the legacy one.
	if _, err := signRootRegistryRecord(context.Background(), got, knirvsigning.RegistryRecord{ChainID: "knirvroot1x", Nonce: 1}, "knirvroot1x", 1); err != nil {
		t.Fatalf("sign: %v", err)
	}
	if remote.signCount() != 1 || legacy.signCount() != 0 {
		t.Fatalf("signatures remote=%d legacy=%d, want 1/0", remote.signCount(), legacy.signCount())
	}
	if strings.Contains(buf.String(), "WARN") {
		t.Fatalf("unexpected WARN on healthy OpenBao path: %q", buf.String())
	}
}

// TestRootRegistrySigner_BaoDown_FallsBackWithWarn: with OpenBao not ready the
// legacy-local signer is used, the remote constructor is never reached, and a WARN
// names the fallback. The legacy signer is the one that signs.
func TestRootRegistrySigner_BaoDown_FallsBackWithWarn(t *testing.T) {
	t.Setenv("KNIRV_BAO_TOKEN", "test-token-not-real")
	legacy, remote := rootKeyPair()
	calls := stubBaoSignerSeam(t, func(context.Context, string, string) (knirvsigning.RemoteSigner, error) {
		return remote, nil
	})
	buf := captureRootSignerLog(t)

	bao := &fakeRootBao{status: knirvbao.Status{State: knirvbao.Degraded}, socket: "/nonexistent/bao.sock"}
	app := &ServerApp{bao: bao}
	got, source := app.rootRegistrySigner(context.Background(), legacy)
	if source != "legacy-local" {
		t.Fatalf("source = %q, want legacy-local", source)
	}
	if got != knirvsigning.RemoteSigner(legacy) {
		t.Fatal("expected the legacy signer on fallback")
	}
	if *calls != 0 {
		t.Fatalf("bao signer constructor called %d times while OpenBao is down, want 0", *calls)
	}
	if _, err := signRootRegistryRecord(context.Background(), got, knirvsigning.RegistryRecord{ChainID: "knirvroot1x", Nonce: 1}, "knirvroot1x", 1); err != nil {
		t.Fatalf("sign: %v", err)
	}
	if legacy.signCount() != 1 || remote.signCount() != 0 {
		t.Fatalf("signatures legacy=%d remote=%d, want 1/0", legacy.signCount(), remote.signCount())
	}
	logged := buf.String()
	if !strings.Contains(logged, "WARN root registry signing: legacy-local fallback") {
		t.Fatalf("missing degraded WARN, log=%q", logged)
	}
	if !strings.Contains(logged, string(knirvbao.Degraded)) {
		t.Fatalf("WARN does not name the OpenBao state, log=%q", logged)
	}
}

// TestRootRegistrySigner_BaoDown_NoToken: OpenBao ready but the launcher holds no
// token must fall back with a WARN and never call the remote constructor.
func TestRootRegistrySigner_BaoDown_NoToken(t *testing.T) {
	t.Setenv("KNIRV_BAO_TOKEN", "")
	legacy, _ := rootKeyPair()
	calls := stubBaoSignerSeam(t, func(context.Context, string, string) (knirvsigning.RemoteSigner, error) {
		t.Error("remote constructor must not run without a token")
		return nil, nil
	})
	buf := captureRootSignerLog(t)

	app := &ServerApp{bao: readyBao()}
	if _, source := app.rootRegistrySigner(context.Background(), legacy); source != "legacy-local" {
		t.Fatalf("source = %q, want legacy-local", source)
	}
	if *calls != 0 {
		t.Fatalf("constructor calls = %d, want 0", *calls)
	}
	if !strings.Contains(buf.String(), "WARN root registry signing: legacy-local fallback") {
		t.Fatalf("missing WARN, log=%q", buf.String())
	}
}

// TestRootRegistrySigner_BaoUnavailableError: Bao reports ready but the root-owner
// signer cannot be resolved. The error is logged and the legacy signer is used.
func TestRootRegistrySigner_BaoUnavailableError(t *testing.T) {
	t.Setenv("KNIRV_BAO_TOKEN", "test-token-not-real")
	legacy, _ := rootKeyPair()
	stubBaoSignerSeam(t, func(context.Context, string, string) (knirvsigning.RemoteSigner, error) {
		return nil, errors.New("root-owner not provisioned")
	})
	buf := captureRootSignerLog(t)

	app := &ServerApp{bao: readyBao()}
	got, source := app.rootRegistrySigner(context.Background(), legacy)
	if source != "legacy-local" || got != knirvsigning.RemoteSigner(legacy) {
		t.Fatalf("source=%q, want legacy-local fallback", source)
	}
	if !strings.Contains(buf.String(), "WARN root registry signing: legacy-local fallback (OpenBao root-owner unavailable: root-owner not provisioned)") {
		t.Fatalf("missing WARN with reason, log=%q", buf.String())
	}
}

// TestRootRegistrySigner_BaoKeyMismatch: a root-owner key from OpenBao that differs
// from root.key must never be used. Falls back with a WARN.
func TestRootRegistrySigner_BaoKeyMismatch(t *testing.T) {
	t.Setenv("KNIRV_BAO_TOKEN", "test-token-not-real")
	legacy, _ := rootKeyPair()
	other := &fakeRootSigner{name: "bao-other", pub: []byte("a-different-public-key")}
	stubBaoSignerSeam(t, func(context.Context, string, string) (knirvsigning.RemoteSigner, error) {
		return other, nil
	})
	buf := captureRootSignerLog(t)

	app := &ServerApp{bao: readyBao()}
	got, source := app.rootRegistrySigner(context.Background(), legacy)
	if source != "legacy-local" || got != knirvsigning.RemoteSigner(legacy) {
		t.Fatalf("mismatched key was accepted: source=%q", source)
	}
	if !strings.Contains(buf.String(), "does not match root.key") {
		t.Fatalf("missing key-mismatch WARN, log=%q", buf.String())
	}
}
