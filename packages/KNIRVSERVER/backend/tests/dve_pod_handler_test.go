package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"backend_server/internal/web"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDVEPodRegisterPod(t *testing.T) {
	handler := web.NewDVEPodHandler("http://localhost:8084")
	require.NotNil(t, handler)

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := map[string]interface{}{
		"attestation": map[string]interface{}{
			"node_id":     "dvepod-test1234",
			"tee_type":    "wasmer",
			"wasm_hash":   "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			"measurement": "abcdef1234567890",
			"public_key":  "04deadbeef",
			"timestamp":   1700000000,
			"version":     "dvepod/1.0",
		},
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/dve/pod/register", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			SessionID string `json:"session_id"`
			NodeID    string `json:"node_id"`
			WSURL     string `json:"ws_url"`
			Status    string `json:"status"`
		} `json:"data"`
		Message   string `json:"message"`
		Timestamp string `json:"timestamp"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.True(t, strings.HasPrefix(resp.Data.SessionID, "dvepod-"))
	assert.Equal(t, "dvepod-test1234", resp.Data.NodeID)
	assert.Equal(t, "registered", resp.Data.Status)
	assert.Contains(t, resp.Data.WSURL, "/api/v1/dve/pod/ws/")
}

func TestDVEPodRegisterPodMissingNodeID(t *testing.T) {
	handler := web.NewDVEPodHandler("http://localhost:8084")
	require.NotNil(t, handler)

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := map[string]interface{}{
		"attestation": map[string]interface{}{
			"tee_type":  "wasmer",
			"wasm_hash": "abcdef",
		},
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/dve/pod/register", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Error, "node_id")
}

func TestDVEPodRegisterPodInvalidJSON(t *testing.T) {
	handler := web.NewDVEPodHandler("http://localhost:8084")
	require.NotNil(t, handler)

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest("POST", "/dve/pod/register", bytes.NewReader([]byte("{invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDVEPodListSessions(t *testing.T) {
	handler := web.NewDVEPodHandler("http://localhost:8084")
	require.NotNil(t, handler)

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// Register a pod first
	body := map[string]interface{}{
		"attestation": map[string]interface{}{
			"node_id":   "dvepod-test",
			"tee_type":  "wasmer",
			"wasm_hash": "abc",
		},
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/dve/pod/register", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// List sessions
	req2 := httptest.NewRequest("GET", "/dve/pod/sessions", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)

	var resp struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	json.Unmarshal(w2.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
	assert.True(t, len(resp.Data) > 0)
}

func TestDVEPodListSessionsEmpty(t *testing.T) {
	handler := web.NewDVEPodHandler("http://localhost:8084")
	require.NotNil(t, handler)

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/dve/pod/sessions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
	assert.True(t, strings.Contains(string(resp.Data), "[]"))
}

func TestDVEPodGetSession(t *testing.T) {
	handler := web.NewDVEPodHandler("http://localhost:8084")
	require.NotNil(t, handler)

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// Register a pod
	body := map[string]interface{}{
		"attestation": map[string]interface{}{
			"node_id":   "dvepod-get-session",
			"tee_type":  "browser-wasm",
			"wasm_hash": "def",
		},
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/dve/pod/register", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp struct {
		Data struct {
			SessionID string `json:"session_id"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	sessionID := createResp.Data.SessionID
	require.NotEmpty(t, sessionID)

	// Get session by ID
	req2 := httptest.NewRequest("GET", "/dve/pod/sessions/"+sessionID, nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)

	var getResp struct {
		Success bool `json:"success"`
		Data    struct {
			SessionID string `json:"session_id"`
			NodeID    string `json:"node_id"`
		} `json:"data"`
	}
	json.Unmarshal(w2.Body.Bytes(), &getResp)
	assert.True(t, getResp.Success)
	assert.Equal(t, sessionID, getResp.Data.SessionID)
	assert.Equal(t, "dvepod-get-session", getResp.Data.NodeID)
}

func TestDVEPodGetSessionNotFound(t *testing.T) {
	handler := web.NewDVEPodHandler("http://localhost:8084")
	require.NotNil(t, handler)

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/dve/pod/sessions/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.False(t, resp.Success)
}

