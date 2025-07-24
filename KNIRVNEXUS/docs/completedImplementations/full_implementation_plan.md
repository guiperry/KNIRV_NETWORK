# 🎯 Full Implementation Plan

## 📋 **Overview**
This document outlines the comprehensive implementation plan for the Inc-Line Inference Engine, combining both frontend and backend development requirements. It includes the detailed implementation plan for missing front-end functionality identified during a comprehensive audit, as well as the refactoring plan for the existing Golang inference backend to match the new JavaScript GUI frontend features.

For the detailed Google Agent Development Kit (ADK) integration plan, please refer to the companion document `adk_integration_plan.md`, which provides a comprehensive approach to integrating ADK with our existing Inference Engine.

## 🔍 **Current Architecture Analysis**

### Frontend (JavaScript/React)
- **UI Framework**: React with TypeScript
- **Styling**: TailwindCSS
- **Key Components**:
  - Dashboard: Overview of agent activities and system status
  - AgentManager: Management of NFT-Agents
  - CapabilityStore: Repository of agent capabilities
  - TargetManager: Management of target systems
  - InferenceOrchestrator: Orchestration of inference workflows
  - Analytics: Visualization of inference data
  - Settings: System configuration

### Backend (Golang)
- **Core Services**:
  - InferenceService: Manages LLM interactions
  - DelegatorService: Handles request delegation between LLMs
  - ContextManager: Manages chunking and processing of large inputs
  - ConversationMemory: Stores conversation history
  - WordPressService: Connects to WordPress (legacy)

## 🔎 **Gap Analysis**

1. **Agent Management**:
   - Frontend shows NFT-Agent management
   - Backend has no concept of agents, only LLM providers

2. **Target Systems**:
   - Frontend displays target system management
   - Backend only has WordPress integration

3. **Capabilities**:
   - Frontend shows capability management
   - Backend has inference capabilities but not as modular components

4. **Orchestration**:
   - Frontend shows workflow orchestration
   - Backend has basic delegation but no complex workflow support

5. **Analytics**:
   - Frontend shows analytics dashboards
   - Backend has minimal logging but no analytics

6. **Persistence**:
   - Frontend assumes persistent storage of entities
   - Backend uses in-memory storage with WordPress as the only external system

7. **Authentication**:
   - Frontend has user management UI
   - Backend lacks proper authentication system

## 🔴 **HIGH PRIORITY IMPLEMENTATIONS**

### 1. **Agent Creation Flow** 
**Status**: ❌ Not Implemented  
**Priority**: Critical  
**Estimated Effort**: 2-3 days  

**Requirements**:
- Modal dialog for agent creation
- Form fields: name, collection, capabilities, target types
- Integration with backend API POST `/api/v1/agents`
- Image upload/selection functionality
- Validation and error handling

**Implementation Steps**:
1. Create `AgentCreationModal.jsx` component
2. Add form validation with React Hook Form
3. Implement image upload/selection
4. Connect to backend API
5. Add success/error notifications
6. Update agent list after creation

**API Endpoints Needed**:
- `POST /api/v1/agents` ✅ (Already exists)
- `GET /api/v1/capabilities` (Need to create)
- `GET /api/v1/target-types` (Need to create)

---

### 2. **Agent Deployment System**
**Status**: ❌ Not Implemented  
**Priority**: Critical  
**Estimated Effort**: 3-4 days  

**Requirements**:
- Target system selection
- Capability assignment
- Deployment configuration
- Real-time deployment status
- Error handling and rollback

**Implementation Steps**:
1. Create `AgentDeploymentModal.jsx` component
2. Add target system selection dropdown
3. Implement capability configuration
4. Add deployment progress tracking
5. Connect to backend deployment API
6. Update agent status in real-time

**API Endpoints Needed**:
- `POST /api/v1/agents/{id}/deploy` (Need to create)
- `GET /api/v1/target-systems` (Need to create)
- `POST /api/v1/agents/{id}/stop` (Need to create)

---

### 3. **Agent Control (Start/Stop)**
**Status**: ❌ Not Implemented  
**Priority**: Critical  
**Estimated Effort**: 1-2 days  

**Requirements**:
- Start/stop agent functionality
- Status updates in real-time
- Confirmation dialogs for destructive actions
- Error handling

**Implementation Steps**:
1. Add agent control functions to AgentManager
2. Implement confirmation dialogs
3. Connect to backend control APIs
4. Add real-time status updates
5. Handle error states gracefully

**API Endpoints Needed**:
- `POST /api/v1/agents/{id}/start` (Need to create)
- `POST /api/v1/agents/{id}/stop` (Need to create)
- `GET /api/v1/agents/{id}/status` (Need to create)

