package oracle

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/oracle/consensus"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/oracle/crosschain"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/oracle/economics"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/oracle/governance"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/oracle/ibc"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/oracle/p2p"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/oracle/token"
	"go.uber.org/zap"
)

// Oracle is the main coordinator for all oracle functionality
// It integrates token, governance, economics, consensus, IBC, cross-chain, and P2P
type Oracle struct {
	// Core components
	nrnToken         *token.NRN
	governanceSystem *governance.GovernanceSystem
	economicsEngine  *economics.EconomicsEngine
	consensusEngine  *consensus.ConsensusEngine
	ibcHandler       *ibc.Handler
	crossChainRouter *crosschain.Router
	p2pManager       *p2p.P2PManager
	bridgeManager    *crosschain.BridgeManager

	// Configuration
	config *OracleConfig
	logger *zap.Logger

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
}

// OracleConfig contains configuration for the oracle
type OracleConfig struct {
	// Chain configuration
	ChainID   string        `json:"chain_id"`
	NetworkID string        `json:"network_id"`
	BlockTime time.Duration `json:"block_time"`

	// Token configuration
	TokenName       string   `json:"token_name"`
	TokenSymbol     string   `json:"token_symbol"`
	InitialSupply   *big.Int `json:"initial_supply"`
	MaxSupply       *big.Int `json:"max_supply"`
	OwnerPrivateKey string   `json:"owner_private_key"`
	ContractAddress string   `json:"contract_address,omitempty"`
	XionRPC         string   `json:"xion_rpc,omitempty"`

	// P2P configuration
	P2PListenAddr    string   `json:"p2p_listen_addr"`
	BootstrapPeers   []string `json:"bootstrap_peers"`
	DHTEnabled       bool     `json:"dht_enabled"`
	GossipSubEnabled bool     `json:"gossipsub_enabled"`

	// Validator configuration
	ValidatorMode bool   `json:"validator_mode"`
	ValidatorKey  string `json:"validator_key,omitempty"`

	// IBC configuration
	IBCEnabled bool `json:"ibc_enabled"`

	// RPC & API
	RPCAddr string `json:"rpc_addr"`
	APIAddr string `json:"api_addr"`

	// Storage configuration
	DataDir   string `json:"data_dir"`
	DBBackend string `json:"db_backend"`
}

// DefaultOracleConfig returns default configuration
func DefaultOracleConfig() *OracleConfig {
	initialSupply, _ := new(big.Int).SetString("1000000000", 10)
	maxSupply, _ := new(big.Int).SetString("10000000000", 10)

	return &OracleConfig{
		ChainID:          "knirvoracle-1",
		NetworkID:        "knirv-testnet",
		BlockTime:        5 * time.Second,
		TokenName:        "KNIRV Network Token",
		TokenSymbol:      "NRN",
		InitialSupply:    initialSupply,
		MaxSupply:        maxSupply,
		OwnerPrivateKey:  "",
		ContractAddress:  "",
		XionRPC:          "https://rpc.xion.testnet",
		P2PListenAddr:    "/ip4/0.0.0.0/tcp/26656",
		BootstrapPeers:   []string{},
		DHTEnabled:       true,
		GossipSubEnabled: true,
		ValidatorMode:    false,
		IBCEnabled:       true,
		RPCAddr:          "127.0.0.1:26657",
		APIAddr:          "0.0.0.0:8080",
		DataDir:          "./data/oracle",
		DBBackend:        "badger",
	}
}

// NewOracle creates a new Oracle instance
func NewOracle(config *OracleConfig, logger *zap.Logger) (*Oracle, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize NRN token
	nrnToken, err := initializeNRNToken(config, logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize NRN token: %w", err)
	}

	// Initialize governance
	governanceSystem := governance.NewGovernanceSystem(logger)

	// Initialize economics engine
	economicsEngine := economics.NewEconomicsEngine(nrnToken, logger)

	// Initialize consensus engine
	consensusEngine := consensus.NewConsensusEngine(
		config.ChainID,
		config.BlockTime,
		logger,
	)

	// Initialize bridge manager
	bridgeManager := crosschain.NewBridgeManager()
	if err := bridgeManager.RegisterBridge(crosschain.DefaultBridgeConfigs()[0]); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to register default bridges: %w", err)
	}

	// Initialize IBC handler
	ibcHandler := ibc.NewHandler(logger)

	// Initialize cross-chain router
	crossChainRouter := crosschain.NewRouter(ibcHandler, bridgeManager, logger)

	// Initialize P2P manager
	p2pConfig := &p2p.P2PConfig{
		ListenAddr:     config.P2PListenAddr,
		BootstrapPeers: config.BootstrapPeers,
		EnableDHT:      config.DHTEnabled,
		EnableGossip:   config.GossipSubEnabled,
	}
	p2pManager := p2p.NewP2PManager(p2pConfig, logger)

	oracle := &Oracle{
		nrnToken:         nrnToken,
		governanceSystem: governanceSystem,
		economicsEngine:  economicsEngine,
		consensusEngine:  consensusEngine,
		ibcHandler:       ibcHandler,
		crossChainRouter: crossChainRouter,
		p2pManager:       p2pManager,
		bridgeManager:    bridgeManager,
		config:           config,
		logger:           logger,
		ctx:              ctx,
		cancel:           cancel,
	}

	logger.Info("Oracle initialized",
		zap.String("chain_id", config.ChainID),
		zap.String("network_id", config.NetworkID),
	)

	return oracle, nil
}

