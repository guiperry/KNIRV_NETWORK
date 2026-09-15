package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"

	"ulora/internal/api"
	"ulora/internal/connector"
)

// deriveConnectorRequest asks the runtime to derive and cache the connector for
// one model.
//
// The weight matrix is part of the request because that is where a connector
// comes from: it is built from the model's own principal subspaces. Omitting it
// is a refusal, not a degraded mode — an invented basis would produce a connector
// that binds and loads while aligning nothing.
type deriveConnectorRequest struct {
	TargetModel  api.BaseModelSpec `json:"target_model"`
	WeightMatrix [][]float64       `json:"weight_matrix"`
	// CanonicalDim overrides K. Left unset it uses the protocol default, which is
	// what compilation uses, so a derived connector matches the cores in bundles.
	CanonicalDim int `json:"canonical_dim,omitempty"`
	Rank         int `json:"rank,omitempty"`
}

type deriveConnectorResponse struct {
	Key          string `json:"key"`
	Family       string `json:"family"`
	ParamCount   string `json:"param_count"`
	CanonicalDim int    `json:"canonical_dim"`
	PInShape     []int  `json:"p_in_shape"`
	POutShape    []int  `json:"p_out_shape"`
	// Digest identifies the projection a subsequent bind used, so a manifest's
	// numbers can be traced to the connector that produced them.
	Digest string `json:"digest"`
}

// connectorCache returns the connector cache this server reads and writes.
//
// Falls back to <data_dir>/connectors when the field is unset: a Config built
// outside config.Load would otherwise have no cache at all, and every bind would
// fail with "no connector available" — which reads like a missing derivation
// rather than a missing directory.
func (s *Server) connectorCache() *connector.Cache {
	dir := s.cfg.ConnectorDir
	if dir == "" {
		dir = filepath.Join(s.cfg.DataDir, "connectors")
	}
	return connector.NewCache(dir)
}

func (s *Server) handleDeriveConnector(w http.ResponseWriter, r *http.Request) {
	var req deriveConnectorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	if len(req.WeightMatrix) == 0 {
		http.Error(w,
			"weight_matrix is required: a connector is derived from the target model's own "+
				"principal subspaces, and one cannot be invented without silently misaligning the adapter",
			http.StatusBadRequest)
		return
	}
	if req.TargetModel.Family == "" {
		http.Error(w, "target_model.family is required: it is the cache key a connector is stored under",
			http.StatusBadRequest)
		return
	}

	conn, err := s.compiler.DeriveConnector(req.TargetModel, req.WeightMatrix, req.CanonicalDim, req.Rank)
	if err != nil {
		// A refusal from the engine (bad shapes, no basis) is the caller's
		// problem; anything else is the server's.
		http.Error(w, fmt.Sprintf("derive connector: %v", err), http.StatusUnprocessableEntity)
		return
	}

	if err := s.connectorCache().Put(conn); err != nil {
		http.Error(w, fmt.Sprintf("cache connector: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(deriveConnectorResponse{
		Key:          connector.Key(conn.Family, conn.ParamCount),
		Family:       conn.Family,
		ParamCount:   conn.ParamCount,
		CanonicalDim: conn.CanonicalDim,
		PInShape:     []int{len(conn.PIn), colsOf(conn.PIn)},
		POutShape:    []int{len(conn.POut), colsOf(conn.POut)},
		Digest:       conn.Digest(),
	})
}

func colsOf(m [][]float32) int {
	if len(m) == 0 {
		return 0
	}
	return len(m[0])
}
