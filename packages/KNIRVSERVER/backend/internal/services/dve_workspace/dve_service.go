package dve_workspace

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	// data_engine "backend_server/internal/data_engine" // TODO: Fix data_engine compilation issues
	data_engine "backend_server/internal/data_engine"
	ebpf "backend_server/internal/ebpf"
	"backend_server/internal/services/teesecurity"
)

// DVEService manages DVE Workspace environments
type DVEService struct {
	// Core components
	teeSecurityService  *teesecurity.TEESecurityService
	dataEngine          *data_engine.BuntDBDataEngine
	virtualContainerMgr *ebpf.VirtualContainerManager

	// Environment management
	environments map[string]*DVEEnvironment
	sessions     map[string]*DVESession
	projects     map[string]*DVEProject

	// Resource management
	resourcePool *DVEResourcePool

	// Namespace handles for OverlayFS workspaces
	namespaces map[string]*DVENamespace

	// Configuration
	config DVEConfig

	// State management
	isRunning bool
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// DVEConfig contains configuration for the DVE workspace service
type DVEConfig struct {
	// Environment settings
	BaseImagePath   string        `yaml:"base_image_path"`
	WorkspaceRoot   string        `yaml:"workspace_root"`
	MaxEnvironments int           `yaml:"max_environments"`
	DefaultTimeout  time.Duration `yaml:"default_timeout"`

	// Resource limits
	MaxCPUPerEnv    float64 `yaml:"max_cpu_per_env"`
	MaxMemoryPerEnv uint64  `yaml:"max_memory_per_env"`
	MaxDiskPerEnv   uint64  `yaml:"max_disk_per_env"`

	// Security settings
	EnableSandboxing       bool  `yaml:"enable_sandboxing"`
	EnableNetworkIsolation bool  `yaml:"enable_network_isolation"`
	AllowedPorts           []int `yaml:"allowed_ports"`

	// Session management
	SessionTimeout     time.Duration `yaml:"session_timeout"`
	MaxSessionsPerUser int           `yaml:"max_sessions_per_user"`

	// Project management
	MaxProjectsPerUser int    `yaml:"max_projects_per_user"`
	ProjectStoragePath string `yaml:"project_storage_path"`

	// OverlayFS settings
	EnableOverlayFS   bool   `yaml:"enable_overlayfs"`
	BusyBoxRootfsPath string `yaml:"busybox_rootfs_path"`
	BusyBoxSource     string `yaml:"busybox_source"`       // "embedded" | "package" | "download"
	BusyBoxVersion    string `yaml:"busybox_version"`      // only used with "download"
	FuseOverlayFSBin  string `yaml:"fuse_overlayfs_bin"`   // path, default "fuse-overlayfs"

	// Wazero skill executor settings
	SkillExecTimeout  time.Duration `yaml:"skill_exec_timeout"`
	SkillMaxMemoryMB  uint32        `yaml:"skill_max_memory_mb"`
	MaxConcurrentWASM int           `yaml:"max_concurrent_wasm"`

	// Workspace persistence
	WorkspaceRetentionHours int `yaml:"workspace_retention_hours"`

	// File explorer settings
	ExplorerEnabled        bool     `yaml:"explorer_enabled"`
	ExplorerAllowWrite     bool     `yaml:"explorer_allow_write"`
	ExplorerShowLayerBadges bool    `yaml:"explorer_show_layer_badges"`
	ExplorerHideSystemDirs bool     `yaml:"explorer_hide_system_dirs"`
	ExplorerMaxPreviewKB   int      `yaml:"explorer_max_preview_kb"`
}

// DVEEnvironment represents a development environment
type DVEEnvironment struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	UserID       string            `json:"user_id"`
	Status       EnvironmentStatus `json:"status"`
	CreatedAt    time.Time         `json:"created_at"`
	LastAccessed time.Time         `json:"last_accessed"`

	// Environment configuration
	BaseImage       string          `json:"base_image"`
	WorkspacePath   string          `json:"workspace_path"`
	EnvironmentType EnvironmentType `json:"environment_type"`

	// Resource allocation
	Resources *DVEResourceAllocation `json:"resources"`

	// Runtime information
	ContainerID string         `json:"container_id,omitempty"`
	IPAddress   string         `json:"ip_address,omitempty"`
	Ports       map[string]int `json:"ports"`

	// Project association
	ProjectID string `json:"project_id,omitempty"`

	// Configuration
	Config      map[string]interface{} `json:"config"`
	Environment map[string]string      `json:"environment"`

	// Metrics
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage uint64  `json:"memory_usage"`
	DiskUsage   uint64  `json:"disk_usage"`
	NetworkRx   uint64  `json:"network_rx"`
	NetworkTx   uint64  `json:"network_tx"`
}

// EnvironmentStatus represents the status of an environment
type EnvironmentStatus string

const (
	EnvStatusCreating   EnvironmentStatus = "creating"
	EnvStatusRunning    EnvironmentStatus = "running"
	EnvStatusStopped    EnvironmentStatus = "stopped"
	EnvStatusSuspended  EnvironmentStatus = "suspended"
	EnvStatusTerminated EnvironmentStatus = "terminated"
	EnvStatusError      EnvironmentStatus = "error"
)

// EnvironmentType represents the type of development environment
type EnvironmentType string

const (
	EnvTypeGeneral    EnvironmentType = "general"
	EnvTypePython     EnvironmentType = "python"
	EnvTypeNodeJS     EnvironmentType = "nodejs"
	EnvTypeGo         EnvironmentType = "go"
	EnvTypeRust       EnvironmentType = "rust"
	EnvTypeJava       EnvironmentType = "java"
	EnvTypeDocker     EnvironmentType = "docker"
	EnvTypeKubernetes EnvironmentType = "kubernetes"
)