---

### 4. **Agent Configuration Interface**
**Status**: ❌ Not Implemented  
**Priority**: Critical  
**Estimated Effort**: 2-3 days  

**Requirements**:
- Agent settings configuration
- Capability management
- Target system preferences
- Performance tuning options

**Implementation Steps**:
1. Create `AgentConfigModal.jsx` component
2. Add configuration form sections
3. Implement capability management
4. Add target system preferences
5. Connect to backend configuration API

**API Endpoints Needed**:
- `GET /api/v1/agents/{id}/config` (Need to create)
- `PUT /api/v1/agents/{id}/config` (Need to create)

---

### 5. **WordPress Deprecation and Removal**
**Status**: ❌ Not Implemented  
**Priority**: Critical  
**Estimated Effort**: 1 week  

**Requirements**:
- Identify all WordPress dependencies
- Create deprecation interfaces
- Remove WordPress service

**Implementation Steps**:
1. Audit codebase for all WordPress service references
2. Document all WordPress-dependent functionality
3. Create migration plan for each WordPress-dependent feature
4. Implement adapter interfaces for WordPress functionality
5. Add deprecation warnings to WordPress-related code
6. Create stubs for replacement functionality
7. Remove WordPress service imports from main.go
8. Delete WordPress-related UI components
9. Remove WordPress service package entirely
10. Update documentation to reflect removal

**Code Example**:
```go
// DeprecatedWordPressService wraps the legacy WordPress service with warnings
type DeprecatedWordPressService struct {
    inner *wordpress.WordPressService
}

func (d *DeprecatedWordPressService) Connect(url, username, password string) error {
    log.Println("WARNING: Using deprecated WordPress service. This functionality will be removed in a future version.")
    return d.inner.Connect(url, username, password)
}

// Additional methods with deprecation warnings...
```

---

### 6. **Database Implementation**
**Status**: ❌ Not Implemented  
**Priority**: Critical  
**Estimated Effort**: 2 weeks  

**Requirements**:
- Set up SQLite database for authentication
- Implement chromem-go for domain entities persistence (as detailed in `adk_integration_plan.md`)
- Create database management tools

**Implementation Steps**:
1. Implement SQLite database for user authentication
2. Create schema for users, roles, and permissions
3. Implement repository layer for auth entities
4. Add migration tools for schema updates
5. Implement chromem-go for domain entities persistence (leveraging the `AgentRegistry` from ADK integration)
6. Create collections for agents, targets, capabilities, and workflows
7. Implement repository layer for domain entities
8. Add data migration and synchronization tools
9. Create migration framework for SQLite schema evolution
10. Implement initial schema creation scripts for authentication database
11. Add version tracking for database schemas
12. Develop chromem-go collection management and backup functionality

**Code Examples**:
```go
// AuthDB manages the authentication database
type AuthDB struct {
    db *sql.DB
}

// NewAuthDB creates a new authentication database connection
func NewAuthDB(dbPath string) (*AuthDB, error) {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, fmt.Errorf("failed to open auth database: %w", err)
    }
    
    // Initialize schema if needed
    if err := initAuthSchema(db); err != nil {
        db.Close()
        return nil, err
    }
    
    return &AuthDB{db: db}, nil
}
```

```go
// DomainDB manages the domain entity database using chromem-go
type DomainDB struct {
    db          *chromem.DB
    collections map[string]*chromem.Collection
}

// NewDomainDB creates a new chromem-go database for domain entities
func NewDomainDB(persistencePath string) (*DomainDB, error) {
    // Configure chromem-go with optional persistence
    opts := &chromem.DBOpts{
        PersistencePath: persistencePath,
    }
    
    // Create new database
    db := chromem.NewDBWithOpts(opts)
    
    domainDB := &DomainDB{
        db:          db,
        collections: make(map[string]*chromem.Collection),
    }
    
    // Initialize collections if needed
    if err := domainDB.initCollections(); err != nil {
        return nil, err
    }
    
    return domainDB, nil
}
```

---

### 7. **Core Backend API Refactoring**
**Status**: ❌ Not Implemented  
**Priority**: Critical  
**Estimated Effort**: 2 weeks  

**Requirements**:
- Create RESTful API layer
- Refactor InferenceService for agent-based architecture using ADK integration

