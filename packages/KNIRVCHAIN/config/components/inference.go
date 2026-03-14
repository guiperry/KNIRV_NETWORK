package components

import (
	"fmt"
	"net/url"
	"strings"
)

// InferenceConfig represents inference service component configuration
type InferenceConfig struct {
	Enabled         bool                   `json:"enabled" mapstructure:"enabled"`
	Provider        string                 `json:"provider" mapstructure:"provider"`
	APIKey          string                 `json:"api_key" mapstructure:"api_key"`
	BaseURL         string                 `json:"base_url" mapstructure:"base_url"`
	DefaultModel    string                 `json:"default_model" mapstructure:"default_model"`
	MaxTokens       int                    `json:"max_tokens" mapstructure:"max_tokens"`
	Temperature     float64                `json:"temperature" mapstructure:"temperature"`
	Timeout         int                    `json:"timeout" mapstructure:"timeout"`
	RetryAttempts   int                    `json:"retry_attempts" mapstructure:"retry_attempts"`
	RateLimitRPM    int                    `json:"rate_limit_rpm" mapstructure:"rate_limit_rpm"`
	Models          map[string]ModelConfig `json:"models" mapstructure:"models"`
	CerebrasConfig  CerebrasConfig         `json:"cerebras" mapstructure:"cerebras"`
	DeepseekConfig  DeepseekConfig         `json:"deepseek" mapstructure:"deepseek"`
	GeminiConfig    GeminiConfig           `json:"gemini" mapstructure:"gemini"`
}

// ModelConfig represents configuration for a specific model
type ModelConfig struct {
	Name        string  `json:"name" mapstructure:"name"`
	Provider    string  `json:"provider" mapstructure:"provider"`
	MaxTokens   int     `json:"max_tokens" mapstructure:"max_tokens"`
	Temperature float64 `json:"temperature" mapstructure:"temperature"`
	Enabled     bool    `json:"enabled" mapstructure:"enabled"`
}

// CerebrasConfig represents Cerebras-specific configuration
type CerebrasConfig struct {
	APIKey  string `json:"api_key" mapstructure:"api_key"`
	BaseURL string `json:"base_url" mapstructure:"base_url"`
	Model   string `json:"model" mapstructure:"model"`
}

// DeepseekConfig represents Deepseek-specific configuration
type DeepseekConfig struct {
	APIKey  string `json:"api_key" mapstructure:"api_key"`
	BaseURL string `json:"base_url" mapstructure:"base_url"`
	Model   string `json:"model" mapstructure:"model"`
}

// GeminiConfig represents Gemini-specific configuration
type GeminiConfig struct {
	APIKey    string `json:"api_key" mapstructure:"api_key"`
	ProjectID string `json:"project_id" mapstructure:"project_id"`
	Model     string `json:"model" mapstructure:"model"`
}

// GetConfigKey returns the configuration key prefix for inference
func (c *InferenceConfig) GetConfigKey() string {
	return "inference"
}

