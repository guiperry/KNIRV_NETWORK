package launcher

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/guiperry/knirv-sdk-go/noderole"
)

func TestReplicationHealthOverUnixSocket(t *testing.T) {
	fixture, err := os.ReadFile("../../../KNIRVSDK/testvectors/replication/replica_status_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "oracle.sock")
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(fixture) })}
	go server.Serve(listener)
	t.Cleanup(func() { _ = server.Close() })
	got := replicationHealth(context.Background(), path)
	if got["available"] != true || got["head_seq"] != uint64(8) || got["lag_seq"] != uint64(2) || got["diverged"] != true {
		t.Fatalf("health=%+v", got)
	}
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	if got := replicationHealth(context.Background(), path); got["available"] != false {
		t.Fatalf("offline health=%+v", got)
	}
}

func TestHealthReportsResolvedRoleAndReplication(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("KNIRV_APP_DATA_DIR", dir)
	if err := os.MkdirAll(filepath.Join(dir, "sockets"), 0700); err != nil {
		t.Fatal(err)
	}
	fixture, err := os.ReadFile("../../../KNIRVSDK/testvectors/replication/replica_status_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("unix", filepath.Join(dir, "sockets", "oracle.sock"))
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(fixture) })}
	go server.Serve(listener)
	t.Cleanup(func() { _ = server.Close() })
	app := &ServerApp{config: &Config{BackendSocket: filepath.Join(dir, "backend.sock"), GatewaySocket: filepath.Join(dir, "gateway.sock")}}
	app.roleResolution.Store(&noderole.Resolution{Kind: noderole.KindBootnode, Oracle: noderole.OracleStandby, Validator: true})
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/health", app.handleHealth)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	var body struct {
		Roles       map[string]any `json:"roles"`
		Replication map[string]any `json:"replication"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Roles["kind"] != "bootnode" || body.Roles["validator"] != true || body.Replication["lag_seq"] != float64(2) {
		t.Fatalf("health=%s", rec.Body.String())
	}
}
