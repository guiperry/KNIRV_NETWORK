package governance

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordComplianceEventCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/compliance/events", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "node-1", body["node_id"])
		assert.Equal(t, "violation", body["event_type"])

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"evt-1"}`))
	}))
	defer server.Close()

	oldURL := serverURL
	serverURL = server.URL
	defer func() { serverURL = oldURL }()

	recordComplianceEventCmd.Flags().Set("node-id", "node-1")
	recordComplianceEventCmd.Flags().Set("agent-id", "agent-1")
	recordComplianceEventCmd.Flags().Set("event-type", "violation")
	recordComplianceEventCmd.Flags().Set("severity", "high")

	err := recordComplianceEventCmd.Execute()
	require.NoError(t, err)
}

func TestListComplianceEventsCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/compliance/events", r.URL.Path)
		assert.Equal(t, "node-1", r.URL.Query().Get("node_id"))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"evt-1"}]`))
	}))
	defer server.Close()

	oldURL := serverURL
	serverURL = server.URL
	defer func() { serverURL = oldURL }()

	listComplianceEventsCmd.Flags().Set("node-id", "node-1")
	err := listComplianceEventsCmd.Execute()
	require.NoError(t, err)
}

func TestListFrameworksCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/compliance/frameworks", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"knirv-network"}]`))
	}))
	defer server.Close()

	oldURL := serverURL
	serverURL = server.URL
	defer func() { serverURL = oldURL }()

	err := listFrameworksCmd.Execute()
	require.NoError(t, err)
}

func TestGetComplianceStatusCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/compliance/status", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"node_id":"node-1","status":"compliant"}`))
	}))
	defer server.Close()

	oldURL := serverURL
	serverURL = server.URL
	defer func() { serverURL = oldURL }()

	getComplianceStatusCmd.Flags().Set("node-id", "node-1")
	err := getComplianceStatusCmd.Execute()
	require.NoError(t, err)
}

func TestVerifyComplianceChainCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/compliance/chain/verify", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"valid":true}`))
	}))
	defer server.Close()

	oldURL := serverURL
	serverURL = server.URL
	defer func() { serverURL = oldURL }()

	err := verifyComplianceChainCmd.Execute()
	require.NoError(t, err)
}

func TestRecordBreakerSuccessCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/reliability/breakers/breaker-1/success", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	oldURL := serverURL
	serverURL = server.URL
	defer func() { serverURL = oldURL }()

	recordSuccessCmd.SetArgs([]string{"breaker-1"})
	err := recordSuccessCmd.Execute()
	require.NoError(t, err)
}

func TestRecordBreakerFailureCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/reliability/breakers/breaker-1/failure", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	oldURL := serverURL
	serverURL = server.URL
	defer func() { serverURL = oldURL }()

	recordFailureCmd.SetArgs([]string{"breaker-1"})
	err := recordFailureCmd.Execute()
	require.NoError(t, err)
}

func TestDefineSLOCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/reliability/slos/define", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "uptime", body["name"])
		assert.Equal(t, 0.99, body["target"])

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	oldURL := serverURL
	serverURL = server.URL
	defer func() { serverURL = oldURL }()

	defineSLO.Flags().Set("name", "uptime")
	defineSLO.Flags().Set("target", "0.99")
	err := defineSLO.Execute()
	require.NoError(t, err)
}

func TestListSLOCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/reliability/slos", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"name":"uptime"}]`))
	}))
	defer server.Close()

	oldURL := serverURL
	serverURL = server.URL
	defer func() { serverURL = oldURL }()

	err := listSLOsCmd.Execute()
	require.NoError(t, err)
}

func TestArmKillSwitchCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/reliability/kill-switches", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	oldURL := serverURL
	serverURL = server.URL
	defer func() { serverURL = oldURL }()

	armKillSwitchCmd.Flags().Set("agent-id", "agent-1")
	armKillSwitchCmd.Flags().Set("node-id", "node-1")
	err := armKillSwitchCmd.Execute()
	require.NoError(t, err)
}

func TestCreateEnvelopeCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/identity/envelopes", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"env-1"}`))
	}))
	defer server.Close()

	oldURL := serverURL
	serverURL = server.URL
	defer func() { serverURL = oldURL }()

	createEnvelopeCmd.Flags().Set("node-id", "node-1")
	createEnvelopeCmd.Flags().Set("agent-id", "agent-1")
	createEnvelopeCmd.Flags().Set("source", "test")
	err := createEnvelopeCmd.Execute()
	require.NoError(t, err)
}

func TestRevokeNodeCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/identity/revoke", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"revoked"}`))
	}))
	defer server.Close()

	oldURL := serverURL
	serverURL = server.URL
	defer func() { serverURL = oldURL }()

	revokeNodeCmd.Flags().Set("identity-id", "id-1")
	revokeNodeCmd.Flags().Set("node-id", "node-1")
	revokeNodeCmd.Flags().Set("reason", "compromised")
	revokeNodeCmd.Flags().Set("revoked-by", "admin")
	err := revokeNodeCmd.Execute()
	require.NoError(t, err)
}

func TestCheckRevokedCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/identity/revoked/node-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"revoked":false}`))
	}))
	defer server.Close()

	oldURL := serverURL
	serverURL = server.URL
	defer func() { serverURL = oldURL }()

	checkRevokedCmd.SetArgs([]string{"node-1"})
	err := checkRevokedCmd.Execute()
	require.NoError(t, err)
}

func TestNormalizeInputCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/policy/inputs", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"normalized":true}`))
	}))
	defer server.Close()

	oldURL := serverURL
	serverURL = server.URL
	defer func() { serverURL = oldURL }()

	normalizeInputCmd.Flags().Set("action", "read")
	normalizeInputCmd.Flags().Set("action-type", "api")
	normalizeInputCmd.Flags().Set("node-id", "node-1")
	err := normalizeInputCmd.Execute()
	require.NoError(t, err)
}

func TestRegisterToolCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/mcp/tools", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"registered":true}`))
	}))
	defer server.Close()

	oldURL := serverURL
	serverURL = server.URL
	defer func() { serverURL = oldURL }()

	registerToolCmd.Flags().Set("name", "test-tool")
	registerToolCmd.Flags().Set("description", "A test tool")
	registerToolCmd.Flags().Set("schema", `{"type":"object"}`)
	err := registerToolCmd.Execute()
	require.NoError(t, err)
}

func TestValidateSchemaCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/mcp/schema/validate", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"valid":true}`))
	}))
	defer server.Close()

	oldURL := serverURL
	serverURL = server.URL
	defer func() { serverURL = oldURL }()

	validateSchemaCmd.Flags().Set("tool-name", "test-tool")
	validateSchemaCmd.Flags().Set("args", `{}`)
	err := validateSchemaCmd.Execute()
	require.NoError(t, err)
}

func TestScanInjectionCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/mcp/injection/scan", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"flagged":[]}`))
	}))
	defer server.Close()

	oldURL := serverURL
	serverURL = server.URL
	defer func() { serverURL = oldURL }()

	scanInjectionCmd.Flags().Set("tool-name", "test-tool")
	scanInjectionCmd.Flags().Set("args", `{}`)
	err := scanInjectionCmd.Execute()
	require.NoError(t, err)
}

func TestSanitizeResponseCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/mcp/response/sanitize", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"sanitized":"ok"}`))
	}))
	defer server.Close()

	oldURL := serverURL
	serverURL = server.URL
	defer func() { serverURL = oldURL }()

	sanitizeResponseCmd.Flags().Set("content", "safe content")
	err := sanitizeResponseCmd.Execute()
	require.NoError(t, err)
}

func TestRevokeIdentityCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/identity/revoke", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"revoked"}`))
	}))
	defer server.Close()

	oldURL := serverURL
	serverURL = server.URL
	defer func() { serverURL = oldURL }()

	revokeIdentityCmd.Flags().Set("identity-id", "id-1")
	revokeIdentityCmd.Flags().Set("node-id", "node-1")
	revokeIdentityCmd.Flags().Set("reason", "compromised")
	revokeIdentityCmd.Flags().Set("revoked-by", "admin")
	err := revokeIdentityCmd.Execute()
	require.NoError(t, err)
}

func TestVerifyRevocationChainCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/identity/revocation/chain/verify", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"valid":true}`))
	}))
	defer server.Close()

	oldURL := serverURL
	serverURL = server.URL
	defer func() { serverURL = oldURL }()

	err := verifyRevocationChainCmd.Execute()
	require.NoError(t, err)
}

func TestListRevocationsCmd(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/identity/revocation/list", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"identity_id":"id-1"}]`))
	}))
	defer server.Close()

	oldURL := serverURL
	serverURL = server.URL
	defer func() { serverURL = oldURL }()

	err := listRevocationsCmd.Execute()
	require.NoError(t, err)
}
