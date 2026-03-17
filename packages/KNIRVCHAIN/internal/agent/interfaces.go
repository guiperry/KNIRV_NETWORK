package agent

import (
	"context"
	"sync"
	"time"

	"KNIRVCHAIN/internal/database"
)

// AgentChromemStore wraps database.ChromemManager with agent-specific NFT and entity
// persistence. The agent package owns the serialisation logic; the database package owns
// the chromem connection and collection lifecycle.
type AgentChromemStore struct {
	db *database.ChromemManager
	mu sync.RWMutex
}

// NewAgentChromemStore creates an AgentChromemStore backed by an existing ChromemManager.
func NewAgentChromemStore(db *database.ChromemManager) *AgentChromemStore {
	return &AgentChromemStore{db: db}
}

// AgentManagerInterface defines the interface for agent management operations
type AgentManagerInterface interface {
	// Agent lifecycle
	RegisterAgent(agent *Agent) error
	UnregisterAgent(agentID string) error
	GetAgent(agentID string) (*Agent, error)
	GetAllAgents() ([]*Agent, error)

	// Agent operations
	StartAgent(agentID string) error
	StopAgent(agentID string) error
	RestartAgent(agentID string) error
	GetAgentStatus(agentID string) AgentStatus

	// Agent communication
	SendMessage(agentID string, message *AgentMessage) error
	BroadcastMessage(message *AgentMessage) error
	OnMessageReceived(handler MessageHandler) error

	// Lifecycle management
	Start(ctx context.Context) error
	Stop() error
}

// DelegationManager defines the interface for delegation operations
type DelegationManager interface {
	// Delegation operations
	CreateDelegation(delegation *Delegation) error
	AcceptDelegation(delegationID string) error
	RejectDelegation(delegationID string, reason string) error
	RevokeDelegation(delegationID string) error

	// Delegation queries
	GetDelegation(delegationID string) (*Delegation, error)
	GetDelegationsByAgent(agentID string) ([]*Delegation, error)
	GetActiveDelegations() ([]*Delegation, error)

	// Authority management
	CheckAuthority(agentID string, operation string) bool
	GrantAuthority(agentID string, authority *Authority) error
	RevokeAuthority(agentID string, authorityID string) error
}

// FailoverManagerInterface defines the interface for failover operations
type FailoverManagerInterface interface {
	// Failover operations
	DetectFailure(agentID string) error
	TriggerFailover(agentID string) error
	RestoreFromFailover(agentID string) error

	// Failover configuration
	SetFailoverPolicy(policy *FailoverPolicy) error
	GetFailoverPolicy() *FailoverPolicy

	// Health monitoring
	StartHealthMonitoring(ctx context.Context) error
	StopHealthMonitoring() error
	GetHealthStatus(agentID string) HealthStatus

	// Event handling
	OnFailureDetected(handler FailureHandler) error
	OnFailoverCompleted(handler FailoverHandler) error
}

// ConsensusManagerInterface defines the interface for agent consensus operations
type ConsensusManagerInterface interface {
	// Consensus operations
	ProposeDecision(proposal *Proposal) error
	VoteOnProposal(proposalID string, vote Vote) error
	GetConsensusResult(proposalID string) (*ConsensusResult, error)

	// Consensus participation
	JoinConsensus(agentID string) error
	LeaveConsensus(agentID string) error
	GetConsensusParticipants() []string

	// Consensus state
	GetConsensusState() ConsensusState
	IsConsensusReached(proposalID string) bool

	// Event handling
	OnConsensusReached(handler ConsensusHandler) error
	OnProposalSubmitted(handler ProposalHandler) error
}

// CapabilityManager defines the interface for agent capability management
type CapabilityManager interface {
	// Capability registration
	RegisterCapability(agentID string, capability *Capability) error
	UnregisterCapability(agentID string, capabilityID string) error

	// Capability queries
	GetCapabilities(agentID string) ([]*Capability, error)
	FindAgentsByCapability(capabilityType string) ([]string, error)
	CheckCapability(agentID string, capabilityType string) bool

	// Capability execution
	ExecuteCapability(agentID string, capabilityID string, params map[string]interface{}) (*CapabilityResult, error)

	// Capability discovery
	DiscoverCapabilities() ([]*Capability, error)
	GetCapabilityMetadata(capabilityID string) (*CapabilityMetadata, error)
}

// Agent type is defined in agent_manager.go

// AgentStatus represents the status of an agent
type AgentStatus struct {
	State      string        `json:"state"`  // running, stopped, error, etc.
	Health     string        `json:"health"` // healthy, degraded, unhealthy
	LastUpdate time.Time     `json:"last_update"`
	ErrorCount int           `json:"error_count"`
	LastError  string        `json:"last_error,omitempty"`
	Uptime     time.Duration `json:"uptime"`
	CPU        float64       `json:"cpu_usage"`
	Memory     uint64        `json:"memory_usage"`
}

// AgentMessage represents a message between agents
type AgentMessage struct {
	ID        string                 `json:"id"`
	From      string                 `json:"from"`
	To        string                 `json:"to,omitempty"`
	Type      string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
	Priority  int                    `json:"priority"`
}

