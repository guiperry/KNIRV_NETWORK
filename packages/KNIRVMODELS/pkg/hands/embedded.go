package hands

import (
	"embed"
	"encoding/json"
)

//go:embed assets/*
var assets embed.FS

// DefaultConfig returns the embedded default HANDS architecture parameters. It is
// used to self-initialize the engine when no model file is present and no
// external weights are downloaded (per the migration constraints, no network
// fetch happens here).
func DefaultConfig() map[string]any {
	data, err := assets.ReadFile("assets/defaults.json")
	if err != nil {
		return map[string]any{}
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		return map[string]any{}
	}
	return cfg
}
