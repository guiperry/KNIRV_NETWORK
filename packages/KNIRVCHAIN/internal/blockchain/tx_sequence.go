package blockchain

// Sequencing of signed chain transactions.
//
// A signed transaction is replayable unless the chain orders each account's
// transactions. Each account has a next expected sequence, stored in the
// database. A transaction is valid only when it carries exactly that sequence,
// and applying it advances the sequence by one. The check runs at block
// application (consensus state) and at intake (a pre-check against committed
// state plus pending pool transactions).
//
// Enforcement is on from genesis. KNIRV_TX_SEQUENCE_ACTIVATION_HEIGHT can move the
// activation height for a ledger that already has history.

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strconv"

	"KNIRVCHAIN/internal/database"
	"KNIRVCHAIN/internal/types"
	"KNIRVCHAIN/internal/utils"
)

const txSequenceActivationEnv = "KNIRV_TX_SEQUENCE_ACTIVATION_HEIGHT"

// txSequenceActivation returns the height from which sequencing applies. Sequencing is
// on from genesis by default; KNIRV_TX_SEQUENCE_ACTIVATION_HEIGHT overrides the height
// (for example to activate it on a ledger that already has history).
func txSequenceActivation() uint64 {
	raw := os.Getenv(txSequenceActivationEnv)
	if raw == "" {
		return 0
	}
	activation, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0
	}
	return activation
}

// sequenceEnforcedAt reports whether transactions in the block at height must
// carry their sender's next sequence and account number.
func sequenceEnforcedAt(height uint64) bool {
	return height >= txSequenceActivation()
}

// AccountNumberFor is the account number a signer must use. See utils.AccountNumberFor.
func AccountNumberFor(address string) uint64 {
	return utils.AccountNumberFor(address)
}

// blockStateEffects collects the sequence and capability changes of one block.
// They are written only after the block has been validated, so a rejected block
// leaves no partial state behind.
type blockStateEffects struct {
	sequences    map[string]uint64
	capabilities []interface{}
	pendingIDs   map[string]bool
}

func newBlockStateEffects() *blockStateEffects {
	return &blockStateEffects{sequences: make(map[string]uint64), pendingIDs: make(map[string]bool)}
}

// applySequence checks a transaction's account number and sequence against the
// sender's state at height, and advances the sender's sequence.
func (e *blockStateEffects) applySequence(db *database.LevelDB, height uint64, tx *Transaction) error {
	if !sequenceEnforcedAt(height) || tx.From == utils.BLOCKCHAIN_ADDRESS {
		return nil
	}
	if tx.AccountNumber != AccountNumberFor(tx.From) {
		return fmt.Errorf("transaction %s: account number %d does not belong to %s", tx.TransactionHash, tx.AccountNumber, tx.From)
	}
	expected, loaded := e.sequences[tx.From]
	if !loaded {
		stored, err := db.GetAccountSequence(tx.From)
		if err != nil {
			return err
		}
		expected = stored
	}
	if tx.Sequence != expected {
		return fmt.Errorf("transaction %s: sequence %d does not match the next sequence %d of %s", tx.TransactionHash, tx.Sequence, expected, tx.From)
	}
	e.sequences[tx.From] = expected + 1
	return nil
}

// addRegisteredCapability validates a register transaction's capability and queues
// it for storage. A capability ID that already exists, or is registered twice in
// the same block, is rejected.
func (e *blockStateEffects) addRegisteredCapability(db *database.LevelDB, tx *Transaction) error {
	var registration types.MCPRegisterCapabilityData
	if err := json.Unmarshal(tx.Data, &registration); err != nil {
		return fmt.Errorf("transaction %s: decode capability registration: %w", tx.TransactionHash, err)
	}
	exists, err := db.CapabilityExists(registration.CapabilityID)
	if err != nil {
		return err
	}
	if exists || e.pendingIDs[registration.CapabilityID] {
		return fmt.Errorf("capability %s already exists", registration.CapabilityID)
	}
	capability, err := capabilityFromRegistration(registration, tx.Timestamp)
	if err != nil {
		return fmt.Errorf("transaction %s: %w", tx.TransactionHash, err)
	}
	e.pendingIDs[registration.CapabilityID] = true
	e.capabilities = append(e.capabilities, capability)
	return nil
}