// DVESession represents an active development session
type DVESession struct {
	ID            string        `json:"id"`
	EnvironmentID string        `json:"environment_id"`
	UserID        string        `json:"user_id"`
	Status        SessionStatus `json:"status"`
	StartTime     time.Time     `json:"start_time"`
	LastActivity  time.Time     `json:"last_activity"`
	ExpiresAt     time.Time     `json:"expires_at"`

	// Connection information
	ConnectionType string            `json:"connection_type"` // ssh, websocket, vnc
	ConnectionInfo map[string]string `json:"connection_info"`

	// Session data
	WorkingDirectory string   `json:"working_directory"`
	OpenFiles        []string `json:"open_files"`
	RunningProcesses []string `json:"running_processes"`
}

// SessionStatus represents the status of a session
type SessionStatus string

const (
	SessionStatusActive     SessionStatus = "active"
	SessionStatusIdle       SessionStatus = "idle"
	SessionStatusExpired    SessionStatus = "expired"
	SessionStatusTerminated SessionStatus = "terminated"
)

// DVEProject represents a development project
type DVEProject struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	UserID      string    `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Project configuration
	ProjectType ProjectType `json:"project_type"`
	Language    string      `json:"language"`
	Framework   string      `json:"framework"`

	// Storage
	StoragePath   string `json:"storage_path"`
	GitRepository string `json:"git_repository,omitempty"`

	// Environment requirements
	RequiredEnvType EnvironmentType `json:"required_env_type"`
	Dependencies    []string        `json:"dependencies"`

	// Metadata
	Tags   map[string]string      `json:"tags"`
	Config map[string]interface{} `json:"config"`

	// Statistics
	TotalSessions int       `json:"total_sessions"`
	LastAccessed  time.Time `json:"last_accessed"`
}

// ProjectType represents the type of project
type ProjectType string

const (
	ProjectTypeWebApp          ProjectType = "webapp"
	ProjectTypeAPI             ProjectType = "api"
	ProjectTypeMicroservice    ProjectType = "microservice"
	ProjectTypeLibrary         ProjectType = "library"
	ProjectTypeCLI             ProjectType = "cli"
	ProjectTypeDataScience     ProjectType = "datascience"
	ProjectTypeMachineLearning ProjectType = "ml"
	ProjectTypeBlockchain      ProjectType = "blockchain"
)

// DVEResourceAllocation represents resource allocation for an environment
type DVEResourceAllocation struct {
	CPUCores         float64 `json:"cpu_cores"`
	MemoryBytes      uint64  `json:"memory_bytes"`
	DiskBytes        uint64  `json:"disk_bytes"`
	NetworkBandwidth uint64  `json:"network_bandwidth"`

	// Limits
	CPULimit    float64 `json:"cpu_limit"`
	MemoryLimit uint64  `json:"memory_limit"`
	DiskLimit   uint64  `json:"disk_limit"`
}

// DVEResourcePool manages available resources for CDE environments
type DVEResourcePool struct {
	TotalCPU    float64 `json:"total_cpu"`
	TotalMemory uint64  `json:"total_memory"`
	TotalDisk   uint64  `json:"total_disk"`

	AvailableCPU    float64 `json:"available_cpu"`
	AvailableMemory uint64  `json:"available_memory"`
	AvailableDisk   uint64  `json:"available_disk"`

	AllocatedCPU    float64 `json:"allocated_cpu"`
	AllocatedMemory uint64  `json:"allocated_memory"`
	AllocatedDisk   uint64  `json:"allocated_disk"`

	mu sync.RWMutex
}

// NewDVEService creates a new CDE service
func NewDVEService(teeSecurityService *teesecurity.TEESecurityService, dataEngine *data_engine.BuntDBDataEngine, virtualContainerMgr *ebpf.VirtualContainerManager, config DVEConfig) (*DVEService, error) {
	ctx, cancel := context.WithCancel(context.Background())

	service := &DVEService{
		teeSecurityService:  teeSecurityService,
		dataEngine:          dataEngine,
		virtualContainerMgr: virtualContainerMgr,
		environments:        make(map[string]*DVEEnvironment),
		sessions:            make(map[string]*DVESession),
		projects:            make(map[string]*DVEProject),
		namespaces:          make(map[string]*DVENamespace),
		config:              config,
		ctx:                 ctx,
		cancel:              cancel,
	}

	// Initialize resource pool
	var err error
	service.resourcePool, err = NewDVEResourcePool()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create resource pool: %w", err)
	}

	// Ensure workspace directories exist
	if err := service.initializeWorkspaceDirectories(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize workspace directories: %w", err)
	}

	return service, nil
}

// Start starts the CDE service
func (cde *DVEService) Start() error {
	cde.mu.Lock()
	defer cde.mu.Unlock()

	if cde.isRunning {
		return fmt.Errorf("CDE service is already running")
	}

	// Start monitoring loops
	go cde.environmentMonitoringLoop()
	go cde.sessionManagementLoop()
	go cde.resourceMonitoringLoop()
	go cde.cleanupLoop()

	cde.isRunning = true
	log.Println("DVEService: Started successfully")

	return nil
}

// Stop stops the DVE service
func (s *DVEService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return nil
	}

	// Stop all environments
	for _, env := range s.environments {
		if env.Status == EnvStatusRunning {
			s.stopEnvironmentInternal(env)
		}
	}

	// Terminate all sessions
	for _, session := range s.sessions {
		if session.Status == SessionStatusActive {
			session.Status = SessionStatusTerminated
		}
	}

	// Tear down all namespace helpers (OverlayFS unmount)
	for id, ns := range s.namespaces {
		log.Printf("DVEService: tearing down namespace for %s", id)
		if err := ns.Teardown(); err != nil {
			log.Printf("DVEService: namespace teardown error for %s: %v", id, err)
		}
	}
	s.namespaces = make(map[string]*DVENamespace)

	// Cancel context
	s.cancel()

	s.isRunning = false
	log.Println("DVEService: Stopped successfully")

	return nil
}

// CreateVirtualDVE creates a new virtual CDE using eBPF-based virtual containers
func (cde *DVEService) CreateVirtualDVE(userID, name string, envType EnvironmentType, config map[string]interface{}) (*DVEEnvironment, error) {
	cde.mu.Lock()
	defer cde.mu.Unlock()

	if !cde.isRunning {
		return nil, fmt.Errorf("CDE service is not running")
	}

	// Check if workspace root is available
	if cde.config.WorkspaceRoot == "" {
		return nil, fmt.Errorf("workspace root not configured")
	}

	// Check environment limit
	userEnvCount := cde.countUserEnvironments(userID)
	if userEnvCount >= cde.config.MaxEnvironments {
		return nil, fmt.Errorf("maximum environments (%d) reached for user %s", cde.config.MaxEnvironments, userID)
	}

	// Allocate resources
	resources := &DVEResourceAllocation{
		CPUCores:    cde.config.MaxCPUPerEnv,
		MemoryBytes: cde.config.MaxMemoryPerEnv,
		DiskBytes:   cde.config.MaxDiskPerEnv,
		CPULimit:    cde.config.MaxCPUPerEnv,
		MemoryLimit: cde.config.MaxMemoryPerEnv,
		DiskLimit:   cde.config.MaxDiskPerEnv,
	}

	if !cde.resourcePool.CanAllocate(resources) {
		return nil, fmt.Errorf("insufficient resources available")
	}

	// Create environment
	env := &DVEEnvironment{
		ID:              fmt.Sprintf("env-virtual-%d", time.Now().UnixNano()),
		Name:            name,
		UserID:          userID,
		Status:          EnvStatusCreating,
		CreatedAt:       time.Now(),
		LastAccessed:    time.Now(),
		EnvironmentType: envType,
		BaseImage:       cde.getBaseImageForType(envType),
		WorkspacePath:   filepath.Join(cde.config.WorkspaceRoot, userID, name),
		Resources:       resources,
		Ports:           make(map[string]int),
		Config:          config,
		Environment:     make(map[string]string),
	}

	// Allocate resources
	cde.resourcePool.AllocateResources(resources)

	// Add to environments
	cde.environments[env.ID] = env

	// Start virtual CDE creation
	go cde.createVirtualDVEAsync(env)

	log.Printf("DVEService: Created virtual CDE %s for user %s", env.ID, userID)

	return env, nil
}

// CreateEnvironment creates a new development environment
func (cde *DVEService) CreateEnvironment(userID, name string, envType EnvironmentType, config map[string]interface{}) (*DVEEnvironment, error) {
	cde.mu.Lock()
	defer cde.mu.Unlock()

	if !cde.isRunning {
		return nil, fmt.Errorf("CDE service is not running")
	}

	// Check if workspace root is available
	if cde.config.WorkspaceRoot == "" {
		return nil, fmt.Errorf("workspace root not configured")
	}

	// Check environment limit
	userEnvCount := cde.countUserEnvironments(userID)
	if userEnvCount >= cde.config.MaxEnvironments {
		return nil, fmt.Errorf("maximum environments (%d) reached for user %s", cde.config.MaxEnvironments, userID)
	}

	// Allocate resources
	resources := &DVEResourceAllocation{
		CPUCores:    cde.config.MaxCPUPerEnv,
		MemoryBytes: cde.config.MaxMemoryPerEnv,
		DiskBytes:   cde.config.MaxDiskPerEnv,
		CPULimit:    cde.config.MaxCPUPerEnv,
		MemoryLimit: cde.config.MaxMemoryPerEnv,
		DiskLimit:   cde.config.MaxDiskPerEnv,
	}

	if !cde.resourcePool.CanAllocate(resources) {
		return nil, fmt.Errorf("insufficient resources available")
	}

	// Create environment
	env := &DVEEnvironment{
		ID:              fmt.Sprintf("env-%d", time.Now().UnixNano()),
		Name:            name,
		UserID:          userID,
		Status:          EnvStatusCreating,
		CreatedAt:       time.Now(),
		LastAccessed:    time.Now(),
		EnvironmentType: envType,
		BaseImage:       cde.getBaseImageForType(envType),
		WorkspacePath:   filepath.Join(cde.config.WorkspaceRoot, userID, name),
		Resources:       resources,
		Ports:           make(map[string]int),
		Config:          config,
		Environment:     make(map[string]string),
	}

	// Allocate resources
	cde.resourcePool.AllocateResources(resources)

	// Add to environments
	cde.environments[env.ID] = env

	// Start environment creation
	go cde.createEnvironmentAsync(env)

	log.Printf("DVEService: Created environment %s for user %s", env.ID, userID)

	return env, nil
}

// createVirtualDVEAsync creates an isolated DVE workspace using
// OverlayFS + Linux user namespaces. Replaces the former eBPF placeholder.
func (s *DVEService) createVirtualDVEAsync(env *DVEEnvironment) {
	// 1. Ensure BusyBox rootfs is bootstrapped
	if s.config.EnableOverlayFS {
		if err := EnsureBusyBoxRootfs(s.config.BusyBoxRootfsPath, s.config); err != nil {
			env.Status = EnvStatusError
			log.Printf("DVEService: busybox rootfs bootstrap failed for %s: %v", env.ID, err)
			return
		}

		// 2. Allocate overlay workspace directories
		ws, err := NewOverlayWorkspace(s.config.WorkspaceRoot, s.config.BusyBoxRootfsPath, env.ID)
		if err != nil {
			env.Status = EnvStatusError
			log.Printf("DVEService: overlay workspace alloc failed for %s: %v", env.ID, err)
			return
		}

		// 3. Spawn user+mount namespace and mount OverlayFS within it
		fuseBin := s.config.FuseOverlayFSBin
		if fuseBin == "" {
			fuseBin = "fuse-overlayfs"
		}
		ns, err := SpawnNamespaced(ws, os.Getuid(), os.Getgid(), fuseBin)
		if err != nil {
			env.Status = EnvStatusError
			log.Printf("DVEService: namespace spawn failed for %s: %v", env.ID, err)
			return
		}

		// 4. Write DVE identity files into the upper (writable) layer
		identityDir := filepath.Join(ws.UpperDir, "workspace")
		os.MkdirAll(identityDir, 0755)
		os.WriteFile(filepath.Join(identityDir, "IDENTITY.md"),
			[]byte(s.generateDVEIdentity(env)), 0644)
		os.WriteFile(filepath.Join(identityDir, "AGENT.md"),
			[]byte(s.generateAgentInstructions(env)), 0644)

		// 5. Register namespace handle for teardown
		s.mu.Lock()
		s.namespaces[env.ID] = ns
		env.Status = EnvStatusRunning
		env.WorkspacePath = ws.MergedDir // agents see merged/ as their root
		env.ContainerID = fmt.Sprintf("ovl-%s", env.ID)
		s.mu.Unlock()

		log.Printf("DVEService: workspace %s ready at %s (OverlayFS)", env.ID, ws.MergedDir)
		return
	}

	// Fallback: use workspace directory without overlay isolation
	// Create workspace directory
	if err := os.MkdirAll(env.WorkspacePath, 0755); err != nil {
		env.Status = EnvStatusError
		log.Printf("DVEService: Failed to create workspace directory for DVE %s: %v", env.ID, err)
		return
	}

	// Use eBPF-based virtual container for instant creation
	if s.virtualContainerMgr != nil {
		// Launch a simple process to serve as the root of the virtual container
		cmd := exec.Command("/usr/bin/env", "bash", "-c", "sleep 3600")
		cmd.Dir = env.WorkspacePath
		if err := cmd.Start(); err != nil {
			env.Status = EnvStatusError
			log.Printf("DVEService: Failed to start root process for DVE %s: %v", env.ID, err)
			return
		}

		// Create virtual container using eBPF
		container, err := s.virtualContainerMgr.CreateVirtualContainer(
			uint32(cmd.Process.Pid),
			env.WorkspacePath,
		)
		if err != nil {
			cmd.Process.Kill()
			env.Status = EnvStatusError
			log.Printf("DVEService: Failed to create virtual container for DVE %s: %v", env.ID, err)
			return
		}

		// Set up environment in the virtual container
		setupScript := s.generateEnvironmentSetupScript(env)
		if err := s.executeInVirtualContainer(container.ID, setupScript); err != nil {
			s.virtualContainerMgr.DestroyVirtualContainer(container.ID)
			cmd.Process.Kill()
			env.Status = EnvStatusError
			log.Printf("DVEService: Failed to set up DVE %s: %v", env.ID, err)
			return
		}

		// Store container ID for later reference
		env.ContainerID = fmt.Sprintf("virtual-%d", container.ID)

		log.Printf("DVEService: DVE %s created using eBPF, container ID: %d", env.ID, container.ID)
	} else {
		// Fallback: simulate DVE creation
		log.Printf("DVEService: Virtual container manager not available, simulating DVE creation for %s", env.ID)
		time.Sleep(10 * time.Millisecond) // Simulate fast creation
	}

	s.mu.Lock()
	env.Status = EnvStatusRunning
	env.IPAddress = "172.20.0.10" // Simulated IP
	env.Ports["ssh"] = 22
	env.Ports["http"] = 8082
	s.mu.Unlock()

	// Log metrics
	if s.dataEngine != nil {
		s.dataEngine.ProcessMetricEvent(
			"dve-service",
			"dve_created",
			1.0,
			"count",
			map[string]string{
				"user_id":  env.UserID,
				"env_type": string(env.EnvironmentType),
				"env_id":   env.ID,
			},
		)
	}

	log.Printf("DVEService: DVE %s is now running", env.ID)
}

// executeInVirtualContainer executes a script in a virtual container
func (cde *DVEService) executeInVirtualContainer(containerID uint64, script string) error {
	// In a real implementation, this would use the eBPF manager to
	// execute the script within the virtual container's context
	// For now, we simulate this by just running the script

	// Create a temporary script file
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("virtual-cde-script-%d.sh", containerID))
	if err := os.WriteFile(tmpFile, []byte(script), 0755); err != nil {
		return fmt.Errorf("write script file: %w", err)
	}
	defer os.Remove(tmpFile)

	// Execute the script
	cmd := exec.Command("bash", tmpFile)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("execute script: %w", err)
	}

	return nil
}

// createEnvironmentAsync creates an environment asynchronously using TEE security service
func (cde *DVEService) createEnvironmentAsync(env *DVEEnvironment) {
	// Create workspace directory
	if err := os.MkdirAll(env.WorkspacePath, 0755); err != nil {
		env.Status = EnvStatusError
		log.Printf("DVEService: Failed to create workspace directory for %s: %v", env.ID, err)
		return
	}

	// Use TEE security service to create sandboxed environment
	if cde.teeSecurityService != nil {
		// Create a skill script that sets up the development environment
		setupScript := cde.generateEnvironmentSetupScript(env)

		// Execute in sandbox using TEE security service
		result, err := cde.teeSecurityService.ExecuteSkillInSandbox(
			context.Background(),
			setupScript,
			[]string{}, // No test cases for environment setup
		)

		if err != nil {
			env.Status = EnvStatusError
			log.Printf("DVEService: Failed to create environment %s via TEE security: %v", env.ID, err)
			return
		}

		log.Printf("DVEService: Environment setup completed for %s, exit code: %d", env.ID, result.ExitCode)

		if result.ExitCode != 0 {
			env.Status = EnvStatusError
			log.Printf("DVEService: Environment setup failed for %s: %s", env.ID, result.Stderr)
			return
		}
	} else {
		// Fallback: simulate environment creation
		log.Printf("DVEService: TEE security service not available, simulating environment creation for %s", env.ID)
		time.Sleep(5 * time.Second)
	}

	cde.mu.Lock()
	env.Status = EnvStatusRunning
	env.IPAddress = "172.20.0.10" // Simulated IP (in real implementation, this would come from container)
	env.Ports["ssh"] = 22
	env.Ports["http"] = 8082
	cde.mu.Unlock()

	// Log metrics
	if cde.dataEngine != nil {
		cde.dataEngine.ProcessMetricEvent(
			"cde-service",
			"environment_created",
			1.0,
			"count",
			map[string]string{
				"user_id":  env.UserID,
				"env_type": string(env.EnvironmentType),
				"env_id":   env.ID,
			},
		)
	}

	log.Printf("DVEService: Environment %s is now running", env.ID)
}

// StopEnvironment stops a development environment
func (cde *DVEService) StopEnvironment(envID string) error {
	cde.mu.Lock()
	defer cde.mu.Unlock()

	env, exists := cde.environments[envID]
	if !exists {
		return fmt.Errorf("environment %s not found", envID)
	}

	return cde.stopEnvironmentInternal(env)
}

// stopEnvironmentInternal stops an environment (internal method, assumes lock is held)
func (cde *DVEService) stopEnvironmentInternal(env *DVEEnvironment) error {
	if env.Status != EnvStatusRunning {
		return fmt.Errorf("environment %s is not running", env.ID)
	}

	env.Status = EnvStatusStopped

	// Release resources
	cde.resourcePool.ReleaseResources(env.Resources)

	// Terminate any active sessions
	for _, session := range cde.sessions {
		if session.EnvironmentID == env.ID && session.Status == SessionStatusActive {
			session.Status = SessionStatusTerminated
		}
	}

	log.Printf("DVEService: Stopped environment %s", env.ID)

	return nil
}

// CreateSession creates a new development session
func (cde *DVEService) CreateSession(userID, envID string, connectionType string) (*DVESession, error) {
	cde.mu.Lock()
	defer cde.mu.Unlock()

	if !cde.isRunning {
		return nil, fmt.Errorf("CDE service is not running")
	}

	// Check if environment exists and is running
	env, exists := cde.environments[envID]
	if !exists {
		return nil, fmt.Errorf("environment %s not found", envID)
	}

	if env.Status != EnvStatusRunning {
		return nil, fmt.Errorf("environment %s is not running", envID)
	}

	// Check user permissions
	if env.UserID != userID {
		return nil, fmt.Errorf("user %s does not have access to environment %s", userID, envID)
	}

	// Check session limit
	userSessionCount := cde.countUserSessions(userID)
	if userSessionCount >= cde.config.MaxSessionsPerUser {
		return nil, fmt.Errorf("maximum sessions (%d) reached for user %s", cde.config.MaxSessionsPerUser, userID)
	}

	// Create session
	session := &DVESession{
		ID:               fmt.Sprintf("sess-%d", time.Now().UnixNano()),
		EnvironmentID:    envID,
		UserID:           userID,
		Status:           SessionStatusActive,
		StartTime:        time.Now(),
		LastActivity:     time.Now(),
		ExpiresAt:        time.Now().Add(cde.config.SessionTimeout),
		ConnectionType:   connectionType,
		ConnectionInfo:   make(map[string]string),
		WorkingDirectory: "/workspace",
		OpenFiles:        []string{},
		RunningProcesses: []string{},
	}

	// Set connection info based on type
	switch connectionType {
	case "ssh":
		session.ConnectionInfo["host"] = env.IPAddress
		session.ConnectionInfo["port"] = fmt.Sprintf("%d", env.Ports["ssh"])
		session.ConnectionInfo["username"] = "developer"
	case "websocket":
		session.ConnectionInfo["url"] = fmt.Sprintf("ws://%s:8082/terminal", env.IPAddress)
	case "vnc":
		session.ConnectionInfo["host"] = env.IPAddress
		session.ConnectionInfo["port"] = "5900"
	}

	// Add to sessions
	cde.sessions[session.ID] = session

	// Update environment last accessed
	env.LastAccessed = time.Now()

	log.Printf("DVEService: Created session %s for user %s in environment %s", session.ID, userID, envID)

	return session, nil
}

// CreateProject creates a new development project
func (cde *DVEService) CreateProject(userID, name, description string, projectType ProjectType, language string) (*DVEProject, error) {
	cde.mu.Lock()
	defer cde.mu.Unlock()

	if !cde.isRunning {
		return nil, fmt.Errorf("CDE service is not running")
	}

	// Check if project storage path is available
	if cde.config.ProjectStoragePath == "" {
		return nil, fmt.Errorf("project storage path not configured")
	}

	// Check project limit
	userProjectCount := cde.countUserProjects(userID)
	if userProjectCount >= cde.config.MaxProjectsPerUser {
		return nil, fmt.Errorf("maximum projects (%d) reached for user %s", cde.config.MaxProjectsPerUser, userID)
	}

	// Create project
	project := &DVEProject{
		ID:              fmt.Sprintf("proj-%d", time.Now().UnixNano()),
		Name:            name,
		Description:     description,
		UserID:          userID,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		ProjectType:     projectType,
		Language:        language,
		StoragePath:     filepath.Join(cde.config.ProjectStoragePath, userID, name),
		RequiredEnvType: cde.getRequiredEnvType(language),
		Dependencies:    []string{},
		Tags:            make(map[string]string),
		Config:          make(map[string]interface{}),
		LastAccessed:    time.Now(),
	}

	// Create project directory
	if err := os.MkdirAll(project.StoragePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create project directory: %w", err)
	}

	// Add to projects
	cde.projects[project.ID] = project

	log.Printf("DVEService: Created project %s for user %s", project.ID, userID)

	return project, nil
}

// GetEnvironment returns an environment by ID
func (cde *DVEService) GetEnvironment(envID string) (*DVEEnvironment, error) {
	cde.mu.RLock()
	defer cde.mu.RUnlock()

	env, exists := cde.environments[envID]
	if !exists {
		return nil, fmt.Errorf("environment %s not found", envID)
	}

	// Return a copy
	envCopy := *env
	return &envCopy, nil
}

// ListUserEnvironments returns all environments for a user
func (cde *DVEService) ListUserEnvironments(userID string) []*DVEEnvironment {
	cde.mu.RLock()
	defer cde.mu.RUnlock()

	var userEnvs []*DVEEnvironment
	for _, env := range cde.environments {
		if env.UserID == userID {
			envCopy := *env
			userEnvs = append(userEnvs, &envCopy)
		}
	}

	return userEnvs
}

// ListUserSessions returns all sessions for a user
func (cde *DVEService) ListUserSessions(userID string) []*DVESession {
	cde.mu.RLock()
	defer cde.mu.RUnlock()

	var userSessions []*DVESession
	for _, session := range cde.sessions {
		if session.UserID == userID {
			sessionCopy := *session
			userSessions = append(userSessions, &sessionCopy)
		}
	}

	return userSessions
}

// ListUserProjects returns all projects for a user
func (cde *DVEService) ListUserProjects(userID string) []*DVEProject {
	cde.mu.RLock()
	defer cde.mu.RUnlock()

	var userProjects []*DVEProject
	for _, project := range cde.projects {
		if project.UserID == userID {
			projectCopy := *project
			userProjects = append(userProjects, &projectCopy)
		}
	}

	return userProjects
}

// Helper methods

// countUserEnvironments counts environments for a user
func (cde *DVEService) countUserEnvironments(userID string) int {
	count := 0
	for _, env := range cde.environments {
		if env.UserID == userID && env.Status != EnvStatusTerminated {
			count++
		}
	}
	return count
}

// countUserSessions counts active sessions for a user
func (cde *DVEService) countUserSessions(userID string) int {
	count := 0
	for _, session := range cde.sessions {
		if session.UserID == userID && session.Status == SessionStatusActive {
			count++
		}
	}
	return count
}

// countUserProjects counts projects for a user
func (cde *DVEService) countUserProjects(userID string) int {
	count := 0
	for _, project := range cde.projects {
		if project.UserID == userID {
			count++
		}
	}
	return count
}

// getBaseImageForType returns the base image for an environment type
func (cde *DVEService) getBaseImageForType(envType EnvironmentType) string {
	switch envType {
	case EnvTypePython:
		return "python:3.11-slim"
	case EnvTypeNodeJS:
		return "node:18-alpine"
	case EnvTypeGo:
		return "golang:1.21-alpine"
	case EnvTypeRust:
		return "rust:1.70-slim"
	case EnvTypeJava:
		return "openjdk:17-jdk-slim"
	default:
		return "ubuntu:22.04"
	}
}

// getRequiredEnvType returns the required environment type for a language
func (cde *DVEService) getRequiredEnvType(language string) EnvironmentType {
	switch language {
	case "python":
		return EnvTypePython
	case "javascript", "typescript":
		return EnvTypeNodeJS
	case "go":
		return EnvTypeGo
	case "rust":
		return EnvTypeRust
	case "java", "kotlin", "scala":
		return EnvTypeJava
	default:
		return EnvTypeGeneral
	}
}

// generateEnvironmentSetupScript generates a shell script to set up the development environment
func (cde *DVEService) generateEnvironmentSetupScript(env *DVEEnvironment) string {
	var script strings.Builder

	// Common setup
	script.WriteString("#!/bin/bash\n")
	script.WriteString("set -e\n\n")
	script.WriteString("echo 'Setting up CDE environment...'\n\n")

	// Create workspace directory
	script.WriteString("mkdir -p /workspace\n")
	script.WriteString("cd /workspace\n\n")

	// Environment-specific setup
	switch env.EnvironmentType {
	case EnvTypePython:
		script.WriteString("# Python environment setup\n")
		script.WriteString("echo 'Setting up Python development environment...'\n")
		script.WriteString("pip install --upgrade pip\n")
		script.WriteString("pip install virtualenv\n")
		script.WriteString("python -m venv .venv\n")
		script.WriteString("source .venv/bin/activate\n")
		script.WriteString("pip install jupyter notebook\n")
		script.WriteString("echo 'Python environment ready'\n")

	case EnvTypeNodeJS:
		script.WriteString("# Node.js environment setup\n")
		script.WriteString("echo 'Setting up Node.js development environment...'\n")
		script.WriteString("npm init -y\n")
		script.WriteString("npm install --save-dev nodemon\n")
		script.WriteString("echo 'Node.js environment ready'\n")

	case EnvTypeGo:
		script.WriteString("# Go environment setup\n")
		script.WriteString("echo 'Setting up Go development environment...'\n")
		script.WriteString("go mod init cde-project\n")
		script.WriteString("go get github.com/gin-gonic/gin\n")
		script.WriteString("echo 'Go environment ready'\n")

	case EnvTypeRust:
		script.WriteString("# Rust environment setup\n")
		script.WriteString("echo 'Setting up Rust development environment...'\n")
		script.WriteString("cargo init\n")
		script.WriteString("echo 'Rust environment ready'\n")

	case EnvTypeJava:
		script.WriteString("# Java environment setup\n")
		script.WriteString("echo 'Setting up Java development environment...'\n")
		script.WriteString("mkdir -p src/main/java\n")
		script.WriteString("echo 'public class Main { public static void main(String[] args) { System.out.println(\"Hello CDE!\"); } }' > src/main/java/Main.java\n")
		script.WriteString("echo 'Java environment ready'\n")

	case EnvTypeDocker:
		script.WriteString("# Docker environment setup\n")
		script.WriteString("echo 'Setting up Docker development environment...'\n")
		script.WriteString("echo 'FROM ubuntu:22.04' > Dockerfile\n")
		script.WriteString("echo 'Docker environment ready'\n")

	default: // EnvTypeGeneral
		script.WriteString("# General development environment setup\n")
		script.WriteString("echo 'Setting up general development environment...'\n")
		script.WriteString("apt-get update && apt-get install -y git curl wget\n")
		script.WriteString("echo 'General environment ready'\n")
	}

	// Common finalization
	script.WriteString("\n# Environment setup complete\n")
	script.WriteString("echo 'CDE environment setup completed successfully'\n")
	script.WriteString("echo 'Workspace: /workspace'\n")

	return script.String()
}

// initializeWorkspaceDirectories creates necessary workspace directories
func (cde *DVEService) initializeWorkspaceDirectories() error {
	dirs := []string{
		cde.config.WorkspaceRoot,
		cde.config.ProjectStoragePath,
	}

	// BaseImagePath is already set to the images directory, no need to add subdirectory

	for _, dir := range dirs {
		// Skip empty directory paths
		if dir == "" {
			continue
		}

		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// generateDVEIdentity generates the IDENTITY.md file content for a DVE workspace.
func (s *DVEService) generateDVEIdentity(env *DVEEnvironment) string {
	return fmt.Sprintf(`# DVE Workspace Identity

