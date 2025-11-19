package installation

import (
	"context"
	"time"
)

// InstallationManager defines the interface for installation management
type InstallationManager interface {
	// Installation operations
	Install(config *InstallationConfig) (*InstallationResult, error)
	Uninstall(config *UninstallationConfig) (*UninstallationResult, error)
	Upgrade(config *UpgradeConfig) (*UpgradeResult, error)
	
	// Installation validation
	ValidateInstallation() (*ValidationResult, error)
	CheckDependencies() (*DependencyResult, error)
	VerifyIntegrity() (*IntegrityResult, error)
	
	// Installation status
	GetInstallationStatus() (*InstallationStatus, error)
	IsInstalled() bool
	GetInstalledVersion() (string, error)
	
	// Lifecycle
	Start(ctx context.Context) error
	Stop() error
}

// DependencyManager defines the interface for dependency management
type DependencyManager interface {
	// Dependency operations
	InstallDependency(dependency *Dependency) error
	UninstallDependency(dependencyID string) error
	UpdateDependency(dependencyID string, version string) error
	
	// Dependency queries
	GetDependency(dependencyID string) (*Dependency, error)
	ListDependencies() ([]*Dependency, error)
	CheckDependency(dependencyID string) (*DependencyStatus, error)
	
	// Dependency resolution
	ResolveDependencies(requirements []*DependencyRequirement) ([]*Dependency, error)
	ValidateDependencies() (*DependencyValidation, error)
	
	// Lifecycle
	Initialize() error
	Cleanup() error
}

// SystemIntegration defines the interface for system integration
type SystemIntegration interface {
	// Service management
	InstallService(service *ServiceConfig) error
	UninstallService(serviceName string) error
	StartService(serviceName string) error
	StopService(serviceName string) error
	GetServiceStatus(serviceName string) (*ServiceStatus, error)
	
	// System configuration
	ConfigureSystem(config *SystemConfig) error
	CreateDirectories(paths []string) error
	SetPermissions(path string, permissions *Permissions) error
	
	// Environment setup
	SetEnvironmentVariables(vars map[string]string) error
	CreateSymlinks(links map[string]string) error
	UpdatePath(paths []string) error
	
	// Registry operations (Windows)
	SetRegistryKey(key string, value interface{}) error
	GetRegistryKey(key string) (interface{}, error)
	DeleteRegistryKey(key string) error
}

// UserInteraction defines the interface for user interaction during installation
type UserInteraction interface {
	// User prompts
	PromptForPassword(prompt string) (string, error)
	PromptForConfirmation(message string) (bool, error)
	PromptForInput(prompt string, defaultValue string) (string, error)
	PromptForChoice(prompt string, choices []string) (string, error)
	
	// Progress reporting
	ShowProgress(message string, progress float64) error
	ShowMessage(message string, level MessageLevel) error
	ShowError(error error) error
	
	// License and agreements
	ShowLicense(license string) (bool, error)
	ShowAgreement(agreement string) (bool, error)
}

// InstallationConfig represents installation configuration
type InstallationConfig struct {
	Version         string                 `json:"version"`
	InstallPath     string                 `json:"install_path"`
	DataPath        string                 `json:"data_path"`
	ConfigPath      string                 `json:"config_path"`
	CreateService   bool                   `json:"create_service"`
	StartService    bool                   `json:"start_service"`
	CreateShortcuts bool                   `json:"create_shortcuts"`
	UpdatePath      bool                   `json:"update_path"`
	Dependencies    []*DependencyRequirement `json:"dependencies"`
	Options         map[string]interface{} `json:"options,omitempty"`
}

// UninstallationConfig represents uninstallation configuration
type UninstallationConfig struct {
	RemoveData      bool   `json:"remove_data"`
	RemoveConfig    bool   `json:"remove_config"`
	RemoveService   bool   `json:"remove_service"`
	RemoveShortcuts bool   `json:"remove_shortcuts"`
	Force           bool   `json:"force"`
	BackupData      bool   `json:"backup_data"`
	BackupPath      string `json:"backup_path,omitempty"`
}

// UpgradeConfig represents upgrade configuration
type UpgradeConfig struct {
	FromVersion     string                 `json:"from_version"`
	ToVersion       string                 `json:"to_version"`
	BackupData      bool                   `json:"backup_data"`
	MigrateConfig   bool                   `json:"migrate_config"`
	PreserveData    bool                   `json:"preserve_data"`
	Options         map[string]interface{} `json:"options,omitempty"`
}

