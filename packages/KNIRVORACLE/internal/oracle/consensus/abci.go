package consensus

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/knirvcorp/knirvoracle/internal/oracle/mmr"
	"go.uber.org/zap"
)

// ABCIApplication implements a simplified ABCI interface
// In production, this would implement github.com/cometbft/cometbft/abci/types.Application
type ABCIApplication struct {
	state           *State
	validators      *ValidatorSet
	consensusParams *ConsensusParams
	auditMMR        *mmr.MMR
	auditMMRPath    string
	// onCheckpointTx routes a consensus-delivered checkpoint/finality transaction
	// to the Oracle's admission path (merkle-math.md §3.3 — same path as HTTP).
	onCheckpointTx func(txType string, payload []byte) error
	logger         *zap.Logger
	mu             sync.RWMutex
}

// NewABCIApplication creates a new ABCI application. If dataDir is non-empty the
// audit MMR (the source of the block AppHash) is recovered from disk and
// re-persisted on every Commit, so the AppHash survives an Oracle restart
// (merkle-math.md Phase 3).
func NewABCIApplication(chainID, dataDir string, logger *zap.Logger) *ABCIApplication {
	app := &ABCIApplication{
		state: &State{
			Version: ConsensusVersion{
				Block: 11,
				App:   1,
			},
			ChainID:         chainID,
			InitialHeight:   1,
			LastBlockHeight: 0,
			Validators:      NewValidatorSet([]*ConsensusValidator{}),
			NextValidators:  NewValidatorSet([]*ConsensusValidator{}),
			LastValidators:  NewValidatorSet([]*ConsensusValidator{}),
			ConsensusParams: *DefaultConsensusParams(),
		},
		consensusParams: DefaultConsensusParams(),
		auditMMR:        mmr.New(),
		logger:          logger,
	}
	if dataDir != "" {
		app.auditMMRPath = filepath.Join(dataDir, "audit_mmr_leaf_log.json")
		if data, err := os.ReadFile(app.auditMMRPath); err == nil {
			var leaves []mmr.Hash
			if jerr := json.Unmarshal(data, &leaves); jerr != nil {
				logger.Error("failed to decode audit MMR leaf log", zap.Error(jerr))
			} else {
				app.auditMMR = mmr.FromLeaves(leaves)
			}
		}
	}
	return app
}

// SetCheckpointTxHandler registers the consensus-path admission hook used by
// DeliverTx for checkpoint/finality transactions.
func (app *ABCIApplication) SetCheckpointTxHandler(fn func(txType string, payload []byte) error) {
	app.mu.Lock()
	defer app.mu.Unlock()
	app.onCheckpointTx = fn
}

// SetAuditMMR installs the Oracle checkpoint log as the single MMR instance
// used for both inclusion proofs and AppHash commits.
func (app *ABCIApplication) SetAuditMMR(shared *mmr.MMR) {
	app.mu.Lock()
	defer app.mu.Unlock()
	if shared == nil {
		shared = mmr.New()
	}
	app.auditMMR = shared
}

// AddAuditLeaf appends through the application lock to the canonical MMR.
func (app *ABCIApplication) AddAuditLeaf(data []byte) (uint64, mmr.Hash) {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.auditMMR.AddRaw(mmr.LeafHash(data))
}

func (app *ABCIApplication) AuditSnapshot() (mmr.Hash, uint64) {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return app.auditMMR.BagRoot(), app.auditMMR.Size()
}

// Info returns information about the application state
func (app *ABCIApplication) Info() map[string]interface{} {
	app.mu.RLock()
	defer app.mu.RUnlock()

	return map[string]interface{}{
		"chain_id":          app.state.ChainID,
		"last_block_height": app.state.LastBlockHeight,
		"app_hash":          fmt.Sprintf("%x", app.state.AppHash),
		"version":           app.state.Version,
	}
}

// InitChain initializes the blockchain
func (app *ABCIApplication) InitChain(chainID string, validators []*ConsensusValidator) error {
	app.mu.Lock()
	defer app.mu.Unlock()

	app.state.ChainID = chainID
	app.state.InitialHeight = 1
	app.state.Validators = NewValidatorSet(validators)
	app.state.NextValidators = NewValidatorSet(validators)
	app.state.LastValidators = NewValidatorSet([]*ConsensusValidator{})

	app.logger.Info("Chain initialized",
		zap.String("chain_id", chainID),
		zap.Int("validators", len(validators)),
	)

	return nil
}

// BeginBlock signals the beginning of a new block
func (app *ABCIApplication) BeginBlock(header *BlockHeader) error {
	app.mu.Lock()
	defer app.mu.Unlock()

	app.logger.Debug("Begin block",
		zap.Uint64("height", uint64(header.Height)),
		zap.String("proposer", header.ProposerAddress.String()),
	)

	return nil
}

