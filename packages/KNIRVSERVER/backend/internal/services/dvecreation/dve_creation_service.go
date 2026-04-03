package dvecreation

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"backend_server/internal/database"
	"backend_server/internal/objects"
	"backend_server/internal/services/blockchain"

	"github.com/google/uuid"
	"github.com/tidwall/buntdb"
)

type ChainClientInterface interface {
	VerifyPaymentTransaction(txHash string, expectedAmount int64, expectedRecipient string) (*objects.NRNPayment, error)
	GetTransactionPool() ([]*blockchain.Transaction, error)
	SubmitTransaction(tx *blockchain.Transaction) (string, error)
	GetAccountBalance(address string) (int64, error)
	GetBlockHeight() (uint64, error)
	GetChainID() (string, error)
	RegisterDVENode(nodeID, ownerAddress string, stakeAmount int64) (string, error)
	CreateChainSession(dveNodeID, ownerAddress string) (*objects.ChainSession, error)
	ValidateSession(sessionID string) (*objects.ChainSession, error)
	GetSecret(sessionID, secretKey string) (string, error)
	Close() error
}

type DVECreationService struct {
	db                    *database.BuntDBManager
	mu                    sync.RWMutex
	running               bool
	chainClient           ChainClientInterface
	dveManager            interface{}
	containerOrchestrator interface{}
	activeCreations       map[string]*objects.DVECreation
	activeSessions        map[string]*objects.DVESession
	cleanupInterval       time.Duration
	minStakeAmount        int64
	gracePeriod           int64
}

func NewDVECreationService(db *database.BuntDBManager) (*DVECreationService, error) {
	log.Printf("[DVE Creation] Creating new DVE creation service")
	service := &DVECreationService{
		db:              db,
		activeCreations: make(map[string]*objects.DVECreation),
		activeSessions:  make(map[string]*objects.DVESession),
		cleanupInterval: 5 * time.Minute,
		minStakeAmount:  1000,
		gracePeriod:     7 * 24 * 60 * 60,
	}

	if err := service.loadFromDatabase(); err != nil {
		log.Printf("Warning: Failed to load DVE creation data from database: %v", err)
	}

	log.Printf("[DVE Creation] Service initialized")
	return service, nil
}

func (dcs *DVECreationService) SetChainClient(client ChainClientInterface) {
	dcs.mu.Lock()
	defer dcs.mu.Unlock()
	dcs.chainClient = client
}

func (dcs *DVECreationService) SetTransactionChainClient(client ChainClientInterface) {
	dcs.SetChainClient(client)
}

func (dcs *DVECreationService) SetDveManager(manager interface{}) {
	dcs.mu.Lock()
	defer dcs.mu.Unlock()
	dcs.dveManager = manager
}

func (dcs *DVECreationService) SetContainerOrchestrator(orchestrator interface{}) {
	dcs.mu.Lock()
	defer dcs.mu.Unlock()
	dcs.containerOrchestrator = orchestrator
}

func (dcs *DVECreationService) Start() error {
	dcs.mu.Lock()
	defer dcs.mu.Unlock()

	if dcs.running {
		return fmt.Errorf("DVE creation service is already running")
	}

	log.Println("Starting DVE creation service...")

	go dcs.cleanupRoutine()
	go dcs.sessionValidationRoutine()

	dcs.running = true
	log.Println("DVE creation service started successfully")
	return nil
}

func (dcs *DVECreationService) Stop() error {
	dcs.mu.Lock()
	defer dcs.mu.Unlock()

	if !dcs.running {
		return nil
	}

	log.Println("Stopping DVE creation service...")

	if err := dcs.saveToDatabase(); err != nil {
		log.Printf("Warning: Failed to save DVE creation data to database: %v", err)
	}

	dcs.running = false
	log.Println("DVE creation service stopped")
	return nil
}

func (dcs *DVECreationService) IsRunning() bool {
	dcs.mu.RLock()
	defer dcs.mu.RUnlock()
	return dcs.running
}

