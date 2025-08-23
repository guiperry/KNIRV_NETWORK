package api

import (
	"crypto-wallet-backend/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MultichainWalletHandler handles multichain wallet API endpoints
type MultichainWalletHandler struct {
	walletService *services.MultichainWalletService
}

// CreateWalletRequest represents the request to create a new wallet
type CreateWalletRequest struct {
	Name     string   `json:"name" binding:"required"`
	Chains   []string `json:"chains" binding:"required"`
	Mnemonic string   `json:"mnemonic,omitempty"`
}

// ImportWalletRequest represents the request to import a wallet from private key
type ImportWalletRequest struct {
	Name       string `json:"name" binding:"required"`
	Chain      string `json:"chain" binding:"required"`
	PrivateKey string `json:"private_key" binding:"required"`
}

// GenerateMnemonicRequest represents the request to generate a mnemonic
type GenerateMnemonicRequest struct {
	Size int `json:"size,omitempty"`
}

// GetSupportedChains returns all supported blockchain networks
// @Summary Get supported chains
// @Description Get list of all supported blockchain networks
// @Tags multichain
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/multichain/chains [get]
func (h *MultichainWalletHandler) GetSupportedChains(c *gin.Context) {
	chains := h.walletService.GetSupportedChains()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"chains": chains,
		},
	})
}

// GenerateMnemonic generates a new mnemonic phrase
// @Summary Generate mnemonic
// @Description Generate a new mnemonic phrase for wallet creation
// @Tags multichain
// @Accept json
// @Produce json
// @Param request body GenerateMnemonicRequest true "Mnemonic generation request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/v1/multichain/mnemonic/generate [post]
func (h *MultichainWalletHandler) GenerateMnemonic(c *gin.Context) {
	var req GenerateMnemonicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	if req.Size == 0 {
		req.Size = 12 // Default to 12 words
	}

	mnemonic, err := h.walletService.GenerateMnemonic(req.Size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to generate mnemonic",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"mnemonic": mnemonic,
			"size":     req.Size,
		},
	})
}

// CreateMultichainWallet creates wallets for multiple chains from a single mnemonic
// @Summary Create multichain wallet
// @Description Create wallets for multiple blockchain networks from a single mnemonic
// @Tags multichain
// @Accept json
// @Produce json
// @Param request body CreateWalletRequest true "Wallet creation request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/multichain/wallet/create [post]
func (h *MultichainWalletHandler) CreateMultichainWallet(c *gin.Context) {
	var req CreateWalletRequest
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
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid user ID",
		})
		return
	}

	// Generate mnemonic if not provided
	mnemonic := req.Mnemonic
	if mnemonic == "" {
		mnemonic, err = h.walletService.GenerateMnemonic(12)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to generate mnemonic",
			})
			return
		}
	}

	wallets, err := h.walletService.CreateMultichainWallet(userID, req.Name, mnemonic, req.Chains)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create wallets",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"wallets":  wallets,
			"mnemonic": mnemonic,
		},
	})
}

// ImportWallet imports a wallet from a private key
// @Summary Import wallet from private key
// @Description Import a wallet from a private key for a specific blockchain
// @Tags multichain
// @Accept json
// @Produce json
// @Param request body ImportWalletRequest true "Wallet import request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/multichain/wallet/import [post]
func (h *MultichainWalletHandler) ImportWallet(c *gin.Context) {
	var req ImportWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	// Get user ID from context
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid user ID",
		})
		return
	}

	wallet, err := h.walletService.ImportWalletFromPrivateKey(userID, req.Name, req.PrivateKey, req.Chain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to import wallet",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"wallet": wallet,
		},
	})
}

// GenerateWalletForChain generates a wallet for a specific chain
// @Summary Generate wallet for specific chain
// @Description Generate a wallet address and private key for a specific blockchain
// @Tags multichain
// @Accept json
// @Produce json
// @Param chain path string true "Blockchain symbol (BTC, ETH, SOL, etc.)"
// @Param mnemonic query string true "Mnemonic phrase"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/multichain/wallet/generate/{chain} [post]
func (h *MultichainWalletHandler) GenerateWalletForChain(c *gin.Context) {
	chain := c.Param("chain")
	mnemonic := c.Query("mnemonic")

	if chain == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chain parameter is required",
		})
		return
	}

	if mnemonic == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Mnemonic parameter is required",
		})
		return
	}

	walletResult, err := h.walletService.GenerateWalletForChain(mnemonic, chain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to generate wallet for chain",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"chain":       chain,
			"address":     walletResult.Address,
			"private_key": walletResult.PrivateKey,
		},
	})
}

// GetWalletBalance retrieves the balance for a wallet
// @Summary Get wallet balance
// @Description Get the balance for a wallet on a specific blockchain
// @Tags multichain
// @Accept json
// @Produce json
// @Param chain path string true "Blockchain symbol"
// @Param address path string true "Wallet address"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/multichain/balance/{chain}/{address} [get]
func (h *MultichainWalletHandler) GetWalletBalance(c *gin.Context) {
	chain := c.Param("chain")
	address := c.Param("address")

	if chain == "" || address == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chain and address parameters are required",
		})
		return
	}

	balance, err := h.walletService.GetWalletBalance(address, chain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get wallet balance",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"chain":   chain,
			"address": address,
			"balance": balance,
		},
	})
}

// NewMultichainWalletHandler creates a new multichain wallet handler
func NewMultichainWalletHandler(walletService *services.MultichainWalletService) *MultichainWalletHandler {
	return &MultichainWalletHandler{
		walletService: walletService,
	}
}
