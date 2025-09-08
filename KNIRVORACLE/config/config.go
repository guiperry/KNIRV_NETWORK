package config

import (
	"KNIRVORACLE/utils"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/google/uuid"
	circuit "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	"github.com/spf13/viper"
)

// EmbeddedRootKeyData is defined in build-tag specific files:
// - embedded_key_testnet.go (for testnet builds with -tags testnet)
// - embedded_key_production.go (for production builds without tags)

const AppName = "KNIRVORACLE"

// Role type (if not already defined elsewhere, or import if it is)
type Role string

const (
	Root           Role = "Root"
	RoleBootnode   Role = "Bootnode"
	RoleReflection Role = "Reflection"
	RolePeer       Role = "Peer"
	RoleClient     Role = "Client"
	// Add other roles if any
)

func (r Role) String() string {
	return string(r)
}

// DetermineRole determines the node's role based on flags.
// This function can be used both with command-line flags and with Config struct fields
func DetermineRole(IsRoot, IsBootnode, IsPeer, IsClientOnly bool) Role {
	if IsRoot {
		return Root
	}
	if IsBootnode {
		return RoleBootnode
	}
	if IsPeer {
		return RolePeer
	}
	if IsClientOnly {
		return RoleClient
	}
	return RoleClient // Default to Client if no specific role flag is set
}

// DetermineRoleFromConfig determines the node's role based on build flags first, then Config struct fields
func DetermineRoleFromConfig(cfg *Config) Role {
	// Check for build flags first (these will be set at compile time)
	// The actual build flag checking is done in the role-specific entry point files

	// If no build flags are set, fall back to config values
	return DetermineRole(cfg.IsRoot, cfg.IsBootnode, cfg.IsPeer, cfg.ClientOnly)
}

// ExperimentalFeaturesConfig holds configuration for experimental features
type ExperimentalFeaturesConfig struct {
	Enabled bool `json:"enabled" mapstructure:"enabled"`
	Debug   bool `json:"debug" mapstructure:"debug"`
}

// PaymentProcessorConfig holds configuration for the payment processor
type PaymentProcessorConfig struct {
	Enabled               bool    `json:"enabled" mapstructure:"enabled"`
	NodeRPC               string  `json:"node_rpc" mapstructure:"nodeRPC"` // URL of the KNIRVORACLE node
	WebhookPort           int     `json:"webhook_port" mapstructure:"webhookPort"`
	StripeSecretKey       string  `json:"stripe_secret_key" mapstructure:"stripeSecretKey"`
	StripeWebhookSecret   string  `json:"stripe_webhook_secret" mapstructure:"stripeWebhookSecret"`
	CoinbaseAPIKey        string  `json:"coinbase_api_key" mapstructure:"coinbaseAPIKey"`
	CoinbaseWebhookSecret string  `json:"coinbase_webhook_secret" mapstructure:"coinbaseWebhookSecret"`
	TokenSymbol           string  `json:"token_symbol" mapstructure:"tokenSymbol"`
	TokenDecimals         int     `json:"token_decimals" mapstructure:"tokenDecimals"`
	USDPerToken           float64 `json:"usd_per_token" mapstructure:"usdPerToken"`
	ETHPerToken           float64 `json:"eth_per_token" mapstructure:"ethPerToken"`
}

// BootnodeConfig holds configuration specific to bootnodes
type BootnodeConfig struct {
	Enabled            bool   `json:"enabled"`
	ServiceUser        string `json:"service_user,omitempty"`
	ServiceWorkingDir  string `json:"service_working_dir,omitempty"`
	ServiceLogPath     string `json:"service_log_path,omitempty" mapstructure:"serviceLogPath"`
	ServiceDisplayName string `json:"service_display_name,omitempty" mapstructure:"serviceDisplayName"`
	ServiceDescription string `json:"service_description,omitempty" mapstructure:"serviceDescription"`
}

// RootConstants holds constants from the main package that are needed for root configuration
type RootConstants struct {
	BlockchainAddress    string
	BlockchainPrivateKey string
	RootchainURL         string
	BlockchainName       string
	CurrencyName         string
	Decimal              int
	MiningDifficulty     int
	MiningReward         int
}

// Global root constants that can be set at initialization
var rootConstants RootConstants

// SetRootConstants sets the global root constants
func SetRootConstants(constants RootConstants) {
	rootConstants = constants
	log.Printf("Root constants set: BlockchainName=%s, CurrencyName=%s",
		constants.BlockchainName, constants.CurrencyName)
}

// ApplyTestnetDefaults applies testnet-specific configuration defaults
func ApplyTestnetDefaults(cfg *Config) {
	if cfg.Testnet.Enabled {
		// Set testnet-specific defaults
		if cfg.Testnet.ChainID == "" {
			cfg.Testnet.ChainID = "knirv-testnet-1"
		}
		if cfg.Testnet.APIPort == 0 {
			cfg.Testnet.APIPort = 1317
		}
		if cfg.Testnet.RPCPort == 0 {
			cfg.Testnet.RPCPort = 26657
		}
		if cfg.Testnet.P2PPort == 0 {
			cfg.Testnet.P2PPort = 26656
		}
		if cfg.Testnet.Validators == 0 {
			cfg.Testnet.Validators = 3
		}
		if cfg.Testnet.InitialNRN == 0 {
			cfg.Testnet.InitialNRN = 1000000000000
		}
		if cfg.Testnet.LogLevel == "" {
			cfg.Testnet.LogLevel = "debug"
		}

		// Apply testnet overrides to main config
		cfg.ChainID = cfg.Testnet.ChainID
		cfg.Port = uint64(cfg.Testnet.APIPort)
		cfg.P2PPort = uint64(cfg.Testnet.P2PPort)
		cfg.Testnet.DisableXIONBridge = true
		cfg.Testnet.SimplifiedConsensus = true

		// Enable network monitor in testnet mode
		cfg.NetworkMonitor.Enabled = true
		cfg.NetworkMonitor.AutoStart = true
		cfg.NetworkMonitor.WebMode = true
		if cfg.NetworkMonitor.Port == 0 {
			cfg.NetworkMonitor.Port = 8091
		}

		log.Printf("Applied testnet defaults: ChainID=%s, APIPort=%d, RPCPort=%d, P2PPort=%d, NetworkMonitor=%v",
			cfg.Testnet.ChainID, cfg.Testnet.APIPort, cfg.Testnet.RPCPort, cfg.Testnet.P2PPort, cfg.NetworkMonitor.Enabled)
	}
}

