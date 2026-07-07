package config

import (
	"log"
)

// RoleSettings defines which features are enabled for each role
type RoleSettings struct {
	// Core role settings
	IsRoot     bool
	IsBootnode bool
	IsPeer     bool
	ClientOnly bool
	UseGUI     bool

	// Network settings
	DefaultPort       int
	DefaultP2PPort    int
	DefaultWalletPort int
	NoWalletServer    bool

	// Payment processor settings
	PaymentProcessor PaymentProcessorSettings

	// Bootnode settings
	BootnodeSettings BootnodeSettings

	// Tunnel client settings
	TunnelClient TunnelClientSettings

	// Reverse proxy settings
	ReverseProxy ReverseProxySettings
}

// PaymentProcessorSettings defines payment processor configuration
type PaymentProcessorSettings struct {
	Enabled            bool
	DefaultNodeRPC     string
	DefaultWebhookPort int
	TokenSymbol        string
	TokenDecimals      int
	USDPerToken        float64
	ETHPerToken        float64
}

// BootnodeSettings defines bootnode configuration
type BootnodeSettings struct {
	Enabled bool
}

// TunnelRegistrySettings defines tunnel registry configuration
type TunnelRegistrySettings struct {
	Enabled          bool
	ScriptPath       string
	HTTPPort         int
	ControlPort      int
	PublicRelayPort  int
	STUNPort         int
	ServerPublicHost string
}

// PaymentGatewaySettings defines payment gateway configuration
type PaymentGatewaySettings struct {
	Enabled    bool
	ScriptPath string
	HTTPPort   int
}

// TunnelClientSettings defines tunnel client configuration
type TunnelClientSettings struct {
	Enabled        bool
	ServerAddress  string
	ControlPort    int
	PingInterval   int
	ReconnectDelay int
}

// ReverseProxySettings defines reverse proxy configuration
type ReverseProxySettings struct {
	Enabled    bool
	ListenAddr string
}

// RoleSettingsMatrix maps roles to their default settings
var RoleSettingsMatrix = map[Role]RoleSettings{
	Root: {
		// Core role settings
		IsRoot:     true,
		IsBootnode: false,
		IsPeer:     false,
		ClientOnly: false,
		UseGUI:     false,

		// Network settings
		DefaultPort:       9999,
		DefaultP2PPort:    19999,
		DefaultWalletPort: 0,
		NoWalletServer:    true,

		// Payment processor settings
		PaymentProcessor: PaymentProcessorSettings{
			Enabled:            true,
			DefaultNodeRPC:     "http://localhost:5000",
			DefaultWebhookPort: 8088,
			TokenSymbol:        "NRN", // Will be overridden by constants
			TokenDecimals:      2,     // Will be overridden by constants
			USDPerToken:        0.01,
			ETHPerToken:        0.00001,
		},

		// Bootnode settings
		BootnodeSettings: BootnodeSettings{
			Enabled: false,
		},

		// Tunnel client settings
		TunnelClient: TunnelClientSettings{
			Enabled:        false,
			ServerAddress:  "", // Will be set to ROOTCHAIN_URL
			ControlPort:    4001,
			PingInterval:   30,
			ReconnectDelay: 5,
		},

		// Reverse proxy settings
		ReverseProxy: ReverseProxySettings{
			Enabled:    false,
			ListenAddr: ":8080",
		},
	},

	RoleBootnode: {
		// Core role settings
		IsRoot:     false,
		IsBootnode: true,
		IsPeer:     false,
		ClientOnly: false,
		UseGUI:     true,

		// Network settings
		DefaultPort:       5000,
		DefaultP2PPort:    6000,
		DefaultWalletPort: 0,
		NoWalletServer:    true,

		// Payment processor settings
		PaymentProcessor: PaymentProcessorSettings{
			Enabled:            false,
			DefaultNodeRPC:     "http://localhost:5000",
			DefaultWebhookPort: 8088,
			TokenSymbol:        "NRN",
			TokenDecimals:      2,
			USDPerToken:        0.01,
			ETHPerToken:        0.00001,
		},

		// Bootnode settings
		BootnodeSettings: BootnodeSettings{
			Enabled: true,
		},

		// Node.js services
		// Node.js services removed from role settings; management moved to Go components

		// Tunnel client settings
		TunnelClient: TunnelClientSettings{
			Enabled:        false,
			ServerAddress:  "",
			ControlPort:    4001,
			PingInterval:   30,
			ReconnectDelay: 5,
		},

		// Reverse proxy settings
		ReverseProxy: ReverseProxySettings{
			Enabled:    false,
			ListenAddr: ":8080",
		},
	},

	RolePeer: {
		// Core role settings
		IsRoot:     false,
		IsBootnode: false,
		IsPeer:     true,
		ClientOnly: false,
		UseGUI:     true,

		// Network settings
		DefaultPort:       5001,
		DefaultP2PPort:    6001,
		DefaultWalletPort: 0,
		NoWalletServer:    true,

		// Payment processor settings
		PaymentProcessor: PaymentProcessorSettings{
			Enabled:            false,
			DefaultNodeRPC:     "http://localhost:5000",
			DefaultWebhookPort: 8088,
			TokenSymbol:        "NRN",
			TokenDecimals:      2,
			USDPerToken:        0.01,
			ETHPerToken:        0.00001,
		},

		// Bootnode settings
		BootnodeSettings: BootnodeSettings{
			Enabled: false,
		},

		// Node.js services
		// Node.js services removed from role settings; management moved to Go components

		// Tunnel client settings
		TunnelClient: TunnelClientSettings{
			Enabled:        true,
			ServerAddress:  "", // Will be set to ROOTCHAIN_URL
			ControlPort:    4001,
			PingInterval:   30,
			ReconnectDelay: 5,
		},

		// Reverse proxy settings
		ReverseProxy: ReverseProxySettings{
			Enabled:    false,
			ListenAddr: ":8080",
		},
	},

	RoleClient: {
		// Core role settings
		IsRoot:     false,
		IsBootnode: false,
		IsPeer:     false,
		ClientOnly: true,
		UseGUI:     true,

		// Network settings
		DefaultPort:       5002,
		DefaultP2PPort:    6002,
		DefaultWalletPort: 0,
		NoWalletServer:    true,

		// Payment processor settings
		PaymentProcessor: PaymentProcessorSettings{
			Enabled:            false,
			DefaultNodeRPC:     "http://localhost:5000",
			DefaultWebhookPort: 8088,
			TokenSymbol:        "NRN",
			TokenDecimals:      2,
			USDPerToken:        0.01,
			ETHPerToken:        0.00001,
		},

		// Bootnode settings
		BootnodeSettings: BootnodeSettings{
			Enabled: false,
		},

		// Node.js services
		// Node.js services removed from role settings; management moved to Go components

		// Tunnel client settings
		TunnelClient: TunnelClientSettings{
			Enabled:        true,
			ServerAddress:  "", // Will be set to ROOTCHAIN_URL
			ControlPort:    4001,
			PingInterval:   30,
			ReconnectDelay: 5,
		},

		// Reverse proxy settings
		ReverseProxy: ReverseProxySettings{
			Enabled:    false,
			ListenAddr: ":8080",
		},
	},
}

