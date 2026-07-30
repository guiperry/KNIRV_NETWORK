package p2pconsensus

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/multiformats/go-multiaddr"
)

type recordingHandler struct {
	mu  sync.Mutex
	ops []OperationEnvelope
}

func (h *recordingHandler) OnOperationReceived(op OperationEnvelope) error {
	h.mu.Lock()
	h.ops = append(h.ops, op)
	h.mu.Unlock()
	return nil
}
func (h *recordingHandler) OnSyncRequestReceived(req SyncRequest) (*SyncResponse, error) {
	return &SyncResponse{NetworkID: req.NetworkID, Collection: req.Collection}, nil
}
func (h *recordingHandler) OnPeerDiscovered(PeerInfo) error { return nil }

func TestStandaloneConsensusBroadcastsNetworkScopedOperations(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h1 := &recordingHandler{}
	h2 := &recordingHandler{}
	cfg := ConsensusConfig{Enabled: true, NetworkID: "standalone-test", Mode: "standalone", ListenAddr: "127.0.0.1", Port: 0}
	s1 := NewStandaloneConsensus(cfg, h1)
	if err := s1.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer s1.Stop()
	if len(s1.host.Addrs()) == 0 {
		t.Fatal("standalone host did not listen")
	}
	addr, err := multiaddr.NewMultiaddr(fmt.Sprintf("%s/p2p/%s", s1.host.Addrs()[0], s1.host.ID()))
	if err != nil {
		t.Fatal(err)
	}
	cfg.BootstrapPeers = []string{addr.String()}
	s2 := NewStandaloneConsensus(cfg, h2)
	if err := s2.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer s2.Stop()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && (len(s2.host.Network().Peers()) == 0 || len(s1.pubsub.ListPeers(Topic(cfg.NetworkID, cfg.NetworkSecret))) == 0 || len(s2.pubsub.ListPeers(Topic(cfg.NetworkID, cfg.NetworkSecret))) == 0) {
		time.Sleep(50 * time.Millisecond)
	}
	if len(s2.host.Network().Peers()) == 0 {
		t.Fatal("second standalone peer did not connect")
	}
	time.Sleep(1 * time.Second)
	if err := s1.BroadcastOperation(ctx, OperationEnvelope{NetworkID: cfg.NetworkID, Collection: "c", DocumentID: "d", Data: []byte(`{"ok":true}`)}); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		h2.mu.Lock()
		n := len(h2.ops)
		h2.mu.Unlock()
		if n > 0 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("operation was not delivered to the second peer")
}
