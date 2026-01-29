// Package embedded_knirvchain provides HTTP endpoints for the embedded KNIRVCHAIN
package embedded_knirvchain

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// EndpointHandler handles HTTP endpoints for embedded KNIRVCHAIN
type EndpointHandler struct {
	embeddedChain *EmbeddedKNIRVChain
}

// NewEndpointHandler creates a new endpoint handler
func NewEndpointHandler(embeddedChain *EmbeddedKNIRVChain) *EndpointHandler {
	return &EndpointHandler{
		embeddedChain: embeddedChain,
	}
}

// RegisterRoutes registers all HTTP routes for embedded KNIRVCHAIN
func (eh *EndpointHandler) RegisterRoutes(router *mux.Router) {
	// Revolutionary /invoke endpoint - replaces traditional /generate
	router.HandleFunc("/invoke", eh.handleInvokeSkill).Methods("POST")

	// Revolutionary /invoke endpoint with protobuf response
	router.HandleFunc("/invoke/protobuf", eh.handleInvokeSkillProtobuf).Methods("POST")

	// Skill management endpoints
	router.HandleFunc("/skills", eh.handleGetSkills).Methods("GET")
	router.HandleFunc("/skills", eh.handleRegisterSkill).Methods("POST")
	router.HandleFunc("/skills/{skillId}", eh.handleGetSkill).Methods("GET")

	// Skill chain endpoints
	router.HandleFunc("/chains", eh.handleGetSkillChains).Methods("GET")
	router.HandleFunc("/chains", eh.handleCreateSkillChain).Methods("POST")
	router.HandleFunc("/chains/{chainId}", eh.handleGetSkillChain).Methods("GET")

	// LoRA adapter filtering endpoints
	router.HandleFunc("/skills/filter", eh.handleFilterSkills).Methods("POST")

	// Health and status endpoints
	router.HandleFunc("/health", eh.handleHealth).Methods("GET")
	router.HandleFunc("/status", eh.handleStatus).Methods("GET")

	// Protobuf serialization endpoints
	router.HandleFunc("/protobuf/serialize", eh.handleSerializeResponse).Methods("POST")

	log.Println("Embedded KNIRVCHAIN endpoints registered")
}

