package dveevidence

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func testBundle(t *testing.T, signer *Signer) (*Bundle, *Evidence) {
	t.Helper()
	events := []Event{
		{Timestamp: "2026-07-07T00:00:00Z", Type: "session.start"},
		{Timestamp: "2026-07-07T00:00:01Z", Type: "command.requested", Payload: []byte(`{"cmd":"echo hi"}`)},
	}
	chained, root, err := BuildEventLog(events)
	if err != nil {
		t.Fatalf("BuildEventLog failed: %v", err)
	}
	ev := &Evidence{
		Events:         chained,
		ArtifactHashes: []string{"sha256:aaa", "sha256:bbb"},
	}
	b := &Bundle{
		SchemaVersion:      SchemaVersion,
		SessionID:          "session-1",
		DVEID:              "dve-1",
		UserID:             "user-1",
		ProjectID:          "project-1",
		StartedAt:          "2026-07-07T00:00:00Z",
		CompletedAt:        "2026-07-07T00:00:02Z",
		WorkspaceBaseHash:  "sha256:base",
		WorkspaceFinalHash: "sha256:final",
		PolicyHash:         "sha256:policy",
		ToolRuns:           []ToolRun{{Tool: "codex", Version: "1.0.0"}},
		PermissionDecisions: []PermissionDecision{{
			ID:           "dec-1",
			Timestamp:    "2026-07-07T00:00:01Z",
			EventType:    "command.requested",
			Action:       "approved",
			Input:        "echo hi",
			PolicyHash:   "sha256:policy",
			DecisionHash: "sha256:decision",
			Denied:       false,
		}},
		Artifacts:          []ArtifactRef{{Name: "patch", Class: "diff", Hash: "sha256:aaa"}},
		MemvidRefs:         []MemvidRef{{MediaRef: "memvid.m4v", IndexRef: "index.json"}},
		EventLogRoot:       root,
		ArtifactMerkleRoot: MerkleRoot(ev.ArtifactHashes),
	}
	if err := signer.Sign(b); err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	return b, ev
}

func TestVerifyBundleSignedAndConsistent(t *testing.T) {
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	signer, err := SignerFromSeed("key-1", seed)
	if err != nil {
		t.Fatalf("SignerFromSeed failed: %v", err)
	}
	b, ev := testBundle(t, signer)
	report, err := VerifyBundle(b, VerifyOptions{
		Resolver: ResolverFromPublicKeys(map[string]ed25519.PublicKey{"key-1": signer.Public()}),
		Evidence: ev,
		Policy: &Policy{
			AllowedCommands: []string{"echo"},
		},
	})
	if err != nil {
		t.Fatalf("VerifyBundle failed: %v", err)
	}
	if report.Status != StatusVerified && report.Status != StatusVerifiedWithWarnings {
		t.Fatalf("unexpected status %s", report.Status)
	}
	if report.CertificateHash == "" {
		t.Fatal("expected certificate hash")
	}
}

func TestVerifyBundleRejectsTamperedMerkle(t *testing.T) {
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	signer, _ := SignerFromSeed("key-1", seed)
	b, ev := testBundle(t, signer)
	ev.ArtifactHashes[0] = "sha256:zzz"
	report, err := VerifyBundle(b, VerifyOptions{
		Resolver: ResolverFromPublicKeys(map[string]ed25519.PublicKey{"key-1": signer.Public()}),
		Evidence: ev,
	})
	if err != nil {
		t.Fatalf("VerifyBundle failed: %v", err)
	}
	if report.Status != StatusRejected {
		t.Fatalf("expected rejected status, got %s", report.Status)
	}
}

func TestIngestRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	signer, _ := SignerFromSeed("key-1", seed)
	b, ev := testBundle(t, signer)
	store := NewFileStore(t.TempDir())
	svc := NewIngestService(store, ResolverFromPublicKeys(map[string]ed25519.PublicKey{"key-1": signer.Public()}), &Policy{AllowedCommands: []string{"echo"}})
	router := gin.New()
	RegisterDVEIngestRoutes(router, svc)

	body := fmt.Sprintf(`{"bundle":%s,"evidence":%s}`, mustJSON(t, b), mustJSON(t, ev))
	req := httptest.NewRequest(http.MethodPost, "/api/dve/dve-1/sessions/ingest", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected ingest code %d: %s", resp.Code, resp.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/dve/dve-1/sessions/session-1/report", nil)
	getResp := httptest.NewRecorder()
	router.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("unexpected report code %d: %s", getResp.Code, getResp.Body.String())
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return string(raw)
}
