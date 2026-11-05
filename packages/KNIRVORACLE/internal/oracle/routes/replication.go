package routes

import (
	"net/http"
	"strconv"
	"time"

	"github.com/knirvcorp/knirvoracle/internal/oracle"
)

// replicationAuthHeader carries a base64-encoded signed replication-pull
// envelope (ORACLE_REPLICATION_SPEC.md §3).
const replicationAuthHeader = "X-Knirv-Replica-Auth"

func (r *OracleRoutes) replicationSource(w http.ResponseWriter) (*oracle.ReplicationSource, bool) {
	source := r.oracle.Replication()
	if source == nil {
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "replication source unavailable"})
		return nil, false
	}
	return source, true
}

func (r *OracleRoutes) requirePullAuth(w http.ResponseWriter, req *http.Request) bool {
	source := r.oracle.Replication()
	if source == nil {
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "replication source unavailable"})
		return false
	}
	if err := source.AuthorizePull(req.Header.Get(replicationAuthHeader), time.Now()); err != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return false
	}
	return true
}

func queryUint(req *http.Request, name string, fallback uint64) uint64 {
	raw := req.URL.Query().Get(name)
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return value
}

func queryInt(req *http.Request, name string, fallback int) int {
	raw := req.URL.Query().Get(name)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func (r *OracleRoutes) handleReplicationHead(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	source, ok := r.replicationSource(w)
	if !ok {
		return
	}
	head, exists := source.CurrentHead()
	if !exists {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "no head produced yet"})
		return
	}
	respondJSON(w, http.StatusOK, head)
}

func (r *OracleRoutes) handleReplicationHeads(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	source, ok := r.replicationSource(w)
	if !ok {
		return
	}
	after := queryUint(req, "after_seq", 0)
	heads := source.HeadsSince(after, queryInt(req, "limit", 128))
	respondJSON(w, http.StatusOK, map[string]any{
		"source": sourceRole(r.oracle.Mode()),
		"heads":  heads,
	})
}

func (r *OracleRoutes) handleReplicationLeaves(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	source, ok := r.replicationSource(w)
	if !ok {
		return
	}
	from := queryUint(req, "from", 0)
	respondJSON(w, http.StatusOK, map[string]any{
		"from":   from,
		"leaves": source.Leaves(from, queryInt(req, "limit", 1024)),
	})
}

func (r *OracleRoutes) handleReplicationCheckpoints(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	chain := req.URL.Query().Get("chain")
	if chain == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "chain is required"})
		return
	}
	after := queryUint(req, "after_height", 0)
	limit := queryInt(req, "limit", 256)
	if limit <= 0 || limit > 256 {
		limit = 256
	}
	records := make([]any, 0, limit)
	for _, record := range r.oracle.GetCheckpointRecords(chain) {
		if record.Checkpoint.EndHeight <= after {
			continue
		}
		records = append(records, record)
		if len(records) >= limit {
			break
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{"chain": chain, "records": records})
}

func (r *OracleRoutes) handleReplicationRegistry(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !r.requirePullAuth(w, req) {
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"chains": r.oracle.RegistrySnapshot(),
		"hash":   r.oracle.ReplicationRegistryHash(),
	})
}

func (r *OracleRoutes) handleReplicationState(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !r.requirePullAuth(w, req) {
		return
	}
	source, ok := r.replicationSource(w)
	if !ok {
		return
	}
	from := queryUint(req, "from", 0)
	respondJSON(w, http.StatusOK, map[string]any{
		"from":    from,
		"records": source.StateRecords(from, queryInt(req, "limit", 256)),
	})
}

func (r *OracleRoutes) handleReplicationAux(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !r.requirePullAuth(w, req) {
		return
	}
	source, ok := r.replicationSource(w)
	if !ok {
		return
	}
	respondJSON(w, http.StatusOK, source.AuxSnapshot())
}

func sourceRole(mode string) string {
	if mode == oracle.ModeStandby {
		return "standby"
	}
	return "active"
}
