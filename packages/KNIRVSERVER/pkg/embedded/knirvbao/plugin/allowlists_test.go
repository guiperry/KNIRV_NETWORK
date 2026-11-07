package main

import "testing"

// TestStandbyAllowlistDeniesUnlistedPurpose enforces DEC-1 §2a.6: a mirrored Oracle
// may only sign replication-pull. Any other purpose (including the active-mode
// purposes and an unknown authority purpose) must be denied by the deny-by-default
// Policy, so a widened token cannot obtain an unlisted signature.
func TestStandbyAllowlistDeniesUnlistedPurpose(t *testing.T) {
	pol := PolicyForAllowlist(StandbyAllowlist(), "chain-x", "0", 10)
	if !messageAllowed(pol, DomainReplication, PurposeReplicationPull, "chain-x") {
		t.Fatal("standby must allow replication-pull")
	}
	denied := []struct{ domain, purpose string }{
		{DomainReplication, PurposeReplicationHead},
		{DomainAuthority, PurposeAuthorityRecord},
		{DomainAuthority, PurposeBondRelease},
		{DomainAuthority, PurposeBondHold},
		{DomainAuthority, "bootnode-enroll"},
		{"knirv.authority", PurposeReplicationPull},
		{"", PurposeReplicationPull},
	}
	for _, d := range denied {
		if messageAllowed(pol, d.domain, d.purpose, "chain-x") {
			t.Fatalf("standby must DENY (%s,%s)", d.domain, d.purpose)
		}
	}
}

// TestActiveAllowlistGrantsAuthorityPurposes enforces DEC-1 §2a.6: the active Oracle
// adds replication-head and the individually named Oracle authority purposes, while
// still denying anything not explicitly listed.
func TestActiveAllowlistGrantsAuthorityPurposes(t *testing.T) {
	pol := PolicyForAllowlist(ActiveAllowlist(), "chain-x", "0", 10)
	allowed := []struct{ domain, purpose string }{
		{DomainReplication, PurposeReplicationPull},
		{DomainReplication, PurposeReplicationHead},
		{DomainAuthority, PurposeAuthorityRecord},
		{DomainAuthority, PurposeBondRelease},
		{DomainAuthority, PurposeBondHold},
	}
	for _, a := range allowed {
		if !messageAllowed(pol, a.domain, a.purpose, "chain-x") {
			t.Fatalf("active must allow (%s,%s)", a.domain, a.purpose)
		}
	}
	if messageAllowed(pol, DomainAuthority, "bootnode-enroll", "chain-x") {
		t.Fatal("active must DENY an unlisted authority purpose")
	}
}