// MergeConfigs merges two configs, with src values overriding dst where set
func MergeConfigs(dst, src *Config) *Config {
	if dst == nil {
		return src
	}
	if src == nil {
		return dst
	}

	merged := *dst

	// Merge simple fields
	if src.Port != 0 {
		merged.Port = src.Port
	}
	if src.ConsensusPauseTime != 0 {
		merged.ConsensusPauseTime = src.ConsensusPauseTime
	}
	if src.P2PPort != 0 {
		merged.P2PPort = src.P2PPort
	}
	if src.WalletPort != 0 {
		merged.WalletPort = src.WalletPort
	}
	if src.BlockchainDatabasePath != "" {
		merged.BlockchainDatabasePath = src.BlockchainDatabasePath
	}
	if src.SearchableDatabasePath != "" {
		merged.SearchableDatabasePath = src.SearchableDatabasePath
	}
	if src.MinersAddress != "" {
		merged.MinersAddress = src.MinersAddress
	}
	if src.MasterAddress != "" {
		merged.MasterAddress = src.MasterAddress
	}
	merged.NoWalletServer = src.NoWalletServer
	merged.ClientOnly = src.ClientOnly
	merged.UseGUI = src.UseGUI
	if len(src.ReflectionURLs) > 0 {
		merged.ReflectionURLs = src.ReflectionURLs
	}
	if src.ChainID != "" {
		merged.ChainID = src.ChainID
	}
	merged.InstallComplete = src.InstallComplete
	merged.IsRoot = src.IsRoot
	merged.IsBootnode = src.IsBootnode
	merged.IsPeer = src.IsPeer
	if src.ReflectionHTTPPort != 0 {
		merged.ReflectionHTTPPort = src.ReflectionHTTPPort
	}
	if src.ReflectionP2PPort != 0 {
		merged.ReflectionP2PPort = src.ReflectionP2PPort
	}

	// Merge nested configs
	if src.PaymentProcessor != (PaymentProcessorConfig{}) {
		merged.PaymentProcessor = src.PaymentProcessor
	}
	if src.Bootnode != (BootnodeConfig{}) {
		merged.Bootnode = src.Bootnode
	}

	// Merge NodeJSServices config
	if src.NodeJSServices.Enabled {
		merged.NodeJSServices.Enabled = true
	}
	if src.NodeJSServices.TunnelRegistry.Enabled {
		merged.NodeJSServices.TunnelRegistry.Enabled = true
	}
	if src.NodeJSServices.TunnelRegistry.ScriptPath != "" {
		merged.NodeJSServices.TunnelRegistry.ScriptPath = src.NodeJSServices.TunnelRegistry.ScriptPath
	}
	if src.NodeJSServices.TunnelRegistry.HTTPPort != 0 {
		merged.NodeJSServices.TunnelRegistry.HTTPPort = src.NodeJSServices.TunnelRegistry.HTTPPort
	}
	if src.NodeJSServices.TunnelRegistry.ControlPort != 0 {
		merged.NodeJSServices.TunnelRegistry.ControlPort = src.NodeJSServices.TunnelRegistry.ControlPort
	}
	if src.NodeJSServices.TunnelRegistry.PublicRelayPort != 0 {
		merged.NodeJSServices.TunnelRegistry.PublicRelayPort = src.NodeJSServices.TunnelRegistry.PublicRelayPort
	}
	if src.NodeJSServices.TunnelRegistry.STUNPort != 0 {
		merged.NodeJSServices.TunnelRegistry.STUNPort = src.NodeJSServices.TunnelRegistry.STUNPort
	}
	if src.NodeJSServices.TunnelRegistry.ServerPublicHost != "" {
		merged.NodeJSServices.TunnelRegistry.ServerPublicHost = src.NodeJSServices.TunnelRegistry.ServerPublicHost
	}

	if src.NodeJSServices.PaymentGateway.Enabled {
		merged.NodeJSServices.PaymentGateway.Enabled = true
	}
	if src.NodeJSServices.PaymentGateway.ScriptPath != "" {
		merged.NodeJSServices.PaymentGateway.ScriptPath = src.NodeJSServices.PaymentGateway.ScriptPath
	}
	if src.NodeJSServices.PaymentGateway.HTTPPort != 0 {
		merged.NodeJSServices.PaymentGateway.HTTPPort = src.NodeJSServices.PaymentGateway.HTTPPort
	}
	if src.NodeJSServices.PaymentGateway.APIKey != "" {
		merged.NodeJSServices.PaymentGateway.APIKey = src.NodeJSServices.PaymentGateway.APIKey
	}

	if src.NodeJSServices.DeveloperPortal.Enabled {
		merged.NodeJSServices.DeveloperPortal.Enabled = true
	}
	if src.NodeJSServices.DeveloperPortal.ScriptPath != "" {
		merged.NodeJSServices.DeveloperPortal.ScriptPath = src.NodeJSServices.DeveloperPortal.ScriptPath
	}
	if src.NodeJSServices.DeveloperPortal.HTTPPort != 0 {
		merged.NodeJSServices.DeveloperPortal.HTTPPort = src.NodeJSServices.DeveloperPortal.HTTPPort
	}
	if src.NodeJSServices.DeveloperPortal.APIKey != "" {
		merged.NodeJSServices.DeveloperPortal.APIKey = src.NodeJSServices.DeveloperPortal.APIKey
	}

	// Merge new Node.js services
	if src.NodeJSServices.BootnodeRegistry.Enabled {
		merged.NodeJSServices.BootnodeRegistry.Enabled = true
	}
	if src.NodeJSServices.BootnodeRegistry.ScriptPath != "" {
		merged.NodeJSServices.BootnodeRegistry.ScriptPath = src.NodeJSServices.BootnodeRegistry.ScriptPath
	}
	if src.NodeJSServices.BootnodeRegistry.HTTPPort != 0 {
		merged.NodeJSServices.BootnodeRegistry.HTTPPort = src.NodeJSServices.BootnodeRegistry.HTTPPort
	}

	if src.NodeJSServices.NotarySystem.Enabled {
		merged.NodeJSServices.NotarySystem.Enabled = true
	}
	if src.NodeJSServices.NotarySystem.ScriptPath != "" {
		merged.NodeJSServices.NotarySystem.ScriptPath = src.NodeJSServices.NotarySystem.ScriptPath
	}
	if src.NodeJSServices.NotarySystem.HTTPPort != 0 {
		merged.NodeJSServices.NotarySystem.HTTPPort = src.NodeJSServices.NotarySystem.HTTPPort
	}

	if src.NodeJSServices.NetworkMonitor.Enabled {
		merged.NodeJSServices.NetworkMonitor.Enabled = true
	}
	if src.NodeJSServices.NetworkMonitor.ScriptPath != "" {
		merged.NodeJSServices.NetworkMonitor.ScriptPath = src.NodeJSServices.NetworkMonitor.ScriptPath
	}
	if src.NodeJSServices.NetworkMonitor.HTTPPort != 0 {
		merged.NodeJSServices.NetworkMonitor.HTTPPort = src.NodeJSServices.NetworkMonitor.HTTPPort
	}

	// NANDA-ANS
	if src.NodeJSServices.NANDAANS.Enabled {
		merged.NodeJSServices.NANDAANS.Enabled = true
	}
	if src.NodeJSServices.NANDAANS.HTTPPort != 0 {
		merged.NodeJSServices.NANDAANS.HTTPPort = src.NodeJSServices.NANDAANS.HTTPPort
	}

	// Merge TunnelClient config
	if src.TunnelClient.Enabled {
		merged.TunnelClient.Enabled = true
	}
	if src.TunnelClient.ServerAddress != "" {
		merged.TunnelClient.ServerAddress = src.TunnelClient.ServerAddress
	}
	if src.TunnelClient.ControlPort != 0 {
		merged.TunnelClient.ControlPort = src.TunnelClient.ControlPort
	}
	if src.TunnelClient.PingInterval != 0 {
		merged.TunnelClient.PingInterval = src.TunnelClient.PingInterval
	}
	if src.TunnelClient.ReconnectDelay != 0 {
		merged.TunnelClient.ReconnectDelay = src.TunnelClient.ReconnectDelay
	}

	return &merged
}

// DataEngineConfig defines settings for the data engine
type DataEngineConfig struct {
	Enabled          bool     `mapstructure:"enabled" json:"enabled"`
	KafkaBrokers     []string `mapstructure:"kafka_brokers" json:"kafka_brokers"`
	KafkaClientID    string   `mapstructure:"kafka_client_id" json:"kafka_client_id"`
	ChromaDBURL      string   `mapstructure:"chromadb_url" json:"chromadb_url"`
	ChromaCollection string   `mapstructure:"chroma_collection" json:"chroma_collection"`
	EnableKafka      bool     `mapstructure:"enable_kafka" json:"enable_kafka"`
	EnableChromaDB   bool     `mapstructure:"enable_chromadb" json:"enable_chromadb"`
	EnableWebSocket  bool     `mapstructure:"enable_websocket" json:"enable_websocket"`
	EnableRESTAPI    bool     `mapstructure:"enable_restapi" json:"enable_restapi"`
	WebSocketPort    int      `mapstructure:"websocket_port" json:"websocket_port"`
	RESTAPIPort      int      `mapstructure:"restapi_port" json:"restapi_port"`
	WindowSize       string   `mapstructure:"window_size" json:"window_size"`           // Duration string like "5m"
	MetricsInterval  string   `mapstructure:"metrics_interval" json:"metrics_interval"` // Duration string like "10s"
}

