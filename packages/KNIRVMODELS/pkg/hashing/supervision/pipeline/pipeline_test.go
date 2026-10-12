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
