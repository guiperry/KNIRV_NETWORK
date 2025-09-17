package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// ValidationResult represents the result of a configuration validation
type ValidationResult struct {
	Valid    bool                `json:"valid"`
	Errors   []ValidationError   `json:"errors,omitempty"`
	Warnings []ValidationWarning `json:"warnings,omitempty"`
	Summary  ValidationSummary   `json:"summary"`
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// ValidationWarning represents a configuration validation warning
type ValidationWarning struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// ValidationSummary provides a summary of validation results
type ValidationSummary struct {
	TotalFields    int `json:"total_fields"`
	ValidFields    int `json:"valid_fields"`
	ErrorCount     int `json:"error_count"`
	WarningCount   int `json:"warning_count"`
	ComponentCount int `json:"component_count"`
}

// ConfigurationValidator provides comprehensive configuration validation
type ConfigurationValidator struct {
	rules      map[string][]ValidationRule
	validators map[string]ConfigValidator
}

// ValidationRule represents a single validation rule
type ValidationRule struct {
	Name        string
	Description string
	Validator   func(value interface{}) error
	Required    bool
	Warning     bool // If true, failures generate warnings instead of errors
}

// NewConfigurationValidator creates a new configuration validator
func NewConfigurationValidator() *ConfigurationValidator {
	validator := &ConfigurationValidator{
		rules:      make(map[string][]ValidationRule),
		validators: make(map[string]ConfigValidator),
	}

	// Set up default validation rules
	validator.setupDefaultRules()

	return validator
}

// setupDefaultRules sets up default validation rules for common configuration fields
func (v *ConfigurationValidator) setupDefaultRules() {
	// Port validation rules
	portRule := ValidationRule{
		Name:        "port_range",
		Description: "Port must be in valid range (1-65535)",
		Validator: func(value interface{}) error {
			port, err := convertToInt(value)
			if err != nil {
				return fmt.Errorf("invalid port format: %v", value)
			}
			if port <= 0 || port > 65535 {
				return fmt.Errorf("port %d out of valid range (1-65535)", port)
			}
			return nil
		},
		Required: true,
	}

	// Apply port rule to all port fields
	portFields := []string{
		"port", "http_port", "p2p_port", "wallet_port", "alt_gui_port",
		"mcp.port", "economics.port", "network_monitor.port", "network_monitor.metrics_port",
	}
	for _, field := range portFields {
		v.AddRule(field, portRule)
	}

	// URL validation rule
	urlRule := ValidationRule{
		Name:        "valid_url",
		Description: "Must be a valid URL",
		Validator: func(value interface{}) error {
			if str, ok := value.(string); ok && str != "" {
				if _, err := url.Parse(str); err != nil {
					return fmt.Errorf("invalid URL format: %s", str)
				}
			}
			return nil
		},
		Required: false,
	}

	// Apply URL rule to URL fields
	urlFields := []string{
		"inference.base_url", "inference.cerebras.base_url", "inference.deepseek.base_url",
		"economics.xion_rpc", "economics.knirvchain_url", "economics.knirvnexus_url",
		"economics.knirvoracle_url", "economics.knirvgraph_url",
	}
	for _, field := range urlFields {
		v.AddRule(field, urlRule)
	}

	// Chain ID validation rule
	chainIDRule := ValidationRule{
		Name:        "chain_id_format",
		Description: "Chain ID must be non-empty and at least 3 characters",
		Validator: func(value interface{}) error {
			if str, ok := value.(string); ok {
				if str == "" {
					return fmt.Errorf("chain ID cannot be empty")
				}
				if len(str) < 3 {
					return fmt.Errorf("chain ID too short: %s (minimum 3 characters)", str)
				}
				// Check for valid characters (alphanumeric, hyphens, underscores)
				if matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, str); !matched {
					return fmt.Errorf("chain ID contains invalid characters: %s", str)
				}
			}
			return nil
		},
		Required: true,
	}
	v.AddRule("chain_id", chainIDRule)

	// API key validation rule
	apiKeyRule := ValidationRule{
		Name:        "api_key_format",
		Description: "API key should be at least 10 characters if provided",
		Validator: func(value interface{}) error {
			if str, ok := value.(string); ok && str != "" {
				if len(str) < 10 {
					return fmt.Errorf("API key too short: %d characters (minimum 10)", len(str))
				}
			}
			return nil
		},
		Required: false,
		Warning:  true, // API keys are optional, so generate warnings
	}

	// Apply API key rule to API key fields
	apiKeyFields := []string{
		"inference.api_key", "inference.cerebras.api_key", "inference.deepseek.api_key", "inference.gemini.api_key",
	}
	for _, field := range apiKeyFields {
		v.AddRule(field, apiKeyRule)
	}

	// File path validation rule
	pathRule := ValidationRule{
		Name:        "valid_path",
		Description: "Path must be valid and accessible",
		Validator: func(value interface{}) error {
			if str, ok := value.(string); ok && str != "" {
				// Check if path is absolute or relative
				if !filepath.IsAbs(str) && !strings.HasPrefix(str, "./") && !strings.HasPrefix(str, "../") {
					return fmt.Errorf("path should be absolute or explicitly relative: %s", str)
				}

				// Check if parent directory exists (for file paths)
				dir := filepath.Dir(str)
				if dir != "." && dir != "/" {
					if _, err := os.Stat(dir); os.IsNotExist(err) {
						return fmt.Errorf("parent directory does not exist: %s", dir)
					}
				}
			}
			return nil
		},
		Required: false,
		Warning:  true, // Path issues are warnings since they might be created later
	}

	// Apply path rule to path fields
	pathFields := []string{
		"blockchain_database_path", "searchable_database_path", "reflection_database_path",
		"economics.database_path", "mcp.cert_file", "mcp.key_file",
	}
	for _, field := range pathFields {
		v.AddRule(field, pathRule)
	}

	// Log level validation rule
	logLevelRule := ValidationRule{
		Name:        "valid_log_level",
		Description: "Log level must be one of: debug, info, warn, error, fatal",
		Validator: func(value interface{}) error {
			if str, ok := value.(string); ok && str != "" {
				validLevels := map[string]bool{
					"debug": true, "info": true, "warn": true, "error": true, "fatal": true,
				}
				if !validLevels[strings.ToLower(str)] {
					return fmt.Errorf("invalid log level: %s (must be debug, info, warn, error, or fatal)", str)
				}
			}
			return nil
		},
		Required: false,
	}

	// Apply log level rule to log level fields
	logLevelFields := []string{
		"log_level", "mcp.log_level", "economics.log_level", "network_monitor.log_level", "inference.log_level",
	}
	for _, field := range logLevelFields {
		v.AddRule(field, logLevelRule)
	}

	// Percentage validation rule
	percentageRule := ValidationRule{
		Name:        "valid_percentage",
		Description: "Percentage must be between 0 and 100",
		Validator: func(value interface{}) error {
			percentage, err := convertToFloat(value)
			if err != nil {
				return fmt.Errorf("invalid percentage format: %v", value)
			}
			if percentage < 0 || percentage > 100 {
				return fmt.Errorf("percentage %f out of valid range (0-100)", percentage)
			}
			return nil
		},
		Required: false,
	}

	// Apply percentage rule to percentage fields
	percentageFields := []string{
		"network_monitor.alert_thresholds.cpu_percent",
		"network_monitor.alert_thresholds.memory_percent",
		"network_monitor.alert_thresholds.disk_percent",
	}
	for _, field := range percentageFields {
		v.AddRule(field, percentageRule)
	}
}

