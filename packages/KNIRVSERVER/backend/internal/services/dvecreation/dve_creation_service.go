package dvecreation

import (
	"context"
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
	"backend_server/internal/services/blockchain/transactionchain"
	"backend_server/internal/services/container"

	"github.com/google/uuid"
	"github.com/tidwall/buntdb"
)

type ChainClientInterface interface {
	VerifyPaymentTransaction(txHash string, expectedAmount int64, expectedRecipient string) (*objects.NRNPayment, error)
	GetTransactionPool() ([]*transactionchain.Transaction, error)
	SubmitTransaction(tx *transactionchain.Transaction) (string, error)
	GetAccountBalance(address string) (int64, error)
	GetBlockHeight() (uint64, error)
	GetChainID() (string, error)
	RegisterDVENode(nodeID, ownerAddress string, stakeAmount int64) (string, error)
	CreateChainSession(dveNodeID, ownerAddress string) (*objects.ChainSession, error)
	ValidateSession(sessionID string) (*objects.ChainSession, error)
	GetSecret(sessionID, secretKey string) (string, error)
	Close() error
}

type ContainerOrchestratorInterface interface {
	ProvisionContainer(rentalID string) (*container.Container, error)
	GetSSHPrivateKey(containerID string) (string, error)
	TerminateContainer(containerID string) error
}

// KnirvagentManagerInterface defines the subset of knirvagent.Manager methods
// needed by DVECreationService to auto-provision the DVE Supervisor Agent.
type KnirvagentManagerInterface interface {
	StartAgent(ctx context.Context, dveID string, startTimeout time.Duration) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsRunning() bool
	HealthCheck(ctx context.Context) error
	RunningCount() int
}

type DVEManagerInterface interface {
	RegisterNode(req *objects.RegisterNodeRequest) (*objects.DVENode, error)
	UpdateNode(nodeID string, updates map[string]interface{}) (*objects.DVENode, error)
}

