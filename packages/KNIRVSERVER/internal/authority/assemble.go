package authority

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/guiperry/knirv-sdk-go/secrets"
)

// AssembleConfig is everything the launcher knows.
type AssembleConfig struct {
	// Dir holds records.jsonl, anchor.json and the optional policy.json.
	Dir      string
	Mode     string
	Provider secrets.Provider
	// Poster, when set, anchors the chain on the Oracle. Nil disables anchoring.
	Poster    OraclePoster
	Bond      uint64
	BondOwner string
	// Entropy defaults to the OS CSPRNG.
	Entropy EntropySource
	Now     func() time.Time
	Logf    func(format string, args ...any)
}

// Assemble builds the service for this node. It never returns a nil Service and
// never fails the node: any problem (no key file, locked key, bad policy, a
// corrupt store) yields a DEGRADED service that serves /authority/status with the
// reason and refuses to issue. A missing key is normal for a client node; the rest
// are logged as warnings because an operator has to fix them.
func Assemble(ctx context.Context, c AssembleConfig) (*Service, *Anchor) {
	logf := c.Logf
	if logf == nil {
		logf = func(string, ...any) {}
	}
	degrade := func(reason string) (*Service, *Anchor) {
		svc, _ := New(Config{Mode: c.Mode, DegradedReason: reason, Logf: logf})
		return svc, nil
	}

	id, err := ResolveIdentity(ctx, c.Provider)
	if err != nil {
		if errors.Is(err, ErrNoKey) {
			logf("authority: %v; the service is idle (degraded) on this node", err)
		} else {
			logf("WARN authority: %v; the service is degraded", err)
		}
		return degrade(err.Error())
	}
	policy, err := LoadPolicy(filepath.Join(c.Dir, "policy.json"))
	if err != nil {
		logf("WARN authority: %v; the service is degraded (fix or remove policy.json)", err)
		return degrade(err.Error())
	}
	store, repaired, err := OpenStore(c.Dir)
	if err != nil {
		logf("WARN authority: %v; the service is degraded and will not append to a damaged chain", err)
		return degrade(err.Error())
	}
	if repaired {
		logf("WARN authority: dropped a torn final record from %s (a crash during the last append)", c.Dir)
	}
	entropy := c.Entropy
	if entropy == nil {
		entropy = CSPRNG{}
	}
	scope := id.Signer.Address()
	if id.Kind == KindRoot {
		scope = "*"
	}
	sessions, err := NewSessionManager(SessionConfig{
		LongTerm: id.Signer, Entropy: entropy, Scope: scope,
		TTL: time.Duration(policy.CertTTLSeconds) * time.Second, EpochFloor: policy.Epoch,
		Dir: c.Dir, Now: c.Now, Logf: logf,
	})
	if err != nil {
		store.Close()
		logf("WARN authority: %v; the service is degraded", err)
		return degrade(err.Error())
	}
	// First ceremony. A failure is not fatal: the service starts without a session
	// and fails closed (T2 and T3 refused; T1 only if policy opts into break-glass)
	// until the background loop manages to run one.
	if _, err := sessions.Ensure(ctx); err != nil {
		logf("WARN authority: the first session ceremony failed: %v; records are refused until one succeeds", err)
	}
	svc, err := New(Config{Mode: c.Mode, Kind: id.Kind, Signer: id.Signer, Policy: policy, Store: store, Sessions: sessions, Now: c.Now, Logf: logf})
	if err != nil {
		store.Close()
		logf("WARN authority: %v; the service is degraded", err)
		return degrade(err.Error())
	}
	logf("authority: %s node %s, mode %s, policy %s, %d record(s) on chain %s",
		id.Kind, id.Signer.Address(), svc.Status().Mode, policy.Hash()[:12], store.Count(id.Signer.Address()), ChainID(id.Signer.Address()))

	if c.Poster == nil {
		return svc, nil
	}
	anchor, err := NewAnchor(AnchorConfig{Poster: c.Poster, Signer: id.Signer, Store: store, Bond: c.Bond, BondOwner: c.BondOwner, Now: c.Now, Logf: logf})
	if err != nil {
		logf("WARN authority: anchoring is disabled: %v", err)
		return svc, nil
	}
	return svc, anchor
}
