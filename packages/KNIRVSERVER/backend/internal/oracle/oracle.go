package oracle

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"

	"backend_server/internal/oracle/consensus"
	"backend_server/internal/oracle/crosschain"
	"backend_server/internal/oracle/crypto"
	"backend_server/internal/oracle/economics"
	"backend_server/internal/oracle/governance"
	"backend_server/internal/oracle/ibc"
	"backend_server/internal/oracle/p2p"
	"backend_server/internal/oracle/token"
	"backend_server/internal/oracle/types"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
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
	rollupsMu        sync.RWMutex
	rollups          map[string]*types.RollupRecord
	rollupsPath      string

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
		config.ValidatorMode,
		logger,
	)

	// Initialize consensus validators
	var validators []*consensus.ConsensusValidator
	if config.ValidatorMode && config.ValidatorKey != "" {
		kp, err := crypto.PrivateKeyFromHex(config.ValidatorKey)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to parse validator key: %w", err)
		}

		validator := &consensus.ConsensusValidator{
			Address:          kp.Address,
			PubKey:           ethcrypto.FromECDSAPub(kp.PublicKey),
			VotingPower:      big.NewInt(100), // Default voting power
			ProposerPriority: big.NewInt(0),
		}
		validators = append(validators, validator)

		logger.Info("Validator initialized",
			zap.String("address", kp.Address.String()),
			zap.Int64("voting_power", 100),
		)
	}

	// Initialize chain with validators
	if err := consensusEngine.InitChain(validators); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize consensus chain: %w", err)
	}

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
		rollups:          make(map[string]*types.RollupRecord),
		rollupsPath:      filepath.Join(config.DataDir, "rollups.json"),
		config:           config,
		logger:           logger,
		ctx:              ctx,
		cancel:           cancel,
	}

	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create oracle data directory: %w", err)
	}
	if err := oracle.loadRollups(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to load persisted oracle rollups: %w", err)
	}

	logger.Info("Oracle initialized",
		zap.String("chain_id", config.ChainID),
		zap.String("network_id", config.NetworkID),
	)

	return oracle, nil
}

func (o *Oracle) SubmitRollup(record *types.RollupRecord) error {
	if record == nil {
		return fmt.Errorf("rollup record is required")
	}
	o.rollupsMu.Lock()
	defer o.rollupsMu.Unlock()
	o.rollups[record.ID] = record
	return o.persistRollupsLocked()
}

func (o *Oracle) GetRollup(id string) (*types.RollupRecord, bool) {
	o.rollupsMu.RLock()
	defer o.rollupsMu.RUnlock()
	record, ok := o.rollups[id]
	return record, ok
}

func (o *Oracle) FinalizeRollup(id string, finalizedAt time.Time) (*types.RollupRecord, error) {
	o.rollupsMu.Lock()
	defer o.rollupsMu.Unlock()

	record, ok := o.rollups[id]
	if !ok {
		return nil, fmt.Errorf("rollup not found: %s", id)
	}

	record.Status = types.RollupStatusFinalized
	timestamp := finalizedAt.UTC()
	record.FinalizedAt = &timestamp
	if err := o.persistRollupsLocked(); err != nil {
		return nil, err
	}
	return record, nil
}

func (o *Oracle) DisputeRollup(id string, reason string, disputedAt time.Time) (*types.RollupRecord, error) {
	o.rollupsMu.Lock()
	defer o.rollupsMu.Unlock()

	record, ok := o.rollups[id]
	if !ok {
		return nil, fmt.Errorf("rollup not found: %s", id)
	}

	record.Status = types.RollupStatusDisputed
	record.Dispute = reason
	timestamp := disputedAt.UTC()
	record.DisputedAt = &timestamp
	if err := o.persistRollupsLocked(); err != nil {
		return nil, err
	}
	return record, nil
}

func (o *Oracle) loadRollups() error {
	o.rollupsMu.Lock()
	defer o.rollupsMu.Unlock()

	data, err := os.ReadFile(o.rollupsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read oracle rollup state: %w", err)
	}

	var records []*types.RollupRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return fmt.Errorf("failed to decode oracle rollup state: %w", err)
	}

	o.rollups = make(map[string]*types.RollupRecord, len(records))
	for _, record := range records {
		if record == nil {
			continue
		}
		o.rollups[record.ID] = record
	}

	return nil
}

func (o *Oracle) persistRollupsLocked() error {
	if o.rollupsPath == "" {
		return nil
	}

	records := make([]*types.RollupRecord, 0, len(o.rollups))
	for _, record := range o.rollups {
		records = append(records, record)
	}

	payload, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode oracle rollup state: %w", err)
	}

	tempPath := o.rollupsPath + ".tmp"
	if err := os.WriteFile(tempPath, payload, 0644); err != nil {
		return fmt.Errorf("failed to write oracle rollup temp file: %w", err)
	}

	if err := os.Rename(tempPath, o.rollupsPath); err != nil {
		return fmt.Errorf("failed to move oracle rollup state into place: %w", err)
	}

	return nil
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
