package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"
)

type stubSessionAuthorizer struct{ err error }

func (s stubSessionAuthorizer) Authorize(_ context.Context, authorization string) error {
	if authorization != "Bearer valid-session" {
		return errInvalidSession
	}
	return s.err
}

func TestEventBundleMintProxyRequiresValidatedSession(t *testing.T) {
	upstreamCalls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalls++
		if r.Header.Get("X-KNIRV-Internal-Token") != "internal-token" {
			t.Fatalf("internal token was not injected")
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer upstream.Close()
	target, _ := url.Parse(upstream.URL)
	proxy := httputil.NewSingleHostReverseProxy(target)
	handler := newEventBundleMintProxy(proxy, "internal-token", stubSessionAuthorizer{})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/event-bundles/mint", strings.NewReader(`{}`))
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted || upstreamCalls != 1 {
		t.Fatalf("valid mint = %d, calls=%d", response.Code, upstreamCalls)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/event-bundles/mint", strings.NewReader(`{}`))
	request.Header.Set("Authorization", "Bearer forged")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized || upstreamCalls != 1 {
		t.Fatalf("forged mint = %d, calls=%d", response.Code, upstreamCalls)
	}
}

func TestEventBundleMintProxyFailsClosedWhenAuthUnavailable(t *testing.T) {
	target, _ := url.Parse("http://unused.invalid")
	proxy := httputil.NewSingleHostReverseProxy(target)
	handler := newEventBundleMintProxy(proxy, "internal-token", stubSessionAuthorizer{err: errors.New("backend down")})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/event-bundles/mint", nil)
	request.Header.Set("Authorization", "Bearer valid-session")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", response.Code)
	}
}