// AddRule adds a validation rule for a specific field
func (v *ConfigurationValidator) AddRule(field string, rule ValidationRule) {
	if v.rules[field] == nil {
		v.rules[field] = make([]ValidationRule, 0)
	}
	v.rules[field] = append(v.rules[field], rule)
}

// AddValidator adds a custom validator for a specific field
func (v *ConfigurationValidator) AddValidator(field string, validator ConfigValidator) {
	v.validators[field] = validator
}

// ValidateConfiguration validates a complete configuration
func (v *ConfigurationValidator) ValidateConfiguration(config *Config) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
		Summary: ValidationSummary{
			ComponentCount: 1, // Main config counts as one component
		},
	}

	// Validate using reflection to walk through all fields
	v.validateStruct(reflect.ValueOf(config).Elem(), "", result)

	// Update summary
	result.Summary.ErrorCount = len(result.Errors)
	result.Summary.WarningCount = len(result.Warnings)
	result.Valid = result.Summary.ErrorCount == 0

	return result
}

// validateStruct validates a struct using reflection
func (v *ConfigurationValidator) validateStruct(value reflect.Value, prefix string, result *ValidationResult) {
	valueType := value.Type()

	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		fieldType := valueType.Field(i)

		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}

		// Build field name
		fieldName := fieldType.Name
		if prefix != "" {
			fieldName = prefix + "." + strings.ToLower(fieldName)
		} else {
			fieldName = strings.ToLower(fieldName)
		}

		// Handle different field types
		switch field.Kind() {
		case reflect.Struct:
			// Recursively validate nested structs
			v.validateStruct(field, fieldName, result)
		case reflect.Ptr:
			// Handle pointer fields
			if !field.IsNil() {
				v.validateStruct(field.Elem(), fieldName, result)
			}
		default:
			// Validate primitive fields
			v.validateField(fieldName, field.Interface(), result)
		}

		result.Summary.TotalFields++
	}
}