**Implementation Steps**:
1. Implement a new HTTP server in Golang using standard library or Gin
2. Define API endpoints that match frontend requirements
3. Implement CORS support for local development
4. Add JWT authentication middleware
5. Modify InferenceService to support agent-based architecture
6. Add support for agent metadata and NFT properties
7. Implement agent state management (using ADK's state management system)
8. Connect to domain database for persistence (using chromem-go as detailed in `adk_integration_plan.md`)
9. Implement cross-compilation support using the Makefile and shell script from `adk_integration_plan.md`

**Code Examples**:
```go
// Example API structure with authentication
func setupRouter(authService *AuthService) *gin.Engine {
    router := gin.Default()
    
    // CORS middleware
    router.Use(cors.Default())
    
    // Public routes
    public := router.Group("/api/v1")
    {
        public.POST("/auth/login", authService.Login)
        public.POST("/auth/register", authService.Register)
    }
    
    // Protected routes
    protected := router.Group("/api/v1")
    protected.Use(authMiddleware(authService))
    {
        // Agent endpoints
        protected.GET("/agents", getAgents)
        protected.POST("/agents", createAgent)
        protected.GET("/agents/:id", getAgent)
        protected.PUT("/agents/:id", updateAgent)
        protected.DELETE("/agents/:id", deleteAgent)
        
        // Target endpoints
        protected.GET("/targets", getTargets)
        protected.POST("/targets", createTarget)
        // ...
        
        // Capability endpoints
        protected.GET("/capabilities", getCapabilities)
        // ...
        
        // Orchestration endpoints
        protected.POST("/orchestrate", orchestrateWorkflow)
        protected.GET("/orchestration/:id", getOrchestrationStatus)
        // ...
        
        // Analytics endpoints
        protected.GET("/analytics/summary", getAnalyticsSummary)
        // ...
    }
    
    return router
}
```

```go
// Agent represents an NFT-Agent
type Agent struct {
    ID           string    `json:"id" db:"id"`
    Name         string    `json:"name" db:"name"`
    Collection   string    `json:"collection" db:"collection"`
    ImageURL     string    `json:"image_url" db:"image_url"`
    Status       string    `json:"status" db:"status"`
    Capabilities []string  `json:"capabilities" db:"-"` // Handled separately in DB
    CreatedAt    time.Time `json:"created_at" db:"created_at"`
    // NFT-specific properties
    TokenID      string    `json:"token_id" db:"token_id"`
    ContractAddr string    `json:"contract_addr" db:"contract_addr"`
    // Owner information
    OwnerID      int64     `json:"owner_id" db:"owner_id"`
}
```

---

## 🟡 **MEDIUM PRIORITY IMPLEMENTATIONS**

### 8. **Target System Configuration**
**Status**: ❌ Not Implemented  
**Priority**: Important  
**Estimated Effort**: 2-3 days  

**Requirements**:
- Add new target systems
- Configure connection settings
- Test connections
- Manage permissions

**Implementation Steps**:
1. Create target system data models with database mappings
2. Implement target system repository for SQLite persistence
3. Add target system service layer with business logic

**Code Example**:
```go
// TargetSystem represents a system that agents can interact with
type TargetSystem struct {
    ID           string    `json:"id" db:"id"`
    Name         string    `json:"name" db:"name"`
    Type         string    `json:"type" db:"type"` // browser, filesystem, application, etc.
    Status       string    `json:"status" db:"status"`
    Capabilities []string  `json:"capabilities" db:"-"` // Handled separately in DB
    LastActivity time.Time `json:"last_activity" db:"last_activity"`
    // Owner information
    OwnerID      int64     `json:"owner_id" db:"owner_id"`
}
```

### 9. **MCP Capability Management**
**Status**: ❌ Not Implemented  
**Priority**: Important  
**Estimated Effort**: 2-3 days  

**Requirements**:
- KNIRVCHAIN Service implementation
- Capability ingestion and management
- Automated MCP server discovery and connection
- Health monitoring and diagnostics
- Capability versioning and compatibility checks

**Implementation Steps**:
1. Create `MCPCapabilityManager.jsx` component
2. Integrate KNIRVCHAIN SDK for capability ingestion
3. Implement capability browser and selection interface
4. Add automated MCP server management features
5. Create health monitoring dashboard
6. Implement capability version control
7. Create capability data models with database mappings
8. Implement capability repository for SQLite persistence
9. Add capability service layer with business logic

**Technical Components**:
- **KNIRVCHAIN SDK Integration**: Utilize the included SDK to handle capability ingestion
- **Automated Discovery**: Leverage SDK's built-in MCP server management functionality
- **Capability Registry**: Create UI for browsing and managing available capabilities
- **Health Metrics**: Implement real-time monitoring of capability performance
- **Version Control**: Add capability versioning and compatibility verification

**API Endpoints Needed**:
- `GET /api/v1/capabilities` (Need to create)
- `POST /api/v1/capabilities/register` (Need to create)
- `GET /api/v1/capabilities/health` (Need to create)
- `PUT /api/v1/capabilities/version` (Need to create)

**Code Example**:
```go
// Capability represents a function that agents can perform
type Capability struct {
    ID            string    `json:"id" db:"id"`
    Name          string    `json:"name" db:"name"`
    Provider      string    `json:"provider" db:"provider"`
    Type          string    `json:"type" db:"type"`
    EstimatedTime string    `json:"estimated_time" db:"estimated_time"`
    Description   string    `json:"description" db:"description"`
    // System capability or user-defined
    System        bool      `json:"system" db:"system"`
    // Owner information (null for system capabilities)
    OwnerID       *int64    `json:"owner_id" db:"owner_id"`
}
```

### 10. **Advanced Filter System**
**Status**: ❌ Not Implemented  
**Priority**: Important  
**Estimated Effort**: 1-2 days  

**Requirements**:
- Multi-criteria filtering
- Saved filter presets
- Advanced search options
- Filter persistence

### 11. **Agent Detail Views**
**Status**: ❌ Not Implemented  
**Priority**: Important  
**Estimated Effort**: 2-3 days  

**Requirements**:
- Detailed agent information
- Performance metrics
- Activity history
- Configuration details

### 12. **Data Engineering Architectural Interface**
**Status**: ❌ Not Implemented  
**Priority**: Important  
**Estimated Effort**: 4-5 days  

**Requirements**:
- Interactive dialogue for configuring data engineering pipelines
- Real-time data flow visualization
- Component configuration interface
- Monitoring and alerting setup
- Real-time dashboard for system metrics and performance
- Customizable views and filters
- Historical data analysis
- Alert notifications and status indicators

**Implementation Steps**:
1. Create `DataEngineeringModal.jsx` component
2. Implement pipeline configuration interface
3. Add component selection and configuration
4. Create real-time visualization of data flow
5. Implement monitoring and alerting setup
6. Add dashboard views for system-wide monitoring
7. Implement historical data visualization

**Technical Components**:
- **Event Ingestion**: Configure real-time clickstream or user interaction data using Kafka producers (Go package: `github.com/segmentio/kafka-go v0.4.47`) or Python scripts sending JSON to Kafka topics
- **Stream Processing**: Set up Apache Kafka + Kafka Streams (Go package: `github.com/confluentinc/confluent-kafka-go/v2 v2.3.0`) or Apache Flink (Go package: `github.com/apache/flink-kubernetes-operator v1.7.0`) for data cleaning, filtering, and enrichment
- **Windowed Aggregations**: Configure sliding windows for metrics calculation (page views per minute, bounce rate, active users)
- **Output & Serving**: Configure data sinks to Chromem-go (Go package: `github.com/philippgille/chromem-go v0.7.0`) for fast lookup or push updates to real-time dashboards via WebSockets or REST APIs
- **Real-Time Alerting**: Set up rule-based engine or anomaly detector (Go package: `github.com/prometheus/alertmanager v0.26.0`) to trigger alerts for error spikes or latency issues
- **Dashboard Visualization**: Implement interactive charts and graphs for system monitoring

**API Endpoints Needed**:
- `POST /api/v1/data-pipelines` (Need to create)
- `GET /api/v1/data-components` (Need to create)
- `POST /api/v1/data-alerts` (Need to create)
- `GET /api/v1/metrics` (Need to create)
- `GET /api/v1/metrics/history` (Need to create)

### 13. **Google A2A Protocol Implementation**
**Status**: ❌ Not Implemented  
**Priority**: Important  
**Estimated Effort**: 3-4 days  

**Requirements**:
- Agent-to-Agent communication protocol implementation
- Agent discovery and capability negotiation
- Secure message exchange
- Task delegation and collaboration
- Support for synchronous and asynchronous communication

**Implementation Steps**:
1. Create `A2AProtocolManager.jsx` component
2. Implement Agent Card creation and discovery
3. Add message exchange interface
4. Implement task delegation and status tracking
5. Create capability negotiation interface

**Technical Components**:
- **Agent Cards**: Implement standardized agent capability descriptions
- **JSON-RPC 2.0**: Set up communication over HTTP(S)
- **Flexible Interaction**: Support synchronous request/response, streaming (SSE), and asynchronous push notifications
- **Rich Data Exchange**: Handle text, files, and structured JSON data
- **Security & Authentication**: Implement enterprise-ready security measures

**API Endpoints Needed**:
- `GET /api/v1/agents/discover` (Need to create)
- `POST /api/v1/agents/message` (Need to create)
- `GET /api/v1/agents/tasks/{id}` (Need to create)
- `POST /api/v1/agents/capabilities` (Need to create)

**Google Agent Development Kit (ADK) Integration**:
- Implement agent orchestration using ADK patterns as detailed in `adk_integration_plan.md`
- Support for flexible workflows (Sequential, Parallel, Loop) via ADK's agent composition
- Enable multi-agent architecture for complex tasks through chromem-go persistence
- Integrate with tool ecosystem for enhanced capabilities using ADK's tool system
- Implement built-in evaluation mechanisms with ADK callbacks
- Single binary compilation with embedded Python runtime

### 14. **Web Connections**
**Status**: ❌ Not Implemented  
**Priority**: Important  
**Estimated Effort**: 2-3 days  

**Requirements**:
- Integration with third-party services from established corporate brands
- Authentication and authorization via picaos.com auth tool
- Secure credential management
- Connection status monitoring
- OAuth 2.0 flow implementation
- API key management

**Implementation Steps**:
1. Create `WebConnectionsManager.jsx` component
2. Implement OAuth 2.0 authorization flows
3. Add API key management interface
4. Create connection status dashboard
5. Implement secure credential storage integration
6. Add connection testing functionality

**Technical Components**:
- **Picaos.com Auth Integration**: Utilize backend auth tool for secure third-party connections
- **OAuth Handler**: Implement standard OAuth 2.0 flows for various service providers
- **Credential Vault**: Create secure interface for managing API keys and tokens
- **Connection Registry**: Develop UI for managing and monitoring active connections
- **Service Discovery**: Implement automated service capability detection

**API Endpoints Needed**:
- `GET /api/v1/web-connections` (Need to create)
- `POST /api/v1/web-connections` (Need to create)
- `DELETE /api/v1/web-connections/{id}` (Need to create)
- `GET /api/v1/web-connections/oauth/callback` (Need to create)
- `POST /api/v1/web-connections/test` (Need to create)

### 15. **Workflow Orchestration Engine**
**Status**: ❌ Not Implemented  
**Priority**: Important  
**Estimated Effort**: 2 weeks  

**Requirements**:
- Create workflow data models with database mappings
- Implement workflow repository for SQLite persistence
- Implement workflow execution engine
- Add support for sequential and parallel execution
- Enhance DelegatorService for workflow-based delegation

**Implementation Steps**:
1. Create workflow data models with database mappings
2. Implement workflow repository for SQLite persistence
3. Implement workflow execution engine
4. Add support for sequential and parallel execution
5. Modify DelegatorService to support workflow-based delegation
6. Add support for capability-specific execution
7. Implement target system integration
8. Store execution history in database

**Code Example**:
```go
// Workflow represents an orchestration workflow
type Workflow struct {
    ID           string    `json:"id" db:"id"`
    AgentID      string    `json:"agent_id" db:"agent_id"`
    TargetID     string    `json:"target_id" db:"target_id"`
    CapabilityID string    `json:"capability_id" db:"capability_id"`
    Status       string    `json:"status" db:"status"`
    StartTime    time.Time `json:"start_time" db:"start_time"`
    EndTime      time.Time `json:"end_time,omitempty" db:"end_time"`
    Result       string    `json:"result,omitempty" db:"result"`
    // Owner information
    OwnerID      int64     `json:"owner_id" db:"owner_id"`
}
```

### 16. **Analytics and Monitoring**
**Status**: ❌ Not Implemented  
**Priority**: Important  
**Estimated Effort**: 2 weeks  

**Requirements**:
- Create analytics data models with database mappings
- Implement analytics repository for SQLite persistence
- Implement data collection mechanisms
- Add aggregation and reporting functions
- Create monitoring data models with database mappings
- Implement real-time monitoring with WebSocket support
- Add alerting mechanisms
- Implement system health checks

**Code Examples**:
```go
// AnalyticsService collects and processes analytics data
type AnalyticsService struct {
    repository     *AnalyticsRepository
    workflowEngine *WorkflowEngine
    agentService   *AgentService
    mutex          sync.RWMutex
}

// AnalyticsSummary provides an overview of system activity
type AnalyticsSummary struct {
    ActiveAgents    int     `json:"active_agents"`
    TargetSystems   int     `json:"target_systems"`
    InferencesToday int     `json:"inferences_today"`
    SuccessRate     float64 `json:"success_rate"`
    // Additional metrics
    TotalWorkflows  int     `json:"total_workflows"`
    AvgDuration     float64 `json:"avg_duration_ms"`
    TopCapabilities []struct {
        Name  string `json:"name"`
        Count int    `json:"count"`
    } `json:"top_capabilities"`
}
```

```go
// MonitoringService provides real-time system monitoring
type MonitoringService struct {
    repository *MonitoringRepository
    clients    map[string]*websocket.Conn
    mutex      sync.RWMutex
}

// SystemHealth represents the current health of the system
type SystemHealth struct {
    Status           string    `json:"status"` // "healthy", "degraded", "unhealthy"
    Components       []string  `json:"components"`
    LastChecked      time.Time `json:"last_checked"`
    ActiveWorkflows  int       `json:"active_workflows"`
    SystemLoad       float64   `json:"system_load"`
    MemoryUsage      float64   `json:"memory_usage"`
    DatabaseLatency  float64   `json:"database_latency_ms"`
    InferenceLatency float64   `json:"inference_latency_ms"`
}
```

### 17. **Authentication and Authorization**
**Status**: ❌ Not Implemented  
**Priority**: Important  
**Estimated Effort**: 1 week  

**Requirements**:
- Create user authentication service
- Implement JWT token generation and validation
- Add password hashing and verification
- Implement user registration and login endpoints
- Create role-based access control system
- Implement permission checking middleware
- Add user role management
- Implement resource ownership validation

**Code Examples**:
```go
// AuthService handles user authentication
type AuthService struct {
    repository *UserRepository
    jwtSecret  []byte
}

// Login authenticates a user and returns a JWT token
func (s *AuthService) Login(c *gin.Context) {
    var credentials struct {
        Username string `json:"username" binding:"required"`
        Password string `json:"password" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&credentials); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
        return
    }
    
    // Get user from database
    user, err := s.repository.GetUserByUsername(credentials.Username)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
        return
    }
    
    // Verify password
    if !s.verifyPassword(user, credentials.Password) {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
        return
    }
    
    // Generate JWT token
    token, err := s.generateToken(user)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "token": token,
        "user": gin.H{
            "id":       user.ID,
            "username": user.Username,
            "email":    user.Email,
            "role":     user.Role,
        },
    })
}
```

```go
// Permission represents an action that can be performed
type Permission string

