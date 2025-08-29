package utils

import (
	"embed"
	"log"
	"os"

	"github.com/joho/godotenv"
)

//go:embed default.env
var defaultEnvFile embed.FS

// loadEmbeddedEnv loads the default environment variables from the embedded default.env file
// into the environment if they are not already set.
func LoadEmbeddedEnv() error {
	// Read the embedded default.env file
	envBytes, err := defaultEnvFile.ReadFile("default.env")
	if err != nil {
		return err
	}

	// Parse the environment variables from the file content
	envMap, err := godotenv.Unmarshal(string(envBytes))
	if err != nil {
		return err
	}

	// Set the environment variables if they are not already set
	for key, value := range envMap {
		if value != "" && key != "" {
			// Only set if not already defined in the environment
			if currentValue := os.Getenv(key); currentValue == "" {
				log.Printf("Setting default environment variable: %s", key)
				os.Setenv(key, value)
			}
		}
	}

	return nil
}
