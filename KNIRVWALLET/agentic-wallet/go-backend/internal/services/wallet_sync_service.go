package services

import (
	"crypto-wallet-backend/internal/models"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WalletSyncService handles synchronization between native and browser wallets
type WalletSyncService struct {
	container       *Container
	activeSessions  map[string]*SyncSession
	sessionsMutex   sync.RWMutex
	wsConnections   map[string]*websocket.Conn
	connectionsMutex sync.RWMutex
}

// SyncSession represents an active synchronization session
type SyncSession struct {
	ID              string                 `json:"id"`
	UserID          uuid.UUID              `json:"user_id"`
	NativeWalletID  string                 `json:"native_wallet_id"`
	BrowserWalletID string                 `json:"browser_wallet_id"`
	Status          string                 `json:"status"` // pending, connected, syncing, completed, failed
	CreatedAt       time.Time              `json:"created_at"`
	LastSyncAt      *time.Time             `json:"last_sync_at,omitempty"`
	SyncData        map[string]interface{} `json:"sync_data"`
}

// SyncRequest represents a synchronization request
type SyncRequest struct {
	Type      string                 `json:"type"` // connect, sync_wallets, sync_transactions, sync_settings
	SessionID string                 `json:"session_id"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// SyncResponse represents a synchronization response
type SyncResponse struct {
	Success   bool                   `json:"success"`
	SessionID string                 `json:"session_id"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// WalletData represents wallet information for synchronization
type WalletData struct {
	Address     string                 `json:"address"`
	Name        string                 `json:"name"`
	Network     string                 `json:"network"`
	Balance     string                 `json:"balance"`
	IsActive    bool                   `json:"is_active"`
	Metadata    map[string]interface{} `json:"metadata"`
	LastUpdated time.Time              `json:"last_updated"`
}

// TransactionData represents transaction information for synchronization
type TransactionData struct {
	Hash        string                 `json:"hash"`
	From        string                 `json:"from"`
	To          string                 `json:"to"`
	Amount      string                 `json:"amount"`
	Token       string                 `json:"token"`
	Status      string                 `json:"status"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// NewWalletSyncService creates a new wallet synchronization service
func NewWalletSyncService(container *Container) *WalletSyncService {
	return &WalletSyncService{
		container:       container,
		activeSessions:  make(map[string]*SyncSession),
		wsConnections:   make(map[string]*websocket.Conn),
	}
}

// CreateSyncSession creates a new synchronization session
func (s *WalletSyncService) CreateSyncSession(userID uuid.UUID, nativeWalletID string) (*SyncSession, error) {
	sessionID := s.generateSessionID()
	
	session := &SyncSession{
		ID:             sessionID,
		UserID:         userID,
		NativeWalletID: nativeWalletID,
		Status:         "pending",
		CreatedAt:      time.Now(),
		SyncData:       make(map[string]interface{}),
	}

	s.sessionsMutex.Lock()
	s.activeSessions[sessionID] = session
	s.sessionsMutex.Unlock()

	log.Printf("Created sync session: %s for user: %s", sessionID, userID.String())
	return session, nil
}

// ConnectBrowserWallet connects a browser wallet to an existing sync session
func (s *WalletSyncService) ConnectBrowserWallet(sessionID, browserWalletID string) error {
	s.sessionsMutex.Lock()
	defer s.sessionsMutex.Unlock()

	session, exists := s.activeSessions[sessionID]
	if !exists {
		return fmt.Errorf("sync session not found: %s", sessionID)
	}

	if session.Status != "pending" {
		return fmt.Errorf("session is not in pending state: %s", session.Status)
	}

	session.BrowserWalletID = browserWalletID
	session.Status = "connected"

	log.Printf("Connected browser wallet %s to session %s", browserWalletID, sessionID)
	return nil
}

// SyncWallets synchronizes wallet data between platforms
func (s *WalletSyncService) SyncWallets(sessionID string) (*SyncResponse, error) {
	s.sessionsMutex.RLock()
	session, exists := s.activeSessions[sessionID]
	s.sessionsMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("sync session not found: %s", sessionID)
	}

	if session.Status != "connected" {
		return nil, fmt.Errorf("session is not connected: %s", session.Status)
	}

	// Update session status
	s.updateSessionStatus(sessionID, "syncing")

	// Get wallet data from native wallet
	nativeWallets, err := s.getNativeWalletData(session.UserID)
	if err != nil {
		s.updateSessionStatus(sessionID, "failed")
		return nil, fmt.Errorf("failed to get native wallet data: %w", err)
	}

	// Prepare sync response
	response := &SyncResponse{
		Success:   true,
		SessionID: sessionID,
		Type:      "sync_wallets",
		Data: map[string]interface{}{
			"wallets": nativeWallets,
		},
		Timestamp: time.Now(),
	}

	// Update session
	now := time.Now()
	session.LastSyncAt = &now
	session.SyncData["last_wallet_sync"] = now
	s.updateSessionStatus(sessionID, "completed")

	log.Printf("Synchronized wallets for session: %s", sessionID)
	return response, nil
}

// SyncTransactions synchronizes transaction data between platforms
func (s *WalletSyncService) SyncTransactions(sessionID string, walletAddress string) (*SyncResponse, error) {
	s.sessionsMutex.RLock()
	session, exists := s.activeSessions[sessionID]
	s.sessionsMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("sync session not found: %s", sessionID)
	}

	// Get transaction data
	transactions, err := s.getTransactionData(session.UserID, walletAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction data: %w", err)
	}

	response := &SyncResponse{
		Success:   true,
		SessionID: sessionID,
		Type:      "sync_transactions",
		Data: map[string]interface{}{
			"transactions": transactions,
			"wallet":       walletAddress,
		},
		Timestamp: time.Now(),
	}

	log.Printf("Synchronized transactions for wallet %s in session: %s", walletAddress, sessionID)
	return response, nil
}

// HandleWebSocketConnection handles WebSocket connections for real-time sync
func (s *WalletSyncService) HandleWebSocketConnection(sessionID string, conn *websocket.Conn) {
	s.connectionsMutex.Lock()
	s.wsConnections[sessionID] = conn
	s.connectionsMutex.Unlock()

	defer func() {
		s.connectionsMutex.Lock()
		delete(s.wsConnections, sessionID)
		s.connectionsMutex.Unlock()
		conn.Close()
	}()

	// Send initial connection confirmation
	s.sendWebSocketMessage(sessionID, SyncResponse{
		Success:   true,
		SessionID: sessionID,
		Type:      "connection_established",
		Data:      map[string]interface{}{"status": "connected"},
		Timestamp: time.Now(),
	})

	// Listen for messages
	for {
		var request SyncRequest
		err := conn.ReadJSON(&request)
		if err != nil {
			log.Printf("WebSocket read error for session %s: %v", sessionID, err)
			break
		}

		response := s.handleSyncRequest(&request)
		if err := conn.WriteJSON(response); err != nil {
			log.Printf("WebSocket write error for session %s: %v", sessionID, err)
			break
		}
	}
}

// BroadcastToSession sends a message to all connected clients in a session
func (s *WalletSyncService) BroadcastToSession(sessionID string, message interface{}) {
	s.connectionsMutex.RLock()
	conn, exists := s.wsConnections[sessionID]
	s.connectionsMutex.RUnlock()

	if exists {
		if err := conn.WriteJSON(message); err != nil {
			log.Printf("Failed to broadcast to session %s: %v", sessionID, err)
		}
	}
}

// GetActiveSession retrieves an active sync session
func (s *WalletSyncService) GetActiveSession(sessionID string) (*SyncSession, error) {
	s.sessionsMutex.RLock()
	defer s.sessionsMutex.RUnlock()

	session, exists := s.activeSessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("sync session not found: %s", sessionID)
	}

	return session, nil
}

// CleanupExpiredSessions removes expired sync sessions
func (s *WalletSyncService) CleanupExpiredSessions() {
	s.sessionsMutex.Lock()
	defer s.sessionsMutex.Unlock()

	now := time.Now()
	expiredThreshold := 30 * time.Minute

	for sessionID, session := range s.activeSessions {
		if now.Sub(session.CreatedAt) > expiredThreshold {
			delete(s.activeSessions, sessionID)
			log.Printf("Cleaned up expired session: %s", sessionID)
		}
	}
}

// Private helper methods

func (s *WalletSyncService) generateSessionID() string {
	return fmt.Sprintf("sync_%d_%s", time.Now().Unix(), uuid.New().String()[:8])
}

func (s *WalletSyncService) updateSessionStatus(sessionID, status string) {
	s.sessionsMutex.Lock()
	defer s.sessionsMutex.Unlock()

	if session, exists := s.activeSessions[sessionID]; exists {
		session.Status = status
	}
}

func (s *WalletSyncService) getNativeWalletData(userID uuid.UUID) ([]WalletData, error) {
	// TODO: Implement actual database query
	// This is a placeholder implementation
	wallets := []WalletData{
		{
			Address:     "knirv1example1234567890abcdef",
			Name:        "Main Wallet",
			Network:     "knirv_network",
			Balance:     "1000.50",
			IsActive:    true,
			Metadata:    map[string]interface{}{"type": "HD"},
			LastUpdated: time.Now(),
		},
	}

	return wallets, nil
}

func (s *WalletSyncService) getTransactionData(userID uuid.UUID, walletAddress string) ([]TransactionData, error) {
	// TODO: Implement actual database query
	// This is a placeholder implementation
	transactions := []TransactionData{
		{
			Hash:      "0x1234567890abcdef",
			From:      walletAddress,
			To:        "knirv1recipient1234567890",
			Amount:    "100.00",
			Token:     "NRN",
			Status:    "confirmed",
			Timestamp: time.Now().Add(-1 * time.Hour),
			Metadata:  map[string]interface{}{"gas_used": "21000"},
		},
	}

	return transactions, nil
}

func (s *WalletSyncService) handleSyncRequest(request *SyncRequest) *SyncResponse {
	switch request.Type {
	case "sync_wallets":
		response, err := s.SyncWallets(request.SessionID)
		if err != nil {
			return &SyncResponse{
				Success:   false,
				SessionID: request.SessionID,
				Type:      request.Type,
				Error:     err.Error(),
				Timestamp: time.Now(),
			}
		}
		return response

	case "sync_transactions":
		walletAddress, ok := request.Data["wallet_address"].(string)
		if !ok {
			return &SyncResponse{
				Success:   false,
				SessionID: request.SessionID,
				Type:      request.Type,
				Error:     "wallet_address is required",
				Timestamp: time.Now(),
			}
		}

		response, err := s.SyncTransactions(request.SessionID, walletAddress)
		if err != nil {
			return &SyncResponse{
				Success:   false,
				SessionID: request.SessionID,
				Type:      request.Type,
				Error:     err.Error(),
				Timestamp: time.Now(),
			}
		}
		return response

	default:
		return &SyncResponse{
			Success:   false,
			SessionID: request.SessionID,
			Type:      request.Type,
			Error:     "unsupported request type",
			Timestamp: time.Now(),
		}
	}
}

func (s *WalletSyncService) sendWebSocketMessage(sessionID string, message interface{}) {
	s.connectionsMutex.RLock()
	conn, exists := s.wsConnections[sessionID]
	s.connectionsMutex.RUnlock()

	if exists {
		if err := conn.WriteJSON(message); err != nil {
			log.Printf("Failed to send WebSocket message to session %s: %v", sessionID, err)
		}
	}
}