func (dcs *DVECreationService) CreateDVENode(req *objects.DVECreationRequest) (*objects.DVECreationResponse, error) {
	dcs.mu.Lock()
	defer dcs.mu.Unlock()

	if req.StakeAmount < dcs.minStakeAmount {
		return &objects.DVECreationResponse{
			Success: false,
			Error:   fmt.Sprintf("minimum stake amount is %d NRN", dcs.minStakeAmount),
		}, nil
	}

	if req.OwnerID == "" || req.OwnerAddress == "" {
		return &objects.DVECreationResponse{
			Success: false,
			Error:   "owner ID and address are required",
		}, nil
	}

	dveNodeID := fmt.Sprintf("dve-%s", uuid.New().String()[:12])

	var registrationTxHash string
	if dcs.chainClient != nil {
		txHash, err := dcs.chainClient.RegisterDVENode(dveNodeID, req.OwnerAddress, req.StakeAmount)
		if err != nil {
			log.Printf("Chain registration failed: %v, proceeding with local registration", err)
		} else {
			registrationTxHash = txHash
		}
	}

	creation := &objects.DVECreation{
		ID:                 uuid.New().String(),
		Name:               req.Name,
		OwnerID:            req.OwnerID,
		OwnerAddress:       req.OwnerAddress,
		DVENodeID:          dveNodeID,
		StakeAmount:        req.StakeAmount,
		RegistrationTxHash: registrationTxHash,
		Status:             "pending",
		TEEType:            req.TEEType,
		TEEAttestation:     req.TEEAttestation,
		SessionKeyID:       generateSessionKeyID(),
		Capabilities:       req.Capabilities,
		ResourceLimits:     req.ResourceLimits,
		RegisteredAt:       time.Now(),
		LastHeartbeat:      time.Now(),
		UpdatedAt:          time.Now(),
		Persistent:         req.Persistent,
		GracePeriod:        dcs.gracePeriod,
	}

	dcs.activeCreations[creation.ID] = creation

	if err := dcs.saveCreationToDatabase(creation); err != nil {
		log.Printf("Warning: Failed to save DVE creation to database: %v", err)
	}

	chainSession, err := dcs.establishChainSession(creation)
	if err != nil {
		log.Printf("Warning: Failed to establish chain session: %v", err)
	}

	session := &objects.DVESession{
		ID:             uuid.New().String(),
		DVECreationID:  creation.ID,
		DVENodeID:      creation.DVENodeID,
		OwnerID:        creation.OwnerID,
		SessionKey:     generateSessionKey(),
		SessionKeyID:   creation.SessionKeyID,
		TEEBinding:     req.TEEAttestation,
		ChainSessionID: "",
		Status:         "pending",
		CreatedAt:      time.Now(),
		ExpiresAt:      time.Now().Add(30 * time.Minute),
	}

	if chainSession != nil {
		session.ChainSessionID = chainSession.SessionID
		session.Status = "active"
		session.ExpiresAt = chainSession.ExpiresAt
		session.PQCSignature = chainSession.PQCSignature
	}

	dcs.activeSessions[session.ID] = session

	creation.Status = "active"
	creation.ChainSessionID = session.ChainSessionID

	now := time.Now()
	creation.ActivatedAt = &now
	creation.UpdatedAt = time.Now()

	if err := dcs.saveCreationToDatabase(creation); err != nil {
		log.Printf("Warning: Failed to save updated DVE creation: %v", err)
	}

	if err := dcs.saveSessionToDatabase(session); err != nil {
		log.Printf("Warning: Failed to save DVE session: %v", err)
	}

	return &objects.DVECreationResponse{
		Success:            true,
		DVECreation:        creation,
		Session:            session,
		RegistrationTxHash: registrationTxHash,
		ChainSession:       chainSession,
		Message:            "DVE node created successfully",
	}, nil
}

func (dcs *DVECreationService) GetDVECreation(creationID string) (*objects.DVECreation, error) {
	dcs.mu.RLock()
	defer dcs.mu.RUnlock()

	if creation, exists := dcs.activeCreations[creationID]; exists {
		return creation, nil
	}

	var creation objects.DVECreation
	err := dcs.db.ViewTransaction(func(tx *buntdb.Tx) error {
		val, err := tx.Get("dve_creation:" + creationID)
		if err != nil {
			return err
		}
		return json.Unmarshal([]byte(val), &creation)
	})

	if err != nil {
		return nil, fmt.Errorf("DVE creation not found: %w", err)
	}

	return &creation, nil
}

