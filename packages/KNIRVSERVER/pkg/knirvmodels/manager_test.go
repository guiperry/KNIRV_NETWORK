package knirvmodels

import (
	"reflect"
	"testing"
)

func TestAppendDirectArgs(t *testing.T) {
	for _, tc := range []struct {
		direct, jitter bool
		want           []string
	}{
		{false, false, []string{"base"}},
		{false, true, []string{"base"}},
		{true, false, []string{"base", "--direct"}},
		{true, true, []string{"base", "--direct", "--direct-jitter"}},
	} {
		if got := appendDirectArgs([]string{"base"}, tc.direct, tc.jitter); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("direct=%t jitter=%t: got %v want %v", tc.direct, tc.jitter, got, tc.want)
		}
	}
}

func TestSetEnvOverrideReplacesInheritedValue(t *testing.T) {
	env := []string{"DEVICE_IP=stale.example", "OTHER=value", "DEVICE_IP=older.example"}
	got := setEnvOverride(env, "DEVICE_IP", "asic.example:8888")
	want := []string{"OTHER=value", "DEVICE_IP=asic.example:8888"}
	if len(got) != len(want) {
		t.Fatalf("setEnvOverride() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("setEnvOverride()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
