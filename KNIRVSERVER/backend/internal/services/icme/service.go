package icme

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type Service struct {
	intentRegistry *IntentRegistry
	graphEngine    *TemporalHypergraph
	faissManager   *FAISSIndexManager
	searchEngine   *HybridSearchEngine
	delegation     *DelegationFramework
	alignmentLoop  *AlignmentLoop
	signalRouter   *SignalRouter
	logger         *zap.Logger
}

func NewService(
	intentRegistry *IntentRegistry,
	graphEngine *TemporalHypergraph,
	faissManager *FAISSIndexManager,
	searchEngine *HybridSearchEngine,
	delegation *DelegationFramework,
	alignmentLoop *AlignmentLoop,
	signalRouter *SignalRouter,
	logger *zap.Logger,
) *Service {
	return &Service{
		intentRegistry: intentRegistry,
		graphEngine:    graphEngine,
		faissManager:   faissManager,
		searchEngine:   searchEngine,
		delegation:     delegation,
		alignmentLoop:  alignmentLoop,
		signalRouter:   signalRouter,
		logger:         logger,
	}
}

func (s *Service) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/icme/objectives", s.handleListObjectives).Methods("GET")
	r.HandleFunc("/api/icme/objectives", s.handleCreateObjective).Methods("POST")
	r.HandleFunc("/api/icme/objectives/{name}", s.handleGetObjective).Methods("GET")
	r.HandleFunc("/api/icme/objectives/{name}", s.handleUpdateObjective).Methods("PUT")
	r.HandleFunc("/api/icme/agents/{agentID}/bind", s.handleBindAgent).Methods("POST")
	r.HandleFunc("/api/icme/dve/{dveID}/agents/{agentID}/bind", s.handleBindAgentDVE).Methods("POST")
	r.HandleFunc("/api/icme/agents/{agentID}/objective", s.handleGetAgentObjective).Methods("GET")
	r.HandleFunc("/api/icme/search", s.handleSearch).Methods("GET")
	r.HandleFunc("/api/icme/alignment/{agentID}", s.handleGetAlignment).Methods("GET")
	r.HandleFunc("/api/icme/alignment/evaluate", s.handleEvaluateAlignment).Methods("POST")
	r.HandleFunc("/api/icme/graph/snapshot", s.handleGraphSnapshot).Methods("GET")
	r.HandleFunc("/api/icme/graph/neighbors/{nodeID}", s.handleGraphNeighbors).Methods("GET")
	r.HandleFunc("/api/icme/delegate", s.handleDelegate).Methods("POST")
}

func (s *Service) handleListObjectives(w http.ResponseWriter, r *http.Request) {
	dveID := r.URL.Query().Get("dve_id")
	objectives := s.intentRegistry.ListObjectives(dveID)
	json.NewEncoder(w).Encode(objectives)
}

func (s *Service) handleCreateObjective(w http.ResponseWriter, r *http.Request) {
	var obj IntentObjective
	if err := json.NewDecoder(r.Body).Decode(&obj); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.intentRegistry.RegisterObjective(&obj); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(obj)
}

func (s *Service) handleGetObjective(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	objectives := s.intentRegistry.ListObjectives("")
	for _, obj := range objectives {
		if obj.Name == name {
			json.NewEncoder(w).Encode(obj)
			return
		}
	}
	http.Error(w, "objective not found", http.StatusNotFound)
}

func (s *Service) handleUpdateObjective(w http.ResponseWriter, r *http.Request) {
	s.handleCreateObjective(w, r)
}

func (s *Service) handleBindAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["agentID"]

	var req struct {
		ObjectiveName string `json:"objective_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.intentRegistry.BindAgentToObjective(agentID, req.ObjectiveName, ""); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Service) handleBindAgentDVE(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["agentID"]
	dveID := vars["dveID"]

	var req struct {
		ObjectiveName string `json:"objective_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.intentRegistry.BindAgentToObjective(agentID, req.ObjectiveName, dveID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Service) handleGetAgentObjective(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["agentID"]
	dveID := r.URL.Query().Get("dve_id")

	obj := s.intentRegistry.GetObjectiveForAgent(agentID, dveID)
	if obj == nil {
		http.Error(w, "no objective bound", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(obj)
}

func (s *Service) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	agentID := r.URL.Query().Get("agent_id")
	dveID := r.URL.Query().Get("dve_id")
	topK := 10

	results, err := s.searchEngine.Search(context.Background(), query, agentID, dveID, topK)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(results)
}

func (s *Service) handleGetAlignment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["agentID"]

	records := s.intentRegistry.GetRecentAlignmentRecords(agentID, 50)
	json.NewEncoder(w).Encode(records)
}

func (s *Service) handleEvaluateAlignment(w http.ResponseWriter, r *http.Request) {
	var signal IntentionalSignal
	if err := json.NewDecoder(r.Body).Decode(&signal); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	record, err := s.alignmentLoop.Evaluate(context.Background(), &signal)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(record)
}

func (s *Service) handleGraphSnapshot(w http.ResponseWriter, r *http.Request) {
	nodes, edges := s.graphEngine.Snapshot()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"nodes": nodes,
		"edges": edges,
	})
}

func (s *Service) handleGraphNeighbors(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["nodeID"]

	maxHops := 2
	if hops := r.URL.Query().Get("hops"); hops != "" {
		fmt.Sscanf(hops, "%d", &maxHops)
	}

	nodes := s.graphEngine.Neighbors(nodeID, maxHops, "")
	json.NewEncoder(w).Encode(nodes)
}

func (s *Service) handleDelegate(w http.ResponseWriter, r *http.Request) {
	var ctx DecisionContext
	if err := json.NewDecoder(r.Body).Decode(&ctx); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result := s.delegation.Resolve(ctx)
	json.NewEncoder(w).Encode(result)
}

func (s *Service) GetHypergraph() *TemporalHypergraph {
	return s.graphEngine
}
