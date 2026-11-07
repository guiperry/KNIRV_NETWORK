package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	sdk "github.com/guiperry/knirv-sdk-go/signing"
	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/logical"
)

func testEnvelope(domain, purpose, chainID string) sdk.MessageEnvelope {
	now := time.Now()
	return sdk.MessageEnvelope{
		SchemaVersion: "1",
		Domain:        domain,
		Purpose:       purpose,
		ChainID:       chainID,
		Nonce:         "n1",
		IssuedAtUnix:  now.Unix() - 10,
		ExpiresAtUnix: now.Unix() + 600,
	}
}

// TestManagedKeyPolicyMatchesAllowlists verifies the production wiring maps each managed
// plugin key to the exact DEC-1 allowlist: bootnode-master -> standby (replication-pull
// only); root-owner and failover-identity -> active (replication-head + authority
// purposes). Unmanaged names are rejected.
func TestManagedKeyPolicyMatchesAllowlists(t *testing.T) {
	standby, ok := ManagedKeyPolicy("bootnode-master", "chain-x")
	if !ok {
		t.Fatal("bootnode-master must be a managed key")
	}
	if len(standby.Messages) != len(StandbyAllowlist()) {
		t.Fatalf("bootnode-master policy mismatch: %+v", standby.Messages)
	}
	for _, m := range StandbyAllowlist() {
		if !messageAllowed(standby, m.Domain, m.Purpose, "chain-x") {
			t.Fatalf("standby policy must allow %s/%s", m.Domain, m.Purpose)
		}
	}

	for _, name := range []string{"root-owner", "failover-identity"} {
		active, ok := ManagedKeyPolicy(name, "chain-x")
		if !ok {
			t.Fatalf("%s must be a managed key", name)
		}
		if len(active.Messages) != len(ActiveAllowlist()) {
			t.Fatalf("%s policy size mismatch: %+v", name, active.Messages)
		}
	}

	if _, ok := ManagedKeyPolicy("random-key", "chain-x"); ok {
		t.Fatal("unmanaged key must not resolve to a managed policy")
	}
}

// TestPluginAllowlistEnforcesDEC1 exercises the real plugin authorization gate
// (Policy.allowMessage — the exact function signerService.message/direct call) for the
// DEC-1 allowlists attached by ProvisionManagedKey. A mirrored standby Oracle may only
// sign replication-pull; an active Oracle may sign the authority purposes but not
// replication-pull; node/validator writes are never in any allowlist and are denied.
//
// NOTE: full ProvisionManagedKey/key-generation requires the in-process SDK core, which is
// unavailable in this headless test environment (pre-existing: TestSignerService... also
// fails at key generation). This test covers the server-side deny-by-default policy gate
// that the provisioned key would carry.
func TestPluginAllowlistEnforcesDEC1(t *testing.T) {
	const chain = "chain-x"

	standby, _ := ManagedKeyPolicy("bootnode-master", chain)
	if err := standby.allowMessage(testEnvelope(DomainReplication, PurposeReplicationPull, chain)); err != nil {
		t.Fatalf("standby must allow replication-pull: %v", err)
	}
	standbyDenied := []struct{ domain, purpose string }{
		{DomainAuthority, PurposeAuthorityRecord},
		{DomainAuthority, PurposeBondRelease},
		{DomainAuthority, PurposeBondHold},
		{DomainReplication, PurposeReplicationHead},
		{DomainAuthority, "root-owner"},
		{DomainAuthority, "failover-identity"},
		{"knirv.node", "validator-set"},      // node/validator write
		{"knirv.validator", "validator-set"}, // node/validator write
	}
	for _, d := range standbyDenied {
		if err := standby.allowMessage(testEnvelope(d.domain, d.purpose, chain)); err == nil {
			t.Fatalf("standby must DENY (%s,%s)", d.domain, d.purpose)
		}
	}

	active, _ := ManagedKeyPolicy("root-owner", chain)
	if err := active.allowMessage(testEnvelope(DomainAuthority, PurposeAuthorityRecord, chain)); err != nil {
		t.Fatalf("active must allow authority-record: %v", err)
	}
	// Active extends (does not narrow) standby, so replication-pull stays allowed.
	if err := active.allowMessage(testEnvelope(DomainReplication, PurposeReplicationPull, chain)); err != nil {
		t.Fatalf("active must still allow replication-pull: %v", err)
	}
	// But an unlisted purpose (e.g. node/validator write or bootnode-enroll) is denied.
	activeDenied := []struct{ domain, purpose string }{
		{"knirv.node", "validator-set"},
		{DomainAuthority, "bootnode-enroll"},
	}
	for _, d := range activeDenied {
		if err := active.allowMessage(testEnvelope(d.domain, d.purpose, chain)); err == nil {
			t.Fatalf("active must DENY (%s,%s)", d.domain, d.purpose)
		}
	}
}