const (
    PermissionReadAgent       Permission = "agent:read"
    PermissionCreateAgent     Permission = "agent:create"
    PermissionUpdateAgent     Permission = "agent:update"
    PermissionDeleteAgent     Permission = "agent:delete"
    PermissionExecuteWorkflow Permission = "workflow:execute"
    // Additional permissions...
)

// RequirePermission creates middleware that checks for a specific permission
func (s *AuthorizationService) RequirePermission(permission Permission) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Get user ID from context (set by auth middleware)
        userID, exists := c.Get("userID")
        if !exists {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
            return
        }
        
        // Check permission
        hasPermission, err := s.HasPermission(userID.(int64), permission)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to check permission"})
            return
        }
        
        if !hasPermission {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
            return
        }
        
        c.Next()
    }
}
```

---

## 🟢 **LOW PRIORITY IMPLEMENTATIONS**

### 18. **Context Menus**
**Status**: ❌ Not Implemented  
**Priority**: Nice to Have  
**Estimated Effort**: 1 day  

### 19. **Security Settings**
**Status**: ❌ Not Implemented  
**Priority**: Nice to Have  
**Estimated Effort**: 2-3 days  

### 20. **Integration and Testing**
**Status**: ❌ Not Implemented  
**Priority**: Nice to Have  
**Estimated Effort**: 2 weeks  

**Requirements**:
- Update frontend to use real API endpoints
- Implement authentication flow in frontend
- Add error handling and loading states
- Create WebSocket connections for real-time updates
- Create test suite for API endpoints
- Implement integration tests for workflows
- Add performance testing
- Implement database testing with test fixtures
- Update API documentation with OpenAPI/Swagger
- Create user guides
- Document system architecture
- Add developer documentation for extending the system

**Code Example**:
```go
// TestAgentAPI tests the agent API endpoints
func TestAgentAPI(t *testing.T) {
    // Set up test environment
    router, cleanup := setupTestEnvironment()
    defer cleanup()
    
    // Create test client
    client := &http.Client{}
    
    // Login to get token
    token := loginTestUser(t, client, router)
    
    // Test creating an agent
    agent := &Agent{
        Name:       "Test Agent",
        Collection: "Test Collection",
        ImageURL:   "https://example.com/image.jpg",
        Status:     "idle",
    }
    
    // Serialize agent to JSON
    agentJSON, err := json.Marshal(agent)
    require.NoError(t, err)
    
    // Create request
    req, err := http.NewRequest("POST", "/api/v1/agents", bytes.NewBuffer(agentJSON))
    require.NoError(t, err)
    
    // Add authorization header
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Content-Type", "application/json")
    
    // Execute request
    resp := httptest.NewRecorder()
    router.ServeHTTP(resp, req)
    
    // Check response
    require.Equal(t, http.StatusCreated, resp.Code)
    
    // Parse response
    var createdAgent Agent
    err = json.Unmarshal(resp.Body.Bytes(), &createdAgent)
    require.NoError(t, err)
    
    // Verify agent was created with ID
    require.NotEmpty(t, createdAgent.ID)
    require.Equal(t, agent.Name, createdAgent.Name)
}
```

### 21. **Deployment and DevOps**
**Status**: ❌ Not Implemented  
**Priority**: Nice to Have  
**Estimated Effort**: 1 week  

**Requirements**:
- Set up GitHub Actions for continuous integration
- Implement automated testing
- Add deployment automation
- Implement database migration in CI/CD
- Set up centralized logging
- Implement metrics collection
- Add alerting for production issues
- Create dashboards for system monitoring

---

## 📊 **Implementation Timeline**

### **Phase 0: WordPress Removal and Initial Setup (Week 1)**
- Identify and document WordPress dependencies
- Create deprecation interfaces
- Remove WordPress service
- Set up development environment
- Install required dependencies

### **Phase 1: Database Implementation (Week 2-3)**
- Set up SQLite authentication database
- Set up chromem-go for domain entities
- Implement repositories for both databases
- Create database migrations and collection management

### **Phase 2: Core Frontend Features (Week 4-5)**
- Day 1-2: Agent Creation Flow
- Day 3-4: Agent Control (Start/Stop)
- Day 5: Testing and bug fixes
- Day 6-8: Agent Deployment System
- Day 9-10: Agent Configuration Interface

### **Phase 3: Core Backend API (Week 6-7)**
- Set up HTTP server
- Implement authentication
- Implement basic API endpoints
- Add CORS support
- Implement domain models for agents, targets, and capabilities

### **Phase 4: System Management and Advanced Features (Week 8-9)**
- Day 1-2: Target System Configuration
- Day 3: MCP Server Management
- Day 4-5: Advanced Filters and Agent Detail Views
- Day 6-10: Workflow Orchestration Engine

### **Phase 5: Data Engineering and A2A Protocol (Week 10-11)**
- Day 1-3: Data Engineering Architectural Interface
- Day 3-5: Google A2A Protocol Implementation
- Day 6-10: Analytics and Monitoring

### **Phase 6: Authentication and Web Connections (Week 12)**
- Day 1-5: Authentication and Authorization
- Day 6-10: Web Connections

### **Phase 7: Integration and Testing (Week 13-14)**
- Integrate frontend with backend
- Implement end-to-end testing
- Create documentation
- Day 1-2: Integration of all components
- Day 3: Context Menus and Security Settings
- Day 4-5: Final testing and optimization

### **Phase 8: Deployment and DevOps (Week 15)**
- Set up containerization
- Implement CI/CD pipeline
- Configure monitoring and logging

---

## 🔧 **Technical Requirements**

### **Frontend Dependencies**:
- React Hook Form (for form management)
- React Query (for API state management)
- Socket.io-client (for real-time updates)
- React Hot Toast (for notifications)
- D3.js (for data flow visualization)
- React Flow (for pipeline configuration)
- Google A2A SDK (for agent communication)
- Google ADK (for agent development, see `adk_integration_plan.md`)

### **Backend Dependencies**:
- Gin (for HTTP routing)
- SQLite (for authentication database)
- chromem-go (for domain entity database)
- JWT (for authentication)
- WebSocket (for real-time updates)
- Kafka (for data engineering)
- Chromem-go (for caching)
- Prometheus (for monitoring)

### **Backend API Extensions Needed**:
- Agent lifecycle management endpoints
- Target system management endpoints
- MCP server management endpoints
- Real-time WebSocket connections
- A2A protocol endpoints
- Data engineering pipeline endpoints
- Stream processing configuration endpoints

### **Database Schema Extensions**:
- Agent deployment configurations
- Target system definitions
- MCP server registrations
- Activity logging tables
- A2A agent cards and capabilities
- Data pipeline configurations
- Stream processing rules
- Alert configurations

### **Infrastructure Requirements**:
- Apache Kafka cluster
- Chromem-go instance
- Prometheus and AlertManager
- A2A protocol support
- Secure agent communication channels

### **Database Schema**:

#### Authentication Database (SQLite)
```sql
-- Users table
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'user',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Roles table
CREATE TABLE roles (
    name TEXT PRIMARY KEY,
    description TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Permissions table
CREATE TABLE permissions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL
);

