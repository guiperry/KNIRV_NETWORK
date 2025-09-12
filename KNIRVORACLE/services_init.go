package main

import (
	"KNIRVORACLE/config"
	"context"
	"log"
	"os"
)

// initPaymentProcessor initializes and starts the payment processor if root mode is enabled
func initPaymentProcessor(cfg *config.Config, db *LevelDB, role config.Role) (*PaymentProcessor, error) {
	// Payment processor is enabled for Root and Bootnode roles
	if (!cfg.IsRoot && role != config.RoleBootnode) || !cfg.PaymentProcessor.Enabled {
		return nil, nil // Payment processor not enabled
	}

	log.Printf("[%s] Initializing payment processor for %s role...", cfg.ChainID, role.String())

	// Get or create master wallet for token disbursement
	masterWallet, err := getMasterWallet(db, role)
	if err != nil {
		return nil, err
	}
	log.Printf("[%s] Using master wallet with address %s for token disbursement", cfg.ChainID, masterWallet.GetAddress())

	// Initialize payment processor
	paymentProcessor, err := NewPaymentProcessor(cfg.PaymentProcessor, masterWallet)
	if err != nil {
		return nil, err
	}

	// Start payment processor
	if err := paymentProcessor.Start(); err != nil {
		return nil, err
	}
	log.Printf("[%s] Payment processor started successfully", cfg.ChainID)

	return paymentProcessor, nil
}

