package monitor

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"knirv-server/internal/monitor/api"
)

func TestReplicaMetricsFromOracleSocket(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oracle.sock")
	fixture, err := os.ReadFile("../../../KNIRVSDK/testvectors/replication/replica_status_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oracle/v3/status" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write(fixture)
	})}
	go server.Serve(listener)
	t.Cleanup(func() { _ = server.Close() })
	s := &Service{server: api.NewServer(&api.ServerConfig{}), oracle: &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", path)
	}}}}
	s.refreshReplica(context.Background())
	families, err := s.server.Registry().Registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]float64{"oracle_replica_head_seq": 8, "oracle_replica_lag_seq": 2, "oracle_replica_lag_seconds": 4, "oracle_replica_diverged": 1, "oracle_replica_paused": 0, "oracle_replica_available": 1, "oracle_mode": 2, "oracle_replica_unverified_classes": 1}
	for _, family := range families {
		if expected, ok := want[family.GetName()]; ok {
			if got := family.Metric[0].Gauge.GetValue(); got != expected {
				t.Fatalf("%s=%v, want %v", family.GetName(), got, expected)
			}
			delete(want, family.GetName())
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing metrics: %v", want)
	}
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	s.refreshReplica(context.Background())
	families, err = s.server.Registry().Registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() == "oracle_replica_available" && family.Metric[0].Gauge.GetValue() != 0 {
			t.Fatal("failed scrape stayed available")
		}
	}
}
