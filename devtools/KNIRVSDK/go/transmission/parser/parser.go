package parser

import (
	"fmt"
	"net/url"
	"strings"
)

// KnirvURI represents a parsed knirv:// URI
type KnirvURI struct {
	ID           string
	ResourceType string
	Path         string
	Query        url.Values
	Raw          string
}

// ErrInvalidURI is returned when a URI is invalid
type ErrInvalidURI struct {
	Message string
	URI     string
}

func (e ErrInvalidURI) Error() string {
	return fmt.Sprintf("invalid knirv URI (%s): %s", e.URI, e.Message)
}

// Parse parses a knirv:// URI string into a KnirvURI struct
func Parse(uriString string) (*KnirvURI, error) {
	parsedURL, err := url.Parse(uriString)
	if err != nil {
		return nil, &ErrInvalidURI{
			Message: fmt.Sprintf("failed to parse URI: %v", err),
			URI:     uriString,
		}
	}

	// Check scheme
	if parsedURL.Scheme != "knirv" {
		return nil, &ErrInvalidURI{
			Message: fmt.Sprintf("invalid scheme: expected 'knirv', got '%s'", parsedURL.Scheme),
			URI:     uriString,
		}
	}

	// Extract ID and ResourceType from authority
	host := parsedURL.Host
	hostParts := strings.Split(host, ".")
	if len(hostParts) != 2 {
		return nil, &ErrInvalidURI{
			Message: "invalid authority format: expected <ID>.<ResourceType>",
			URI:     uriString,
		}
	}

	id := hostParts[0]
	resourceType := hostParts[1]

	// Validate ID and ResourceType are non-empty
	if id == "" || resourceType == "" {
		return nil, &ErrInvalidURI{
			Message: "ID and ResourceType cannot be empty",
			URI:     uriString,
		}
	}

	// Normalize path
	path := parsedURL.Path
	if path == "" {
		path = "/" // Default path
	}

	return &KnirvURI{
		ID:           id,
		ResourceType: resourceType,
		Path:         path,
		Query:        parsedURL.Query(),
		Raw:          uriString,
	}, nil
}

// String returns the string representation of the KnirvURI
func (k *KnirvURI) String() string {
	return k.Raw
}

// GetQueryParam returns the first value for the given query parameter
func (k *KnirvURI) GetQueryParam(key string) string {
	return k.Query.Get(key)
}

// HasQueryParam checks if the given query parameter exists
func (k *KnirvURI) HasQueryParam(key string) bool {
	_, exists := k.Query[key]
	return exists
}

// GetQueryParams returns all values for the given query parameter
func (k *KnirvURI) GetQueryParams(key string) []string {
	return k.Query[key]
}

// GetAllQueryParams returns all query parameters
func (k *KnirvURI) GetAllQueryParams() map[string][]string {
	return k.Query
}