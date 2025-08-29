package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment   string
	Port          string
	DatabaseURL   string
	RedisURL      string
	JWTSecret     string
	LogLevel      string
	
	// Blockchain RPC URLs
	EthereumRPC   string
	BitcoinRPC    string
	SolanaRPC     string
	
	// Security
	EncryptionKey string
	HSMEndpoint   string
	
	// AI Agent Configuration
	MaxAgentsPerUser    int
	AgentMemoryLimit    int64
	AgentCPULimit       float64
	AgentNetworkTimeout int
	
	// External APIs
	CoinGeckoAPIKey     string
	AlchemyAPIKey       string
	InfuraProjectID     string
}

func Load() (*Config, error) {
	// Load .env file if it exists
	godotenv.Load()

	cfg := &Config{
		Environment:         getEnv("ENVIRONMENT", "development"),
		Port:               getEnv("PORT", "8080"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://localhost/crypto_wallet?sslmode=disable"),
		RedisURL:           getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:          getEnv("JWT_SECRET", "your-secret-key"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		EthereumRPC:        getEnv("ETHEREUM_RPC", "https://mainnet.infura.io/v3/YOUR_PROJECT_ID"),
		BitcoinRPC:         getEnv("BITCOIN_RPC", "https://bitcoin-rpc.example.com"),
		SolanaRPC:          getEnv("SOLANA_RPC", "https://api.mainnet-beta.solana.com"),
		EncryptionKey:      getEnv("ENCRYPTION_KEY", "32-byte-encryption-key-here!!!"),
		HSMEndpoint:        getEnv("HSM_ENDPOINT", ""),
		MaxAgentsPerUser:   getEnvInt("MAX_AGENTS_PER_USER", 10),
		AgentMemoryLimit:   getEnvInt64("AGENT_MEMORY_LIMIT", 256*1024*1024), // 256MB
		AgentCPULimit:      getEnvFloat("AGENT_CPU_LIMIT", 0.5),
		AgentNetworkTimeout: getEnvInt("AGENT_NETWORK_TIMEOUT", 30),
		CoinGeckoAPIKey:    getEnv("COINGECKO_API_KEY", ""),
		AlchemyAPIKey:      getEnv("ALCHEMY_API_KEY", ""),
		InfuraProjectID:    getEnv("INFURA_PROJECT_ID", ""),
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}