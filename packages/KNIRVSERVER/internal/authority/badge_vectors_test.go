package authority

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	sdk "github.com/guiperry/knirv-sdk-go/authority"
)

// The badge surfaces of Task 2d.1 share two vector files with BS and KNIRVCHAIN:
//
//	badge_actions.json  the action bodies and their canonical hashes (independent
//	                    Python computation); BS builds the body, this service hashes
//	                    it, KNIRVCHAIN re-derives it from the NFT.
//	badge_records.json  REAL records issued by this service for those bodies, with
//	                    the registry entry and the as_of time, so KNIRVCHAIN can
//	                    prove it re-verifies a record from the block alone.
func vectorPath(name string) string {
	return filepath.Join("..", "..", "..", "KNIRVSDK", "testvectors", "authority", name)
}

type badgeAction struct {
	Name                string         `json:"name"`
	Surface             string         `json:"surface"`
	Tier                uint8          `json:"tier"`
	Domain              string         `json:"domain"`
	Action              map[string]any `json:"action"`
	Canonical           string         `json:"canonical"`
	CanonicalActionHash string         `json:"canonical_action_hash"`
}

func loadBadgeActions(t *testing.T) []badgeAction {
	t.Helper()
	data, err := os.ReadFile(vectorPath("badge_actions.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Cases []badgeAction `json:"cases"`
	}
	if err := json.Unmarshal(data, &doc); err != nil || len(doc.Cases) == 0 {
		t.Fatalf("badge_actions.json: %v", err)
	}
	return doc.Cases
}

func TestBadgeActionVectorsMatchTheCanonicalProfile(t *testing.T) {
	for _, c := range loadBadgeActions(t) {
		t.Run(c.Name, func(t *testing.T) {
			raw, _ := json.Marshal(c.Action)
			canonical, err := CanonicalJSON(raw)
			if err != nil || string(canonical) != c.Canonical {
				t.Fatalf("canonical = %q, %v\nwant        %q", canonical, err, c.Canonical)
			}
			h, err := ActionHash(raw)
			if err != nil || hex.EncodeToString(h[:]) != c.CanonicalActionHash {
				t.Fatalf("hash = %x, %v; want %s", h, err, c.CanonicalActionHash)
			}
		})
	}
}

type badgeRecordFixture struct {
	AsOf          int64             `json:"as_of"`
	RegistryEntry *RegistryEntry    `json:"registry_entry"`
	Cases         []badgeRecordCase `json:"cases"`
	Note          string            `json:"note"`
}

type badgeRecordCase struct {
	Name   string         `json:"name"`
	Action map[string]any `json:"action"`
	Tier   uint8          `json:"tier"`
	Domain string         `json:"domain"`
	Hash   string         `json:"hash"`
	Record RecordJSON     `json:"record"`
}

// Regenerate with:
//
//	UPDATE_AUTHORITY_VECTORS=1 go test ./internal/authority -run TestBadgeRecordFixture
func TestBadgeRecordFixture(t *testing.T) {
	path := vectorPath("badge_records.json")
	if os.Getenv("UPDATE_AUTHORITY_VECTORS") == "1" {
		h := newSessionHarness(t, KindRoot, 1, nil)
		fx := badgeRecordFixture{AsOf: h.clock.Now().Unix(), Note: "Real records issued by KNIRVSERVER internal/authority (a root node, a session key certified by the long-term key) for the bodies in badge_actions.json. Regenerate with UPDATE_AUTHORITY_VECTORS=1. as_of is the service clock when they were issued."}
		for _, c := range loadBadgeActions(t) {
			raw, _ := json.Marshal(c.Action)
			res, err := h.svc.Authorize(context.Background(), Action{Tier: c.Tier, Domain: c.Domain, Body: raw})
			if err != nil || !res.Stored.Verified {
				t.Fatalf("%s: %v %+v", c.Name, err, res)
			}
			fx.Cases = append(fx.Cases, badgeRecordCase{Name: c.Name, Action: c.Action, Tier: c.Tier, Domain: c.Domain, Hash: res.Stored.Hash(), Record: res.Stored.Record})
		}
		fx.RegistryEntry = h.svc.RegistryEntry()
		out, _ := json.MarshalIndent(fx, "", "  ")
		if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (generate it with UPDATE_AUTHORITY_VECTORS=1)", err)
	}
	var fx badgeRecordFixture
	if err := json.Unmarshal(data, &fx); err != nil || fx.RegistryEntry == nil || len(fx.Cases) == 0 {
		t.Fatalf("badge_records.json: %v", err)
	}
	pub, _ := hex.DecodeString(fx.RegistryEntry.LongTermPub)
	reg := sdk.Registry{Operators: map[string]sdk.Operator{fx.RegistryEntry.Address: {
		Address: fx.RegistryEntry.Address, Kind: sdk.OperatorKind(fx.RegistryEntry.Kind), LongTermPub: pub, Scopes: fx.RegistryEntry.Scopes,
	}}}
	for _, c := range fx.Cases {
		t.Run(c.Name, func(t *testing.T) {
			rec, err := FromJSON(c.Record)
			if err != nil {
				t.Fatal(err)
			}
			if err := sdk.Verify(rec, reg, time.Unix(fx.AsOf, 0)); err != nil {
				t.Fatalf("the committed record no longer verifies: %v", err)
			}
			raw, _ := json.Marshal(c.Action)
			want, _ := ActionHash(raw)
			if c.Record.CanonicalActionHash != hex.EncodeToString(want[:]) || c.Record.Hash != c.Hash {
				t.Fatal("the record is not bound to its action, or the hash drifted")
			}
		})
	}
}