// Delegation represents a delegation of authority between agents
type Delegation struct {
	ID        string     `json:"id"`
	From      string     `json:"from"`
	To        string     `json:"to"`
	Authority *Authority `json:"authority"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt time.Time  `json:"expires_at"`
	Reason    string     `json:"reason,omitempty"`
}

// Authority represents authority that can be delegated
type Authority struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Operations  []string               `json:"operations"`
	Scope       string                 `json:"scope,omitempty"`
	Constraints map[string]interface{} `json:"constraints,omitempty"`
}

// FailoverPolicy defines the policy for failover operations
type FailoverPolicy struct {
	MaxFailures         int           `json:"max_failures"`
	FailureWindow       time.Duration `json:"failure_window"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	FailoverTimeout     time.Duration `json:"failover_timeout"`
	BackupAgents        []string      `json:"backup_agents"`
	AutoRestore         bool          `json:"auto_restore"`
}

// HealthStatus represents the health status of an agent
type HealthStatus struct {
	AgentID      string             `json:"agent_id"`
	Healthy      bool               `json:"healthy"`
	LastCheck    time.Time          `json:"last_check"`
	FailureCount int                `json:"failure_count"`
	Metrics      map[string]float64 `json:"metrics"`
}

// Proposal represents a proposal for consensus
type Proposal struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Proposer      string                 `json:"proposer"`
	Content       map[string]interface{} `json:"content"`
	Timestamp     time.Time              `json:"timestamp"`
	ExpiresAt     time.Time              `json:"expires_at"`
	RequiredVotes int                    `json:"required_votes"`
}

// Vote represents a vote on a proposal
type Vote struct {
	ProposalID string    `json:"proposal_id"`
	AgentID    string    `json:"agent_id"`
	Decision   bool      `json:"decision"`
	Reason     string    `json:"reason,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// ConsensusResult represents the result of a consensus process
type ConsensusResult struct {
	ProposalID string    `json:"proposal_id"`
	Approved   bool      `json:"approved"`
	VoteCount  int       `json:"vote_count"`
	Votes      []*Vote   `json:"votes"`
	ReachedAt  time.Time `json:"reached_at"`
}

// ConsensusState represents the current consensus state
type ConsensusState struct {
	ActiveProposals int       `json:"active_proposals"`
	Participants    []string  `json:"participants"`
	LastConsensus   time.Time `json:"last_consensus"`
	TotalVotes      int       `json:"total_votes"`
}

// Capability type is defined in agent_manager.go

// CapabilityParameter represents a parameter for a capability
type CapabilityParameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Description string      `json:"description,omitempty"`
}

// CapabilityResult represents the result of capability execution
type CapabilityResult struct {
	Success   bool                   `json:"success"`
	Result    map[string]interface{} `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Duration  time.Duration          `json:"duration"`
	Timestamp time.Time              `json:"timestamp"`
}

// CapabilityMetadata represents metadata about a capability
type CapabilityMetadata struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Version       string    `json:"version"`
	Author        string    `json:"author"`
	Documentation string    `json:"documentation"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Event handler function types
type MessageHandler func(message *AgentMessage) error
type FailureHandler func(agentID string, failure *FailureEvent) error
type FailoverHandler func(agentID string, result *FailoverResult) error
type ConsensusHandler func(result *ConsensusResult) error
type ProposalHandler func(proposal *Proposal) error

// FailureEvent represents a failure event
type FailureEvent struct {
	AgentID   string    `json:"agent_id"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Severity  string    `json:"severity"`
}

// FailoverResult represents the result of a failover operation
type FailoverResult struct {
	OriginalAgent string        `json:"original_agent"`
	BackupAgent   string        `json:"backup_agent"`
	Success       bool          `json:"success"`
	Duration      time.Duration `json:"duration"`
	Timestamp     time.Time     `json:"timestamp"`
}

// AgentConfig represents agent configuration
type AgentConfig struct {
	MaxAgents           int           `json:"max_agents"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	MessageTimeout      time.Duration `json:"message_timeout"`
	FailoverEnabled     bool          `json:"failover_enabled"`
	ConsensusEnabled    bool          `json:"consensus_enabled"`
	CapabilityDiscovery bool          `json:"capability_discovery"`
}

// Error types for agent operations
var (
	ErrAgentNotFound         = NewAgentError("agent not found")
	ErrAgentAlreadyExists    = NewAgentError("agent already exists")
	ErrDelegationFailed      = NewAgentError("delegation failed")
	ErrFailoverFailed        = NewAgentError("failover failed")
	ErrConsensusTimeout      = NewAgentError("consensus timeout")
	ErrCapabilityNotFound    = NewAgentError("capability not found")
	ErrInsufficientAuthority = NewAgentError("insufficient authority")
)

// AgentError represents an agent-specific error
type AgentError struct {
	Message string
	Code    string
}

func (e *AgentError) Error() string {
	return e.Message
}

func NewAgentError(message string) *AgentError {
	return &AgentError{Message: message}
}
