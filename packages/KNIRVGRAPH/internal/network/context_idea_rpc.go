package network

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// Context and Idea node routes (arena_fixes.md §3.6). NRVSystem already
// implements CreateContextNode / CreateIdeaNode; these expose them the same way
// /nrv/errors is exposed: creation is internal-only (KNIRVSERVER registers
// nodes for a signed-in user), reads are public like the rest of /nrv/*.
//
// Promotion (Context → Capability on KNIRVCHAIN's MCP registry, Idea → Property
// via KNIRVCHAIN's PropertyMaker) is a separate step and is not done here.

const maxNodeDescription = 4000

var (
	allowedContextTypes = map[string]bool{"mcp_server": true, "api_endpoint": true, "tool": true}
	allowedIdeaTypes    = map[string]bool{
		"asset": true, "characteristic": true, "attribute": true,
		"innovation": true, "improvement": true, "feature": true,
	}
)

func (rpc *RPCServer) registerContextIdeaRoutes(router *mux.Router) {
	router.HandleFunc("/nrv/contexts", rpc.getAllContexts).Methods("GET", "OPTIONS")
	router.HandleFunc("/nrv/contexts", rpc.createContext).Methods("POST")
	router.HandleFunc("/nrv/contexts/{id}", rpc.getContext).Methods("GET", "OPTIONS")
	router.HandleFunc("/nrv/ideas", rpc.getAllIdeas).Methods("GET", "OPTIONS")
	router.HandleFunc("/nrv/ideas", rpc.createIdea).Methods("POST")
	router.HandleFunc("/nrv/ideas/{id}", rpc.getIdea).Methods("GET", "OPTIONS")
}

func (rpc *RPCServer) requireNRV(w http.ResponseWriter) bool {
	if rpc.nrvSystem == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "the NRV system is not running on this node")
		return false
	}
	return true
}

func (rpc *RPCServer) refuseWhilePaused(w http.ResponseWriter) bool {
	if rpc.app != nil && rpc.app.IsNetworkPaused() {
		writeJSONError(w, http.StatusServiceUnavailable, "network is paused, node creation not allowed")
		return true
	}
	return false
}

func validDescription(d string) bool {
	d = strings.TrimSpace(d)
	return d != "" && len(d) <= maxNodeDescription
}

func (rpc *RPCServer) createContext(w http.ResponseWriter, r *http.Request) {
	if !requireInternalToken(w, r) || !rpc.requireNRV(w) || rpc.refuseWhilePaused(w) {
		return
	}
	var req struct {
		ContextType   string                 `json:"context_type"`
		Description   string                 `json:"description"`
		Schema        map[string]interface{} `json:"schema"`
		LocationHints []string               `json:"location_hints"`
		GasFeeNRN     uint64                 `json:"gas_fee_nrn"`
		SubmittedBy   string                 `json:"submitted_by"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10)).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid context node")
		return
	}
	if !allowedContextTypes[req.ContextType] {
		writeJSONError(w, http.StatusBadRequest, "context_type must be one of mcp_server, api_endpoint, tool")
		return
	}
	if !validDescription(req.Description) {
		writeJSONError(w, http.StatusBadRequest, "description is required (at most 4000 characters)")
		return
	}
	schema := req.Schema
	if schema == nil {
		schema = map[string]interface{}{}
	}
	if req.SubmittedBy != "" {
		schema["submitted_by"] = req.SubmittedBy
	}
	node, err := rpc.nrvSystem.CreateContextNode(req.ContextType, strings.TrimSpace(req.Description), schema, req.LocationHints, req.GasFeeNRN)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if rpc.logger != nil {
		rpc.logger.Info("Context node created", zap.String("id", node.ID), zap.String("type", node.ContextType))
	}
	writeJSON(w, http.StatusCreated, node)
}

func (rpc *RPCServer) createIdea(w http.ResponseWriter, r *http.Request) {
	if !requireInternalToken(w, r) || !rpc.requireNRV(w) || rpc.refuseWhilePaused(w) {
		return
	}
	var req struct {
		IdeaType        string                 `json:"idea_type"`
		Description     string                 `json:"description"`
		FeasibilityData map[string]interface{} `json:"feasibility_data"`
		SubmittedBy     string                 `json:"submitted_by"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10)).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid idea node")
		return
	}
	if !allowedIdeaTypes[req.IdeaType] {
		writeJSONError(w, http.StatusBadRequest, "idea_type must be one of asset, characteristic, attribute, innovation, improvement, feature")
		return
	}
	if !validDescription(req.Description) {
		writeJSONError(w, http.StatusBadRequest, "description is required (at most 4000 characters)")
		return
	}
	feasibility := req.FeasibilityData
	if feasibility == nil {
		feasibility = map[string]interface{}{}
	}
	if req.SubmittedBy != "" {
		feasibility["submitted_by"] = req.SubmittedBy
	}
	node, err := rpc.nrvSystem.CreateIdeaNode(req.IdeaType, strings.TrimSpace(req.Description), feasibility)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if rpc.logger != nil {
		rpc.logger.Info("Idea node created", zap.String("id", node.ID), zap.String("type", node.IdeaType))
	}
	writeJSON(w, http.StatusCreated, node)
}

func (rpc *RPCServer) getAllContexts(w http.ResponseWriter, _ *http.Request) {
	if !rpc.requireNRV(w) {
		return
	}
	writeJSON(w, http.StatusOK, rpc.nrvSystem.GetAllContextNodes())
}

func (rpc *RPCServer) getContext(w http.ResponseWriter, r *http.Request) {
	if !rpc.requireNRV(w) {
		return
	}
	node, ok := rpc.nrvSystem.GetContextNode(mux.Vars(r)["id"])
	if !ok {
		writeJSONError(w, http.StatusNotFound, "context node not found")
		return
	}
	writeJSON(w, http.StatusOK, node)
}

func (rpc *RPCServer) getAllIdeas(w http.ResponseWriter, _ *http.Request) {
	if !rpc.requireNRV(w) {
		return
	}
	writeJSON(w, http.StatusOK, rpc.nrvSystem.GetAllIdeaNodes())
}

func (rpc *RPCServer) getIdea(w http.ResponseWriter, r *http.Request) {
	if !rpc.requireNRV(w) {
		return
	}
	node, ok := rpc.nrvSystem.GetIdeaNode(mux.Vars(r)["id"])
	if !ok {
		writeJSONError(w, http.StatusNotFound, "idea node not found")
		return
	}
	writeJSON(w, http.StatusOK, node)
}
