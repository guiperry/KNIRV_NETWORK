package main

import (
	"os"
	"testing"
)

func TestApplyDirectFlags(t *testing.T) {
	for _, tc := range []struct {
		direct, jitter bool
		wantDirect     string
		wantJitter     string
	}{
		{false, false, "", ""},
		{false, true, "", ""},
		{true, false, "true", ""},
		{true, true, "true", "true"},
	} {
		t.Setenv("DIRECT_MODE", "")
		t.Setenv("DIRECT_JITTER", "")
		applyDirectFlags(tc.direct, tc.jitter)
		if got := os.Getenv("DIRECT_MODE"); got != tc.wantDirect {
			t.Fatalf("direct=%t jitter=%t: DIRECT_MODE=%q", tc.direct, tc.jitter, got)
		}
		if got := os.Getenv("DIRECT_JITTER"); got != tc.wantJitter {
			t.Fatalf("direct=%t jitter=%t: DIRECT_JITTER=%q", tc.direct, tc.jitter, got)
		}
	}
}
