package main

import (
	"context"
	"errors"
	"testing"

	"github.com/guiperry/knirv-sdk-go/secrets"
	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
	"go.uber.org/zap"
)

// fakeValidatorSigner is a test double for knirvsigning.RemoteSigner. It never
// touches the Rust core (no FFI), so the test runs without a seeded wasm build.
type fakeValidatorSigner struct {
	addr   string
	pubkey []byte
}

func (f fakeValidatorSigner) Address() string { return f.addr }

func (f fakeValidatorSigner) PublicKey() []byte { return f.pubkey }

func (f fakeValidatorSigner) SignDirect(ctx context.Context, req knirvsigning.SignRequest) (knirvsigning.SignedTransaction, error) {
	return knirvsigning.SignedTransaction{}, nil
}

func (f fakeValidatorSigner) SignMessage(ctx context.Context, env knirvsigning.MessageEnvelope) (knirvsigning.SignedMessage, error) {
	return knirvsigning.SignedMessage{}, nil
}

// fakeProvider is a secrets.Provider used to drive buildValidatorSigner (the real
// production constructor for the Oracle's validator identity) without a live OpenBao
// or key file. It records which resolverMax was handed to ValidatorSettings so the
// tests can prove the role-resolver ceiling is threaded through.
type fakeProvider struct {
	signer        knirvsigning.RemoteSigner
	settings      *secrets.ValidatorSettingsResult
	signerErr     error
	settingsErr   error
	capturedMax   string
}

func (f *fakeProvider) Root(ctx context.Context) (*secrets.RootSecrets, error) {
	return nil, nil
}

func (f *fakeProvider) Boot(ctx context.Context) (*secrets.BootSecrets, error) {
	return nil, nil
}

func (f *fakeProvider) Signer(ctx context.Context, name string) (knirvsigning.RemoteSigner, error) {
	if f.signerErr != nil {
		return nil, f.signerErr
	}
	return f.signer, nil
}

func (f *fakeProvider) Health(ctx context.Context) secrets.Status {
	return secrets.Status{}
}

func (f *fakeProvider) ValidatorSettings(ctx context.Context, resolverMax string) (*secrets.ValidatorSettingsResult, error) {
	f.capturedMax = resolverMax
	if f.settingsErr != nil {
		return nil, f.settingsErr
	}
	return f.settings, nil
}

// TestBuildValidatorSignerWiresProvider proves the production path resolves the
// validator RemoteSigner from secrets.Provider (not from the root-key content):
// buildValidatorSigner is the real constructor logic, driven here by a fake
// provider rather than a test fixture config.
func TestBuildValidatorSignerWiresProvider(t *testing.T) {
	signer := fakeValidatorSigner{addr: "knirv1w508d6qejxtdg4y5r3zarvary0c5xw7kaqx36e", pubkey: []byte("vb")}
	fp := &fakeProvider{
		signer:   signer,
		settings: &secrets.ValidatorSettingsResult{Enabled: true, OracleModeMax: "active"},
	}

	got, err := buildValidatorSigner(context.Background(), fp, zap.NewNop())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected a validator signer from the provider; got nil")
	}
	if got.Address() != signer.Address() {
		t.Fatalf("validator signer not wired from provider: got %q want %q", got.Address(), signer.Address())
	}
}

// TestBuildValidatorSignerThreadsResolverMax proves the role-resolver ceiling (8.1)
// is passed into Provider.ValidatorSettings so the lowering-only clamp can apply.
func TestBuildValidatorSignerThreadsResolverMax(t *testing.T) {
	t.Setenv("ORACLE_MODE", "standby")
	signer := fakeValidatorSigner{addr: "knirv1w508d6qejxtdg4y5r3zarvary0c5xw7kaqx36e", pubkey: []byte("vb")}
	fp := &fakeProvider{
		signer:   signer,
		settings: &secrets.ValidatorSettingsResult{Enabled: true, OracleModeMax: "standby"},
	}

	if _, err := buildValidatorSigner(context.Background(), fp, zap.NewNop()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fp.capturedMax != "standby" {
		t.Fatalf("resolver ceiling not threaded to ValidatorSettings: got %q want %q", fp.capturedMax, "standby")
	}
}

// TestBuildValidatorSignerOffDisables proves oracle_mode_max=off yields no signer
// (the Oracle stays disabled rather than starting with a wrong identity).
func TestBuildValidatorSignerOffDisables(t *testing.T) {
	signer := fakeValidatorSigner{addr: "knirv1w508d6qejxtdg4y5r3zarvary0c5xw7kaqx36e", pubkey: []byte("vb")}
	fp := &fakeProvider{
		signer:   signer,
		settings: &secrets.ValidatorSettingsResult{Enabled: true, OracleModeMax: "off"},
	}

	got, err := buildValidatorSigner(context.Background(), fp, zap.NewNop())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected no validator signer when oracle_mode_max=off; got %v", got)
	}
}

// TestBuildValidatorSignerSignerError proves a provider that cannot produce the
// "validation" identity fails loudly (no silent wrong identity).
func TestBuildValidatorSignerSignerError(t *testing.T) {
	fp := &fakeProvider{
		signerErr: errors.New("no validation identity"),
		settings:  &secrets.ValidatorSettingsResult{Enabled: true, OracleModeMax: "active"},
	}

	got, err := buildValidatorSigner(context.Background(), fp, zap.NewNop())
	if err == nil {
		t.Fatal("expected error when the validation signer is unavailable")
	}
	if got != nil {
		t.Fatalf("expected nil signer on error; got %v", got)
	}
}

// TestBuildValidatorSignerSettingsError proves a provider settings failure is
// surfaced rather than swallowed.
func TestBuildValidatorSignerSettingsError(t *testing.T) {
	fp := &fakeProvider{
		settingsErr: errors.New("bao unavailable"),
	}

	got, err := buildValidatorSigner(context.Background(), fp, zap.NewNop())
	if err == nil {
		t.Fatal("expected error when ValidatorSettings fails")
	}
	if got != nil {
		t.Fatalf("expected nil signer on error; got %v", got)
	}
}
