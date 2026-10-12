package headless

import (
	"log"
	"os"
	"strings"

	"knirvmodels/internal/cli/controller"
	"knirvmodels/pkg/hashing/transformer"
	"knirvmodels/pkg/head"
	"knirvmodels/pkg/models"
)

// registerModels mounts the daemon's model surface under /models/*: the registry
// plus HASHER, HEART and HEAD. backend_server reaches it over this socket and
// authenticates callers; nothing here does.
//
// HEART is built on first use (it loads a tokenizer, semantic memory and an
// engine), so a daemon that never serves a generate request never pays for it.
func (s *Server) registerModels() {
	reg := models.NewRegistry()

	hasher := models.NewHasher(nil)
	reg.Register(hasher)
	reg.Register(models.NewHeart(func() (*transformer.HEARTService, error) {
		cfg := heartConfigFromEnv()
		// Share the engine the HASHER model serves; if it cannot be built HEART
		// builds its own, as it always has.
		if e, err := hasher.Engine(); err == nil {
			cfg.Engine = e
		} else {
			log.Printf("HEART: shared HASHER engine unavailable (%v); building its own", err)
		}
		return transformer.NewHEARTServiceWithConfig(cfg)
	}))

	if dir, err := controller.HEADDir(); err != nil {
		log.Printf("HEAD serving disabled: %v", err)
	} else {
		reg.Register(models.NewHead(head.NewService(dir)))
		log.Printf("HEAD served from %s (/models/head/*)", dir)
	}

	reg.Mount(s.mux)
	s.registry = reg
}

// heartConfigFromEnv is the daemon's equivalent of cmd/heart's flags.
//
//	KNIRV_HEART_EXTERNAL_GENERATOR_URL  bootstrap: an external LLM writes WASM until the internal model is trained
//	KNIRV_HEART_CHECKPOINT              opt in to the legacy Gorgonite GPT checkpoint
func heartConfigFromEnv() *transformer.HEARTConfig {
	cfg := transformer.DefaultHEARTConfig(false, false)
	if u := strings.TrimSpace(os.Getenv("KNIRV_HEART_EXTERNAL_GENERATOR_URL")); u != "" {
		cfg.ExternalGeneratorURL = u
		log.Printf("HEART: external generator %s (bootstrap mode)", u)
	}
	if p := strings.TrimSpace(os.Getenv("KNIRV_HEART_CHECKPOINT")); p != "" {
		cfg.ModelCheckpointPath = p
	}
	return cfg
}