// Start starts all oracle services
func (o *Oracle) Start() error {
	o.logger.Info("Starting Oracle services")

	// Start economics engine
	if err := o.economicsEngine.Start(); err != nil {
		return fmt.Errorf("failed to start economics engine: %w", err)
	}

	// Start consensus engine
	if err := o.consensusEngine.Start(); err != nil {
		return fmt.Errorf("failed to start consensus engine: %w", err)
	}

	// Start IBC handler
	if o.config.IBCEnabled {
		if err := o.ibcHandler.Start(); err != nil {
			return fmt.Errorf("failed to start IBC handler: %w", err)
		}
	}

	// Start P2P manager
	if err := o.p2pManager.Start(); err != nil {
		return fmt.Errorf("failed to start P2P manager: %w", err)
	}

	o.logger.Info("Oracle services started successfully")

	return nil
}

// Stop stops all oracle services
func (o *Oracle) Stop() error {
	o.logger.Info("Stopping Oracle services")

	// Stop in reverse order
	o.p2pManager.Stop()

	if o.config.IBCEnabled {
		o.ibcHandler.Stop()
	}

	o.consensusEngine.Stop()
	o.economicsEngine.Stop()

	o.cancel()

	o.logger.Info("Oracle services stopped")

	return nil
}

// GetNRNToken returns the NRN token instance
func (o *Oracle) GetNRNToken() *token.NRN {
	return o.nrnToken
}

// GetGovernanceSystem returns the governance system
func (o *Oracle) GetGovernanceSystem() *governance.GovernanceSystem {
	return o.governanceSystem
}

// GetEconomicsEngine returns the economics engine
func (o *Oracle) GetEconomicsEngine() *economics.EconomicsEngine {
	return o.economicsEngine
}

// GetConsensusEngine returns the consensus engine
func (o *Oracle) GetConsensusEngine() *consensus.ConsensusEngine {
	return o.consensusEngine
}

// GetIBCHandler returns the IBC handler
func (o *Oracle) GetIBCHandler() *ibc.Handler {
	return o.ibcHandler
}

// GetCrossChainRouter returns the cross-chain router
func (o *Oracle) GetCrossChainRouter() *crosschain.Router {
	return o.crossChainRouter
}

// GetP2PManager returns the P2P manager
func (o *Oracle) GetP2PManager() *p2p.P2PManager {
	return o.p2pManager
}

// GetBridgeManager returns the bridge manager
func (o *Oracle) GetBridgeManager() *crosschain.BridgeManager {
	return o.bridgeManager
}

// GetStatus returns the current status of the oracle
func (o *Oracle) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"chain_id":   o.config.ChainID,
		"network_id": o.config.NetworkID,
		"token":      o.nrnToken.Info(),
		"consensus":  o.consensusEngine.GetInfo(),
		"governance": o.governanceSystem.GetValidatorStats(),
		"economics":  o.economicsEngine.GetEconomicSnapshot(),
		"p2p":        o.p2pManager.GetStats(),
		"ibc": map[string]interface{}{
			"enabled": o.config.IBCEnabled,
		},
	}
}

// Helper: initialize NRN token
func initializeNRNToken(config *OracleConfig, logger *zap.Logger) (*token.NRN, error) {
	// Validate supply values
	if config.InitialSupply == nil {
		return nil, fmt.Errorf("initial supply is required")
	}
	if config.MaxSupply == nil {
		return nil, fmt.Errorf("max supply is required")
	}

	// Create NRN token
	nrnToken, err := token.NewNRN(
		config.TokenName,
		config.TokenSymbol,
		config.InitialSupply,
		config.MaxSupply,
		config.OwnerPrivateKey,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create NRN token: %w", err)
	}

	logger.Info("NRN token initialized",
		zap.String("name", config.TokenName),
		zap.String("symbol", config.TokenSymbol),
		zap.String("initial_supply", config.InitialSupply.String()),
		zap.String("max_supply", config.MaxSupply.String()),
	)

	return nrnToken, nil
}
