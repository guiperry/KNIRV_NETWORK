package headless

import (
	"testing"
	"time"
)

// headCheckInterval reads KNIRV_HEART_DECISION_CHECK_MINUTES (legacy KNIRV_HEAD_CHECK_MINUTES).
// Note: the headless package panics in NewServer when the KNIRVSDK working tree is used
// (frame-core digest mismatch). This test does not call NewServer, but it only runs in a
// tree where the package test binary starts, i.e. with the HEAD KNIRVSDK.
func TestHeadCheckIntervalEnvNames(t *testing.T) {
	cases := []struct {
		name   string
		newVal string
		legacy string
		want   time.Duration
	}{
		{name: "unset defaults to 30 minutes", want: 30 * time.Minute},
		{name: "new name is used", newVal: "5", want: 5 * time.Minute},
		{name: "legacy name is used when new is unset", legacy: "7", want: 7 * time.Minute},
		{name: "new name wins when both set", newVal: "5", legacy: "7", want: 5 * time.Minute},
		{name: "non-positive value falls back to default", newVal: "0", want: 30 * time.Minute},
		{name: "non-numeric value falls back to default", newVal: "soon", want: 30 * time.Minute},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear every variable the helper can read, then set only what the case needs.
			for _, k := range []string{
				"KNIRV_HEART_DECISION_CHECK_MINUTES",
				"KNIRV_HEAD_CHECK_MINUTES",
			} {
				t.Setenv(k, "")
			}
			if tc.newVal != "" {
				t.Setenv("KNIRV_HEART_DECISION_CHECK_MINUTES", tc.newVal)
			}
			if tc.legacy != "" {
				t.Setenv("KNIRV_HEAD_CHECK_MINUTES", tc.legacy)
			}
			if got := headCheckInterval(); got != tc.want {
				t.Fatalf("headCheckInterval() = %v, want %v", got, tc.want)
			}
		})
	}
}