// capabilityFromRegistration builds the typed descriptor a register transaction declares.
func capabilityFromRegistration(registration types.MCPRegisterCapabilityData, timestamp int64) (interface{}, error) {
	capabilityType := types.CapabilityType(registration.CapabilityType)
	if !types.IsValidCapabilityType(capabilityType) {
		return nil, fmt.Errorf("unsupported capability type %q", registration.CapabilityType)
	}
	base := types.BaseDescriptor{
		ID:             registration.CapabilityID,
		CapabilityType: capabilityType,
		Name:           registration.Name,
		Owner:          registration.Owner,
		Version:        registration.Version,
		Description:    registration.Description,
		GasFeeNRN:      registration.GasFeeNRN,
		Timestamp:      timestamp,
	}
	raw, err := json.Marshal(registration.CapabilityDescriptor)
	if err != nil {
		return nil, fmt.Errorf("encode capability descriptor: %w", err)
	}
	switch capabilityType {
	case types.CapabilityTypeTool:
		var tool types.ToolDescriptor
		if err := json.Unmarshal(raw, &tool); err != nil {
			return nil, fmt.Errorf("decode tool descriptor: %w", err)
		}
		tool.BaseDescriptor = base
		return tool, nil
	case types.CapabilityTypeResource:
		var resource types.ResourceDescriptor
		if err := json.Unmarshal(raw, &resource); err != nil {
			return nil, fmt.Errorf("decode resource descriptor: %w", err)
		}
		resource.BaseDescriptor = base
		return resource, nil
	case types.CapabilityTypePrompt:
		var prompt types.PromptDescriptor
		if err := json.Unmarshal(raw, &prompt); err != nil {
			return nil, fmt.Errorf("decode prompt descriptor: %w", err)
		}
		prompt.BaseDescriptor = base
		return prompt, nil
	case types.CapabilityTypeMemoryService:
		var memory types.MemoryServiceDescriptor
		if err := json.Unmarshal(raw, &memory); err != nil {
			return nil, fmt.Errorf("decode memory service descriptor: %w", err)
		}
		memory.BaseDescriptor = base
		return memory, nil
	}
	return nil, fmt.Errorf("unsupported capability type %q", capabilityType)
}

// checkIntakeSequence is the intake pre-check. It runs before a transaction enters
// the pool, against committed state plus transactions already pending from the sender.
// The caller holds sequenceMu across this check and the pool admission.
func (bcs *BlockchainServer) checkIntakeSequence(tx *Transaction) (int, error) {
	if tx.Type == TransactionTypeMCPRegisterCapability {
		exists, err := bcs.db.CapabilityExists(capabilityIDOf(tx))
		if err != nil {
			return http.StatusInternalServerError, err
		}
		if exists {
			return http.StatusConflict, fmt.Errorf("capability %s already exists", capabilityIDOf(tx))
		}
	}

	bcs.BlockchainPtr.Lock()
	height := uint64(len(bcs.BlockchainPtr.Blocks))
	pending := uint64(0)
	for _, queued := range bcs.BlockchainPtr.TransactionPool {
		if queued.From == tx.From {
			pending++
		}
	}
	bcs.BlockchainPtr.Unlock()

	if !sequenceEnforcedAt(height) {
		return 0, nil
	}
	if tx.AccountNumber != AccountNumberFor(tx.From) {
		return http.StatusConflict, fmt.Errorf("account number %d does not belong to %s", tx.AccountNumber, tx.From)
	}
	committed, err := bcs.db.GetAccountSequence(tx.From)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	next := committed + pending
	if tx.Sequence != next {
		return http.StatusConflict, fmt.Errorf("sequence %d does not match the next sequence %d of %s", tx.Sequence, next, tx.From)
	}
	return 0, nil
}

