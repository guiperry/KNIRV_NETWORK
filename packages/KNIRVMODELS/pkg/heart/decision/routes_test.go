package decision

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	sdk "github.com/guiperry/knirv-sdk-go/models"
)

// Routes mounts the HEART decision surface, not the retired HEAD prefix: the
// decision core is served under /models/heart/decide and nothing lives at
// /models/head/... any more.
func TestRoutesMountsHeartDecisionPrefixNotHead(t *testing.T) {
	svc := NewService(publish(t, "v1-deadbeef"))
	defer svc.Close(context.Background())
	mux := http.NewServeMux()
	Routes(svc, mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// The route exists: a GET to the POST-only reason endpoint is 405, not 404.
	if got := status(t, srv.URL+sdk.HeartDecisionPrefix+"/reason"); got != http.StatusMethodNotAllowed {
		t.Fatalf("GET %s/reason = %d, want 405 (route mounted under HeartDecisionPrefix)", sdk.HeartDecisionPrefix, got)
	}
	if got := status(t, srv.URL+sdk.HeartDecisionPrefix+"/models"); got == http.StatusNotFound {
		t.Fatalf("GET %s/models = 404, want the decision manifest route", sdk.HeartDecisionPrefix)
	}

	// The retired HEAD prefix is not mounted.
	if got := status(t, srv.URL+"/models/head/reason"); got != http.StatusNotFound {
		t.Fatalf("GET /models/head/reason = %d, want 404", got)
	}
	if got := status(t, srv.URL+"/models/head/models"); got != http.StatusNotFound {
		t.Fatalf("GET /models/head/models = %d, want 404", got)
	}
}

func status(t *testing.T, url string) int {
	t.Helper()
	res, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer res.Body.Close()
	return res.StatusCode
}
