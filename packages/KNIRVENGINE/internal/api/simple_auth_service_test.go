package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestSimpleAuthLoginReturnsToken(t *testing.T) {
	service := NewSimpleAuthService()
	router := mux.NewRouter()
	service.RegisterHandlers(router)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"admin","password":"admin"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("login returned %d: %s", response.Code, response.Body.String())
	}
	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected JSON response, got %q", response.Header().Get("Content-Type"))
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"token"`)) {
		t.Fatalf("login response did not include a token: %s", response.Body.String())
	}
}