// DeliverTx delivers a transaction to the application
func (app *ABCIApplication) DeliverTx(tx []byte) error {
	// Parse and validate transaction
	var txData map[string]interface{}
	if err := json.Unmarshal(tx, &txData); err != nil {
		return fmt.Errorf("invalid transaction format: %w", err)
	}

	// Route checkpoint/finality transactions arriving through consensus to the
	// same Oracle admission path used by the HTTP endpoints (merkle-math.md §3.3).
	txType, _ := txData["type"].(string)
	if txType == "checkpoint" || txType == "finality" {
		app.mu.RLock()
		handler := app.onCheckpointTx
		app.mu.RUnlock()
		if handler != nil {
			payload, _ := json.Marshal(txData["payload"])
			return handler(txType, payload)
		}
	}

	app.logger.Debug("Transaction delivered",
		zap.Int("size", len(tx)),
	)

	return nil
}

// EndBlock signals the end of a block
func (app *ABCIApplication) EndBlock(height BlockHeight) (*ValidatorSet, error) {
	app.mu.Lock()
	defer app.mu.Unlock()

	app.logger.Debug("End block",
		zap.Uint64("height", uint64(height)),
	)

	// Return validator updates (if any)
	return app.state.NextValidators, nil
}

// Commit commits the current state
func (app *ABCIApplication) Commit() ([]byte, error) {
	app.mu.Lock()
	defer app.mu.Unlock()

	// Update state
	app.state.LastBlockHeight++

	// Every Oracle block commits to the append-only audit log. Before the first
	// audit leaf, this is the canonical SHA256("knirv-mmr-empty") root.
	root := app.auditMMR.BagRoot()
	appHash := append([]byte(nil), root[:]...)
	app.state.AppHash = appHash

	// Persist the audit MMR so the AppHash recovers exactly after a restart.
	if app.auditMMRPath != "" {
		leaves := app.auditMMR.Leaves()
		payload, err := json.MarshalIndent(leaves, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("encode audit mmr: %w", err)
		}
		tmp := app.auditMMRPath + ".tmp"
		if werr := os.WriteFile(tmp, payload, 0644); werr != nil {
			return nil, fmt.Errorf("write audit mmr: %w", werr)
		}
		if rerr := os.Rename(tmp, app.auditMMRPath); rerr != nil {
			return nil, fmt.Errorf("rename audit mmr: %w", rerr)
		}
	}

	app.logger.Info("Block committed",
		zap.Uint64("height", uint64(app.state.LastBlockHeight)),
		zap.String("app_hash", fmt.Sprintf("%x", appHash)),
	)

	return appHash, nil
}

// Query queries the application state
func (app *ABCIApplication) Query(path string, data []byte) ([]byte, error) {
	app.mu.RLock()
	defer app.mu.RUnlock()

	switch path {
	case "/info":
		info := app.Info()
		return json.Marshal(info)
	case "/state":
		return json.Marshal(app.state)
	case "/validators":
		return json.Marshal(app.state.Validators)
	default:
		return nil, fmt.Errorf("unknown query path: %s", path)
	}
}

// CheckTx checks if a transaction is valid
func (app *ABCIApplication) CheckTx(tx []byte) error {
	// Parse transaction
	var txData map[string]interface{}
	if err := json.Unmarshal(tx, &txData); err != nil {
		return fmt.Errorf("invalid transaction format: %w", err)
	}

	// Basic validation
	if len(tx) == 0 {
		return fmt.Errorf("empty transaction")
	}

	if len(tx) > int(app.consensusParams.Block.MaxBytes) {
		return fmt.Errorf("transaction too large")
	}

	return nil
}

// SetValidators updates the validator set
func (app *ABCIApplication) SetValidators(validators []*ConsensusValidator) error {
	app.mu.Lock()
	defer app.mu.Unlock()

	app.state.NextValidators = NewValidatorSet(validators)
	app.state.LastHeightValidatorsChanged = app.state.LastBlockHeight

	app.logger.Info("Validator set updated",
		zap.Int("validators", len(validators)),
		zap.Uint64("height", uint64(app.state.LastBlockHeight)),
	)

	return nil
}

// GetState returns the current state
func (app *ABCIApplication) GetState() *State {
	app.mu.RLock()
	defer app.mu.RUnlock()

	return app.state
}

// GetValidators returns the current validator set
func (app *ABCIApplication) GetValidators() *ValidatorSet {
	app.mu.RLock()
	defer app.mu.RUnlock()

	return app.state.Validators
}

// UpdateConsensusParams updates consensus parameters
func (app *ABCIApplication) UpdateConsensusParams(params *ConsensusParams) error {
	app.mu.Lock()
	defer app.mu.Unlock()

	app.state.ConsensusParams = *params
	app.consensusParams = params
	app.state.LastHeightConsensusParamsChanged = app.state.LastBlockHeight

	app.logger.Info("Consensus params updated",
		zap.Uint64("height", uint64(app.state.LastBlockHeight)),
	)

	return nil
}