func (dcs *DVECreationService) GetDVECreationByNodeID(nodeID string) (*objects.DVECreation, error) {
	dcs.mu.RLock()
	defer dcs.mu.RUnlock()

	for _, creation := range dcs.activeCreations {
		if creation.DVENodeID == nodeID {
			return creation, nil
		}
	}

	var result *objects.DVECreation
	err := dcs.db.ViewTransaction(func(tx *buntdb.Tx) error {
		return tx.Ascend("", func(key, value string) bool {
			if len(key) > 13 && key[:13] == "dve_creation:" {
				var creation objects.DVECreation
				if err := json.Unmarshal([]byte(value), &creation); err == nil {
					if creation.DVENodeID == nodeID {
						result = &creation
						return false
					}
				}
			}
			return true
		})
	})

	if err != nil {
		return nil, fmt.Errorf("DVE creation not found: %w", err)
	}

	return result, nil
}

func (dcs *DVECreationService) GetUserDVECreations(userID string) ([]*objects.DVECreation, error) {
	dcs.mu.RLock()
	defer dcs.mu.RUnlock()

	var userCreations []*objects.DVECreation
	for _, creation := range dcs.activeCreations {
		if creation.OwnerID == userID {
			userCreations = append(userCreations, creation)
		}
	}

	err := dcs.db.ViewTransaction(func(tx *buntdb.Tx) error {
		return tx.Ascend("", func(key, value string) bool {
			if len(key) > 13 && key[:13] == "dve_creation:" {
				var creation objects.DVECreation
				if err := json.Unmarshal([]byte(value), &creation); err == nil {
					if creation.OwnerID == userID {
						found := false
						for _, c := range userCreations {
							if c.ID == creation.ID {
								found = true
								break
							}
						}
						if !found {
							userCreations = append(userCreations, &creation)
						}
					}
				}
			}
			return true
		})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to query user DVE creations: %w", err)
	}

	return userCreations, nil
}

func (dcs *DVECreationService) GetDVESession(sessionID string) (*objects.DVESession, error) {
	dcs.mu.RLock()
	defer dcs.mu.RUnlock()

	if session, exists := dcs.activeSessions[sessionID]; exists {
		return session, nil
	}

	var session objects.DVESession
	err := dcs.db.ViewTransaction(func(tx *buntdb.Tx) error {
		val, err := tx.Get("dve_session:" + sessionID)
		if err != nil {
			return err
		}
		return json.Unmarshal([]byte(val), &session)
	})

	if err != nil {
		return nil, fmt.Errorf("DVE session not found: %w", err)
	}

	return &session, nil
}

func (dcs *DVECreationService) GetActiveSessionByCreationID(creationID string) (*objects.DVESession, error) {
	dcs.mu.RLock()
	defer dcs.mu.RUnlock()

	for _, session := range dcs.activeSessions {
		if session.DVECreationID == creationID && session.Status == "active" {
			return session, nil
		}
	}

	var result *objects.DVESession
	err := dcs.db.ViewTransaction(func(tx *buntdb.Tx) error {
		return tx.Ascend("", func(key, value string) bool {
			if len(key) > 11 && key[:11] == "dve_session:" {
				var session objects.DVESession
				if err := json.Unmarshal([]byte(value), &session); err == nil {
					if session.DVECreationID == creationID && session.Status == "active" {
						result = &session
						return false
					}
				}
			}
			return true
		})
	})

	if err != nil {
		return nil, fmt.Errorf("active session not found: %w", err)
	}

	return result, nil
}

func (dcs *DVECreationService) RefreshSession(creationID string) (*objects.DVESession, error) {
	dcs.mu.Lock()
	defer dcs.mu.Unlock()

	creation, exists := dcs.activeCreations[creationID]
	if !exists {
		return nil, fmt.Errorf("DVE creation not found: %s", creationID)
	}

	chainSession, err := dcs.establishChainSession(creation)
	if err != nil {
		return nil, fmt.Errorf("failed to establish chain session: %w", err)
	}

	session := &objects.DVESession{
		ID:             uuid.New().String(),
		DVECreationID:  creation.ID,
		DVENodeID:      creation.DVENodeID,
		OwnerID:        creation.OwnerID,
		SessionKey:     generateSessionKey(),
		SessionKeyID:   creation.SessionKeyID,
		TEEBinding:     creation.TEEAttestation,
		ChainSessionID: chainSession.SessionID,
		Status:         "active",
		CreatedAt:      time.Now(),
		ExpiresAt:      chainSession.ExpiresAt,
		PQCSignature:   chainSession.PQCSignature,
	}

	dcs.activeSessions[session.ID] = session

	if err := dcs.saveSessionToDatabase(session); err != nil {
		log.Printf("Warning: Failed to save refreshed session: %v", err)
	}

	creation.LastHeartbeat = time.Now()
	creation.UpdatedAt = time.Now()

	if err := dcs.saveCreationToDatabase(creation); err != nil {
		log.Printf("Warning: Failed to update DVE creation: %v", err)
	}

	return session, nil
}

func (dcs *DVECreationService) RevokeSession(sessionID string) error {
	dcs.mu.Lock()
	defer dcs.mu.Unlock()

	session, exists := dcs.activeSessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.Status = "revoked"

	if err := dcs.saveSessionToDatabase(session); err != nil {
		return fmt.Errorf("failed to save revoked session: %w", err)
	}

	delete(dcs.activeSessions, sessionID)

	return nil
}

func (dcs *DVECreationService) DecommissionDVENode(creationID, ownerID string) error {
	dcs.mu.Lock()
	defer dcs.mu.Unlock()

	creation, exists := dcs.activeCreations[creationID]
	if !exists {
		return fmt.Errorf("DVE creation not found: %s", creationID)
	}

	if creation.OwnerID != ownerID {
		return fmt.Errorf("unauthorized: not the owner of this DVE")
	}

	creation.Status = "decommissioned"
	creation.UpdatedAt = time.Now()

	for sessionID, session := range dcs.activeSessions {
		if session.DVECreationID == creationID {
			session.Status = "revoked"
			dcs.saveSessionToDatabase(session)
			delete(dcs.activeSessions, sessionID)
		}
	}

	if err := dcs.saveCreationToDatabase(creation); err != nil {
		return fmt.Errorf("failed to save decommissioned DVE: %w", err)
	}

	delete(dcs.activeCreations, creationID)

	return nil
}

func (dcs *DVECreationService) Heartbeat(creationID string) error {
	dcs.mu.Lock()
	defer dcs.mu.Unlock()

	creation, exists := dcs.activeCreations[creationID]
	if !exists {
		return fmt.Errorf("DVE creation not found: %s", creationID)
	}

	creation.LastHeartbeat = time.Now()
	creation.UpdatedAt = time.Now()

	if err := dcs.saveCreationToDatabase(creation); err != nil {
		return fmt.Errorf("failed to save heartbeat: %w", err)
	}

	return nil
}

func (dcs *DVECreationService) GetSecret(sessionID, secretKey string) (string, error) {
	dcs.mu.RLock()
	defer dcs.mu.RUnlock()

	if dcs.chainClient == nil {
		return "", fmt.Errorf("chain client not configured")
	}

	return dcs.chainClient.GetSecret(sessionID, secretKey)
}

func (dcs *DVECreationService) GetStats() (*DVECreationStats, error) {
	dcs.mu.RLock()
	defer dcs.mu.RUnlock()

	stats := &DVECreationStats{
		TotalCreations:  int64(len(dcs.activeCreations)),
		ActiveCreations: dcs.countActiveCreations(),
		ActiveSessions:  dcs.countActiveSessions(),
		Timestamp:       time.Now(),
	}

	return stats, nil
}

type DVECreationStats struct {
	TotalCreations  int64     `json:"total_creations"`
	ActiveCreations int64     `json:"active_creations"`
	ActiveSessions  int64     `json:"active_sessions"`
	Timestamp       time.Time `json:"timestamp"`
}

func (dcs *DVECreationService) establishChainSession(creation *objects.DVECreation) (*objects.ChainSession, error) {
	if dcs.chainClient == nil {
		return nil, fmt.Errorf("chain client not configured")
	}

	return dcs.chainClient.CreateChainSession(creation.DVENodeID, creation.OwnerAddress)
}

func (dcs *DVECreationService) loadFromDatabase() error {
	if dcs.db == nil {
		return fmt.Errorf("database not initialized")
	}

	err := dcs.db.GetObjectsByPrefix("dve_creation:", func(key string, value []byte) bool {
		var creation objects.DVECreation
		if err := json.Unmarshal(value, &creation); err == nil {
			if creation.Status == "active" || creation.Status == "pending" {
				dcs.activeCreations[creation.ID] = &creation
			}
		}
		return true
	})
	if err != nil {
		return err
	}

	return dcs.db.GetObjectsByPrefix("dve_session:", func(key string, value []byte) bool {
		var session objects.DVESession
		if err := json.Unmarshal(value, &session); err == nil {
			if session.Status == "active" {
				dcs.activeSessions[session.ID] = &session
			}
		}
		return true
	})
}

func (dcs *DVECreationService) saveToDatabase() error {
	if dcs.db == nil {
		return fmt.Errorf("database not initialized")
	}

	return dcs.db.Transaction(func(tx *buntdb.Tx) error {
		for _, creation := range dcs.activeCreations {
			if data, err := json.Marshal(creation); err == nil {
				tx.Set("dve_creation:"+creation.ID, string(data), nil)
			}
		}
		for _, session := range dcs.activeSessions {
			if data, err := json.Marshal(session); err == nil {
				tx.Set("dve_session:"+session.ID, string(data), nil)
			}
		}
		return nil
	})
}

func (dcs *DVECreationService) saveCreationToDatabase(creation *objects.DVECreation) error {
	if dcs.db == nil {
		return fmt.Errorf("database not initialized")
	}

	data, err := json.Marshal(creation)
	if err != nil {
		return fmt.Errorf("failed to marshal creation: %w", err)
	}

	return dcs.db.Transaction(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set("dve_creation:"+creation.ID, string(data), nil)
		return err
	})
}

