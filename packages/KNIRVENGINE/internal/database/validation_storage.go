package database

import (
	"context"
	"fmt"

	"github.com/go-playground/validator/v10"
)

// ValidatedStorage provides validation before database operations
type ValidatedStorage struct {
	validator *validator.Validate
}

// NewValidatedStorage creates a new validated storage instance
func NewValidatedStorage() *ValidatedStorage {
	v := validator.New()
	
	// Register custom validation functions
	v.RegisterValidation("agent_type", validateAgentType)
	v.RegisterValidation("build_target", validateBuildTarget)
	v.RegisterValidation("status", validateStatus)
	v.RegisterValidation("capability", validateCapability)
	
	return &ValidatedStorage{
		validator: v,
	}
}

// ValidateBeforeCreate validates data before creation
func (vs *ValidatedStorage) ValidateBeforeCreate(ctx context.Context, data interface{}) error {
	if err := vs.validator.Struct(data); err != nil {
		return fmt.Errorf("validation failed before create: %w", err)
	}
	return nil
}

// ValidateBeforeUpdate validates data before update
func (vs *ValidatedStorage) ValidateBeforeUpdate(ctx context.Context, data interface{}) error {
	if err := vs.validator.Struct(data); err != nil {
		return fmt.Errorf("validation failed before update: %w", err)
	}
	return nil
}

// ValidateBeforeDelete validates data before deletion (if needed)
func (vs *ValidatedStorage) ValidateBeforeDelete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("validation failed: ID cannot be empty")
	}
	
	// Validate UUID format
	if err := vs.validator.Var(id, "uuid"); err != nil {
		return fmt.Errorf("validation failed: invalid ID format: %w", err)
	}
	
	return nil
}

// ValidateAgentData validates agent-specific data
func (vs *ValidatedStorage) ValidateAgentData(ctx context.Context, agent interface{}) error {
	return vs.ValidateBeforeCreate(ctx, agent)
}

// ValidateWorkflowData validates workflow-specific data
func (vs *ValidatedStorage) ValidateWorkflowData(ctx context.Context, workflow interface{}) error {
	return vs.ValidateBeforeCreate(ctx, workflow)
}

// ValidateUserData validates user-specific data
func (vs *ValidatedStorage) ValidateUserData(ctx context.Context, user interface{}) error {
	return vs.ValidateBeforeCreate(ctx, user)
}

// Custom validation functions (duplicated from api package for database use)

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

// ValidationError represents a validation error with context
type ValidationError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Value   string `json:"value"`
	Message string `json:"message"`
	Context string `json:"context"`
}

// FormatValidationErrors formats validator errors into a readable format
func (vs *ValidatedStorage) FormatValidationErrors(err error, context string) []ValidationError {
	var validationErrors []ValidationError
	
	if validatorErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validatorErrors {
			validationError := ValidationError{
				Field:   fieldError.Field(),
				Tag:     fieldError.Tag(),
				Value:   fmt.Sprintf("%v", fieldError.Value()),
				Message: vs.getErrorMessage(fieldError),
				Context: context,
			}
			validationErrors = append(validationErrors, validationError)
		}
	}
	
	return validationErrors
}

// getErrorMessage returns a human-readable error message for a field error
func (vs *ValidatedStorage) getErrorMessage(fieldError validator.FieldError) string {
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
		return fmt.Sprintf("%s must be a valid agent type (llm, tool, workflow, monitor, standard)", field)
	case "build_target":
		return fmt.Sprintf("%s must be a valid build target (plugin, wasm, template)", field)
	case "status":
		return fmt.Sprintf("%s must be a valid status", field)
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

// GetValidator returns the underlying validator instance
func (vs *ValidatedStorage) GetValidator() *validator.Validate {
	return vs.validator
}

// RegisterCustomValidation registers a custom validation function
func (vs *ValidatedStorage) RegisterCustomValidation(tag string, fn validator.Func) error {
	return vs.validator.RegisterValidation(tag, fn)
}

// ValidateValue validates a single value against validation tags
func (vs *ValidatedStorage) ValidateValue(value interface{}, tag string) error {
	return vs.validator.Var(value, tag)
}

// ValidatedAgentStorage wraps UnifiedAgentStorage with validation
type ValidatedAgentStorage struct {
	storage   interface{} // The actual storage implementation
	validator *ValidatedStorage
}

// NewValidatedAgentStorage creates a new validated agent storage
func NewValidatedAgentStorage(storage interface{}) *ValidatedAgentStorage {
	return &ValidatedAgentStorage{
		storage:   storage,
		validator: NewValidatedStorage(),
	}
}

// ValidateAndCreate validates data before creating an agent
func (vas *ValidatedAgentStorage) ValidateAndCreate(ctx context.Context, agent interface{}) error {
	// Validate the agent data
	if err := vas.validator.ValidateAgentData(ctx, agent); err != nil {
		return fmt.Errorf("agent validation failed: %w", err)
	}
	
	// If validation passes, proceed with creation
	// Note: This would call the actual storage implementation
	// For now, we just return nil to indicate validation passed
	return nil
}

// ValidateAndUpdate validates data before updating an agent
func (vas *ValidatedAgentStorage) ValidateAndUpdate(ctx context.Context, agent interface{}) error {
	// Validate the agent data
	if err := vas.validator.ValidateBeforeUpdate(ctx, agent); err != nil {
		return fmt.Errorf("agent update validation failed: %w", err)
	}
	
	// If validation passes, proceed with update
	return nil
}

// ValidateAndDelete validates data before deleting an agent
func (vas *ValidatedAgentStorage) ValidateAndDelete(ctx context.Context, id string) error {
	// Validate the ID
	if err := vas.validator.ValidateBeforeDelete(ctx, id); err != nil {
		return fmt.Errorf("agent deletion validation failed: %w", err)
	}
	
	// If validation passes, proceed with deletion
	return nil
}
