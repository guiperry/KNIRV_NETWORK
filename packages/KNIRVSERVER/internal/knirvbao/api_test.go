package knirvbao

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestBootstrapUsesNarrowRoles(t *testing.T) {
	var paths []string
	api := &baoAPI{client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		paths = append(paths, req.URL.Path)
		if req.Header.Get("X-Vault-Token") != "test-bootstrap-token" {
			t.Error("missing bootstrap authorization")
		}
		if strings.HasPrefix(req.URL.Path, "/v1/auth/approle/role/") {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Error(err)
			}
			if !strings.Contains(string(body), `"token_no_default_policy":true`) || !strings.Contains(string(body), `"secret_id_num_uses":1`) {
				t.Error("role grants a default policy or reusable secret ID")
			}
		}
		if req.Method == http.MethodGet {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":{}}`)), Header: make(http.Header)}, nil
		}
		return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})}}
	if err := api.bootstrap(context.Background(), "test-bootstrap-token", nil); err != nil {
		t.Fatal(err)
	}
	// 3 mounts (GET+POST each) + 4 policies (PUT) + 4 roles (POST) = 14 requests.
	if len(paths) != 14 || !strings.HasSuffix(paths[len(paths)-1], "/auth/approle/role/backend") {
		t.Fatalf("unexpected bootstrap request count or final operation: %d %s", len(paths), paths[len(paths)-1])
	}
	// OB-R7: the persisted admin token is intentionally NOT revoked here; a later
	// owner-approved recovery/control identity is required before any revocation.
	if strings.Contains(strings.Join(paths, " "), "revoke-self") {
		t.Fatal("bootstrap must not revoke the persisted admin token (OB-R7)")
	}
	for _, policy := range bootstrapPolicies {
		if strings.Contains(policy.Policy, "sudo") || strings.Contains(policy.Policy, "delete") || strings.Contains(policy.Policy, "root-owner") && policy.Name != "oracle" {
			t.Fatalf("policy %s exceeds its intended permissions", policy.Name)
		}
	}
}

func TestRequestDoesNotExposeErrorBody(t *testing.T) {
	api := &baoAPI{client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusForbidden, Body: io.NopCloser(strings.NewReader("sensitive-test-value")), Header: make(http.Header)}, nil
	})}}
	_, err := api.request(context.Background(), http.MethodGet, "sys/example", "", nil, nil)
	if err == nil || strings.Contains(err.Error(), "sensitive-test-value") {
		t.Fatal("error response body was exposed")
	}
}