// InstallationResult represents the result of an installation
type InstallationResult struct {
	Success       bool                   `json:"success"`
	Version       string                 `json:"version"`
	InstallPath   string                 `json:"install_path"`
	Duration      time.Duration          `json:"duration"`
	ComponentsInstalled []string         `json:"components_installed"`
	ServicesCreated     []string         `json:"services_created"`
	Errors        []string               `json:"errors,omitempty"`
	Warnings      []string               `json:"warnings,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
}

// UninstallationResult represents the result of an uninstallation
type UninstallationResult struct {
	Success       bool                   `json:"success"`
	Duration      time.Duration          `json:"duration"`
	ComponentsRemoved []string           `json:"components_removed"`
	ServicesRemoved   []string           `json:"services_removed"`
	DataBackedUp      bool               `json:"data_backed_up"`
	BackupPath        string             `json:"backup_path,omitempty"`
	Errors            []string           `json:"errors,omitempty"`
	Warnings          []string           `json:"warnings,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	Timestamp         time.Time          `json:"timestamp"`
}

// UpgradeResult represents the result of an upgrade
type UpgradeResult struct {
	Success       bool                   `json:"success"`
	FromVersion   string                 `json:"from_version"`
	ToVersion     string                 `json:"to_version"`
	Duration      time.Duration          `json:"duration"`
	ComponentsUpgraded []string          `json:"components_upgraded"`
	DataMigrated       bool              `json:"data_migrated"`
	ConfigMigrated     bool              `json:"config_migrated"`
	BackupCreated      bool              `json:"backup_created"`
	BackupPath         string            `json:"backup_path,omitempty"`
	Errors             []string          `json:"errors,omitempty"`
	Warnings           []string          `json:"warnings,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	Timestamp          time.Time         `json:"timestamp"`
}

// ValidationResult represents the result of installation validation
type ValidationResult struct {
	Valid         bool                   `json:"valid"`
	Version       string                 `json:"version"`
	InstallPath   string                 `json:"install_path"`
	Issues        []ValidationIssue      `json:"issues,omitempty"`
	Recommendations []string             `json:"recommendations,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
}

// ValidationIssue represents an issue found during validation
type ValidationIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Component   string `json:"component"`
	Description string `json:"description"`
	Solution    string `json:"solution,omitempty"`
}

// DependencyResult represents the result of dependency checking
type DependencyResult struct {
	AllSatisfied    bool                 `json:"all_satisfied"`
	Dependencies    []*DependencyStatus  `json:"dependencies"`
	MissingCount    int                  `json:"missing_count"`
	OutdatedCount   int                  `json:"outdated_count"`
	Recommendations []string             `json:"recommendations,omitempty"`
	Timestamp       time.Time            `json:"timestamp"`
}

// IntegrityResult represents the result of integrity verification
type IntegrityResult struct {
	Valid         bool                   `json:"valid"`
	ChecksumValid bool                   `json:"checksum_valid"`
	SignatureValid bool                  `json:"signature_valid"`
	FilesChecked  int                    `json:"files_checked"`
	Issues        []IntegrityIssue       `json:"issues,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
}

// IntegrityIssue represents an integrity issue
type IntegrityIssue struct {
	File        string `json:"file"`
	Type        string `json:"type"`
	Expected    string `json:"expected"`
	Actual      string `json:"actual"`
	Description string `json:"description"`
}

// InstallationStatus represents the current installation status
type InstallationStatus struct {
	Installed     bool                   `json:"installed"`
	Version       string                 `json:"version"`
	InstallPath   string                 `json:"install_path"`
	InstallDate   time.Time              `json:"install_date"`
	LastUpdated   time.Time              `json:"last_updated"`
	Components    []*ComponentStatus     `json:"components"`
	Services      []*ServiceStatus       `json:"services"`
	Health        string                 `json:"health"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ComponentStatus represents the status of an installed component
type ComponentStatus struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Status      string    `json:"status"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	InstallDate time.Time `json:"install_date"`
}

// Dependency represents a software dependency
type Dependency struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Type        string                 `json:"type"`
	Source      string                 `json:"source"`
	Required    bool                   `json:"required"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// DependencyRequirement represents a dependency requirement
type DependencyRequirement struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	MinVersion      string `json:"min_version,omitempty"`
	MaxVersion      string `json:"max_version,omitempty"`
	ExactVersion    string `json:"exact_version,omitempty"`
	Required        bool   `json:"required"`
	InstallIfMissing bool  `json:"install_if_missing"`
}