// validateField validates a single field
func (v *ConfigurationValidator) validateField(fieldName string, value interface{}, result *ValidationResult) {
	// Check custom validators first
	if validator, exists := v.validators[fieldName]; exists {
		if err := validator(fieldName, value); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Field:   fieldName,
				Value:   fmt.Sprintf("%v", value),
				Message: err.Error(),
				Code:    "CUSTOM_VALIDATION_FAILED",
			})
			return
		}
	}

	// Check validation rules
	if rules, exists := v.rules[fieldName]; exists {
		for _, rule := range rules {
			if err := rule.Validator(value); err != nil {
				if rule.Warning {
					result.Warnings = append(result.Warnings, ValidationWarning{
						Field:   fieldName,
						Value:   fmt.Sprintf("%v", value),
						Message: err.Error(),
						Code:    strings.ToUpper(rule.Name),
					})
				} else {
					result.Errors = append(result.Errors, ValidationError{
						Field:   fieldName,
						Value:   fmt.Sprintf("%v", value),
						Message: err.Error(),
						Code:    strings.ToUpper(rule.Name),
					})
					return
				}
			}
		}
	}

	result.Summary.ValidFields++
}

// ValidateEnvironmentVariables validates environment variables
func (v *ConfigurationValidator) ValidateEnvironmentVariables() *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
		Summary:  ValidationSummary{},
	}

	// Check for required environment variables
	requiredEnvVars := []string{
		"KNIRV_CHAIN_ID",
		"KNIRV_HTTP_PORT",
		"KNIRV_P2P_PORT",
	}

	for _, envVar := range requiredEnvVars {
		if value := os.Getenv(envVar); value == "" {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   envVar,
				Value:   "",
				Message: fmt.Sprintf("Required environment variable %s is not set", envVar),
				Code:    "MISSING_ENV_VAR",
			})
		}
		result.Summary.TotalFields++
	}

	// Check for deprecated environment variables
	deprecatedEnvVars := map[string]string{
		"agent_HTTPPORT":           "KNIRV_HTTP_PORT",
		"ECONOMICS_PORT":           "KNIRV_ECONOMICS_PORT",
		"DEFAULT_CEREBRAS_API_KEY": "KNIRV_CEREBRAS_API_KEY",
		"HTTP_API_PORT":            "KNIRV_NODEJS_PORT",
	}

	for oldVar, newVar := range deprecatedEnvVars {
		if value := os.Getenv(oldVar); value != "" {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   oldVar,
				Value:   value,
				Message: fmt.Sprintf("Deprecated environment variable %s should be migrated to %s", oldVar, newVar),
				Code:    "DEPRECATED_ENV_VAR",
			})
		}
	}

	result.Summary.WarningCount = len(result.Warnings)
	result.Summary.ErrorCount = len(result.Errors)
	result.Valid = result.Summary.ErrorCount == 0

	return result
}

// Helper functions

// convertToInt converts various types to int
func convertToInt(value interface{}) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("cannot convert %T to int", value)
	}
}

// convertToFloat converts various types to float64
func convertToFloat(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", value)
	}
}

// isValidHost checks if a host is valid (IP address or hostname)
func isValidHost(host string) bool {
	// Check if it's a valid IP address
	if net.ParseIP(host) != nil {
		return true
	}

	// Check if it's a valid hostname
	if matched, _ := regexp.MatchString(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`, host); matched {
		return true
	}

	return false
}

// PrintValidationResult prints a formatted validation result
func PrintValidationResult(result *ValidationResult) {
	fmt.Println("=== Configuration Validation Result ===")
	fmt.Printf("Overall Status: ")
	if result.Valid {
		fmt.Println("✅ VALID")
	} else {
		fmt.Println("❌ INVALID")
	}

	fmt.Printf("Summary: %d total fields, %d valid, %d errors, %d warnings\n",
		result.Summary.TotalFields, result.Summary.ValidFields,
		result.Summary.ErrorCount, result.Summary.WarningCount)

	if len(result.Errors) > 0 {
		fmt.Println("\n🚨 Errors:")
		for _, err := range result.Errors {
			fmt.Printf("  - %s: %s (value: %s)\n", err.Field, err.Message, err.Value)
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Println("\n⚠️  Warnings:")
		for _, warning := range result.Warnings {
			fmt.Printf("  - %s: %s (value: %s)\n", warning.Field, warning.Message, warning.Value)
		}
	}

	fmt.Println("=====================================")
}