// InferenceEngineConfig defines settings for the inference engine
type InferenceEngineConfig struct {
	Enabled         bool   `mapstructure:"enabled" json:"enabled"`
	PluginsDir      string `mapstructure:"plugins_dir" json:"plugins_dir"`
	DefaultAgentID  string `mapstructure:"default_agent_id" json:"default_agent_id"`
	DefaultVersion  string `mapstructure:"default_version" json:"default_version"`
	ShareDHTMetrics bool   `mapstructure:"share_dht_metrics" json:"share_dht_metrics"` // Share metrics with DHT
	APIPort         int    `mapstructure:"api_port" json:"api_port"`                   // Port for inference API
}

// AgentModeConfig defines settings for agent mode operation
type AgentModeConfig struct {
	Enabled bool `mapstructure:"enabled" json:"enabled"`
}

// TestnetConfig defines settings for testnet mode operation
type TestnetConfig struct {
	Enabled             bool   `mapstructure:"enabled" json:"enabled"`
	ChainID             string `mapstructure:"chain_id" json:"chain_id"`
	APIPort             int    `mapstructure:"api_port" json:"api_port"`
	RPCPort             int    `mapstructure:"rpc_port" json:"rpc_port"`
	P2PPort             int    `mapstructure:"p2p_port" json:"p2p_port"`
	Validators          int    `mapstructure:"validators" json:"validators"`
	InitialNRN          int64  `mapstructure:"initial_nrn" json:"initial_nrn"`
	DisableXIONBridge   bool   `mapstructure:"disable_xion_bridge" json:"disable_xion_bridge"`
	SimplifiedConsensus bool   `mapstructure:"simplified_consensus" json:"simplified_consensus"`
	LogLevel            string `mapstructure:"log_level" json:"log_level"`
}

// ReverseProxyConfig defines settings for the built-in Go reverse proxy
type ReverseProxyConfig struct {
	Enabled         bool   `mapstructure:"enabled" json:"enabled"`
	ListenAddr      string `mapstructure:"listen_addr" json:"listen_addr"`             // e.g., ":80" or ":443"
	EmbedDataEngine bool   `mapstructure:"embed_data_engine" json:"embed_data_engine"` // Whether to embed DataEngine handlers
	// Add TLS cert/key paths here if you want the Go proxy to handle HTTPS
}

// RelayConfig defines circuit relay configuration for libp2p
type RelayConfig struct {
	Enabled            bool              `json:"enabled" mapstructure:"enabled"`
	Resources          circuit.Resources `json:"resources" mapstructure:"resources"`
	AdvertiseInterval  time.Duration     `json:"advertise_interval" mapstructure:"advertise_interval"`
	DiscoveryNamespace string            `json:"discovery_namespace" mapstructure:"discovery_namespace"`
}

// NodeJSServicesConfig configuration (for Root mode)
type NodeJSServicesConfig struct {
	Enabled bool `json:"enabled"` // Master switch for Node.js services in Root mode

	// TunnelRegistry is the combined tunnel and registry service
	TunnelRegistry struct {
		Enabled          bool   `json:"enabled"`
		ScriptPath       string `json:"script_path"`        // Path to tunnel-registry-service.js
		HTTPPort         uint   `json:"http_port"`          // Port for HTTP API (e.g., 3003)
		ControlPort      uint   `json:"control_port"`       // Port for internal nodes (e.g., 4001)
		PublicRelayPort  uint   `json:"public_relay_port"`  // Port for external clients (e.g., 4000)
		STUNPort         uint   `json:"stun_port"`          // Port for STUN service (e.g., 3478)
		ServerPublicHost string `json:"server_public_host"` // Public hostname/IP of the server
	} `json:"tunnel_registry"`

	// PaymentGateway is a separate service
	PaymentGateway struct {
		Enabled    bool   `json:"enabled"`
		ScriptPath string `json:"script_path"`    // Path to payment-gateway-service.js
		HTTPPort   uint   `json:"http_port"`      // Port for gateway's API (e.g., 3004)
		APIKey     string `json:"api_key_secret"` // API key for interacting with payment providers
	} `json:"payment_gateway"`

	// DeveloperPortal is the Next.js based UI
	DeveloperPortal struct {
		Enabled    bool   `json:"enabled"`
		ScriptPath string `json:"script_path"`    // Path to developer-portal server.js
		HTTPPort   uint   `json:"http_port"`      // Port for portal's API (e.g., 3005)
		APIKey     string `json:"api_key_secret"` // API key for interacting with the portal
	} `json:"developer_portal"`

	// BootnodeRegistry service
	BootnodeRegistry struct {
		Enabled    bool   `json:"enabled"`
		ScriptPath string `json:"script_path"` // Path to bootnode-registry-service.js
		HTTPPort   uint   `json:"http_port"`   // Port for bootnode registry API (e.g., 3006)
	} `json:"bootnode_registry"`

	// NotarySystem service
	NotarySystem struct {
		Enabled    bool   `json:"enabled"`
		ScriptPath string `json:"script_path"` // Path to notary-system-service.js
		HTTPPort   uint   `json:"http_port"`   // Port for notary system API (e.g., 3007)
	} `json:"notary_system"`

	// NetworkMonitor service
	NetworkMonitor struct {
		Enabled    bool   `json:"enabled"`
		ScriptPath string `json:"script_path"` // Path to network-monitor-service.js
		HTTPPort   uint   `json:"http_port"`   // Port for network monitor API (e.g., 3008)
	} `json:"network_monitor"`

	// NANDA-ANS service (served as static files from Go binary via main HTTP server)
	NANDAANS struct {
		Enabled  bool `json:"enabled" mapstructure:"enabled"`
		HTTPPort uint `json:"http_port" mapstructure:"http_port"` // Port for NANDA-ANS static files (0 = use main HTTP server port)
	} `json:"nanda_ans" mapstructure:"nanda_ans"`
}

// TunnelClientConfig configuration for nodes that need NAT traversal
type TunnelClientConfig struct {
	Enabled        bool   `json:"enabled"`         // Whether to connect to a tunnel server
	ServerAddress  string `json:"server_address"`  // Address of the tunnel server
	ControlPort    uint   `json:"control_port"`    // Port for control connection
	PingInterval   uint   `json:"ping_interval"`   // Seconds between ping messages (default: 30)
	ReconnectDelay uint   `json:"reconnect_delay"` // Seconds to wait before reconnect attempts (default: 5)
}

// TerminalIntegration represents terminal integration configuration
type TerminalIntegration struct {
	Enabled           bool   `json:"enabled"`
	ZshCompletionsDir string `json:"zsh_completions_dir"`
	Theme             string `json:"theme"`
}

// PoAuDConfig holds configuration for PoAu-D consensus mechanism
type PoAuDConfig struct {
	Enabled                 bool          `json:"enabled" mapstructure:"enabled"`
	DelegationInterval      time.Duration `json:"delegation_interval" mapstructure:"delegation_interval"`
	MaxSubpoolStaleTime     time.Duration `json:"max_subpool_stale_time" mapstructure:"max_subpool_stale_time"`
	MaxPapSubpoolQueue      int           `json:"max_pap_subpool_queue" mapstructure:"max_pap_subpool_queue"`
	StatusAdvertiseInterval time.Duration `json:"status_advertise_interval" mapstructure:"status_advertise_interval"`
}

