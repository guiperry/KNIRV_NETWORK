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

	// Node.js services
	NodeJSServices NodeJSServicesSettings

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

// NodeJSServicesSettings defines Node.js services configuration
type NodeJSServicesSettings struct {
	Enabled        bool
	TunnelRegistry TunnelRegistrySettings
	PaymentGateway PaymentGatewaySettings
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

		// Node.js services
		NodeJSServices: NodeJSServicesSettings{
			Enabled: true,
			TunnelRegistry: TunnelRegistrySettings{
				Enabled:          true,
				ScriptPath:       "agent-tunnel-registry/server.js",
				HTTPPort:         3003,
				ControlPort:      4001,
				PublicRelayPort:  4000,
				STUNPort:         3478,
				ServerPublicHost: "localhost", // Will be set dynamically
			},
			PaymentGateway: PaymentGatewaySettings{
				Enabled:    true,
				ScriptPath: "agent-payment-gateway/server.js",
				HTTPPort:   3004,
			},
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
		DefaultWalletPort: 7000,
		NoWalletServer:    false,

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
		NodeJSServices: NodeJSServicesSettings{
			Enabled: false,
			TunnelRegistry: TunnelRegistrySettings{
				Enabled:          false,
				ScriptPath:       "agent-tunnel-registry/server.js",
				HTTPPort:         3003,
				ControlPort:      4001,
				PublicRelayPort:  4000,
				STUNPort:         3478,
				ServerPublicHost: "localhost",
			},
			PaymentGateway: PaymentGatewaySettings{
				Enabled:    false,
				ScriptPath: "agent-payment-gateway/server.js",
				HTTPPort:   3004,
			},
		},

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
		DefaultWalletPort: 7001,
		NoWalletServer:    false,

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
		NodeJSServices: NodeJSServicesSettings{
			Enabled: false,
			TunnelRegistry: TunnelRegistrySettings{
				Enabled:          false,
				ScriptPath:       "agent-tunnel-registry/server.js",
				HTTPPort:         3003,
				ControlPort:      4001,
				PublicRelayPort:  4000,
				STUNPort:         3478,
				ServerPublicHost: "localhost",
			},
			PaymentGateway: PaymentGatewaySettings{
				Enabled:    false,
				ScriptPath: "agent-payment-gateway/server.js",
				HTTPPort:   3004,
			},
		},

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
		DefaultWalletPort: 7002,
		NoWalletServer:    false,

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
		NodeJSServices: NodeJSServicesSettings{
			Enabled: false,
			TunnelRegistry: TunnelRegistrySettings{
				Enabled:          false,
				ScriptPath:       "agent-tunnel-registry/server.js",
				HTTPPort:         3003,
				ControlPort:      4001,
				PublicRelayPort:  4000,
				STUNPort:         3478,
				ServerPublicHost: "localhost",
			},
			PaymentGateway: PaymentGatewaySettings{
				Enabled:    false,
				ScriptPath: "agent-payment-gateway/server.js",
				HTTPPort:   3004,
			},
		},

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

	// Apply Node.js services settings
	cfg.NodeJSServices.Enabled = settings.NodeJSServices.Enabled

	// Apply tunnel registry settings
	cfg.NodeJSServices.TunnelRegistry.Enabled = settings.NodeJSServices.TunnelRegistry.Enabled
	if cfg.NodeJSServices.TunnelRegistry.ScriptPath == "" {
		cfg.NodeJSServices.TunnelRegistry.ScriptPath = settings.NodeJSServices.TunnelRegistry.ScriptPath
	}
	if cfg.NodeJSServices.TunnelRegistry.HTTPPort == 0 {
		cfg.NodeJSServices.TunnelRegistry.HTTPPort = uint(settings.NodeJSServices.TunnelRegistry.HTTPPort)
	}
	if cfg.NodeJSServices.TunnelRegistry.ControlPort == 0 {
		cfg.NodeJSServices.TunnelRegistry.ControlPort = uint(settings.NodeJSServices.TunnelRegistry.ControlPort)
	}
	if cfg.NodeJSServices.TunnelRegistry.PublicRelayPort == 0 {
		cfg.NodeJSServices.TunnelRegistry.PublicRelayPort = uint(settings.NodeJSServices.TunnelRegistry.PublicRelayPort)
	}
	if cfg.NodeJSServices.TunnelRegistry.STUNPort == 0 {
		cfg.NodeJSServices.TunnelRegistry.STUNPort = uint(settings.NodeJSServices.TunnelRegistry.STUNPort)
	}
	if cfg.NodeJSServices.TunnelRegistry.ServerPublicHost == "" {
		cfg.NodeJSServices.TunnelRegistry.ServerPublicHost = settings.NodeJSServices.TunnelRegistry.ServerPublicHost
	}

	// Apply payment gateway settings
	cfg.NodeJSServices.PaymentGateway.Enabled = settings.NodeJSServices.PaymentGateway.Enabled
	if cfg.NodeJSServices.PaymentGateway.ScriptPath == "" {
		cfg.NodeJSServices.PaymentGateway.ScriptPath = settings.NodeJSServices.PaymentGateway.ScriptPath
	}
	if cfg.NodeJSServices.PaymentGateway.HTTPPort == 0 {
		cfg.NodeJSServices.PaymentGateway.HTTPPort = uint(settings.NodeJSServices.PaymentGateway.HTTPPort)
	}

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