func TestDVEPodMultipleRegistrations(t *testing.T) {
	handler := web.NewDVEPodHandler("http://localhost:8084")
	require.NotNil(t, handler)

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// Register three pods
	for i := 0; i < 3; i++ {
		body := map[string]interface{}{
			"attestation": map[string]interface{}{
				"node_id":  "dvepod-multi-" + string(rune('0'+i)),
				"tee_type": "wasmer",
			},
		}
		payload, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/dve/pod/register", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	}

	// Verify all three are listed
	req := httptest.NewRequest("GET", "/dve/pod/sessions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp struct {
		Data []interface{} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Len(t, resp.Data, 3)
}

func TestDVEPodAttestUpgrade(t *testing.T) {
	handler := web.NewDVEPodHandler("http://localhost:8084")
	require.NotNil(t, handler)

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := map[string]interface{}{
		"attestation": map[string]interface{}{
			"node_id":  "dvepod-attest-test",
			"tee_type": "wasmer",
		},
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/dve/pod/register", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp struct {
		Data struct {
			SessionID string `json:"session_id"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &createResp)

	attestBody := map[string]interface{}{
		"tee_type":    "sgx",
		"measurement": "abcdef1234567890",
		"public_key":  "04deadbeef",
		"signature":   "signed",
	}
	attestPayload, _ := json.Marshal(attestBody)
	req2 := httptest.NewRequest("POST", "/dve/pod/dvepod-attest-test/attest", bytes.NewReader(attestPayload))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)

	var attResp struct {
		Success bool `json:"success"`
		Data    struct {
			SessionID  string `json:"session_id"`
			NodeID     string `json:"node_id"`
			TrustLevel string `json:"trust_level"`
			Status     string `json:"status"`
		} `json:"data"`
	}
	err := json.Unmarshal(w2.Body.Bytes(), &attResp)
	require.NoError(t, err)
	assert.True(t, attResp.Success)
	assert.Equal(t, "dvepod-attest-test", attResp.Data.NodeID)
	assert.Equal(t, "L3", attResp.Data.TrustLevel)
	assert.Equal(t, "upgraded", attResp.Data.Status)

	req3 := httptest.NewRequest("GET", "/dve/pod/sessions/"+createResp.Data.SessionID, nil)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)

	var sessionResp struct {
		Data struct {
			TrustLevel string `json:"trust_level"`
		} `json:"data"`
	}
	json.Unmarshal(w3.Body.Bytes(), &sessionResp)
	assert.Equal(t, "L3", sessionResp.Data.TrustLevel)
}

func TestDVEPodAttestSameLevelRejected(t *testing.T) {
	handler := web.NewDVEPodHandler("http://localhost:8084")
	require.NotNil(t, handler)

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := map[string]interface{}{
		"attestation": map[string]interface{}{
			"node_id":  "dvepod-attest-same",
			"tee_type": "sgx",
		},
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/dve/pod/register", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	attestBody := map[string]interface{}{
		"tee_type":    "sgx",
		"measurement": "abc",
	}
	attestPayload, _ := json.Marshal(attestBody)
	req2 := httptest.NewRequest("POST", "/dve/pod/dvepod-attest-same/attest", bytes.NewReader(attestPayload))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusConflict, w2.Code)
}

func TestDVEPodAttestPodNotFound(t *testing.T) {
	handler := web.NewDVEPodHandler("http://localhost:8084")
	require.NotNil(t, handler)

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	attestBody := map[string]interface{}{
		"tee_type": "sgx",
	}
	attestPayload, _ := json.Marshal(attestBody)
	req := httptest.NewRequest("POST", "/dve/pod/nonexistent/attest", bytes.NewReader(attestPayload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDVEPodAttestInvalidJSON(t *testing.T) {
	handler := web.NewDVEPodHandler("http://localhost:8084")
	require.NotNil(t, handler)

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest("POST", "/dve/pod/somepod/attest", bytes.NewReader([]byte("{invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDVEPodWithNodeIDFallback(t *testing.T) {
	handler := web.NewDVEPodHandler("http://localhost:8084")
	require.NotNil(t, handler)

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := map[string]interface{}{
		"node_id":  "dvepod-fallback",
		"tee_type": "wasmer",
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/dve/pod/register", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			NodeID string `json:"node_id"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "dvepod-fallback", resp.Data.NodeID)
}
