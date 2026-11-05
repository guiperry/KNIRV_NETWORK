package launcher

import (
	"github.com/spf13/viper"

	"knirv-server/internal/launcher/factualityenv"
)

// factualityEmbeddingEnv passes explicit operator embedding configuration to the
// mapper through the backend and MODELS process environments. The pure mapping
// lives in the dependency-free factualityenv package so its contract test runs
// without the launcher's GPU linker dependencies.
func factualityEmbeddingEnv(env []string, config *viper.Viper) []string {
	return factualityenv.EmbeddingEnv(env, config)
}
