package main

import "testing"

func TestSetChildEnvPropagatesResolvedRootKeyToBackend(t *testing.T) {
	const rootKeyPath = "/secure/custom/root.key"
	env := []string{
		"ORACLE_KEY_PATH=/stale/root.key",
		"KNIRV_ROOT_KEY_PATH=/another/stale/root.key",
	}

	env = setChildEnv(env, "ORACLE_KEY_PATH", rootKeyPath)
	env = setChildEnv(env, "KNIRV_ROOT_KEY_PATH", rootKeyPath)

	want := []string{
		"ORACLE_KEY_PATH=" + rootKeyPath,
		"KNIRV_ROOT_KEY_PATH=" + rootKeyPath,
	}
	if len(env) != len(want) {
		t.Fatalf("root key environment = %#v, want %#v", env, want)
	}
	for i := range want {
		if env[i] != want[i] {
			t.Fatalf("root key environment[%d] = %q, want %q", i, env[i], want[i])
		}
	}
}
