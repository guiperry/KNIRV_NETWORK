package api

import (
	"encoding/json"
	"net/http"
)

// StandardResponse represents the standardized API response format
type StandardResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
	Code    string      `json:"code,omitempty"`
}

// StandardListResponse represents a standardized response for list operations
type StandardListResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Count   int         `json:"count"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
	Code    string      `json:"code,omitempty"`
}

// RespondWithSuccess sends a successful response with data
func RespondWithSuccess(w http.ResponseWriter, data interface{}, message string) {
	response := StandardResponse{
		Success: true,
		Data:    data,
		Message: message,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// RespondWithSuccessAndStatus sends a successful response with custom status code
func RespondWithSuccessAndStatus(w http.ResponseWriter, statusCode int, data interface{}, message string) {
	response := StandardResponse{
		Success: true,
		Data:    data,
		Message: message,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// RespondWithList sends a successful response for list operations
func RespondWithList(w http.ResponseWriter, data interface{}, count int, message string) {
	response := StandardListResponse{
		Success: true,
		Data:    data,
		Count:   count,
		Message: message,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// RespondWithError sends an error response
func RespondWithError(w http.ResponseWriter, statusCode int, errorMessage string, errorCode string) {
	response := StandardResponse{
		Success: false,
		Error:   errorMessage,
		Code:    errorCode,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// RespondWithValidationError sends a validation error response
func RespondWithValidationError(w http.ResponseWriter, errorMessage string) {
	RespondWithError(w, http.StatusBadRequest, errorMessage, "VALIDATION_ERROR")
}

// RespondWithNotFound sends a not found error response
func RespondWithNotFound(w http.ResponseWriter, resource string) {
	RespondWithError(w, http.StatusNotFound, resource+" not found", "NOT_FOUND")
}

// RespondWithInternalError sends an internal server error response
func RespondWithInternalError(w http.ResponseWriter, errorMessage string) {
	RespondWithError(w, http.StatusInternalServerError, errorMessage, "INTERNAL_ERROR")
}

// RespondWithUnauthorized sends an unauthorized error response
func RespondWithUnauthorized(w http.ResponseWriter, errorMessage string) {
	RespondWithError(w, http.StatusUnauthorized, errorMessage, "UNAUTHORIZED")
}

// RespondWithForbidden sends a forbidden error response
func RespondWithForbidden(w http.ResponseWriter, errorMessage string) {
	RespondWithError(w, http.StatusForbidden, errorMessage, "FORBIDDEN")
}

// RespondWithConflict sends a conflict error response
func RespondWithConflict(w http.ResponseWriter, errorMessage string) {
	RespondWithError(w, http.StatusConflict, errorMessage, "CONFLICT")
}

// RespondWithCreated sends a successful creation response
func RespondWithCreated(w http.ResponseWriter, data interface{}, message string) {
	RespondWithSuccessAndStatus(w, http.StatusCreated, data, message)
}

// RespondWithNoContent sends a successful no content response
func RespondWithNoContent(w http.ResponseWriter, message string) {
	response := StandardResponse{
		Success: true,
		Message: message,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
	json.NewEncoder(w).Encode(response)
}

// RespondWithAccepted sends an accepted response for async operations
func RespondWithAccepted(w http.ResponseWriter, data interface{}, message string) {
	RespondWithSuccessAndStatus(w, http.StatusAccepted, data, message)
}

// Common error codes
const (
	ErrorCodeValidation    = "VALIDATION_ERROR"
	ErrorCodeNotFound      = "NOT_FOUND"
	ErrorCodeUnauthorized  = "UNAUTHORIZED"
	ErrorCodeForbidden     = "FORBIDDEN"
	ErrorCodeConflict      = "CONFLICT"
	ErrorCodeInternal      = "INTERNAL_ERROR"
	ErrorCodeBadRequest    = "BAD_REQUEST"
	ErrorCodeTimeout       = "TIMEOUT"
	ErrorCodeRateLimit     = "RATE_LIMIT"
	ErrorCodeMaintenance   = "MAINTENANCE"
)

// Common success messages
const (
	MessageCreated         = "Resource created successfully"
	MessageUpdated         = "Resource updated successfully"
	MessageDeleted         = "Resource deleted successfully"
	MessageRetrieved       = "Resource retrieved successfully"
	MessageListRetrieved   = "Resources retrieved successfully"
	MessageOperationSuccess = "Operation completed successfully"
)

// Helper function to extract error details from Go errors
func ExtractErrorDetails(err error) (string, string) {
	if err == nil {
		return "", ""
	}
	
	// You can add more sophisticated error parsing here
	// For now, return the error message and a generic code
	return err.Error(), ErrorCodeInternal
}

// RespondWithGoError is a convenience function for handling Go errors
func RespondWithGoError(w http.ResponseWriter, err error, defaultMessage string) {
	if err == nil {
		RespondWithInternalError(w, "Unexpected nil error")
		return
	}
	
	errorMessage, errorCode := ExtractErrorDetails(err)
	if errorMessage == "" {
		errorMessage = defaultMessage
	}
	
	// Determine status code based on error type
	statusCode := http.StatusInternalServerError
	switch errorCode {
	case ErrorCodeValidation, ErrorCodeBadRequest:
		statusCode = http.StatusBadRequest
	case ErrorCodeNotFound:
		statusCode = http.StatusNotFound
	case ErrorCodeUnauthorized:
		statusCode = http.StatusUnauthorized
	case ErrorCodeForbidden:
		statusCode = http.StatusForbidden
	case ErrorCodeConflict:
		statusCode = http.StatusConflict
	case ErrorCodeTimeout:
		statusCode = http.StatusRequestTimeout
	case ErrorCodeRateLimit:
		statusCode = http.StatusTooManyRequests
	}
	
	RespondWithError(w, statusCode, errorMessage, errorCode)
}

// ValidateJSONRequest validates that the request has valid JSON content type
func ValidateJSONRequest(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	return contentType == "application/json" || contentType == "application/json; charset=utf-8"
}

// SetStandardHeaders sets common headers for all API responses
func SetStandardHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-API-Version", "v1")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}
