package main

import (
	"context"
	"fmt"

	"github.com/guiperry/knirv-sdk-go/signing"
	"github.com/openbao/openbao/sdk/v2/logical"
)

// Canonical, narrow signing allowlists (owner decision DEC-1, 2026-11-06). Each
// permitted signature is one explicit (domain, purpose) pair. Anything not listed
// here is denied by the plugin's deny-by-default Policy, so a widened OpenBao token
// still cannot obtain an unlisted signature. These strings are the single source of
// truth for the standby/active split; the OpenBao AppRole policies grant the paths
// that map to them, but the server-side allowlist below is what actually permits a
// signature (OpenBao.md §4.7).
const (
	DomainReplication = "knirv.replication"
	DomainAuthority   = "knirv.authority"

	PurposeReplicationPull = "replication-pull"
	PurposeReplicationHead = "replication-head"
	PurposeAuthorityRecord = "authority-record"
	PurposeBondRelease     = "bond-release"
	PurposeBondHold        = "bond-hold"
)

// StandbyAllowlist is the only signing a mirrored (failover) Oracle may do: pull
// replicated data. DEC-1 §2a.6.
func StandbyAllowlist() []MessagePair {
	return []MessagePair{{Domain: DomainReplication, Purpose: PurposeReplicationPull}}
}

// ActiveAllowlist adds replication-head (producing heads) and the individually
// named Oracle authority purposes used by the tx-chain. DEC-1 §2a.6.
func ActiveAllowlist() []MessagePair {
	return append(StandbyAllowlist(),
		MessagePair{Domain: DomainReplication, Purpose: PurposeReplicationHead},
		MessagePair{Domain: DomainAuthority, Purpose: PurposeAuthorityRecord},
		MessagePair{Domain: DomainAuthority, Purpose: PurposeBondRelease},
		MessagePair{Domain: DomainAuthority, Purpose: PurposeBondHold},
	)
}

// PolicyForAllowlist builds a deny-by-default Policy permitting exactly the given
// (domain, purpose) pairs on chainID, with a generous fee cap and rate limit.
func PolicyForAllowlist(pairs []MessagePair, chainID, maxFee string, rate int) Policy {
	if rate <= 0 {
		rate = 1
	}
	return Policy{
		ChainIDs:  []string{chainID},
		Messages:  pairs,
		MaxFee:    maxFee,
		RateLimit: rate,
	}
}

func messageAllowed(p Policy, domain, purpose, chainID string) bool {
	return p.allowMessage(signing.MessageEnvelope{Domain: domain, Purpose: purpose, ChainID: chainID}) == nil
}

// ManagedKeyRole reports whether name is a decision-managed plugin key and, if so,
// whether it requires the active (vs standby) allowlist. bootnode-master is always
// standby-scoped (replication-pull only); root-owner and failover-identity require the
// active allowlist (authority purposes). DEC-1 §2a.6.
func ManagedKeyRole(name string) (active bool, managed bool) {
	switch name {
	case "bootnode-master":
		return false, true
	case "root-owner", "failover-identity":
		return true, true
	}
	return false, false
}

// ManagedKeyPolicy returns the decision allowlist Policy that must be attached to a
// managed plugin key on chainID. Returns ok=false for unmanaged names.
func ManagedKeyPolicy(name, chainID string) (Policy, bool) {
	active, managed := ManagedKeyRole(name)
	if !managed {
		return Policy{}, false
	}
	if active {
		return PolicyForAllowlist(ActiveAllowlist(), chainID, "0", 1), true
	}
	return PolicyForAllowlist(StandbyAllowlist(), chainID, "0", 1), true
}

// ProvisionManagedKey creates a managed plugin key and applies its decision allowlist
// Policy. This is the production caller for DEC-1: every bootnode-master / root-owner /
// failover-identity key is created with the narrow server-side policy, so a widened
// OpenBao AppRole token cannot obtain an unlisted signature.
func ProvisionManagedKey(ctx context.Context, svc *signerService, storage logical.Storage, name, kind, chainID string) error {
	policy, ok := ManagedKeyPolicy(name, chainID)
	if !ok {
		return fmt.Errorf("not a managed plugin key: %s", name)
	}
	if _, err := svc.createKey(ctx, storage, name, kind, ""); err != nil {
		return err
	}
	return svc.setPolicy(ctx, storage, name, policy)
}

// RotateManagedKey rotates a managed plugin key and re-applies its decision allowlist,
// so rotation preserves the narrow server-side policy (DEC-1 §2a.6).
func RotateManagedKey(ctx context.Context, svc *signerService, storage logical.Storage, name string) error {
	if _, managed := ManagedKeyRole(name); !managed {
		return fmt.Errorf("not a managed plugin key: %s", name)
	}
	if _, err := svc.rotateKey(ctx, storage, name); err != nil {
		return err
	}
	pol, err := loadPolicy(ctx, storage, name)
	if err != nil {
		return err
	}
	chainID := ""
	if len(pol.ChainIDs) > 0 {
		chainID = pol.ChainIDs[0]
	}
	newPol, ok := ManagedKeyPolicy(name, chainID)
	if !ok {
		return fmt.Errorf("managed key policy unavailable: %s", name)
	}
	return svc.setPolicy(ctx, storage, name, newPol)
}