// Config holds the application configuration
type Config struct {
	NodeName               string                     `json:"node_name,omitempty" mapstructure:"node_name,nodeName"` // Node name for identification
	ExperimentalFeatures   ExperimentalFeaturesConfig `json:"experimental_features" mapstructure:"experimentalFeatures"`
	Port                   uint64                     `json:"port" mapstructure:"port,httpPort"`        // Accept both port and httpPort in config
	P2PPort                uint64                     `json:"p2p_port" mapstructure:"p2p_port,p2pPort"` // Accept both p2p_port and p2pPort in config
	Relay                  RelayConfig                `json:"relay" mapstructure:"relay"`               // Circuit relay configuration
	ConsensusPauseTime     int                        `json:"consensus_pause_time" mapstructure:"consensus_pause_time,consensusPauseTime"`
	WalletPort             uint64                     `json:"wallet_port" mapstructure:"wallet_port,walletPort"`
	AltGUIPort             uint64                     `json:"alt_gui_port" mapstructure:"alt_gui_port,altGUIPort"` // Port for alternate GUI (Next.js)
	BlockchainDatabasePath string                     `json:"shared_database_path" mapstructure:"shared_database_path,sharedDatabasePath"`
	SearchableDatabasePath string                     `json:"searchable_database_path,omitempty" mapstructure:"searchable_database_path,searchableDatabasePath"` // Specific for dev mode, corrected tag
	ReflectionDatabasePath string                     `json:"reflection_database_path,omitempty" mapstructure:"reflection_database_path,reflectionDatabasePath"` // Specific for network mode
	MinersAddress          string                     `json:"miners_address" mapstructure:"miners_address,minersAddress"`
	MasterAddress          string                     `json:"master_address,omitempty" mapstructure:"master_address,masterAddress"`
	NoWalletServer         bool                       `json:"no_wallet_server" mapstructure:"no_wallet_server,noWalletServer"`
	ClientOnly             bool                       `json:"client_only" mapstructure:"client_only,clientOnly"`
	UseGUI                 bool                       `json:"use_gui" mapstructure:"use_gui,useGUI"`
	ReflectionURLs         []string                   `json:"reflection_urls" mapstructure:"reflection_urls,reflectionURLs"`
	ChainID                string                     `json:"chain_id" mapstructure:"chainID"` // Changed to just "chainID" to exactly match JSON key
	InstallComplete        bool                       `json:"install_complete" mapstructure:"install_complete,installComplete"`
	IsRoot                 bool                       `json:"is_root" mapstructure:"is_root,IsRoot"`
	IsBootnode             bool                       `json:"is_bootnode" mapstructure:"is_bootnode,IsBootnode"`            // General bootnode flag
	IsPeer                 bool                       `json:"is_dev" mapstructure:"is_dev,IsPeer"`                          // New field
	IsNetworkMode          bool                       `json:"is_network_mode" mapstructure:"is_network_mode,isNetworkMode"` // Network mode flag
	PaymentProcessor       PaymentProcessorConfig     `json:"payment_processor" mapstructure:"paymentProcessor"`
	Bootnode               BootnodeConfig             `json:"bootnode_settings" mapstructure:"bootnodeSettings"`                // Specific bootnode settings
	ReflectionHTTPPort     uint64                     `json:"reflection_http_port,omitempty" mapstructure:"reflectionHTTPPort"` // Specific for network mode
	ReflectionP2PPort      uint64                     `json:"reflection_p2p_port,omitempty" mapstructure:"reflectionP2PPort"`   // Specific for network mode
	NodeJSServices         NodeJSServicesConfig       `json:"node_js_services" mapstructure:"nodeJSServices"`                   // Node.js services configuration
	TunnelClient           TunnelClientConfig         `json:"tunnel_client" mapstructure:"tunnelClient"`                        // Tunnel client configuration
	ReverseProxy           ReverseProxyConfig         `json:"reverse_proxy" mapstructure:"reverse_proxy"`
	DataEngine             DataEngineConfig           `json:"data_engine" mapstructure:"data_engine"`               // Data engine configuration
	InferenceEngine        InferenceEngineConfig      `json:"inference_engine" mapstructure:"inference_engine"`     // Inference engine configuration
	AgentMode              AgentModeConfig            `json:"agent_mode" mapstructure:"agent_mode"`                 // Agent mode configuration
	Testnet                TestnetConfig              `json:"testnet" mapstructure:"testnet"`                       // Testnet configuration
	PublicIPInfo           map[string]interface{}     `json:"public_ip_info,omitempty" mapstructure:"publicIPInfo"` // Stores the full JSON response from IPinfo.io
	Chromem                ChromemConfig              `json:"chromem_config" mapstructure:"chromem"`                // Add Chromem config struct
	P2P                    struct {
		RootNodeURI string `json:"root_node_uri" mapstructure:"rootNodeURI"`
	} `json:"p2p" mapstructure:"p2p"`
	TerminalIntegration *TerminalIntegration `json:"terminal_integration"`

	// PoAu-D specific configuration
	PoAuD PoAuDConfig `json:"poaud" mapstructure:"poaud"`

	// Network Monitor configuration
	NetworkMonitor NetworkMonitorConfig `json:"network_monitor" mapstructure:"network_monitor"`
}

// Add a struct to hold Chromem-specific configuration within the main config
type ChromemConfig struct {
	Path           string          // Directory path for persistent storage// Path is now part of the main Config struct (SearchableDatabasePath)
	CerebrasConfig *CerebrasConfig `mapstructure:"cerebras_config"`
}

// CerebrasConfig holds configuration for Cerebras embedding function
type CerebrasConfig struct {
	APIKey  string `json:"api_key,omitempty" mapstructure:"api_key"`   // API key for Cerebras service
	BaseURL string `json:"base_url,omitempty" mapstructure:"base_url"` // Base URL for Cerebras API
}

// NetworkMonitorConfig defines settings for the embedded network monitor
type NetworkMonitorConfig struct {
	Enabled   bool `json:"enabled" mapstructure:"enabled"`       // Whether to enable network monitor
	WebMode   bool `json:"web_mode" mapstructure:"web_mode"`     // Run in web mode (headless)
	Port      int  `json:"port" mapstructure:"port"`             // Port for web interface
	AutoStart bool `json:"auto_start" mapstructure:"auto_start"` // Auto-start with KNIRVORACLE
}

