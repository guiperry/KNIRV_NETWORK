package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ValidationMiddleware provides request validation functionality
type ValidationMiddleware struct {
	validator *validator.Validate
}

// NewValidationMiddleware creates a new validation middleware
func NewValidationMiddleware() *ValidationMiddleware {
	v := validator.New()

	// Register custom validation functions
	v.RegisterValidation("agent_type", validateAgentType)
	v.RegisterValidation("build_target", validateBuildTarget)
	v.RegisterValidation("status", validateStatus)
	v.RegisterValidation("capability", validateCapability)

	return &ValidationMiddleware{
		validator: v,
	}
}

// ValidateStruct validates a struct using the validator
func (vm *ValidationMiddleware) ValidateStruct(s interface{}) error {
	return vm.validator.Struct(s)
}

// ValidationError represents a validation error with field details
type ValidationError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

// ValidationResponse represents the response for validation errors
type ValidationResponse struct {
	Success bool              `json:"success"`
	Error   string            `json:"error"`
	Code    string            `json:"code"`
	Details []ValidationError `json:"details,omitempty"`
}

// ValidateRequestBody validates a request body against a struct type
func (vm *ValidationMiddleware) ValidateRequestBody(r *http.Request, target interface{}) error {
	// Decode JSON body
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() // Strict validation

	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid JSON format: %v", err)
	}

	// Validate struct
	if err := vm.validator.Struct(target); err != nil {
		return err
	}

	return nil
}

// CreateValidationMiddleware creates a middleware function for HTTP handlers
func (vm *ValidationMiddleware) CreateValidationMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip validation for GET requests and certain paths
			if r.Method == "GET" || vm.shouldSkipValidation(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Validate Content-Type for POST/PUT/PATCH requests
			if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
				contentType := r.Header.Get("Content-Type")
				if !strings.Contains(contentType, "application/json") {
					vm.writeValidationError(w, "Content-Type must be application/json", "INVALID_CONTENT_TYPE", nil)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// shouldSkipValidation determines if validation should be skipped for a path
func (vm *ValidationMiddleware) shouldSkipValidation(path string) bool {
	skipPaths := []string{
		"/api/v1/health",
		"/api/v1/status",
		"/api/v1/debug",
		"/api/v1/metrics",
		"/ws/",
	}

	for _, skipPath := range skipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}

	return false
}

// writeValidationError writes a validation error response
func (vm *ValidationMiddleware) writeValidationError(w http.ResponseWriter, message, code string, details []ValidationError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	response := ValidationResponse{
		Success: false,
		Error:   message,
		Code:    code,
		Details: details,
	}

	json.NewEncoder(w).Encode(response)
}

// FormatValidationErrors formats validator errors into a readable format
func (vm *ValidationMiddleware) FormatValidationErrors(err error) []ValidationError {
	var validationErrors []ValidationError

	if validatorErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validatorErrors {
			validationError := ValidationError{
				Field:   fieldError.Field(),
				Tag:     fieldError.Tag(),
				Value:   fmt.Sprintf("%v", fieldError.Value()),
				Message: vm.getErrorMessage(fieldError),
			}
			validationErrors = append(validationErrors, validationError)
		}
	}

	return validationErrors
}

// getErrorMessage returns a human-readable error message for a field error
func (vm *ValidationMiddleware) getErrorMessage(fieldError validator.FieldError) string {
	field := fieldError.Field()
	tag := fieldError.Tag()
	param := fieldError.Param()

	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", field, param)
	case "max":
		return fmt.Sprintf("%s must be at most %s characters long", field, param)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)
	case "uuid":
		return fmt.Sprintf("%s must be a valid UUID", field)
	case "agent_type":
		return fmt.Sprintf("%s must be a valid agent type (llm, tool, workflow, monitor)", field)
	case "build_target":
		return fmt.Sprintf("%s must be a valid build target (plugin, wasm, template)", field)
	case "status":
		return fmt.Sprintf("%s must be a valid status (idle, running, stopped, error)", field)
	case "capability":
		return fmt.Sprintf("%s must be a valid capability", field)
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, param)
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", field, param)
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", field, param)
	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}

// Custom validation functions

// validateAgentType validates agent type values
func validateAgentType(fl validator.FieldLevel) bool {
	agentType := fl.Field().String()
	validTypes := []string{"llm", "tool", "workflow", "monitor", "standard"}

	for _, validType := range validTypes {
		if agentType == validType {
			return true
		}
	}

	return false
}

// validateBuildTarget validates build target values
func validateBuildTarget(fl validator.FieldLevel) bool {
	buildTarget := fl.Field().String()
	validTargets := []string{"plugin", "wasm", "template"}

	for _, validTarget := range validTargets {
		if buildTarget == validTarget {
			return true
		}
	}

	return false
}

// validateStatus validates status values
func validateStatus(fl validator.FieldLevel) bool {
	status := fl.Field().String()
	validStatuses := []string{"idle", "running", "stopped", "error", "building", "deployed"}

	for _, validStatus := range validStatuses {
		if status == validStatus {
			return true
		}
	}

	return false
}

// validateCapability validates capability values
func validateCapability(fl validator.FieldLevel) bool {
	capability := fl.Field().String()

	// Basic capability validation - non-empty and reasonable length
	if len(capability) == 0 || len(capability) > 100 {
		return false
	}

	// Check for valid capability format (alphanumeric, spaces, hyphens, underscores)
	for _, char := range capability {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == ' ' || char == '-' || char == '_') {
			return false
		}
	}

	return true
}

// ValidateAndRespond validates a request body and writes error response if validation fails
func (vm *ValidationMiddleware) ValidateAndRespond(w http.ResponseWriter, r *http.Request, target interface{}) bool {
	if err := vm.ValidateRequestBody(r, target); err != nil {
		if validatorErrors, ok := err.(validator.ValidationErrors); ok {
			details := vm.FormatValidationErrors(validatorErrors)
			vm.writeValidationError(w, "Validation failed", "VALIDATION_ERROR", details)
		} else {
			vm.writeValidationError(w, err.Error(), "INVALID_REQUEST", nil)
		}
		return false
	}

	return true
}

// GetValidator returns the underlying validator instance
func (vm *ValidationMiddleware) GetValidator() *validator.Validate {
	return vm.validator
}

// RegisterCustomValidation registers a custom validation function
func (vm *ValidationMiddleware) RegisterCustomValidation(tag string, fn validator.Func) error {
	return vm.validator.RegisterValidation(tag, fn)
}

// ValidateValue validates a single value against validation tags
func (vm *ValidationMiddleware) ValidateValue(value interface{}, tag string) error {
	return vm.validator.Var(value, tag)
}
