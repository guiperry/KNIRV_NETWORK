// Package config defines the persistent configuration for the llama service.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	DefaultModelName = "TinyLlama-1.1B-Chat-v1.0-Q4_0.gguf"
	DefaultModelURL  = "https://huggingface.co/TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF/resolve/main/tinyllama-1.1b-chat-v1.0.Q4_0.gguf"
)

// Config contains only paths and model metadata; it deliberately contains no
// secrets and is stored with user-only permissions.
type Config struct {
	ServerPath string `json:"server_path"`
	ModelPath  string `json:"model_path"`
	ModelName  string `json:"model_name"`
}

func Paths(dataDir string) (data, configFile string, err error) {
	if dataDir != "" {
		return dataDir, filepath.Join(dataDir, "config.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("find home directory: %w", err)
	}
	dataRoot := os.Getenv("XDG_DATA_HOME")
	if dataRoot == "" {
		dataRoot = filepath.Join(home, ".local", "share")
	}
	configRoot := os.Getenv("XDG_CONFIG_HOME")
	if configRoot == "" {
		configRoot = filepath.Join(home, ".config")
	}
	return filepath.Join(dataRoot, "llama"), filepath.Join(configRoot, "llama", "config.json"), nil
}

func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return Config{}, fmt.Errorf("parse configuration: %w", err)
	}
	return c, nil
}

func Save(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0600)
}
