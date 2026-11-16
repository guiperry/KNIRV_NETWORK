package locator

import (
	"fmt"
	"log"

	"google.golang.org/protobuf/encoding/protojson"
)

// ErrorNodeKV is the persistence error nodes need: the suite store's KV plus a
// prefix scan to reload them. KNIRVGRAPH's storage.Storage satisfies it.
type ErrorNodeKV interface {
	KV
	ScanPrefix(prefix []byte) (map[string][]byte, error)
}

// errorNodeKeyPrefix is where error nodes are written.
const errorNodeKeyPrefix = "graph:error_nodes:"

// legacyErrorNodeKeyPrefix is the pre-rename prefix. It is read as a fallback
// only, so rows written before `migrate-announcement-ref` still load; nothing
// writes to it. Rows move to errorNodeKeyPrefix when the migration runs.
const legacyErrorNodeKeyPrefix = "nrv:error_nodes:"

// SetErrorNodeStore makes error nodes durable: every node created afterwards is
// written to kv, and the nodes already in kv are loaded now, so error nodes
// survive a restart alongside the test suites keyed on their ids. Without a
// store error nodes are memory-only. A node under the current prefix wins over
// a legacy row with the same id.
func (nrv *LocatorSystem) SetErrorNodeStore(kv ErrorNodeKV) error {
	current, err := kv.ScanPrefix([]byte(errorNodeKeyPrefix))
	if err != nil {
		return fmt.Errorf("load persisted error nodes: %w", err)
	}
	legacy, err := kv.ScanPrefix([]byte(legacyErrorNodeKeyPrefix))
	if err != nil {
		return fmt.Errorf("load legacy error nodes: %w", err)
	}
	loaded := make([]*ErrorNode, 0, len(current)+len(legacy))
	seen := make(map[string]bool, len(current)+len(legacy))
	load := func(persisted map[string][]byte) {
		for key, raw := range persisted {
			node := &ErrorNode{}
			if err := unmarshalPersistedErrorNode(raw, node); err != nil || node.Id == "" {
				log.Printf("Warning: skipping unreadable persisted error node %s: %v", key, err)
				continue
			}
			if seen[node.Id] {
				continue
			}
			seen[node.Id] = true
			loaded = append(loaded, node)
		}
	}
	load(current)
	load(legacy)

	nrv.errorsMutex.Lock()
	nrv.errorStore = kv
	for _, node := range loaded {
		nrv.errorNodes[node.Id] = node
	}
	nrv.errorsMutex.Unlock()

	// Restore each node's resolution vector, which is derived, not stored.
	for _, node := range loaded {
		metadata := map[string]interface{}{"node_type": "error", "error_id": node.Id, "severity": int(node.Severity)}
		if _, err := nrv.CreateVector(node.Id, nrv.calculateErrorCoordinates(node), metadata); err != nil {
			log.Printf("Warning: Failed to restore NRV for error node %s: %v", node.Id, err)
		}
	}
	if len(loaded) > 0 {
		log.Printf("Loaded %d persisted error nodes", len(loaded))
	}
	return nil
}

// persistErrorNodeLocked writes node to the error store, if one is set.
// Callers hold errorsMutex.
func (nrv *LocatorSystem) persistErrorNodeLocked(node *ErrorNode) error {
	if nrv.errorStore == nil {
		return nil
	}
	raw, err := protojson.Marshal(node)
	if err != nil {
		return err
	}
	return nrv.errorStore.Put([]byte(errorNodeKeyPrefix+node.Id), raw)
}
