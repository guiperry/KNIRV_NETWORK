package gateway

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/knirv/nexus-backend/internal/config"
	"github.com/knirv/nexus-backend/internal/models"
	"github.com/knirv/nexus-backend/pkg/sse"
	"github.com/tidwall/buntdb"
)

// APIGateway manages HTTP API endpoints and SSE connections
type APIGateway struct {
	db          *buntdb.DB
	sseManager  *sse.SSEManager
	config      *config.Config
	authService *AuthService
	rateLimiter *RateLimiter
	userManager *UserManager
}

// AuthService handles authentication
type AuthService struct {
	jwtSecret string
}

// RateLimiter handles rate limiting
type RateLimiter struct {
	// Implementation details
}

// UserManager handles user management
type UserManager struct {
	db *buntdb.DB
}

// NewAPIGateway creates a new API Gateway instance
func NewAPIGateway(db *buntdb.DB, sseManager *sse.SSEManager, cfg *config.Config) (*APIGateway, error) {
	return &APIGateway{
		db:         db,
		sseManager: sseManager,
		config:     cfg,
		authService: &AuthService{
			jwtSecret: cfg.Auth.JWTSecret,
		},
		rateLimiter: &RateLimiter{},
		userManager: &UserManager{
			db: db,
		},
	}, nil
}

// SetupRoutes sets up all API routes
func (gw *APIGateway) SetupRoutes(router *gin.Engine) {
	// Health check
	router.GET("/health", gw.handleHealth)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Authentication routes
		auth := v1.Group("/auth")
		{
			auth.POST("/login", gw.handleLogin)
			auth.POST("/logout", gw.handleLogout)
			auth.POST("/refresh", gw.handleRefreshToken)
		}

		// Protected routes (require authentication)
		protected := v1.Group("/")
		protected.Use(gw.authMiddleware())
		{
			// DVE Node routes
			nodes := protected.Group("/dve-nodes")
			{
				nodes.GET("", gw.handleGetNodes)
				nodes.POST("", gw.handleCreateNode)
				nodes.GET("/:id", gw.handleGetNode)
				nodes.PUT("/:id", gw.handleUpdateNode)
				nodes.DELETE("/:id", gw.handleDeleteNode)
				nodes.GET("/:id/metrics", gw.handleGetNodeMetrics)
			}

			// Validation Task routes
			tasks := protected.Group("/validation-tasks")
			{
				tasks.GET("", gw.handleGetTasks)
				tasks.POST("", gw.handleCreateTask)
				tasks.GET("/:id", gw.handleGetTask)
				tasks.PUT("/:id", gw.handleUpdateTask)
				tasks.DELETE("/:id", gw.handleDeleteTask)
				tasks.POST("/:id/execute", gw.handleExecuteTask)
			}

			// System Health routes
			health := protected.Group("/system")
			{
				health.GET("/health", gw.handleGetSystemHealth)
				health.GET("/metrics", gw.handleGetSystemMetrics)
				health.GET("/alerts", gw.handleGetAlerts)
				health.POST("/alerts/:id/acknowledge", gw.handleAcknowledgeAlert)
			}

			// Report routes
			reports := protected.Group("/reports")
			{
				reports.GET("", gw.handleGetReports)
				reports.POST("", gw.handleGenerateReport)
				reports.GET("/:id", gw.handleGetReport)
				reports.GET("/:id/download", gw.handleDownloadReport)
				reports.POST("/:id/share", gw.handleShareReport)
				reports.DELETE("/:id", gw.handleDeleteReport)
				reports.GET("/templates", gw.handleGetReportTemplates)
				reports.POST("/templates", gw.handleCreateReportTemplate)
			}

			// User Management routes (admin only)
			users := protected.Group("/users")
			users.Use(gw.adminOnlyMiddleware())
			{
				users.GET("", gw.handleGetUsers)
				users.POST("", gw.handleCreateUser)
				users.GET("/:id", gw.handleGetUser)
				users.PUT("/:id", gw.handleUpdateUser)
				users.DELETE("/:id", gw.handleDeleteUser)
			}
		}

		// SSE endpoint
		v1.GET("/sse", gw.handleSSE)
	}
}

