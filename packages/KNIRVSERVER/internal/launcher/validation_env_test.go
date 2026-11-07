package launcher

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// childEnvValue returns the value for key in an env slice, or "" if absent.
func childEnvValue(env []string, key string) string {
	prefix := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return strings.TrimPrefix(e, prefix)
		}
	}
	return ""
}

// TestValidationServiceChildEnv asserts the launcher propagates the validation
// service endpoint + token-file handoff to the graph child exactly as the DRQ
// client expects: a unix socket is derived from the backend socket, a TCP override
// is dropped (fail-closed), and an explicit unix override is honored.
func TestValidationServiceChildEnv(t *testing.T) {
	cfg := &Config{BackendSocket: "/run/knirv/backend.sock"}

	t.Setenv("KNIRV_VALIDATION_SERVICE_URL", "")
	t.Setenv("KNIRV_VALIDATION_SERVICE_TOKEN_FILE", "")
	env := appendValidationServiceChildEnv(cfg, []string{})
	require.Equal(t, "http+unix:///run/knirv/backend.sock",
		childEnvValue(env, "KNIRV_VALIDATION_SERVICE_URL"),
		"child must receive the backend unix socket as the validation endpoint")
	require.Contains(t, childEnvValue(env, "KNIRV_VALIDATION_SERVICE_TOKEN_FILE"),
		"validation_service_token",
		"child must receive the token-file handoff path")

	// TCP override must be ignored so the surface can never be networked.
	t.Setenv("KNIRV_VALIDATION_SERVICE_URL", "http://127.0.0.1:9999")
	envTCP := appendValidationServiceChildEnv(cfg, []string{})
	require.Empty(t, childEnvValue(envTCP, "KNIRV_VALIDATION_SERVICE_URL"),
		"a TCP override must NOT propagate to the child")

	// Explicit unix override is honored.
	t.Setenv("KNIRV_VALIDATION_SERVICE_URL", "http+unix:///custom/validation.sock")
	envOverride := appendValidationServiceChildEnv(cfg, []string{})
	require.Equal(t, "http+unix:///custom/validation.sock",
		childEnvValue(envOverride, "KNIRV_VALIDATION_SERVICE_URL"),
		"an explicit http+unix override must propagate to the child")
}
