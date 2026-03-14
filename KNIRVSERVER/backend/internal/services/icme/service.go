package icme

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"backend_server/internal/database"
)

type Service struct {
	intentRegistry *IntentRegistry
	graphEngine    *TemporalHypergraph
	faissManager   *FAISSIndexManager
	searchEngine   *HybridSearchEngine
	delegation     *DelegationFramework
	alignLoop      *AlignmentLoop
	signalRouter   *SignalRouter
	logger         *zap.Logger
	db             *database.BuntDBManager

	blockchainClient interface {
		SubmitTransaction(tx interface{}) (string, error)
	}
}

type Policy struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Rules       map[string]interface{} `json:"rules"`
	Priority    int                    `json:"priority"`
	Enabled     bool                   `json:"enabled"`
	TargetDVE   string                 `json:"target_dve,omitempty"`
	CreatedAt   string                 `json:"created_at"`
	CommittedAt string                 `json:"committed_at,omitempty"`
	TxHash      string                 `json:"tx_hash,omitempty"`
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
	db *database.BuntDBManager,
) *Service {
	return &Service{
		intentRegistry: intentRegistry,
		graphEngine:    graphEngine,
		faissManager:   faissManager,
		searchEngine:   searchEngine,
		delegation:     delegation,
		alignLoop:      alignmentLoop,
		signalRouter:   signalRouter,
		logger:         logger,
		db:             db,
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
	r.HandleFunc("/api/icme/policy/commit", s.handleCommitPolicy).Methods("POST")
	r.HandleFunc("/api/icme/policy/list", s.handleListPolicies).Methods("GET")
	r.HandleFunc("/api/icme/onboarding/complete", s.handleOnboardingComplete).Methods("POST")
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

	record, err := s.alignLoop.Evaluate(context.Background(), &signal)
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

func (s *Service) handleCommitPolicy(w http.ResponseWriter, r *http.Request) {
	var policy Policy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if policy.Name == "" {
		http.Error(w, "policy name is required", http.StatusBadRequest)
		return
	}

	if s.blockchainClient != nil {
		policyData, err := json.Marshal(policy)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tx := struct {
			Type string `json:"type"`
			Data string `json:"data"`
		}{
			Type: "policy_commit",
			Data: string(policyData),
		}

		txHash, err := s.blockchainClient.SubmitTransaction(tx)
		if err != nil {
			s.logger.Error("failed to commit policy to blockchain",
				zap.String("policy", policy.Name),
				zap.Error(err))
			http.Error(w, "failed to commit policy to blockchain: "+err.Error(), http.StatusInternalServerError)
			return
		}

		policy.TxHash = txHash
		policy.CommittedAt = "committed"
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"policy":  policy,
		"message": "Policy committed successfully",
	})
}

func (s *Service) handleListPolicies(w http.ResponseWriter, r *http.Request) {
	dveID := r.URL.Query().Get("dve_id")
	enabledOnly := r.URL.Query().Get("enabled") == "true"

	var policies []Policy

	err := s.intentRegistry.db.GetObjectsByPrefix("icme:policy:", func(key string, value []byte) bool {
		var policy Policy
		if err := json.Unmarshal(value, &policy); err != nil {
			return true
		}

		if dveID != "" && policy.TargetDVE != dveID {
			return true
		}
		if enabledOnly && !policy.Enabled {
			return true
		}

		policies = append(policies, policy)
		return true
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if policies == nil {
		policies = []Policy{}
	}

	json.NewEncoder(w).Encode(policies)
}

func (s *Service) SetBlockchainClient(client interface {
	SubmitTransaction(tx interface{}) (string, error)
}) {
	s.blockchainClient = client
}

func (s *Service) GetHypergraph() *TemporalHypergraph {
	return s.graphEngine
}

type OnboardingCompleteRequest struct {
	UserID          string                 `json:"user_id"`
	WalletName      string                 `json:"wallet_name"`
	FabricInputs    []string               `json:"fabric_inputs"`
	Guardrails      map[string]bool        `json:"guardrails"`
	ConnectionData  map[string]interface{} `json:"connection_data"`
	PrivacySettings map[string]interface{} `json:"privacy_settings"`
	CommitToChain   bool                   `json:"commit_to_chain"`
}

type OnboardingCompleteResponse struct {
	Success        bool     `json:"success"`
	Message        string   `json:"message"`
	PolicyTxHashes []string `json:"policy_tx_hashes,omitempty"`
	SavedPolicies  []Policy `json:"saved_policies,omitempty"`
}

func (s *Service) handleOnboardingComplete(w http.ResponseWriter, r *http.Request) {
	var req OnboardingCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := OnboardingCompleteResponse{
		Success:        true,
		Message:        "Onboarding completed successfully",
		PolicyTxHashes: []string{},
		SavedPolicies:  []Policy{},
	}

	// Convert connectionData policyCerts and customRules to Policy format
	var policies []Policy
	if connData, ok := req.ConnectionData["policyCerts"].([]interface{}); ok {
		for _, p := range connData {
			if pMap, ok := p.(map[string]interface{}); ok {
				policy := Policy{
					Name:      getString(pMap, "name"),
					Type:      getString(pMap, "category"),
					Enabled:   getBool(pMap, "enabled"),
					TargetDVE: req.UserID,
					CreatedAt: getString(pMap, "created_at"),
				}

				// Convert value to Rules map
				if val, ok := pMap["value"]; ok {
					policy.Rules = map[string]interface{}{
						"value": val,
					}
				}
				if desc, ok := pMap["description"].(string); ok {
					policy.Rules["description"] = desc
				}

				policies = append(policies, policy)
			}
		}
	}

	// Add custom rules as policies
	if customRules, ok := req.ConnectionData["customRules"].([]interface{}); ok {
		for _, r := range customRules {
			if rMap, ok := r.(map[string]interface{}); ok {
				priority := getString(rMap, "priority")
				ruleType := getString(rMap, "ruleType")

				policy := Policy{
					Name: getString(rMap, "name"),
					Type: "custom_rule",
					Rules: map[string]interface{}{
						"description": getString(rMap, "description"),
						"rule_type":   ruleType,
						"priority":    priority,
					},
					Enabled:   true,
					TargetDVE: req.UserID,
				}
				policies = append(policies, policy)
			}
		}
	}

	// Commit policies to blockchain if requested
	if req.CommitToChain && s.blockchainClient != nil && len(policies) > 0 {
		for i, policy := range policies {
			policyData, err := json.Marshal(policy)
			if err != nil {
				s.logger.Error("failed to marshal policy", zap.Error(err))
				continue
			}

			tx := struct {
				Type string `json:"type"`
				Data string `json:"data"`
			}{
				Type: "policy_commit",
				Data: string(policyData),
			}

			txHash, err := s.blockchainClient.SubmitTransaction(tx)
			if err != nil {
				s.logger.Error("failed to commit policy to blockchain",
					zap.String("policy", policy.Name),
					zap.Error(err))
				continue
			}

			policies[i].TxHash = txHash
			policies[i].CommittedAt = "committed"
			response.PolicyTxHashes = append(response.PolicyTxHashes, txHash)
		}
		response.SavedPolicies = policies
	}

	// Save full onboarding data to database (KNIRVBASE)
	if s.db != nil && req.UserID != "" {
		onboardingData := map[string]interface{}{
			"wallet_name":      req.WalletName,
			"fabric_inputs":    req.FabricInputs,
			"guardrails":       req.Guardrails,
			"connection_data":  req.ConnectionData,
			"privacy_settings": req.PrivacySettings,
			"completed_at":     "now",
			"policies":         policies,
		}

		preferencesKey := fmt.Sprintf("users:preferences:%s", req.UserID)
		if err := s.db.StoreJSON(preferencesKey, onboardingData); err != nil {
			s.logger.Error("failed to save onboarding data",
				zap.String("user_id", req.UserID),
				zap.Error(err))
			// Don't fail the request, just log the error
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}
