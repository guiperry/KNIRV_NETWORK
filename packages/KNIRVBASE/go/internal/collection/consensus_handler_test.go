package collection

import (
	"context"
	"testing"

	"github.com/knirvcorp/knirvbase/internal/p2pconsensus"
	typ "github.com/knirvcorp/knirvbase/internal/types"
)

func TestCollectionEventHandlerOnOperationReceived(t *testing.T) {
	store := &mockStorage{}
	net := &mockNetwork{}
	coll := NewDistributedCollection("hcol", net, store)
	coll.AttachToNetwork("hnet")

	h := NewCollectionEventHandler(coll)
	op := p2pconsensus.OperationEnvelope{
		Collection:  "hcol",
		DocumentID:  "doc42",
		Data:        mustMarshal(t, typ.CRDTOperation{ID: "op1", Type: typ.OpInsert, Collection: "hcol", DocumentID: "doc42", Data: &typ.DistributedDocument{ID: "doc42", Payload: map[string]interface{}{"id": "doc42", "v": 1}}}),
		VectorClock: map[string]int64{"peerX": 1},
		Timestamp:   1,
		PeerID:      "peerX",
	}
	if err := h.OnOperationReceived(op); err != nil {
		t.Fatalf("OnOperationReceived: %v", err)
	}
	// The operation should have been applied to the local store via the handler.
	if _, err := coll.Find(context.Background(), "doc42"); err != nil {
		t.Fatalf("expected document to be applied locally: %v", err)
	}
}

func TestCollectionEventHandlerOnSyncRequest(t *testing.T) {
	store := &mockStorage{}
	net := &mockNetwork{}
	coll := NewDistributedCollection("scol", net, store)
	coll.AttachToNetwork("snet")

	// Seed an operation into the log.
	coll.operationLog = append(coll.operationLog, typ.CRDTOperation{
		ID: "op1", Type: typ.OpInsert, Collection: "scol", DocumentID: "d1",
		Data: &typ.DistributedDocument{ID: "d1"}, Vector: map[string]int64{"peerA": 5}, PeerID: "peerA",
	})

	h := NewCollectionEventHandler(coll)
	resp, err := h.OnSyncRequestReceived(p2pconsensus.SyncRequest{
		NetworkID:   "snet",
		Collection:  "scol",
		VectorClock: map[string]int64{"peerA": 0}, // missing everything
	})
	if err != nil {
		t.Fatalf("OnSyncRequestReceived: %v", err)
	}
	if len(resp.Operations) != 1 {
		t.Fatalf("expected 1 missing op, got %d", len(resp.Operations))
	}
	if resp.Operations[0].DocumentID != "d1" {
		t.Fatalf("unexpected op returned: %+v", resp.Operations[0])
	}

	// A peer that already has the op should receive nothing.
	resp2, err := h.OnSyncRequestReceived(p2pconsensus.SyncRequest{
		NetworkID:   "snet",
		Collection:  "scol",
		VectorClock: map[string]int64{"peerA": 5},
	})
	if err != nil {
		t.Fatalf("OnSyncRequestReceived(2): %v", err)
	}
	if len(resp2.Operations) != 0 {
		t.Fatalf("expected 0 missing ops, got %d", len(resp2.Operations))
	}
}

func TestCollectionEventHandlerOnPeerDiscoveredTriggersSync(t *testing.T) {
	store := &mockStorage{}
	net := &mockNetwork{}
	coll := NewDistributedCollection("pcol", net, store)
	// Not attached: OnPeerDiscovered should be a no-op and not error.
	h := NewCollectionEventHandler(coll)
	if err := h.OnPeerDiscovered(p2pconsensus.PeerInfo{ID: "peerZ"}); err != nil {
		t.Fatalf("OnPeerDiscovered (detached) should not error: %v", err)
	}

	coll.AttachToNetwork("pnet")
	if err := h.OnPeerDiscovered(p2pconsensus.PeerInfo{ID: "peerZ"}); err != nil {
		t.Fatalf("OnPeerDiscovered (attached) should not error: %v", err)
	}
}

func mustMarshal(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := jsonMarshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}
