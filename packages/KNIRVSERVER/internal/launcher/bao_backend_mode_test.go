package launcher

import "testing"

// appWithOpenBao returns a ServerApp whose config has secrets.openbao.enabled set as given.
func appWithOpenBao(enabled bool) *ServerApp {
	cfg := &Config{}
	cfg.Secrets.OpenBao.Enabled = enabled
	return &ServerApp{config: cfg}
}

// TestSecretsBackendModeDefaultsAndValidates: unset means auto; unknown values fail
// loudly rather than silently downgrading.
func TestSecretsBackendModeDefaultsAndValidates(t *testing.T) {
	t.Setenv(secretsBackendEnv, "")
	if m, err := secretsBackendMode(); err != nil || m != "auto" {
		t.Fatalf("unset: mode=%q err=%v, want auto", m, err)
	}
	for _, v := range []string{"auto", "file", "bao", " BAO "} {
		t.Setenv(secretsBackendEnv, v)
		if _, err := secretsBackendMode(); err != nil {
			t.Fatalf("%q: unexpected error %v", v, err)
		}
	}
	t.Setenv(secretsBackendEnv, "vault")
	if _, err := secretsBackendMode(); err == nil {
		t.Fatal("unknown backend must be an error, not a silent downgrade")
	}
}

// TestOpenBaoEnabledFollowsBackendMode: file never runs OpenBao, bao always does,
// auto keeps the existing config-driven default.
func TestOpenBaoEnabledFollowsBackendMode(t *testing.T) {
	cases := []struct {
		mode  string
		cfgOn bool
		want  bool
	}{
		{"file", true, false},
		{"file", false, false},
		{"bao", false, true},
		{"bao", true, true},
		{"auto", false, false},
		{"auto", true, true},
	}
	for _, c := range cases {
		t.Setenv(secretsBackendEnv, c.mode)
		if got := appWithOpenBao(c.cfgOn).openBaoEnabled(); got != c.want {
			t.Errorf("mode=%s cfg=%v: enabled=%v, want %v", c.mode, c.cfgOn, got, c.want)
		}
	}
}

// TestOpenBaoEnabledUnknownModeIsOff: an invalid mode never starts OpenBao on its own.
func TestOpenBaoEnabledUnknownModeIsOff(t *testing.T) {
	t.Setenv(secretsBackendEnv, "bogus")
	if appWithOpenBao(true).openBaoEnabled() {
		t.Fatal("invalid KNIRV_SECRETS_BACKEND must not start OpenBao")
	}
}
