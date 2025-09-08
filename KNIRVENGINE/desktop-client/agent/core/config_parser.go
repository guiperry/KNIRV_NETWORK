package core

// ConfigParser provides utilities for parsing agent configurations
type ConfigParser struct{}

// NewConfigParser creates a new config parser
func NewConfigParser() *ConfigParser {
	return &ConfigParser{}
}

// Helper functions for safely extracting typed values from maps
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
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

func getFloatPtr(m map[string]interface{}, key string) *float64 {
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

func getInt64Ptr(m map[string]interface{}, key string) *int64 {
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

// ParseAgentConfig parses an agent configuration and returns a structured representation
func (p *ConfigParser) ParseAgentConfig(config map[string]interface{}) *AgentConfigData {
	if config == nil {
		return &AgentConfigData{}
	}

	return &AgentConfigData{
		Name:        getString(config, "name"),
		Type:        getString(config, "type"),
		Version:     getString(config, "version"),
		Description: getString(config, "description"),
		Author:      getString(config, "author"),
		License:     getString(config, "license"),
		MaxTokens:   getInt(config, "max_tokens"),
		Temperature: getFloatPtr(config, "temperature"),
		OwnerID:     getInt64Ptr(config, "owner_id"),
	}
}

// AgentConfigData represents structured agent configuration data
type AgentConfigData struct {
	Name        string
	Type        string
	Version     string
	Description string
	Author      string
	License     string
	MaxTokens   int
	Temperature *float64
	OwnerID     *int64
}

// ParseTerminalConfig parses terminal configuration
func (p *ConfigParser) ParseTerminalConfig(config map[string]interface{}) *TerminalConfig {
	if config == nil {
		return nil
	}

	return &TerminalConfig{
		DefaultRows:    getInt(config, "rows"),
		DefaultCols:    getInt(config, "cols"),
		FontSize:       getInt(config, "font_size"),
		FontFamily:     getString(config, "font_family"),
		Theme:          getString(config, "theme"),
		ScrollbackSize: getInt(config, "scrollback"),
		AutoOpen:       config["auto_open"] != nil && config["auto_open"] == true,
		CustomCSS:      getString(config, "custom_css"),
	}
}
