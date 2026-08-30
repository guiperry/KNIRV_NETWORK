package knirvhasher

import "testing"

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
