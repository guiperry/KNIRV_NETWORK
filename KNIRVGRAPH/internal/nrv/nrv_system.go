package nrv

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sync"
	"time"
)

// NRVSystem manages Network Resolution Vectors
type NRVSystem struct {
	localPeerID   string
	vectors       map[string]*NetworkResolutionVector
	errorNodes    map[string]*ErrorNode
	skillNodes    map[string]*SkillNode
	vectorsMutex  sync.RWMutex
	errorsMutex   sync.RWMutex
	skillsMutex   sync.RWMutex
	updateChannel chan VectorUpdate
	config        *NRVConfig
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewNRVSystem creates a new NRV system instance
func NewNRVSystem(peerID string, config *NRVConfig) *NRVSystem {
	if config == nil {
		config = DefaultNRVConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &NRVSystem{
		localPeerID:   peerID,
		vectors:       make(map[string]*NetworkResolutionVector),
		errorNodes:    make(map[string]*ErrorNode),
		skillNodes:    make(map[string]*SkillNode),
		updateChannel: make(chan VectorUpdate, 100),
		config:        config,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start initializes and starts the NRV system
func (nrv *NRVSystem) Start() error {
	log.Println("Starting NRV System...")

	go nrv.processVectorUpdates()
	go nrv.periodicVectorMaintenance()

	return nil
}

// Stop shuts down the NRV system
func (nrv *NRVSystem) Stop() error {
	log.Println("Stopping NRV System...")
	nrv.cancel()
	close(nrv.updateChannel)
	return nil
}

// CreateVector creates a new network resolution vector
func (nrv *NRVSystem) CreateVector(targetHash string, coordinates []float64, metadata map[string]interface{}) (*NetworkResolutionVector, error) {
	vectorID := nrv.generateVectorID(targetHash, coordinates)

	vector := &NetworkResolutionVector{
		ID:          vectorID,
		SourcePeer:  nrv.localPeerID,
		TargetHash:  targetHash,
		Coordinates: coordinates,
		Confidence:  1.0, // Initial confidence
		Timestamp:   time.Now(),
		Metadata:    metadata,
		Signatures:  []VectorSignature{},
	}

	// Sign the vector
	signature, err := nrv.signVector(vector)
	if err != nil {
		return nil, fmt.Errorf("failed to sign vector: %w", err)
	}

	vector.Signatures = append(vector.Signatures, VectorSignature{
		PeerID:    nrv.localPeerID,
		Signature: signature,
		Timestamp: time.Now(),
	})

	// Store locally
	nrv.vectorsMutex.Lock()
	nrv.vectors[vectorID] = vector
	nrv.vectorsMutex.Unlock()

	// Notify update channel
	select {
	case nrv.updateChannel <- VectorUpdate{
		Vector:    vector,
		Operation: "create",
	}:
	default:
		log.Printf("Warning: Update channel full, dropping vector update")
	}

	return vector, nil
}

// ResolveTarget finds vectors for a given target hash
func (nrv *NRVSystem) ResolveTarget(targetHash string) ([]*NetworkResolutionVector, error) {
	// Check local vectors
	var localVectors []*NetworkResolutionVector
	nrv.vectorsMutex.RLock()
	for _, vector := range nrv.vectors {
		if vector.TargetHash == targetHash {
			localVectors = append(localVectors, vector)
		}
	}
	nrv.vectorsMutex.RUnlock()

	// Sort by confidence
	nrv.sortVectorsByConfidence(localVectors)

	return localVectors, nil
}

// CreateErrorNode creates a new error node
func (nrv *NRVSystem) CreateErrorNode(errorType, description string, context map[string]interface{}, severity int) (*ErrorNode, error) {
	errorID := nrv.generateErrorID(errorType, description)

	errorNode := &ErrorNode{
		ID:          errorID,
		ErrorType:   errorType,
		Description: description,
		Context:     context,
		Severity:    severity,
		Timestamp:   time.Now(),
	}

	// Attempt to find resolution path
	resolutionPath, err := nrv.findResolutionPath(errorNode)
	if err != nil {
		log.Printf("Warning: Could not find resolution path for error %s: %v", errorID, err)
	} else {
		errorNode.Resolution = resolutionPath
	}

	// Store error node
	nrv.errorsMutex.Lock()
	nrv.errorNodes[errorID] = errorNode
	nrv.errorsMutex.Unlock()

	// Create NRV for error resolution
	coordinates := nrv.calculateErrorCoordinates(errorNode)
	metadata := map[string]interface{}{
		"node_type": "error",
		"error_id":  errorID,
		"severity":  severity,
	}

	_, err = nrv.CreateVector(errorID, coordinates, metadata)
	if err != nil {
		log.Printf("Warning: Failed to create NRV for error node: %v", err)
	}

	return errorNode, nil
}

// CreateSkillNode creates a new skill node
func (nrv *NRVSystem) CreateSkillNode(skillType string, capabilities []string, requirements map[string]interface{}) (*SkillNode, error) {
	skillID := nrv.generateSkillID(skillType, capabilities)

	skillNode := &SkillNode{
		ID:           skillID,
		SkillType:    skillType,
		Capabilities: capabilities,
		Requirements: requirements,
		Performance: &PerformanceMetrics{
			SuccessRate:      0.0,
			AverageLatency:   0.0,
			TotalInvocations: 0,
			LastUpdated:      time.Now(),
		},
		Validation: &ValidationStatus{
			IsValidated:     false,
			ValidatedBy:     []string{},
			ValidationScore: 0.0,
			LastValidated:   time.Time{},
		},
		Timestamp: time.Now(),
	}

	// Store skill node
	nrv.skillsMutex.Lock()
	nrv.skillNodes[skillID] = skillNode
	nrv.skillsMutex.Unlock()

	// Create NRV for skill discovery
	coordinates := nrv.calculateSkillCoordinates(skillNode)
	metadata := map[string]interface{}{
		"node_type":    "skill",
		"skill_id":     skillID,
		"skill_type":   skillType,
		"capabilities": capabilities,
	}

	_, err := nrv.CreateVector(skillID, coordinates, metadata)
	if err != nil {
		log.Printf("Warning: Failed to create NRV for skill node: %v", err)
	}

	return skillNode, nil
}

// GetSkillsForErrorType returns skills that can handle a specific error type
func (nrv *NRVSystem) GetSkillsForErrorType(errorType string) ([]*SkillNode, error) {
	var matchingSkills []*SkillNode

	nrv.skillsMutex.RLock()
	for _, skill := range nrv.skillNodes {
		// Check if skill can handle this error type
		for _, capability := range skill.Capabilities {
			if capability == errorType || capability == "general" {
				matchingSkills = append(matchingSkills, skill)
				break
			}
		}
	}
	nrv.skillsMutex.RUnlock()

	return matchingSkills, nil
}

// GetAllVectors returns all vectors
func (nrv *NRVSystem) GetAllVectors() []*NetworkResolutionVector {
	nrv.vectorsMutex.RLock()
	defer nrv.vectorsMutex.RUnlock()

	vectors := make([]*NetworkResolutionVector, 0, len(nrv.vectors))
	for _, vector := range nrv.vectors {
		vectors = append(vectors, vector)
	}

	return vectors
}

// GetAllErrorNodes returns all error nodes
func (nrv *NRVSystem) GetAllErrorNodes() []*ErrorNode {
	nrv.errorsMutex.RLock()
	defer nrv.errorsMutex.RUnlock()

	errors := make([]*ErrorNode, 0, len(nrv.errorNodes))
	for _, errorNode := range nrv.errorNodes {
		errors = append(errors, errorNode)
	}

	return errors
}

// GetAllSkillNodes returns all skill nodes
func (nrv *NRVSystem) GetAllSkillNodes() []*SkillNode {
	nrv.skillsMutex.RLock()
	defer nrv.skillsMutex.RUnlock()

	skills := make([]*SkillNode, 0, len(nrv.skillNodes))
	for _, skillNode := range nrv.skillNodes {
		skills = append(skills, skillNode)
	}

	return skills
}

// Helper methods
func (nrv *NRVSystem) generateVectorID(targetHash string, coordinates []float64) string {
	data := fmt.Sprintf("%s:%v:%d", targetHash, coordinates, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (nrv *NRVSystem) generateErrorID(errorType, description string) string {
	data := fmt.Sprintf("error:%s:%s:%d", errorType, description, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (nrv *NRVSystem) generateSkillID(skillType string, capabilities []string) string {
	data := fmt.Sprintf("skill:%s:%v:%d", skillType, capabilities, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (nrv *NRVSystem) calculateErrorCoordinates(errorNode *ErrorNode) []float64 {
	// Calculate coordinates based on error characteristics
	return []float64{float64(errorNode.Severity), float64(len(errorNode.Description))}
}

func (nrv *NRVSystem) calculateSkillCoordinates(skillNode *SkillNode) []float64 {
	// Calculate coordinates based on skill characteristics
	return []float64{float64(len(skillNode.Capabilities)), skillNode.Performance.SuccessRate}
}

// findResolutionPath attempts to find a resolution path for an error
func (nrv *NRVSystem) findResolutionPath(errorNode *ErrorNode) (*ResolutionPath, error) {
	// Query for relevant skills that can resolve this error type
	skills, err := nrv.GetSkillsForErrorType(errorNode.ErrorType)
	if err != nil {
		return nil, err
	}

	if len(skills) == 0 {
		return nil, fmt.Errorf("no skills found for error type: %s", errorNode.ErrorType)
	}

	// Calculate resolution path
	var steps []ResolutionStep
	totalConfidence := 0.0
	totalCost := 0.0

	for _, skill := range skills {
		step := ResolutionStep{
			Action: "invoke_skill",
			Parameters: map[string]interface{}{
				"skill_id": skill.ID,
				"context":  errorNode.Context,
			},
			SkillID:    skill.ID,
			Confidence: skill.Validation.ValidationScore,
		}

		steps = append(steps, step)
		totalConfidence += skill.Validation.ValidationScore
		totalCost += nrv.estimateSkillCost(skill)
	}

	avgConfidence := totalConfidence / float64(len(skills))

	return &ResolutionPath{
		Steps:         steps,
		Confidence:    avgConfidence,
		EstimatedCost: totalCost,
	}, nil
}

// processVectorUpdates processes vector updates in the background
func (nrv *NRVSystem) processVectorUpdates() {
	for {
		select {
		case update := <-nrv.updateChannel:
			switch update.Operation {
			case "create":
				log.Printf("Processing vector creation: %s", update.Vector.ID)
			case "update":
				log.Printf("Processing vector update: %s", update.Vector.ID)
			case "validate":
				log.Printf("Processing vector validation: %s", update.Vector.ID)
				nrv.validateVector(update.Vector)
			}
		case <-nrv.ctx.Done():
			return
		}
	}
}

// validateVector validates a vector
func (nrv *NRVSystem) validateVector(vector *NetworkResolutionVector) {
	// Implement vector validation logic
	if len(vector.Signatures) > 0 {
		vector.Confidence = math.Min(vector.Confidence*1.1, 1.0)
	}
}

// periodicVectorMaintenance performs periodic maintenance tasks
func (nrv *NRVSystem) periodicVectorMaintenance() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			nrv.cleanupExpiredVectors()
			nrv.updateVectorConfidences()
		case <-nrv.ctx.Done():
			return
		}
	}
}

// signVector signs a vector (placeholder implementation)
func (nrv *NRVSystem) signVector(vector *NetworkResolutionVector) ([]byte, error) {
	// Implement vector signing
	data, err := json.Marshal(vector)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(data)
	return hash[:], nil
}

// sortVectorsByConfidence sorts vectors by confidence in descending order
func (nrv *NRVSystem) sortVectorsByConfidence(vectors []*NetworkResolutionVector) {
	// Simple bubble sort for now
	n := len(vectors)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if vectors[j].Confidence < vectors[j+1].Confidence {
				vectors[j], vectors[j+1] = vectors[j+1], vectors[j]
			}
		}
	}
}

// estimateSkillCost estimates the cost of invoking a skill
func (nrv *NRVSystem) estimateSkillCost(skill *SkillNode) float64 {
	// Base cost calculation
	baseCost := 1.0

	// Adjust based on performance metrics
	if skill.Performance.SuccessRate > 0 {
		baseCost = baseCost / skill.Performance.SuccessRate
	}

	return baseCost
}

// cleanupExpiredVectors removes expired vectors
func (nrv *NRVSystem) cleanupExpiredVectors() {
	nrv.vectorsMutex.Lock()
	defer nrv.vectorsMutex.Unlock()

	now := time.Now()
	for id, vector := range nrv.vectors {
		if now.Sub(vector.Timestamp) > nrv.config.VectorTTL {
			delete(nrv.vectors, id)
			log.Printf("Cleaned up expired vector: %s", id)
		}
	}
}

// updateVectorConfidences updates vector confidences based on age
func (nrv *NRVSystem) updateVectorConfidences() {
	nrv.vectorsMutex.Lock()
	defer nrv.vectorsMutex.Unlock()

	for _, vector := range nrv.vectors {
		// Apply confidence decay over time
		vector.Confidence *= nrv.config.ConfidenceDecay
	}
}
