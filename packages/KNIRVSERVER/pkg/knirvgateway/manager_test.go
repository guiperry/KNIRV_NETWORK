package knirvgateway

import "testing"

func TestWithEnvOverridesReplacesInheritedValues(t *testing.T) {
	env := []string{
		"CHAIN_NODE_ROLE=Client",
		"CLOUDFLARE_API_TOKEN=stale-token",
		"PATH=/usr/bin",
	}

	got := withEnvOverrides(env, map[string]string{
		"CHAIN_NODE_ROLE":      "Root",
		"CLOUDFLARE_API_TOKEN": "authoritative-token",
	})

	values := make(map[string][]string)
	for _, entry := range got {
		key, value, ok := splitEnvEntry(entry)
		if ok {
			values[key] = append(values[key], value)
		}
	}
	for key, want := range map[string]string{
		"CHAIN_NODE_ROLE":      "Root",
		"CLOUDFLARE_API_TOKEN": "authoritative-token",
		"PATH":                 "/usr/bin",
	} {
		if len(values[key]) != 1 || values[key][0] != want {
			t.Fatalf("%s = %v, want one value %q", key, values[key], want)
		}
	}
}

func splitEnvEntry(entry string) (string, string, bool) {
	for i := 0; i < len(entry); i++ {
		if entry[i] == '=' {
			return entry[:i], entry[i+1:], true
		}
	}
	return "", "", false
}
