package parser

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		uriString   string
		wantID      string
		wantType    string
		wantPath    string
		wantQuery   map[string]string
		wantErr     bool
		wantErrType string
	}{
		{
			name:      "Valid chain URI with path and query",
			uriString: "knirv://mychain-alpha.chain/block?number=123&validator=node1",
			wantID:    "mychain-alpha",
			wantType:  "chain",
			wantPath:  "/block",
			wantQuery: map[string]string{
				"number":    "123",
				"validator": "node1",
			},
			wantErr: false,
		},
		{
			name:      "Valid nrn URI with path and query",
			uriString: "knirv://content-xyz-123.nrn/data?version=1.2&format=json",
			wantID:    "content-xyz-123",
			wantType:  "nrn",
			wantPath:  "/data",
			wantQuery: map[string]string{
				"version": "1.2",
				"format":  "json",
			},
			wantErr: false,
		},
		{
			name:      "Valid URI with default path",
			uriString: "knirv://another-id.chain/",
			wantID:    "another-id",
			wantType:  "chain",
			wantPath:  "/",
			wantQuery: map[string]string{},
			wantErr:   false,
		},
		{
			name:      "Valid URI without path",
			uriString: "knirv://short.nrn",
			wantID:    "short",
			wantType:  "nrn",
			wantPath:  "/",
			wantQuery: map[string]string{},
			wantErr:   false,
		},
		{
			name:        "Invalid scheme",
			uriString:   "invalid://mychain.chain/",
			wantErr:     true,
			wantErrType: "ErrInvalidURI",
		},
		{
			name:        "Invalid authority format",
			uriString:   "knirv://mychainchain",
			wantErr:     true,
			wantErrType: "ErrInvalidURI",
		},
		{
			name:        "Empty ID",
			uriString:   "knirv://.chain",
			wantErr:     true,
			wantErrType: "ErrInvalidURI",
		},
		{
			name:        "Empty ResourceType",
			uriString:   "knirv://mychain.",
			wantErr:     true,
			wantErrType: "ErrInvalidURI",
		},
		{
			name:        "Malformed URI",
			uriString:   "knirv://mychain.chain:invalid",
			wantErr:     true,
			wantErrType: "ErrInvalidURI",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.uriString)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				// Check error type if specified
				if tt.wantErrType != "" {
					_, ok := err.(*ErrInvalidURI)
					if !ok {
						t.Errorf("Parse() error type = %T, want %s", err, tt.wantErrType)
					}
				}
				return
			}

			if got.ID != tt.wantID {
				t.Errorf("Parse() ID = %v, want %v", got.ID, tt.wantID)
			}
			if got.ResourceType != tt.wantType {
				t.Errorf("Parse() ResourceType = %v, want %v", got.ResourceType, tt.wantType)
			}
			if got.Path != tt.wantPath {
				t.Errorf("Parse() Path = %v, want %v", got.Path, tt.wantPath)
			}

			// Check query parameters
			for k, v := range tt.wantQuery {
				if got.Query.Get(k) != v {
					t.Errorf("Parse() Query[%s] = %v, want %v", k, got.Query.Get(k), v)
				}
			}

			// Check that Raw is set correctly
			if got.Raw != tt.uriString {
				t.Errorf("Parse() Raw = %v, want %v", got.Raw, tt.uriString)
			}
		})
	}
}

func TestKnirvURI_GetQueryParam(t *testing.T) {
	uri := &KnirvURI{
		ID:           "test",
		ResourceType: "chain",
		Path:         "/block",
		Query: map[string][]string{
			"number": {"123"},
			"multi":  {"value1", "value2"},
		},
		Raw: "knirv://test.chain/block?number=123&multi=value1&multi=value2",
	}

	if got := uri.GetQueryParam("number"); got != "123" {
		t.Errorf("KnirvURI.GetQueryParam() = %v, want %v", got, "123")
	}

	if got := uri.GetQueryParam("multi"); got != "value1" {
		t.Errorf("KnirvURI.GetQueryParam() = %v, want %v", got, "value1")
	}

	if got := uri.GetQueryParam("nonexistent"); got != "" {
		t.Errorf("KnirvURI.GetQueryParam() = %v, want %v", got, "")
	}
}

