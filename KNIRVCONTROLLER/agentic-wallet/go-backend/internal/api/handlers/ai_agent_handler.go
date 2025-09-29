package handlers

import (
	"crypto-wallet-backend/internal/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AIAgentHandler struct {
	agentService *services.AIAgentService
}

func NewAIAgentHandler(agentService *services.AIAgentService) *AIAgentHandler {
	return &AIAgentHandler{
		agentService: agentService,
	}
}

func (h *AIAgentHandler) GetUserAgents(c *gin.Context) {
	userID := c.GetString("user_id")
	uid, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	agents, err := h.agentService.GetUserAgents(c.Request.Context(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"agents": agents})
}

func (h *AIAgentHandler) CreateAgent(c *gin.Context) {
	userID := c.GetString("user_id")
	uid, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req services.CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	agent, err := h.agentService.CreateAgent(c.Request.Context(), uid, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"agent": agent})
}

func (h *AIAgentHandler) ExecuteAgent(c *gin.Context) {
	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	var req services.ExecuteAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	execution, err := h.agentService.ExecuteAgent(c.Request.Context(), agentID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"execution": execution})
}

func (h *AIAgentHandler) GetMarketplaceAgents(c *gin.Context) {
	category := c.Query("category")
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit"})
		return
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid offset"})
		return
	}

	agents, err := h.agentService.GetMarketplaceAgents(c.Request.Context(), category, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"agents": agents})
}

func (h *AIAgentHandler) ActivateAgent(c *gin.Context) {
	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	if err := h.agentService.ActivateAgent(c.Request.Context(), agentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Agent activated successfully"})
}

func (h *AIAgentHandler) DeactivateAgent(c *gin.Context) {
	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	if err := h.agentService.DeactivateAgent(c.Request.Context(), agentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Agent deactivated successfully"})
}

func (h *AIAgentHandler) GetAgent(c *gin.Context) {
	// Implementation for getting a specific agent
	c.JSON(http.StatusOK, gin.H{"message": "Get agent endpoint"})
}

func (h *AIAgentHandler) GetExecutions(c *gin.Context) {
	// Implementation for getting agent executions
	c.JSON(http.StatusOK, gin.H{"message": "Get executions endpoint"})
}

func (h *AIAgentHandler) GetTrades(c *gin.Context) {
	// Implementation for getting agent trades
	c.JSON(http.StatusOK, gin.H{"message": "Get trades endpoint"})
}