// Health check handler
func (gw *APIGateway) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	})
}

// SSE handler
func (gw *APIGateway) handleSSE(c *gin.Context) {
	// Get user ID from token or query parameter
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
		return
	}

	// Create SSE handler for this user
	handler := gw.sseManager.CreateSSEHandler(userID)
	handler(c.Writer, c.Request)
}

// Authentication middleware
func (gw *APIGateway) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Validate token (simplified)
		userID, err := gw.authService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

// Admin only middleware
func (gw *APIGateway) adminOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			c.Abort()
			return
		}

		// Check if user is admin (simplified)
		isAdmin, err := gw.userManager.IsAdmin(userID)
		if err != nil || !isAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Node handlers
func (gw *APIGateway) handleGetNodes(c *gin.Context) {
	// Parse query parameters for filtering
	status := c.Query("status")
	teeType := c.Query("tee_type")
	location := c.Query("location")

	// Get pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// TODO: Implement actual node retrieval logic using filters
	nodes := []models.DVENode{} // Placeholder

	// Use the filter parameters (to avoid unused variable warnings)
	_ = status
	_ = teeType
	_ = location

	c.JSON(http.StatusOK, gin.H{
		"nodes": nodes,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": len(nodes),
		},
	})
}

func (gw *APIGateway) handleCreateNode(c *gin.Context) {
	var req struct {
		Name         string   `json:"name" binding:"required"`
		TEEType      string   `json:"tee_type" binding:"required"`
		StakeAmount  int64    `json:"stake_amount"`
		Location     string   `json:"location"`
		IPAddress    string   `json:"ip_address" binding:"required"`
		PublicKey    string   `json:"public_key" binding:"required"`
		Capabilities []string `json:"capabilities"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Implement actual node creation logic
	node := models.DVENode{
		ID:           "node-" + strconv.FormatInt(time.Now().Unix(), 10),
		Name:         req.Name,
		TEEType:      req.TEEType,
		StakeAmount:  req.StakeAmount,
		Location:     req.Location,
		IPAddress:    req.IPAddress,
		PublicKey:    req.PublicKey,
		Capabilities: req.Capabilities,
		Status:       "online",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Broadcast update via SSE
	gw.sseManager.BroadcastUpdate("dve-nodes", map[string]interface{}{
		"type": "node_created",
		"data": node,
	})

	c.JSON(http.StatusCreated, node)
}

func (gw *APIGateway) handleGetNode(c *gin.Context) {
	nodeID := c.Param("id")

	// TODO: Implement actual node retrieval logic
	node := models.DVENode{
		ID:     nodeID,
		Status: "online",
	}

	c.JSON(http.StatusOK, node)
}

func (gw *APIGateway) handleUpdateNode(c *gin.Context) {
	nodeID := c.Param("id")

	var req struct {
		Status string `json:"status"`
		Name   string `json:"name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Implement actual node update logic

	// Broadcast update via SSE
	gw.sseManager.BroadcastUpdate("dve-nodes", map[string]interface{}{
		"type": "node_updated",
		"data": map[string]interface{}{
			"id":     nodeID,
			"status": req.Status,
			"name":   req.Name,
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "Node updated successfully"})
}

func (gw *APIGateway) handleDeleteNode(c *gin.Context) {
	nodeID := c.Param("id")

	// TODO: Implement actual node deletion logic

	// Broadcast update via SSE
	gw.sseManager.BroadcastUpdate("dve-nodes", map[string]interface{}{
		"type": "node_deleted",
		"data": map[string]interface{}{
			"id": nodeID,
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "Node deleted successfully"})
}

func (gw *APIGateway) handleGetNodeMetrics(c *gin.Context) {
	nodeID := c.Param("id")

	// TODO: Implement actual metrics retrieval logic
	metrics := map[string]interface{}{
		"node_id":         nodeID,
		"cpu_usage":       45.2,
		"memory_usage":    67.8,
		"network_latency": 25,
		"timestamp":       time.Now().Unix(),
	}

	c.JSON(http.StatusOK, metrics)
}

// Authentication handlers
func (gw *APIGateway) handleLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Implement actual authentication logic
	token := "dummy-jwt-token"

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"expires_in": 3600,
		"user": gin.H{
			"id":    "user-123",
			"email": req.Email,
			"role":  "admin",
		},
	})
}

