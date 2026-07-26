package p2pconsensus

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGatewayProxySignsPublishWithSecret(t *testing.T) {
	var gotSig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/knirvbase/register-callback":
			_ = r.ParseForm()
		case "/knirvbase/publish-op":
			gotSig = r.Header.Get("X-KNIRV-Signature")
			body := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(body)
			// Server-side recompute using the shared secret.
			if !VerifyMessage("net", "hush", gotSig, body) {
				t.Errorf("server-side recomputed signature mismatch")
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	g := NewGatewayProxy(srv.URL, "net", "/tmp/knirv.sock", "hush")
	if err := g.Register(); err != nil {
		t.Fatalf("register: %v", err)
	}
	op := OperationEnvelope{Collection: "c", DocumentID: "d", Data: []byte(`{"x":1}`)}
	if err := g.PublishOperation(context.Background(), op); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if gotSig == "" {
		t.Fatal("expected X-KNIRV-Signature header on publish")
	}
}

func TestGatewayProxyPublishOpenNetworkNoSignature(t *testing.T) {
	var gotSig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSig = r.Header.Get("X-KNIRV-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	g := NewGatewayProxy(srv.URL, "net", "/tmp/knirv.sock")
	op := OperationEnvelope{Collection: "c", DocumentID: "d"}
	if err := g.PublishOperation(context.Background(), op); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if gotSig != "" {
		t.Fatalf("open network should not sign, got %q", gotSig)
	}
}
