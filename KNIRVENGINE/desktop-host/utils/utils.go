package utils

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// --- Helper functions ---
// These utility functions are designed for safely extracting typed values from maps
// They are currently unused but maintained for future use in configuration parsing

// GetString safely extracts a string value from a map
func GetString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

// GetInt safely extracts an integer value from a map
func GetInt(m map[string]interface{}, key string) int {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		}
	}
	return 0
}

// GetFloatPtr safely extracts a float64 pointer from a map
func GetFloatPtr(m map[string]interface{}, key string) *float64 {
	if val, ok := m[key]; ok {
		if f, ok := val.(float64); ok {
			return &f
		}
		if i, ok := val.(int); ok {
			f := float64(i)
			return &f
		}
	}
	return nil
}

// GetInt64Ptr safely extracts an int64 pointer from a map
func GetInt64Ptr(m map[string]interface{}, key string) *int64 {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case int:
			i64 := int64(v)
			return &i64
		case int64:
			return &v
		case float64:
			i64 := int64(v)
			return &i64
		}
	}
	return nil
}

// PortConfig holds port configuration
type PortConfig struct {
	APIPort int
	GUIPort int
}

// ReadPortConfig reads port configuration from ports.config file
func ReadPortConfig(configPath string) (*PortConfig, error) {
	// Default values
	config := &PortConfig{
		APIPort: 8081,
		GUIPort: 8080,
	}

	file, err := os.Open(configPath)
	if err != nil {
		// If file doesn't exist, return defaults
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, fmt.Errorf("failed to open ports config file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key=value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "API_PORT":
			if port, err := strconv.Atoi(value); err == nil {
				config.APIPort = port
			}
		case "GUI_PORT":
			if port, err := strconv.Atoi(value); err == nil {
				config.GUIPort = port
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading ports config file: %w", err)
	}

	return config, nil
}