// DependencyStatus represents the status of a dependency
type DependencyStatus struct {
	Dependency  *Dependency `json:"dependency"`
	Installed   bool        `json:"installed"`
	Version     string      `json:"version,omitempty"`
	Satisfied   bool        `json:"satisfied"`
	Issues      []string    `json:"issues,omitempty"`
}

// DependencyValidation represents the result of dependency validation
type DependencyValidation struct {
	Valid         bool                 `json:"valid"`
	Dependencies  []*DependencyStatus  `json:"dependencies"`
	Conflicts     []DependencyConflict `json:"conflicts,omitempty"`
	Recommendations []string           `json:"recommendations,omitempty"`
}

// DependencyConflict represents a dependency conflict
type DependencyConflict struct {
	Dependency1 string `json:"dependency1"`
	Dependency2 string `json:"dependency2"`
	Reason      string `json:"reason"`
	Resolution  string `json:"resolution,omitempty"`
}

// ServiceConfig represents service configuration
type ServiceConfig struct {
	Name        string            `json:"name"`
	DisplayName string            `json:"display_name"`
	Description string            `json:"description"`
	Executable  string            `json:"executable"`
	Arguments   []string          `json:"arguments,omitempty"`
	WorkingDir  string            `json:"working_dir,omitempty"`
	User        string            `json:"user,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	StartType   string            `json:"start_type"`
	Dependencies []string         `json:"dependencies,omitempty"`
}

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	StartType   string    `json:"start_type"`
	ProcessID   int       `json:"process_id,omitempty"`
	StartTime   time.Time `json:"start_time,omitempty"`
	Description string    `json:"description,omitempty"`
}

// SystemConfig represents system configuration
type SystemConfig struct {
	Directories     []string          `json:"directories"`
	Permissions     map[string]*Permissions `json:"permissions,omitempty"`
	Environment     map[string]string `json:"environment,omitempty"`
	PathAdditions   []string          `json:"path_additions,omitempty"`
	Symlinks        map[string]string `json:"symlinks,omitempty"`
	RegistryKeys    map[string]interface{} `json:"registry_keys,omitempty"`
}

// Permissions represents file/directory permissions
type Permissions struct {
	Owner string `json:"owner,omitempty"`
	Group string `json:"group,omitempty"`
	Mode  string `json:"mode"`
}

// MessageLevel represents the level of a message
type MessageLevel string

const (
	MessageLevelInfo    MessageLevel = "info"
	MessageLevelWarning MessageLevel = "warning"
	MessageLevelError   MessageLevel = "error"
	MessageLevelSuccess MessageLevel = "success"
)

// InstallationEvent represents installation events
type InstallationEvent struct {
	Type      string                 `json:"type"`
	Phase     string                 `json:"phase"`
	Component string                 `json:"component,omitempty"`
	Progress  float64                `json:"progress"`
	Message   string                 `json:"message"`
	Level     MessageLevel           `json:"level"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// EventHandler defines the function signature for installation event handlers
type EventHandler func(event *InstallationEvent) error

// Error types for installation operations
var (
	ErrInstallationFailed   = NewInstallationError("installation failed")
	ErrUninstallationFailed = NewInstallationError("uninstallation failed")
	ErrUpgradeFailed        = NewInstallationError("upgrade failed")
	ErrDependencyMissing    = NewInstallationError("dependency missing")
	ErrPermissionDenied     = NewInstallationError("permission denied")
	ErrInvalidPath          = NewInstallationError("invalid path")
	ErrServiceFailed        = NewInstallationError("service operation failed")
	ErrIntegrityCheckFailed = NewInstallationError("integrity check failed")
)

// InstallationError represents an installation-specific error
type InstallationError struct {
	Message string
	Code    string
}

func (e *InstallationError) Error() string {
	return e.Message
}

func NewInstallationError(message string) *InstallationError {
	return &InstallationError{Message: message}
}