-- Role permissions junction table
CREATE TABLE role_permissions (
    role_name TEXT NOT NULL,
    permission_id INTEGER NOT NULL,
    PRIMARY KEY (role_name, permission_id),
    FOREIGN KEY (role_name) REFERENCES roles(name) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
);

-- API tokens table
CREATE TABLE api_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    token TEXT NOT NULL UNIQUE,
    description TEXT,
    expires_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

#### Domain Database with chromem-go

**Collections Structure**:

1. **agents Collection**
   - Document ID: Agent UUID
   - Content: Textual description of the agent and its capabilities
   - Metadata:
     - name: Agent name
     - collection: NFT collection name
     - image_url: URL to agent image
     - status: Agent status (idle, active, etc.)
     - token_id: NFT token ID
     - contract_addr: NFT contract address
     - owner_id: Owner user ID
     - created_at: Creation timestamp

2. **targets Collection**
   - Document ID: Target UUID
   - Content: Textual description of the target system
   - Metadata:
     - name: Target system name
     - type: Target type (browser, filesystem, etc.)
     - status: Target status (connected, disconnected, etc.)
     - owner_id: Owner user ID
     - last_activity: Last activity timestamp
     - created_at: Creation timestamp

3. **capabilities Collection**
   - Document ID: Capability UUID
   - Content: Detailed description of the capability
   - Metadata:
     - name: Capability name
     - provider: Provider name (OpenAI, Anthropic, etc.)
     - type: Capability type (vision, nlp, etc.)
     - estimated_time: Estimated execution time
     - system: Whether it's a system capability
     - owner_id: Owner user ID (for user-defined capabilities)
     - created_at: Creation timestamp