// initNodeJSServices initializes and starts the Node.js services for root and bootnode roles
func initNodeJSServices(cfg *config.Config, discoveryMgr *DiscoveryManager) (*NodeJSManager, error) {
	// Node.js services are enabled for Root and Bootnode roles
	if (!cfg.IsRoot && !cfg.IsBootnode) || !cfg.NodeJSServices.Enabled {
		return nil, nil // Node.js services not enabled for this role
	}

	log.Printf("[%s] Initializing Node.js services for %s role...", cfg.ChainID, config.DetermineRoleFromConfig(cfg).String())

	// Set default script paths if not provided
	if cfg.NodeJSServices.TunnelRegistry.Enabled && cfg.NodeJSServices.TunnelRegistry.ScriptPath == "" {
		cfg.NodeJSServices.TunnelRegistry.ScriptPath = "agent-tunnel-registry/server.js"
		log.Printf("[%s] Using default script path for tunnel registry: %s", cfg.ChainID, cfg.NodeJSServices.TunnelRegistry.ScriptPath)
	}

	if cfg.NodeJSServices.PaymentGateway.Enabled && cfg.NodeJSServices.PaymentGateway.ScriptPath == "" {
		cfg.NodeJSServices.PaymentGateway.ScriptPath = "agent-payment-gateway/server.js"
		log.Printf("[%s] Using default script path for payment gateway: %s", cfg.ChainID, cfg.NodeJSServices.PaymentGateway.ScriptPath)
	}

	if cfg.NodeJSServices.OperatorRegistry.Enabled && cfg.NodeJSServices.OperatorRegistry.ScriptPath == "" {
		cfg.NodeJSServices.OperatorRegistry.ScriptPath = "operator-registry/registry-service.js"
		log.Printf("[%s] Using default script path for bootnode registry: %s", cfg.ChainID, cfg.NodeJSServices.OperatorRegistry.ScriptPath)
	}

	if cfg.NodeJSServices.WebGUI.Enabled && cfg.NodeJSServices.WebGUI.ScriptPath == "" {
		cfg.NodeJSServices.WebGUI.ScriptPath = "webGUI/server.js"
		log.Printf("[%s] Using default script path for Web GUI: %s", cfg.ChainID, cfg.NodeJSServices.WebGUI.ScriptPath)
	}

	if cfg.NodeJSServices.NetworkMonitor.Enabled && cfg.NodeJSServices.NetworkMonitor.ScriptPath == "" {
		cfg.NodeJSServices.NetworkMonitor.ScriptPath = "agent-network-monitor/server.js"
		log.Printf("[%s] Using default script path for network monitor: %s", cfg.ChainID, cfg.NodeJSServices.NetworkMonitor.ScriptPath)
	}

	// Check if script files exist
	if cfg.NodeJSServices.TunnelRegistry.Enabled {
		if _, err := os.Stat(cfg.NodeJSServices.TunnelRegistry.ScriptPath); os.IsNotExist(err) {
			log.Printf("[%s] WARNING: Tunnel registry script not found: %s", cfg.ChainID, cfg.NodeJSServices.TunnelRegistry.ScriptPath)
			log.Printf("[%s] Disabling tunnel registry service", cfg.ChainID)
			cfg.NodeJSServices.TunnelRegistry.Enabled = false
		}
	}

	if cfg.NodeJSServices.PaymentGateway.Enabled {
		if _, err := os.Stat(cfg.NodeJSServices.PaymentGateway.ScriptPath); os.IsNotExist(err) {
			log.Printf("[%s] WARNING: Payment gateway script not found: %s", cfg.ChainID, cfg.NodeJSServices.PaymentGateway.ScriptPath)
			log.Printf("[%s] Disabling payment gateway service", cfg.ChainID)
			cfg.NodeJSServices.PaymentGateway.Enabled = false
		}
	}

	if cfg.NodeJSServices.OperatorRegistry.Enabled {
		if _, err := os.Stat(cfg.NodeJSServices.OperatorRegistry.ScriptPath); os.IsNotExist(err) {
			log.Printf("[%s] WARNING: Bootnode registry script not found: %s", cfg.ChainID, cfg.NodeJSServices.OperatorRegistry.ScriptPath)
			log.Printf("[%s] Disabling bootnode registry service", cfg.ChainID)
			cfg.NodeJSServices.OperatorRegistry.Enabled = false
		}
	}

	if cfg.NodeJSServices.WebGUI.Enabled {
		if _, err := os.Stat(cfg.NodeJSServices.WebGUI.ScriptPath); os.IsNotExist(err) {
			log.Printf("[%s] WARNING: Notary system script not found: %s", cfg.ChainID, cfg.NodeJSServices.WebGUI.ScriptPath)
			log.Printf("[%s] Disabling notary system service", cfg.ChainID)
			cfg.NodeJSServices.WebGUI.Enabled = false
		}
	}

	if cfg.NodeJSServices.NetworkMonitor.Enabled {
		if _, err := os.Stat(cfg.NodeJSServices.NetworkMonitor.ScriptPath); os.IsNotExist(err) {
			log.Printf("[%s] WARNING: Network monitor script not found: %s", cfg.ChainID, cfg.NodeJSServices.NetworkMonitor.ScriptPath)
			log.Printf("[%s] Disabling network monitor service", cfg.ChainID)
			cfg.NodeJSServices.NetworkMonitor.Enabled = false
		}
	}

	// Create Node.js manager
	nodejsManager := NewNodeJSManager(&cfg.NodeJSServices, discoveryMgr.host.ID().String(), discoveryMgr, nil)

	// Check if any services are enabled
	if !cfg.NodeJSServices.TunnelRegistry.Enabled &&
		!cfg.NodeJSServices.PaymentGateway.Enabled &&
		!cfg.NodeJSServices.OperatorRegistry.Enabled &&
		!cfg.NodeJSServices.WebGUI.Enabled &&
		!cfg.NodeJSServices.NetworkMonitor.Enabled &&
		!cfg.NodeJSServices.NANDAANS.Enabled {
		log.Printf("[%s] No Node.js services are enabled or have valid script paths", cfg.ChainID)
		return nodejsManager, nil
	}

	// Log NANDA-ANS service status (served as static files from Go binary)
	if cfg.NodeJSServices.NANDAANS.Enabled {
		log.Printf("[%s] NANDA-ANS service enabled (served as static files from Go binary)", cfg.ChainID)
	}

	// Start all enabled services
	if err := nodejsManager.StartAllServices(); err != nil {
		return nodejsManager, err
	}

	log.Printf("[%s] Node.js services started successfully", cfg.ChainID)

	return nodejsManager, nil
}

// initEconomicsIntegration initializes the economics integration service
func initEconomicsIntegration(cfg *config.Config) (*EconomicsIntegration, error) {
	log.Printf("[%s] Initializing economics integration...", cfg.ChainID)

	// Create economics integration instance
	economicsIntegration := NewEconomicsIntegration()

	// Enable local mode for root nodes by default
	if cfg.IsRoot {
		os.Setenv("ECONOMICS_LOCAL_MODE", "true")
		log.Printf("[%s] Economics service running in local mode (integrated)", cfg.ChainID)
	} else {
		log.Printf("[%s] Economics service running in remote mode", cfg.ChainID)
	}

	// Start background sync
	economicsIntegration.StartBackgroundSync(context.Background())

	// Add economics endpoints to HTTP server
	economicsIntegration.AddEconomicsEndpoints()

	log.Printf("[%s] Economics integration initialized successfully", cfg.ChainID)
	return economicsIntegration, nil
}
