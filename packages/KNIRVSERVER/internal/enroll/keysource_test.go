package enroll

import (
	"context"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/guiperry/knirv-sdk-go/secrets"
	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
)

// fakeProvider is an in-memory secrets.Provider. It never touches a vault or a
// key file; the keys are the fixed in-source test vectors.
type fakeProvider struct {
	root *secrets.RootSecrets
	boot *secrets.BootSecrets
	err  error
}

func (p fakeProvider) Root(context.Context) (*secrets.RootSecrets, error) { return p.root, p.err }
func (p fakeProvider) Boot(context.Context) (*secrets.BootSecrets, error) { return p.boot, p.err }
func (p fakeProvider) Signer(context.Context, string) (knirvsigning.RemoteSigner, error) {
	return nil, errors.New("fake provider has no signer")
}
func (p fakeProvider) Health(context.Context) secrets.Status { return secrets.Status{} }
func (p fakeProvider) ValidatorSettings(context.Context, string) (*secrets.ValidatorSettingsResult, error) {
	return nil, errors.New("fake provider has no validator settings")
}

var _ secrets.Provider = fakeProvider{}

func TestProviderKeySourceReturnsKeysFromProvider(t *testing.T) {
	src := ProviderKeySource{Provider: fakeProvider{
		root: &secrets.RootSecrets{RootPrivateKeyHex: testRootKeyHex},
		boot: &secrets.BootSecrets{MasterWalletKeyHex: testBootNodeKeyHex},
	}}
	boot, err := src.BootKey(context.Background())
	if err != nil || hex.EncodeToString(boot) != testBootNodeKeyHex {
		t.Fatalf("boot key: err=%v got=%x", err, boot)
	}
	root, err := src.RootKey(context.Background())
	if err != nil || hex.EncodeToString(root) != testRootKeyHex {
		t.Fatalf("root key: err=%v got=%x", err, root)
	}
}

func TestProviderKeySourceFailsWithoutKeys(t *testing.T) {
	ctx := context.Background()
	if _, err := (ProviderKeySource{Provider: fakeProvider{}}).BootKey(ctx); err == nil {
		t.Fatal("missing boot key must fail")
	}
	if _, err := (ProviderKeySource{Provider: fakeProvider{}}).RootKey(ctx); err == nil {
		t.Fatal("missing root key must fail")
	}
	if _, err := (ProviderKeySource{Provider: fakeProvider{err: errors.New("sealed")}}).RootKey(ctx); err == nil {
		t.Fatal("provider error must surface")
	}
	if _, err := (ProviderKeySource{}).BootKey(ctx); err == nil {
		t.Fatal("nil provider must fail")
	}
}

func TestProviderKeySourceRejectsMalformedKey(t *testing.T) {
	src := ProviderKeySource{Provider: fakeProvider{boot: &secrets.BootSecrets{MasterWalletKeyHex: "zz"}}}
	if _, err := src.BootKey(context.Background()); err == nil {
		t.Fatal("non-hex key must fail")
	}
	short := ProviderKeySource{Provider: fakeProvider{boot: &secrets.BootSecrets{MasterWalletKeyHex: "abcd"}}}
	if _, err := short.BootKey(context.Background()); err == nil {
		t.Fatal("short key must fail")
	}
}

func TestRawFileKeySourceIsDevOnlyAndChecksLength(t *testing.T) {
	dir := t.TempDir() // test-generated scratch, not a real key location
	good := filepath.Join(dir, "boot.key")
	raw, _ := hex.DecodeString(testBootNodeKeyHex)
	if err := os.WriteFile(good, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	src := RawFileKeySource{BootPath: good, RootPath: filepath.Join(dir, "missing.key")}
	if k, err := src.BootKey(context.Background()); err != nil || len(k) != 32 {
		t.Fatalf("raw boot key: err=%v len=%d", err, len(k))
	}
	if _, err := src.RootKey(context.Background()); err == nil {
		t.Fatal("missing root file must fail")
	}
	bad := filepath.Join(dir, "short.key")
	_ = os.WriteFile(bad, []byte("short"), 0o600)
	if _, err := (RawFileKeySource{BootPath: bad}).BootKey(context.Background()); err == nil {
		t.Fatal("non-32-byte raw key must fail")
	}
}

// A registration's challenge is signed by the provider's boot key, so the address
// the server derives from the challenge equals the boot identity the provider holds.
func TestProviderBootKeySignsChallengeForRegisteredIdentity(t *testing.T) {
	src := ProviderKeySource{Provider: fakeProvider{boot: &secrets.BootSecrets{MasterWalletKeyHex: testBootNodeKeyHex}}}
	key, err := src.BootKey(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	challenge, err := knirvsigning.SignBootnodeEnrollChallenge(key, testChainID, "enroll:x", 1_700_000_000, 1_700_000_600, testOperator, nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	want, _ := knirvsigning.Address(challenge.PublicKey, knirvsigning.DefaultAddressPrefix)
	direct, _ := knirvsigning.SignBootnodeEnrollChallenge(mustHex(t, testBootNodeKeyHex), testChainID, "enroll:x", 1_700_000_000, 1_700_000_600, testOperator, nil)
	got, _ := knirvsigning.Address(direct.PublicKey, knirvsigning.DefaultAddressPrefix)
	if want != got {
		t.Fatalf("provider key yields a different identity: %s vs %s", want, got)
	}
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