4. **workflows Collection**
   - Document ID: Workflow UUID
   - Content: Description of the workflow and its results
   - Metadata:
     - agent_id: Agent UUID
     - target_id: Target UUID
     - capability_id: Capability UUID
     - status: Workflow status (running, completed, failed)
     - start_time: Start timestamp
     - end_time: End timestamp
     - result: Execution result summary
     - owner_id: Owner user ID
     - created_at: Creation timestamp

5. **analytics Collection**
   - Document ID: Analytics UUID
   - Content: Detailed description of the analytics event
   - Metadata:
     - workflow_id: Workflow UUID
     - agent_id: Agent UUID
     - target_id: Target UUID
     - capability_id: Capability UUID
     - duration_ms: Execution duration in milliseconds
     - success: Whether execution was successful
     - timestamp: Execution timestamp

---

## ✅ **Success Criteria**

### **Functional Requirements**:
- [ ] Users can create new agents through the UI
- [ ] Users can deploy agents to target systems
- [ ] Users can start/stop agents with real-time feedback
- [ ] Users can configure agent settings
- [ ] All buttons and forms are functional
- [ ] Error handling is comprehensive
- [ ] Real-time updates work correctly
- [ ] Agents can communicate using the A2A protocol
- [ ] Data engineering pipelines can be configured and visualized
- [ ] Real-time data processing is operational
- [ ] WordPress dependencies are completely removed
- [ ] Authentication system works properly
- [ ] Database persistence is reliable
- [ ] API endpoints return proper responses
- [ ] Workflow orchestration functions correctly