// fundsAfterPending checks that the sender can pay the fee (and value) of tx after
// the transactions it already has pending. Block validation rejects the whole block
// for an unfunded transaction, so this check keeps such a transaction out of the pool.
func fundsAfterPending(committed uint64, pending []*Transaction, tx *Transaction) error {
	balance := new(big.Int).SetUint64(committed)
	for _, queued := range pending {
		balance.Sub(balance, cost(queued))
	}
	if balance.Sign() < 0 {
		return fmt.Errorf("account %s has pending transactions it cannot fund", tx.From)
	}
	if balance.Cmp(cost(tx)) < 0 {
		return fmt.Errorf("insufficient balance for transaction %s: balance %s, required %s", tx.TransactionHash, balance.String(), cost(tx).String())
	}
	return nil
}

// cost is what a transaction takes from its sender: its network fee plus its value.
func cost(tx *Transaction) *big.Int {
	total := new(big.Int).SetUint64(tx.Fee)
	return total.Add(total, new(big.Int).SetUint64(tx.Value))
}

// checkRegistrationContent rejects a registration whose capability block validation
// would reject, so the block is never rejected for it.
func checkRegistrationContent(tx *Transaction) error {
	var registration types.MCPRegisterCapabilityData
	if err := json.Unmarshal(tx.Data, &registration); err != nil {
		return fmt.Errorf("decode capability registration: %w", err)
	}
	if registration.CapabilityID == "" || registration.Name == "" || registration.Owner == "" {
		return fmt.Errorf("capability ID, name and owner are required")
	}
	_, err := capabilityFromRegistration(registration, tx.Timestamp)
	return err
}

// checkIntakeFunds runs the funds and content checks against the sender's committed
// state plus its pending transactions.
func (bcs *BlockchainServer) checkIntakeFunds(tx *Transaction) (int, error) {
	if tx.Type == TransactionTypeMCPRegisterCapability {
		if err := checkRegistrationContent(tx); err != nil {
			return http.StatusUnprocessableEntity, err
		}
	}
	if tx.From == utils.BLOCKCHAIN_ADDRESS {
		return 0, nil
	}
	committed, err := bcs.db.GetAccountBalance(tx.From)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	bcs.BlockchainPtr.Lock()
	var pending []*Transaction
	for _, queued := range bcs.BlockchainPtr.TransactionPool {
		if queued.From == tx.From {
			pending = append(pending, queued)
		}
	}
	bcs.BlockchainPtr.Unlock()
	if err := fundsAfterPending(committed, pending, tx); err != nil {
		return http.StatusUnprocessableEntity, err
	}
	return 0, nil
}

func capabilityIDOf(tx *Transaction) string {
	var registration types.MCPRegisterCapabilityData
	if err := json.Unmarshal(tx.Data, &registration); err != nil {
		return ""
	}
	return registration.CapabilityID
}

// handleAccountInfo reports what a signer needs to build a valid transaction: its
// account number, the committed sequence, pending transactions, and the next
// sequence to sign with. Route: GET /accounts/info?address=<addr>.
func (bcs *BlockchainServer) handleAccountInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	address := r.URL.Query().Get("address")
	if address == "" {
		http.Error(w, "Missing address query parameter", http.StatusBadRequest)
		return
	}
	if bcs.db == nil || bcs.BlockchainPtr == nil {
		http.Error(w, "Chain not initialized", http.StatusServiceUnavailable)
		return
	}
	committed, err := bcs.db.GetAccountSequence(address)
	if err != nil {
		http.Error(w, "Failed to read account sequence", http.StatusInternalServerError)
		return
	}
	bcs.BlockchainPtr.Lock()
	height := uint64(len(bcs.BlockchainPtr.Blocks))
	pending := uint64(0)
	for _, queued := range bcs.BlockchainPtr.TransactionPool {
		if queued.From == address {
			pending++
		}
	}
	chainID := bcs.BlockchainPtr.ChainID
	bcs.BlockchainPtr.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"address":           address,
		"chain_id":          chainID,
		"account_number":    AccountNumberFor(address),
		"sequence":          committed,
		"pending":           pending,
		"next_sequence":     committed + pending,
		"sequence_enforced": sequenceEnforcedAt(height),
	}); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
