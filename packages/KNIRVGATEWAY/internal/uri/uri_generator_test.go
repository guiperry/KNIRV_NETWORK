package uri

import (
	"strings"
	"testing"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/types"
)

func TestParseResourceURI(t *testing.T) {
	testCases := []struct {
		name          string
		uri           string
		expectedID    string
		expectedType  string
		expectedPath  string
		expectedError bool
	}{
		{
			name:         "Valid Chain URI",
			uri:          "knirv://abc123.chain/",
			expectedID:   "abc123",
			expectedType: "chain",
			expectedPath: "/",
		},
		{
			name:         "Valid Chain URI with Path",
			uri:          "knirv://abc123.chain/block",
			expectedID:   "abc123",
			expectedType: "chain",
			expectedPath: "/block",
		},
		{
			name:         "Valid Chain URI with Query",
			uri:          "knirv://abc123.chain/block?hash=xyz789",
			expectedID:   "abc123",
			expectedType: "chain",
			expectedPath: "/block",
		},
		{
			name:         "Valid NRN URI",
			uri:          "knirv://content123.nrn/",
			expectedID:   "content123",
			expectedType: "nrn",
			expectedPath: "/",
		},
		{
			name:          "Invalid Scheme",
			uri:           "http://abc123.chain/",
			expectedError: true,
		},
		{
			name:          "Invalid Authority Format",
			uri:           "knirv://abc123/",
			expectedError: true,
		},
		{
			name:          "Invalid Resource Type",
			uri:           "knirv://abc123.invalid/",
			expectedError: false, // We don't validate resource type in ParseResourceURI
			expectedID:    "abc123",
			expectedType:  "invalid",
			expectedPath:  "/",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id, resourceType, path, params, err := ParseResourceURI(tc.uri)

			if tc.expectedError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if id != tc.expectedID {
				t.Errorf("Expected ID '%s', got '%s'", tc.expectedID, id)
			}

			if resourceType != tc.expectedType {
				t.Errorf("Expected resource type '%s', got '%s'", tc.expectedType, resourceType)
			}

			if path != tc.expectedPath {
				t.Errorf("Expected path '%s', got '%s'", tc.expectedPath, path)
			}

			t.Logf("Params: %v", params)
		})
	}
}

func TestGenerateResourceURI(t *testing.T) {
	testCases := []struct {
		name           string
		id             string
		resourceType   string
		path           string
		params         map[string]string
		expectedPrefix string
	}{
		{
			name:           "Chain URI",
			id:             "abc123",
			resourceType:   "chain",
			path:           "",
			params:         nil,
			expectedPrefix: "knirv://abc123.chain/",
		},
		{
			name:           "Chain URI with Path",
			id:             "abc123",
			resourceType:   "chain",
			path:           "block",
			params:         nil,
			expectedPrefix: "knirv://abc123.chain/block",
		},
		{
			name:           "Chain URI with Path and Params",
			id:             "abc123",
			resourceType:   "chain",
			path:           "block",
			params:         map[string]string{"hash": "xyz789"},
			expectedPrefix: "knirv://abc123.chain/block?hash=xyz789",
		},
		{
			name:           "NRN URI",
			id:             "content123",
			resourceType:   "nrn",
			path:           "",
			params:         nil,
			expectedPrefix: "knirv://content123.nrn/",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			uri := GenerateResourceURI(tc.id, tc.resourceType, tc.path, tc.params)

			if !strings.HasPrefix(uri, tc.expectedPrefix) {
				t.Errorf("Expected URI to start with '%s', got '%s'", tc.expectedPrefix, uri)
			}

			t.Logf("Generated URI: %s", uri)
		})
	}
}

func TestGenerateCapabilityID(t *testing.T) {
	testCases := []struct {
		name           string
		capabilityName string
		capType        string
		expected       string
	}{
		{
			name:           "Resource capability",
			capabilityName: "file-handler",
			capType:        "RESOURCE",
			expected:       "file-handler.resource",
		},
		{
			name:           "Tool capability",
			capabilityName: "calculator",
			capType:        "TOOL",
			expected:       "calculator.tool",
		},
		{
			name:           "Prompt capability",
			capabilityName: "code-review",
			capType:        "PROMPT",
			expected:       "code-review.prompt",
		},
		{
			name:           "Empty name fallback",
			capabilityName: "",
			capType:        "TOOL",
			expected:       "unnamed-capability.tool",
		},
		{
			name:           "Name with spaces",
			capabilityName: "my test tool",
			capType:        "TOOL",
			expected:       "my-test-tool.tool",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GenerateCapabilityID(tc.capabilityName, types.CapabilityType(tc.capType))
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}
