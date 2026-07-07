package governance

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterDID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/oracle/v3/did/register", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewGovernanceClient(server.URL, "")
	err := client.RegisterDID(`{"id":"did:knirv:test"}`)
	assert.NoError(t, err)
}

func TestRegisterDIDError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid did"}`))
	}))
	defer server.Close()

	client := NewGovernanceClient(server.URL, "")
	err := client.RegisterDID(`invalid`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid did")
}

func TestResolveDID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/oracle/v3/did/did:knirv:test", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"did:knirv:test"}`))
	}))
	defer server.Close()

	client := NewGovernanceClient(server.URL, "")
	doc, err := client.ResolveDID("did:knirv:test")
	require.NoError(t, err)
	assert.Contains(t, string(doc), "did:knirv:test")
}

func TestResolveDIDError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer server.Close()

	client := NewGovernanceClient(server.URL, "")
	_, err := client.ResolveDID("did:knirv:nonexistent")
	assert.Error(t, err)
}

func TestCreateEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/identity/envelopes", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"env-1","status":"active"}`))
	}))
	defer server.Close()

	client := NewGovernanceClient("", server.URL)
	result, err := client.CreateEnvelope("node-1", "agent-1", "test")
	require.NoError(t, err)
	assert.Contains(t, string(result), "env-1")
}

func TestRevokeIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/identity/revoke", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "identity-1", body["identity_id"])
		assert.Equal(t, "compromised", body["reason"])

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"revoked"}`))
	}))
	defer server.Close()

	client := NewGovernanceClient("", server.URL)
	result, err := client.RevokeIdentity("identity-1", "node-1", "compromised", "admin")
	require.NoError(t, err)
	assert.Contains(t, string(result), "revoked")
}

func TestCheckRevoked(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/identity/revoked/node-1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"revoked":true}`))
	}))
	defer server.Close()

	client := NewGovernanceClient("", server.URL)
	revoked, err := client.CheckRevoked("node-1")
	require.NoError(t, err)
	assert.True(t, revoked)
}

func TestCheckRevokedFalse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"revoked":false}`))
	}))
	defer server.Close()

	client := NewGovernanceClient("", server.URL)
	revoked, err := client.CheckRevoked("node-1")
	require.NoError(t, err)
	assert.False(t, revoked)
}

func TestNormalizeInput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/policy/inputs", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "read", body["action"])

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"normalized":true}`))
	}))
	defer server.Close()

	client := NewGovernanceClient("", server.URL)
	result, err := client.NormalizeInput("node-1", "read", "api")
	require.NoError(t, err)
	assert.Contains(t, string(result), "normalized")
}

func TestRecordBreakerSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/reliability/breakers/breaker-1/success", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewGovernanceClient("", server.URL)
	err := client.RecordBreakerSuccess("breaker-1")
	assert.NoError(t, err)
}

func TestRecordBreakerFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/reliability/breakers/breaker-1/failure", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewGovernanceClient("", server.URL)
	err := client.RecordBreakerFailure("breaker-1")
	assert.NoError(t, err)
}

func TestIsBreakerAllowed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/reliability/breakers/breaker-1/allow", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"allowed":true}`))
	}))
	defer server.Close()

	client := NewGovernanceClient("", server.URL)
	allowed, err := client.IsBreakerAllowed("breaker-1")
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestDefineSLO(t *testing.T) {
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

	client := NewGovernanceClient("", server.URL)
	err := client.DefineSLO("uptime", 0.99)
	assert.NoError(t, err)
}

func TestRecordComplianceEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/compliance/events", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "violation", body["event_type"])
		assert.Equal(t, "high", body["severity"])

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"evt-1"}`))
	}))
	defer server.Close()

	client := NewGovernanceClient("", server.URL)
	result, err := client.RecordComplianceEvent("node-1", "agent-1", "violation", "high")
	require.NoError(t, err)
	assert.Contains(t, string(result), "evt-1")
}

func TestGetComplianceStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/compliance/status", r.URL.Path)
		assert.Equal(t, "node-1", r.URL.Query().Get("node_id"))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"node_id":"node-1","status":"compliant"}`))
	}))
	defer server.Close()

	client := NewGovernanceClient("", server.URL)
	result, err := client.GetComplianceStatus("node-1")
	require.NoError(t, err)
	assert.Contains(t, string(result), "compliant")
}

func TestVerifyComplianceChain(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/governance/compliance/chain/verify", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"valid":true}`))
	}))
	defer server.Close()

	client := NewGovernanceClient("", server.URL)
	valid, err := client.VerifyComplianceChain()
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal error"}`))
	}))
	defer server.Close()

	client := NewGovernanceClient("", server.URL)
	_, err := client.CreateEnvelope("n", "a", "s")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestNewGovernanceClient(t *testing.T) {
	client := NewGovernanceClient("http://oracle:1317", "http://server:8084")
	assert.NotNil(t, client)
	assert.NotNil(t, client.httpClient)
}

func TestBreakerAllowedFalse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"allowed":false}`))
	}))
	defer server.Close()

	client := NewGovernanceClient("", server.URL)
	allowed, err := client.IsBreakerAllowed("breaker-1")
	require.NoError(t, err)
	assert.False(t, allowed)
}
