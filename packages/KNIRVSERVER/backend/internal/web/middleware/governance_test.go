package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type mockIdentityChecker struct {
	revoked map[string]bool
}

func (m *mockIdentityChecker) IsRevoked(nodeID string) bool {
	return m.revoked[nodeID]
}

type mockPolicyEval struct {
	results map[string]bool
	reasons map[string]string
}

func (m *mockPolicyEval) EvaluateAction(nodeID, action string, context map[string]interface{}) (bool, string) {
	if m.results != nil {
		if allowed, ok := m.results[action]; ok {
			reason := ""
			if m.reasons != nil {
				reason = m.reasons[action]
			}
			return allowed, reason
		}
	}
	return true, ""
}

func setupRouter(gm *GovernanceMiddleware) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	protected := r.Group("/api/v1")
	protected.Use(gm.ZeroTrustMiddleware())
	protected.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	return r
}

func TestZeroTrustMiddlewareNoHeader(t *testing.T) {
	gm := NewGovernanceMiddleware(nil, nil)
	router := setupRouter(gm)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	assert.Contains(t, body["error"], "X-Node-ID")
}

func TestZeroTrustMiddlewareRevoked(t *testing.T) {
	ic := &mockIdentityChecker{revoked: map[string]bool{"revoked-node": true}}
	gm := NewGovernanceMiddleware(nil, ic)
	router := setupRouter(gm)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("X-Node-ID", "revoked-node")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	assert.Contains(t, body["error"], "revoked")
}

func TestZeroTrustMiddlewarePolicyDenied(t *testing.T) {
	pe := &mockPolicyEval{
		results: map[string]bool{"GET /api/v1/test": false},
		reasons: map[string]string{"GET /api/v1/test": "action not allowed"},
	}
	gm := NewGovernanceMiddleware(pe, nil)
	router := setupRouter(gm)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("X-Node-ID", "node-1")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestZeroTrustMiddlewareAllowed(t *testing.T) {
	ic := &mockIdentityChecker{revoked: make(map[string]bool)}
	pe := &mockPolicyEval{}
	gm := NewGovernanceMiddleware(pe, ic)
	router := setupRouter(gm)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	req.Header.Set("X-Node-ID", "node-42")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestZeroTrustMiddlewareHTTPNoHeader(t *testing.T) {
	gm := NewGovernanceMiddleware(nil, nil)
	handler := gm.ZeroTrustMiddlewareHTTP()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestZeroTrustMiddlewareHTTPRevoked(t *testing.T) {
	ic := &mockIdentityChecker{revoked: map[string]bool{"bad-node": true}}
	gm := NewGovernanceMiddleware(nil, ic)
	handler := gm.ZeroTrustMiddlewareHTTP()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Node-ID", "bad-node")
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestZeroTrustMiddlewareHTTPAllowed(t *testing.T) {
	gm := NewGovernanceMiddleware(nil, nil)
	handler := gm.ZeroTrustMiddlewareHTTP()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Node-ID", "good-node")
	req.Header.Set("X-Agent-ID", "agent-1")
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