// TestProvisionManagedKeyWiring confirms the production caller applies the decision
// allowlist to a real plugin key. It is skipped in environments where the SDK core cannot
// generate keys (pre-existing headless limitation); the policy it would attach is covered
// by TestPluginAllowlistEnforcesDEC1.
func TestProvisionManagedKeyWiring(t *testing.T) {
	ctx := context.Background()
	storage := &logical.InmemStorage{}
	svc := &signerService{}

	err := ProvisionManagedKey(ctx, svc, storage, "bootnode-master", "node-identity", "chain-x")
	if err != nil {
		// No silent skip: by default an unavailable core fails the test loudly so CI
		// never treats unverified key wiring as green. Opt-in only when an explicit
		// env (set by a CI that intentionally has no wasm build) allows it.
		if strings.Contains(err.Error(), "invalid private key") && os.Getenv("KNIRV_TEST_ALLOW_SKIP") == "1" {
			t.Skip("SDK core unavailable in headless env; key generation cannot run (opt-in skip via KNIRV_TEST_ALLOW_SKIP)")
		}
		t.Fatalf("provision bootnode-master: %v", err)
	}
	pol, err := loadPolicy(ctx, storage, "bootnode-master")
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}
	if err := pol.allowMessage(testEnvelope(DomainAuthority, PurposeAuthorityRecord, "chain-x")); err == nil {
		t.Fatal("provisioned standby key must deny authority-record")
	}
}

// TestManagedKeyHandlerRoutesThroughDEC1Policy is an end-to-end, in-process test (no
// listener) that drives the real plugin HTTP handler for `keys/{name}` and asserts a
// managed key's creation routes through ProvisionManagedKey and attaches the correct
// DEC-1 server-side allowlist: bootnode-master -> standby (denies authority-record);
// root-owner -> active (allows authority-record). This proves the production handler
// path applies the managed policy, not just the helper in isolation.
func TestManagedKeyHandlerRoutesThroughDEC1Policy(t *testing.T) {
	ctx := context.Background()
	storage := &logical.InmemStorage{}
	svc := &signerService{}
	paths := candidatePaths(svc)

	var keyPath *framework.Path
	for _, p := range paths {
		if strings.HasPrefix(p.Pattern, "keys/") && !strings.Contains(p.Pattern, "/policy") {
			keyPath = p
			break
		}
	}
	if keyPath == nil {
		t.Fatal("keys/{name} handler path not found")
	}

	nameField := &framework.FieldSchema{Type: framework.TypeString}
	typeField := &framework.FieldSchema{Type: framework.TypeString}
	pkField := &framework.FieldSchema{Type: framework.TypeString}
	chainField := &framework.FieldSchema{Type: framework.TypeString}

	create := func(name, kind, chain string) {
		t.Helper()
		d := &framework.FieldData{
			Raw:    map[string]interface{}{"name": name, "type": kind, "private_key": "", "chain_id": chain},
			Schema: map[string]*framework.FieldSchema{"name": nameField, "type": typeField, "private_key": pkField, "chain_id": chainField},
		}
		resp, err := keyPath.Callbacks[logical.UpdateOperation](ctx, &logical.Request{Storage: storage}, d)
		if err != nil {
			t.Fatalf("create %s via handler: %v", name, err)
		}
		if managed, _ := resp.Data["managed"].(bool); !managed {
			t.Fatalf("managed key %s handler response should mark managed=true", name)
		}
	}

	// bootnode-master -> standby policy must deny authority-record.
	create("bootnode-master", "node-identity", "chain-x")
	standby, err := loadPolicy(ctx, storage, "bootnode-master")
	if err != nil {
		t.Fatalf("load bootnode-master policy: %v", err)
	}
	if err := standby.allowMessage(testEnvelope(DomainAuthority, PurposeAuthorityRecord, "chain-x")); err == nil {
		t.Fatal("bootnode-master handler must attach standby policy denying authority-record")
	}

	// root-owner -> active policy must allow authority-record.
	create("root-owner", "service", "chain-x")
	active, err := loadPolicy(ctx, storage, "root-owner")
	if err != nil {
		t.Fatalf("load root-owner policy: %v", err)
	}
	if err := active.allowMessage(testEnvelope(DomainAuthority, PurposeAuthorityRecord, "chain-x")); err != nil {
		t.Fatalf("root-owner handler must attach active policy allowing authority-record: %v", err)
	}

	// A non-managed key must NOT be marked managed.
	d := &framework.FieldData{
		Raw:    map[string]interface{}{"name": "custom-key", "type": "service", "private_key": "", "chain_id": "chain-x"},
		Schema: map[string]*framework.FieldSchema{"name": nameField, "type": typeField, "private_key": pkField, "chain_id": chainField},
	}
	resp, err := keyPath.Callbacks[logical.UpdateOperation](ctx, &logical.Request{Storage: storage}, d)
	if err != nil {
		t.Fatalf("create custom-key via handler: %v", err)
	}
	if managed, _ := resp.Data["managed"].(bool); managed {
		t.Fatal("non-managed key must not be marked managed")
	}
	if _, err := loadPolicy(ctx, storage, "custom-key"); err == nil {
		t.Fatal("non-managed key must not carry a managed policy")
	}
}
