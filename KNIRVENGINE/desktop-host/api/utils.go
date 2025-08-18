package api

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// respondWithJSON sends a JSON response
func respondWithJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondWithError sends an error response
func respondWithError(w http.ResponseWriter, status int, message string) {
	respondWithJSON(w, status, map[string]string{"error": message})
}

// parseID parses a string ID into an int64
func parseID(id string) (int64, error) {
	return strconv.ParseInt(id, 10, 64)
}

// Shared utility functions for API handlers and connections

// getStringParam safely gets a string parameter from a map
func getStringParam(params map[string]interface{}, key string, defaultValue string) string {
	if val, ok := params[key].(string); ok {
		return val
	}
	return defaultValue
}

// getIntParam safely gets an int parameter from a map
func getIntParam(params map[string]interface{}, key string, defaultValue int) int {
	if val, ok := params[key].(float64); ok {
		return int(val)
	}
	if val, ok := params[key].(int); ok {
		return val
	}
	return defaultValue
}

// getBoolParam safely gets a bool parameter from a map
func getBoolParam(params map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := params[key].(bool); ok {
		return val
	}
	return defaultValue
}

// getStringFromConfig safely gets a string value from a config map
func getStringFromConfig(config map[string]interface{}, key, defaultValue string) string {
	if val, ok := config[key].(string); ok {
		return val
	}
	return defaultValue
}

// getStringFromMap safely gets a string value from a map (alias for consistency)
func getStringFromMap(m map[string]interface{}, key, defaultValue string) string {
	if value, ok := m[key].(string); ok {
		return value
	}
	return defaultValue
}

// getBoolFromMap safely gets a bool value from a map
func getBoolFromMap(m map[string]interface{}, key string, defaultValue bool) bool {
	if value, ok := m[key].(bool); ok {
		return value
	}
	return defaultValue
}

// getIntFromMap safely gets an int value from a map
func getIntFromMap(m map[string]interface{}, key string, defaultValue int) int {
	if value, ok := m[key].(float64); ok {
		return int(value)
	}
	if value, ok := m[key].(int); ok {
		return value
	}
	return defaultValue
}
