package api

import (
	"crypto-wallet-backend/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WalletSyncHandler handles wallet synchronization API endpoints
type WalletSyncHandler struct {
	syncService *services.WalletSyncService
	upgrader    websocket.Upgrader
}

// CreateSyncSessionRequest represents the request to create a sync session
type CreateSyncSessionRequest struct {
	NativeWalletID string `json:"native_wallet_id" binding:"required"`
}

// ConnectBrowserWalletRequest represents the request to connect a browser wallet
type ConnectBrowserWalletRequest struct {
	SessionID       string `json:"session_id" binding:"required"`
	BrowserWalletID string `json:"browser_wallet_id" binding:"required"`
}

// SyncWalletsRequest represents the request to sync wallets
type SyncWalletsRequest struct {
	SessionID string `json:"session_id" binding:"required"`
}

// SyncTransactionsRequest represents the request to sync transactions
type SyncTransactionsRequest struct {
	SessionID     string `json:"session_id" binding:"required"`
	WalletAddress string `json:"wallet_address" binding:"required"`
}

// NewWalletSyncHandler creates a new wallet sync handler
func NewWalletSyncHandler(syncService *services.WalletSyncService) *WalletSyncHandler {
	return &WalletSyncHandler{
		syncService: syncService,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// TODO: Implement proper origin checking for production
				return true
			},
		},
	}
}

// CreateSyncSession creates a new synchronization session
// @Summary Create sync session
// @Description Create a new synchronization session between native and browser wallets
// @Tags wallet-sync
// @Accept json
// @Produce json
// @Param request body CreateSyncSessionRequest true "Sync session creation request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/sync/session/create [post]
func (h *WalletSyncHandler) CreateSyncSession(c *gin.Context) {
	var req CreateSyncSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	// Get user ID from context (assuming it's set by auth middleware)
	userIDStr, exists := c.Get("user_id")
	if !exists {
		// For testing purposes, use a default user ID
		userIDStr = "00000000-0000-0000-0000-000000000000"
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid user ID",
		})
		return
	}

	session, err := h.syncService.CreateSyncSession(userID, req.NativeWalletID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create sync session",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"session": session,
		},
	})
}

// ConnectBrowserWallet connects a browser wallet to an existing sync session
// @Summary Connect browser wallet
// @Description Connect a browser wallet to an existing synchronization session
// @Tags wallet-sync
// @Accept json
// @Produce json
// @Param request body ConnectBrowserWalletRequest true "Browser wallet connection request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/sync/browser/connect [post]
func (h *WalletSyncHandler) ConnectBrowserWallet(c *gin.Context) {
	var req ConnectBrowserWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	err := h.syncService.ConnectBrowserWallet(req.SessionID, req.BrowserWalletID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message": "Browser wallet connected successfully",
		},
	})
}

// SyncWallets synchronizes wallet data between platforms
// @Summary Sync wallets
// @Description Synchronize wallet data between native and browser platforms
// @Tags wallet-sync
// @Accept json
// @Produce json
// @Param request body SyncWalletsRequest true "Wallet sync request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/sync/wallets [post]
func (h *WalletSyncHandler) SyncWallets(c *gin.Context) {
	var req SyncWalletsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	response, err := h.syncService.SyncWallets(req.SessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// SyncTransactions synchronizes transaction data between platforms
// @Summary Sync transactions
// @Description Synchronize transaction data between native and browser platforms
// @Tags wallet-sync
// @Accept json
// @Produce json
// @Param request body SyncTransactionsRequest true "Transaction sync request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/sync/transactions [post]
func (h *WalletSyncHandler) SyncTransactions(c *gin.Context) {
	var req SyncTransactionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	response, err := h.syncService.SyncTransactions(req.SessionID, req.WalletAddress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// GetSyncSession retrieves information about a sync session
// @Summary Get sync session
// @Description Get information about an active synchronization session
// @Tags wallet-sync
// @Accept json
// @Produce json
// @Param session_id path string true "Session ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/sync/session/{session_id} [get]
func (h *WalletSyncHandler) GetSyncSession(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Session ID is required",
		})
		return
	}

	session, err := h.syncService.GetActiveSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"session": session,
		},
	})
}

// HandleWebSocket handles WebSocket connections for real-time synchronization
// @Summary WebSocket sync connection
// @Description Establish WebSocket connection for real-time wallet synchronization
// @Tags wallet-sync
// @Param session_id query string true "Session ID"
// @Router /api/v1/sync/ws [get]
func (h *WalletSyncHandler) HandleWebSocket(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Session ID is required",
		})
		return
	}

	// Verify session exists
	_, err := h.syncService.GetActiveSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Session not found",
		})
		return
	}

	// Upgrade to WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to upgrade to WebSocket",
		})
		return
	}

	// Handle WebSocket connection
	h.syncService.HandleWebSocketConnection(sessionID, conn)
}

// CleanupExpiredSessions cleans up expired synchronization sessions
// @Summary Cleanup expired sessions
// @Description Clean up expired synchronization sessions
// @Tags wallet-sync
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/sync/cleanup [post]
func (h *WalletSyncHandler) CleanupExpiredSessions(c *gin.Context) {
	h.syncService.CleanupExpiredSessions()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message": "Expired sessions cleaned up successfully",
		},
	})
}
