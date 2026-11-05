package factualityenv

import (
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

// setChildEnv replaces any existing value for key with value, or appends it.
func setChildEnv(env []string, key, value string) []string {
	prefix := key + "="
	filtered := env[:0]
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			continue
		}
		filtered = append(filtered, entry)
	}
	return append(filtered, prefix+value)
}

// EmbeddingEnv passes explicit operator embedding configuration to the mapper
// through the backend and MODELS process environments, then delegates to FlashEnv.
// It is kept in this dependency-free package so its contract test runs without the
// launcher's GPU linker dependencies.
func EmbeddingEnv(env []string, config *viper.Viper) []string {
	const allowKey = "factuality.embeddings.allow_cloudflare_for_private"
	if config.IsSet(allowKey) {
		env = setChildEnv(env, "KNIRV_EMBEDDINGS_ALLOW_CLOUDFLARE_FOR_PRIVATE", strconv.FormatBool(config.GetBool(allowKey)))
	}
	const classKey = "factuality.embeddings.data_class"
	if config.IsSet(classKey) {
		env = setChildEnv(env, "KNIRV_EMBEDDING_DATA_CLASS", strings.ToLower(strings.TrimSpace(config.GetString(classKey))))
	}
	return FlashEnv(env, config)
}

// FlashEnv passes the operator's flash (live lookup, Task 3.8) configuration to
// the MODELS daemon environment. Flash is ON by default (owner decision O5); the
// operator may turn it off or point it at a self-hosted metasearch endpoint through
// the SERVER yaml, and the launcher turns those keys into the KNIRV_FLASH /
// KNIRV_METASEARCH_* variables the daemon reads (see KNIRVMODELS/pkg/livesearch/production.go).
func FlashEnv(env []string, config *viper.Viper) []string {
	const enabledKey = "factuality.flash.enabled"
	if config.IsSet(enabledKey) {
		env = setChildEnv(env, "KNIRV_FLASH", strconv.FormatBool(config.GetBool(enabledKey)))
	}

	const urlKey = "factuality.flash.metasearch_url"
	if url := strings.TrimSpace(config.GetString(urlKey)); url != "" {
		env = setChildEnv(env, "KNIRV_METASEARCH_URL", url)
	}

	mapList := func(key, envVar string) {
		vals := config.GetStringSlice(key)
		out := make([]string, 0, len(vals))
		for _, v := range vals {
			if v = strings.TrimSpace(v); v != "" {
				out = append(out, v)
			}
		}
		if len(out) > 0 {
			env = setChildEnv(env, envVar, strings.Join(out, ","))
		}
	}
	mapList("factuality.flash.metasearch_engines", "KNIRV_METASEARCH_ENGINES")
	mapList("factuality.flash.allow_domains", "KNIRV_METASEARCH_ALLOW")
	mapList("factuality.flash.deny_domains", "KNIRV_METASEARCH_DENY")
	return env
}