// Validate validates the inference configuration
func (c *InferenceConfig) Validate() error {
	if c.Enabled {
		// Validate provider
		validProviders := map[string]bool{
			"cerebras": true, "deepseek": true, "gemini": true, "openai": true, "anthropic": true,
		}
		if !validProviders[strings.ToLower(c.Provider)] {
			return fmt.Errorf("invalid inference provider: %s", c.Provider)
		}
		
		// Validate API key
		if c.APIKey == "" {
			return fmt.Errorf("inference API key cannot be empty when enabled")
		}
		
		// Validate base URL if provided
		if c.BaseURL != "" {
			if _, err := url.Parse(c.BaseURL); err != nil {
				return fmt.Errorf("invalid inference base URL: %s", c.BaseURL)
			}
		}
		
		// Validate default model
		if c.DefaultModel == "" {
			return fmt.Errorf("inference default model cannot be empty when enabled")
		}
		
		// Validate max tokens
		if c.MaxTokens <= 0 || c.MaxTokens > 100000 {
			return fmt.Errorf("invalid inference max_tokens: %d (must be 1-100000)", c.MaxTokens)
		}
		
		// Validate temperature
		if c.Temperature < 0.0 || c.Temperature > 2.0 {
			return fmt.Errorf("invalid inference temperature: %f (must be 0.0-2.0)", c.Temperature)
		}
		
		// Validate timeout
		if c.Timeout <= 0 {
			return fmt.Errorf("inference timeout must be positive, got: %d", c.Timeout)
		}
		
		// Validate retry attempts
		if c.RetryAttempts < 0 || c.RetryAttempts > 10 {
			return fmt.Errorf("invalid inference retry_attempts: %d (must be 0-10)", c.RetryAttempts)
		}
		
		// Validate rate limit
		if c.RateLimitRPM < 0 {
			return fmt.Errorf("inference rate_limit_rpm cannot be negative, got: %d", c.RateLimitRPM)
		}
		
		// Validate provider-specific configurations
		switch strings.ToLower(c.Provider) {
		case "cerebras":
			if err := c.validateCerebrasConfig(); err != nil {
				return fmt.Errorf("cerebras config validation failed: %w", err)
			}
		case "deepseek":
			if err := c.validateDeepseekConfig(); err != nil {
				return fmt.Errorf("deepseek config validation failed: %w", err)
			}
		case "gemini":
			if err := c.validateGeminiConfig(); err != nil {
				return fmt.Errorf("gemini config validation failed: %w", err)
			}
		}
		
		// Validate model configurations
		for modelName, modelConfig := range c.Models {
			if err := c.validateModelConfig(modelName, modelConfig); err != nil {
				return fmt.Errorf("model %s validation failed: %w", modelName, err)
			}
		}
	}
	
	return nil
}

// validateCerebrasConfig validates Cerebras-specific configuration
func (c *InferenceConfig) validateCerebrasConfig() error {
	if c.CerebrasConfig.APIKey == "" {
		return fmt.Errorf("cerebras API key cannot be empty")
	}
	if c.CerebrasConfig.BaseURL != "" {
		if _, err := url.Parse(c.CerebrasConfig.BaseURL); err != nil {
			return fmt.Errorf("invalid cerebras base URL: %s", c.CerebrasConfig.BaseURL)
		}
	}
	return nil
}

// validateDeepseekConfig validates Deepseek-specific configuration
func (c *InferenceConfig) validateDeepseekConfig() error {
	if c.DeepseekConfig.APIKey == "" {
		return fmt.Errorf("deepseek API key cannot be empty")
	}
	if c.DeepseekConfig.BaseURL != "" {
		if _, err := url.Parse(c.DeepseekConfig.BaseURL); err != nil {
			return fmt.Errorf("invalid deepseek base URL: %s", c.DeepseekConfig.BaseURL)
		}
	}
	return nil
}

// validateGeminiConfig validates Gemini-specific configuration
func (c *InferenceConfig) validateGeminiConfig() error {
	if c.GeminiConfig.APIKey == "" {
		return fmt.Errorf("gemini API key cannot be empty")
	}
	if c.GeminiConfig.ProjectID == "" {
		return fmt.Errorf("gemini project ID cannot be empty")
	}
	return nil
}

// validateModelConfig validates a specific model configuration
func (c *InferenceConfig) validateModelConfig(_ string, config ModelConfig) error {
	if config.Name == "" {
		return fmt.Errorf("model name cannot be empty")
	}
	if config.Provider == "" {
		return fmt.Errorf("model provider cannot be empty")
	}
	if config.MaxTokens <= 0 {
		return fmt.Errorf("model max_tokens must be positive")
	}
	if config.Temperature < 0.0 || config.Temperature > 2.0 {
		return fmt.Errorf("model temperature must be between 0.0 and 2.0")
	}
	return nil
}