func (dcs *DVECreationService) saveSessionToDatabase(session *objects.DVESession) error {
	if dcs.db == nil {
		return fmt.Errorf("database not initialized")
	}

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	return dcs.db.Transaction(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set("dve_session:"+session.ID, string(data), nil)
		return err
	})
}

func (dcs *DVECreationService) countActiveCreations() int64 {
	count := int64(0)
	now := time.Now()
	for _, creation := range dcs.activeCreations {
		if creation.Status == "active" && now.Sub(creation.LastHeartbeat) < 10*time.Minute {
			count++
		}
	}
	return count
}

func (dcs *DVECreationService) countActiveSessions() int64 {
	count := int64(0)
	now := time.Now()
	for _, session := range dcs.activeSessions {
		if session.Status == "active" && now.Before(session.ExpiresAt) {
			count++
		}
	}
	return count
}

func (dcs *DVECreationService) cleanupRoutine() {
	ticker := time.NewTicker(dcs.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		if !dcs.running {
			return
		}
		dcs.cleanupExpiredSessions()
	}
}

func (dcs *DVECreationService) cleanupExpiredSessions() {
	dcs.mu.Lock()
	defer dcs.mu.Unlock()

	now := time.Now()
	expiredCount := 0

	for id, session := range dcs.activeSessions {
		if now.After(session.ExpiresAt) {
			session.Status = "expired"
			dcs.saveSessionToDatabase(session)
			delete(dcs.activeSessions, id)
			expiredCount++
		}
	}

	if expiredCount > 0 {
		log.Printf("Cleaned up %d expired DVE sessions", expiredCount)
	}
}

func (dcs *DVECreationService) sessionValidationRoutine() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if !dcs.running {
			return
		}
		dcs.validateChainSessions()
	}
}

func (dcs *DVECreationService) validateChainSessions() {
	dcs.mu.RLock()
	sessions := make([]*objects.DVESession, 0, len(dcs.activeSessions))
	for _, session := range dcs.activeSessions {
		if session.Status == "active" {
			sessions = append(sessions, session)
		}
	}
	dcs.mu.RUnlock()

	if dcs.chainClient == nil {
		return
	}

	for _, session := range sessions {
		_, err := dcs.chainClient.ValidateSession(session.ChainSessionID)
		if err != nil {
			log.Printf("Session validation failed for %s: %v", session.ChainSessionID, err)
			dcs.mu.Lock()
			session.Status = "expired"
			dcs.saveSessionToDatabase(session)
			delete(dcs.activeSessions, session.ID)
			dcs.mu.Unlock()
		}
	}
}

func generateSessionKey() []byte {
	key := make([]byte, 32)
	rand.Read(key)
	return key
}

func generateSessionKeyID() string {
	hash := sha256.Sum256([]byte(uuid.New().String()))
	return base64.URLEncoding.EncodeToString(hash[:])[:16]
}
