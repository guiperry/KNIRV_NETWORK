package authority

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/guiperry/knirv-sdk-go/secrets"
	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
)

// fakeProvider is a secrets.Provider with scripted answers.
type fakeProvider struct {
	root, boot       *secrets.RootSecrets
	bootS            *secrets.BootSecrets
	rootErr, bootErr error
	signerErr        error
	signers          map[string]knirvsigning.RemoteSigner
}

func (f *fakeProvider) Root(context.Context) (*secrets.RootSecrets, error) { return f.root, f.rootErr }
func (f *fakeProvider) Boot(context.Context) (*secrets.BootSecrets, error) { return f.bootS, f.bootErr }
func (f *fakeProvider) Signer(_ context.Context, name string) (knirvsigning.RemoteSigner, error) {
	if f.signerErr != nil {
		return nil, f.signerErr
	}
	s, ok := f.signers[name]
	if !ok {
		return nil, errors.New("no such signer " + name)
	}
	return s, nil
}
func (f *fakeProvider) Health(context.Context) secrets.Status { return secrets.Status{} }

func TestResolveIdentity(t *testing.T) {
	rootS, bootS := testSigner(t, 1), testSigner(t, 2)
	signers := map[string]knirvsigning.RemoteSigner{"root": rootS, "boot": bootS}
	hasRoot := &secrets.RootSecrets{RootPrivateKeyHex: "00"}
	hasBoot := &secrets.BootSecrets{MasterWalletKeyHex: "00"}
	ctx := context.Background()

	t.Run("root.key wins", func(t *testing.T) {
		id, err := ResolveIdentity(ctx, &fakeProvider{root: hasRoot, bootS: hasBoot, signers: signers})
		if err != nil || id.Kind != KindRoot || id.Signer.Address() != rootS.Address() {
			t.Fatalf("%+v %v", id, err)
		}
	})
	t.Run("boot.key only", func(t *testing.T) {
		id, err := ResolveIdentity(ctx, &fakeProvider{bootS: hasBoot, signers: signers})
		if err != nil || id.Kind != KindBoot || id.Signer.Address() != bootS.Address() {
			t.Fatalf("%+v %v", id, err)
		}
	})
	t.Run("neither key is ErrNoKey", func(t *testing.T) {
		if _, err := ResolveIdentity(ctx, &fakeProvider{signers: signers}); !errors.Is(err, ErrNoKey) {
			t.Fatalf("want ErrNoKey, got %v", err)
		}
	})
	t.Run("a locked root.key does not fall back to boot.key", func(t *testing.T) {
		_, err := ResolveIdentity(ctx, &fakeProvider{rootErr: errors.New("ORACLE_KEY_PASSWORD is not set"), bootS: hasBoot, signers: signers})
		if err == nil || !strings.Contains(err.Error(), "no fallback") || errors.Is(err, ErrNoKey) {
			t.Fatalf("want a no-fallback error, got %v", err)
		}
	})
	t.Run("root.key without a signing key does not fall back either", func(t *testing.T) {
		_, err := ResolveIdentity(ctx, &fakeProvider{root: &secrets.RootSecrets{}, bootS: hasBoot, signers: signers})
		if err == nil || !strings.Contains(err.Error(), "no fallback") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("unusable boot.key", func(t *testing.T) {
		if _, err := ResolveIdentity(ctx, &fakeProvider{bootErr: errors.New("wrong password")}); err == nil {
			t.Fatal("expected an error")
		}
		if _, err := ResolveIdentity(ctx, &fakeProvider{bootS: &secrets.BootSecrets{}}); err == nil {
			t.Fatal("a boot.key without a master wallet key must be refused")
		}
	})
	t.Run("signer errors propagate", func(t *testing.T) {
		if _, err := ResolveIdentity(ctx, &fakeProvider{root: hasRoot, signerErr: errors.New("signer offline")}); err == nil {
			t.Fatal("expected an error")
		}
	})
}