// GetDefaults returns default configuration values for inference
func (c *InferenceConfig) GetDefaults() map[string]interface{} {
	return map[string]interface{}{
		"enabled":         false,
		"provider":        "cerebras",
		"api_key":         "",
		"base_url":        "",
		"default_model":   "llama3.1-8b",
		"max_tokens":      4096,
		"temperature":     0.7,
		"timeout":         30,
		"retry_attempts":  3,
		"rate_limit_rpm":  60,
		"models":          make(map[string]ModelConfig),
		"cerebras": map[string]interface{}{
			"api_key":  "",
			"base_url": "https://api.cerebras.ai/v1",
			"model":    "llama3.1-8b",
		},
		"deepseek": map[string]interface{}{
			"api_key":  "",
			"base_url": "https://api.deepseek.com/v1",
			"model":    "deepseek-chat",
		},
		"gemini": map[string]interface{}{
			"api_key":    "",
			"project_id": "",
			"model":      "gemini-pro",
		},
	}
}

// GetEnvironmentMappings returns environment variable mappings for inference
func (c *InferenceConfig) GetEnvironmentMappings() map[string]string {
	return map[string]string{
		"inference.enabled":                "KNIRV_INFERENCE_ENABLED",
		"inference.provider":               "KNIRV_INFERENCE_PROVIDER",
		"inference.api_key":                "KNIRV_INFERENCE_API_KEY",
		"inference.base_url":               "KNIRV_INFERENCE_BASE_URL",
		"inference.default_model":          "KNIRV_INFERENCE_DEFAULT_MODEL",
		"inference.max_tokens":             "KNIRV_INFERENCE_MAX_TOKENS",
		"inference.temperature":            "KNIRV_INFERENCE_TEMPERATURE",
		"inference.timeout":                "KNIRV_INFERENCE_TIMEOUT",
		"inference.retry_attempts":         "KNIRV_INFERENCE_RETRY_ATTEMPTS",
		"inference.rate_limit_rpm":         "KNIRV_INFERENCE_RATE_LIMIT_RPM",
		"inference.cerebras.api_key":       "KNIRV_CEREBRAS_API_KEY",
		"inference.cerebras.base_url":      "KNIRV_CEREBRAS_BASE_URL",
		"inference.cerebras.model":         "KNIRV_CEREBRAS_MODEL",
		"inference.deepseek.api_key":       "KNIRV_DEEPSEEK_API_KEY",
		"inference.deepseek.base_url":      "KNIRV_DEEPSEEK_BASE_URL",
		"inference.deepseek.model":         "KNIRV_DEEPSEEK_MODEL",
		"inference.gemini.api_key":         "KNIRV_GEMINI_API_KEY",
		"inference.gemini.project_id":      "KNIRV_GEMINI_PROJECT_ID",
		"inference.gemini.model":           "KNIRV_GEMINI_MODEL",
	}
}

// NewInferenceConfig creates a new inference configuration with defaults
func NewInferenceConfig() *InferenceConfig {
	return &InferenceConfig{
		Enabled:       false,
		Provider:      "cerebras",
		APIKey:        "",
		BaseURL:       "",
		DefaultModel:  "llama3.1-8b",
		MaxTokens:     4096,
		Temperature:   0.7,
		Timeout:       30,
		RetryAttempts: 3,
		RateLimitRPM:  60,
		Models:        make(map[string]ModelConfig),
		CerebrasConfig: CerebrasConfig{
			APIKey:  "",
			BaseURL: "https://api.cerebras.ai/v1",
			Model:   "llama3.1-8b",
		},
		DeepseekConfig: DeepseekConfig{
			APIKey:  "",
			BaseURL: "https://api.deepseek.com/v1",
			Model:   "deepseek-chat",
		},
		GeminiConfig: GeminiConfig{
			APIKey:    "",
			ProjectID: "",
			Model:     "gemini-pro",
		},
	}
}