### **Technical Requirements**:
- [ ] All API endpoints return proper responses
- [ ] Frontend state management is consistent
- [ ] Performance is acceptable (< 2s response times)
- [ ] Error boundaries prevent crashes
- [ ] Accessibility standards are met
- [ ] A2A protocol implementation is compliant with specifications
- [ ] Data pipeline configurations are correctly persisted
- [ ] Stream processing components function as expected
- [ ] Alerting system triggers notifications appropriately
- [ ] Database migrations work correctly
- [ ] Authentication system is secure
- [ ] Authorization system enforces permissions correctly

### **User Experience Requirements**:
- [ ] Intuitive navigation and workflows
- [ ] Clear feedback for all actions
- [ ] Responsive design works on all devices
- [ ] Loading states are informative
- [ ] Success/error messages are helpful
- [ ] Data flow visualizations are clear and interactive
- [ ] Agent communication interfaces are user-friendly
- [ ] Configuration wizards guide users effectively

---

## 📝 **Next Steps**

1. **Prioritize Implementation**: Start with HIGH PRIORITY items
2. **Create Backend APIs**: Implement missing API endpoints
3. **Set Up Development Environment**: Install required dependencies
4. **Begin Implementation**: Start with WordPress removal and database implementation
5. **Iterative Testing**: Test each feature as it's implemented
6. **User Feedback**: Gather feedback and iterate
7. **Research Priority Components**: Focus on Data Engineering and A2A Protocol research early
8. **Infrastructure Planning**: Set up Kafka, Chromem-go, and monitoring tools
9. **Integration Strategy**: Develop plan for integrating agent system with data engineering and A2A protocol

---

*This plan will be updated as implementation progresses and requirements evolve.*