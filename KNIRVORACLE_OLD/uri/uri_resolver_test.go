package uri

import (
	"testing"
)

func TestParseURI(t *testing.T) {
	resolver := NewURIResolver()

	tests := []struct {
		name           string
		uri            string
		wantAuthority  string
		wantIdentifier string
		wantType       string
		wantSubPath    string
		wantErr        bool
	}{
		{
			name:           "Valid URI with subpath",
			uri:            "agent://tunnel.agent.com/abc123.dev/some/path?query=value",
			wantAuthority:  "tunnel.agent.com",
			wantIdentifier: "abc123",
			wantType:       "dev",
			wantSubPath:    "/some/path?query=value",
			wantErr:        false,
		},
		{
			name:           "Valid URI without subpath",
			uri:            "agent://tunnel.agent.com/abc123.dev",
			wantAuthority:  "tunnel.agent.com",
			wantIdentifier: "abc123",
			wantType:       "dev",
			wantSubPath:    "",
			wantErr:        false,
		},
		{
			name:           "Valid URI with chainID",
			uri:            "agent://tunnel.agent.com/agent-default.chain",
			wantAuthority:  "tunnel.agent.com",
			wantIdentifier: "agent-default",
			wantType:       "chain",
			wantSubPath:    "",
			wantErr:        false,
		},
		{
			name:    "Invalid scheme",
			uri:     "http://tunnel.agent.com/abc123.dev",
			wantErr: true,
		},
		{
			name:    "Missing path",
			uri:     "agent://tunnel.agent.com",
			wantErr: true,
		},
		{
			name:    "Missing resource type",
			uri:     "agent://tunnel.agent.com/abc123",
			wantErr: true,
		},
		{
			name:    "Invalid Resource Type",
			uri:     "agent://tunnel.agent.com/abc123.invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authority, identifier, resourceType, subPath, err := resolver.ParseURI(tt.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseURI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if authority != tt.wantAuthority {
				t.Errorf("ParseURI() authority = %v, want %v", authority, tt.wantAuthority)
			}
			if identifier != tt.wantIdentifier {
				t.Errorf("ParseURI() identifier = %v, want %v", identifier, tt.wantIdentifier)
			}
			if resourceType != tt.wantType {
				t.Errorf("ParseURI() resourceType = %v, want %v", resourceType, tt.wantType)
			}
			if subPath != tt.wantSubPath {
				t.Errorf("ParseURI() subPath = %v, want %v", subPath, tt.wantSubPath)
			}
		})
	}
}