// DefaultConfig returns a default configuration
// This is a base configuration that will be customized by role-specific settings
func DefaultConfig() *Config {
	return &Config{
		NodeName: "KNIRVORACLE Node", // Default node name
		ExperimentalFeatures: ExperimentalFeaturesConfig{
			Enabled: false, // Experimental features disabled by default
			Debug:   false, // Debug mode disabled by default
		},
		Port:    0, // Changed from 5050 to 0 to ensure role defaults are applied
		P2PPort: 0, // Changed from 6050 to 0 to ensure role defaults are applied
		DataEngine: DataEngineConfig{
			Enabled:          true, // Changed from false to true to enable data engine by default for all roles
			KafkaBrokers:     []string{"localhost:9092"},
			KafkaClientID:    "KNIRVORACLE-client",
			ChromaDBURL:      "http://localhost:8000",
			ChromaCollection: "KNIRVORACLE-events",
			EnableKafka:      false,
			EnableChromaDB:   false,
			EnableWebSocket:  true,
			EnableRESTAPI:    true,
			WebSocketPort:    8080,
			RESTAPIPort:      7080,
			WindowSize:       "5m",
			MetricsInterval:  "10s",
		},
		InferenceEngine: InferenceEngineConfig{
			Enabled:         false, // Disabled by default, enabled when agent flag is used
			PluginsDir:      "plugins",
			DefaultAgentID:  "default",
			DefaultVersion:  "latest",
			ShareDHTMetrics: true, // Share metrics with DHT when enabled
			APIPort:         9080,
		},
		AgentMode: AgentModeConfig{
			Enabled: false, // Disabled by default, enabled when agent flag is used
		},
		Relay: RelayConfig{
			Enabled: false,
			Resources: circuit.Resources{
				MaxCircuits:            128,
				MaxReservations:        128,
				ReservationTTL:         time.Hour,
				MaxReservationsPerPeer: 4,
				MaxReservationsPerIP:   8,
				MaxReservationsPerASN:  16,
			},
			AdvertiseInterval:  time.Minute * 10,
			DiscoveryNamespace: "KNIRVORACLE-relay",
		},
		WalletPort:             3001, // Will be set based on role
		AltGUIPort:             3000, // Default Next.js port
		BlockchainDatabasePath: "",   // Will be set based on role
		ReflectionDatabasePath: "",
		SearchableDatabasePath: "",
		MinersAddress:          "",
		MasterAddress:          "",
		NoWalletServer:         false, // Will be set based on role
		ClientOnly:             false, // Will be set based on role
		UseGUI:                 false, // Will be set based on role
		ReflectionURLs:         []string{},
		ChainID:                "KNIRVORACLE",
		InstallComplete:        false,
		IsRoot:                 false, // Will be set based on role
		IsBootnode:             false, // Will be set based on role
		IsPeer:                 false, // Will be set based on role
		IsNetworkMode:          false, // Default to false
		Chromem: ChromemConfig{
			CerebrasConfig: &CerebrasConfig{
				APIKey:  "", // Default empty, loaded from env/config
				BaseURL: "", // Default empty, loaded from env/config
			},
		},
		PaymentProcessor: PaymentProcessorConfig{
			Enabled:       false, // Will be set based on role
			NodeRPC:       "",    // Will be set based on role
			WebhookPort:   0,     // Will be set based on role
			TokenSymbol:   "",    // Will be set based on role
			TokenDecimals: 0,     // Will be set based on role
			USDPerToken:   0,     // Will be set based on role
			ETHPerToken:   0,     // Will be set based on role
		},
		Bootnode: BootnodeConfig{
			Enabled: false, // Will be set based on role
		},
		NodeJSServices: NodeJSServicesConfig{
			Enabled: false, // Will be set based on role
			TunnelRegistry: struct {
				Enabled          bool   `json:"enabled"`
				ScriptPath       string `json:"script_path"`
				HTTPPort         uint   `json:"http_port"`
				ControlPort      uint   `json:"control_port"`
				PublicRelayPort  uint   `json:"public_relay_port"`
				STUNPort         uint   `json:"stun_port"`
				ServerPublicHost string `json:"server_public_host"`
			}{
				Enabled:          true, // Changed from false to true to enable data engine by default for all roles
				ScriptPath:       "agent-tunnel-registry/server.js",
				HTTPPort:         3003,
				ControlPort:      4001,
				PublicRelayPort:  4000,
				STUNPort:         3478,
				ServerPublicHost: "localhost", // Will be overridden in production
			},
			PaymentGateway: struct {
				Enabled    bool   `json:"enabled"`
				ScriptPath string `json:"script_path"`
				HTTPPort   uint   `json:"http_port"`
				APIKey     string `json:"api_key_secret"`
			}{
				Enabled:    false,
				ScriptPath: "agent-payment-gateway/server.js",
				HTTPPort:   3004,
				APIKey:     "", // Will be set from environment or config
			},
			BootnodeRegistry: struct {
				Enabled    bool   `json:"enabled"`
				ScriptPath string `json:"script_path"`
				HTTPPort   uint   `json:"http_port"`
			}{
				Enabled:    true, // Enable by default for root nodes
				ScriptPath: "agent-bootnode-registry/registry-service.js",
				HTTPPort:   3003,
			},
			NotarySystem: struct {
				Enabled    bool   `json:"enabled"`
				ScriptPath string `json:"script_path"`
				HTTPPort   uint   `json:"http_port"`
			}{
				Enabled:    true, // Enable by default for root nodes
				ScriptPath: "agent-notary-system/server.js",
				HTTPPort:   3007,
			},
			NetworkMonitor: struct {
				Enabled    bool   `json:"enabled"`
				ScriptPath string `json:"script_path"`
				HTTPPort   uint   `json:"http_port"`
			}{
				Enabled:    false, // NetworkMonitor is managed by NetworkMonitorManager, not NodeJS manager
				ScriptPath: "",
				HTTPPort:   0,
			},
			NANDAANS: struct {
				Enabled  bool `json:"enabled" mapstructure:"enabled"`
				HTTPPort uint `json:"http_port" mapstructure:"http_port"`
			}{
				Enabled:  true, // Enable by default for root nodes
				HTTPPort: 0,    // Use main HTTP server port (served as static files)
			},
		},
		TunnelClient: TunnelClientConfig{
			Enabled:        false,           // Disabled by default
			ServerAddress:  "ROOTCHAIN_URL", // Default tunnel server
			ControlPort:    4001,            // Default control port
			PingInterval:   30,              // 30 seconds between pings
			ReconnectDelay: 5,               // 5 seconds between reconnect attempts
		},
		ReverseProxy: ReverseProxyConfig{
			Enabled:         false,   // Default to false
			ListenAddr:      ":8080", // Default listen address if enabled
			EmbedDataEngine: false,   // Default to false
		},
		PublicIPInfo:        nil, // Default to nil
		TerminalIntegration: DefaultTerminalIntegration(),

		// PoAu-D configuration defaults
		PoAuD: PoAuDConfig{
			Enabled:                 false,            // Disabled by default for backward compatibility
			DelegationInterval:      10 * time.Second, // 10 seconds between delegation scans
			MaxSubpoolStaleTime:     5 * time.Minute,  // 5 minutes before reclaiming stale transactions
			MaxPapSubpoolQueue:      100,              // Maximum 100 transactions in PAP subpool
			StatusAdvertiseInterval: 30 * time.Minute, // 30 minutes between status advertisements
		},

		// Network Monitor configuration defaults
		NetworkMonitor: NetworkMonitorConfig{
			Enabled:   false, // Disabled by default, enabled in testnet mode
			WebMode:   true,  // Default to web mode (headless)
			Port:      8091,  // Default port for web interface
			AutoStart: false, // Don't auto-start by default
		},
	}
}

// CreateRootConfigFromMatrixAndConstants creates a configuration for a Root node using the settings matrix and constants
// This merged version combines the best aspects of both previous implementations
func CreateRootConfigFromMatrixAndConstants() *Config {
	cfg := DefaultConfig() // Start with common defaults

	// Apply role-specific defaults from settings matrix
	ApplyRoleDefaults(cfg, Root)

	// Set root-specific values
	cfg.IsRoot = true
	cfg.InstallComplete = true // Root doesn't need typical installation steps
	cfg.UseGUI = true
	cfg.ClientOnly = false
	cfg.NoWalletServer = false

	// Set default ports (from older version)
	cfg.Port = 8080
	cfg.P2PPort = 3030
	cfg.WalletPort = 8081
	cfg.AltGUIPort = 3000

	// Use constants from main package for critical values
	if rootConstants.BlockchainAddress != "" {
		cfg.MinersAddress = rootConstants.BlockchainAddress
		cfg.MasterAddress = rootConstants.BlockchainAddress
		cfg.ChainID = rootConstants.BlockchainName
		log.Printf("Using BlockchainAddress from constants: %s", rootConstants.BlockchainAddress)
	} else {
		log.Printf("WARNING: Root constants not set! Using empty values for critical fields.")
		cfg.MinersAddress = ""
		cfg.MasterAddress = ""
		cfg.ChainID = fmt.Sprintf("KNIRVORACLE-Faucet%d", cfg.Port)
	}

	// Database paths are now handled by resolveDynamicPathsViper
	// Clear them here to ensure resolveDynamicPathsViper sets them correctly for Root.
	cfg.BlockchainDatabasePath = ""
	cfg.SearchableDatabasePath = ""
	cfg.ReflectionDatabasePath = ""

	// Payment processor settings from constants
	if rootConstants.CurrencyName != "" {
		cfg.PaymentProcessor.TokenSymbol = rootConstants.CurrencyName
	}
	if rootConstants.Decimal != 0 {
		cfg.PaymentProcessor.TokenDecimals = rootConstants.Decimal
	}

	// Hostname for tunnel server
	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("Warning: Could not determine hostname for tunnel server: %v. Using 'localhost'.", err)
		hostname = "localhost"
	}
	cfg.NodeJSServices.TunnelRegistry.ServerPublicHost = hostname

	log.Println("Created merged Root configuration from constants and settings matrix")
	return cfg
}