func (gw *APIGateway) handleLogout(c *gin.Context) {
	// TODO: Implement token invalidation
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (gw *APIGateway) handleRefreshToken(c *gin.Context) {
	// TODO: Implement token refresh logic
	c.JSON(http.StatusOK, gin.H{
		"token":      "new-jwt-token",
		"expires_in": 3600,
	})
}

// ValidateToken validates a JWT token and returns the user ID
func (as *AuthService) ValidateToken(token string) (string, error) {
	// TODO: Implement actual JWT validation
	// For now, return a dummy user ID
	if token == "Bearer dummy-jwt-token" {
		return "user-123", nil
	}
	return "", fmt.Errorf("invalid token")
}

// IsAdmin checks if a user is an admin
func (um *UserManager) IsAdmin(userID string) (bool, error) {
	// TODO: Implement actual admin check
	// For now, return true for demo purposes
	return true, nil
}

// Additional handler methods for completeness
func (gw *APIGateway) handleGetTasks(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"tasks": []interface{}{}})
}

func (gw *APIGateway) handleCreateTask(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "Task created"})
}

func (gw *APIGateway) handleGetTask(c *gin.Context) {
	taskID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"task_id": taskID})
}

func (gw *APIGateway) handleUpdateTask(c *gin.Context) {
	taskID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"message": "Task updated", "task_id": taskID})
}

func (gw *APIGateway) handleDeleteTask(c *gin.Context) {
	taskID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"message": "Task deleted", "task_id": taskID})
}

func (gw *APIGateway) handleExecuteTask(c *gin.Context) {
	taskID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"message": "Task execution started", "task_id": taskID})
}

func (gw *APIGateway) handleGetSystemHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"nodes": gin.H{
			"total":  10,
			"active": 8,
		},
	})
}

func (gw *APIGateway) handleGetSystemMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"cpu_usage":       45.2,
		"memory_usage":    67.8,
		"network_latency": 25,
	})
}

func (gw *APIGateway) handleGetAlerts(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"alerts": []interface{}{}})
}

func (gw *APIGateway) handleAcknowledgeAlert(c *gin.Context) {
	alertID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"message": "Alert acknowledged", "alert_id": alertID})
}

func (gw *APIGateway) handleGetReports(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"reports": []interface{}{}})
}

func (gw *APIGateway) handleGenerateReport(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "Report generation started"})
}

func (gw *APIGateway) handleGetReport(c *gin.Context) {
	reportID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"report_id": reportID})
}

func (gw *APIGateway) handleDownloadReport(c *gin.Context) {
	reportID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"download_url": "/downloads/" + reportID})
}

func (gw *APIGateway) handleShareReport(c *gin.Context) {
	reportID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"share_token": "share-" + reportID})
}

func (gw *APIGateway) handleDeleteReport(c *gin.Context) {
	reportID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"message": "Report deleted", "report_id": reportID})
}

func (gw *APIGateway) handleGetReportTemplates(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"templates": []interface{}{}})
}

func (gw *APIGateway) handleCreateReportTemplate(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "Template created"})
}

func (gw *APIGateway) handleGetUsers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"users": []interface{}{}})
}

func (gw *APIGateway) handleCreateUser(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "User created"})
}

func (gw *APIGateway) handleGetUser(c *gin.Context) {
	userID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"user_id": userID})
}

func (gw *APIGateway) handleUpdateUser(c *gin.Context) {
	userID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"message": "User updated", "user_id": userID})
}

func (gw *APIGateway) handleDeleteUser(c *gin.Context) {
	userID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"message": "User deleted", "user_id": userID})
}
