package launcher

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"knirv-server/internal/enroll"
)

func TestBackendEnrollmentAdmitterPostsOverUnixSocket(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "backend.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	var got struct {
		path  string
		token string
		body  map[string]string
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.path = r.URL.Path
		got.token = r.Header.Get("X-KNIRV-Internal-Token")
		_ = json.NewDecoder(r.Body).Decode(&got.body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"version":2,"changed":true}`))
	})}
	go server.Serve(listener)
	defer func() {
		_ = server.Close()
		_ = listener.Close()
	}()

	admitter := newBackendEnrollmentAdmitter(socketPath, "internal-token")
	err = admitter.Admit(context.Background(), enroll.ValidatorAdmission{
		ChainID:        "knirvoracle-1",
		RegistrationID: "reg-1",
		ValidatorID:    "knirv1aaa",
		PublicKey:      "02aabb",
	})
	if err != nil {
		t.Fatalf("admit: %v", err)
	}
	if got.path != "/internal/v1/knirv/enroll/register" {
		t.Fatalf("path = %q", got.path)
	}
	if got.token != "internal-token" {
		t.Fatalf("token = %q", got.token)
	}
	want := map[string]string{
		"chain_id":        "knirvoracle-1",
		"registration_id": "reg-1",
		"validator_id":    "knirv1aaa",
		"public_key":      "02aabb",
	}
	for key, value := range want {
		if got.body[key] != value {
			t.Fatalf("body[%s] = %q, want %q", key, got.body[key], value)
		}
	}
}

func TestBackendEnrollmentAdmitterReportsBackendRejection(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "backend.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":"validator set changed concurrently; retry"}`))
	})}
	go server.Serve(listener)
	defer func() {
		_ = server.Close()
		_ = listener.Close()
	}()

	admitter := newBackendEnrollmentAdmitter(socketPath, "internal-token")
	err = admitter.Admit(context.Background(), enroll.ValidatorAdmission{ChainID: "knirvoracle-1"})
	if err == nil {
		t.Fatal("expected backend rejection")
	}
	if !strings.Contains(err.Error(), "409") || !strings.Contains(err.Error(), "concurrently") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBackendEnrollmentAdmitterRequiresSocketAndToken(t *testing.T) {
	if err := newBackendEnrollmentAdmitter("", "token").Admit(context.Background(), enroll.ValidatorAdmission{}); err == nil {
		t.Fatal("missing socket must fail")
	}
	if err := newBackendEnrollmentAdmitter("/tmp/unused.sock", "").Admit(context.Background(), enroll.ValidatorAdmission{}); err == nil {
		t.Fatal("missing token must fail")
	}
}

func TestBackendEnrollmentAdmitterHonorsContextTimeout(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "backend.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	release := make(chan struct{})
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-release
		_, _ = w.Write([]byte(`{"ok":true}`))
	})}
	go server.Serve(listener)
	defer func() {
		close(release)
		_ = server.Close()
		_ = listener.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := newBackendEnrollmentAdmitter(socketPath, "token").Admit(ctx, enroll.ValidatorAdmission{}); err == nil {
		t.Fatal("expected timeout")
	}
}