// Legacy function for backward compatibility
func CreateRootConfigFromConstants(v *viper.Viper) *Config {
	cfg := DefaultConfig() // Start with common defaults

	// Set root-specific values
	cfg.IsRoot = true
	cfg.InstallComplete = true // Root doesn't need the typical installation steps like wallet gen

	// Use constants from main package
	if rootConstants.BlockchainAddress != "" {
		cfg.MinersAddress = rootConstants.BlockchainAddress
		cfg.MasterAddress = rootConstants.BlockchainAddress
		log.Printf("Using BlockchainAddress from constants: %s", rootConstants.BlockchainAddress)
	} else {
		log.Printf("WARNING: Root constants not set! Using empty values for critical fields.")
		cfg.MinersAddress = ""
		cfg.MasterAddress = ""
	}

	// Network ports (dynamic for Root)
	cfg.Port = utils.FindAvailablePort(9999)     // Dynamic default for Root HTTP
	cfg.P2PPort = utils.FindAvailablePort(19999) // Dynamic default for Root P2P
	cfg.WalletPort = 1000                        // Root typically doesn't run a wallet server for others
	cfg.NoWalletServer = true

	// Network ports (dynamic for Root)
	cfg.Port = utils.FindAvailablePort(9999)     // Dynamic default for Root HTTP
	cfg.P2PPort = utils.FindAvailablePort(19999) // Dynamic default for Root P2P
	cfg.WalletPort = 1001                        // Root typically doesn't run a wallet server for others
	cfg.NoWalletServer = true

	// Database path for Root
	dbPath, err := GetBlockchainDatabasePath(v, "agent_root.db", Root)
	if err != nil {
		log.Printf("Warning: Could not determine default DB path for Root: %v. Using local path 'agent_root.db'.", err)
		dbPath = "agent_root.db" // Fallback to current directory
	}
	cfg.BlockchainDatabasePath = dbPath

	// Set reflection database path for network mode
	reflectionDbPath, err := GetReflectionDatabasePath(Root)
	if err != nil {
		log.Printf("Warning: Could not determine default reflection DB path: %v. Using local path 'agent_reflection.db'.", err)
		reflectionDbPath = "agent_reflection.db" // Fallback to current directory
	}
	cfg.ReflectionDatabasePath = reflectionDbPath
	log.Printf("Set reflection database path for Root: %s", cfg.ReflectionDatabasePath)

	// Set ChainID
	cfg.ChainID = fmt.Sprintf("agent-root-%d", cfg.Port)

	// Enable payment processor for Root
	cfg.PaymentProcessor.Enabled = true

	// Set other values from constants
	if rootConstants.CurrencyName != "" {
		cfg.PaymentProcessor.TokenSymbol = rootConstants.CurrencyName
	}

	if rootConstants.Decimal != 0 {
		cfg.PaymentProcessor.TokenDecimals = rootConstants.Decimal
	}

	// Enable Node.js services for Root nodes
	cfg.NodeJSServices.Enabled = true
	cfg.NodeJSServices.TunnelRegistry.Enabled = true

	// Get the hostname for the tunnel server
	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("Warning: Could not determine hostname for tunnel server: %v. Using 'localhost'.", err)
		hostname = "localhost"
	}
	cfg.NodeJSServices.TunnelRegistry.ServerPublicHost = hostname

	// Enable payment gateway if payment processor is enabled
	if cfg.PaymentProcessor.Enabled {
		cfg.NodeJSServices.PaymentGateway.Enabled = true
	}

	cfg.ClientOnly = false
	cfg.IsBootnode = false
	cfg.UseGUI = false // Default GUI to false for root, can be overridden by flag

	log.Println("Created Root configuration directly from constants (no file operations)")
	return cfg
}

// GetConfigDir returns the base configuration directory (e.g., ~/.config/KNIRVORACLE)
func GetConfigDir() (string, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}
	appConfigDir := filepath.Join(userConfigDir, AppName) // Use the AppName constant
	if err := os.MkdirAll(appConfigDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create app config directory %s: %w", appConfigDir, err)
	}
	return appConfigDir, nil
}

// GetConfigPath returns the full path to the config.json file based on role.
// Each role has its own config file in its role-specific data directory:
// - Root: ~/.config/KNIRVORACLE/root_data/config.json
// - Bootnode: ~/.config/KNIRVORACLE/bootnode_data/config.json
// - Peer: ~/.config/KNIRVORACLE/dev_data/config.json
// - Client: ~/.config/KNIRVORACLE/client_data/config.json
func GetConfigPath(role ...Role) (string, error) {
	currentRole := RoleClient // Default role
	if len(role) > 0 {
		currentRole = role[0]
	}

	// Get the role-specific data directory
	dataDir, err := GetDataDir(currentRole)
	if err != nil {
		return "", err
	}

	// For Root, we'll still return a path, but SaveConfig will handle
	// whether to actually write the file or not based on the role.
	configPath := filepath.Join(dataDir, "config.json")
	log.Printf("Config path for role %s: %s", currentRole, configPath)
	return configPath, nil
}

// GetDataDir returns the data directory path based on role
// For each role, it creates a specific subdirectory:
// - Root: ~/.config/KNIRVORACLE/root_data
// - Bootnode: ~/.config/KNIRVORACLE/bootnode_data
// - Peer: ~/.config/KNIRVORACLE/dev_data
// - Client: ~/.config/KNIRVORACLE/client_data
func GetDataDir(role ...Role) (string, error) {
	currentRole := RoleClient // Default
	if len(role) > 0 {
		currentRole = role[0]
	}

	baseDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	// Determine role-specific data directory name
	var dataDirName string
	switch currentRole {
	case Root:
		dataDirName = "root_data"
	case RoleBootnode:
		dataDirName = "bootnode_data"
	case RolePeer:
		dataDirName = "dev_data"
	case RoleClient:
		dataDirName = "client_data"
	default:
		dataDirName = "data" // Fallback
	}

	dataDirPath := filepath.Join(baseDir, dataDirName)
	if err := os.MkdirAll(dataDirPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create %s directory %s: %w", currentRole, dataDirPath, err)
	}

	log.Printf("Using role-specific data directory: %s for role %s", dataDirPath, currentRole)
	return dataDirPath, nil
}

// GetAppDataDir returns the root directory for application data based on OS and role.
// For Root role, it returns "." (current directory).
// For other roles, it returns the OS-specific app data directory.
func GetAppDataDir(role ...Role) (string, error) {
	currentRole := RoleClient // Default
	if len(role) > 0 {
		currentRole = role[0]
	}

	// For Root role, use the current directory
	if currentRole == Root {
		return ".", nil
	}

	var dir string
	var err error

	switch runtime.GOOS {
	case "windows":
		dir = os.Getenv("APPDATA")
		if dir == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		dir = filepath.Join(dir, AppName)
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		// Standard macOS location: ~/Library/Application Support/<AppName>
		// Using ~/.config/<AppName> for consistency with Linux for now, adjust if needed
		dir = filepath.Join(home, ".config", AppName)
	case "linux":
		// Use XDG_CONFIG_HOME if set, otherwise default to ~/.config
		xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfigHome != "" {
			dir = filepath.Join(xdgConfigHome, AppName)
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("failed to get user home directory: %w", err)
			}
			dir = filepath.Join(home, ".config", AppName)
		}
	default:
		// Fallback for other systems (e.g., BSD) - use ~/.<AppName>
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		dir = filepath.Join(home, "."+AppName)
	}

	// Validate path isn't empty (should never happen with above logic)
	if dir == "" {
		return "", fmt.Errorf("internal error: constructed empty app data directory path")
	}

	// Create the directory if it doesn't exist
	err = os.MkdirAll(dir, 0750) // Use 0750 for permissions (owner rwx, group rx)
	if err != nil {
		return "", fmt.Errorf("failed to create app data directory '%s': %w", dir, err)
	}

	// Final validation that directory exists
	if _, err := os.Stat(dir); err != nil {
		return "", fmt.Errorf("app data directory verification failed for '%s': %w", dir, err)
	}

	return dir, nil
}