type DVECreationService struct {
	db                    *database.BuntDBManager
	mu                    sync.RWMutex
	running               bool
	chainClient           ChainClientInterface
	dveManager            DVEManagerInterface
	containerOrchestrator ContainerOrchestratorInterface
	knirvagentManager     KnirvagentManagerInterface
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

func (dcs *DVECreationService) SetDveManager(manager DVEManagerInterface) {
	dcs.mu.Lock()
	defer dcs.mu.Unlock()
	dcs.dveManager = manager
}

func (dcs *DVECreationService) SetContainerOrchestrator(orchestrator ContainerOrchestratorInterface) {
	dcs.mu.Lock()
	defer dcs.mu.Unlock()
	dcs.containerOrchestrator = orchestrator
}

func (dcs *DVECreationService) SetKnirvagentManager(mgr KnirvagentManagerInterface) {
	dcs.mu.Lock()
	defer dcs.mu.Unlock()
	dcs.knirvagentManager = mgr
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
	go dcs.nodeHeartbeatRoutine()

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

	dcs.mu.Lock()
	dveNodeID := fmt.Sprintf("dve-%s", uuid.New().String()[:12])

	creation := &objects.DVECreation{
		ID:             uuid.New().String(),
		Name:           req.Name,
		OwnerID:        req.OwnerID,
		OwnerAddress:   req.OwnerAddress,
		DVENodeID:      dveNodeID,
		StakeAmount:    req.StakeAmount,
		Status:         "pending",
		TEEType:        req.TEEType,
		TEEAttestation: req.TEEAttestation,
		SessionKeyID:   generateSessionKeyID(),
		Capabilities:   req.Capabilities,
		ResourceLimits: req.ResourceLimits,
		RegisteredAt:   time.Now(),
		LastHeartbeat:  time.Now(),
		UpdatedAt:      time.Now(),
		Persistent:     req.Persistent,
		GracePeriod:    dcs.gracePeriod,
	}

	dcs.activeCreations[creation.ID] = creation
	dcs.mu.Unlock()

	// Synchronously register the node in the DVE manager so it appears in the
	// nodes list immediately — this is fast (in-memory + DB write, no I/O).
	if dcs.dveManager != nil {
		nodeReq := &objects.RegisterNodeRequest{
			Name:         creation.Name,
			TEEType:      creation.TEEType,
			StakeAmount:  creation.StakeAmount,
			Location:     "local-dve",
			Capabilities: creation.Capabilities,
		}
		if node, err := dcs.dveManager.RegisterNode(nodeReq); err != nil {
			log.Printf("[DVE Creation] Warning: Failed to register node in manager: %v", err)
		} else {
			dcs.mu.Lock()
			creation.DVENodeID = node.ID
			dcs.mu.Unlock()
			log.Printf("[DVE Creation] Registered DVE node %s immediately for creation %s", node.ID, creation.ID)
		}
	}

	// Save initial pending record to database (no lock needed — buntdb is thread-safe)
	if err := dcs.saveCreationToDatabase(creation); err != nil {
		log.Printf("Warning: Failed to save DVE creation to database: %v", err)
	}

	// Provision DVE asynchronously — container, agent, chain session
	go dcs.provisionDVEInBackground(creation, req)

	return &objects.DVECreationResponse{
		Success:     true,
		DVECreation: creation,
		Message:     "DVE node creation initiated — provisioning in background",
	}, nil
}

// provisionDVEInBackground runs in a goroutine. It does the slow work of
// container provisioning, DVE manager registration, KNIRVAGENT startup,
// and chain session establishment. Updates creation status to "active"
// on success or logs errors on failure.
func (dcs *DVECreationService) provisionDVEInBackground(creation *objects.DVECreation, req *objects.DVECreationRequest) {
	// 1. Attempt on-chain registration (best-effort)
	var registrationTxHash string
	if dcs.chainClient != nil {
		txHash, err := dcs.chainClient.RegisterDVENode(creation.DVENodeID, req.OwnerAddress, req.StakeAmount)
		if err != nil {
			log.Printf("[DVE Creation] Chain registration failed for %s: %v, proceeding with local provisioning", creation.ID, err)
		} else {
			registrationTxHash = txHash
		}
	}
	creation.RegistrationTxHash = registrationTxHash

	// 2. Provision container if orchestrator is available
	if dcs.containerOrchestrator != nil {
		log.Printf("[DVE Creation] Provisioning container for creation %s", creation.ID)
		container, err := dcs.containerOrchestrator.ProvisionContainer(creation.ID)
		if err != nil {
			log.Printf("[DVE Creation] Error provisioning container for %s: %v", creation.ID, err)
			dcs.updateCreationStatus(creation.ID, "failed")
			return
		}

		// Preserve the manager node ID that was registered synchronously in
		// CreateDVENode — container.ID is only used internally by the orchestrator
		// and must not overwrite the stable DVENodeID that the nodes list uses.
		managerNodeID := creation.DVENodeID
		creation.SSHPublicKey = container.Spec.SSHPublicKey
		creation.SSHPrivateKey = container.SSHKeys.PrivateKey
		creation.SSHPort = container.Endpoints.SSHPort
		creation.IPAddress = "localhost"
		creation.ValidationPort = container.Endpoints.ValidationPort
		creation.ErrorResPort = container.Endpoints.ErrorResPort

		// 3. Update the DVE node in the manager with container details so
		// SSH/IP info is available for Access.  If no manager node was registered
		// yet (no dveManager at creation time) fall back to registering now.
		if dcs.dveManager != nil {
			if managerNodeID != "" {
				updates := map[string]interface{}{
					"ip_address": creation.IPAddress,
					"ssh_port":   creation.SSHPort,
					"public_key": creation.SSHPublicKey,
					"status":     "online",
				}
				if _, err := dcs.dveManager.UpdateNode(managerNodeID, updates); err != nil {
					log.Printf("[DVE Creation] Warning: Failed to update DVE node details: %v", err)
				} else {
					log.Printf("[DVE Creation] Updated DVE node %s with container details", managerNodeID)
				}
			} else {
				// Fallback: manager not wired at creation time — register now with full details
				nodeReq := &objects.RegisterNodeRequest{
					Name:         creation.Name,
					TEEType:      creation.TEEType,
					StakeAmount:  creation.StakeAmount,
					Location:     "local-dve",
					IPAddress:    creation.IPAddress,
					SSHPort:      creation.SSHPort,
					PublicKey:    creation.SSHPublicKey,
					Capabilities: creation.Capabilities,
				}
				if node, err := dcs.dveManager.RegisterNode(nodeReq); err != nil {
					log.Printf("[DVE Creation] Warning: Failed to register DVE node in tracker: %v", err)
				} else {
					creation.DVENodeID = node.ID
					log.Printf("[DVE Creation] Registered DVE node %s (fallback)", node.ID)
				}
			}
		}

		// 4. Auto-start KNIRVAGENT supervisor
		if dcs.knirvagentManager != nil {
			agentDVEID := creation.DVENodeID
			startCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := dcs.knirvagentManager.StartAgent(startCtx, agentDVEID, 30*time.Second); err != nil {
				log.Printf("[DVE Creation] Warning: Failed to start KNIRVAGENT supervisor for DVE %s: %v", agentDVEID, err)
			} else {
				creation.SupervisorAgentID = agentDVEID
				log.Printf("[DVE Creation] KNIRVAGENT supervisor started for DVE %s (node %s)", creation.Name, agentDVEID)
			}
		}
	}

	// 5. Establish chain session (best-effort)
	chainSession, err := dcs.establishChainSession(creation)
	if err != nil {
		log.Printf("[DVE Creation] Warning: Failed to establish chain session: %v", err)
	}

	// 6. Create DVE session
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

	now := time.Now()
	creation.Status = "active"
	creation.ChainSessionID = session.ChainSessionID
	creation.ActivatedAt = &now
	creation.UpdatedAt = now

	// 7. Update in-memory maps and persist
	dcs.mu.Lock()
	dcs.activeCreations[creation.ID] = creation
	dcs.activeSessions[session.ID] = session
	dcs.mu.Unlock()

	if err := dcs.saveCreationToDatabase(creation); err != nil {
		log.Printf("[DVE Creation] Warning: Failed to save updated DVE creation: %v", err)
	}
	if err := dcs.saveSessionToDatabase(session); err != nil {
		log.Printf("[DVE Creation] Warning: Failed to save DVE session: %v", err)
	}

	log.Printf("[DVE Creation] DVE %s (%s) fully provisioned and active", creation.Name, creation.ID)
}

// updateCreationStatus sets the status of an in-memory creation and persists it.
// Must be called from the background goroutine (not while holding the mutex).
func (dcs *DVECreationService) updateCreationStatus(id, status string) {
	dcs.mu.Lock()
	creation, exists := dcs.activeCreations[id]
	if exists {
		creation.Status = status
		creation.UpdatedAt = time.Now()
	}
	dcs.mu.Unlock()

	if exists {
		if err := dcs.saveCreationToDatabase(creation); err != nil {
			log.Printf("Warning: Failed to save status update for creation %s: %v", id, err)
		}
	}
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

// nodeHeartbeatRoutine keeps active DVE nodes alive in the DVEManager so the
// health-check goroutine (which marks nodes offline after 2 min without a
// heartbeat) does not silently hide them from the KNIRVGATEWAY WebGUI list.
func (dcs *DVECreationService) nodeHeartbeatRoutine() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if !dcs.running {
			return
		}
		dcs.refreshActiveNodeHeartbeats()
	}
}

func (dcs *DVECreationService) refreshActiveNodeHeartbeats() {
	if dcs.dveManager == nil {
		return
	}

	dcs.mu.RLock()
	var nodeIDs []string
	for _, creation := range dcs.activeCreations {
		if creation.Status == "active" && creation.DVENodeID != "" {
			nodeIDs = append(nodeIDs, creation.DVENodeID)
		}
	}
	dcs.mu.RUnlock()

	for _, nodeID := range nodeIDs {
		if _, err := dcs.dveManager.UpdateNode(nodeID, map[string]interface{}{"status": "online"}); err != nil {
			log.Printf("[DVE Creation] Warning: Failed to refresh heartbeat for node %s: %v", nodeID, err)
		}
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
