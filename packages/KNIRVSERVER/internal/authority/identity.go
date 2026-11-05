package authority

import (
	"context"
	"errors"
	"fmt"

	"github.com/guiperry/knirv-sdk-go/secrets"
	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
)

// ErrNoKey means the node has neither root.key nor boot.key. It is a normal
// state (a Client node), not a failure: the service starts degraded.
var ErrNoKey = errors.New("authority: no root.key or boot.key on this node")

// Identity is the node's long-term signing identity and kind.
type Identity struct {
	Kind   string
	Signer knirvsigning.RemoteSigner
}

// ResolveIdentity derives the node kind from which key file the provider finds
// (plan step 3) and returns the matching signer, consumed through the SDK
// provider unchanged.
//
// Conservative reading, marked: if a root.key exists but cannot be used (no
// password, wrong password, no signing key inside), the node does NOT fall back
// to boot.key. A root host acting as a boot node would be quietly demoted, and
// its records would then be scoped differently from what its operator expects.
// It degrades and says why. The launcher resolves root before boot the same way.
func ResolveIdentity(ctx context.Context, p secrets.Provider) (Identity, error) {
	root, err := p.Root(ctx)
	if err != nil {
		return Identity{}, fmt.Errorf("root.key present but unusable (no fallback to boot.key): %w", err)
	}
	if root != nil {
		if root.RootPrivateKeyHex == "" {
			return Identity{}, errors.New("root.key has no signing key (no fallback to boot.key)")
		}
		signer, err := p.Signer(ctx, "root")
		if err != nil {
			return Identity{}, fmt.Errorf("root signer: %w", err)
		}
		return Identity{Kind: KindRoot, Signer: signer}, nil
	}
	boot, err := p.Boot(ctx)
	if err != nil {
		return Identity{}, fmt.Errorf("boot.key present but unusable: %w", err)
	}
	if boot != nil {
		if boot.MasterWalletKeyHex == "" {
			return Identity{}, errors.New("boot.key has no master wallet key")
		}
		signer, err := p.Signer(ctx, "boot")
		if err != nil {
			return Identity{}, fmt.Errorf("boot signer: %w", err)
		}
		return Identity{Kind: KindBoot, Signer: signer}, nil
	}
	return Identity{}, ErrNoKey
}
