package controller

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

// resetLegacyEnvLog clears the once-per-name deprecation state so each test sees
// its own first use.
func resetLegacyEnvLog() {
	legacyEnvLogMu.Lock()
	legacyEnvLogged = map[string]bool{}
	legacyEnvLogMu.Unlock()
}

// captureLog routes the standard logger into a buffer for the duration of the test.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prevOut, prevFlags := log.Writer(), log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
	})
	return &buf
}

func TestHeartDecisionEnvNewNameWorksWithoutDeprecation(t *testing.T) {
	resetLegacyEnvLog()
	buf := captureLog(t)
	t.Setenv("KNIRV_HEART_DECISION_PRESET", "small")
	t.Setenv("KNIRV_HEAD_PRESET", "")

	if got := HeartDecisionEnv("PRESET"); got != "small" {
		t.Fatalf("new name: got %q, want %q", got, "small")
	}
	if buf.Len() != 0 {
		t.Fatalf("new name must not log a deprecation, got %q", buf.String())
	}
}

func TestHeartDecisionEnvLegacyNameWorksAndLogsOnce(t *testing.T) {
	resetLegacyEnvLog()
	buf := captureLog(t)
	t.Setenv("KNIRV_HEART_DECISION_EPOCHS", "")
	t.Setenv("KNIRV_HEAD_EPOCHS", "7")

	for i := 0; i < 5; i++ {
		if got := HeartDecisionEnv("EPOCHS"); got != "7" {
			t.Fatalf("legacy name call %d: got %q, want %q", i, got, "7")
		}
	}
	lines := strings.Count(buf.String(), "DEPRECATED")
	if lines != 1 {
		t.Fatalf("legacy name must log exactly once across 5 calls, logged %d times: %q", lines, buf.String())
	}
	if !strings.Contains(buf.String(), "KNIRV_HEAD_EPOCHS") || !strings.Contains(buf.String(), "KNIRV_HEART_DECISION_EPOCHS") {
		t.Fatalf("deprecation line must name both old and new vars: %q", buf.String())
	}
}

func TestHeartDecisionEnvNewWinsWhenBothSet(t *testing.T) {
	resetLegacyEnvLog()
	buf := captureLog(t)
	t.Setenv("KNIRV_HEART_DECISION_MIN_NEW", "11")
	t.Setenv("KNIRV_HEAD_MIN_NEW", "99")

	if got := HeartDecisionEnv("MIN_NEW"); got != "11" {
		t.Fatalf("both set: got %q, want new value %q", got, "11")
	}
	if buf.Len() != 0 {
		t.Fatalf("when the new name wins the legacy name is not read, so no log; got %q", buf.String())
	}
}

func TestHeartDecisionEnvUnsetReturnsEmpty(t *testing.T) {
	resetLegacyEnvLog()
	buf := captureLog(t)
	t.Setenv("KNIRV_HEART_DECISION_POLICY_HASH", "")
	t.Setenv("KNIRV_HEAD_POLICY_HASH", "")

	if got := HeartDecisionEnv("POLICY_HASH"); got != "" {
		t.Fatalf("unset: got %q, want empty so callers keep their defaults", got)
	}
	if buf.Len() != 0 {
		t.Fatalf("unset must not log; got %q", buf.String())
	}
}

func TestHeartDecisionEnvLegacyEmptyTreatedAsUnset(t *testing.T) {
	resetLegacyEnvLog()
	buf := captureLog(t)
	t.Setenv("KNIRV_HEART_DECISION_BRONZE_DIR", "")
	t.Setenv("KNIRV_HEAD_BRONZE_DIR", "")

	if got := HeartDecisionEnv("BRONZE_DIR"); got != "" {
		t.Fatalf("empty legacy value: got %q, want empty", got)
	}
	if buf.Len() != 0 {
		t.Fatalf("empty legacy value must not log; got %q", buf.String())
	}
}