// GetBlockchainDatabasePath determines the path to the blockchain database file.
// It prioritizes explicitly set paths in the config, then falls back to role-based defaults.
func GetBlockchainDatabasePath(v *viper.Viper, defaultFilename string, role ...Role) (string, error) {
	currentRole := RoleClient // Default
	if len(role) > 0 {
		currentRole = role[0]
	}

	// 1. Check if the path is explicitly set in the loaded Viper configuration
	// First check for the path in the main config structure
	if v != nil && v.IsSet("shared_database_path") {
		explicitPath := v.GetString("shared_database_path")
		if explicitPath != "" {
			log.Printf("Viper: Using explicit BlockchainDatabasePath from config.shared_database_path: %s", explicitPath)
			return filepath.Clean(explicitPath), nil
		}
	}

	// Then check for the path in the paths section
	explicitPathKey := "paths.blockchain_database_path"
	if v != nil && v.IsSet(explicitPathKey) {
		explicitPath := v.GetString(explicitPathKey)
		if explicitPath != "" {
			log.Printf("Viper: Using explicit BlockchainDatabasePath from config.%s: %s", explicitPathKey, explicitPath)
			return filepath.Clean(explicitPath), nil
		}
	}

	// Use test-specific temp directory if in test mode
	if os.Getenv("agent_TEST_MODE") == "true" {
		testDir := os.Getenv("agent_TEST_DIR")
		if testDir == "" {
			testDir = os.TempDir()
		}
		dbFilename := fmt.Sprintf("test_%s_%s.db", currentRole, uuid.New().String())
		finalPath := filepath.Join(testDir, dbFilename)
		log.Printf("TEST MODE: Using isolated test database path: %s", finalPath)
		return finalPath, nil
	}

	// 2. If not explicitly set, then derive based on role (fallback)
	log.Printf("Viper: BlockchainDatabasePath not explicitly set in config for role %s, deriving default.", currentRole)
	dataDir, err := GetDataDir(role...)
	if err != nil {
		return "", fmt.Errorf("failed to get data dir for role %s: %w", currentRole, err)
	}

	dbFilename := defaultFilename
	switch currentRole {
	case Root:
		if dbFilename == "" || dbFilename == "agent.db" {
			dbFilename = "agent_root.db"
		}
	case RoleBootnode:
		if dbFilename == "" {
			dbFilename = "bootnode.db"
		}
	case RoleClient:
		if dbFilename == "" {
			dbFilename = "client_agent.db"
		}
	default:
		if dbFilename == "" {
			dbFilename = "agent.db"
		}
	}

	resolvedPath := filepath.Join(dataDir, dbFilename)
	log.Printf("Viper: Resolved BlockchainDatabasePath for role %s to: %s", currentRole, resolvedPath)
	return resolvedPath, nil
}

// GetReflectionDatabasePath constructs the database path for a reflection node.
func GetReflectionDatabasePath(role ...Role) (string, error) {
	dataDir, err := GetDataDir(role...)
	if err != nil {
		return "", err
	}

	// Get the current role
	currentRole := Root
	if len(role) > 0 {
		currentRole = role[0]
	}

	// Create a dedicated reflectionDatabase directory to avoid conflicts with root database
	reflectionDir := filepath.Join(dataDir, "reflectionDatabase")

	// Ensure the directory exists
	if err := os.MkdirAll(reflectionDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create reflection database directory: %w", err)
	}

	// Store the database in the dedicated directory with a role-specific name
	dbFilename := "agent_reflection.db"
	switch currentRole {
	case Root:
		dbFilename = "agent_reflection_root.db"
	case RoleBootnode:
		dbFilename = "agent_reflection_bootnode.db"
	case RolePeer:
		dbFilename = "agent_reflection_dev.db"
	case RoleClient:
		dbFilename = "agent_reflection_client.db"
	}

	finalPath := filepath.Join(reflectionDir, dbFilename)
	log.Printf("DEBUG: GetReflectionDatabasePath - Constructed finalPath: '%s' for role %s", finalPath, currentRole)
	return finalPath, nil
}

// EnsurePaths ensures all necessary paths are set and directories exist
// This is particularly important for the reflection database path
func (cfg *Config) EnsurePaths() error {
	// Determine DataDir based on role or use BlockchainDatabasePath
	var dataDir string
	if cfg.BlockchainDatabasePath != "" {
		dataDir = filepath.Dir(cfg.BlockchainDatabasePath)
	} else {
		// Use default data directory
		var err error
		dataDir, err = GetDataDir(RoleReflection) // Default to node role
		if err != nil {
			return fmt.Errorf("failed to get data directory: %w", err)
		}
	}

	// Ensure the data directory exists
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// If ReflectionDatabasePath is not set but we need it (for network mode)
	if cfg.ReflectionDatabasePath == "" {
		// Create reflection database directory
		reflectionDbDir := filepath.Join(dataDir, "reflection_db")
		if err := os.MkdirAll(reflectionDbDir, 0700); err != nil {
			return fmt.Errorf("failed to create reflection database directory: %w", err)
		}

		// Set the reflection database path
		cfg.ReflectionDatabasePath = filepath.Join(reflectionDbDir, "reflection.db")
	} else {
		// Ensure the directory for the existing reflection database path exists
		reflectionDbDir := filepath.Dir(cfg.ReflectionDatabasePath)
		if err := os.MkdirAll(reflectionDbDir, 0700); err != nil {
			return fmt.Errorf("failed to create reflection database directory: %w", err)
		}
	}

	return nil
}

// GetSearchableDatabasePath determines the path for the ChromemDB/searchable database.
// It prioritizes explicitly set paths in the config, then falls back to role-based defaults.
func GetSearchableDatabasePath(v *viper.Viper, chainID string, role ...Role) (string, error) {
	currentRole := RoleClient // Default
	if len(role) > 0 {
		currentRole = role[0]
	}

	// 1. Check if the path is explicitly set in the loaded Viper configuration
	// First check for the path in the main config structure
	if v != nil && v.IsSet("local_database_path") {
		explicitPath := v.GetString("local_database_path")
		if explicitPath != "" {
			log.Printf("Viper: Using explicit SearchableDatabasePath from config.local_database_path: %s", explicitPath)
			return filepath.Clean(explicitPath), nil
		}
	}

	// Also check for the path in the main config structure under searchable_database_path
	if v != nil && v.IsSet("searchable_database_path") {
		explicitPath := v.GetString("searchable_database_path")
		if explicitPath != "" {
			log.Printf("Viper: Using explicit SearchableDatabasePath from config.searchable_database_path: %s", explicitPath)
			return filepath.Clean(explicitPath), nil
		}
	}

	// Then check for the path in the paths section
	explicitPathKey := "paths.searchable_database_path"
	if v != nil && v.IsSet(explicitPathKey) {
		explicitPath := v.GetString(explicitPathKey)
		if explicitPath != "" {
			log.Printf("Viper: Using explicit SearchableDatabasePath from config.%s: %s", explicitPathKey, explicitPath)
			return filepath.Clean(explicitPath), nil
		}
	}

	// Use test-specific temp directory if in test mode
	if os.Getenv("agent_TEST_MODE") == "true" {
		testDir := os.Getenv("agent_TEST_DIR")
		if testDir == "" {
			testDir = os.TempDir()
		}
		dbFilename := fmt.Sprintf("test_%s_%s_%s.db", currentRole, chainID, uuid.New().String())
		finalPath := filepath.Join(testDir, dbFilename)
		log.Printf("TEST MODE: Using isolated test database path: %s", finalPath)
		return finalPath, nil
	}

	// 2. If not explicitly set, then derive based on role (fallback)
	log.Printf("Viper: SearchableDatabasePath not explicitly set in config for role %s, deriving default.", currentRole)
	dataDir, err := GetDataDir(role...)
	if err != nil {
		return "", err
	}
	if chainID == "" {
		return "", fmt.Errorf("chainID cannot be empty for dev database path")
	}

	var dbFilename string
	switch currentRole {
	case RoleBootnode:
		dbFilename = fmt.Sprintf("bootnode_%s.db", chainID)
	default:
		dbFilename = fmt.Sprintf("dev_%s.db", chainID)
	}

	resolvedPath := filepath.Join(dataDir, dbFilename)
	log.Printf("Viper: Resolved SearchableDatabasePath for role %s to: %s", currentRole, resolvedPath)
	return resolvedPath, nil
}

