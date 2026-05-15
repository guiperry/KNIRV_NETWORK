package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestHandleStatus_Available(t *testing.T) {
	logger := zap.NewNop()
	h := NewHasherHandlers(logger, func() bool { return true }, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/hasher/status", nil)
	rec := httptest.NewRecorder()

	h.HandleStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body["available"] != true {
		t.Errorf("expected available true, got %v", body["available"])
	}

	if body["status"] != "available" {
		t.Errorf("expected status 'available', got %v", body["status"])
	}
}

func TestHandleStatus_Unavailable(t *testing.T) {
	logger := zap.NewNop()
	h := NewHasherHandlers(logger, func() bool { return false }, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/hasher/status", nil)
	rec := httptest.NewRecorder()

	h.HandleStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body["available"] != false {
		t.Errorf("expected available false, got %v", body["available"])
	}

	if body["status"] != "unavailable" {
		t.Errorf("expected status 'unavailable', got %v", body["status"])
	}
}

func TestHandleStatus_NilCallback_DefaultsToUnavailable(t *testing.T) {
	logger := zap.NewNop()
	h := NewHasherHandlers(logger, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/hasher/status", nil)
	rec := httptest.NewRecorder()

	h.HandleStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body["available"] != false {
		t.Errorf("expected available false, got %v", body["available"])
	}

	if body["status"] != "unavailable" {
		t.Errorf("expected status 'unavailable', got %v", body["status"])
	}
}

func TestHandlePing_Success(t *testing.T) {
	logger := zap.NewNop()
	h := NewHasherHandlers(logger, nil, func() error { return nil }, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/hasher/ping", nil)
	rec := httptest.NewRecorder()

	h.HandlePing(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", body["status"])
	}

	if body["message"] != "hasher reachable" {
		t.Errorf("expected message 'hasher reachable', got %v", body["message"])
	}
}

func TestHandlePing_Failure(t *testing.T) {
	logger := zap.NewNop()
	h := NewHasherHandlers(logger, nil, func() error { return errors.New("connection refused") }, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/hasher/ping", nil)
	rec := httptest.NewRecorder()

	h.HandlePing(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rec.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body["status"] != "error" {
		t.Errorf("expected status 'error', got %v", body["status"])
	}

	if body["message"] != "connection refused" {
		t.Errorf("expected message 'connection refused', got %v", body["message"])
	}
}

func TestHandlePing_NilCallback_Returns503(t *testing.T) {
	logger := zap.NewNop()
	h := NewHasherHandlers(logger, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/hasher/ping", nil)
	rec := httptest.NewRecorder()

	h.HandlePing(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rec.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body["status"] != "error" {
		t.Errorf("expected status 'error', got %v", body["status"])
	}

	if body["message"] != "hasher ping not configured" {
		t.Errorf("expected message 'hasher ping not configured', got %v", body["message"])
	}
}