func TestKnirvURI_HasQueryParam(t *testing.T) {
	uri := &KnirvURI{
		ID:           "test",
		ResourceType: "chain",
		Path:         "/block",
		Query: map[string][]string{
			"number": {"123"},
			"empty":  {},
		},
		Raw: "knirv://test.chain/block?number=123&empty",
	}

	if got := uri.HasQueryParam("number"); !got {
		t.Errorf("KnirvURI.HasQueryParam() = %v, want %v", got, true)
	}

	if got := uri.HasQueryParam("empty"); !got {
		t.Errorf("KnirvURI.HasQueryParam() = %v, want %v", got, true)
	}

	if got := uri.HasQueryParam("nonexistent"); got {
		t.Errorf("KnirvURI.HasQueryParam() = %v, want %v", got, false)
	}
}

func TestKnirvURI_GetQueryParams(t *testing.T) {
	uri := &KnirvURI{
		ID:           "test",
		ResourceType: "chain",
		Path:         "/block",
		Query: map[string][]string{
			"single": {"value"},
			"multi":  {"value1", "value2"},
			"empty":  {},
		},
		Raw: "knirv://test.chain/block?single=value&multi=value1&multi=value2&empty",
	}

	if got := uri.GetQueryParams("single"); len(got) != 1 || got[0] != "value" {
		t.Errorf("KnirvURI.GetQueryParams() = %v, want %v", got, []string{"value"})
	}

	if got := uri.GetQueryParams("multi"); len(got) != 2 || got[0] != "value1" || got[1] != "value2" {
		t.Errorf("KnirvURI.GetQueryParams() = %v, want %v", got, []string{"value1", "value2"})
	}

	if got := uri.GetQueryParams("empty"); len(got) != 0 {
		t.Errorf("KnirvURI.GetQueryParams() = %v, want %v", got, []string{})
	}

	if got := uri.GetQueryParams("nonexistent"); got != nil {
		t.Errorf("KnirvURI.GetQueryParams() = %v, want %v", got, nil)
	}
}

func TestKnirvURI_GetAllQueryParams(t *testing.T) {
	uri := &KnirvURI{
		ID:           "test",
		ResourceType: "chain",
		Path:         "/block",
		Query: map[string][]string{
			"param1": {"value1"},
			"param2": {"value2a", "value2b"},
		},
		Raw: "knirv://test.chain/block?param1=value1&param2=value2a&param2=value2b",
	}

	got := uri.GetAllQueryParams()
	if len(got) != 2 {
		t.Errorf("KnirvURI.GetAllQueryParams() returned %d params, want 2", len(got))
	}

	if len(got["param1"]) != 1 || got["param1"][0] != "value1" {
		t.Errorf("KnirvURI.GetAllQueryParams()[param1] = %v, want %v", got["param1"], []string{"value1"})
	}

	if len(got["param2"]) != 2 || got["param2"][0] != "value2a" || got["param2"][1] != "value2b" {
		t.Errorf("KnirvURI.GetAllQueryParams()[param2] = %v, want %v", got["param2"], []string{"value2a", "value2b"})
	}
}

func TestKnirvURI_String(t *testing.T) {
	uri := &KnirvURI{
		ID:           "test",
		ResourceType: "chain",
		Path:         "/block",
		Query: map[string][]string{
			"number": {"123"},
		},
		Raw: "knirv://test.chain/block?number=123",
	}

	if got := uri.String(); got != "knirv://test.chain/block?number=123" {
		t.Errorf("KnirvURI.String() = %v, want %v", got, "knirv://test.chain/block?number=123")
	}
}