// GetWalletPath returns the path to the wallet file based on role.
// Each role has its wallet file in its role-specific data directory:
// - Root: No wallet file (returns empty string)
// - Bootnode: ~/.config/KNIRVORACLE/bootnode_data/wallet.dat
// - Peer: ~/.config/KNIRVORACLE/dev_data/wallet.dat
// - Client: ~/.config/KNIRVORACLE/client_data/wallet.dat
func GetWalletPath(role ...Role) (string, error) {
	currentRole := RoleClient // Default if no role specified
	if len(role) > 0 {
		currentRole = role[0]
	}

	if currentRole == Root {
		return "", nil // Root does not use a wallet.dat file
	}

	// Get the role-specific data directory
	dataDir, err := GetDataDir(currentRole)
	if err != nil {
		return "", fmt.Errorf("failed to get data directory: %w", err)
	}

	walletPath := filepath.Join(dataDir, "wallet.dat")
	log.Printf("Wallet path for role %s: %s", currentRole, walletPath)
	return walletPath, nil
}

// GetPeerWalletPath returns the path to the dev-specific wallet file.
// This might be different from the root node's wallet.
func GetPeerWalletPath(role ...Role) (string, error) {
	// For now, let's assume devs use the same wallet.dat structure as other non-root roles.
	// If devs need distinctly named wallet files (e.g., dev_wallet.dat), adjust here.
	return GetWalletPath(role...)
}

// GetMasterWalletPath returns the path to the master wallet file based on role.
// Each role has its master wallet file in its role-specific data directory:
// - Root: No master wallet file (returns empty string)
// - Bootnode: ~/.config/KNIRVORACLE/bootnode_data/master_wallet.dat
// - Peer: ~/.config/KNIRVORACLE/dev_data/master_wallet.dat (typically not used)
// - Client: ~/.config/KNIRVORACLE/client_data/master_wallet.dat (typically not used)
func GetMasterWalletPath(role ...Role) (string, error) {
	currentRole := RoleClient // Default
	if len(role) > 0 {
		currentRole = role[0]
	}

	if currentRole == Root {
		return "", nil // Root does not use a master_wallet.dat file
	}

	// Get the role-specific data directory
	dataDir, err := GetDataDir(currentRole)
	if err != nil {
		return "", fmt.Errorf("failed to get data directory: %w", err)
	}

	masterWalletPath := filepath.Join(dataDir, "master_wallet.dat")
	log.Printf("Master wallet path for role %s: %s", currentRole, masterWalletPath)
	return masterWalletPath, nil
}

// LoadConfig loads the configuration from the given path or default locations.
func LoadConfig(configPath string, role ...Role) (*Config, string, error) {
	currentRole := RoleClient // Default role
	if len(role) > 0 {
		currentRole = role[0]
	}

	pathsToTry := []string{}
	if configPath != "" {
		pathsToTry = append(pathsToTry, configPath)
	} else {
		defaultPath, err := GetConfigPath(currentRole)
		if err == nil && defaultPath != "" { // Only add if GetConfigPath gives a valid path
			pathsToTry = append(pathsToTry, defaultPath)
		}
	}

	for _, p := range pathsToTry {
		log.Printf("Checking config path: %s for role %s", p, currentRole)
		data, errFile := ioutil.ReadFile(p)
		if errFile == nil {
			var cfg Config
			if err := json.Unmarshal(data, &cfg); err != nil {
				log.Printf("Error parsing config file %s: %v", p, err)
				continue
			}
			// Initialize with defaults first, then overlay loaded config
			// This helps if config file is partial.
			defaultCfg := DefaultConfig()
			// Merge loaded config over defaults
			defaultCfg = MergeConfigs(defaultCfg, &cfg)
			if currentRole == Root {
				// For root, if a file IS found, it might only contain overrides like UseGUI.
				// So, start with settings from matrix and constants.
				defaultCfg = CreateRootConfigFromMatrixAndConstants()
			}

			// Unmarshal read data into the default structure
			if errJson := json.Unmarshal(data, defaultCfg); errJson != nil {
				return nil, p, fmt.Errorf("failed to parse config file %s: %w", p, errJson)
			}
			log.Printf("Loading config from: %s", p)
			log.Printf("Loaded config values - HTTPPort: %d, P2PPort: %d", defaultCfg.Port, defaultCfg.P2PPort)
			return defaultCfg, p, nil
		}
		if !os.IsNotExist(errFile) {
			return nil, p, fmt.Errorf("failed to read config file %s: %w", p, errFile)
		}
	}

	if currentRole == Root {
		log.Printf("No config file found for Root role. Returning default config and will save it later.")
		rootCfg := CreateRootConfigFromMatrixAndConstants()

		// Get the path where we should save the config
		rootConfigPath, err := GetConfigPath(Root)
		if err == nil && rootConfigPath != "" {
			log.Printf("Will save Root config to: %s after initialization", rootConfigPath)
			return rootCfg, rootConfigPath, nil
		}

		return rootCfg, "", nil
	}

	// For non-Root roles, if no config file is found, return a default config.
	// This default config will have InstallComplete = false, triggering the installer in main.go.
	log.Printf("No config file found for role %s. Returning default config to trigger installer.", currentRole)
	defaultCfg := DefaultConfig()
	return defaultCfg, "", nil // Return default config, empty path, and no error
}

// SaveConfig saves the configuration to the specified JSON file path.
// The caller is responsible for providing the full, correct path.
// This function will NOT save config for the Root role for security reasons.
func SaveConfig(configPath string, cfg *Config) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("SaveConfig: cannot save nil config")
	}
	if configPath == "" {
		return "", fmt.Errorf("SaveConfig: configPath cannot be empty")
	}

	// Skip saving for Root role for security reasons
	if cfg.IsRoot {
		log.Printf("SECURITY: Skipping config file save for Root role to prevent sensitive data exposure")
		return configPath, nil
	}

	// Determine the role from the config
	var role Role
	if cfg.IsRoot {
		role = Root
	} else if cfg.IsBootnode {
		role = RoleBootnode
	} else if cfg.ClientOnly {
		role = RoleClient
	} else {
		role = RolePeer
	}

	// Use the minimal config approach
	return SaveMinimalConfig(configPath, cfg, role)
}

// SaveConfigToUserDir saves the configuration to the user config directory only
// This function should be used instead of saving to the current directory
// This function will NOT save config for the Root role for security reasons.
func SaveConfigToUserDir(cfg *Config, role Role) {
	// Use the minimal config approach directly
	SaveMinimalConfigToUserDir(cfg, role)
}

// BLOCKCHAIN_ADDRESS_CONSTANT_FROM_MAIN_OR_CONFIG is a placeholder.
// You'll need to access the actual BLOCKCHAIN_ADDRESS from your constants.go.
const BLOCKCHAIN_ADDRESS_CONSTANT_FROM_MAIN_OR_CONFIG = "KNIRVORACLEb53c1e30b8a578c091dd40612bfd1433991b4e09" // Replace with actual access

// DefaultTerminalIntegration returns the default terminal integration configuration
func DefaultTerminalIntegration() *TerminalIntegration {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	return &TerminalIntegration{
		Enabled:           true,
		ZshCompletionsDir: filepath.Join(homeDir, ".KNIRVORACLE", "completions"),
		Theme:             "dark",
	}
}

// GetString returns a string value from the config
func (c *Config) GetString(key string) (string, bool) {
	// Use viper to get the value if available
	if viper.IsSet(key) {
		return viper.GetString(key), true
	}

	// Otherwise, try to get it from the config struct
	// This is a simplified implementation - you might need to expand it
	// based on your actual config structure
	switch key {
	case "node.name":
		return c.NodeName, c.NodeName != ""
	default:
		return "", false
	}
}
