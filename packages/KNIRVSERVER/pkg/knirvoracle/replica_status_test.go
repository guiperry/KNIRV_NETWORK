package knirvoracle

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
)

func TestReplicaStatusRequiresCompleteResponse(t *testing.T) {
	fixture, err := os.ReadFile("../../../KNIRVSDK/testvectors/replication/replica_status_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var body atomic.Value
	body.Store(string(fixture))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body.Load().(string))) }))
	defer server.Close()
	client := NewHTTPClient(server.URL)
	status, err := client.ReplicaStatus(context.Background())
	if err != nil || status.HeadSeq != 8 || status.LagSeq != 2 || status.Paused || !status.Diverged || len(status.UnverifiedClasses) != 1 {
		t.Fatalf("status=%+v error=%v", status, err)
	}
	body.Store(`{"mode":"standby"}`)
	if _, err := client.ReplicaStatus(context.Background()); err == nil {
		t.Fatal("incomplete status accepted")
	}
}
