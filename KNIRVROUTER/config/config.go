package config

import (
	"log"
	"os"
	"github.com/joho/godotenv"
)

// Configuration (Read from environment variables or .env file)
type Config struct {
	RPCEndpoint    string
	ServiceAddress string // The address that the backend will use
	Port           string
	TURNAddress    string
	TURNPort       string
}

var AppConfig Config

// LoadConfig loads the configuration from environment variables.
func LoadConfig() {
	// Load .env file if it exists (for local development)
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file: ", err) // Non-fatal; use env vars in production
	}

	AppConfig.RPCEndpoint = os.Getenv("RPC_ENDPOINT")
	if AppConfig.RPCEndpoint == "" {
		log.Fatal("RPC_ENDPOINT environment variable is required")
	}

	AppConfig.ServiceAddress = os.Getenv("SERVICE_ADDRESS")
	if AppConfig.ServiceAddress == "" {
		log.Fatal("SERVICE_ADDRESS environment variable is required")
	}

	AppConfig.Port = os.Getenv("PORT")
	if AppConfig.Port == "" {
		AppConfig.Port = "3001" // Default port
	}

	AppConfig.TURNAddress = os.Getenv("TURN_ADDRESS")
	if AppConfig.TURNAddress == "" {
		AppConfig.TURNAddress = "0.0.0.0" // Listen on all available interfaces
	}

	AppConfig.TURNPort = os.Getenv("TURN_PORT")
	if AppConfig.TURNPort == "" {
		AppConfig.TURNPort = "3478" // Default turn port
	}
}