- ID: %s
- Name: %s
- User: %s
- Type: %s
- Created: %s
- Environment: KNIRV DVE (OverlayFS)

This workspace is an isolated development environment powered by OverlayFS
and Linux user namespaces. All filesystem operations are confined to this
workspace.
`, env.ID, env.Name, env.UserID, env.EnvironmentType, env.CreatedAt.Format(time.RFC3339))
}

// generateAgentInstructions generates the AGENT.md file content for a DVE workspace.
func (s *DVEService) generateAgentInstructions(env *DVEEnvironment) string {
	return fmt.Sprintf(`# Agent Instructions

This is an autonomous DVE workspace for %s.

## Available Tools
- Shell access via /bin/sh (BusyBox)
- curl, wget for network access
- tar, gzip for archive operations
- grep, find, sed, awk for text processing

## Workspace Layout
- /workspace - main working directory
- /tmp - temporary files
- IDENTITY.md - workspace identity

## Rules
1. All work must happen within /workspace
2. Do not modify /etc or system files
3. Write output files to /workspace/results/ for durable storage
`, env.Name)
}

// Monitoring loops

// environmentMonitoringLoop monitors environment health and metrics
func (cde *DVEService) environmentMonitoringLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cde.ctx.Done():
			return
		case <-ticker.C:
			cde.monitorEnvironments()
		}
	}
}

// monitorEnvironments monitors all environments
func (cde *DVEService) monitorEnvironments() {
	cde.mu.RLock()
	environments := make([]*DVEEnvironment, 0, len(cde.environments))
	for _, env := range cde.environments {
		environments = append(environments, env)
	}
	cde.mu.RUnlock()

	for _, env := range environments {
		if env.Status == EnvStatusRunning {
			// Update environment metrics (simulated)
			cde.updateEnvironmentMetrics(env)

			// Log metrics to data engine
			if cde.dataEngine != nil {
				cde.dataEngine.ProcessMetricEvent(
					"cde-environment",
					"cpu_usage",
					env.CPUUsage,
					"percent",
					map[string]string{
						"env_id":   env.ID,
						"user_id":  env.UserID,
						"env_type": string(env.EnvironmentType),
					},
				)

				cde.dataEngine.ProcessMetricEvent(
					"cde-environment",
					"memory_usage",
					float64(env.MemoryUsage),
					"bytes",
					map[string]string{
						"env_id":   env.ID,
						"user_id":  env.UserID,
						"env_type": string(env.EnvironmentType),
					},
				)
			}
		}
	}
}

// updateEnvironmentMetrics updates metrics for an environment
func (cde *DVEService) updateEnvironmentMetrics(env *DVEEnvironment) {
	// Simulate metric collection
	// In a real implementation, this would query container metrics
	env.CPUUsage = 15.5 + float64(time.Now().Unix()%10)                       // Simulated CPU usage
	env.MemoryUsage = 512*1024*1024 + uint64(time.Now().Unix()%100)*1024*1024 // Simulated memory usage
	env.DiskUsage = 1024 * 1024 * 1024                                        // Simulated disk usage
}

// sessionManagementLoop manages session lifecycle
func (cde *DVEService) sessionManagementLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-cde.ctx.Done():
			return
		case <-ticker.C:
			cde.manageSessions()
		}
	}
}

// manageSessions manages session timeouts and cleanup
func (cde *DVEService) manageSessions() {
	cde.mu.Lock()
	defer cde.mu.Unlock()

	now := time.Now()

	for _, session := range cde.sessions {
		if session.Status == SessionStatusActive {
			// Check for session timeout
			if now.After(session.ExpiresAt) {
				session.Status = SessionStatusExpired
				log.Printf("DVEService: Session %s expired", session.ID)

				// Log session expiry
				if cde.dataEngine != nil {
					cde.dataEngine.ProcessMetricEvent(
						"cde-session",
						"session_expired",
						1.0,
						"count",
						map[string]string{
							"session_id": session.ID,
							"user_id":    session.UserID,
							"env_id":     session.EnvironmentID,
						},
					)
				}
			} else if now.Sub(session.LastActivity) > 30*time.Minute {
				// Mark as idle if no activity for 30 minutes
				session.Status = SessionStatusIdle
			}
		}
	}
}

// resourceMonitoringLoop monitors resource usage
func (cde *DVEService) resourceMonitoringLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-cde.ctx.Done():
			return
		case <-ticker.C:
			cde.monitorResources()
		}
	}
}

// monitorResources monitors resource pool usage
func (cde *DVEService) monitorResources() {
	// Update resource pool
	cde.resourcePool.UpdateAvailableResources()

	// Get resource usage
	usage := cde.resourcePool.GetResourceUsage()

	// Log resource metrics
	if cde.dataEngine != nil {
		if cpuData, ok := usage["cpu"].(map[string]interface{}); ok {
			if usagePercent, ok := cpuData["usage_percent"].(float64); ok {
				cde.dataEngine.ProcessMetricEvent(
					"cde-resources",
					"cpu_usage_percent",
					usagePercent,
					"percent",
					map[string]string{
						"component": "resource-pool",
					},
				)
			}
		}

		if memData, ok := usage["memory"].(map[string]interface{}); ok {
			if usagePercent, ok := memData["usage_percent"].(float64); ok {
				cde.dataEngine.ProcessMetricEvent(
					"cde-resources",
					"memory_usage_percent",
					usagePercent,
					"percent",
					map[string]string{
						"component": "resource-pool",
					},
				)
			}
		}
	}

	// Check for resource alerts
	cde.checkResourceAlerts(usage)
}

// checkResourceAlerts checks for resource usage alerts
func (cde *DVEService) checkResourceAlerts(usage map[string]interface{}) {
	if cde.dataEngine == nil {
		return
	}

	// Check CPU usage
	if cpuData, ok := usage["cpu"].(map[string]interface{}); ok {
		if usagePercent, ok := cpuData["usage_percent"].(float64); ok && usagePercent > 80 {
			cde.dataEngine.ProcessAlertEvent(
				"High CPU Usage",
				fmt.Sprintf("CDE resource pool CPU usage is %.1f%%", usagePercent),
				"warning",
				"cde-service",
				map[string]string{
					"component": "resource-pool",
					"usage":     fmt.Sprintf("%.1f%%", usagePercent),
				},
			)
		}
	}

	// Check memory usage
	if memData, ok := usage["memory"].(map[string]interface{}); ok {
		if usagePercent, ok := memData["usage_percent"].(float64); ok && usagePercent > 85 {
			cde.dataEngine.ProcessAlertEvent(
				"High Memory Usage",
				fmt.Sprintf("CDE resource pool memory usage is %.1f%%", usagePercent),
				"warning",
				"cde-service",
				map[string]string{
					"component": "resource-pool",
					"usage":     fmt.Sprintf("%.1f%%", usagePercent),
				},
			)
		}
	}
}

// cleanupLoop performs periodic cleanup
func (cde *DVEService) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-cde.ctx.Done():
			return
		case <-ticker.C:
			cde.performCleanup()
		}
	}
}

// performCleanup performs cleanup of old environments and sessions
func (cde *DVEService) performCleanup() {
	cde.mu.Lock()
	defer cde.mu.Unlock()

	now := time.Now()

	// Clean up old terminated environments
	for envID, env := range cde.environments {
		if env.Status == EnvStatusTerminated && now.Sub(env.LastAccessed) > 24*time.Hour {
			delete(cde.environments, envID)
			log.Printf("DVEService: Cleaned up terminated environment %s", envID)
		}
	}

	// Clean up old terminated sessions
	for sessionID, session := range cde.sessions {
		if session.Status == SessionStatusTerminated && now.Sub(session.LastActivity) > 24*time.Hour {
			delete(cde.sessions, sessionID)
			log.Printf("DVEService: Cleaned up terminated session %s", sessionID)
		}
	}
}

// GetStatus returns the current status of the CDE service
func (cde *DVEService) GetStatus() map[string]interface{} {
	cde.mu.RLock()
	defer cde.mu.RUnlock()

	status := map[string]interface{}{
		"running": cde.isRunning,
		"config":  cde.config,
	}

	// Count environments by status
	envCounts := make(map[string]int)
	for _, env := range cde.environments {
		envCounts[string(env.Status)]++
	}
	status["environments"] = envCounts

	// Count sessions by status
	sessionCounts := make(map[string]int)
	for _, session := range cde.sessions {
		sessionCounts[string(session.Status)]++
	}
	status["sessions"] = sessionCounts

	// Add resource usage
	if cde.resourcePool != nil {
		status["resources"] = cde.resourcePool.GetResourceUsage()
	}

	// Add project count
	status["projects"] = len(cde.projects)

	return status
}

// IsRunning returns whether the CDE service is running
func (cde *DVEService) IsRunning() bool {
	cde.mu.RLock()
	defer cde.mu.RUnlock()
	return cde.isRunning
}

// GetWorkspace returns the OverlayWorkspace for a DVE by ID.
func (s *DVEService) GetWorkspace(dveID string) (*OverlayWorkspace, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ns, ok := s.namespaces[dveID]
	if !ok {
		return nil, fmt.Errorf("no active workspace for DVE %s", dveID)
	}
	return ns.Workspace, nil
}