// ApplyRoleDefaults applies the default settings for the specified role to the config
func ApplyRoleDefaults(cfg *Config, role Role) {
	log.Printf("Applying role defaults for role %s. Current Port: %d, P2PPort: %d", role, cfg.Port, cfg.P2PPort)

	settings, exists := RoleSettingsMatrix[role]
	if !exists {
		log.Printf("Warning: No default settings found for role %s. Using generic defaults.", role)
		return
	}

	// Log the default settings from the matrix
	log.Printf("Role %s default settings - DefaultPort: %d, DefaultP2PPort: %d",
		role, settings.DefaultPort, settings.DefaultP2PPort)

	// Apply core role settings
	cfg.IsRoot = settings.IsRoot
	cfg.IsBootnode = settings.IsBootnode
	cfg.IsPeer = settings.IsPeer // Now we have the IsPeer field in Config
	cfg.ClientOnly = settings.ClientOnly
	cfg.UseGUI = settings.UseGUI

	// Apply network settings if not already set
	if cfg.Port == 0 {
		cfg.Port = uint64(settings.DefaultPort)
		log.Printf("Applied default HTTPPort %d for role %s as it was zero", cfg.Port, role)
	}
	if cfg.P2PPort == 0 {
		cfg.P2PPort = uint64(settings.DefaultP2PPort)
		log.Printf("Applied default P2PPort %d for role %s as it was zero", cfg.P2PPort, role)
	}
	if cfg.WalletPort == 0 {
		cfg.WalletPort = uint64(settings.DefaultWalletPort)
		log.Printf("Applied default WalletPort %d for role %s as it was zero", cfg.WalletPort, role)
	}
	// Only set NoWalletServer if it's not already set to true
	if !cfg.NoWalletServer {
		cfg.NoWalletServer = settings.NoWalletServer
	}

	// Apply payment processor settings
	cfg.PaymentProcessor.Enabled = settings.PaymentProcessor.Enabled
	if cfg.PaymentProcessor.NodeRPC == "" {
		cfg.PaymentProcessor.NodeRPC = settings.PaymentProcessor.DefaultNodeRPC
	}
	if cfg.PaymentProcessor.WebhookPort == 0 {
		cfg.PaymentProcessor.WebhookPort = settings.PaymentProcessor.DefaultWebhookPort
	}
	if cfg.PaymentProcessor.TokenSymbol == "" {
		cfg.PaymentProcessor.TokenSymbol = settings.PaymentProcessor.TokenSymbol
	}
	if cfg.PaymentProcessor.TokenDecimals == 0 {
		cfg.PaymentProcessor.TokenDecimals = settings.PaymentProcessor.TokenDecimals
	}
	if cfg.PaymentProcessor.USDPerToken == 0 {
		cfg.PaymentProcessor.USDPerToken = settings.PaymentProcessor.USDPerToken
	}
	if cfg.PaymentProcessor.ETHPerToken == 0 {
		cfg.PaymentProcessor.ETHPerToken = settings.PaymentProcessor.ETHPerToken
	}

	// Apply bootnode settings
	cfg.Bootnode.Enabled = settings.BootnodeSettings.Enabled

	// Apply tunnel client settings
	cfg.TunnelClient.Enabled = settings.TunnelClient.Enabled
	if cfg.TunnelClient.ServerAddress == "" {
		cfg.TunnelClient.ServerAddress = settings.TunnelClient.ServerAddress
	}
	if cfg.TunnelClient.ControlPort == 0 {
		cfg.TunnelClient.ControlPort = uint(settings.TunnelClient.ControlPort)
	}
	if cfg.TunnelClient.PingInterval == 0 {
		cfg.TunnelClient.PingInterval = uint(settings.TunnelClient.PingInterval)
	}
	if cfg.TunnelClient.ReconnectDelay == 0 {
		cfg.TunnelClient.ReconnectDelay = uint(settings.TunnelClient.ReconnectDelay)
	}

	// Apply reverse proxy settings
	cfg.ReverseProxy.Enabled = settings.ReverseProxy.Enabled
	if cfg.ReverseProxy.ListenAddr == "" {
		cfg.ReverseProxy.ListenAddr = settings.ReverseProxy.ListenAddr
	}

	log.Printf("Applied default settings for role %s", role)
}