// handleInvokeSkill handles the revolutionary /invoke endpoint
func (eh *EndpointHandler) handleInvokeSkill(w http.ResponseWriter, r *http.Request) {
	var request SkillInvocationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Generate invocation ID if not provided
	if request.InvocationID == "" {
		request.InvocationID = uuid.New().String()
	}

	// Set timestamp if not provided
	if request.Timestamp == 0 {
		request.Timestamp = time.Now().Unix()
	}

	// Set default priority if not provided
	if request.Priority == "" {
		request.Priority = "normal"
	}

	log.Printf("Revolutionary skill invocation: %s (agent: %s, URI: %s)", request.InvocationID, request.AgentID, request.SkillURI)

	response, err := eh.embeddedChain.InvokeSkill(&request)
	if err != nil {
		http.Error(w, fmt.Sprintf("Skill invocation failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleInvokeSkillProtobuf handles the revolutionary /invoke/protobuf endpoint
func (eh *EndpointHandler) handleInvokeSkillProtobuf(w http.ResponseWriter, r *http.Request) {
	var request SkillInvocationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Generate invocation ID if not provided
	if request.InvocationID == "" {
		request.InvocationID = uuid.New().String()
	}

	// Set timestamp if not provided
	if request.Timestamp == 0 {
		request.Timestamp = time.Now().Unix()
	}

	// Set default priority if not provided
	if request.Priority == "" {
		request.Priority = "normal"
	}

	log.Printf("Revolutionary skill invocation (protobuf): %s (agent: %s, URI: %s)", request.InvocationID, request.AgentID, request.SkillURI)

	response, err := eh.embeddedChain.InvokeSkill(&request)
	if err != nil {
		http.Error(w, fmt.Sprintf("Skill invocation failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Serialize response as protobuf
	protobufData, err := eh.embeddedChain.SerializeLoRAAdapterResponse(response)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to serialize response: %v", err), http.StatusInternalServerError)
		return
	}

	// Return protobuf data
	w.Header().Set("Content-Type", "application/x-protobuf")
	w.Header().Set("X-KNIRV-Response-Format", "lora-adapter-protobuf-v1")
	w.Header().Set("X-KNIRV-Invocation-ID", response.InvocationID)
	w.Header().Set("X-KNIRV-Status", response.Status)

	if _, err := w.Write(protobufData); err != nil {
		log.Printf("Failed to write protobuf response: %v", err)
	}
}

// handleGetSkills handles GET /skills with optional filtering
func (eh *EndpointHandler) handleGetSkills(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	var filter *LoRAAdapterFilter
	if len(query) > 0 {
		filter = &LoRAAdapterFilter{}

		if baseModel := query.Get("base_model"); baseModel != "" {
			filter.BaseModel = &baseModel
		}

		if skillType := query.Get("skill_type"); skillType != "" {
			filter.SkillType = &skillType
		}

		if minConsensusStr := query.Get("min_consensus_score"); minConsensusStr != "" {
			if minConsensus, err := strconv.ParseFloat(minConsensusStr, 64); err == nil {
				filter.MinConsensusScore = &minConsensus
			}
		}

		if maxRankStr := query.Get("max_rank"); maxRankStr != "" {
			if maxRank, err := strconv.Atoi(maxRankStr); err == nil {
				filter.MaxRank = &maxRank
			}
		}

		if capabilities := query["capabilities"]; len(capabilities) > 0 {
			filter.Capabilities = capabilities
		}

		if excludeSkills := query["exclude_skills"]; len(excludeSkills) > 0 {
			filter.ExcludeSkills = excludeSkills
		}
	}

	skills, err := eh.embeddedChain.GetSkills(filter)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get skills: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"skills": skills,
		"count":  len(skills),
	}); err != nil {
		log.Printf("Failed to encode skills response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleRegisterSkill handles POST /skills
func (eh *EndpointHandler) handleRegisterSkill(w http.ResponseWriter, r *http.Request) {
	var skill LoRAAdapterSkill
	if err := json.NewDecoder(r.Body).Decode(&skill); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Generate skill ID if not provided
	if skill.SkillID == "" {
		skill.SkillID = uuid.New().String()
	}

	log.Printf("Registering skill: %s (%s)", skill.SkillName, skill.SkillID)

	if err := eh.embeddedChain.RegisterSkill(&skill); err != nil {
		http.Error(w, fmt.Sprintf("Failed to register skill: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"skill_id": skill.SkillID,
		"message":  "Skill registered successfully",
	}); err != nil {
		log.Printf("Failed to encode register response: %v", err)
	}
}

// handleGetSkill handles GET /skills/{skillId}
func (eh *EndpointHandler) handleGetSkill(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	skillID := vars["skillId"]

	skills, err := eh.embeddedChain.GetSkills(nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get skills: %v", err), http.StatusInternalServerError)
		return
	}

	for _, skill := range skills {
		if skill.SkillID == skillID {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(skill); err != nil {
				log.Printf("Failed to encode skill response: %v", err)
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			}
			return
		}
	}

	http.Error(w, "Skill not found", http.StatusNotFound)
}

// handleCreateSkillChain handles POST /chains
func (eh *EndpointHandler) handleCreateSkillChain(w http.ResponseWriter, r *http.Request) {
	var request struct {
		SkillIDs []string `json:"skill_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Get skills by IDs
	var skills []*LoRAAdapterSkill
	allSkills, err := eh.embeddedChain.GetSkills(nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get skills: %v", err), http.StatusInternalServerError)
		return
	}

	for _, skillID := range request.SkillIDs {
		for _, skill := range allSkills {
			if skill.SkillID == skillID {
				skills = append(skills, skill)
				break
			}
		}
	}

	if len(skills) == 0 {
		http.Error(w, "No valid skills found", http.StatusBadRequest)
		return
	}

	log.Printf("Creating skill chain with %d skills", len(skills))

	chain, err := eh.embeddedChain.CreateSkillChain(skills)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create skill chain: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(chain); err != nil {
		log.Printf("Failed to encode chain response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleGetSkillChains handles GET /chains
func (eh *EndpointHandler) handleGetSkillChains(w http.ResponseWriter, r *http.Request) {
	chains, err := eh.embeddedChain.GetSkillChains()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get skill chains: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"chains": chains,
		"count":  len(chains),
	}); err != nil {
		log.Printf("Failed to encode chains response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleGetSkillChain handles GET /chains/{chainId}
func (eh *EndpointHandler) handleGetSkillChain(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chainID := vars["chainId"]

	chains, err := eh.embeddedChain.GetSkillChains()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get skill chains: %v", err), http.StatusInternalServerError)
		return
	}

	for _, chain := range chains {
		if chain.ChainID == chainID {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(chain); err != nil {
				log.Printf("Failed to encode chain response: %v", err)
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			}
			return
		}
	}

	http.Error(w, "Skill chain not found", http.StatusNotFound)
}

// handleFilterSkills handles POST /skills/filter
func (eh *EndpointHandler) handleFilterSkills(w http.ResponseWriter, r *http.Request) {
	var filter LoRAAdapterFilter
	if err := json.NewDecoder(r.Body).Decode(&filter); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	skills, err := eh.embeddedChain.GetSkills(&filter)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to filter skills: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"skills": skills,
		"count":  len(skills),
		"filter": filter,
	}); err != nil {
		log.Printf("Failed to encode filter response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleHealth handles GET /health
func (eh *EndpointHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "embedded-knirvchain",
	}); err != nil {
		log.Printf("Failed to encode health response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleStatus handles GET /status
func (eh *EndpointHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
	skills, _ := eh.embeddedChain.GetSkills(nil)
	chains, _ := eh.embeddedChain.GetSkillChains()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"initialized":  eh.embeddedChain.isInitialized,
		"skill_count":  len(skills),
		"chain_count":  len(chains),
		"memory_usage": eh.embeddedChain.calculateMemoryUsage(),
		"timestamp":    time.Now().Unix(),
	}); err != nil {
		log.Printf("Failed to encode status response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleSerializeResponse handles POST /protobuf/serialize
func (eh *EndpointHandler) handleSerializeResponse(w http.ResponseWriter, r *http.Request) {
	var response SkillInvocationResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// In a real implementation, this would use actual protobuf serialization
	// For now, we'll return the JSON as bytes
	serialized, err := json.Marshal(response)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to serialize response: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(serialized)
}
