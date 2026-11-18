package pipeline

import (
	"strings"
	"testing"
)

func TestNormalizeBronzeRedactsEveryNestedTextField(t *testing.T) {
	raw := []byte(`{"event_id":"e","session_id":"s","instance_id":"i","sequence":1,"event_kind":"gate","source_scope":"cli","tenant_scope":"local","workspace":{"fingerprint":"f"},"task":{"request":"review password=supersecret","intent":"review","risk":"low"},"state_before":{"phase":"develop","budget":{}},"policy_bundle_hash":"p","trace":{"agent":"a","proposal":"ok"},"decision":{"recommended_action":"request_review","next_phase":"develop_review"},"outcome":{},"license":"operator-generated","collected_at":"2026-01-01T00:00:00Z"}`)
	ev, ok := NormalizeBronze(raw, DefaultSilverConfig())
	if !ok || strings.Contains(ev.Task.Request, "supersecret") {
		t.Fatalf("redaction failed: %#v", ev)
	}
}

// TestRedactTextRemovesSecretShapedStrings checks the exported redactor against
// obvious fake secret patterns (not real credentials) and that its output no
// longer matches the secret patterns.
func TestRedactTextRemovesSecretShapedStrings(t *testing.T) {
	cases := []string{
		"use password=fakefakefake123 to log in",
		"api_key: fakefakefakefake",
		"AKIA0000000000000000 is the key id",
	}
	for _, in := range cases {
		out := RedactText(in)
		if !strings.Contains(out, "[REDACTED]") {
			t.Fatalf("RedactText(%q) = %q: no redaction applied", in, out)
		}
		if containsSecret(out) {
			t.Fatalf("RedactText(%q) = %q: output still matches a secret pattern", in, out)
		}
	}
	plain := "ordinary sentence with no credentials"
	if got := RedactText(plain); got != plain {
		t.Fatalf("RedactText changed plain text: %q", got)
	}
}

// TestContainsSecretReportsSecretShapedStrings checks the exported fail-closed
// probe against a fake secret and plain text.
func TestContainsSecretReportsSecretShapedStrings(t *testing.T) {
	if !ContainsSecret("api_key: fakefakefakefake") {
		t.Fatal("ContainsSecret missed a secret-shaped string")
	}
	if ContainsSecret("ordinary sentence with no credentials") {
		t.Fatal("ContainsSecret flagged plain text")
	}
}
