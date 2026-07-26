package collection

import (
	"encoding/json"
	"time"

	"github.com/knirvcorp/knirvbase/internal/p2pconsensus"
	typ "github.com/knirvcorp/knirvbase/internal/types"
)

// CollectionEventHandler adapts a DistributedCollection to the
// p2pconsensus.EventHandler interface. Inbound operations from the gateway proxy
// (or standalone consensus) are applied to local storage, and sync requests are
// answered from the collection's operation log. This mirrors the legacy
// OnMessage wiring in setupMessageHandlers but routes through the consensus
// layer instead of the deprecated TCP network.
type CollectionEventHandler struct {
	collection *DistributedCollection
}

// NewCollectionEventHandler wraps a DistributedCollection as a consensus EventHandler.
func NewCollectionEventHandler(c *DistributedCollection) *CollectionEventHandler {
	return &CollectionEventHandler{collection: c}
}

// OnOperationReceived applies a CRDT operation received from a peer via consensus.
func (h *CollectionEventHandler) OnOperationReceived(op p2pconsensus.OperationEnvelope) error {
	var crdtOp typ.CRDTOperation
	if err := json.Unmarshal(op.Data, &crdtOp); err != nil {
		return err
	}
	// Fall back to the envelope fields if the operation did not carry them.
	if crdtOp.DocumentID == "" {
		crdtOp.DocumentID = op.DocumentID
	}
	if crdtOp.Collection == "" {
		crdtOp.Collection = op.Collection
	}
	if crdtOp.PeerID == "" {
		crdtOp.PeerID = op.PeerID
	}
	if len(crdtOp.Vector) == 0 {
		crdtOp.Vector = op.VectorClock
	}
	if crdtOp.Timestamp == 0 {
		crdtOp.Timestamp = op.Timestamp
	}
	h.collection.handleRemoteOperation(crdtOp)
	return nil
}

// OnSyncRequestReceived answers a peer's sync request from the operation log.
func (h *CollectionEventHandler) OnSyncRequestReceived(req p2pconsensus.SyncRequest) (*p2pconsensus.SyncResponse, error) {
	missingOps := h.collection.computeSyncResponse(req.VectorClock)
	ops := make([]p2pconsensus.OperationEnvelope, 0, len(missingOps))
	for _, op := range missingOps {
		data, err := json.Marshal(op)
		if err != nil {
			return nil, err
		}
		ops = append(ops, p2pconsensus.OperationEnvelope{
			Collection:  op.Collection,
			DocumentID:  op.DocumentID,
			Data:        data,
			VectorClock: op.Vector,
			Timestamp:   op.Timestamp,
			PeerID:      op.PeerID,
		})
	}
	return &p2pconsensus.SyncResponse{
		NetworkID:  req.NetworkID,
		Collection: req.Collection,
		Operations: ops,
	}, nil
}

// OnPeerDiscovered reacts to a newly discovered peer by proactively requesting
// a sync so this node converges quickly. It is a no-op when the collection is
// not attached to a network.
func (h *CollectionEventHandler) OnPeerDiscovered(peer p2pconsensus.PeerInfo) error {
	if h.collection.networkID == "" {
		return nil
	}
	return h.collection.requestSync()
}

// computeSyncResponse returns the operations in the log that the requester is
// missing based on its vector clock. Extracted from handleSyncRequest so it can
// be reused by both the legacy network path and the consensus EventHandler.
func (dc *DistributedCollection) computeSyncResponse(remoteVector map[string]int64) []typ.CRDTOperation {
	missing := make([]typ.CRDTOperation, 0)
	for _, op := range dc.operationLog {
		remoteClock := int64(0)
		if vv, ok := remoteVector[op.PeerID]; ok {
			remoteClock = vv
		}
		opClock := int64(0)
		if vv, ok := op.Vector[op.PeerID]; ok {
			opClock = vv
		}
		if opClock > remoteClock {
			missing = append(missing, op)
		}
	}
	return missing
}

// handleSyncRequest now delegates the core logic to computeSyncResponse and
// sends the response over the legacy network.
func (dc *DistributedCollection) handleSyncRequest(msg typ.ProtocolMessage) {
	payload, _ := msg.Payload.(map[string]interface{})
	remoteVector, _ := payload["vector"].(map[string]interface{})

	rv := make(map[string]int64)
	for k, v := range remoteVector {
		switch val := v.(type) {
		case float64:
			rv[k] = int64(val)
		case int64:
			rv[k] = val
		}
	}

	missing := dc.computeSyncResponse(rv)

	_ = dc.network.SendToPeer(msg.SenderID, dc.networkID, typ.ProtocolMessage{Type: typ.MsgSyncResponse, NetworkID: dc.networkID, SenderID: dc.network.GetPeerID(), Timestamp: time.Now().UnixMilli(), Payload: map[string]interface{}{"collection": dc.Name, "operations": missing, "vector": dc.syncStates[dc.networkID].LocalVector}})
}
